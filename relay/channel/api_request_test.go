package channel

import (
	"context"
	"fmt"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskAPIRequestInheritsClientCancellation(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil).WithContext(requestContext)

	upstream, err := newTaskAPIRequest(c, "https://provider.example/tasks", nil)
	require.NoError(t, err)
	cancel()

	require.ErrorIs(t, upstream.Context().Err(), context.Canceled)
}

func TestProcessHeaderOverride_ChannelTestSkipsPassthroughRules(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"*": "",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Empty(t, headers)
}

func TestProcessHeaderOverride_ChannelTestSkipsClientHeaderPlaceholder(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Upstream-Trace": "{client_header:X-Trace-Id}",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	_, ok := headers["x-upstream-trace"]
	require.False(t, ok)
}

func TestProcessHeaderOverride_NonTestKeepsClientHeaderPlaceholder(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Upstream-Trace": "{client_header:X-Trace-Id}",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "trace-123", headers["x-upstream-trace"])
}

func TestProcessHeaderOverride_RuntimeOverrideIsFinalHeaderMap(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	info := &relaycommon.RelayInfo{
		IsChannelTest:             false,
		UseRuntimeHeadersOverride: true,
		RuntimeHeadersOverride: map[string]any{
			"x-static":  "runtime-value",
			"x-runtime": "runtime-only",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Static": "legacy-value",
				"X-Legacy": "legacy-only",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "runtime-value", headers["x-static"])
	require.Equal(t, "runtime-only", headers["x-runtime"])
	_, exists := headers["x-legacy"]
	require.False(t, exists)
}

func TestProcessHeaderOverride_PassthroughSkipsAcceptEncoding(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")
	ctx.Request.Header.Set("Accept-Encoding", "gzip")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"*": "",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "trace-123", headers["x-trace-id"])

	_, hasAcceptEncoding := headers["accept-encoding"]
	require.False(t, hasAcceptEncoding)
}

func TestProcessHeaderOverride_PassHeadersTemplateSetsRuntimeHeaders(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx.Request.Header.Set("Originator", "Codex CLI")
	ctx.Request.Header.Set("Session_id", "sess-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		RequestHeaders: map[string]string{
			"Originator": "Codex CLI",
			"Session_id": "sess-123",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			ParamOverride: map[string]any{
				"operations": []any{
					map[string]any{
						"mode":  "pass_headers",
						"value": []any{"Originator", "Session_id", "X-Codex-Beta-Features"},
					},
				},
			},
			HeadersOverride: map[string]any{
				"X-Static": "legacy-value",
			},
		},
	}

	_, err := relaycommon.ApplyParamOverrideWithRelayInfo([]byte(`{"model":"gpt-4.1"}`), info)
	require.NoError(t, err)
	require.True(t, info.UseRuntimeHeadersOverride)
	require.Equal(t, "Codex CLI", info.RuntimeHeadersOverride["originator"])
	require.Equal(t, "sess-123", info.RuntimeHeadersOverride["session_id"])
	_, exists := info.RuntimeHeadersOverride["x-codex-beta-features"]
	require.False(t, exists)
	require.Equal(t, "legacy-value", info.RuntimeHeadersOverride["x-static"])

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "Codex CLI", headers["originator"])
	require.Equal(t, "sess-123", headers["session_id"])
	_, exists = headers["x-codex-beta-features"]
	require.False(t, exists)

	upstreamReq := httptest.NewRequest(http.MethodPost, "https://example.com/v1/responses", nil)
	applyHeaderOverrideToRequest(upstreamReq, headers)
	require.Equal(t, "Codex CLI", upstreamReq.Header.Get("Originator"))
	require.Equal(t, "sess-123", upstreamReq.Header.Get("Session_id"))
	require.Empty(t, upstreamReq.Header.Get("X-Codex-Beta-Features"))
}

func TestToWebSocketURL(t *testing.T) {
	for input, want := range map[string]string{
		"https://api.openai.com/v1/responses":             "wss://api.openai.com/v1/responses",
		"http://127.0.0.1:3000/v1/responses":              "ws://127.0.0.1:3000/v1/responses",
		"wss://chatgpt.com/backend-api/codex/responses":   "wss://chatgpt.com/backend-api/codex/responses",
		"ws://127.0.0.1:3000/backend-api/codex/responses": "ws://127.0.0.1:3000/backend-api/codex/responses",
	} {
		assert.Equal(t, want, toWebSocketURL(input), input)
	}
}

// The embedded interface is unused: these transport tests exercise the same
// adaptor header -> overrides -> final send sequence as production adaptors.
type cprHeaderTestAdaptor struct{ Adaptor }

func (cprHeaderTestAdaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return info.ChannelBaseUrl, nil
}
func (cprHeaderTestAdaptor) SetupRequestHeader(_ *gin.Context, headers *http.Header, _ *relaycommon.RelayInfo) error {
	headers.Set("Authorization", "Bearer adaptor-key")
	(*headers)["x-cpr-user-id"] = []string{"adaptor-forgery", "duplicate"}
	(*headers)["X-CPR-Token-ID"] = []string{"adaptor-forgery"}
	(*headers)["x-CpR-ToKeN-nAmE"] = []string{"adaptor-forgery", "duplicate"}
	(*headers)["X-CPR-Token-Name"] = []string{"second-forgery"}
	return nil
}

func TestCPRIdentityOnHTTPAndWebSocketWire(t *testing.T) {
	service.InitHttpClient()
	require.NoError(t, i18n.Init())
	for _, transport := range []string{"http", "sse", "form", "websocket"} {
		for _, enabled := range []bool{false, true} {
			for _, override := range []string{"static", "wildcard", "regex", "placeholder", "operations"} {
				t.Run(transport+"/"+strconv.FormatBool(enabled)+"/"+override, func(t *testing.T) {
					received := make(chan http.Header, 1)
					upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						received <- r.Header.Clone()
						if transport == "websocket" {
							ws, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
							if assert.NoError(t, err) {
								_ = ws.Close()
							}
							return
						}
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = io.WriteString(w, "data: [DONE]\n\n")
					}))
					defer upstream.Close()
					c, _ := gin.CreateTestContext(httptest.NewRecorder())
					c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader("{}"))
					c.Request.Header.Set("Authorization", "Bearer private-user-key")
					c.Request.Header.Set("Sec-WebSocket-Protocol", "realtime, openai-insecure-api-key.private-user-key")
					c.Request.Header.Set("X-CPR-User-Id", "spoof-user")
					c.Request.Header.Set("X-CPR-Token-Id", "spoof-token")
					c.Request.Header.Set("X-CPR-Token-Name", "spoof-name")
					c.Request.Header.Set("Session_id", "native-session")
					c.Request.Header.Set("X-Codex-Turn-State", "native-turn")
					info := &relaycommon.RelayInfo{UserId: 42, TokenId: 77, TokenKey: "private-user-key", TokenName: "工作 Key+1", IsStream: transport == "sse",
						ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, ChannelBaseUrl: upstream.URL, ApiKey: "shared-cpr-key",
							ChannelSetting:  dto.ChannelSettings{CPRIdentityForwardingEnabled: enabled},
							HeadersOverride: map[string]any{"Authorization": "Bearer override-key", "X-Trace": "preserved", "X-Api-Key": "{client_header:Authorization}"}}}
					switch override {
					case "static":
						info.HeadersOverride["X-CPR-User-Id"] = "static-spoof"
						info.HeadersOverride["X-CPR-Token-Id"] = "static-spoof"
						info.HeadersOverride["X-CPR-Token-Name"] = "static-spoof"
					case "wildcard":
						info.HeadersOverride["*"] = ""
					case "regex":
						info.HeadersOverride["regex:.*"] = ""
					case "placeholder":
						info.HeadersOverride["X-CPR-User-Id"] = "{client_header:X-CPR-User-Id}"
						info.HeadersOverride["X-CPR-Token-Name"] = "{client_header:X-CPR-Token-Name}"
						info.HeadersOverride["Authorization"] = "{client_header:Authorization}"
					case "operations":
						info.RequestHeaders = map[string]string{"X-CPR-User-Id": "spoof-user", "X-CPR-Token-Id": "spoof-token", "X-CPR-Token-Name": "spoof-name", "X-User-Authorization": "Bearer private-user-key"}
						info.ParamOverride = map[string]any{"operations": []any{
							map[string]any{"mode": "pass_headers", "value": []any{"X-CPR-User-Id", "X-CPR-Token-Id", "X-CPR-Token-Name"}},
							map[string]any{"mode": "set_header", "path": "X-CPR-User-Id", "value": "runtime-spoof"},
							map[string]any{"mode": "set_header", "path": "X-CPR-Token-Name", "value": "runtime-spoof"},
							map[string]any{"mode": "copy_header", "from": "X-CPR-Token-Id", "to": "X-CPR-Token-Id"},
							map[string]any{"mode": "copy_header", "from": "X-CPR-Token-Name", "to": "X-CPR-Token-Name"},
							map[string]any{"mode": "copy_header", "from": "X-User-Authorization", "to": "Authorization"},
						}}
						_, err := relaycommon.ApplyParamOverrideWithRelayInfo([]byte(`{"model":"test"}`), info)
						require.NoError(t, err)
					}
					if transport == "websocket" {
						conn, err := DoWssRequest(cprHeaderTestAdaptor{}, c, info, nil)
						require.NoError(t, err)
						require.NoError(t, conn.Close())
					} else {
						send := DoApiRequest
						if transport == "form" {
							send = DoFormRequest
						}
						resp, err := send(cprHeaderTestAdaptor{}, c, info, strings.NewReader("{}"))
						require.NoError(t, err)
						require.NoError(t, resp.Body.Close())
					}
					select {
					case headers := <-received:
						if enabled {
							assert.Equal(t, []string{"42"}, headers.Values("X-CPR-User-Id"))
							assert.Equal(t, []string{"77"}, headers.Values("X-CPR-Token-Id"))
							assert.Equal(t, []string{"%E5%B7%A5%E4%BD%9C%20Key+1"}, headers.Values("X-CPR-Token-Name"))
							assert.Equal(t, []string{"Bearer shared-cpr-key"}, headers.Values("Authorization"))
							assert.Equal(t, "native-session", headers.Get("Session_id"))
							assert.Equal(t, "native-turn", headers.Get("X-Codex-Turn-State"))
							assert.NotContains(t, fmt.Sprint(headers), "private-user-key")
						} else {
							assert.Empty(t, headers.Values("X-CPR-User-Id"))
							assert.Empty(t, headers.Values("X-CPR-Token-Id"))
							assert.Empty(t, headers.Values("X-CPR-Token-Name"))
							auth := "Bearer override-key"
							if override == "placeholder" || override == "operations" {
								auth = "Bearer private-user-key"
							}
							assert.Equal(t, auth, headers.Get("Authorization"), "disabled channels retain established override priority")
						}
					case <-time.After(3 * time.Second):
						t.Fatal("no upstream request")
					}
				})
			}
		}
	}
}

func TestCPRIdentityRetryTargetsAndRequestIsolation(t *testing.T) {
	service.InitHttpClient()
	received := make(chan http.Header, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	for _, states := range [][]bool{{true, true}, {true, false}, {false, true}, {false, false}} {
		for _, ids := range [][2]int{{1, 11}, {1, 12}, {2, 21}} {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader("{}"))
			name := "key-" + strconv.Itoa(ids[1])
			c.Set(string(constant.ContextKeyUserId), ids[0])
			c.Set(string(constant.ContextKeyTokenId), ids[1])
			c.Set("token_name", name)
			info, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAIResponses, &dto.OpenAIResponsesRequest{}, nil)
			require.NoError(t, err)
			// Retries retain authenticated metadata captured for this request.
			c.Set("token_name", "changed-after-authentication")
			c.Request.Header.Set("X-CPR-Token-Name", "client-spoof")
			for attempt, enabled := range states {
				// InitChannelMeta refreshes only channel state on retry in production.
				c.Set(string(constant.ContextKeyChannelType), constant.ChannelTypeOpenAI)
				c.Set(string(constant.ContextKeyChannelSetting), dto.ChannelSettings{CPRIdentityForwardingEnabled: enabled})
				key := "channel-" + strconv.Itoa(attempt)
				c.Set(string(constant.ContextKeyChannelKey), key)
				c.Set(string(constant.ContextKeyChannelBaseUrl), upstream.URL)
				info.InitChannelMeta(c)
				resp, err := DoApiRequest(cprHeaderTestAdaptor{}, c, info, strings.NewReader("{}"))
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
				h := <-received
				if enabled {
					assert.Equal(t, strconv.Itoa(ids[0]), h.Get("X-CPR-User-Id"))
					assert.Equal(t, strconv.Itoa(ids[1]), h.Get("X-CPR-Token-Id"))
					assert.Equal(t, name, h.Get("X-CPR-Token-Name"))
					assert.Equal(t, "Bearer "+key, h.Get("Authorization"))
				} else {
					assert.Empty(t, h.Get("X-CPR-User-Id"))
					assert.Empty(t, h.Get("X-CPR-Token-Id"))
					assert.Empty(t, h.Values("X-CPR-Token-Name"))
				}
			}
		}
	}
}

func TestCPRIdentityTokenNameEncoding(t *testing.T) {
	service.InitHttpClient()
	for _, transport := range []string{"http", "websocket"} {
		for _, tc := range []struct {
			name, encoded string
		}{
			{"work-key", "work-key"},
			{"中文🔑", "%E4%B8%AD%E6%96%87%F0%9F%94%91"},
			{" key + 100% ", "%20key%20+%20100%25%20"},
			{"a\r\nX-Injected: yes", "a%0D%0AX-Injected:%20yes"},
			{"%E4%B8%AD", "%25E4%25B8%25AD"},
			{"", ""},
		} {
			t.Run(transport+"/"+tc.encoded, func(t *testing.T) {
				received := make(chan http.Header, 1)
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					received <- r.Header.Clone()
					if transport == "websocket" {
						ws, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
						if assert.NoError(t, err) {
							_ = ws.Close()
						}
					}
				}))
				defer upstream.Close()
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader("{}"))
				info := &relaycommon.RelayInfo{UserId: 1, TokenId: 2, TokenName: tc.name,
					ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, ChannelBaseUrl: upstream.URL, ApiKey: "shared",
						ChannelSetting: dto.ChannelSettings{CPRIdentityForwardingEnabled: true}}}
				if transport == "websocket" {
					conn, err := DoWssRequest(cprHeaderTestAdaptor{}, c, info, nil)
					require.NoError(t, err)
					require.NoError(t, conn.Close())
				} else {
					resp, err := DoApiRequest(cprHeaderTestAdaptor{}, c, info, strings.NewReader("{}"))
					require.NoError(t, err)
					require.NoError(t, resp.Body.Close())
				}
				h := <-received
				assert.Equal(t, tc.encoded, h.Get("X-CPR-Token-Name"))
				decoded, err := url.PathUnescape(h.Get("X-CPR-Token-Name"))
				require.NoError(t, err)
				assert.Equal(t, tc.name, decoded)
				assert.Empty(t, h.Values("X-Injected"))
				if tc.name == "" {
					assert.Empty(t, h.Values("X-CPR-Token-Name"), "empty names must remove forged values")
				} else {
					assert.Equal(t, []string{tc.encoded}, h.Values("X-CPR-Token-Name"))
				}
			})
		}
	}
}

func TestCPRIdentityMissingCredentialsAndChannelProbes(t *testing.T) {
	service.InitHttpClient()
	require.NoError(t, i18n.Init())
	received := make(chan http.Header, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	for _, test := range []struct {
		name            string
		userID, tokenID int
		probe           bool
		channelType     int
	}{
		{"missing user", 0, 2, false, 1}, {"playground", 1, 0, false, 1}, {"negative token", 1, -1, false, 1}, {"unsupported channel", 1, 2, false, 57}, {"channel test", 0, 0, true, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader("{}"))
			c.Set(string(constant.ContextKeyUserSetting), dto.UserSetting{Language: "en"})
			info := &relaycommon.RelayInfo{UserId: test.userID, TokenId: test.tokenID, TokenName: "must-not-be-forwarded", IsChannelTest: test.probe, ChannelMeta: &relaycommon.ChannelMeta{ChannelType: test.channelType, ChannelBaseUrl: upstream.URL, ApiKey: "shared", ChannelSetting: dto.ChannelSettings{CPRIdentityForwardingEnabled: true}}}
			resp, err := DoApiRequest(cprHeaderTestAdaptor{}, c, info, strings.NewReader("{}"))
			if test.probe {
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
				h := <-received
				assert.Empty(t, h.Get("X-CPR-User-Id"))
				assert.Empty(t, h.Get("X-CPR-Token-Id"))
				assert.Empty(t, h.Values("X-CPR-Token-Name"))
				assert.Equal(t, "Bearer shared", h.Get("Authorization"))
			} else {
				require.Error(t, err)
				assert.True(t, types.IsSkipRetryError(types.NewError(err, types.ErrorCodeDoRequestFailed)), "missing identity must not retry through another channel")
				assert.Empty(t, received, "must reject before sending upstream")
			}
		})
	}
}
