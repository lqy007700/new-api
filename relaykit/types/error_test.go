package types

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAPIErrorClientResponseSanitization(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		errorCode  ErrorCode
		wantStatus int
		wantType   string
		wantCode   string
		wantText   string
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantStatus: http.StatusUnauthorized,
			wantType:   "authentication_error",
			wantCode:   "invalid_api_key",
			wantText:   ClientErrorMessageUnauthorized,
		},
		{
			name:       "quota",
			statusCode: http.StatusForbidden,
			errorCode:  ErrorCodeInsufficientUserQuota,
			wantStatus: http.StatusPaymentRequired,
			wantType:   "insufficient_quota",
			wantCode:   "insufficient_quota",
			wantText:   ClientErrorMessageQuota,
		},
		{
			name:       "rate limit",
			statusCode: http.StatusTooManyRequests,
			wantStatus: http.StatusTooManyRequests,
			wantType:   "rate_limit_error",
			wantCode:   "rate_limit_exceeded",
			wantText:   ClientErrorMessageRateLimit,
		},
		{
			name:       "upstream 404 is hidden as server error",
			statusCode: http.StatusNotFound,
			wantStatus: http.StatusInternalServerError,
			wantType:   "server_error",
			wantCode:   "server_error",
			wantText:   ClientErrorMessageServer,
		},
		{
			name:       "upstream 503 keeps a 5xx status",
			statusCode: http.StatusServiceUnavailable,
			wantStatus: http.StatusServiceUnavailable,
			wantType:   "server_error",
			wantCode:   "server_error",
			wantText:   ClientErrorMessageServer,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewErrorWithStatusCode(errors.New("upstream secret: model and provider details"), test.errorCode, test.statusCode)
			require.Equal(t, test.wantStatus, err.ClientStatusCode())

			clientErr := err.ToClientOpenAIError("")
			require.Equal(t, test.wantType, clientErr.Type)
			require.Equal(t, test.wantCode, clientErr.Code)
			require.Equal(t, test.wantText, clientErr.Message)
			require.NotContains(t, clientErr.Message, "upstream secret")

			withID := err.ToClientOpenAIError("req_test_123")
			require.Contains(t, withID.Message, "req_test_123")
		})
	}
}

func TestNewAPIErrorClientClaudeResponseSanitization(t *testing.T) {
	err := NewErrorWithStatusCode(errors.New("provider details"), ErrorCodeBadResponseStatusCode, http.StatusNotFound)
	clientErr := err.ToClientClaudeError("req_test_456")
	require.Equal(t, "api_error", clientErr.Type)
	require.Equal(t, ClientErrorMessageServer+" (request id: req_test_456)", clientErr.Message)
	require.NotContains(t, clientErr.Message, "provider details")
}
