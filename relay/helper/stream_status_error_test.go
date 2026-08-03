package helper

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamStatusErrorKeepsNormalEOFSuccessful(t *testing.T) {
	info := &relaycommon.RelayInfo{StreamStatus: relaycommon.NewStreamStatus()}
	info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonEOF, nil)

	assert.Nil(t, StreamStatusError(info))
}

func TestStreamStatusErrorRejectsErrorsAfterNormalEnd(t *testing.T) {
	info := &relaycommon.RelayInfo{StreamStatus: relaycommon.NewStreamStatus()}
	info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonDone, nil)
	info.StreamStatus.RecordError("invalid stream chunk")

	err := StreamStatusError(info)
	require.NotNil(t, err)
	assert.Equal(t, types.ErrorCodeBadResponse, err.GetErrorCode())
	assert.Equal(t, 502, err.StatusCode)
}

func TestStreamStatusErrorMapsTimeoutToTransportFailure(t *testing.T) {
	info := &relaycommon.RelayInfo{StreamStatus: relaycommon.NewStreamStatus()}
	info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonTimeout, nil)

	err := StreamStatusError(info)
	require.NotNil(t, err)
	assert.Equal(t, types.ErrorCodeChannelResponseTimeExceeded, err.GetErrorCode())
	assert.Equal(t, 504, err.StatusCode)
}

func TestStreamStatusErrorDoesNotCircuitOnClientDisconnect(t *testing.T) {
	info := &relaycommon.RelayInfo{StreamStatus: relaycommon.NewStreamStatus()}
	info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, nil)

	err := StreamStatusError(info)
	require.NotNil(t, err)
	assert.Equal(t, 499, err.StatusCode)
	assert.True(t, types.IsSkipRetryError(err))
}
