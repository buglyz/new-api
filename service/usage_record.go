package service

import (
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

func usageLogOther(ctx *gin.Context, info *relaycommon.RelayInfo) map[string]interface{} {
	other := make(map[string]interface{})
	other["frt"] = float64(info.FirstResponseTime.UnixMilli() - info.StartTime.UnixMilli())
	if info.ReasoningEffort != "" {
		other["reasoning_effort"] = info.ReasoningEffort
	}
	if info.IsModelMapped {
		other["is_model_mapped"] = true
		other["upstream_model_name"] = info.UpstreamModelName
	}
	if common.GetContextKeyBool(ctx, constant.ContextKeySystemPromptOverride) {
		other["is_system_prompt_overwritten"] = true
	}

	adminInfo := map[string]interface{}{
		"use_channel": ctx.GetStringSlice("use_channel"),
	}
	if common.GetContextKeyBool(ctx, constant.ContextKeyChannelIsMultiKey) {
		adminInfo["is_multi_key"] = true
		adminInfo["multi_key_index"] = common.GetContextKeyInt(ctx, constant.ContextKeyChannelMultiKeyIndex)
	}
	if common.GetContextKeyBool(ctx, constant.ContextKeyLocalCountTokens) {
		adminInfo["local_count_tokens"] = true
	}
	AppendChannelAffinityAdminInfo(ctx, adminInfo)
	AppendSuccessfulRelayTrace(ctx, adminInfo, info.ChannelId)
	other["admin_info"] = adminInfo

	appendRequestPath(ctx, info, other)
	appendRequestConversionChain(info, other)
	appendFinalRequestFormat(info, other)
	appendParamOverrideInfo(info, other)
	appendStreamStatus(info, other)
	return other
}

func normalizeUsage(info *relaycommon.RelayInfo, usage *dto.Usage) *dto.Usage {
	if usage != nil {
		return usage
	}
	estimated := info.GetEstimatePromptTokens()
	return &dto.Usage{
		PromptTokens: estimated,
		TotalTokens:  estimated,
	}
}

func normalizeLogModel(modelName string) (string, string) {
	if strings.HasPrefix(modelName, "gpt-4-gizmo") || strings.HasPrefix(modelName, "gpt-4o-gizmo") {
		prefix := "gpt-4-gizmo-*"
		if strings.HasPrefix(modelName, "gpt-4o-gizmo") {
			prefix = "gpt-4o-gizmo-*"
		}
		return prefix, "模型 " + modelName
	}
	return modelName, ""
}

func RecordTextUsage(ctx *gin.Context, info *relaycommon.RelayInfo, usage *dto.Usage, extraContent []string) {
	originalUsage := usage
	usage = normalizeUsage(info, usage)
	if originalUsage != nil {
		ObserveChannelAffinityUsageCacheByRelayFormat(ctx, usage, info.GetFinalRequestRelayFormat())
	} else {
		extraContent = append(extraContent, "上游未返回 usage，使用请求估算")
	}

	logModel, modelNote := normalizeLogModel(info.OriginModelName)
	if modelNote != "" {
		extraContent = append(extraContent, modelNote)
	}
	other := usageLogOther(ctx, info)
	other["cache_tokens"] = usage.PromptTokensDetails.CachedTokens
	if cacheWriteTokens := usage.PromptTokensDetails.CacheCreationTokensTotal(); cacheWriteTokens > 0 {
		other["cache_write_tokens"] = cacheWriteTokens
	}
	if usage.PromptTokensDetails.ImageTokens > 0 {
		other["image"] = true
		other["image_output"] = usage.PromptTokensDetails.ImageTokens
	}
	if usage.PromptTokensDetails.AudioTokens > 0 {
		other["audio_input_token_count"] = usage.PromptTokensDetails.AudioTokens
	}
	if usage.UsageSource != "" {
		other["usage_source"] = usage.UsageSource
	}
	if usage.UsageSemantic != "" {
		other["usage_semantic"] = usage.UsageSemantic
	} else if info.GetFinalRequestRelayFormat() == types.RelayFormatClaude {
		other["usage_semantic"] = "anthropic"
	}
	if rejectReason := common.GetContextKeyString(ctx, constant.ContextKeyAdminRejectReason); rejectReason != "" {
		other["reject_reason"] = rejectReason
	}

	recordUsageLog(ctx, info, usage.PromptTokens, usage.CompletionTokens, logModel, strings.Join(extraContent, ", "), other)
}

func RecordAudioUsage(ctx *gin.Context, info *relaycommon.RelayInfo, usage *dto.Usage, extraContent string) {
	usage = normalizeUsage(info, usage)
	other := usageLogOther(ctx, info)
	other["audio"] = true
	other["audio_input"] = usage.PromptTokensDetails.AudioTokens
	other["audio_output"] = usage.CompletionTokenDetails.AudioTokens
	other["text_input"] = usage.PromptTokensDetails.TextTokens
	other["text_output"] = usage.CompletionTokenDetails.TextTokens
	recordUsageLog(ctx, info, usage.PromptTokens, usage.CompletionTokens, info.OriginModelName, extraContent, other)
}

func RecordRealtimeUsage(ctx *gin.Context, info *relaycommon.RelayInfo, usage *dto.RealtimeUsage, extraContent string) {
	if usage == nil {
		usage = &dto.RealtimeUsage{}
	}
	other := usageLogOther(ctx, info)
	other["ws"] = true
	other["audio_input"] = usage.InputTokenDetails.AudioTokens
	other["audio_output"] = usage.OutputTokenDetails.AudioTokens
	other["text_input"] = usage.InputTokenDetails.TextTokens
	other["text_output"] = usage.OutputTokenDetails.TextTokens
	recordUsageLog(ctx, info, usage.InputTokens, usage.OutputTokens, info.OriginModelName, extraContent, other)
}

func recordUsageLog(ctx *gin.Context, info *relaycommon.RelayInfo, promptTokens, completionTokens int, modelName, content string, other map[string]interface{}) {
	model.RecordConsumeLog(ctx, info.UserId, model.RecordConsumeLogParams{
		ChannelId:        info.ChannelId,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		ModelName:        modelName,
		TokenName:        ctx.GetString("token_name"),
		Quota:            0,
		Content:          content,
		TokenId:          info.TokenId,
		UseTimeSeconds:   int(time.Since(info.StartTime).Seconds()),
		IsStream:         info.IsStream,
		Group:            info.UsingGroup,
		Other:            other,
	})
	model.IncrementUserRequestCount(info.UserId)
	gopool.Go(func() {
		perfmetrics.RecordRelaySample(info, true, int64(completionTokens))
	})
}
