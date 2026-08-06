package service

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/bytedance/gopkg/util/gopool"
)

func PersonalCircuitCanAttempt(channelID int, modelName string) bool {
	return !operation_setting.SelfUseModeEnabled || personalCircuits.canAttempt(channelID, modelName)
}

// ClaimPersonalCircuit accepts the retired force argument for source
// compatibility only. Cooldowns are always enforced by the manager.
func ClaimPersonalCircuit(channelID int, modelName string, _ ...bool) bool {
	return !operation_setting.SelfUseModeEnabled || personalCircuits.claim(channelID, modelName)
}

func RecordPersonalCircuitFailure(channelID int, channelName, modelName string, attempt RelayAttempt) {
	if !operation_setting.SelfUseModeEnabled {
		return
	}
	if transition := personalCircuits.recordFailure(channelID, channelName, modelName, attempt); transition != nil {
		notifyPersonalCircuitTransition(*transition)
	}
}

func RecordPersonalCircuitSuccess(channelID int, modelName string, attempt ...RelayAttempt) {
	if !operation_setting.SelfUseModeEnabled {
		return
	}
	for _, transition := range personalCircuits.recordSuccess(channelID, modelName, attempt...) {
		notifyPersonalCircuitTransition(transition)
	}
}

func ResetPersonalCircuits(channelIDs []int) int {
	if !operation_setting.SelfUseModeEnabled {
		return 0
	}
	ids := positiveChannelIDs(channelIDs)
	if len(ids) == 0 {
		return 0
	}
	transitions := personalCircuits.reset(ids, "")
	for _, transition := range transitions {
		notifyPersonalCircuitTransition(transition)
	}
	return len(transitions)
}

func positiveChannelIDs(channelIDs []int) map[int]struct{} {
	ids := make(map[int]struct{}, len(channelIDs))
	for _, channelID := range channelIDs {
		if channelID > 0 {
			ids[channelID] = struct{}{}
		}
	}
	return ids
}

// ForgetPersonalCircuits drops stale cooldown state after channel changes.
func ForgetPersonalCircuits(channelIDs ...int) {
	if !operation_setting.SelfUseModeEnabled {
		return
	}
	if ids := positiveChannelIDs(channelIDs); len(ids) > 0 {
		personalCircuits.reset(ids, "")
	}
}

func ResetPersonalCircuit(channelID int, modelName string) bool {
	if !operation_setting.SelfUseModeEnabled || channelID <= 0 || modelName == "" {
		return false
	}
	transitions := personalCircuits.reset(map[int]struct{}{channelID: {}}, modelName)
	for _, transition := range transitions {
		notifyPersonalCircuitTransition(transition)
	}
	return len(transitions) > 0
}

func GetPersonalCircuitSnapshot() ([]PersonalCircuit, []PersonalCircuitTransition, PersonalCircuitPolicy) {
	circuits, transitions := personalCircuits.snapshot()
	return circuits, transitions, PersonalCircuitPolicy{
		BaseBackoffSeconds:    int64(personalCircuitBaseBackoff / time.Second),
		MaxBackoffSeconds:     int64(personalCircuitMaxBackoff / time.Second),
		ModelBackoffSeconds:   int64(personalCircuitModelBackoff / time.Second),
		AuthBackoffSeconds:    int64(personalCircuitAuthBackoff / time.Second),
		ChannelBackoffSeconds: int64(personalCircuitChannelBackoff / time.Second),
		HalfOpenLeaseSeconds:  int64(personalCircuitHalfOpenLease / time.Second),
		Volatile:              true,
	}
}

func notifyPersonalCircuitTransition(transition PersonalCircuitTransition) {
	if !personalCircuits.shouldNotify(transition) || (transition.To != PersonalCircuitOpen && transition.To != PersonalCircuitClosed) {
		return
	}
	gopool.Go(func() {
		state := "恢复"
		if transition.To == PersonalCircuitOpen {
			state = "临时熔断"
		}
		subject := fmt.Sprintf("渠道 #%d 模型 %s %s", transition.ChannelID, transition.Model, state)
		content := fmt.Sprintf("渠道 #%d，模型 %s，状态 %s", transition.ChannelID, transition.Model, transition.To)
		if transition.StatusCode > 0 {
			content += fmt.Sprintf("，HTTP %d", transition.StatusCode)
		}
		if transition.Outcome != "" {
			content += fmt.Sprintf("，结果 %s", transition.Outcome)
		}
		NotifyRootUser(fmt.Sprintf("personal_circuit_%d_%s_%s", transition.ChannelID, transition.Model, transition.To), subject, content)
	})
}
