package channel

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVisionTTFTDeadlineReleaseDoesNotCancelLongStream(t *testing.T) {
	ctx, deadline := newVisionTTFTDeadline(context.Background(), time.Now().Add(time.Hour))
	deadline.release()
	deadline.expire()

	select {
	case <-ctx.Done():
		require.Failf(t, "released TTFT deadline canceled the stream", "cause: %v", context.Cause(ctx))
	default:
	}

	deadline.close()
	assert.ErrorIs(t, context.Cause(ctx), context.Canceled)
}

func TestVisionTTFTDeadlineUsesEndToEndCap(t *testing.T) {
	requestStarted := time.Now()
	c := &gin.Context{}
	common.SetContextKey(c, constant.ContextKeyRequestStartTime, requestStarted)

	deadline := visionTTFTDeadlineAt(c, requestStarted.Add(30*time.Second), 60*time.Second)
	assert.Equal(t, requestStarted.Add(helper.VisionEndToEndTTFTTimeout), deadline)
}

func TestVisionTTFTDeadlineExpiryPreservesTimeoutCause(t *testing.T) {
	ctx, deadline := newVisionTTFTDeadline(context.Background(), time.Now().Add(time.Hour))
	deadline.expire()
	deadline.timer.Stop()
	deadline.close()

	require.ErrorIs(t, context.Cause(ctx), helper.ErrVisionTTFTTimeout)
	apiErr := helper.VisionTTFTError(context.Cause(ctx))
	require.NotNil(t, apiErr)
	assert.Equal(t, 504, apiErr.StatusCode)
}

func TestVisionTTFTDeadlineAlreadyExpiredIsSynchronous(t *testing.T) {
	ctx, deadline := newVisionTTFTDeadline(context.Background(), time.Now().Add(-time.Second))
	defer deadline.close()
	require.ErrorIs(t, context.Cause(ctx), helper.ErrVisionTTFTTimeout)
	assert.False(t, deadline.release())
}
