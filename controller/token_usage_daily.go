package controller

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetTokenUsageDailySelf(c *gin.Context) {
	from, err := service.ParseTokenUsageDate(c.Query("from"))
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	to, err := service.ParseTokenUsageDate(c.Query("to"))
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if err := service.ValidateTokenUsageDateRange(from, to, service.TokenUsageMaxQueryDays, false); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	mode := model.TokenUsageQueryMode(c.DefaultQuery("group_by", string(model.TokenUsageQueryModeDetail)))
	if mode != model.TokenUsageQueryModeDetail && mode != model.TokenUsageQueryModeDaily && mode != model.TokenUsageQueryModeToken {
		common.ApiErrorMsg(c, "group_by must be detail, daily, or token")
		return
	}
	tokenID, err := strconv.Atoi(c.Query("token_id"))
	if err != nil && c.Query("token_id") != "" {
		common.ApiErrorMsg(c, "token_id must be an integer")
		return
	}

	pageInfo := common.GetPageQuery(c)
	rows, total, summary, err := model.QueryTokenUsageDaily(model.TokenUsageQuery{
		UserID:    c.GetInt("id"),
		FromDate:  model.TokenUsageDateKey(from),
		ToDate:    model.TokenUsageDateKey(to),
		TokenID:   tokenID,
		TokenName: c.Query("token_name"),
		Mode:      mode,
		Offset:    pageInfo.GetStartIdx(),
		Limit:     pageInfo.GetPageSize(),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(rows)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"page":    pageInfo,
			"summary": summary,
		},
	})
}

func CreateTokenUsageDailyBackfillTask(c *gin.Context) {
	var payload service.TokenUsageDailyTaskPayload
	if err := common.DecodeJson(c.Request.Body, &payload); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	task, created, err := service.StartTokenUsageDailyTask(payload)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"created": created,
			"task":    task.ToResponse(),
		},
	})
}

type tokenUsageDailyHandler struct{}

func (tokenUsageDailyHandler) Type() string { return model.SystemTaskTypeTokenUsageDaily }

func (tokenUsageDailyHandler) Enabled() bool {
	if !common.LogConsumeEnabled || !common.GetEnvOrDefaultBool("TOKEN_USAGE_DAILY_TASK_ENABLED", false) {
		return false
	}
	hour := common.GetEnvOrDefault("TOKEN_USAGE_DAILY_TASK_HOUR", 3)
	if hour < 0 || hour > 23 {
		hour = 3
	}
	return time.Now().In(service.TokenUsageLocation).Hour() == hour
}

// A 23-hour interval leaves enough room for scheduler jitter while Enabled
// still restricts creation to the configured local-hour window.
func (tokenUsageDailyHandler) Interval() time.Duration { return 23 * time.Hour }

func (tokenUsageDailyHandler) NewPayload() any { return service.TokenUsageScheduledPayload() }

func (tokenUsageDailyHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	payload := service.TokenUsageDailyTaskPayload{}
	if err := task.DecodePayload(&payload); err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	result, err := service.RunTokenUsageDailyAggregation(ctx, payload, service.NewSystemTaskProgressReporter(task, runnerID))
	if err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, result, nil)
}
