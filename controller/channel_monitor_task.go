package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

type channelMonitorHandler struct{}

func (channelMonitorHandler) Type() string { return model.SystemTaskTypeChannelMonitor }

func (channelMonitorHandler) Enabled() bool {
	return operation_setting.GetNativeMonitorSetting().Enabled
}

func (channelMonitorHandler) Interval() time.Duration {
	minutes := operation_setting.GetNativeMonitorSetting().IntervalMinutes
	if minutes < 1 {
		minutes = 1
	}
	return time.Duration(minutes) * time.Minute
}

func (channelMonitorHandler) NewPayload() any { return channelMonitorTaskPayload{} }

type channelMonitorTaskPayload struct {
	Manual bool `json:"manual,omitempty"`
}

type channelMonitorTaskSummary struct {
	Targets   int `json:"targets"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Skipped   int `json:"skipped"`
}

func (channelMonitorHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	var payload channelMonitorTaskPayload
	if err := task.DecodePayload(&payload); err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	summary, err := runChannelMonitorTask(ctx, service.NewSystemTaskProgressReporter(task, runnerID))
	finishSystemTaskHandlerWithContext(ctx, task, runnerID, summary, err)
}

func runChannelMonitorTask(ctx context.Context, report func(processed, total int)) (channelMonitorTaskSummary, error) {
	setting := operation_setting.GetNativeMonitorSetting()
	if !setting.Enabled {
		return channelMonitorTaskSummary{}, nil
	}
	userID, err := resolveChannelTestUserID(nil)
	if err != nil {
		return channelMonitorTaskSummary{}, err
	}
	channels, err := model.GetAllChannels(0, 0, true, true)
	if err != nil {
		return channelMonitorTaskSummary{}, fmt.Errorf("list monitorable channels: %w", err)
	}
	targets, skipped := collectChannelMonitorTargetsWithSkipped(channels, setting.ExcludePatterns, setting.ExcludeChannelIDs)
	summary := channelMonitorTaskSummary{Targets: len(targets) + skipped, Skipped: skipped}
	knownTargets := make([]model.ChannelMonitorTargetRef, 0)
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		for _, modelName := range channel.GetModels() {
			if modelName = strings.TrimSpace(modelName); modelName != "" {
				knownTargets = append(knownTargets, model.ChannelMonitorTargetRef{ChannelID: channel.Id, Model: modelName})
			}
		}
	}
	if err := model.DeleteStaleChannelMonitorTargets(knownTargets); err != nil {
		return summary, fmt.Errorf("clean stale monitor targets: %w", err)
	}
	if len(targets) == 0 {
		if report != nil {
			report(summary.Skipped, summary.Targets)
		}
		return summary, nil
	}

	final := make(map[string]channelMonitorProbe, len(targets))
	pending := append([]channelMonitorTarget(nil), targets...)
	for attempt := 0; attempt <= setting.ConfirmRetries && len(pending) > 0; attempt++ {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		if attempt > 0 && !waitForChannelMonitorRetry(ctx, setting.ConfirmRetryDelaySeconds) {
			return summary, ctx.Err()
		}
		probes := probeChannelMonitorTargets(ctx, pending, userID, setting)
		nextPending := make([]channelMonitorTarget, 0, len(probes))
		for _, probe := range probes {
			probe.attempts = attempt + 1
			final[channelMonitorTargetKey(probe.target)] = probe
			if !probe.succeeded {
				nextPending = append(nextPending, probe.target)
			}
		}
		pending = nextPending
	}

	processed := summary.Skipped
	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		probe, ok := final[channelMonitorTargetKey(target)]
		if !ok {
			summary.Skipped++
			processed++
			if report != nil {
				report(processed, summary.Targets)
			}
			continue
		}
		status := model.ChannelMonitorStatusFailure
		if probe.succeeded {
			status = model.ChannelMonitorStatusSuccess
			summary.Succeeded++
		} else {
			summary.Failed++
		}
		_, err := model.CreateChannelMonitorResult(model.ChannelMonitorResult{
			ChannelID:   target.channel.Id,
			ChannelName: target.channel.Name,
			Groups:      strings.Join(target.channel.GetGroups(), ","),
			Model:       target.model,
			Status:      status,
			Attempts:    probe.attempts,
			LatencyMS:   probe.latencyMS,
			HTTPStatus:  probe.httpStatus,
			Error:       probe.errorText,
		}, setting.FailureThreshold, operation_setting.NativeMonitorRetentionLimit(setting.IntervalMinutes))
		if err != nil {
			return summary, fmt.Errorf("persist monitor result: %w", err)
		}
		processed++
		if report != nil {
			report(processed, summary.Targets)
		}
	}
	return summary, nil
}

func waitForChannelMonitorRetry(ctx context.Context, delaySeconds int) bool {
	if delaySeconds < 1 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(time.Duration(delaySeconds) * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func channelMonitorTargetKey(target channelMonitorTarget) string {
	return channelMonitorIdentity(target.channel.Id, target.model)
}

func channelMonitorIdentity(channelID int, modelName string) string {
	return fmt.Sprintf("%d:%s", channelID, strings.TrimSpace(modelName))
}
