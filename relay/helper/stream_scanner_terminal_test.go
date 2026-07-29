package helper

import (
	"context"
	"fmt"
	"io"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type closeNotifyingReadCloser struct {
	io.ReadCloser
	closed chan struct{}
}

func (r *closeNotifyingReadCloser) Close() error {
	select {
	case <-r.closed:
	default:
		close(r.closed)
	}
	return r.ReadCloser.Close()
}

func TestStreamScannerHandlerFatalStopOverridesQueuedDone(t *testing.T) {
	reader, writer := io.Pipe()
	c, resp, info := setupStreamTest(t, reader)
	body := &closeNotifyingReadCloser{ReadCloser: reader, closed: make(chan struct{})}
	resp.Body = body

	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	result := make(chan *types.NewAPIError, 1)
	go func() {
		result <- StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
			close(handlerStarted)
			<-releaseHandler
			sr.Stop(fmt.Errorf("fatal conversion error"))
		})
	}()

	_, err := io.WriteString(writer, "data: {\"id\":1}\n")
	require.NoError(t, err)
	<-handlerStarted
	_, err = io.WriteString(writer, "data: [DONE]\n")
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	<-body.closed
	close(releaseHandler)

	streamErr := <-result
	require.NotNil(t, streamErr)
	assert.True(t, types.IsSkipRetryError(streamErr))
	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonHandlerStop, info.StreamStatus.EndReason)
	assert.True(t, info.StreamStatus.HasErrors())
}

func TestStreamRelayErrorIgnoresClientCancellationDuringWrite(t *testing.T) {
	info := &relaycommon.RelayInfo{StreamStatus: relaycommon.NewStreamStatus()}
	info.StreamStatus.SetEndReason(
		relaycommon.StreamEndReasonHandlerStop,
		fmt.Errorf("request context done: %w", context.Canceled),
	)

	assert.Nil(t, streamRelayError(info))
}
