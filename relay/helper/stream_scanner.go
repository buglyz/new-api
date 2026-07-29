package helper

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"

	"github.com/gin-gonic/gin"
)

const (
	InitialScannerBufferSize    = 64 << 10  // 64KB (64*1024)
	DefaultMaxScannerBufferSize = 128 << 20 // 64MB (64*1024*1024) default SSE buffer size
	DefaultPingInterval         = 10 * time.Second
	defaultFirstEventTimeout    = 35 * time.Second
	defaultStreamingTimeout     = 90 * time.Second
	// streamWriteTimeout bounds a single blocked write to a slow client so the
	// unconditional wg.Wait() in cleanup can always finish. Without it, a slow
	// but connected client (full TCP buffer, no server WriteTimeout) could hang
	// the handler forever.
	streamWriteTimeout = 30 * time.Second
)

func getScannerBufferSize() int {
	if constant.StreamScannerMaxBufferMB > 0 {
		return constant.StreamScannerMaxBufferMB << 20
	}
	return DefaultMaxScannerBufferSize
}

func NewStreamScanner(reader io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, InitialScannerBufferSize), getScannerBufferSize())
	return scanner
}

func copyCodexSSEHeaders(c *gin.Context, resp *http.Response) {
	if c == nil || c.Writer == nil || resp == nil {
		return
	}
	// codex
	for _, name := range []string{"X-Reasoning-Included", "X-Codex-Turn-State"} {
		values := resp.Header.Values(name)
		if !service.ShouldCopyUpstreamHeader(c, name, values) {
			continue
		}
		for _, value := range values {
			if value != "" {
				c.Writer.Header().Add(name, value)
			}
		}
	}
}

// ExtendWriteDeadline pushes the connection write deadline forward before each
// stream write. Best-effort: writers that don't support deadlines (e.g.
// httptest recorders) are silently ignored.
func ExtendWriteDeadline(c *gin.Context) {
	if c == nil || c.Writer == nil {
		return
	}
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(streamWriteTimeout))
}

func StreamScannerHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, dataHandler func(data string, sr *StreamResult)) *types.NewAPIError {

	if resp == nil || dataHandler == nil {
		return nil
	}

	// 无条件新建 StreamStatus
	info.StreamStatus = relaycommon.NewStreamStatus()

	ctx, cancel := context.WithCancel(context.Background())

	firstEventTimeout := time.Duration(constant.StreamFirstEventTimeout) * time.Second
	streamingTimeout := time.Duration(constant.StreamingTimeout) * time.Second
	if firstEventTimeout <= 0 {
		firstEventTimeout = defaultFirstEventTimeout
	}
	if streamingTimeout <= 0 {
		streamingTimeout = defaultStreamingTimeout
	}

	var (
		stopChan     = make(chan bool, 3) // 增加缓冲区避免阻塞
		activityChan = make(chan struct{}, 1)
		scanner      = NewStreamScanner(resp.Body)
		streamTimer  = time.NewTimer(firstEventTimeout)
		pingTicker   *time.Ticker
		writeMutex   sync.Mutex     // Mutex to protect concurrent writes
		wg           sync.WaitGroup // 用于等待所有 goroutine 退出
		cleanupOnce  sync.Once
		stopOnce     sync.Once
		scannerEnd   = relaycommon.StreamEndReasonEOF
	)

	stop := func() {
		stopOnce.Do(func() {
			close(stopChan)
		})
	}

	generalSettings := operation_setting.GetGeneralSetting()
	pingEnabled := generalSettings.PingIntervalEnabled && !info.DisablePing
	pingInterval := time.Duration(generalSettings.PingIntervalSeconds) * time.Second
	if pingInterval <= 0 {
		pingInterval = DefaultPingInterval
	}

	if pingEnabled {
		pingTicker = time.NewTicker(pingInterval)
	}

	logger.LogDebug(c, "relay timeout seconds: %d", common.RelayTimeout)
	logger.LogDebug(c, "relay max idle conns: %d", common.RelayMaxIdleConns)
	logger.LogDebug(c, "relay max idle conns per host: %d", common.RelayMaxIdleConnsPerHost)
	logger.LogDebug(c, "stream first event timeout seconds: %d", int64(firstEventTimeout.Seconds()))
	logger.LogDebug(c, "streaming timeout seconds: %d", int64(streamingTimeout.Seconds()))
	logger.LogDebug(c, "ping interval seconds: %d", int64(pingInterval.Seconds()))

	cleanup := func() {
		cleanupOnce.Do(func() {
			cancel()
			stop()
			if resp.Body != nil {
				_ = resp.Body.Close()
			}

			streamTimer.Stop()
			if pingTicker != nil {
				pingTicker.Stop()
			}

			wg.Wait()
		})
	}
	// Ensure gin.Context is not returned to Gin's pool while any stream goroutine can still use it.
	defer cleanup()

	scanner.Split(bufio.ScanLines)
	copyCodexSSEHeaders(c, resp)
	SetEventStreamHeaders(c)

	ctx = context.WithValue(ctx, "stop_chan", stopChan)

	// Handle ping data sending with improved error handling
	if pingEnabled && pingTicker != nil {
		wg.Add(1)
		gopool.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					logger.LogError(c, fmt.Sprintf("ping goroutine panic: %v", r))
					info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPanic, fmt.Errorf("ping panic: %v", r))
					stop()
				}
				logger.LogDebug(c, "ping goroutine exited")
				wg.Done()
			}()

			// 添加超时保护，防止 goroutine 无限运行
			maxPingDuration := 30 * time.Minute // 最大 ping 持续时间
			pingTimeout := time.NewTimer(maxPingDuration)
			defer pingTimeout.Stop()

			for {
				select {
				case <-pingTicker.C:
					var err error
					func() {
						writeMutex.Lock()
						defer writeMutex.Unlock()
						ExtendWriteDeadline(c)
						err = PingData(c)
					}()
					if err != nil {
						logger.LogError(c, "ping data error: "+err.Error())
						info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPingFail, err)
						stop()
						return
					}
					logger.LogDebug(c, "ping data sent")
				case <-ctx.Done():
					return
				case <-stopChan:
					return
				case <-c.Request.Context().Done():
					// 监听客户端断开连接
					return
				case <-pingTimeout.C:
					logger.LogError(c, "ping goroutine max duration reached")
					return
				}
			}
		})
	}

	dataChan := make(chan string, 10)

	wg.Add(1)
	gopool.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				logger.LogError(c, fmt.Sprintf("data handler goroutine panic: %v", r))
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPanic, fmt.Errorf("handler panic: %v", r))
			}
			stop()
			wg.Done()
		}()
		sr := newStreamResult(info.StreamStatus)
		for data := range dataChan {
			sr.reset()
			func() {
				writeMutex.Lock()
				defer writeMutex.Unlock()
				ExtendWriteDeadline(c)
				dataHandler(data, sr)
			}()
			if sr.IsStopped() {
				return
			}
		}
	})

	// Scanner goroutine with improved error handling
	wg.Add(1)
	common.RelayCtxGo(ctx, func() {
		defer func() {
			close(dataChan)
			if r := recover(); r != nil {
				logger.LogError(c, fmt.Sprintf("scanner goroutine panic: %v", r))
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPanic, fmt.Errorf("scanner panic: %v", r))
			}
			stop()
			logger.LogDebug(c, "scanner goroutine exited")
			wg.Done()
		}()

		for scanner.Scan() {
			// 检查是否需要停止
			select {
			case <-stopChan:
				return
			case <-ctx.Done():
				return
			default:
			}

			data := scanner.Text()
			logger.LogDebug(c, "stream scanner data: %s", data)

			if len(data) < 6 {
				continue
			}
			if data[:5] != "data:" && data[:6] != "[DONE]" {
				continue
			}
			data = data[5:]
			data = strings.TrimSpace(data)
			if data == "" {
				continue
			}
			if !strings.HasPrefix(data, "[DONE]") {
				info.SetFirstResponseTime()
				info.ReceivedResponseCount++
				select {
				case activityChan <- struct{}{}:
				default:
				}

				select {
				case dataChan <- data:
				case <-ctx.Done():
					return
				case <-stopChan:
					return
				}
			} else {
				scannerEnd = relaycommon.StreamEndReasonDone
				logger.LogDebug(c, "received [DONE], stopping scanner")
				return
			}
		}

		if err := scanner.Err(); err != nil {
			if err != io.EOF {
				logger.LogError(c, "scanner error: "+err.Error())
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonScannerErr, err)
			}
		}
	})

	// Only this goroutine owns the timer. Meaningful events switch the initial
	// deadline to the idle deadline; comments and heartbeat lines never reach
	// activityChan.
waitForStream:
	for {
		select {
		case <-streamTimer.C:
			info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonTimeout, nil)
			break waitForStream
		case <-activityChan:
			if !streamTimer.Stop() {
				select {
				case <-streamTimer.C:
				default:
				}
			}
			streamTimer.Reset(streamingTimeout)
		case <-stopChan:
			// EndReason already set by the goroutine that triggered stopChan.
			break waitForStream
		case <-c.Request.Context().Done():
			// Closing the upstream body stops generation after the client leaves.
			info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, c.Request.Context().Err())
			break waitForStream
		}
	}

	cleanup()
	// Scanner completion is only committed after queued data handlers finish.
	// A fatal conversion/write outcome must take precedence over DONE or EOF.
	if info.StreamStatus.EndReason == relaycommon.StreamEndReasonNone {
		info.StreamStatus.SetEndReason(scannerEnd, nil)
	}
	if info.StreamStatus.IsNormalEnd() && !info.StreamStatus.HasErrors() {
		logger.LogInfo(c, fmt.Sprintf("stream ended: %s", info.StreamStatus.Summary()))
	} else {
		logger.LogError(c, fmt.Sprintf("stream ended: %s, received=%d", info.StreamStatus.Summary(), info.ReceivedResponseCount))
	}
	return streamRelayError(info)
}

func streamRelayError(info *relaycommon.RelayInfo) *types.NewAPIError {
	if info == nil || info.StreamStatus == nil {
		return nil
	}
	status := info.StreamStatus
	if status.EndReason == relaycommon.StreamEndReasonDone || status.EndReason == relaycommon.StreamEndReasonEOF {
		return nil
	}
	if status.EndReason == relaycommon.StreamEndReasonClientGone ||
		(status.EndReason == relaycommon.StreamEndReasonHandlerStop &&
			(errors.Is(status.EndError, context.Canceled) || errors.Is(status.EndError, context.DeadlineExceeded))) {
		return nil
	}

	streamErr := fmt.Errorf("upstream stream ended abnormally: %s", status.Summary())
	options := make([]types.NewAPIErrorOptions, 0, 2)
	if info.ReceivedResponseCount > 0 {
		options = append(options, types.ErrOptionWithSkipRetry())
	}

	switch status.EndReason {
	case relaycommon.StreamEndReasonClientGone:
		options = append(options, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		return types.NewErrorWithStatusCode(streamErr, types.ErrorCodeReadResponseBodyFailed, 499, options...)
	case relaycommon.StreamEndReasonTimeout:
		return types.NewErrorWithStatusCode(streamErr, types.ErrorCodeChannelResponseTimeExceeded, http.StatusGatewayTimeout, options...)
	default:
		return types.NewErrorWithStatusCode(streamErr, types.ErrorCodeDoRequestFailed, http.StatusBadGateway, options...)
	}
}
