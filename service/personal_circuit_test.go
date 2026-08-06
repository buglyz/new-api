package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func recordFailures(manager *personalCircuitManager, count, channelID int, channelName, modelName string, attempt RelayAttempt) *PersonalCircuitTransition {
	var transition *PersonalCircuitTransition
	for range count {
		transition = manager.recordFailure(channelID, channelName, modelName, attempt)
	}
	return transition
}

func TestPersonalCircuitRequiresConsecutiveFailuresBeforeOpening(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	manager := newPersonalCircuitManager(func() time.Time { return now })
	attempt := RelayAttempt{Outcome: RelayAttemptUpstream5xx, StatusCode: 503}

	for failure := 1; failure < personalCircuitFailureThreshold; failure++ {
		assert.Nil(t, manager.recordFailure(1, "public", "model", attempt))
		assert.True(t, manager.canAttempt(1, "model"))
		assert.True(t, manager.claim(1, "model"))
		circuits, transitions := manager.snapshot()
		assert.Empty(t, circuits)
		assert.Empty(t, transitions)
	}

	transition := manager.recordFailure(1, "public", "model", attempt)
	require.NotNil(t, transition)
	assert.Equal(t, PersonalCircuitOpen, transition.To)
	assert.False(t, manager.canAttempt(1, "model"))
	circuits, _ := manager.snapshot()
	require.Len(t, circuits, 1)
	assert.Equal(t, personalCircuitFailureThreshold, circuits[0].ConsecutiveFailures)
	assert.Equal(t, now.Add(personalCircuitBaseBackoff).Unix(), circuits[0].RetryAt)
}

func TestPersonalCircuitSuccessClearsFailureStreak(t *testing.T) {
	manager := newPersonalCircuitManager(time.Now)
	attempt := RelayAttempt{Outcome: RelayAttemptUpstream5xx, StatusCode: 503}

	recordFailures(manager, personalCircuitFailureThreshold-1, 1, "public", "model", attempt)
	assert.Empty(t, manager.recordSuccess(1, "model"))
	assert.Nil(t, manager.recordFailure(1, "public", "model", attempt))
	assert.True(t, manager.canAttempt(1, "model"))
	circuits, transitions := manager.snapshot()
	assert.Empty(t, circuits)
	assert.Empty(t, transitions)
}

func TestPersonalCircuitScopesModelNotFoundToModel(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	manager := newPersonalCircuitManager(func() time.Time { return now })
	attempt := RelayAttempt{Outcome: RelayAttemptModelUnavailable, StatusCode: 404, ErrorCode: "model_not_found"}

	transition := recordFailures(manager, personalCircuitFailureThreshold, 7, "public", "model-a", attempt)
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

	recordFailures(manager, personalCircuitFailureThreshold, 3, "low-sla", "model-a", attempt)
	circuits, _ := manager.snapshot()
	assert.Equal(t, now.Add(15*time.Second).Unix(), circuits[0].RetryAt)
	assert.False(t, manager.claim(3, "model-a"))

	now = now.Add(30 * time.Second)
	assert.True(t, manager.canAttempt(3, "model-a"))
	assert.True(t, manager.claim(3, "model-a"))
	assert.False(t, manager.claim(3, "model-a"))

	manager.recordFailure(3, "low-sla", "model-a", attempt)
	circuits, _ = manager.snapshot()
	assert.Equal(t, now.Add(30*time.Second).Unix(), circuits[0].RetryAt)

	now = now.Add(60 * time.Second)
	assert.True(t, manager.claim(3, "model-a"))
	transitions := manager.recordSuccess(3, "model-a")
	require.Len(t, transitions, 1)
	assert.Equal(t, PersonalCircuitClosed, transitions[0].To)
	assert.True(t, manager.canAttempt(3, "model-a"))
}

func TestPersonalCircuitOpensChannelWideForAuthAndConfigurationErrors(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	manager := newPersonalCircuitManager(func() time.Time { return now })
	require.NotNil(t, recordFailures(manager, personalCircuitFailureThreshold, 1, "channel", "model-a", RelayAttempt{Outcome: RelayAttemptAuthError, StatusCode: 401}))
	assert.False(t, manager.canAttempt(1, "model-a"))
	assert.False(t, manager.canAttempt(1, "model-b"))

	circuits, _ := manager.snapshot()
	require.Len(t, circuits, 1)
	assert.Equal(t, personalCircuitAllModels, circuits[0].Model)
	assert.Equal(t, "channel", circuits[0].Scope)
	assert.Equal(t, now.Add(personalCircuitAuthBackoff).Unix(), circuits[0].RetryAt)

	manager.recordSuccess(1, "model-b")
	require.NotNil(t, recordFailures(manager, personalCircuitFailureThreshold, 1, "channel", "model-a", RelayAttempt{Outcome: RelayAttemptChannelUnavailable}))
	assert.False(t, manager.canAttempt(1, "model-b"))
	circuits, _ = manager.snapshot()
	require.Len(t, circuits, 1)
	assert.Equal(t, now.Add(personalCircuitChannelBackoff).Unix(), circuits[0].RetryAt)
}

func TestPersonalCircuitScopesModelConfigurationErrorsToModel(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	manager := newPersonalCircuitManager(func() time.Time { return now })
	attempt := RelayAttempt{
		Outcome:   RelayAttemptChannelUnavailable,
		ErrorCode: string(types.ErrorCodeChannelModelMappedError),
	}

	require.NotNil(t, recordFailures(manager, personalCircuitFailureThreshold, 1, "channel", "model-a", attempt))
	assert.False(t, manager.canAttempt(1, "model-a"))
	assert.True(t, manager.canAttempt(1, "model-b"))
	circuits, _ := manager.snapshot()
	require.Len(t, circuits, 1)
	assert.Equal(t, "model", circuits[0].Scope)
	assert.Equal(t, now.Add(personalCircuitBaseBackoff).Unix(), circuits[0].RetryAt)
}

func TestPersonalCircuitHalfOpenClaimHasSingleProbeUntilLeaseExpires(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	manager := newPersonalCircuitManager(func() time.Time { return now })
	recordFailures(manager, personalCircuitFailureThreshold, 1, "channel", "model", RelayAttempt{Outcome: RelayAttemptUpstream5xx})

	now = now.Add(personalCircuitBaseBackoff)
	require.True(t, manager.claim(1, "model"))
	assert.False(t, manager.canAttempt(1, "model"))
	assert.False(t, manager.claim(1, "model"))

	now = now.Add(personalCircuitHalfOpenLease - time.Second)
	assert.False(t, manager.canAttempt(1, "model"))
	assert.False(t, manager.claim(1, "model"))

	now = now.Add(time.Second)
	assert.True(t, manager.canAttempt(1, "model"))
	assert.True(t, manager.claim(1, "model"))
}

func TestPersonalCircuitIgnoresLateResultFromExpiredHalfOpenProbe(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	manager := newPersonalCircuitManager(func() time.Time { return now })
	recordFailures(manager, personalCircuitFailureThreshold, 1, "channel", "model", RelayAttempt{Outcome: RelayAttemptUpstream5xx})

	now = now.Add(personalCircuitBaseBackoff)
	require.True(t, manager.claim(1, "model"))
	firstProbe := RelayAttempt{StartedAtMs: now.UnixMilli()}

	now = now.Add(personalCircuitHalfOpenLease)
	require.True(t, manager.claim(1, "model"))
	secondProbe := RelayAttempt{StartedAtMs: now.UnixMilli()}

	transitions := manager.recordSuccess(1, "model", secondProbe)
	require.Len(t, transitions, 1)
	assert.Equal(t, PersonalCircuitClosed, transitions[0].To)
	assert.Empty(t, manager.recordSuccess(1, "model", firstProbe))
	assert.Nil(t, manager.recordFailure(1, "channel", "model", RelayAttempt{
		StartedAtMs: firstProbe.StartedAtMs,
		Outcome:     RelayAttemptUpstream5xx,
	}))
	circuits, _ := manager.snapshot()
	assert.Empty(t, circuits)
}

func TestPersonalCircuitIgnoresLateFailureAfterNewProbeFails(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	manager := newPersonalCircuitManager(func() time.Time { return now })
	recordFailures(manager, personalCircuitFailureThreshold, 1, "channel", "model", RelayAttempt{Outcome: RelayAttemptUpstream5xx})

	now = now.Add(personalCircuitBaseBackoff)
	require.True(t, manager.claim(1, "model"))
	firstProbe := RelayAttempt{StartedAtMs: now.UnixMilli()}
	now = now.Add(personalCircuitHalfOpenLease)
	require.True(t, manager.claim(1, "model"))
	secondProbe := RelayAttempt{StartedAtMs: now.UnixMilli()}

	require.NotNil(t, manager.recordFailure(1, "channel", "model", RelayAttempt{
		StartedAtMs: secondProbe.StartedAtMs,
		Outcome:     RelayAttemptUpstream5xx,
	}))
	assert.Nil(t, manager.recordFailure(1, "channel", "model", RelayAttempt{
		StartedAtMs: firstProbe.StartedAtMs,
		Outcome:     RelayAttemptUpstream5xx,
	}))
	circuits, _ := manager.snapshot()
	require.Len(t, circuits, 1)
	assert.Equal(t, personalCircuitFailureThreshold+1, circuits[0].ConsecutiveFailures)
}

func TestPersonalCircuitDoesNotOpenForClientOrLocalErrors(t *testing.T) {
	manager := newPersonalCircuitManager(time.Now)
	assert.Nil(t, manager.recordFailure(1, "channel", "model", RelayAttempt{Outcome: RelayAttemptClientError, StatusCode: 400}))
	assert.Nil(t, manager.recordFailure(1, "channel", "model", RelayAttempt{Outcome: RelayAttemptLocalError}))
	circuits, _ := manager.snapshot()
	assert.Empty(t, circuits)
}

func TestPersonalCircuitOpensForRequestTimeouts(t *testing.T) {
	manager := newPersonalCircuitManager(time.Now)
	for _, statusCode := range []int{408, 425} {
		transition := recordFailures(manager, personalCircuitFailureThreshold, 1, "channel", "model", RelayAttempt{
			Outcome:    RelayAttemptTransportError,
			StatusCode: statusCode,
		})
		require.NotNil(t, transition)
		assert.False(t, manager.canAttempt(1, "model"))
		manager.recordSuccess(1, "model")
	}
}

func TestPersonalCircuitHonorsRetryAfterWithinCap(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	manager := newPersonalCircuitManager(func() time.Time { return now })
	recordFailures(manager, personalCircuitFailureThreshold, 1, "limited", "model", RelayAttempt{
		Outcome:           RelayAttemptRateLimited,
		StatusCode:        429,
		RetryAfterSeconds: 600,
	})
	circuits, _ := manager.snapshot()
	require.Len(t, circuits, 1)
	assert.Equal(t, now.Add(personalCircuitMaxBackoff).Unix(), circuits[0].RetryAt)

	recordFailures(manager, personalCircuitFailureThreshold, 2, "limited", "model", RelayAttempt{
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
	recordFailures(manager, personalCircuitFailureThreshold, 1, "channel", "tested-model", attempt)
	recordFailures(manager, personalCircuitFailureThreshold, 1, "channel", "other-model", attempt)

	transitions := manager.reset(map[int]struct{}{1: {}}, "tested-model")
	require.Len(t, transitions, 1)
	assert.True(t, manager.canAttempt(1, "tested-model"))
	assert.False(t, manager.canAttempt(1, "other-model"))
}

func TestPersonalCircuitResetModelAlsoClearsChannelWideCircuit(t *testing.T) {
	manager := newPersonalCircuitManager(time.Now)
	recordFailures(manager, personalCircuitFailureThreshold, 1, "channel", "model", RelayAttempt{Outcome: RelayAttemptAuthError, StatusCode: 401})
	assert.False(t, manager.canAttempt(1, "model"))

	transitions := manager.reset(map[int]struct{}{1: {}}, "model")
	require.Len(t, transitions, 1)
	assert.Equal(t, personalCircuitAllModels, transitions[0].Model)
	assert.True(t, manager.canAttempt(1, "model"))
}

func TestPersonalCircuitGateIsDisabledInStandardMode(t *testing.T) {
	previousMode := operation_setting.SelfUseModeEnabled
	previousCircuits := personalCircuits
	operation_setting.SelfUseModeEnabled = false
	personalCircuits = newPersonalCircuitManager(time.Now)
	recordFailures(personalCircuits, personalCircuitFailureThreshold, 1, "channel", "model", RelayAttempt{Outcome: RelayAttemptUpstream5xx})
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

	recordFailures(personalCircuits, personalCircuitFailureThreshold, 1, "channel", "model", RelayAttempt{Outcome: RelayAttemptUpstream5xx})

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
	recordFailures(personalCircuits, personalCircuitFailureThreshold, 1, "broken", "model-a", attempt)
	recordFailures(personalCircuits, personalCircuitFailureThreshold, 1, "broken", "model-b", attempt)
	recordFailures(personalCircuits, personalCircuitFailureThreshold, 2, "healthy", "model-a", attempt)
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
	recordFailures(personalCircuits, personalCircuitFailureThreshold, 1, "broken", "model-a", RelayAttempt{Outcome: RelayAttemptUpstream5xx})
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
	recordFailures(personalCircuits, personalCircuitFailureThreshold, 1, "edited", "model-a", attempt)
	recordFailures(personalCircuits, personalCircuitFailureThreshold, 1, "edited", "model-b", attempt)
	recordFailures(personalCircuits, personalCircuitFailureThreshold, 2, "untouched", "model-a", attempt)
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
	recordFailures(personalCircuits, personalCircuitFailureThreshold, 1, "channel", "model", RelayAttempt{Outcome: RelayAttemptUpstream5xx})

	operation_setting.SelfUseModeEnabled = false
	ForgetPersonalCircuits(1)

	operation_setting.SelfUseModeEnabled = true
	assert.False(t, PersonalCircuitCanAttempt(1, "model"))
}
