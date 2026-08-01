package operation_setting

import (
	"sync"
	"sync/atomic"

	"github.com/QuantumNous/new-api/setting/config"
)

const NativeMonitorHistoryLimit = 288

func NativeMonitorRetentionLimit(intervalMinutes int) int {
	if intervalMinutes < 1 {
		intervalMinutes = 1
	}
	windowSamples := (24*60+intervalMinutes-1)/intervalMinutes + 1
	if windowSamples < NativeMonitorHistoryLimit {
		return NativeMonitorHistoryLimit
	}
	return windowSamples
}

type NativeMonitorSetting struct {
	Enabled                  bool     `json:"enabled"`
	IntervalMinutes          int      `json:"interval_minutes"`
	Concurrency              int      `json:"concurrency"`
	TimeoutSeconds           int      `json:"timeout_seconds"`
	ConfirmRetries           int      `json:"confirm_retries"`
	ConfirmRetryDelaySeconds int      `json:"confirm_retry_delay_seconds"`
	FailureThreshold         int      `json:"failure_threshold"`
	ExcludePatterns          []string `json:"exclude_patterns"`
}

var (
	nativeMonitorSetting = NativeMonitorSetting{
		Enabled: false, IntervalMinutes: 10, Concurrency: 3, TimeoutSeconds: 30,
		ConfirmRetries: 1, ConfirmRetryDelaySeconds: 3, FailureThreshold: 3,
		ExcludePatterns: []string{},
	}
	nativeMonitorMu       sync.Mutex
	nativeMonitorSnapshot atomic.Pointer[NativeMonitorSetting]
)

func init() {
	config.GlobalConfig.Register("native_monitor_setting", &nativeMonitorSetting)
	PublishNativeMonitorSetting()
}

func PublishNativeMonitorSetting() {
	nativeMonitorMu.Lock()
	defer nativeMonitorMu.Unlock()
	snapshot := cloneNativeMonitorSetting(nativeMonitorSetting)
	nativeMonitorSnapshot.Store(&snapshot)
}

func UpdateNativeMonitorSettingFromMap(values map[string]string) error {
	nativeMonitorMu.Lock()
	defer nativeMonitorMu.Unlock()

	next := GetNativeMonitorSetting()
	if err := config.UpdateConfigFromMap(&next, values); err != nil {
		return err
	}
	serialized, err := config.ConfigToMap(&next)
	if err != nil {
		return err
	}
	if err := config.GlobalConfig.Update("native_monitor_setting", serialized); err != nil {
		return err
	}
	snapshot := cloneNativeMonitorSetting(next)
	nativeMonitorSnapshot.Store(&snapshot)
	return nil
}

func GetNativeMonitorSetting() NativeMonitorSetting {
	snapshot := nativeMonitorSnapshot.Load()
	if snapshot == nil {
		return NativeMonitorSetting{}
	}
	return cloneNativeMonitorSetting(*snapshot)
}

func cloneNativeMonitorSetting(setting NativeMonitorSetting) NativeMonitorSetting {
	setting.ExcludePatterns = append([]string(nil), setting.ExcludePatterns...)
	return setting
}
