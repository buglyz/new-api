package model

import (
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const nativeMonitorOptionPrefix = "native_monitor_setting."

var nativeMonitorOptionUpdateMu sync.Mutex

func applyOptionsFromDatabase(options []*Option) {
	nativeMonitorValues := make(map[string]string)
	for _, option := range options {
		if field, ok := nativeMonitorOptionField(option.Key); ok {
			storeOptionMapValue(option.Key, option.Value)
			nativeMonitorValues[field] = option.Value
			continue
		}
		if err := updateOptionMap(option.Key, option.Value); err != nil {
			common.SysLog("failed to update option map: " + err.Error())
		}
	}
	if err := operation_setting.UpdateNativeMonitorSettingFromMap(nativeMonitorValues); err != nil {
		common.SysLog("failed to update native monitor settings: " + err.Error())
	}
}

func updateNativeMonitorOption(key, value string) error {
	field, ok := nativeMonitorOptionField(key)
	if !ok {
		return nil
	}
	storeOptionMapValue(key, value)
	return operation_setting.UpdateNativeMonitorSettingFromMap(map[string]string{field: value})
}

func applyUpdatedOptionValues(values map[string]string) error {
	nativeMonitorValues := make(map[string]string)
	for key, value := range values {
		if field, ok := nativeMonitorOptionField(key); ok {
			storeOptionMapValue(key, value)
			nativeMonitorValues[field] = value
			continue
		}
		if err := updateOptionMap(key, value); err != nil {
			return err
		}
	}
	if len(nativeMonitorValues) == 0 {
		return nil
	}
	return operation_setting.UpdateNativeMonitorSettingFromMap(nativeMonitorValues)
}

func nativeMonitorOptionField(key string) (string, bool) {
	if !strings.HasPrefix(key, nativeMonitorOptionPrefix) {
		return "", false
	}
	return strings.TrimPrefix(key, nativeMonitorOptionPrefix), true
}

func containsNativeMonitorOptions(values map[string]string) bool {
	for key := range values {
		if _, ok := nativeMonitorOptionField(key); ok {
			return true
		}
	}
	return false
}

func storeOptionMapValue(key, value string) {
	common.OptionMapRWMutex.Lock()
	common.OptionMap[key] = value
	common.OptionMapRWMutex.Unlock()
}
