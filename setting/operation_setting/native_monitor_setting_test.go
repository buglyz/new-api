package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNativeMonitorRetentionLimitCoversTwentyFourHours(t *testing.T) {
	assert.Equal(t, 1441, NativeMonitorRetentionLimit(1))
	assert.Equal(t, 289, NativeMonitorRetentionLimit(5))
	assert.Equal(t, NativeMonitorHistoryLimit, NativeMonitorRetentionLimit(10))
}
