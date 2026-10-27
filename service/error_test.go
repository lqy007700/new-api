package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResetStatusCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		statusCode       int
		statusCodeConfig string
		expectedCode     int
	}{
		{
			name:             "map string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"503"}`,
			expectedCode:     503,
		},
		{
			name:             "map int value",
			statusCode:       429,
			statusCodeConfig: `{"429":503}`,
			expectedCode:     503,
		},
		{
			name:             "skip invalid string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"bad-code"}`,
			expectedCode:     429,
		},
		{
			name:             "skip status code 200",
			statusCode:       200,
			statusCodeConfig: `{"200":503}`,
			expectedCode:     200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			newAPIError := &types.NewAPIError{
				StatusCode: tc.statusCode,
			}
			ResetStatusCode(newAPIError, tc.statusCodeConfig)
			require.Equal(t, tc.expectedCode, newAPIError.StatusCode)
		})
	}
}

func TestRelayErrorHandlerTruncatesInvalidJSONBodyInLog(t *testing.T) {
	withDebugEnabled(t, false)

	body := strings.Repeat("b", common.LocalLogContentLimit+256)
	var logBuffer bytes.Buffer

	common.LogWriterMu.Lock()
	oldWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logBuffer
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = oldWriter
		common.LogWriterMu.Unlock()
	})

	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, "bad response status code 500", newAPIError.Error())
	require.Contains(t, logBuffer.String(), "[truncated")
	require.Contains(t, logBuffer.String(), fmt.Sprintf("original_length=%d", len(body)))
	require.NotContains(t, logBuffer.String(), strings.Repeat("b", common.LocalLogContentLimit+1))
}

func TestRelayErrorHandlerKeepsStructuredErrorMessage(t *testing.T) {
	message := strings.Repeat("c", common.LocalLogContentLimit+256)
	body := `{"message":"` + message + `"}`
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, message, newAPIError.Error())
}

func TestRelayErrorHandlerKeepsOpenAIErrorMessage(t *testing.T) {
	message := strings.Repeat("d", common.LocalLogContentLimit+256)
	body := `{"error":{"message":"` + message + `","type":"server_error","code":"server_error"}}`
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, message, newAPIError.Error())
}

func TestChannel409ReturnsSessionUnavailableWithoutRetry(t *testing.T) {
	previousRetryRanges := operation_setting.AutomaticRetryStatusCodeRanges
	operation_setting.AutomaticRetryStatusCodeRanges = []operation_setting.StatusCodeRange{{Start: 100, End: 599}}
	t.Cleanup(func() { operation_setting.AutomaticRetryStatusCodeRanges = previousRetryRanges })

	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "structured", body: `{"error":{"message":"private upstream session detail","type":"invalid_request_error","code":"history_unavailable","param":"private_parameter","metadata":{"provider":"private_provider"}}}`},
		{name: "channel error code", body: `{"error":{"message":"private upstream session detail","code":"channel:invalid_key"}}`},
		{name: "quota error code", body: `{"error":{"message":"private upstream session detail","code":"insufficient_user_quota"}}`},
		{name: "message only", body: `{"message":"private upstream session detail"}`},
		{name: "invalid json", body: "private upstream session detail"},
		{name: "empty body"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: http.StatusConflict, Body: io.NopCloser(strings.NewReader(tc.body))}
			apiErr := RelayErrorHandler(context.Background(), resp, true)
			require.NotNil(t, apiErr)
			internalMessage := apiErr.Error()
			if tc.body != "" {
				require.Contains(t, internalMessage, "private upstream session detail")
			}

			// Existing channel mappings and broad retry rules cannot hide the
			// conflict or replay the unavailable session on another channel.
			ResetStatusCode(apiErr, `{"409":"500"}`)
			require.Equal(t, http.StatusConflict, apiErr.ClientStatusCode())
			require.False(t, operation_setting.ShouldRetryByStatusCode(http.StatusConflict))
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			require.Equal(t, "stop", DecideRelayRetry(c, apiErr, 3).Action)

			openaiErr := apiErr.ToClientOpenAIError("req-session-409")
			require.Equal(t, types.OpenAIError{
				Type: "session_error", Code: "session_unavailable",
				Message: "当前会话已不可用，请新建会话后重试。 (request id: req-session-409)",
			}, openaiErr)
			encoded, err := common.Marshal(map[string]any{"error": openaiErr})
			require.NoError(t, err)
			require.JSONEq(t, `{"error":{"message":"当前会话已不可用，请新建会话后重试。 (request id: req-session-409)","type":"session_error","code":"session_unavailable","param":""}}`, string(encoded))
			claudeErr := apiErr.ToClientClaudeError("req-session-409")
			require.Equal(t, "session_error", claudeErr.Type)
			require.Equal(t, openaiErr.Message, claudeErr.Message)
			require.Equal(t, internalMessage, apiErr.Error(), "client projection preserves internal diagnostics")
		})
	}
}

func TestRelayErrorHandlerKeepsInvalidJSONBodyInDebugLog(t *testing.T) {
	withDebugEnabled(t, true)

	body := strings.Repeat("e", common.LocalLogContentLimit+256)
	var logBuffer bytes.Buffer

	common.LogWriterMu.Lock()
	oldWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logBuffer
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = oldWriter
		common.LogWriterMu.Unlock()
	})

	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.NotContains(t, logBuffer.String(), "[truncated")
	require.Contains(t, logBuffer.String(), body)
}

func withDebugEnabled(t *testing.T, enabled bool) {
	t.Helper()

	oldDebug := common.DebugEnabled
	common.DebugEnabled = enabled
	t.Cleanup(func() {
		common.DebugEnabled = oldDebug
	})
}
