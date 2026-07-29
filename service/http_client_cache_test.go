package service

import (
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type closeCountingRoundTripper struct {
	closes atomic.Int32
}

func (c *closeCountingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func (c *closeCountingRoundTripper) CloseIdleConnections() {
	c.closes.Add(1)
}

func TestShardedRoundTripperCloseIdleConnectionsFansOut(t *testing.T) {
	trackers := []*closeCountingRoundTripper{{}, {}, {}}
	shards := make([]http.RoundTripper, len(trackers))
	for i, tracker := range trackers {
		shards[i] = tracker
	}
	s := &shardedRoundTripper{shards: shards, n: uint32(len(shards))}
	s.CloseIdleConnections()
	for _, tracker := range trackers {
		assert.Equal(t, int32(1), tracker.closes.Load())
	}
}

func TestInvalidateProxyClientClosesAllPolicyVariants(t *testing.T) {
	initDefaultHTTPClientFixture(t)

	proxyURL := "http://invalidate-proxy.example:8080"
	defaultClient, err := GetHttpClientWithProxy(proxyURL)
	require.NoError(t, err)
	http1Client, err := GetHttpClientWithProxySettings(proxyURL, dto.ChannelSettings{HTTPProtocol: dto.HTTPProtocolHTTP1})
	require.NoError(t, err)
	shardedClient, err := GetHttpClientWithProxySettings(proxyURL, dto.ChannelSettings{HTTP2ConnectionShards: 2})
	require.NoError(t, err)

	InvalidateProxyClient(proxyURL)

	afterDefault, err := GetHttpClientWithProxy(proxyURL)
	require.NoError(t, err)
	afterHTTP1, err := GetHttpClientWithProxySettings(proxyURL, dto.ChannelSettings{HTTPProtocol: dto.HTTPProtocolHTTP1})
	require.NoError(t, err)
	afterSharded, err := GetHttpClientWithProxySettings(proxyURL, dto.ChannelSettings{HTTP2ConnectionShards: 2})
	require.NoError(t, err)

	assert.NotSame(t, defaultClient, afterDefault)
	assert.NotSame(t, http1Client, afterHTTP1)
	assert.NotSame(t, shardedClient, afterSharded)
}

func TestResetProxyClientCacheKeepsDefaultPointerAndRecreatesVariants(t *testing.T) {
	defaultClient := initDefaultHTTPClientFixture(t)

	http1Client, err := GetHttpClientWithProxySettings("", dto.ChannelSettings{HTTPProtocol: dto.HTTPProtocolHTTP1})
	require.NoError(t, err)
	shardedClient, err := GetHttpClientWithProxySettings("", dto.ChannelSettings{HTTP2ConnectionShards: 3})
	require.NoError(t, err)
	proxyClient, err := GetHttpClientWithProxy("http://reset-proxy.example:8080")
	require.NoError(t, err)

	ResetProxyClientCache()

	assert.Same(t, defaultClient, GetHttpClient(), "default httpClient pointer must stay stable across reset")
	aware, err := GetHttpClientWithProxySettings("", dto.ChannelSettings{})
	require.NoError(t, err)
	assert.Same(t, defaultClient, aware)

	afterHTTP1, err := GetHttpClientWithProxySettings("", dto.ChannelSettings{HTTPProtocol: dto.HTTPProtocolHTTP1})
	require.NoError(t, err)
	afterSharded, err := GetHttpClientWithProxySettings("", dto.ChannelSettings{HTTP2ConnectionShards: 3})
	require.NoError(t, err)
	afterProxy, err := GetHttpClientWithProxy("http://reset-proxy.example:8080")
	require.NoError(t, err)
	assert.NotSame(t, http1Client, afterHTTP1)
	assert.NotSame(t, shardedClient, afterSharded)
	assert.NotSame(t, proxyClient, afterProxy)
}

func TestResetProxyClientCacheClosesDefaultIdlePool(t *testing.T) {
	defaultClient := initDefaultHTTPClientFixture(t)
	tracker := &closeCountingRoundTripper{}
	previousTransport := defaultClient.Transport
	defaultClient.Transport = tracker
	t.Cleanup(func() {
		defaultClient.Transport = previousTransport
	})

	ResetProxyClientCache()

	assert.Same(t, defaultClient, GetHttpClient())
	assert.GreaterOrEqual(t, tracker.closes.Load(), int32(1), "reset must close idle connections on the stable default client")
	aware, err := GetHttpClientWithProxySettings("", dto.ChannelSettings{})
	require.NoError(t, err)
	assert.Same(t, defaultClient, aware)
}

func TestResetProxyClientCacheConcurrentWithGetHttpClient(t *testing.T) {
	initDefaultHTTPClientFixture(t)

	const workers = 64
	var wg sync.WaitGroup
	wg.Add(workers * 2)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			_ = GetHttpClient()
		}()
		go func() {
			defer wg.Done()
			ResetProxyClientCache()
		}()
	}
	wg.Wait()
	assert.NotNil(t, GetHttpClient())
	aware, err := GetHttpClientWithProxySettings("", dto.ChannelSettings{})
	require.NoError(t, err)
	assert.Same(t, GetHttpClient(), aware)
}

func TestCloseIdleConnectionsRedialsHTTP2(t *testing.T) {
	withRelayHTTPTransportSettings(t)

	var mu sync.Mutex
	addrs := make([]string, 0, 2)

	server := startHTTP2TLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		addrs = append(addrs, r.RemoteAddr)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	client := newHTTPClientWithPolicyAndTLS(defaultHTTPTransportPolicy(), testTLSClientConfig(t, server))
	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	drainClose(t, resp)

	client.CloseIdleConnections()

	resp, err = client.Get(server.URL)
	require.NoError(t, err)
	drainClose(t, resp)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, addrs, 2)
	assert.NotEqual(t, addrs[0], addrs[1], "after CloseIdleConnections the next request must redial")
}

func TestNormalizeHTTPTransportPolicyClampsWithoutPanic(t *testing.T) {
	assert.Equal(t, defaultHTTPTransportPolicy(), NormalizeHTTPTransportPolicy(dto.ChannelSettings{}))
	assert.Equal(t, HTTPTransportPolicy{Protocol: dto.HTTPProtocolAuto, Shards: 1}, NormalizeHTTPTransportPolicy(dto.ChannelSettings{HTTPProtocol: "AUTO"}))
	assert.Equal(t, HTTPTransportPolicy{Protocol: dto.HTTPProtocolHTTP1, Shards: 1}, NormalizeHTTPTransportPolicy(dto.ChannelSettings{HTTPProtocol: "HTTP1", HTTP2ConnectionShards: 8}))
	assert.Equal(t, HTTPTransportPolicy{Protocol: dto.HTTPProtocolAuto, Shards: 1}, NormalizeHTTPTransportPolicy(dto.ChannelSettings{HTTPProtocol: "http3"}))
	assert.Equal(t, HTTPTransportPolicy{Protocol: dto.HTTPProtocolAuto, Shards: 1}, NormalizeHTTPTransportPolicy(dto.ChannelSettings{HTTP2ConnectionShards: -3}))
	assert.Equal(t, HTTPTransportPolicy{Protocol: dto.HTTPProtocolAuto, Shards: 8}, NormalizeHTTPTransportPolicy(dto.ChannelSettings{HTTP2ConnectionShards: 99}))
}
