package service

import (
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
)

func TestShouldDisableChannelExcludesLocalDeadline(t *testing.T) {
	previous := common.AutomaticDisableChannelEnabled
	common.AutomaticDisableChannelEnabled = true
	t.Cleanup(func() { common.AutomaticDisableChannelEnabled = previous })

	localDeadline := types.NewOpenAIError(
		errors.New("local vision TTFT deadline exceeded"),
		types.ErrorCodeChannelResponseTimeExceeded,
		http.StatusGatewayTimeout,
		types.ErrOptionWithSkipRetry(),
		types.ErrOptionWithLocalDeadline(),
	)
	assert.False(t, ShouldDisableChannel(localDeadline))

	upstreamChannelError := types.NewOpenAIError(
		errors.New("upstream response time exceeded"),
		types.ErrorCodeChannelResponseTimeExceeded,
		http.StatusGatewayTimeout,
	)
	assert.True(t, ShouldDisableChannel(upstreamChannelError))
}
