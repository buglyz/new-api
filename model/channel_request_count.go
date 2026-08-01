package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// IncrementChannelRequestCount records one completed request for a channel.
// It runs before the consume-log feature flag so the counter remains useful
// when detailed consume logs are disabled.
func IncrementChannelRequestCount(channelID int) {
	if channelID <= 0 {
		return
	}

	err := DB.Model(&Channel{}).
		Where("id = ?", channelID).
		Update("request_count", gorm.Expr("request_count + ?", 1)).Error
	if err != nil {
		common.SysLog(fmt.Sprintf(
			"failed to update channel request count: channel_id=%d, error=%v",
			channelID,
			err,
		))
	}
}
