package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelMonitorResultTracksHealthAndPrunesHistory(t *testing.T) {
	truncateTables(t)
	now := common.GetTimestamp()

	first, err := CreateChannelMonitorResult(ChannelMonitorResult{
		ChannelID: 1, ChannelName: "fake", Model: "test-model",
		Status: ChannelMonitorStatusFailure, Error: "first failure", CreatedAt: now - int64((25 * time.Hour).Seconds()),
	}, 2, 3)
	require.NoError(t, err)
	assert.Equal(t, ChannelMonitorHealthDegraded, first.Health)
	assert.False(t, first.StateChanged)

	second, err := CreateChannelMonitorResult(ChannelMonitorResult{
		ChannelID: 1, ChannelName: "fake", Model: "test-model",
		Status: ChannelMonitorStatusFailure, Error: "second failure", CreatedAt: now - 30,
	}, 2, 3)
	require.NoError(t, err)
	assert.Equal(t, ChannelMonitorHealthDown, second.Health)
	assert.True(t, second.StateChanged)

	third, err := CreateChannelMonitorResult(ChannelMonitorResult{
		ChannelID: 1, ChannelName: "fake", Model: "test-model",
		Status: ChannelMonitorStatusSuccess, LatencyMS: 12, CreatedAt: now - 20,
	}, 2, 3)
	require.NoError(t, err)
	assert.Equal(t, ChannelMonitorHealthHealthy, third.Health)
	assert.True(t, third.StateChanged)

	_, err = CreateChannelMonitorResult(ChannelMonitorResult{
		ChannelID: 1, ChannelName: "fake", Model: "test-model",
		Status: ChannelMonitorStatusSuccess, LatencyMS: 9, CreatedAt: now - 10,
	}, 2, 3)
	require.NoError(t, err)

	history, err := ListChannelMonitorHistory(1, "test-model", 10)
	require.NoError(t, err)
	require.Len(t, history, 3)
	assert.Equal(t, ChannelMonitorStatusSuccess, history[0].Status)
	assert.Equal(t, ChannelMonitorStatusSuccess, history[1].Status)
	assert.Equal(t, ChannelMonitorStatusFailure, history[2].Status)

	targets, err := ListChannelMonitorTargets(now - int64((24 * time.Hour).Seconds()))
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, ChannelMonitorHealthHealthy, targets[0].Health)
	assert.Equal(t, int64(3), targets[0].Samples24H)
	assert.InDelta(t, 2.0/3.0, targets[0].SuccessRate24H, 0.001)
}

func TestChannelMonitorTargetsUseSecondTimestampsForStats(t *testing.T) {
	truncateTables(t)
	now := common.GetTimestamp()

	_, err := CreateChannelMonitorResult(ChannelMonitorResult{
		ChannelID: 2, ChannelName: "fake", Model: "window-model",
		Status: ChannelMonitorStatusFailure, CreatedAt: now - int64((25 * time.Hour).Seconds()),
	}, 1, 10)
	require.NoError(t, err)
	_, err = CreateChannelMonitorResult(ChannelMonitorResult{
		ChannelID: 2, ChannelName: "fake", Model: "window-model",
		Status: ChannelMonitorStatusSuccess, CreatedAt: now - 20,
	}, 1, 10)
	require.NoError(t, err)
	_, err = CreateChannelMonitorResult(ChannelMonitorResult{
		ChannelID: 2, ChannelName: "fake", Model: "window-model",
		Status: ChannelMonitorStatusSuccess, CreatedAt: now - 10,
	}, 1, 10)
	require.NoError(t, err)

	targets, err := ListChannelMonitorTargets(now - int64((24 * time.Hour).Seconds()))
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, int64(2), targets[0].Samples24H)
	assert.InDelta(t, 1.0, targets[0].SuccessRate24H, 0.001)
}
