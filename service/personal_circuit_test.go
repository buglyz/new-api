package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersonalCircuitScopesModelNotFoundToChannelAndModel(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	manager := newPersonalCircuitManager(func() time.Time { return now })
	attempt := RelayAttempt{Outcome: RelayAttemptModelUnavailable, StatusCode: 404, ErrorCode: "model_not_found"}

	transition := manager.recordFailure(7, "public", "model-a", attempt)
	require.NotNil(t, transition)
	assert.False(t, manager.canAttempt(7, "model-a"))
	assert.True(t, manager.canAttempt(7, "model-b"))
	assert.True(t, manager.canAttempt(8, "model-a"))

	circuits, _ := manager.snapshot()
	require.Len(t, circuits, 1)
	assert.Equal(t, now.Add(personalCircuitModelBackoff).Unix(), circuits[0].RetryAt)
}

func TestPersonalCircuitUsesExponentialBackoffAndHalfOpenLease(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	manager := newPersonalCircuitManager(func() time.Time { return now })
	attempt := RelayAttempt{Outcome: RelayAttemptUpstream5xx, StatusCode: 503}

	manager.recordFailure(3, "low-sla", "model-a", attempt)
	circuits, _ := manager.snapshot()
	assert.Equal(t, now.Add(30*time.Second).Unix(), circuits[0].RetryAt)

	now = now.Add(30 * time.Second)
	assert.True(t, manager.canAttempt(3, "model-a"))
	assert.True(t, manager.claim(3, "model-a", false))
	assert.False(t, manager.claim(3, "model-a", false))

	manager.recordFailure(3, "low-sla", "model-a", attempt)
	circuits, _ = manager.snapshot()
	assert.Equal(t, now.Add(60*time.Second).Unix(), circuits[0].RetryAt)

	now = now.Add(60 * time.Second)
	assert.True(t, manager.claim(3, "model-a", false))
	transition := manager.recordSuccess(3, "model-a")
	require.NotNil(t, transition)
	assert.Equal(t, PersonalCircuitClosed, transition.To)
	assert.True(t, manager.canAttempt(3, "model-a"))
}

func TestPersonalCircuitDoesNotOpenForAuthOrClientErrors(t *testing.T) {
	manager := newPersonalCircuitManager(time.Now)
	assert.Nil(t, manager.recordFailure(1, "channel", "model", RelayAttempt{Outcome: RelayAttemptAuthError, StatusCode: 401}))
	assert.Nil(t, manager.recordFailure(1, "channel", "model", RelayAttempt{Outcome: RelayAttemptClientError, StatusCode: 400}))
	assert.Nil(t, manager.recordFailure(1, "channel", "model", RelayAttempt{Outcome: RelayAttemptChannelUnavailable}))
	assert.Nil(t, manager.recordFailure(1, "channel", "model", RelayAttempt{Outcome: RelayAttemptLocalError}))
	circuits, _ := manager.snapshot()
	assert.Empty(t, circuits)
}

func TestPersonalCircuitNotificationSuppressionIsPerTransition(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	manager := newPersonalCircuitManager(func() time.Time { return now })
	transition := PersonalCircuitTransition{ChannelID: 1, Model: "m", To: PersonalCircuitOpen}
	assert.True(t, manager.shouldNotify(transition))
	assert.False(t, manager.shouldNotify(transition))

	transition.To = PersonalCircuitClosed
	assert.True(t, manager.shouldNotify(transition))
	now = now.Add(personalCircuitNotifySuppression)
	transition.To = PersonalCircuitOpen
	assert.True(t, manager.shouldNotify(transition))
}

func TestPersonalCircuitResetKeepsUntestedModelsOpen(t *testing.T) {
	manager := newPersonalCircuitManager(time.Now)
	attempt := RelayAttempt{Outcome: RelayAttemptModelUnavailable, StatusCode: 404}
	manager.recordFailure(1, "channel", "tested-model", attempt)
	manager.recordFailure(1, "channel", "other-model", attempt)

	transitions := manager.reset(map[int]struct{}{1: {}}, "tested-model")
	require.Len(t, transitions, 1)
	assert.True(t, manager.canAttempt(1, "tested-model"))
	assert.False(t, manager.canAttempt(1, "other-model"))
}

func TestPersonalCircuitGateIsDisabledInStandardMode(t *testing.T) {
	previousMode := operation_setting.SelfUseModeEnabled
	previousCircuits := personalCircuits
	operation_setting.SelfUseModeEnabled = false
	personalCircuits = newPersonalCircuitManager(time.Now)
	personalCircuits.recordFailure(1, "channel", "model", RelayAttempt{Outcome: RelayAttemptUpstream5xx})
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = previousMode
		personalCircuits = previousCircuits
	})

	assert.True(t, PersonalCircuitCanAttempt(1, "model"))
	assert.True(t, ClaimPersonalCircuit(1, "model", false))
}
