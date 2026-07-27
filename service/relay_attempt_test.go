package service

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyRelayAttemptUsesStructuredErrorFields(t *testing.T) {
	tests := []struct {
		name    string
		err     *types.NewAPIError
		outcome RelayAttemptOutcome
	}{
		{name: "success", outcome: RelayAttemptSuccess},
		{name: "transport", err: types.NewError(errors.New("dial failed"), types.ErrorCodeDoRequestFailed), outcome: RelayAttemptTransportError},
		{name: "rate limited", err: types.NewOpenAIError(errors.New("limited"), types.ErrorCodeBadResponseStatusCode, 429), outcome: RelayAttemptRateLimited},
		{name: "server error", err: types.NewOpenAIError(errors.New("upstream"), types.ErrorCodeBadResponseStatusCode, 503), outcome: RelayAttemptUpstream5xx},
		{name: "auth", err: types.NewOpenAIError(errors.New("denied"), types.ErrorCodeBadResponseStatusCode, 403), outcome: RelayAttemptAuthError},
		{name: "invalid channel key", err: types.NewError(errors.New("invalid key"), types.ErrorCodeChannelInvalidKey), outcome: RelayAttemptAuthError},
		{name: "no channel key", err: types.NewError(errors.New("no key"), types.ErrorCodeChannelNoAvailableKey), outcome: RelayAttemptChannelUnavailable},
		{name: "invalid parameter override", err: types.NewError(errors.New("bad config"), types.ErrorCodeChannelParamOverrideInvalid), outcome: RelayAttemptChannelUnavailable},
		{name: "model", err: types.NewOpenAIError(errors.New("missing"), types.ErrorCodeModelNotFound, 404), outcome: RelayAttemptModelUnavailable},
		{name: "client", err: types.NewOpenAIError(errors.New("bad"), types.ErrorCodeBadResponseStatusCode, 400), outcome: RelayAttemptClientError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.outcome, ClassifyRelayAttempt(test.err))
		})
	}
}

func TestRelayAttemptTraceContainsOnlyStructuredAttemptData(t *testing.T) {
	previous := operation_setting.SelfUseModeEnabled
	operation_setting.SelfUseModeEnabled = true
	t.Cleanup(func() { operation_setting.SelfUseModeEnabled = previous })

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(common.RequestIdKey, "req-123")
	first := BeginRelayAttempt(ctx, 7, "gpt-test")
	CompleteRelayAttempt(ctx, first, types.NewOpenAIError(errors.New("secret raw failure"), types.ErrorCodeBadResponseStatusCode, 503), true)
	second := BeginRelayAttempt(ctx, 9, "gpt-test")
	CompleteRelayAttempt(ctx, second, nil, false)

	adminInfo := map[string]interface{}{}
	AppendSuccessfulRelayTrace(ctx, adminInfo, 9)
	trace, ok := adminInfo["relay_attempts"].(RelayAttemptTrace)
	require.True(t, ok)
	assert.Equal(t, "req-123", trace.RequestID)
	assert.Equal(t, 1, trace.RetryCount)
	assert.Equal(t, 9, trace.FinalChannelID)
	require.Len(t, trace.Attempts, 2)
	assert.Equal(t, RelayAttemptUpstream5xx, trace.Attempts[0].Outcome)
	assert.Equal(t, 503, trace.Attempts[0].StatusCode)
	assert.NotContains(t, common.GetJsonString(trace), "secret raw failure")
}

func TestRelayAttemptTrackingIsDisabledOutsidePersonalMode(t *testing.T) {
	previous := operation_setting.SelfUseModeEnabled
	operation_setting.SelfUseModeEnabled = false
	t.Cleanup(func() { operation_setting.SelfUseModeEnabled = previous })

	ctx, _ := gin.CreateTestContext(nil)
	assert.Equal(t, -1, BeginRelayAttempt(ctx, 1, "model"))
	adminInfo := map[string]interface{}{}
	AppendSuccessfulRelayTrace(ctx, adminInfo, 1)
	assert.Empty(t, adminInfo)
}
