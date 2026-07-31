package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

const NativeMonitorHistoryLimit = 288

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

var nativeMonitorSetting = NativeMonitorSetting{
	Enabled: false, IntervalMinutes: 10, Concurrency: 3, TimeoutSeconds: 30,
	ConfirmRetries: 1, ConfirmRetryDelaySeconds: 3, FailureThreshold: 3,
	ExcludePatterns: []string{},
}

func init() {
	config.GlobalConfig.Register("native_monitor_setting", &nativeMonitorSetting)
}

func GetNativeMonitorSetting() NativeMonitorSetting { return nativeMonitorSetting }
