package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const TokenUsageAggregationVersion = 1

type TokenUsageDaily struct {
	ID                 int64   `json:"id" gorm:"primaryKey"`
	UsageDate          int     `json:"usage_date" gorm:"not null;uniqueIndex:idx_token_usage_daily_unique,priority:2;index:idx_token_usage_daily_user_date,priority:2;index:idx_token_usage_daily_user_token_date,priority:3"`
	UserID             int     `json:"user_id" gorm:"not null;uniqueIndex:idx_token_usage_daily_unique,priority:1;index:idx_token_usage_daily_user_date,priority:1;index:idx_token_usage_daily_user_token_date,priority:1"`
	Username           string  `json:"username" gorm:"type:varchar(64);not null"`
	TokenID            int     `json:"token_id" gorm:"not null;uniqueIndex:idx_token_usage_daily_unique,priority:3;index:idx_token_usage_daily_user_token_date,priority:2"`
	TokenName          string  `json:"token_name" gorm:"type:varchar(191);not null"`
	RequestCount       int64   `json:"request_count" gorm:"not null"`
	InputTokens        int64   `json:"input_tokens" gorm:"not null"`
	OutputTokens       int64   `json:"output_tokens" gorm:"not null"`
	CacheTokens        int64   `json:"cache_tokens" gorm:"not null"`
	TotalTokens        int64   `json:"total_tokens" gorm:"not null"`
	Quota              int64   `json:"quota" gorm:"not null"`
	QuotaPerUnit       float64 `json:"quota_per_unit" gorm:"not null"`
	Cost               float64 `json:"cost" gorm:"type:decimal(20,10);not null"`
	AggregationVersion int     `json:"aggregation_version" gorm:"not null"`
	CreatedAt          int64   `json:"created_at" gorm:"bigint;not null"`
	UpdatedAt          int64   `json:"updated_at" gorm:"bigint;not null"`
}

func (TokenUsageDaily) TableName() string {
	return "token_usage_daily"
}

type TokenUsageQueryMode string

const (
	TokenUsageQueryModeDetail TokenUsageQueryMode = "detail"
	TokenUsageQueryModeDaily  TokenUsageQueryMode = "daily"
	TokenUsageQueryModeToken  TokenUsageQueryMode = "token"
)

type TokenUsageQuery struct {
	UserID    int
	FromDate  int
	ToDate    int
	TokenID   int
	TokenName string
	Mode      TokenUsageQueryMode
	Offset    int
	Limit     int
}

type TokenUsageSummary struct {
	RequestCount int64   `json:"request_count"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CacheTokens  int64   `json:"cache_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	Quota        int64   `json:"quota"`
	Cost         float64 `json:"cost"`
}

type TokenUsageQueryRow struct {
	UsageDate    int     `json:"usage_date"`
	TokenID      int     `json:"token_id"`
	TokenName    string  `json:"token_name"`
	RequestCount int64   `json:"request_count"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CacheTokens  int64   `json:"cache_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	Quota        int64   `json:"quota"`
	Cost         float64 `json:"cost"`
}

type tokenUsageRawLog struct {
	UserID           int
	Username         string
	TokenID          int
	TokenName        string
	PromptTokens     int
	CompletionTokens int
	Quota            int
	Other            string
}

type tokenUsageAggregationKey struct {
	UserID  int
	TokenID int
}

func AggregateTokenUsageDay(ctx context.Context, usageDate int, startTimestamp, endTimestamp int64, quotaPerUnit float64, userID int) (int, error) {
	if usageDate <= 0 || startTimestamp <= 0 || endTimestamp <= startTimestamp {
		return 0, errors.New("invalid token usage aggregation range")
	}
	if quotaPerUnit <= 0 {
		return 0, errors.New("quota per unit must be positive")
	}

	logQuery := LOG_DB.WithContext(ctx).
		Model(&Log{}).
		Select("user_id, username, token_id, token_name, prompt_tokens, completion_tokens, quota, other").
		Where("type = ? AND created_at >= ? AND created_at < ?", LogTypeConsume, startTimestamp, endTimestamp).
		Order("created_at asc")
	if userID > 0 {
		logQuery = logQuery.Where("user_id = ?", userID)
	}
	rows, err := logQuery.Rows()
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	now := common.GetTimestamp()
	aggregates := make(map[tokenUsageAggregationKey]*TokenUsageDaily)
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		var raw tokenUsageRawLog
		if err := LOG_DB.ScanRows(rows, &raw); err != nil {
			return 0, err
		}

		key := tokenUsageAggregationKey{UserID: raw.UserID, TokenID: raw.TokenID}
		aggregate := aggregates[key]
		if aggregate == nil {
			aggregate = &TokenUsageDaily{
				UsageDate:          usageDate,
				UserID:             raw.UserID,
				Username:           raw.Username,
				TokenID:            raw.TokenID,
				TokenName:          raw.TokenName,
				QuotaPerUnit:       quotaPerUnit,
				AggregationVersion: TokenUsageAggregationVersion,
				CreatedAt:          now,
				UpdatedAt:          now,
			}
			aggregates[key] = aggregate
		} else {
			if raw.Username != "" {
				aggregate.Username = raw.Username
			}
			if raw.TokenName != "" {
				aggregate.TokenName = raw.TokenName
			}
		}

		inputTokens := int64(raw.PromptTokens)
		if inputTokens < 0 {
			inputTokens = 0
		}
		outputTokens := int64(raw.CompletionTokens)
		if outputTokens < 0 {
			outputTokens = 0
		}
		quota := int64(raw.Quota)
		if quota < 0 {
			quota = 0
		}

		aggregate.RequestCount++
		aggregate.InputTokens += inputTokens
		aggregate.OutputTokens += outputTokens
		aggregate.TotalTokens += inputTokens + outputTokens
		aggregate.Quota += quota
		aggregate.CacheTokens += cacheTokensFromLogOther(raw.Other)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	items := make([]*TokenUsageDaily, 0, len(aggregates))
	for _, aggregate := range aggregates {
		aggregate.Cost = float64(aggregate.Quota) / quotaPerUnit
		items = append(items, aggregate)
	}

	err = DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		deleteQuery := tx.Where("usage_date = ?", usageDate)
		if userID > 0 {
			deleteQuery = deleteQuery.Where("user_id = ?", userID)
		}
		if err := deleteQuery.Delete(&TokenUsageDaily{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return tx.CreateInBatches(items, 200).Error
	})
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func cacheTokensFromLogOther(other string) int64 {
	if other == "" {
		return 0
	}
	var values map[string]interface{}
	if err := common.UnmarshalJsonStr(other, &values); err != nil {
		return 0
	}
	value, ok := values["cache_tokens"]
	if !ok {
		return 0
	}

	var tokens int64
	switch typed := value.(type) {
	case float64:
		tokens = int64(typed)
	case float32:
		tokens = int64(typed)
	case int:
		tokens = int64(typed)
	case int64:
		tokens = typed
	case string:
		_, _ = fmt.Sscan(typed, &tokens)
	}
	if tokens < 0 {
		return 0
	}
	return tokens
}

func QueryTokenUsageDaily(query TokenUsageQuery) ([]TokenUsageQueryRow, int64, TokenUsageSummary, error) {
	filtered := func() *gorm.DB {
		base := DB.Model(&TokenUsageDaily{}).
			Where("user_id = ? AND usage_date >= ? AND usage_date <= ?", query.UserID, query.FromDate, query.ToDate)
		if query.TokenID > 0 {
			base = base.Where("token_id = ?", query.TokenID)
		}
		if query.TokenName != "" {
			escaped := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(query.TokenName)
			base = base.Where("token_name LIKE ? ESCAPE '!'", "%"+escaped+"%")
		}
		return base
	}

	var summary TokenUsageSummary
	if err := filtered().
		Select("COALESCE(SUM(request_count), 0) AS request_count, COALESCE(SUM(input_tokens), 0) AS input_tokens, COALESCE(SUM(output_tokens), 0) AS output_tokens, COALESCE(SUM(cache_tokens), 0) AS cache_tokens, COALESCE(SUM(total_tokens), 0) AS total_tokens, COALESCE(SUM(quota), 0) AS quota, COALESCE(SUM(cost), 0) AS cost").
		Scan(&summary).Error; err != nil {
		return nil, 0, TokenUsageSummary{}, err
	}

	selectTotals := "SUM(request_count) AS request_count, SUM(input_tokens) AS input_tokens, SUM(output_tokens) AS output_tokens, SUM(cache_tokens) AS cache_tokens, SUM(total_tokens) AS total_tokens, SUM(quota) AS quota, SUM(cost) AS cost"
	rows := make([]TokenUsageQueryRow, 0)
	var total int64
	switch query.Mode {
	case TokenUsageQueryModeDaily:
		if err := filtered().Distinct("usage_date").Count(&total).Error; err != nil {
			return nil, 0, TokenUsageSummary{}, err
		}
		grouped := filtered().
			Select("usage_date, 0 AS token_id, '' AS token_name, " + selectTotals).
			Group("usage_date").
			Order("usage_date desc")
		if query.Limit > 0 {
			grouped = grouped.Offset(query.Offset).Limit(query.Limit)
		}
		err := grouped.Scan(&rows).Error
		return rows, total, summary, err
	case TokenUsageQueryModeToken:
		if err := filtered().Distinct("token_id").Count(&total).Error; err != nil {
			return nil, 0, TokenUsageSummary{}, err
		}
		grouped := filtered().
			Select("0 AS usage_date, token_id, MAX(token_name) AS token_name, " + selectTotals).
			Group("token_id").
			Order("SUM(cost) desc, token_id asc")
		if query.Limit > 0 {
			grouped = grouped.Offset(query.Offset).Limit(query.Limit)
		}
		err := grouped.Scan(&rows).Error
		return rows, total, summary, err
	default:
		if err := filtered().Count(&total).Error; err != nil {
			return nil, 0, TokenUsageSummary{}, err
		}
		detailQuery := filtered().
			Select("usage_date, token_id, token_name, request_count, input_tokens, output_tokens, cache_tokens, total_tokens, quota, cost").
			Order("usage_date desc, token_id asc")
		if query.Limit > 0 {
			detailQuery = detailQuery.Offset(query.Offset).Limit(query.Limit)
		}
		if err := detailQuery.Scan(&rows).Error; err != nil {
			return nil, 0, TokenUsageSummary{}, err
		}
		return rows, total, summary, nil
	}
}

func TokenUsageDateKey(date time.Time) int {
	year, month, day := date.Date()
	return year*10000 + int(month)*100 + day
}
