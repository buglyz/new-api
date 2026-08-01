package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelMonitorUsesRelayMappingAndRetriesAgainstFakeUpstream(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}, &model.ChannelMonitorResult{}))
	configureNativeMonitorForTest(t)

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer monitor-secret", r.Header.Get("Authorization"))
		body := make(map[string]any)
		require.NoError(t, common.DecodeJson(r.Body, &body))
		assert.Equal(t, "upstream-model", body["model"])
		assert.NotNil(t, body["max_tokens"])
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":{"message":"temporary failure"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"fake","choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	root := &model.User{Username: "root", Role: common.RoleRootUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, db.Create(root).Error)
	mapping := `{"monitor-model":"upstream-model"}`
	baseURL := server.URL
	channel := &model.Channel{
		Type: constant.ChannelTypeOpenAI, Key: "monitor-secret", Name: "fake upstream",
		Status: common.ChannelStatusEnabled, Models: "monitor-model", Group: "default",
		BaseURL: &baseURL, ModelMapping: &mapping,
	}
	require.NoError(t, db.Create(channel).Error)

	summary, err := runChannelMonitorTask(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Targets)
	assert.Equal(t, 1, summary.Succeeded)
	assert.Equal(t, int32(2), requests.Load())

	history, err := model.ListChannelMonitorHistory(channel.Id, "monitor-model", 10)
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, model.ChannelMonitorStatusSuccess, history[0].Status)
	assert.Equal(t, 2, history[0].Attempts)
}

func TestChannelMonitorStaysIdleWhenDisabled(t *testing.T) {
	configureNativeMonitorForTest(t)
	require.NoError(t, operation_setting.UpdateNativeMonitorSettingFromMap(map[string]string{"enabled": "false"}))

	summary, err := runChannelMonitorTask(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, channelMonitorTaskSummary{}, summary)
}

func TestChannelMonitorSanitizesChannelCredentials(t *testing.T) {
	baseURL := "https://user:password@example.test"
	channel := &model.Channel{Key: "super-secret", BaseURL: &baseURL}
	message := sanitizeChannelMonitorError("super-secret failed at https://user:password@example.test", channel)

	assert.NotContains(t, message, "super-secret")
	assert.NotContains(t, message, "password")
	assert.NotContains(t, message, "example.test")
	assert.Contains(t, message, "[redacted]")
}

func TestChannelMonitorSanitizesStructuredCredentials(t *testing.T) {
	channel := &model.Channel{Key: "{\"access_token\":\"access-secret\",\"refresh_token\":\"refresh-secret\"}"}
	message := sanitizeChannelMonitorError("Bearer access-secret refresh-secret", channel)

	assert.NotContains(t, message, "access-secret")
	assert.NotContains(t, message, "refresh-secret")
}

func TestChannelMonitorSanitizesRouteAuthAndProxyCredentials(t *testing.T) {
	setting := `{"proxy":"http://proxy-user:proxy-pass@proxy.example:8080?token=proxy-token"}`
	channel := &model.Channel{
		Setting:       &setting,
		OtherSettings: `{"advanced_custom":{"routes":[{"incoming_path":"/v1/chat/completions","upstream_path":"/chat","auth":{"type":"header","name":"X-Key","value":"route-secret"}}]}}`,
	}
	message := sanitizeChannelMonitorError(
		"route-secret proxy-user proxy-pass proxy.example proxy-token",
		channel,
	)

	for _, sensitive := range []string{"route-secret", "proxy-user", "proxy-pass", "proxy.example", "proxy-token"} {
		assert.NotContains(t, message, sensitive)
	}
}

func TestChannelMonitorSkipsExpensiveOrUnsupportedModels(t *testing.T) {
	channel := &model.Channel{
		Type:   constant.ChannelTypeOpenAI,
		Status: common.ChannelStatusEnabled,
		Models: "gpt-4o,dall-e-3,gpt-image-1,whisper-1,tts-1,omni-moderation-latest,seedance-1.0-pro",
	}
	targets, skipped := collectChannelMonitorTargetsWithSkipped([]*model.Channel{channel}, nil, nil)

	require.Len(t, targets, 1)
	assert.Equal(t, 6, skipped)
	assert.Equal(t, "gpt-4o", targets[0].model)
	assert.Equal(t, string(constant.EndpointTypeOpenAI), targets[0].endpointType)
}

func TestChannelMonitorSkipsUnsupportedChannelTypes(t *testing.T) {
	channel := &model.Channel{
		Type:   constant.ChannelTypeSunoAPI,
		Status: common.ChannelStatusEnabled,
		Models: "chirp-v3-5",
	}

	assert.Empty(t, collectChannelMonitorTargets([]*model.Channel{channel}, nil, nil))

	replicate := &model.Channel{
		Type:   constant.ChannelTypeReplicate,
		Status: common.ChannelStatusEnabled,
		Models: "vendor/custom-generation-model",
	}
	assert.Empty(t, collectChannelMonitorTargets([]*model.Channel{replicate}, nil, nil))
}

func TestChannelMonitorSkipsConfiguredChannels(t *testing.T) {
	channels := []*model.Channel{
		{Id: 1, Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Models: "model-a"},
		{Id: 2, Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Models: "model-b"},
	}

	targets, skipped := collectChannelMonitorTargetsWithSkipped(channels, nil, []int{1})

	require.Len(t, targets, 1)
	assert.Equal(t, 0, skipped)
	assert.Equal(t, 2, targets[0].channel.Id)
}

func TestChannelMonitorUsesConfiguredAdvancedCustomEndpoint(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeAdvancedCustom}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{Routes: []dto.AdvancedCustomRoute{{
			IncomingPath: "/v1/chat/completions",
			Models:       []string{"rerank-chat-model"},
		}}},
	})

	endpoint, ok := channelMonitorEndpointType(channel, "rerank-chat-model")
	require.True(t, ok)
	assert.Equal(t, string(constant.EndpointTypeOpenAI), endpoint)
}

func TestChannelMonitorUsesResponsesForCodexModels(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeNewAPI}

	endpoint, ok := channelMonitorEndpointType(channel, "codex-mini")
	require.True(t, ok)
	assert.Equal(t, string(constant.EndpointTypeOpenAIResponse), endpoint)

	_, ok = channelMonitorEndpointType(&model.Channel{Type: constant.ChannelTypeOpenRouter}, "codex-mini")
	assert.False(t, ok)
}

func TestChannelMonitorUsesOnlyDedicatedEmbeddingAndRerankChannels(t *testing.T) {
	embeddingEndpoint, ok := channelMonitorEndpointType(&model.Channel{Type: constant.ChannelTypeMokaAI}, "moka-model")
	require.True(t, ok)
	assert.Equal(t, string(constant.EndpointTypeEmbeddings), embeddingEndpoint)

	rerankEndpoint, ok := channelMonitorEndpointType(&model.Channel{Type: constant.ChannelTypeJina}, "jina-rerank-v2")
	require.True(t, ok)
	assert.Equal(t, string(constant.EndpointTypeJinaRerank), rerankEndpoint)

	_, ok = channelMonitorEndpointType(&model.Channel{Type: constant.ChannelTypeAli}, "text-embedding-v1")
	assert.False(t, ok)
}

func TestChannelMonitorSkipsAliasesMappedToExpensiveModels(t *testing.T) {
	mapping := `{"safe-alias":"image-alias","image-alias":"gpt-image-1"}`
	channel := &model.Channel{Type: constant.ChannelTypeOpenAI, ModelMapping: &mapping}

	_, ok := channelMonitorEndpointType(channel, "safe-alias")
	assert.False(t, ok)

	embeddingMapping := `{"safe-alias":"text-embedding-v1"}`
	channel.ModelMapping = &embeddingMapping
	_, ok = channelMonitorEndpointType(channel, "safe-alias")
	assert.False(t, ok)
}

func TestChannelMonitorOverviewHidesTargetsThatAreNoLongerMonitorable(t *testing.T) {
	channels := []*model.Channel{
		{Id: 1, Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Models: "gpt-4o,excluded-model,gpt-image-1"},
		{Id: 2, Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusManuallyDisabled, Models: "disabled-model"},
	}
	targets := []model.ChannelMonitorTarget{
		{ChannelID: 1, Model: "gpt-4o"},
		{ChannelID: 1, Model: "excluded-model"},
		{ChannelID: 1, Model: "gpt-image-1"},
		{ChannelID: 1, Model: "removed-model"},
		{ChannelID: 2, Model: "disabled-model"},
	}

	filtered := filterChannelMonitorOverviewTargets(targets, channels, []string{"excluded-*"}, nil)
	require.Len(t, filtered, 1)
	assert.Equal(t, "gpt-4o", filtered[0].Model)
}

func TestChannelMonitorAvailabilityFillsEmptyHourlyBuckets(t *testing.T) {
	now := common.GetTimestamp()
	currentBucket := now - now%channelMonitorAvailabilityBucketSeconds
	availability := channelMonitorAvailabilityForTargets(
		[]model.ChannelMonitorTarget{{ChannelID: 4}},
		[]model.ChannelMonitorAvailabilityStat{{
			ChannelID:   4,
			BucketStart: currentBucket - channelMonitorAvailabilityBucketSeconds,
			Total:       4,
			Succeeded:   3,
		}},
		now,
	)

	require.Len(t, availability, 1)
	require.Len(t, availability[0].Points, channelMonitorAvailabilityBucketCount)
	assert.Nil(t, availability[0].Points[0].SuccessRate)
	point := availability[0].Points[channelMonitorAvailabilityBucketCount-2]
	require.NotNil(t, point.SuccessRate)
	assert.InDelta(t, 0.75, *point.SuccessRate, 0.001)
	assert.Equal(t, int64(3), point.Succeeded)
	assert.Equal(t, int64(4), point.Samples)
}

func TestChannelMonitorQuietRequestLimitsOutputTokens(t *testing.T) {
	geminiReq, ok := buildTestRequest("gemini-2.5-pro", string(constant.EndpointTypeGemini), nil, false, true).(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotNil(t, geminiReq.MaxTokens)
	assert.EqualValues(t, 16, *geminiReq.MaxTokens)

	normalGeminiReq, ok := buildTestRequest("gemini-2.5-pro", string(constant.EndpointTypeGemini), nil, false, false).(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotNil(t, normalGeminiReq.MaxTokens)
	assert.EqualValues(t, 3000, *normalGeminiReq.MaxTokens)

	responsesReq, ok := buildTestRequest("codex-mini", string(constant.EndpointTypeOpenAIResponse), nil, false, true).(*dto.OpenAIResponsesRequest)
	require.True(t, ok)
	require.NotNil(t, responsesReq.MaxOutputTokens)
	assert.EqualValues(t, 16, *responsesReq.MaxOutputTokens)

	reasoningReq, ok := buildTestRequest("o1-mini", string(constant.EndpointTypeOpenAI), nil, false, true).(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	assert.Nil(t, reasoningReq.MaxTokens)
	require.NotNil(t, reasoningReq.MaxCompletionTokens)
	assert.EqualValues(t, 16, *reasoningReq.MaxCompletionTokens)
}

func TestFinishSystemTaskHandlerWithContextMarksCanceledRunFailed(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SystemTask{}, &model.SystemTaskLock{}))

	task, err := model.CreateSystemTask(model.SystemTaskTypeChannelMonitor, nil, nil)
	require.NoError(t, err)
	claimedTask, claimed, err := model.ClaimSystemTask(task.ID, task.Type, "runner-cancel", common.GetTimestamp()+60)
	require.NoError(t, err)
	require.True(t, claimed)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	finishSystemTaskHandlerWithContext(ctx, claimedTask, "runner-cancel", channelMonitorTaskSummary{}, nil)

	reloaded, err := model.GetSystemTaskByTaskID(task.TaskID)
	require.NoError(t, err)
	require.NotNil(t, reloaded)
	assert.Equal(t, model.SystemTaskStatusFailed, reloaded.Status)
	assert.Contains(t, reloaded.Error, context.Canceled.Error())
}

func TestNativeMonitorSettingPublishesConsistentSnapshots(t *testing.T) {
	configureNativeMonitorForTest(t)
	for i := 1; i <= 20; i++ {
		require.NoError(t, operation_setting.UpdateNativeMonitorSettingFromMap(map[string]string{
			"concurrency":         fmt.Sprintf("%d", i),
			"exclude_patterns":    fmt.Sprintf("[\"model-%d\"]", i),
			"exclude_channel_ids": fmt.Sprintf("[%d, %d]", i, i),
		}))
		snapshot := operation_setting.GetNativeMonitorSetting()
		assert.Equal(t, i, snapshot.Concurrency)
		assert.Equal(t, []string{fmt.Sprintf("model-%d", i)}, snapshot.ExcludePatterns)
		assert.Equal(t, []int{i}, snapshot.ExcludeChannelIDs)
	}
}

func TestChannelMonitorConfigRejectsUnsafeValues(t *testing.T) {
	_, err := normalizeChannelMonitorConfig(channelMonitorConfigRequest{
		IntervalMinutes: 10, Concurrency: 1, TimeoutSeconds: 5,
		ConfirmRetries: 0, ConfirmRetryDelaySeconds: 0, FailureThreshold: 1,
		ExcludePatterns:   []string{"["},
		ExcludeChannelIDs: []int{1},
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid wildcard"))
}

func configureNativeMonitorForTest(t *testing.T) {
	t.Helper()
	monitorConfig := config.GlobalConfig.Get("native_monitor_setting")
	current := operation_setting.GetNativeMonitorSetting()
	original, err := config.ConfigToMap(&current)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, operation_setting.UpdateNativeMonitorSettingFromMap(original))
	})
	require.NoError(t, operation_setting.UpdateNativeMonitorSettingFromMap(map[string]string{
		"enabled": "true", "interval_minutes": "10", "concurrency": "1",
		"timeout_seconds": "5", "confirm_retries": "1", "confirm_retry_delay_seconds": "0",
		"failure_threshold": "2", "exclude_patterns": "[]", "exclude_channel_ids": "[]",
	}))
	require.NotNil(t, monitorConfig)
}
