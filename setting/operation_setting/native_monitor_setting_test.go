package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNativeMonitorRetentionLimitCoversTwentyFourHours(t *testing.T) {
	assert.Equal(t, 1441, NativeMonitorRetentionLimit(1))
	assert.Equal(t, 289, NativeMonitorRetentionLimit(5))
	assert.Equal(t, NativeMonitorHistoryLimit, NativeMonitorRetentionLimit(10))
}

func TestNormalizeNativeMonitorSettingReturnsNonNilEmptyPatterns(t *testing.T) {
	setting, err := NormalizeNativeMonitorSetting(NativeMonitorSetting{
		IntervalMinutes: 10, Concurrency: 1, TimeoutSeconds: 5, FailureThreshold: 1,
	})
	require.NoError(t, err)
	assert.NotNil(t, setting.ExcludePatterns)
	assert.Empty(t, setting.ExcludePatterns)
	assert.NotNil(t, setting.ExcludeChannelIDs)
	assert.Empty(t, setting.ExcludeChannelIDs)
}

func TestValidateNativeMonitorSettingValuesRejectsUnknownAndUnsafeFields(t *testing.T) {
	assert.Error(t, ValidateNativeMonitorSettingValues(map[string]string{
		"unknown": "1",
	}))
	assert.Error(t, ValidateNativeMonitorSettingValues(map[string]string{
		"timeout_seconds": "121",
	}))
	assert.Error(t, ValidateNativeMonitorSettingValues(map[string]string{
		"exclude_patterns": `["["]`,
	}))
	assert.Error(t, ValidateNativeMonitorSettingValues(map[string]string{
		"exclude_channel_ids": `[0, 2]`,
	}))
}
