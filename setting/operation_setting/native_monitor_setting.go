package operation_setting

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/QuantumNous/new-api/common"
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
	if err := ValidateNativeMonitorSettingValues(values); err != nil {
		return err
	}
	nativeMonitorMu.Lock()
	defer nativeMonitorMu.Unlock()

	next := GetNativeMonitorSetting()
	if err := config.UpdateConfigFromMap(&next, values); err != nil {
		return err
	}
	next, err := NormalizeNativeMonitorSetting(next)
	if err != nil {
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
	setting.ExcludePatterns = append([]string{}, setting.ExcludePatterns...)
	return setting
}

func IsNativeMonitorSettingField(field string) bool {
	switch field {
	case "enabled", "interval_minutes", "concurrency", "timeout_seconds", "confirm_retries",
		"confirm_retry_delay_seconds", "failure_threshold", "exclude_patterns":
		return true
	default:
		return false
	}
}

func ValidateNativeMonitorSettingValues(values map[string]string) error {
	for field, value := range values {
		if !IsNativeMonitorSettingField(field) {
			return fmt.Errorf("unknown native monitor setting field: %s", field)
		}
		switch field {
		case "enabled":
			if _, err := strconv.ParseBool(value); err != nil {
				return fmt.Errorf("enabled must be a boolean")
			}
		case "exclude_patterns":
			var patterns []string
			if err := common.Unmarshal([]byte(value), &patterns); err != nil {
				return fmt.Errorf("exclude_patterns must be a JSON array")
			}
			if _, err := NormalizeNativeMonitorSetting(NativeMonitorSetting{
				IntervalMinutes: 1, Concurrency: 1, TimeoutSeconds: 1, FailureThreshold: 1,
				ExcludePatterns: patterns,
			}); err != nil {
				return err
			}
		default:
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("%s must be an integer", field)
			}
			min, max := nativeMonitorSettingBounds(field)
			if parsed < min || parsed > max {
				return fmt.Errorf("%s must be between %d and %d", field, min, max)
			}
		}
	}
	return nil
}

func NormalizeNativeMonitorSetting(setting NativeMonitorSetting) (NativeMonitorSetting, error) {
	if setting.IntervalMinutes < 1 || setting.IntervalMinutes > 1440 {
		return setting, fmt.Errorf("interval_minutes must be between 1 and 1440")
	}
	if setting.Concurrency < 1 || setting.Concurrency > 32 {
		return setting, fmt.Errorf("concurrency must be between 1 and 32")
	}
	if setting.TimeoutSeconds < 1 || setting.TimeoutSeconds > 120 {
		return setting, fmt.Errorf("timeout_seconds must be between 1 and 120")
	}
	if setting.ConfirmRetries < 0 || setting.ConfirmRetries > 3 {
		return setting, fmt.Errorf("confirm_retries must be between 0 and 3")
	}
	if setting.ConfirmRetryDelaySeconds < 0 || setting.ConfirmRetryDelaySeconds > 60 {
		return setting, fmt.Errorf("confirm_retry_delay_seconds must be between 0 and 60")
	}
	if setting.FailureThreshold < 1 || setting.FailureThreshold > 10 {
		return setting, fmt.Errorf("failure_threshold must be between 1 and 10")
	}
	if len(setting.ExcludePatterns) > 100 {
		return setting, fmt.Errorf("exclude_patterns cannot contain more than 100 entries")
	}
	patterns := make([]string, 0, len(setting.ExcludePatterns))
	for _, pattern := range setting.ExcludePatterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || len(pattern) > 255 {
			return setting, fmt.Errorf("each exclude pattern must be between 1 and 255 characters")
		}
		if _, err := path.Match(strings.ToLower(pattern), "probe-model"); err != nil {
			return setting, fmt.Errorf("exclude_patterns contains an invalid wildcard")
		}
		patterns = append(patterns, pattern)
	}
	setting.ExcludePatterns = patterns
	return setting, nil
}

func nativeMonitorSettingBounds(field string) (int, int) {
	switch field {
	case "interval_minutes":
		return 1, 1440
	case "concurrency":
		return 1, 32
	case "timeout_seconds":
		return 1, 120
	case "confirm_retries":
		return 0, 3
	case "confirm_retry_delay_seconds":
		return 0, 60
	case "failure_threshold":
		return 1, 10
	default:
		return 0, 0
	}
}
