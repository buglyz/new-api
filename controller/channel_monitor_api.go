package controller

import (
	"path"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

const channelMonitorOverviewWindowSeconds = 24 * 60 * 60

type channelMonitorConfigRequest struct {
	Enabled                  bool     `json:"enabled"`
	IntervalMinutes          int      `json:"interval_minutes"`
	Concurrency              int      `json:"concurrency"`
	TimeoutSeconds           int      `json:"timeout_seconds"`
	ConfirmRetries           int      `json:"confirm_retries"`
	ConfirmRetryDelaySeconds int      `json:"confirm_retry_delay_seconds"`
	FailureThreshold         int      `json:"failure_threshold"`
	ExcludePatterns          []string `json:"exclude_patterns"`
}

func GetChannelMonitorOverview(c *gin.Context) {
	setting := operation_setting.GetNativeMonitorSetting()
	targets, err := model.ListChannelMonitorTargets(common.GetTimestamp() - channelMonitorOverviewWindowSeconds)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	channels, err := model.ListChannelMonitorChannels()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	targets = filterChannelMonitorOverviewTargets(targets, channels, setting.ExcludePatterns)
	if c.Query("filter") == "unhealthy" {
		filtered := make([]model.ChannelMonitorTarget, 0, len(targets))
		for _, target := range targets {
			if target.Health != model.ChannelMonitorHealthHealthy {
				filtered = append(filtered, target)
			}
		}
		targets = filtered
	}
	task, err := model.GetActiveSystemTask(model.SystemTaskTypeChannelMonitor)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data := gin.H{
		"settings": setting,
		"targets":  targets,
		"task":     nil,
	}
	if task != nil {
		data["task"] = task.ToResponse()
	}
	common.ApiSuccess(c, data)
}

func filterChannelMonitorOverviewTargets(targets []model.ChannelMonitorTarget, channels []*model.Channel, patterns []string) []model.ChannelMonitorTarget {
	monitorable := collectChannelMonitorTargets(channels, patterns)
	monitorableKeys := make(map[string]struct{}, len(monitorable))
	for _, target := range monitorable {
		monitorableKeys[channelMonitorTargetKey(target)] = struct{}{}
	}
	filtered := make([]model.ChannelMonitorTarget, 0, len(targets))
	for _, target := range targets {
		if _, ok := monitorableKeys[channelMonitorIdentity(target.ChannelID, target.Model)]; ok {
			filtered = append(filtered, target)
		}
	}
	return filtered
}

func UpdateChannelMonitorConfig(c *gin.Context) {
	var request channelMonitorConfigRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "invalid monitor configuration")
		return
	}
	setting, err := normalizeChannelMonitorConfig(request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	values, err := config.ConfigToMap(&setting)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	options := make(map[string]string, len(values))
	for key, value := range values {
		options["native_monitor_setting."+key] = value
	}
	if err := model.UpdateOptionsBulk(options); err != nil {
		common.ApiError(c, err)
		return
	}
	if !setting.Enabled {
		service.CancelSystemTaskType(model.SystemTaskTypeChannelMonitor)
	}
	service.WakeSystemTaskRunner()
	recordManageAudit(c, "channel_monitor.update", map[string]interface{}{"enabled": setting.Enabled})
	common.ApiSuccess(c, setting)
}

func TriggerChannelMonitor(c *gin.Context) {
	if !operation_setting.GetNativeMonitorSetting().Enabled {
		common.ApiErrorMsg(c, "channel monitoring is disabled")
		return
	}
	task, created, err := service.EnqueueSystemTask(
		model.SystemTaskTypeChannelMonitor,
		channelMonitorTaskPayload{Manual: true},
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "channel_monitor.trigger", map[string]interface{}{"created": created})
	common.ApiSuccess(c, gin.H{"created": created, "task": task.ToResponse()})
}

func GetChannelMonitorHistory(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Query("channel_id"))
	modelName := strings.TrimSpace(c.Query("model"))
	if err != nil || channelID < 1 || modelName == "" {
		common.ApiErrorMsg(c, "channel_id and model are required")
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit < 1 {
		limit = 60
	}
	if limit > operation_setting.NativeMonitorHistoryLimit {
		limit = operation_setting.NativeMonitorHistoryLimit
	}
	history, err := model.ListChannelMonitorHistory(channelID, modelName, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, history)
}

func normalizeChannelMonitorConfig(request channelMonitorConfigRequest) (operation_setting.NativeMonitorSetting, error) {
	setting := operation_setting.NativeMonitorSetting{
		Enabled:                  request.Enabled,
		IntervalMinutes:          request.IntervalMinutes,
		Concurrency:              request.Concurrency,
		TimeoutSeconds:           request.TimeoutSeconds,
		ConfirmRetries:           request.ConfirmRetries,
		ConfirmRetryDelaySeconds: request.ConfirmRetryDelaySeconds,
		FailureThreshold:         request.FailureThreshold,
	}
	if setting.IntervalMinutes < 1 || setting.IntervalMinutes > 1440 {
		return setting, channelMonitorConfigError("interval_minutes must be between 1 and 1440")
	}
	if setting.Concurrency < 1 || setting.Concurrency > 32 {
		return setting, channelMonitorConfigError("concurrency must be between 1 and 32")
	}
	if setting.TimeoutSeconds < 1 || setting.TimeoutSeconds > 120 {
		return setting, channelMonitorConfigError("timeout_seconds must be between 1 and 120")
	}
	if setting.ConfirmRetries < 0 || setting.ConfirmRetries > 3 {
		return setting, channelMonitorConfigError("confirm_retries must be between 0 and 3")
	}
	if setting.ConfirmRetryDelaySeconds < 0 || setting.ConfirmRetryDelaySeconds > 60 {
		return setting, channelMonitorConfigError("confirm_retry_delay_seconds must be between 0 and 60")
	}
	if setting.FailureThreshold < 1 || setting.FailureThreshold > 10 {
		return setting, channelMonitorConfigError("failure_threshold must be between 1 and 10")
	}
	if len(request.ExcludePatterns) > 100 {
		return setting, channelMonitorConfigError("exclude_patterns cannot contain more than 100 entries")
	}
	for _, pattern := range request.ExcludePatterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || len(pattern) > 255 {
			return setting, channelMonitorConfigError("each exclude pattern must be between 1 and 255 characters")
		}
		if _, err := path.Match(strings.ToLower(pattern), "probe-model"); err != nil {
			return setting, channelMonitorConfigError("exclude_patterns contains an invalid wildcard")
		}
		setting.ExcludePatterns = append(setting.ExcludePatterns, pattern)
	}
	return setting, nil
}

type channelMonitorConfigError string

func (err channelMonitorConfigError) Error() string { return string(err) }
