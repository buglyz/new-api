package service

import "github.com/QuantumNous/new-api/setting/operation_setting"

// ResetAllPersonalCircuits clears every process-local circuit breaker.
func ResetAllPersonalCircuits() int {
	if !operation_setting.SelfUseModeEnabled {
		return 0
	}
	transitions := personalCircuits.resetAll()
	for _, transition := range transitions {
		notifyPersonalCircuitTransition(transition)
	}
	return len(transitions)
}

func (m *personalCircuitManager) resetAll() []PersonalCircuitTransition {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now().Unix()
	transitions := make([]PersonalCircuitTransition, 0, len(m.entries))
	for key, entry := range m.entries {
		delete(m.entries, key)
		if entry.Status == PersonalCircuitClosed {
			continue
		}
		transition := PersonalCircuitTransition{
			ChannelID: entry.ChannelID, ChannelName: entry.ChannelName, Model: entry.Model,
			From: entry.Status, To: PersonalCircuitClosed, At: now,
		}
		m.appendTransitionLocked(transition)
		transitions = append(transitions, transition)
	}
	return transitions
}
