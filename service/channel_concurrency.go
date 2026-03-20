package service

import (
	"context"
	_ "embed"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/QuantumNous/new-api/common"
)

//go:embed channel_concurrency_acquire.lua
var channelConcurrencyAcquireScript string

//go:embed channel_concurrency_release.lua
var channelConcurrencyReleaseScript string

var (
	concurrencyAcquireSHA string
	concurrencyReleaseSHA string
	concurrencyScriptOnce sync.Once
)

func initConcurrencyScripts() {
	if !common.RedisEnabled {
		return
	}
	ctx := context.Background()
	var err error
	concurrencyAcquireSHA, err = common.RDB.ScriptLoad(ctx, channelConcurrencyAcquireScript).Result()
	if err != nil {
		common.SysLog(fmt.Sprintf("Failed to load channel concurrency acquire script: %v", err))
	}
	concurrencyReleaseSHA, err = common.RDB.ScriptLoad(ctx, channelConcurrencyReleaseScript).Result()
	if err != nil {
		common.SysLog(fmt.Sprintf("Failed to load channel concurrency release script: %v", err))
	}
}

func channelConcurrencyKey(channelId int) string {
	return fmt.Sprintf("channel_concurrency:%d", channelId)
}

// --- Memory implementation ---

var memoryConcurrency sync.Map // map[int]*atomic.Int64

func getMemoryCounter(channelId int) *atomic.Int64 {
	val, ok := memoryConcurrency.Load(channelId)
	if ok {
		return val.(*atomic.Int64)
	}
	counter := &atomic.Int64{}
	actual, _ := memoryConcurrency.LoadOrStore(channelId, counter)
	return actual.(*atomic.Int64)
}

// TryAcquireChannelConcurrency tries to acquire a concurrency slot for the channel.
// Returns true if acquired, false if the channel is at max concurrency.
func TryAcquireChannelConcurrency(channelId int, maxConcurrency int) bool {
	if maxConcurrency <= 0 {
		return true
	}

	if common.RedisEnabled {
		concurrencyScriptOnce.Do(initConcurrencyScripts)
		ctx := context.Background()
		result, err := common.RDB.EvalSha(ctx, concurrencyAcquireSHA, []string{channelConcurrencyKey(channelId)}, maxConcurrency).Int()
		if err != nil {
			common.SysLog(fmt.Sprintf("channel concurrency acquire redis error: %v, falling back to allow", err))
			return true // fail-open
		}
		return result == 1
	}

	// Memory implementation
	counter := getMemoryCounter(channelId)
	for {
		current := counter.Load()
		if current >= int64(maxConcurrency) {
			return false
		}
		if counter.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

// ReleaseChannelConcurrency releases a concurrency slot for the channel.
func ReleaseChannelConcurrency(channelId int) {
	if common.RedisEnabled {
		concurrencyScriptOnce.Do(initConcurrencyScripts)
		ctx := context.Background()
		_, err := common.RDB.EvalSha(ctx, concurrencyReleaseSHA, []string{channelConcurrencyKey(channelId)}).Result()
		if err != nil {
			common.SysLog(fmt.Sprintf("channel concurrency release redis error: %v", err))
		}
		return
	}

	// Memory implementation
	counter := getMemoryCounter(channelId)
	for {
		current := counter.Load()
		if current <= 0 {
			return
		}
		if counter.CompareAndSwap(current, current-1) {
			return
		}
	}
}

// IsChannelConcurrencyAvailable checks if a channel has available concurrency slots
// without acquiring one. Used during channel selection to filter out full channels.
func IsChannelConcurrencyAvailable(channelId int, maxConcurrency int) bool {
	if maxConcurrency <= 0 {
		return true
	}

	if common.RedisEnabled {
		concurrencyScriptOnce.Do(initConcurrencyScripts)
		ctx := context.Background()
		result, err := common.RDB.Get(ctx, channelConcurrencyKey(channelId)).Int()
		if err != nil {
			// Key doesn't exist or error — treat as available
			return true
		}
		return result < maxConcurrency
	}

	// Memory implementation
	counter := getMemoryCounter(channelId)
	return counter.Load() < int64(maxConcurrency)
}
