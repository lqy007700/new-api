package model

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSumUsedQuotaKeepsQuotaWhenCalculatingRecentRPMAndTPM(t *testing.T) {
	previousLogDB := LOG_DB
	logDB, err := gorm.Open(sqlite.Open("file:log_stats_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	LOG_DB = logDB
	t.Cleanup(func() { LOG_DB = previousLogDB })
	require.NoError(t, LOG_DB.AutoMigrate(&Log{}))

	now := time.Now().Unix()
	require.NoError(t, LOG_DB.Create(&Log{
		CreatedAt:        now - 10,
		Type:             LogTypeConsume,
		Quota:            7,
		PromptTokens:     3,
		CompletionTokens: 4,
	}).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		CreatedAt: now - 120,
		Type:      LogTypeConsume,
		Quota:     5,
	}).Error)

	stat, err := SumUsedQuota(0, now-180, now+1, "", "", "", 0, "")
	require.NoError(t, err)
	require.Equal(t, Stat{Quota: 12, Rpm: 1, Tpm: 7}, stat)
}
