package helper

import (
	"fmt"
	"net/http"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
)

const clientClosedRequestStatusCode = 499

// StreamStatusError converts an abnormal stream termination into the same
// structured error path used by non-streaming relay responses. A normal EOF
// remains successful for providers that omit [DONE].
func StreamStatusError(info *relaycommon.RelayInfo) *types.NewAPIError {
	if info == nil || info.StreamStatus == nil {
		return nil
	}
	status := info.StreamStatus
	if status.IsNormalEnd() && !status.HasErrors() {
		return nil
	}

	err := status.EndError
	if err == nil {
		err = fmt.Errorf("stream ended abnormally: reason=%s, errors=%d", status.EndReason, status.TotalErrorCount())
	}
	if status.EndReason == relaycommon.StreamEndReasonTimeout {
		return types.NewOpenAIError(err, types.ErrorCodeChannelResponseTimeExceeded, http.StatusGatewayTimeout)
	}
	if status.EndReason == relaycommon.StreamEndReasonClientGone || status.EndReason == relaycommon.StreamEndReasonPingFail {
		return types.NewErrorWithStatusCode(err, types.ErrorCodeReadResponseBodyFailed, clientClosedRequestStatusCode, types.ErrOptionWithSkipRetry())
	}
	return types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusBadGateway)
}
