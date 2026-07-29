package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayHTTPClientTimesOutWaitingForResponseHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	previous := common.RelayResponseHeaderTimeout
	common.RelayResponseHeaderTimeout = 0
	t.Cleanup(func() { common.RelayResponseHeaderTimeout = previous })
	transport := newRelayHTTPTransport()
	transport.ResponseHeaderTimeout = 40 * time.Millisecond
	client := newRelayHTTPClient(transport)

	_, err := client.Get(server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout awaiting response headers")
	assert.Zero(t, client.Timeout, "relay client must not impose a whole-request timeout on streams")
}
