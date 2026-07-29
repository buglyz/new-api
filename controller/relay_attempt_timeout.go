package controller

import (
	"context"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

func runRelayAttempt(c *gin.Context, info *relaycommon.RelayInfo, run func() *types.NewAPIError) *types.NewAPIError {
	if c == nil || c.Request == nil || info == nil || info.IsStream || common.RelayNonStreamTimeout <= 0 {
		return run()
	}

	originalRequest := c.Request
	ctx, cancel := context.WithTimeout(originalRequest.Context(), time.Duration(common.RelayNonStreamTimeout)*time.Second)
	c.Request = originalRequest.WithContext(ctx)
	defer func() {
		c.Request = originalRequest
		cancel()
	}()

	return run()
}
