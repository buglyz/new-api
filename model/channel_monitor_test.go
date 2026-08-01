package model

import (
	"strconv"
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

func TestListChannelMonitorAvailabilityAggregatesHourlyBuckets(t *testing.T) {
	truncateTables(t)
	now := common.GetTimestamp()
	bucketSize := int64(time.Hour.Seconds())
	currentBucket := now - now%bucketSize

	for _, result := range []ChannelMonitorResult{
		{ChannelID: 4, Model: "model-a", Status: ChannelMonitorStatusSuccess, CreatedAt: now - 10},
		{ChannelID: 4, Model: "model-a", Status: ChannelMonitorStatusFailure, CreatedAt: now - 20},
		{ChannelID: 4, Model: "model-b", Status: ChannelMonitorStatusSuccess, CreatedAt: currentBucket - bucketSize + 10},
		{ChannelID: 5, Model: "model-c", Status: ChannelMonitorStatusFailure, CreatedAt: now - 30},
	} {
		_, err := CreateChannelMonitorResult(result, 1, 10)
		require.NoError(t, err)
	}

	stats, err := ListChannelMonitorAvailability(currentBucket-2*bucketSize, bucketSize)
	require.NoError(t, err)
	byKey := make(map[string]ChannelMonitorAvailabilityStat, len(stats))
	for _, stat := range stats {
		byKey[channelMonitorTargetKey(stat.ChannelID, stat.Model+"#"+strconv.FormatInt(stat.BucketStart, 10))] = stat
	}

	currentChannel := byKey[channelMonitorTargetKey(4, "model-a#"+strconv.FormatInt(currentBucket, 10))]
	assert.Equal(t, int64(2), currentChannel.Total)
	assert.Equal(t, int64(1), currentChannel.Succeeded)
	previousChannel := byKey[channelMonitorTargetKey(4, "model-b#"+strconv.FormatInt(currentBucket-bucketSize, 10))]
	assert.Equal(t, int64(1), previousChannel.Total)
	assert.Equal(t, int64(1), previousChannel.Succeeded)
	otherChannel := byKey[channelMonitorTargetKey(5, "model-c#"+strconv.FormatInt(currentBucket, 10))]
	assert.Equal(t, int64(1), otherChannel.Total)
	assert.Zero(t, otherChannel.Succeeded)
}

func TestChannelMonitorPruningEnforcesLimitForRecentManualRuns(t *testing.T) {
	truncateTables(t)
	now := common.GetTimestamp()
	for index := range 4 {
		_, err := CreateChannelMonitorResult(ChannelMonitorResult{
			ChannelID: 3, ChannelName: "manual", Model: "manual-model",
			Status: ChannelMonitorStatusSuccess, CreatedAt: now - int64(index),
		}, 1, 3)
		require.NoError(t, err)
	}

	history, err := ListChannelMonitorHistory(3, "manual-model", 10)
	require.NoError(t, err)
	assert.Len(t, history, 3)
}

func TestDeleteStaleChannelMonitorTargetsRemovesOnlyUnknownTargets(t *testing.T) {
	truncateTables(t)
	for _, target := range []ChannelMonitorResult{
		{ChannelID: 10, ChannelName: "known", Model: "model-a", Status: ChannelMonitorStatusSuccess},
		{ChannelID: 11, ChannelName: "removed", Model: "model-b", Status: ChannelMonitorStatusFailure},
	} {
		_, err := CreateChannelMonitorResult(target, 1, 10)
		require.NoError(t, err)
	}

	require.NoError(t, DeleteStaleChannelMonitorTargets([]ChannelMonitorTargetRef{{ChannelID: 10, Model: "model-a"}}))
	targets, err := ListChannelMonitorTargets(0)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, 10, targets[0].ChannelID)
	assert.Equal(t, "model-a", targets[0].Model)
}
