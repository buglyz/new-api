package service

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

// LogTaskConsumption records task usage without pricing or quota mutation.
func LogTaskConsumption(c *gin.Context, info *relaycommon.RelayInfo) {
	tokenName := c.GetString("token_name")
	logContent := fmt.Sprintf("操作 %s", info.Action)
	other := make(map[string]interface{})
	other["is_task"] = true
	other["request_path"] = c.Request.URL.Path
	if info.IsModelMapped {
		other["is_model_mapped"] = true
		other["upstream_model_name"] = info.UpstreamModelName
	}
	adminInfo := map[string]interface{}{}
	AppendSuccessfulRelayTrace(c, adminInfo, info.ChannelId)
	if len(adminInfo) > 0 {
		other["admin_info"] = adminInfo
	}
	model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
		ChannelId: info.ChannelId,
		ModelName: info.OriginModelName,
		TokenName: tokenName,
		Quota:     0,
		Content:   logContent,
		TokenId:   info.TokenId,
		Group:     info.UsingGroup,
		Other:     other,
	})
	model.IncrementUserRequestCount(info.UserId)
}

// RefundTaskQuota clears legacy task quota markers. New self-use tasks never
// reserve monetary quota, so no wallet or token balance is mutated.
func RefundTaskQuota(ctx context.Context, task *model.Task, reason string) bool {
	if task == nil || task.Quota == 0 {
		return true
	}
	// Historical tasks may still carry a quota value. Clear the compatibility
	// marker without touching user or token balances.
	task.Quota = 0
	if err := task.UpdateQuota(); err != nil {
		logger.LogError(ctx, fmt.Sprintf("clear legacy task quota failed task %s: %s", task.TaskID, err.Error()))
		return false
	}
	return true
}

// RecalculateTaskQuota is kept as a compatibility hook for existing adaptors.
// Monetary settlement is intentionally disabled in the self-use build.
func RecalculateTaskQuota(ctx context.Context, task *model.Task, actualQuota int, reason string, clamps ...*common.QuotaClamp) {
	_ = ctx
	_ = task
	_ = actualQuota
	_ = reason
	_ = clamps
}

// RecalculateTaskQuotaByTokens keeps the old adaptor contract without billing.
func RecalculateTaskQuotaByTokens(ctx context.Context, task *model.Task, totalTokens int) {
	_ = ctx
	_ = task
	_ = totalTokens
}
