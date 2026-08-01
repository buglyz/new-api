package controller

import (
	"context"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

type channelMonitorTarget struct {
	channel      *model.Channel
	model        string
	endpointType string
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
	targets, _ := collectChannelMonitorTargetsWithSkipped(channels, patterns)
	return targets
}

func collectChannelMonitorTargetsWithSkipped(channels []*model.Channel, patterns []string) ([]channelMonitorTarget, int) {
	targets := make([]channelMonitorTarget, 0)
	skipped := 0
	for _, channel := range channels {
		if channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		seen := make(map[string]struct{})
		for _, modelName := range channel.GetModels() {
			modelName = strings.TrimSpace(modelName)
			if modelName == "" {
				continue
			}
			if _, ok := seen[modelName]; ok {
				continue
			}
			seen[modelName] = struct{}{}
			if channelMonitorModelExcluded(modelName, patterns) {
				skipped++
				continue
			}
			endpointType, ok := channelMonitorEndpointType(channel, modelName)
			if !ok {
				skipped++
				continue
			}
			targets = append(targets, channelMonitorTarget{channel: channel, model: modelName, endpointType: endpointType})
		}
	}
	return targets, skipped
}

func channelMonitorEndpointType(channel *model.Channel, modelName string) (string, bool) {
	if channel == nil || !isChannelTestSupported(channel.Type) ||
		strings.HasSuffix(modelName, ratio_setting.CompactModelSuffix) || channel.Type == constant.ChannelTypeReplicate {
		return "", false
	}
	modelNames := channelMonitorMappedModelNames(channel, modelName)
	for _, candidate := range modelNames {
		if common.IsImageGenerationModel(candidate) {
			return "", false
		}
	}
	unsupported := []string{
		"audio", "realtime", "whisper", "tts", "moderation", "sora", "video",
		"image", "imagen", "seedream", "seedance", "cogview", "kolors", "flux",
		"suno", "chirp", "music", "kling", "vidu", "runway", "hailuo", "veo-", "wan2",
	}
	for _, candidate := range modelNames {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if channel.Type == constant.ChannelTypeVolcEngine && strings.Contains(candidate, "seedream") {
			return "", false
		}
		for _, marker := range unsupported {
			if strings.Contains(candidate, marker) {
				return "", false
			}
		}
	}
	preferredEndpoint := channelMonitorPreferredEndpoint(channel, modelName)
	for _, mappedModel := range modelNames[1:] {
		mappedEndpoint := channelMonitorPreferredEndpoint(channel, mappedModel)
		if mappedEndpoint != "" && mappedEndpoint != preferredEndpoint {
			return "", false
		}
	}

	var endpoints []constant.EndpointType
	if channel.Type == constant.ChannelTypeAdvancedCustom {
		var otherSettings dto.ChannelOtherSettings
		if common.UnmarshalJsonStr(channel.OtherSettings, &otherSettings) != nil || otherSettings.AdvancedCustom == nil {
			return "", false
		}
		endpoints = otherSettings.AdvancedCustom.SupportedEndpointTypesForModel(modelName)
	} else {
		switch preferredEndpoint {
		case constant.EndpointTypeEmbeddings:
			if channel.Type != constant.ChannelTypeMokaAI {
				return "", false
			}
			endpoints = []constant.EndpointType{preferredEndpoint}
		case constant.EndpointTypeJinaRerank:
			if channel.Type != constant.ChannelTypeJina {
				return "", false
			}
			endpoints = []constant.EndpointType{preferredEndpoint}
		default:
			endpoints = common.GetEndpointTypesByChannelType(channel.Type, modelName)
		}
	}
	if preferredEndpoint != "" {
		for _, endpoint := range endpoints {
			if endpoint == preferredEndpoint {
				return string(endpoint), true
			}
		}
		if channel.Type != constant.ChannelTypeAdvancedCustom {
			return "", false
		}
	}
	for _, endpoint := range endpoints {
		switch endpoint {
		case constant.EndpointTypeOpenAIResponse, constant.EndpointTypeAnthropic, constant.EndpointTypeGemini,
			constant.EndpointTypeJinaRerank, constant.EndpointTypeEmbeddings, constant.EndpointTypeOpenAI:
			return string(endpoint), true
		}
	}
	return "", false
}

func channelMonitorPreferredEndpoint(channel *model.Channel, modelName string) constant.EndpointType {
	lower := strings.ToLower(strings.TrimSpace(modelName))
	switch {
	case strings.Contains(lower, "rerank"):
		return constant.EndpointTypeJinaRerank
	case strings.Contains(lower, "embedding"), strings.Contains(lower, "embed"),
		strings.HasPrefix(lower, "m3e"), strings.Contains(lower, "bge-"), channel.Type == constant.ChannelTypeMokaAI:
		return constant.EndpointTypeEmbeddings
	case strings.Contains(lower, "codex"), common.IsOpenAIResponseOnlyModel(modelName):
		return constant.EndpointTypeOpenAIResponse
	default:
		return ""
	}
}

func channelMonitorMappedModelNames(channel *model.Channel, modelName string) []string {
	modelNames := []string{strings.TrimSpace(modelName)}
	if channel == nil || strings.TrimSpace(channel.GetModelMapping()) == "" {
		return modelNames
	}
	modelMapping := make(map[string]string)
	if common.UnmarshalJsonStr(channel.GetModelMapping(), &modelMapping) != nil {
		return modelNames
	}
	visited := map[string]struct{}{modelNames[0]: {}}
	current := modelNames[0]
	for {
		next := strings.TrimSpace(modelMapping[current])
		if next == "" {
			return modelNames
		}
		modelNames = append(modelNames, next)
		if _, ok := visited[next]; ok {
			return modelNames
		}
		visited[next] = struct{}{}
		current = next
	}
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
	result := testChannel(probeCtx, target.channel, userID, target.model, target.endpointType, shouldUseStreamForAutomaticChannelTest(target.channel), false)
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
		for _, key := range channelMonitorSecrets(channel) {
			value = redactChannelMonitorValue(value, key)
		}
		for _, rawURL := range channelMonitorSensitiveURLs(channel) {
			value = redactChannelMonitorValue(value, rawURL)
			parsed, err := url.Parse(rawURL)
			if err != nil {
				continue
			}
			if parsed.User != nil {
				value = redactChannelMonitorValue(value, parsed.User.String())
				value = redactChannelMonitorValue(value, parsed.User.Username())
				if password, ok := parsed.User.Password(); ok {
					value = redactChannelMonitorValue(value, password)
				}
			}
			if parsed.Host != "" {
				value = redactChannelMonitorValue(value, parsed.Host)
			}
			if parsed.Hostname() != "" {
				value = redactChannelMonitorValue(value, parsed.Hostname())
			}
			for _, values := range parsed.Query() {
				for _, queryValue := range values {
					value = redactChannelMonitorValue(value, queryValue)
				}
			}
		}
	}
	value = common.MaskSensitiveInfo(value)
	if len(value) > 500 {
		return value[:500]
	}
	return value
}

func channelMonitorSecrets(channel *model.Channel) []string {
	if channel == nil {
		return nil
	}
	secrets := append([]string(nil), channel.GetKeys()...)
	sources := []string{channel.Key, channel.Other, channel.OtherInfo, channel.OtherSettings}
	for _, value := range []*string{channel.OpenAIOrganization, channel.Setting, channel.ParamOverride, channel.HeaderOverride} {
		if value != nil {
			sources = append(sources, *value)
		}
	}
	for _, source := range sources {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}
		var structured any
		if common.Unmarshal([]byte(source), &structured) != nil {
			secrets = append(secrets, source)
			continue
		}
		var collect func(any)
		collect = func(value any) {
			switch typed := value.(type) {
			case map[string]any:
				for _, nested := range typed {
					collect(nested)
				}
			case []any:
				for _, nested := range typed {
					collect(nested)
				}
			case string:
				if strings.TrimSpace(typed) != "" {
					secrets = append(secrets, typed)
				}
			}
		}
		collect(structured)
	}
	return secrets
}

func channelMonitorSensitiveURLs(channel *model.Channel) []string {
	urls := []string{strings.TrimSpace(channel.GetBaseURL())}
	if channel.Setting != nil && strings.TrimSpace(*channel.Setting) != "" {
		var setting dto.ChannelSettings
		if common.Unmarshal([]byte(*channel.Setting), &setting) == nil {
			urls = append(urls, strings.TrimSpace(setting.Proxy))
		}
	}
	return urls
}

func redactChannelMonitorValue(value, sensitive string) string {
	sensitive = strings.TrimSpace(sensitive)
	if sensitive == "" {
		return value
	}
	value = strings.ReplaceAll(value, sensitive, "[redacted]")
	value = strings.ReplaceAll(value, url.QueryEscape(sensitive), "[redacted]")
	return strings.ReplaceAll(value, url.PathEscape(sensitive), "[redacted]")
}
