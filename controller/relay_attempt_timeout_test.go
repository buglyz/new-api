package controller

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRelayAttemptAppliesNonStreamContextDeadline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	previous := common.RelayNonStreamTimeout
	common.RelayNonStreamTimeout = 1
	t.Cleanup(func() { common.RelayNonStreamTimeout = previous })

	err := runRelayAttempt(ctx, &relaycommon.RelayInfo{}, func() *types.NewAPIError {
		<-ctx.Request.Context().Done()
		return types.NewError(ctx.Request.Context().Err(), types.ErrorCodeBadResponse)
	})

	require.NotNil(t, err)
	assert.ErrorIs(t, err.Err, context.DeadlineExceeded)
}

func TestRunRelayAttemptDoesNotLimitStreamingContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	previous := common.RelayNonStreamTimeout
	common.RelayNonStreamTimeout = 1
	t.Cleanup(func() { common.RelayNonStreamTimeout = previous })

	called := false
	err := runRelayAttempt(ctx, &relaycommon.RelayInfo{IsStream: true}, func() *types.NewAPIError {
		called = true
		_, hasDeadline := ctx.Request.Context().Deadline()
		assert.False(t, hasDeadline)
		return nil
	})

	assert.True(t, called)
	assert.Nil(t, err)
}

func TestShouldRetryNeverSwitchesExplicitChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("specific_channel_id", 7)
	retry := &service.RetryParam{StartedAt: time.Now()}
	err := types.NewErrorWithStatusCode(
		errors.New("invalid upstream key"),
		types.ErrorCodeChannelInvalidKey,
		401,
	)

	assert.False(t, shouldRetry(ctx, err, 1, retry))
}

func TestShouldRetryOnlyExplicitModelNotFoundOn404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	retry := &service.RetryParam{StartedAt: time.Now()}
	modelMissing := types.NewErrorWithStatusCode(
		errors.New("model unavailable"),
		types.ErrorCodeModelNotFound,
		404,
	)
	genericNotFound := types.NewErrorWithStatusCode(
		errors.New("not found"),
		types.ErrorCodeBadResponseStatusCode,
		404,
	)

	assert.True(t, shouldRetry(ctx, modelMissing, 1, retry))
	assert.False(t, shouldRetry(ctx, genericNotFound, 1, retry))
}
