package service

import (
	"time"

	"github.com/QuantumNous/new-api/relaykit/types"
)

const (
	PersonalCircuitClosed   PersonalCircuitStatus = "closed"
	PersonalCircuitOpen     PersonalCircuitStatus = "open"
	PersonalCircuitHalfOpen PersonalCircuitStatus = "half_open"
)

const (
	personalCircuitBaseBackoff       = 15 * time.Second
	personalCircuitMaxBackoff        = 5 * time.Minute
	personalCircuitModelBackoff      = 10 * time.Minute
	personalCircuitAuthBackoff       = 15 * time.Minute
	personalCircuitChannelBackoff    = 10 * time.Minute
	personalCircuitHalfOpenLease     = 2 * time.Minute
	personalCircuitWindow            = 10 * time.Minute
	personalCircuitFailureThreshold  = 10
	personalCircuitNotifySuppression = 10 * time.Minute
	personalCircuitMaxEntries        = 4096
	personalCircuitMaxNotifications  = 1024
	personalCircuitAllModels         = "*"
)

// opensPersonalCircuit reports whether an outcome participates in personal
// circuit state at all. Rate limiting is deliberately excluded: a 429 means
// the upstream is alive but busy, so it must not count toward the
// "persistently unavailable" judgement.
func opensPersonalCircuit(outcome RelayAttemptOutcome) bool {
	return outcome == RelayAttemptTransportError || outcome == RelayAttemptUpstream5xx ||
		outcome == RelayAttemptModelUnavailable || outcome == RelayAttemptAuthError ||
		outcome == RelayAttemptChannelUnavailable
}

// deterministicOutcome reports whether a failure can never self-heal without
// operator action (bad credentials, missing model, bad channel config).
// Such failures trip the circuit on the first occurrence instead of waiting
// for a sample window: every retry would fail the same way.
func deterministicOutcome(attempt RelayAttempt) bool {
	switch attempt.Outcome {
	case RelayAttemptAuthError, RelayAttemptModelUnavailable, RelayAttemptChannelUnavailable:
		return true
	default:
		return false
	}
}

func personalCircuitModel(attempt RelayAttempt, modelName string) string {
	if attempt.Outcome == RelayAttemptAuthError ||
		(attempt.Outcome == RelayAttemptChannelUnavailable && !isModelScopedChannelError(attempt)) {
		return personalCircuitAllModels
	}
	return modelName
}

func isModelScopedChannelError(attempt RelayAttempt) bool {
	switch types.ErrorCode(attempt.ErrorCode) {
	case types.ErrorCodeChannelParamOverrideInvalid,
		types.ErrorCodeChannelHeaderOverrideInvalid,
		types.ErrorCodeChannelModelMappedError:
		return true
	default:
		return false
	}
}

func personalCircuitScope(modelName string) string {
	if modelName == personalCircuitAllModels {
		return "channel"
	}
	return "model"
}

func personalCircuitBackoff(attempt RelayAttempt, failures int) time.Duration {
	if attempt.Outcome == RelayAttemptModelUnavailable {
		return personalCircuitModelBackoff
	}
	if attempt.Outcome == RelayAttemptAuthError {
		return personalCircuitAuthBackoff
	}
	if attempt.Outcome == RelayAttemptChannelUnavailable && !isModelScopedChannelError(attempt) {
		return personalCircuitChannelBackoff
	}
	backoff := personalCircuitBaseBackoff
	for i := 1; i < failures && backoff < personalCircuitMaxBackoff; i++ {
		backoff *= 2
	}
	if backoff > personalCircuitMaxBackoff {
		backoff = personalCircuitMaxBackoff
	}
	return backoff
}
