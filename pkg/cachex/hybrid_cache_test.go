package cachex

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/samber/hot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHybridCacheUsesLocalFallbackWhenRedisContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cache := NewHybridCache[string](HybridCacheConfig[string]{
		Namespace:     Namespace("test"),
		Redis:         redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}),
		RedisEnabled:  func() bool { return true },
		RedisCodec:    StringCodec{},
		LocalFallback: true,
		Memory: func() *hot.HotCache[string, string] {
			return hot.NewHotCache[string, string](hot.LRU, 2).Build()
		},
	})

	err := cache.SetWithTTLContext(ctx, "image", "facts", time.Minute)
	require.ErrorIs(t, err, context.Canceled)

	value, found, err := cache.GetContext(ctx, "image")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "facts", value)
}
