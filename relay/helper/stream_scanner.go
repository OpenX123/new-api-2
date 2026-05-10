package helper

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"

	"github.com/gin-gonic/gin"
)

const (
	InitialScannerBufferSize    = 64 << 10 // 64KB (64*1024)
	DefaultMaxScannerBufferSize = 64 << 20 // 64MB (64*1024*1024) default SSE buffer size
	DefaultPingInterval         = 10 * time.Second
)

func getScannerBufferSize() int {
	if constant.StreamScannerMaxBufferMB > 0 {
		return constant.StreamScannerMaxBufferMB << 20
	}
	return DefaultMaxScannerBufferSize
}

func StreamScannerHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, dataHandler func(data string, sr *StreamResult)) {

	if resp == nil || dataHandler == nil {
		return
	}

	// 无条件新建 StreamStatus
	info.StreamStatus = relaycommon.NewStreamStatus()

	// 确保响应体总是被关闭
	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()

	streamingTimeout := time.Duration(constant.StreamingTimeout) * time.Second

	var (
		stopChan     = make(chan bool, 3) // 增加缓冲区避免阻塞
		scanner      = bufio.NewScanner(resp.Body)
		ticker       = time.NewTicker(streamingTimeout)
		pingTicker   *time.Ticker
		writeMutex   sync.Mutex     // Mutex to protect concurrent writes
		wg           sync.WaitGroup // 用于等待所有 goroutine 退出
		handlerAlive atomic.Bool    // gate: false ⇒ writer is sealed, no further writes to c.Writer allowed
	)
	handlerAlive.Store(true)

	generalSettings := operation_setting.GetGeneralSetting()
	pingEnabled := generalSettings.PingIntervalEnabled && !info.DisablePing
	pingInterval := time.Duration(generalSettings.PingIntervalSeconds) * time.Second
	if pingInterval <= 0 {
		pingInterval = DefaultPingInterval
	}

	if pingEnabled {
		pingTicker = time.NewTicker(pingInterval)
	}

	if common.DebugEnabled {
		// print timeout and ping interval for debugging
		println("relay timeout seconds:", common.RelayTimeout)
		println("relay max idle conns:", common.RelayMaxIdleConns)
		println("relay max idle conns per host:", common.RelayMaxIdleConnsPerHost)
		println("streaming timeout seconds:", int64(streamingTimeout.Seconds()))
		println("ping interval seconds:", int64(pingInterval.Seconds()))
	}

	scanner.Buffer(make([]byte, InitialScannerBufferSize), getScannerBufferSize())
	scanner.Split(bufio.ScanLines)
	SetEventStreamHeaders(c)

	// ctx 父链路接 c.Request.Context()，客户端断开会自动级联 cancel 所有 goroutine。
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ctx = context.WithValue(ctx, "stop_chan", stopChan)

	// goroutine 内的 logger 调用必须用这个 snapshot 而不是 c。
	// 原因：c 是 gin 池化对象，handler 返回后会被复用给下一个请求；
	// 残留 goroutine 用 c 打日志会把 request id 归因到陌生用户。
	logCtx := context.WithValue(context.Background(), common.RequestIdKey, c.Value(common.RequestIdKey))

	// 资源清理 + 写入封禁：handler 返回前必须先把所有 goroutine 通知到，
	// 等优雅退出窗口（或超时），再翻转 alive gate。残留 goroutine 拿到
	// writeMutex 后会重新 check alive，看到 false 就拒写——避免它们写到
	// 被 gin 池化复用的 c.Writer，造成跨用户 SSE 串流。
	//
	// 注意：alive 翻转**不**在 writeMutex 内做。原因：dataHandler 回调可能
	// 因极慢客户端卡在 c.Writer 写上，长时间持锁；若 cleanup 也试图拿锁
	// 翻转 gate，会死锁主 handler。改用 atomic.Store 即可——goroutine 拿到
	// 锁后会 double-check alive，看到 false 立即 unlock+return，不会写。
	defer func() {
		// 1) 通知所有 goroutine 停止
		cancel()
		common.SafeSendBool(stopChan, true)

		ticker.Stop()
		if pingTicker != nil {
			pingTicker.Stop()
		}

		// 2) 给 goroutine 一个优雅退出窗口
		done := make(chan struct{})
		gopool.Go(func() {
			wg.Wait()
			close(done)
		})

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			logger.LogError(c, "timeout waiting for goroutines to exit; sealing writer")
		}

		// 3) 关键：翻转 alive gate（无锁）。
		// goroutine 拿到 writeMutex 后会 double-check alive，看到 false 即拒写。
		// 翻转和 main return 之间还有 outer defer 在跑（close resp.Body 等），
		// c.Writer 此刻仍然指向当前请求；翻转 happens-before goroutine 的 Load 由
		// atomic 语义保证。
		handlerAlive.Store(false)

		close(stopChan)
	}()

	// Handle ping data sending with improved error handling.
	//
	// 历史教训：原实现里每次 ping 触发都再起一个嵌套 goroutine 调用
	// PingData(c)，外层用 10s 超时等内层。内层 goroutine 没注册到 wg，一旦
	// writeMutex 被持锁超过 10s，外层 goroutine 退出，内层就脱缰了——它会
	// 等到 mutex 被释放后调用 c.Writer.Write，写到一个被 gin 池化复用的
	// 上下文（即下一个用户的响应流）。这就是"我的对话里冒出别人的字"的根因。
	//
	// 新实现：单层 goroutine，同步加锁写。所有路径都受 wg.Wait() 跟踪，
	// 加上 defer 的 alive gate 双保险。
	if pingEnabled && pingTicker != nil {
		wg.Add(1)
		gopool.Go(func() {
			defer func() {
				wg.Done()
				if r := recover(); r != nil {
					logger.LogError(logCtx, fmt.Sprintf("ping goroutine panic: %v", r))
					info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPanic, fmt.Errorf("ping panic: %v", r))
					common.SafeSendBool(stopChan, true)
				}
				if common.DebugEnabled {
					println("ping goroutine exited")
				}
			}()

			// 添加超时保护，防止 goroutine 无限运行
			maxPingDuration := 30 * time.Minute // 最大 ping 持续时间
			pingTimeout := time.NewTimer(maxPingDuration)
			defer pingTimeout.Stop()

			for {
				select {
				case <-pingTicker.C:
					// 早期闸门：handler 已宣布退出，立刻 return，不再排队等锁。
					if !handlerAlive.Load() {
						return
					}
					writeMutex.Lock()
					// 拿到锁后必须再 check 一次 alive，防止从 Lock() 等待到
					// 现在的中间窗口里 handler defer 已经翻转 gate。
					if !handlerAlive.Load() {
						writeMutex.Unlock()
						return
					}
					err := PingData(c)
					writeMutex.Unlock()
					if err != nil {
						logger.LogError(logCtx, "ping data error: "+err.Error())
						info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPingFail, err)
						return
					}
					if common.DebugEnabled {
						println("ping data sent")
					}
				case <-ctx.Done():
					return
				case <-stopChan:
					return
				case <-pingTimeout.C:
					logger.LogError(logCtx, "ping goroutine max duration reached")
					return
				}
			}
		})
	}

	dataChan := make(chan string, 10)

	wg.Add(1)
	gopool.Go(func() {
		defer func() {
			wg.Done()
			if r := recover(); r != nil {
				logger.LogError(logCtx, fmt.Sprintf("data handler goroutine panic: %v", r))
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPanic, fmt.Errorf("handler panic: %v", r))
			}
			common.SafeSendBool(stopChan, true)
		}()
		sr := newStreamResult(info.StreamStatus)
		for data := range dataChan {
			sr.reset()
			// 早期闸门：handler 已退出，drain 剩余 data 但不再写 c.Writer。
			if !handlerAlive.Load() {
				return
			}
			writeMutex.Lock()
			if !handlerAlive.Load() {
				writeMutex.Unlock()
				return
			}
			dataHandler(data, sr)
			writeMutex.Unlock()
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
			wg.Done()
			if r := recover(); r != nil {
				logger.LogError(logCtx, fmt.Sprintf("scanner goroutine panic: %v", r))
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPanic, fmt.Errorf("scanner panic: %v", r))
			}
			common.SafeSendBool(stopChan, true)
			if common.DebugEnabled {
				println("scanner goroutine exited")
			}
		}()

		for scanner.Scan() {
			// 检查是否需要停止
			select {
			case <-stopChan:
				return
			case <-ctx.Done():
				// ctx 父链路接 c.Request.Context()，所以 ctx.Done() 同时
				// 覆盖客户端断开和内部 cancel 两种情况。如果是客户端断开，
				// 优先标记 ClientGone 以保留日志归因。
				if reqErr := c.Request.Context().Err(); reqErr != nil {
					info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, reqErr)
				}
				return
			default:
			}

			ticker.Reset(streamingTimeout)
			data := scanner.Text()
			if common.DebugEnabled {
				println(data)
			}

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
				case dataChan <- data:
				case <-ctx.Done():
					return
				case <-stopChan:
					return
				}
			} else {
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonDone, nil)
				if common.DebugEnabled {
					println("received [DONE], stopping scanner")
				}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			if err != io.EOF {
				logger.LogError(logCtx, "scanner error: "+err.Error())
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonScannerErr, err)
			}
		}
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonEOF, nil)
	})

	// 主循环等待完成或超时
	select {
	case <-ticker.C:
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonTimeout, nil)
	case <-stopChan:
		// EndReason already set by the goroutine that triggered stopChan
	case <-c.Request.Context().Done():
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, c.Request.Context().Err())
	}

	if info.StreamStatus.IsNormalEnd() && !info.StreamStatus.HasErrors() {
		logger.LogInfo(c, fmt.Sprintf("stream ended: %s", info.StreamStatus.Summary()))
	} else {
		logger.LogError(c, fmt.Sprintf("stream ended: %s, received=%d", info.StreamStatus.Summary(), info.ReceivedResponseCount))
	}
}
