package controller

import (
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

func shouldRetryRelayAttempt(c *gin.Context, info *relaycommon.RelayInfo, relayErr *types.NewAPIError, retryTimes int) bool {
	if info != nil && info.ReceivedResponseCount > 0 {
		return false
	}
	return shouldRetry(c, relayErr, retryTimes)
}
