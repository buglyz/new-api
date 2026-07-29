package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersonalCircuitScopesModelNotFoundToModel(t *testing.T) {
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
	assert.Equal(t, "model", circuits[0].Scope)
	assert.Equal(t, now.Add(personalCircuitModelBackoff).Unix(), circuits[0].RetryAt)
}

func TestPersonalCircuitUsesExponentialBackoffAndHalfOpenLease(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	manager := newPersonalCircuitManager(func() time.Time { return now })
	attempt := RelayAttempt{Outcome: RelayAttemptUpstream5xx, StatusCode: 503}

	manager.recordFailure(3, "low-sla", "model-a", attempt)
	circuits, _ := manager.snapshot()
	assert.Equal(t, now.Add(30*time.Second).Unix(), circuits[0].RetryAt)
	assert.False(t, manager.claim(3, "model-a"))

	now = now.Add(30 * time.Second)
	assert.True(t, manager.canAttempt(3, "model-a"))
	assert.True(t, manager.claim(3, "model-a"))
	assert.False(t, manager.claim(3, "model-a"))

	manager.recordFailure(3, "low-sla", "model-a", attempt)
	circuits, _ = manager.snapshot()
	assert.Equal(t, now.Add(60*time.Second).Unix(), circuits[0].RetryAt)

	now = now.Add(60 * time.Second)
	assert.True(t, manager.claim(3, "model-a"))
	transitions := manager.recordSuccess(3, "model-a")
	require.Len(t, transitions, 1)
	assert.Equal(t, PersonalCircuitClosed, transitions[0].To)
	assert.True(t, manager.canAttempt(3, "model-a"))
}

func TestPersonalCircuitOpensChannelWideForAuthAndConfigurationErrors(t *testing.T) {
	manager := newPersonalCircuitManager(time.Now)
	require.NotNil(t, manager.recordFailure(1, "channel", "model-a", RelayAttempt{Outcome: RelayAttemptAuthError, StatusCode: 401}))
	assert.False(t, manager.canAttempt(1, "model-a"))
	assert.False(t, manager.canAttempt(1, "model-b"))

	circuits, _ := manager.snapshot()
	require.Len(t, circuits, 1)
	assert.Equal(t, personalCircuitAllModels, circuits[0].Model)
	assert.Equal(t, "channel", circuits[0].Scope)

	manager.recordSuccess(1, "model-b")
	require.NotNil(t, manager.recordFailure(1, "channel", "model-a", RelayAttempt{Outcome: RelayAttemptChannelUnavailable}))
	assert.False(t, manager.canAttempt(1, "model-b"))
}

func TestPersonalCircuitDoesNotOpenForClientOrLocalErrors(t *testing.T) {
	manager := newPersonalCircuitManager(time.Now)
	assert.Nil(t, manager.recordFailure(1, "channel", "model", RelayAttempt{Outcome: RelayAttemptClientError, StatusCode: 400}))
	assert.Nil(t, manager.recordFailure(1, "channel", "model", RelayAttempt{Outcome: RelayAttemptLocalError}))
	circuits, _ := manager.snapshot()
	assert.Empty(t, circuits)
}

func TestPersonalCircuitHonorsRetryAfterWithinCap(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	manager := newPersonalCircuitManager(func() time.Time { return now })
	manager.recordFailure(1, "limited", "model", RelayAttempt{
		Outcome:           RelayAttemptRateLimited,
		StatusCode:        429,
		RetryAfterSeconds: 600,
	})
	circuits, _ := manager.snapshot()
	require.Len(t, circuits, 1)
	assert.Equal(t, now.Add(10*time.Minute).Unix(), circuits[0].RetryAt)

	manager.recordFailure(2, "limited", "model", RelayAttempt{
		Outcome:           RelayAttemptRateLimited,
		StatusCode:        429,
		RetryAfterSeconds: int((time.Hour) / time.Second),
	})
	circuits, _ = manager.snapshot()
	assert.Equal(t, now.Add(personalCircuitMaxBackoff).Unix(), circuits[1].RetryAt)
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
	assert.True(t, ClaimPersonalCircuit(1, "model"))
}

func TestClaimPersonalCircuitLegacyForceCannotBypassCooldown(t *testing.T) {
	previousMode := operation_setting.SelfUseModeEnabled
	previousCircuits := personalCircuits
	operation_setting.SelfUseModeEnabled = true
	personalCircuits = newPersonalCircuitManager(time.Now)
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = previousMode
		personalCircuits = previousCircuits
	})

	personalCircuits.recordFailure(1, "channel", "model", RelayAttempt{Outcome: RelayAttemptUpstream5xx})

	assert.False(t, ClaimPersonalCircuit(1, "model", true))
}

func TestResetPersonalCircuitsClearsEveryModelForAChannel(t *testing.T) {
	previousMode := operation_setting.SelfUseModeEnabled
	previousCircuits := personalCircuits
	operation_setting.SelfUseModeEnabled = true
	personalCircuits = newPersonalCircuitManager(time.Now)
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = previousMode
		personalCircuits = previousCircuits
	})

	attempt := RelayAttempt{Outcome: RelayAttemptUpstream5xx, StatusCode: 503}
	personalCircuits.recordFailure(1, "broken", "model-a", attempt)
	personalCircuits.recordFailure(1, "broken", "model-b", attempt)
	personalCircuits.recordFailure(2, "healthy", "model-a", attempt)
	require.False(t, PersonalCircuitCanAttempt(1, "model-a"))

	assert.Equal(t, 2, ResetPersonalCircuits([]int{1}))
	assert.True(t, PersonalCircuitCanAttempt(1, "model-a"))
	assert.True(t, PersonalCircuitCanAttempt(1, "model-b"))
	assert.False(t, PersonalCircuitCanAttempt(2, "model-a"))
}

func TestResetPersonalCircuitsIsInertInStandardMode(t *testing.T) {
	previousMode := operation_setting.SelfUseModeEnabled
	previousCircuits := personalCircuits
	operation_setting.SelfUseModeEnabled = true
	personalCircuits = newPersonalCircuitManager(time.Now)
	personalCircuits.recordFailure(1, "broken", "model-a", RelayAttempt{Outcome: RelayAttemptUpstream5xx})
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = previousMode
		personalCircuits = previousCircuits
	})

	operation_setting.SelfUseModeEnabled = false
	assert.Equal(t, 0, ResetPersonalCircuits([]int{1}))
}

func TestForgetPersonalCircuitsClearsStateWithoutNotifying(t *testing.T) {
	previousMode := operation_setting.SelfUseModeEnabled
	previousCircuits := personalCircuits
	operation_setting.SelfUseModeEnabled = true
	personalCircuits = newPersonalCircuitManager(time.Now)
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = previousMode
		personalCircuits = previousCircuits
	})

	attempt := RelayAttempt{Outcome: RelayAttemptUpstream5xx, StatusCode: 503}
	personalCircuits.recordFailure(1, "edited", "model-a", attempt)
	personalCircuits.recordFailure(1, "edited", "model-b", attempt)
	personalCircuits.recordFailure(2, "untouched", "model-a", attempt)
	require.False(t, PersonalCircuitCanAttempt(1, "model-a"))

	ForgetPersonalCircuits(1)

	// The reconfigured channel is immediately eligible again on every model,
	// while unrelated channels keep serving out their cooldown.
	assert.True(t, PersonalCircuitCanAttempt(1, "model-a"))
	assert.True(t, PersonalCircuitCanAttempt(1, "model-b"))
	assert.False(t, PersonalCircuitCanAttempt(2, "model-a"))

	circuits, _ := personalCircuits.snapshot()
	require.Len(t, circuits, 1)
	assert.Equal(t, 2, circuits[0].ChannelID)
}

func TestForgetPersonalCircuitsIsInertInStandardMode(t *testing.T) {
	previousMode := operation_setting.SelfUseModeEnabled
	previousCircuits := personalCircuits
	operation_setting.SelfUseModeEnabled = true
	personalCircuits = newPersonalCircuitManager(time.Now)
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = previousMode
		personalCircuits = previousCircuits
	})
	personalCircuits.recordFailure(1, "channel", "model", RelayAttempt{Outcome: RelayAttemptUpstream5xx})

	operation_setting.SelfUseModeEnabled = false
	ForgetPersonalCircuits(1)

	operation_setting.SelfUseModeEnabled = true
	assert.False(t, PersonalCircuitCanAttempt(1, "model"))
}
