package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAggregateTokenUsageDayIncludesSuccessfulLogsAndReplacesExistingRows(t *testing.T) {
	previousDB, previousLogDB := DB, LOG_DB
	mainDB, err := gorm.Open(sqlite.Open("file:token_usage_daily_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	logDB, err := gorm.Open(sqlite.Open("file:token_usage_daily_log_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = mainDB, logDB
	t.Cleanup(func() { DB, LOG_DB = previousDB, previousLogDB })
	require.NoError(t, DB.AutoMigrate(&TokenUsageDaily{}))
	require.NoError(t, LOG_DB.AutoMigrate(&Log{}))

	date := 20260811
	require.NoError(t, LOG_DB.Create(&Log{UserId: 2, Username: "hanzhigang", TokenId: 7, TokenName: "primary", CreatedAt: 1000, Type: LogTypeConsume, PromptTokens: 10, CompletionTokens: 5, Quota: 500, Other: `{"cache_tokens":3}`}).Error)
	require.NoError(t, LOG_DB.Create(&Log{UserId: 2, Username: "hanzhigang", TokenId: 7, TokenName: "primary", CreatedAt: 1001, Type: LogTypeRefund, PromptTokens: 100, CompletionTokens: 100, Quota: 1000, Other: `{"cache_tokens":99}`}).Error)
	require.NoError(t, LOG_DB.Create(&Log{UserId: 9, Username: "other", TokenId: 7, TokenName: "primary", CreatedAt: 1002, Type: LogTypeError, PromptTokens: 100, CompletionTokens: 100, Quota: 1000}).Error)
	require.NoError(t, LOG_DB.Create(&Log{UserId: 2, Username: "hanzhigang", TokenId: 7, TokenName: "renamed", CreatedAt: 1003, Type: LogTypeConsume, PromptTokens: 2, CompletionTokens: 3, Quota: 500, Other: "not-json"}).Error)
	require.NoError(t, DB.Create(&TokenUsageDaily{UsageDate: date, UserID: 2, TokenID: 99, TokenName: "stale", RequestCount: 1}).Error)

	rows, err := AggregateTokenUsageDay(context.Background(), date, 1000, 2000, 500000, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, rows)

	var usage TokenUsageDaily
	require.NoError(t, DB.Where("usage_date = ? AND user_id = ? AND token_id = ?", date, 2, 7).First(&usage).Error)
	assert.Equal(t, "renamed", usage.TokenName)
	assert.Equal(t, int64(2), usage.RequestCount)
	assert.Equal(t, int64(12), usage.InputTokens)
	assert.Equal(t, int64(8), usage.OutputTokens)
	assert.Equal(t, int64(3), usage.CacheTokens)
	assert.Equal(t, int64(20), usage.TotalTokens)
	assert.Equal(t, int64(1000), usage.Quota)
	assert.InDelta(t, 0.002, usage.Cost, 0.000001)

	var count int64
	require.NoError(t, DB.Model(&TokenUsageDaily{}).Where("usage_date = ?", date).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestCacheTokensFromLogOtherHandlesInvalidValues(t *testing.T) {
	assert.Equal(t, int64(12), cacheTokensFromLogOther(`{"cache_tokens":12}`))
	assert.Equal(t, int64(0), cacheTokensFromLogOther(`{"cache_tokens":-1}`))
	assert.Equal(t, int64(0), cacheTokensFromLogOther(`{"cache_tokens":"bad"}`))
	assert.Equal(t, int64(0), cacheTokensFromLogOther("not-json"))
}

func TestAggregateTokenUsageDayCanRefreshOneUserWithoutReplacingOthers(t *testing.T) {
	previousDB, previousLogDB := DB, LOG_DB
	mainDB, err := gorm.Open(sqlite.Open("file:token_usage_daily_user_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	logDB, err := gorm.Open(sqlite.Open("file:token_usage_daily_user_log_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = mainDB, logDB
	t.Cleanup(func() { DB, LOG_DB = previousDB, previousLogDB })
	require.NoError(t, DB.AutoMigrate(&TokenUsageDaily{}))
	require.NoError(t, LOG_DB.AutoMigrate(&Log{}))

	require.NoError(t, DB.Create(&TokenUsageDaily{UsageDate: 20260811, UserID: 9, TokenID: 3, TokenName: "other", RequestCount: 5}).Error)
	require.NoError(t, LOG_DB.Create(&Log{UserId: 2, Username: "hanzhigang", TokenId: 7, TokenName: "primary", CreatedAt: 1000, Type: LogTypeConsume, PromptTokens: 10}).Error)

	_, err = AggregateTokenUsageDay(context.Background(), 20260811, 1000, 2000, 500000, 2)
	require.NoError(t, err)

	var other TokenUsageDaily
	require.NoError(t, DB.Where("usage_date = ? AND user_id = ?", 20260811, 9).First(&other).Error)
	assert.Equal(t, int64(5), other.RequestCount)
}
