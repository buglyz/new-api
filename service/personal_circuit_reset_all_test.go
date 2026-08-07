package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersonalCircuitResetAllClearsActiveAndPendingState(t *testing.T) {
	manager := newPersonalCircuitManager(time.Now)
	attempt := RelayAttempt{Outcome: RelayAttemptUpstream5xx, StatusCode: 503}
	recordFailures(manager, personalCircuitFailureThreshold-1, 1, "pending", "model-a", attempt)
	recordFailures(manager, personalCircuitFailureThreshold, 2, "open", "model-a", attempt)
	recordFailures(manager, personalCircuitFailureThreshold, 3, "open", "model-b", attempt)

	transitions := manager.resetAll()
	require.Len(t, transitions, 2)
	for _, transition := range transitions {
		assert.Equal(t, PersonalCircuitClosed, transition.To)
	}
	circuits, _ := manager.snapshot()
	assert.Empty(t, circuits)

	// The pending failures from before the reset must not count toward the next window.
	recordFailures(manager, personalCircuitFailureThreshold-1, 1, "pending", "model-a", attempt)
	circuits, _ = manager.snapshot()
	assert.Empty(t, circuits)
}

func TestResetAllPersonalCircuitsIsInertInStandardMode(t *testing.T) {
	previousMode := operation_setting.SelfUseModeEnabled
	previousCircuits := personalCircuits
	operation_setting.SelfUseModeEnabled = true
	personalCircuits = newPersonalCircuitManager(time.Now)
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = previousMode
		personalCircuits = previousCircuits
	})

	recordFailures(personalCircuits, personalCircuitFailureThreshold, 1, "broken", "model", RelayAttempt{Outcome: RelayAttemptUpstream5xx})
	operation_setting.SelfUseModeEnabled = false

	assert.Equal(t, 0, ResetAllPersonalCircuits())
	operation_setting.SelfUseModeEnabled = true
	assert.False(t, PersonalCircuitCanAttempt(1, "model"))
}

func TestResetAllPersonalCircuitsReturnsActiveCount(t *testing.T) {
	previousMode := operation_setting.SelfUseModeEnabled
	previousCircuits := personalCircuits
	operation_setting.SelfUseModeEnabled = true
	personalCircuits = newPersonalCircuitManager(time.Now)
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = previousMode
		personalCircuits = previousCircuits
	})

	attempt := RelayAttempt{Outcome: RelayAttemptUpstream5xx, StatusCode: 503}
	recordFailures(personalCircuits, personalCircuitFailureThreshold-1, 1, "pending", "model-a", attempt)
	recordFailures(personalCircuits, personalCircuitFailureThreshold, 2, "open", "model-a", attempt)
	recordFailures(personalCircuits, personalCircuitFailureThreshold, 3, "open", "model-b", attempt)

	assert.Equal(t, 2, ResetAllPersonalCircuits())
	assert.True(t, PersonalCircuitCanAttempt(2, "model-a"))
	assert.True(t, PersonalCircuitCanAttempt(3, "model-b"))
}
