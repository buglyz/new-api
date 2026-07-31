package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/config"
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
	monitorConfig := config.GlobalConfig.Get("native_monitor_setting")
	require.NoError(t, config.UpdateConfigFromMap(monitorConfig, map[string]string{"enabled": "false"}))

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

func TestChannelMonitorConfigRejectsUnsafeValues(t *testing.T) {
	_, err := normalizeChannelMonitorConfig(channelMonitorConfigRequest{
		IntervalMinutes: 10, Concurrency: 1, TimeoutSeconds: 5,
		ConfirmRetries: 0, ConfirmRetryDelaySeconds: 0, FailureThreshold: 1,
		ExcludePatterns: []string{"["},
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid wildcard"))
}

func configureNativeMonitorForTest(t *testing.T) {
	t.Helper()
	monitorConfig := config.GlobalConfig.Get("native_monitor_setting")
	original, err := config.ConfigToMap(monitorConfig)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, config.UpdateConfigFromMap(monitorConfig, original))
	})
	require.NoError(t, config.UpdateConfigFromMap(monitorConfig, map[string]string{
		"enabled": "true", "interval_minutes": "10", "concurrency": "1",
		"timeout_seconds": "5", "confirm_retries": "1", "confirm_retry_delay_seconds": "0",
		"failure_threshold": "2", "exclude_patterns": "[]",
	}))
}
