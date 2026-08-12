package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTokenUsageDateRangeRejectsCurrentAndBroadRanges(t *testing.T) {
	from, err := ParseTokenUsageDate("2026-08-01")
	require.NoError(t, err)
	to, err := ParseTokenUsageDate("2026-08-31")
	require.NoError(t, err)
	assert.NoError(t, ValidateTokenUsageDateRange(from, to, 31, false))
	assert.Error(t, ValidateTokenUsageDateRange(from, to, 30, false))

	today := startOfTokenUsageDay(time.Now().In(TokenUsageLocation))
	assert.Error(t, ValidateTokenUsageDateRange(today, today, 31, true))
}

func TestParseTokenUsageDateUsesShanghaiDayKey(t *testing.T) {
	date, err := ParseTokenUsageDate("2026-08-11")
	require.NoError(t, err)
	assert.Equal(t, 20260811, date.Year()*10000+int(date.Month())*100+date.Day())
	assert.Equal(t, TokenUsageLocation, date.Location())
}
