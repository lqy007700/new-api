package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	TokenUsageMaxQueryDays    = 366
	TokenUsageMaxBackfillDays = 31
)

var TokenUsageLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type TokenUsageDailyTaskPayload struct {
	From   string `json:"from"`
	To     string `json:"to"`
	UserID int    `json:"user_id,omitempty"`
}

type TokenUsageDailyTaskResult struct {
	From           string `json:"from"`
	To             string `json:"to"`
	ProcessedDays  int    `json:"processed_days"`
	AggregatedRows int    `json:"aggregated_rows"`
}

func ParseTokenUsageDate(value string) (time.Time, error) {
	date, err := time.ParseInLocation("2006-01-02", value, TokenUsageLocation)
	if err != nil {
		return time.Time{}, errors.New("date must use YYYY-MM-DD format")
	}
	return date, nil
}

func ValidateTokenUsageDateRange(from, to time.Time, maxDays int, requireCompletedDay bool) error {
	if to.Before(from) {
		return errors.New("end date must not be before start date")
	}
	days := int(to.Sub(from).Hours()/24) + 1
	if days > maxDays {
		return fmt.Errorf("date range cannot exceed %d days", maxDays)
	}
	if requireCompletedDay {
		today := startOfTokenUsageDay(time.Now().In(TokenUsageLocation))
		if !to.Before(today) {
			return errors.New("end date must be a completed day")
		}
	}
	return nil
}

func TokenUsageScheduledPayload() TokenUsageDailyTaskPayload {
	today := startOfTokenUsageDay(time.Now().In(TokenUsageLocation))
	return TokenUsageDailyTaskPayload{
		From: today.AddDate(0, 0, -2).Format("2006-01-02"),
		To:   today.AddDate(0, 0, -1).Format("2006-01-02"),
	}
}

func StartTokenUsageDailyTask(payload TokenUsageDailyTaskPayload) (*model.SystemTask, bool, error) {
	from, err := ParseTokenUsageDate(payload.From)
	if err != nil {
		return nil, false, err
	}
	to, err := ParseTokenUsageDate(payload.To)
	if err != nil {
		return nil, false, err
	}
	if err := ValidateTokenUsageDateRange(from, to, TokenUsageMaxBackfillDays, true); err != nil {
		return nil, false, err
	}
	return EnqueueSystemTask(model.SystemTaskTypeTokenUsageDaily, payload)
}

func RunTokenUsageDailyAggregation(ctx context.Context, payload TokenUsageDailyTaskPayload, progress func(processed, total int)) (TokenUsageDailyTaskResult, error) {
	from, err := ParseTokenUsageDate(payload.From)
	if err != nil {
		return TokenUsageDailyTaskResult{}, err
	}
	to, err := ParseTokenUsageDate(payload.To)
	if err != nil {
		return TokenUsageDailyTaskResult{}, err
	}
	if err := ValidateTokenUsageDateRange(from, to, TokenUsageMaxBackfillDays, true); err != nil {
		return TokenUsageDailyTaskResult{}, err
	}
	if !common.LogConsumeEnabled {
		return TokenUsageDailyTaskResult{}, errors.New("consume logs are disabled")
	}
	if common.QuotaPerUnit <= 0 {
		return TokenUsageDailyTaskResult{}, errors.New("quota per unit must be positive")
	}

	totalDays := int(to.Sub(from).Hours()/24) + 1
	result := TokenUsageDailyTaskResult{From: payload.From, To: payload.To}
	quotaPerUnit := common.QuotaPerUnit
	for date := from; !date.After(to); date = date.AddDate(0, 0, 1) {
		if err := ctx.Err(); err != nil {
			return TokenUsageDailyTaskResult{}, err
		}
		rowCount, err := model.AggregateTokenUsageDay(
			ctx,
			model.TokenUsageDateKey(date),
			date.Unix(),
			date.AddDate(0, 0, 1).Unix(),
			quotaPerUnit,
			payload.UserID,
		)
		if err != nil {
			return TokenUsageDailyTaskResult{}, err
		}
		result.ProcessedDays++
		result.AggregatedRows += rowCount
		if progress != nil {
			progress(result.ProcessedDays, totalDays)
		}
	}
	return result, nil
}

func startOfTokenUsageDay(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, TokenUsageLocation)
}
