package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordConsumeLogIncrementsChannelRequestCountWhenLogsDisabled(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 1, Name: "counted"}).Error)

	previous := common.LogConsumeEnabled
	common.LogConsumeEnabled = false
	t.Cleanup(func() { common.LogConsumeEnabled = previous })

	RecordConsumeLog(nil, 1, RecordConsumeLogParams{ChannelId: 1})

	var channel Channel
	require.NoError(t, DB.First(&channel, 1).Error)
	assert.Equal(t, int64(1), channel.RequestCount)

	var logCount int64
	require.NoError(t, LOG_DB.Model(&Log{}).Count(&logCount).Error)
	assert.Zero(t, logCount)
}

func TestIncrementChannelRequestCountIgnoresInvalidChannelIDs(t *testing.T) {
	truncateTables(t)

	IncrementChannelRequestCount(0)
	IncrementChannelRequestCount(-1)

	var channelCount int64
	require.NoError(t, DB.Model(&Channel{}).Count(&channelCount).Error)
	assert.Zero(t, channelCount)
}
