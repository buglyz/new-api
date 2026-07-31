package controller

import (
	"context"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

type channelMonitorTarget struct {
	channel *model.Channel
	model   string
}

type channelMonitorProbe struct {
	target     channelMonitorTarget
	succeeded  bool
	attempts   int
	latencyMS  int64
	httpStatus int
	errorText  string
}

func collectChannelMonitorTargets(channels []*model.Channel, patterns []string) []channelMonitorTarget {
	targets := make([]channelMonitorTarget, 0)
	for _, channel := range channels {
		if channel.Status != common.ChannelStatusEnabled {
			continue
		}
		seen := make(map[string]struct{})
		for _, modelName := range channel.GetModels() {
			modelName = strings.TrimSpace(modelName)
			if modelName == "" || channelMonitorModelExcluded(modelName, patterns) {
				continue
			}
			if _, ok := seen[modelName]; ok {
				continue
			}
			seen[modelName] = struct{}{}
			targets = append(targets, channelMonitorTarget{channel: channel, model: modelName})
		}
	}
	return targets
}

func channelMonitorModelExcluded(modelName string, patterns []string) bool {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	for _, pattern := range patterns {
		matched, err := path.Match(strings.ToLower(strings.TrimSpace(pattern)), modelName)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func probeChannelMonitorTargets(ctx context.Context, targets []channelMonitorTarget, userID int, setting operation_setting.NativeMonitorSetting) []channelMonitorProbe {
	if len(targets) == 0 {
		return nil
	}
	workers := min(setting.Concurrency, len(targets))
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan channelMonitorTarget)
	results := make(chan channelMonitorProbe, workers)
	var workersWG sync.WaitGroup
	workersWG.Add(workers)
	for range workers {
		go func() {
			defer workersWG.Done()
			for target := range jobs {
				probe := probeChannelMonitorTarget(ctx, target, userID, setting.TimeoutSeconds)
				select {
				case results <- probe:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, target := range targets {
			select {
			case jobs <- target:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		workersWG.Wait()
		close(results)
	}()
	probes := make([]channelMonitorProbe, 0, len(targets))
	for probe := range results {
		probes = append(probes, probe)
	}
	return probes
}

func probeChannelMonitorTarget(ctx context.Context, target channelMonitorTarget, userID, timeoutSeconds int) channelMonitorProbe {
	if timeoutSeconds < 1 {
		timeoutSeconds = 1
	}
	probeCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	startedAt := time.Now()
	result := testChannel(probeCtx, target.channel, userID, target.model, "", shouldUseStreamForAutomaticChannelTest(target.channel))
	probe := channelMonitorProbe{target: target, latencyMS: time.Since(startedAt).Milliseconds(), httpStatus: result.httpStatus}
	if result.localErr == nil && result.newAPIError == nil && probeCtx.Err() == nil {
		probe.succeeded = true
		return probe
	}
	if result.localErr != nil {
		probe.errorText = sanitizeChannelMonitorError(result.localErr.Error(), target.channel)
	} else if result.newAPIError != nil {
		probe.errorText = sanitizeChannelMonitorError(result.newAPIError.Error(), target.channel)
	} else {
		probe.errorText = sanitizeChannelMonitorError(probeCtx.Err().Error(), target.channel)
	}
	return probe
}

func sanitizeChannelMonitorError(value string, channel *model.Channel) string {
	value = strings.TrimSpace(value)
	if channel != nil {
		for _, key := range channel.GetKeys() {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			value = strings.ReplaceAll(value, key, "[redacted]")
			value = strings.ReplaceAll(value, url.QueryEscape(key), "[redacted]")
			value = strings.ReplaceAll(value, url.PathEscape(key), "[redacted]")
		}
		baseURL := strings.TrimSpace(channel.GetBaseURL())
		if baseURL != "" {
			value = strings.ReplaceAll(value, baseURL, "[redacted]")
		}
		if parsed, err := url.Parse(baseURL); err == nil && parsed.User != nil {
			value = strings.ReplaceAll(value, parsed.User.String(), "[redacted]")
		}
	}
	if len(value) > 500 {
		return value[:500]
	}
	return value
}
