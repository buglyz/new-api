package service

import (
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

type RelayAttemptOutcome string

const (
	RelayAttemptSuccess            RelayAttemptOutcome = "success"
	RelayAttemptTransportError     RelayAttemptOutcome = "transport_error"
	RelayAttemptRateLimited        RelayAttemptOutcome = "rate_limited"
	RelayAttemptUpstream5xx        RelayAttemptOutcome = "upstream_5xx"
	RelayAttemptAuthError          RelayAttemptOutcome = "auth_error"
	RelayAttemptModelUnavailable   RelayAttemptOutcome = "model_unavailable"
	RelayAttemptChannelUnavailable RelayAttemptOutcome = "channel_unavailable"
	RelayAttemptClientError        RelayAttemptOutcome = "client_error"
	RelayAttemptLocalError         RelayAttemptOutcome = "local_error"
	RelayAttemptUpstreamError      RelayAttemptOutcome = "upstream_error"
)

type RelayAttempt struct {
	Index             int                 `json:"index"`
	ChannelID         int                 `json:"channel_id"`
	Model             string              `json:"model"`
	StartedAtMs       int64               `json:"started_at_ms"`
	DurationMs        int64               `json:"duration_ms"`
	Outcome           RelayAttemptOutcome `json:"outcome"`
	StatusCode        int                 `json:"status_code,omitempty"`
	ErrorCode         string              `json:"error_code,omitempty"`
	RetryAfterSeconds int                 `json:"retry_after_seconds,omitempty"`
	Retried           bool                `json:"retried"`
	startedAt         time.Time
}

type RelayAttemptTrace struct {
	Version        int            `json:"version"`
	RequestID      string         `json:"request_id,omitempty"`
	Model          string         `json:"model"`
	Result         string         `json:"result"`
	RetryCount     int            `json:"retry_count"`
	FinalChannelID int            `json:"final_channel_id,omitempty"`
	Attempts       []RelayAttempt `json:"attempts"`
}

type relayAttemptState struct {
	mu       sync.Mutex
	attempts []RelayAttempt
}

const relayAttemptContextKey = "personal_relay_attempt_state"

func BeginRelayAttempt(c *gin.Context, channelID int, modelName string) int {
	if c == nil || !operation_setting.SelfUseModeEnabled || channelID <= 0 {
		return -1
	}
	state := getRelayAttemptState(c, true)
	if state == nil {
		return -1
	}
	now := time.Now()
	state.mu.Lock()
	defer state.mu.Unlock()
	index := len(state.attempts)
	state.attempts = append(state.attempts, RelayAttempt{
		Index:       index,
		ChannelID:   channelID,
		Model:       modelName,
		StartedAtMs: now.UnixMilli(),
		startedAt:   now,
	})
	return index
}

func CompleteRelayAttempt(c *gin.Context, index int, relayErr *types.NewAPIError, retried bool) RelayAttempt {
	outcome := ClassifyRelayAttempt(relayErr)
	return completeRelayAttempt(c, index, outcome, relayErr, retried)
}

func CompleteLocalRelayAttempt(c *gin.Context, index int) RelayAttempt {
	return completeRelayAttempt(c, index, RelayAttemptLocalError, nil, false)
}

func completeRelayAttempt(c *gin.Context, index int, outcome RelayAttemptOutcome, relayErr *types.NewAPIError, retried bool) RelayAttempt {
	state := getRelayAttemptState(c, false)
	if state == nil || index < 0 {
		return RelayAttempt{}
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if index >= len(state.attempts) {
		return RelayAttempt{}
	}
	attempt := &state.attempts[index]
	attempt.DurationMs = max(time.Since(attempt.startedAt).Milliseconds(), 0)
	attempt.Outcome = outcome
	attempt.Retried = retried
	if relayErr != nil {
		attempt.StatusCode = relayErr.StatusCode
		attempt.ErrorCode = string(relayErr.GetErrorCode())
		attempt.RetryAfterSeconds = relayErr.RetryAfterSeconds()
	}
	return *attempt
}

func ClassifyRelayAttempt(relayErr *types.NewAPIError) RelayAttemptOutcome {
	if relayErr == nil {
		return RelayAttemptSuccess
	}
	switch relayErr.GetErrorCode() {
	case types.ErrorCodeDoRequestFailed, types.ErrorCodeChannelResponseTimeExceeded:
		return RelayAttemptTransportError
	case types.ErrorCodeModelNotFound:
		return RelayAttemptModelUnavailable
	case types.ErrorCodeChannelNoAvailableKey:
		return RelayAttemptChannelUnavailable
	case types.ErrorCodeChannelInvalidKey:
		return RelayAttemptAuthError
	case types.ErrorCodeChannelParamOverrideInvalid, types.ErrorCodeChannelHeaderOverrideInvalid,
		types.ErrorCodeChannelModelMappedError, types.ErrorCodeChannelAwsClientError:
		return RelayAttemptChannelUnavailable
	case types.ErrorCodeReadRequestBodyFailed, types.ErrorCodeConvertRequestFailed,
		types.ErrorCodeInvalidRequest, types.ErrorCodeBadRequestBody:
		return RelayAttemptLocalError
	}
	if types.IsChannelError(relayErr) {
		return RelayAttemptChannelUnavailable
	}
	switch relayErr.StatusCode {
	case 401, 403:
		return RelayAttemptAuthError
	case 429:
		return RelayAttemptRateLimited
	}
	if relayErr.StatusCode >= 500 && relayErr.StatusCode <= 599 {
		return RelayAttemptUpstream5xx
	}
	if relayErr.StatusCode == 408 || relayErr.StatusCode == 425 {
		return RelayAttemptTransportError
	}
	if relayErr.StatusCode >= 400 && relayErr.StatusCode <= 499 {
		return RelayAttemptClientError
	}
	return RelayAttemptUpstreamError
}

func AppendSuccessfulRelayTrace(c *gin.Context, adminInfo map[string]interface{}, finalChannelID int) {
	appendRelayTrace(c, adminInfo, "success", finalChannelID)
}

func AppendRelayFailureTrace(c *gin.Context, adminInfo map[string]interface{}, retried bool) {
	result := "failed"
	if retried {
		result = "retrying"
	}
	appendRelayTrace(c, adminInfo, result, 0)
}

func appendRelayTrace(c *gin.Context, adminInfo map[string]interface{}, result string, finalChannelID int) {
	if !operation_setting.SelfUseModeEnabled || adminInfo == nil {
		return
	}
	state := getRelayAttemptState(c, false)
	if state == nil {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.attempts) == 0 {
		return
	}
	attempts := append([]RelayAttempt(nil), state.attempts...)
	last := &attempts[len(attempts)-1]
	if last.Outcome == "" && result == "success" {
		last.DurationMs = max(time.Since(last.startedAt).Milliseconds(), 0)
		last.Outcome = RelayAttemptSuccess
	}
	for i := range attempts {
		attempts[i].startedAt = time.Time{}
	}
	trace := RelayAttemptTrace{
		Version:        1,
		RequestID:      c.GetString(common.RequestIdKey),
		Model:          attempts[len(attempts)-1].Model,
		Result:         result,
		RetryCount:     max(len(attempts)-1, 0),
		FinalChannelID: finalChannelID,
		Attempts:       attempts,
	}
	adminInfo["relay_attempts"] = trace
}

func AttemptedChannelIDs(c *gin.Context) map[int]struct{} {
	result := map[int]struct{}{}
	state := getRelayAttemptState(c, false)
	if state == nil {
		return result
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	for _, attempt := range state.attempts {
		result[attempt.ChannelID] = struct{}{}
	}
	return result
}

func getRelayAttemptState(c *gin.Context, create bool) *relayAttemptState {
	if c == nil {
		return nil
	}
	if value, exists := c.Get(relayAttemptContextKey); exists {
		if state, ok := value.(*relayAttemptState); ok {
			return state
		}
	}
	if !create {
		return nil
	}
	state := &relayAttemptState{}
	c.Set(relayAttemptContextKey, state)
	return state
}
