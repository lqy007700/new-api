package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	pluginruntime "github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

func abortWithOpenAiMessage(c *gin.Context, statusCode int, message string, code ...types.ErrorCode) {
	codeStr := ""
	if len(code) > 0 {
		codeStr = string(code[0])
	}
	userId := c.GetInt("id")
	_, preparedPluginRoute := c.Get(pluginruntime.ContextKeyRouteRequest)
	_, pinnedPluginRoute := c.Get(pluginruntime.ContextKeyPinnedRoute)
	_, pinnedPluginEndpoint := c.Get(pluginruntime.ContextKeyPinnedEndpoint)
	if preparedPluginRoute && RespondTaskPluginError(c, &dto.TaskError{
		Code:       codeStr,
		Message:    message,
		StatusCode: statusCode,
	}) {
		c.Abort()
		logger.LogError(c.Request.Context(), fmt.Sprintf("user %d | %s", userId, message))
		return
	}
	if pinnedPluginRoute || pinnedPluginEndpoint {
		c.JSON(statusCode, gin.H{
			"error": gin.H{
				"message": common.MessageWithRequestId(message, c.GetString(common.RequestIdKey)),
				"type":    "new_api_error",
				"code":    codeStr,
			},
		})
		c.Abort()
		logger.LogError(c.Request.Context(), fmt.Sprintf("user %d | %s", userId, message))
		return
	}
	isRelayClientPath := strings.HasPrefix(c.Request.URL.Path, "/v1") || strings.HasPrefix(c.Request.URL.Path, "/v1beta")
	if isRelayClientPath {
		// Relay clients receive only the documented public status families.
		// Permission failures are represented as 401 because the public
		// contract describes both invalid and unauthorized tokens together.
		if statusCode == http.StatusForbidden {
			statusCode = http.StatusUnauthorized
		} else if statusCode != http.StatusUnauthorized && statusCode != http.StatusPaymentRequired &&
			statusCode != http.StatusTooManyRequests && (statusCode < 500 || statusCode > 599) {
			statusCode = http.StatusInternalServerError
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusPaymentRequired ||
		statusCode == http.StatusTooManyRequests || statusCode >= 500 {
		internalCode := types.ErrorCodeBadResponse
		if len(code) > 0 {
			internalCode = code[0]
		}
		err := types.NewErrorWithStatusCode(fmt.Errorf("%s", message), internalCode, statusCode)
		c.JSON(err.ClientStatusCode(), gin.H{"error": err.ToClientOpenAIError(c.GetString(common.RequestIdKey))})
	} else {
		c.JSON(statusCode, gin.H{
			"error": gin.H{
				"message": common.MessageWithRequestId(message, c.GetString(common.RequestIdKey)),
				"type":    "new_api_error",
				"code":    codeStr,
			},
		})
	}
	c.Abort()
	logger.LogError(c.Request.Context(), common.LogText("user %d | %s", userId, message))
}

// invalidRequestStatus is the status of a request whose body could not be
// read or parsed: 413 when the body exceeds MAX_REQUEST_BODY_MB, else 400.
func invalidRequestStatus(err error) int {
	if common.IsRequestBodyTooLargeError(err) {
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}

func abortWithMidjourneyMessage(c *gin.Context, statusCode int, code int, description string) {
	c.JSON(statusCode, gin.H{
		"description": description,
		"type":        "new_api_error",
		"code":        code,
	})
	c.Abort()
	logger.LogError(c.Request.Context(), description)
}
