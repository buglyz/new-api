package controller

import (
	"errors"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newRelayRetryTestContext() *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	return c
}

func TestShouldRetryAllowsChannelConfigurationErrors(t *testing.T) {
	c := newRelayRetryTestContext()
	for _, errorCode := range []types.ErrorCode{
		types.ErrorCodeChannelModelMappedError,
		types.ErrorCodeChannelParamOverrideInvalid,
	} {
		err := types.NewError(errors.New("channel configuration failed"), errorCode, types.ErrOptionWithSkipRetry())
		assert.True(t, shouldRetry(c, err, 1), errorCode)
	}
}

func TestShouldRetryStillRejectsLocalSkipRetryErrors(t *testing.T) {
	c := newRelayRetryTestContext()
	err := types.NewError(errors.New("request conversion failed"), types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())

	assert.False(t, shouldRetry(c, err, 1))
}

func TestShouldRetryRelayAttemptStopsAfterUpstreamStreamData(t *testing.T) {
	c := newRelayRetryTestContext()
	info := &relaycommon.RelayInfo{ReceivedResponseCount: 1}
	err := types.NewOpenAIError(errors.New("stream failed"), types.ErrorCodeBadResponse, 502)

	assert.False(t, shouldRetryRelayAttempt(c, info, err, 1))
}
