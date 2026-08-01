package controller

import (
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

const (
	channelMonitorOverviewWindowSeconds     int64 = 24 * 60 * 60
	channelMonitorAvailabilityBucketSeconds int64 = 60 * 60
	channelMonitorAvailabilityBucketCount         = 24
)

type channelMonitorConfigRequest struct {
	Enabled                  bool     `json:"enabled"`
	IntervalMinutes          int      `json:"interval_minutes"`
	Concurrency              int      `json:"concurrency"`
	TimeoutSeconds           int      `json:"timeout_seconds"`
	ConfirmRetries           int      `json:"confirm_retries"`
	ConfirmRetryDelaySeconds int      `json:"confirm_retry_delay_seconds"`
	FailureThreshold         int      `json:"failure_threshold"`
	ExcludePatterns          []string `json:"exclude_patterns"`
	ExcludeChannelIDs        []int    `json:"exclude_channel_ids"`
}

type channelMonitorChannelOption struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type channelMonitorAvailabilityPoint struct {
	StartAt     int64    `json:"start_at"`
	EndAt       int64    `json:"end_at"`
	SuccessRate *float64 `json:"success_rate"`
	Succeeded   int64    `json:"succeeded"`
	Samples     int64    `json:"samples"`
}

type channelMonitorAvailabilityModel struct {
	Model  string                            `json:"model"`
	Points []channelMonitorAvailabilityPoint `json:"points"`
}

type channelMonitorAvailability struct {
	ChannelID int                               `json:"channel_id"`
	Points    []channelMonitorAvailabilityPoint `json:"points"`
	Models    []channelMonitorAvailabilityModel `json:"models"`
}

type channelMonitorAvailabilityBucketStat struct {
	Total     int64
	Succeeded int64
}

func GetChannelMonitorOverview(c *gin.Context) {
	setting := operation_setting.GetNativeMonitorSetting()
	now := common.GetTimestamp()
	targets, err := model.ListChannelMonitorTargets(now - channelMonitorOverviewWindowSeconds)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	channels, err := model.ListChannelMonitorChannels()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	targets = filterChannelMonitorOverviewTargets(targets, channels, setting.ExcludePatterns, setting.ExcludeChannelIDs)
	if c.Query("filter") == "unhealthy" {
		filtered := make([]model.ChannelMonitorTarget, 0, len(targets))
		for _, target := range targets {
			if target.Health != model.ChannelMonitorHealthHealthy {
				filtered = append(filtered, target)
			}
		}
		targets = filtered
	}
	availabilityStats, err := model.ListChannelMonitorAvailability(
		now-channelMonitorOverviewWindowSeconds,
		channelMonitorAvailabilityBucketSeconds,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	task, err := model.GetActiveSystemTask(model.SystemTaskTypeChannelMonitor)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data := gin.H{
		"settings": setting,
		"targets":  targets,
		"channels": channelMonitorChannelOptions(channels),
		"availability": channelMonitorAvailabilityForTargets(
			targets,
			availabilityStats,
			now,
		),
		"task": nil,
	}
	if task != nil {
		data["task"] = task.ToResponse()
	}
	common.ApiSuccess(c, data)
}

func channelMonitorAvailabilityForTargets(targets []model.ChannelMonitorTarget, stats []model.ChannelMonitorAvailabilityStat, now int64) []channelMonitorAvailability {
	statsByTarget := make(map[string]map[int64]channelMonitorAvailabilityBucketStat)
	for _, stat := range stats {
		targetKey := channelMonitorIdentity(stat.ChannelID, stat.Model)
		byBucket := statsByTarget[targetKey]
		if byBucket == nil {
			byBucket = make(map[int64]channelMonitorAvailabilityBucketStat)
			statsByTarget[targetKey] = byBucket
		}
		bucket := byBucket[stat.BucketStart]
		bucket.Total += stat.Total
		bucket.Succeeded += stat.Succeeded
		byBucket[stat.BucketStart] = bucket
	}

	channelIDs := make([]int, 0, len(targets))
	seenChannels := make(map[int]struct{}, len(targets))
	targetsByChannel := make(map[int][]model.ChannelMonitorTarget)
	for _, target := range targets {
		targetsByChannel[target.ChannelID] = append(targetsByChannel[target.ChannelID], target)
		if _, ok := seenChannels[target.ChannelID]; ok {
			continue
		}
		seenChannels[target.ChannelID] = struct{}{}
		channelIDs = append(channelIDs, target.ChannelID)
	}
	sort.Ints(channelIDs)

	currentBucket := now - now%channelMonitorAvailabilityBucketSeconds
	firstBucket := currentBucket - int64(channelMonitorAvailabilityBucketCount-1)*channelMonitorAvailabilityBucketSeconds
	availability := make([]channelMonitorAvailability, 0, len(channelIDs))
	for _, channelID := range channelIDs {
		channelTargets := targetsByChannel[channelID]
		sort.Slice(channelTargets, func(left, right int) bool {
			return channelTargets[left].Model < channelTargets[right].Model
		})
		channelStats := make(map[int64]channelMonitorAvailabilityBucketStat)
		models := make([]channelMonitorAvailabilityModel, 0, len(channelTargets))
		for _, target := range channelTargets {
			byBucket := statsByTarget[channelMonitorIdentity(target.ChannelID, target.Model)]
			for bucketStart, stat := range byBucket {
				bucket := channelStats[bucketStart]
				bucket.Total += stat.Total
				bucket.Succeeded += stat.Succeeded
				channelStats[bucketStart] = bucket
			}
			models = append(models, channelMonitorAvailabilityModel{
				Model:  target.Model,
				Points: channelMonitorAvailabilityPoints(firstBucket, byBucket),
			})
		}
		availability = append(availability, channelMonitorAvailability{
			ChannelID: channelID,
			Points:    channelMonitorAvailabilityPoints(firstBucket, channelStats),
			Models:    models,
		})
	}
	return availability
}

func channelMonitorAvailabilityPoints(firstBucket int64, stats map[int64]channelMonitorAvailabilityBucketStat) []channelMonitorAvailabilityPoint {
	points := make([]channelMonitorAvailabilityPoint, 0, channelMonitorAvailabilityBucketCount)
	for index := 0; index < channelMonitorAvailabilityBucketCount; index++ {
		startAt := firstBucket + int64(index)*channelMonitorAvailabilityBucketSeconds
		point := channelMonitorAvailabilityPoint{
			StartAt: startAt,
			EndAt:   startAt + channelMonitorAvailabilityBucketSeconds,
		}
		if stat, ok := stats[startAt]; ok && stat.Total > 0 {
			rate := float64(stat.Succeeded) / float64(stat.Total)
			point.SuccessRate = &rate
			point.Succeeded = stat.Succeeded
			point.Samples = stat.Total
		}
		points = append(points, point)
	}
	return points
}

func channelMonitorChannelOptions(channels []*model.Channel) []channelMonitorChannelOption {
	options := make([]channelMonitorChannelOption, 0, len(channels))
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		options = append(options, channelMonitorChannelOption{
			ID: channel.Id, Name: channel.Name, Enabled: channel.Status == common.ChannelStatusEnabled,
		})
	}
	return options
}

func filterChannelMonitorOverviewTargets(targets []model.ChannelMonitorTarget, channels []*model.Channel, patterns []string, excludedChannelIDs []int) []model.ChannelMonitorTarget {
	monitorable := collectChannelMonitorTargets(channels, patterns, excludedChannelIDs)
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
		if err := service.CancelSystemTaskType(model.SystemTaskTypeChannelMonitor); err != nil {
			common.ApiError(c, err)
			return
		}
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
		ExcludePatterns:          request.ExcludePatterns,
		ExcludeChannelIDs:        request.ExcludeChannelIDs,
	}
	return operation_setting.NormalizeNativeMonitorSetting(setting)
}
