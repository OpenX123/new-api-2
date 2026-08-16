package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type trackedVisionBody struct {
	io.Reader
	closed bool
}

func (b *trackedVisionBody) Close() error {
	b.closed = true
	return nil
}

func TestShouldRetryHonorsSkipRetryOnChannelErrors(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	err := types.NewOpenAIError(errors.New("local deadline"), types.ErrorCodeChannelResponseTimeExceeded, http.StatusGatewayTimeout, types.ErrOptionWithSkipRetry())

	assert.False(t, shouldRetry(c, err, 3))
}

func TestShouldRetryVisionTTFTWithinEndToEndBudget(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	started := time.Now().Add(-46 * time.Second)
	c.Set(string(constant.ContextKeyVisionAugmented), true)
	c.Set(string(constant.ContextKeyRequestStartTime), started)
	c.Set(string(constant.ContextKeyUpstreamStartTime), started)
	c.Set(string(constant.ContextKeyVisionTTFTTimeoutMs), 45000)

	assert.True(t, shouldRetry(c, helper.VisionTTFTError(helper.ErrVisionTTFTTimeout), 1))
	c.Set(string(constant.ContextKeyRequestStartTime), time.Now().Add(-61*time.Second))
	assert.False(t, shouldRetry(c, helper.VisionTTFTError(helper.ErrVisionTTFTTimeout), 1))
}

func TestAugmentClaudeRequestAllowsMoreThanEightCachedHistoryImages(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	config := &dto.VisionBridgeConfig{ChannelId: 2, TTFTTimeoutMs: 1000}
	info := &relaycommon.RelayInfo{UserId: 1}
	content := make([]dto.ClaudeMediaMessage, 12)
	for i := range content {
		content[i] = dto.ClaudeMediaMessage{
			Type: "image",
			Source: &dto.ClaudeMessageSource{
				Type:      "url",
				MediaType: "image/png",
				Url:       fmt.Sprintf("https://example.com/%d.png", i),
			},
		}
	}
	request := &dto.ClaudeRequest{Messages: []dto.ClaudeMessage{{Role: "user", Content: content}}}
	images, err := CollectClaudeVisionImages(request)
	require.NoError(t, err)
	cache := getVisualFactsCache()
	for _, image := range images {
		require.NoError(t, cache.SetWithTTLContext(c.Request.Context(), visualFactsCacheKey(info.UserId, config.ChannelId, "vision", image.Hash), `{"summary":"cached"}`, time.Minute))
	}

	apiErr := augmentClaudeRequestWithVision(c, info, request, config, "vision", len(images))
	require.Nil(t, apiErr)
	require.NotNil(t, info.VisionBilling)
	assert.True(t, info.VisionBilling.CacheHit)
	assert.Equal(t, len(images), info.VisionBilling.ImageCount)
	replaced, err := CollectClaudeVisionImages(request)
	require.NoError(t, err)
	assert.Empty(t, replaced)
}

func TestRunVisionChannelCandidatesFallsBackWithoutLosingFailedUsage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	attempted := make([]int, 0, 2)
	failedCosts := make([]int, 0, 1)

	facts, component, apiErr := runVisionChannelCandidates(
		ctx,
		[]int{2, 3},
		700*time.Millisecond,
		func(_ context.Context, channelId int) (map[string]string, *relaycommon.VisionBillingComponent, *types.NewAPIError) {
			attempted = append(attempted, channelId)
			if channelId == 2 {
				return nil, &relaycommon.VisionBillingComponent{
					ChannelId: channelId,
					Usage:     &dto.Usage{PromptTokens: 10, CompletionTokens: 2},
				}, visionUpstreamError()
			}
			return map[string]string{"hash": `{"summary":"backup"}`}, &relaycommon.VisionBillingComponent{ChannelId: channelId}, nil
		},
		func(component *relaycommon.VisionBillingComponent, _ *types.NewAPIError) {
			failedCosts = append(failedCosts, component.ChannelId)
		},
	)

	require.Nil(t, apiErr)
	require.NotNil(t, component)
	assert.Equal(t, 3, component.ChannelId)
	assert.Equal(t, []int{2, 3}, attempted)
	assert.Equal(t, []int{2}, failedCosts)
	assert.Equal(t, `{"summary":"backup"}`, facts["hash"])
}

func TestVisionAttemptTimeoutPreservesPrimaryBudget(t *testing.T) {
	assert.Equal(t, 30*time.Second, visionAttemptTimeout(45*time.Second, 30*time.Second, 2))
	assert.Equal(t, 22500*time.Millisecond, visionAttemptTimeout(45*time.Second, time.Minute, 2))
	assert.Equal(t, 45*time.Second, visionAttemptTimeout(45*time.Second, 30*time.Second, 1))
}

func TestParseVisualFactsOrdersAndValidatesEveryImage(t *testing.T) {
	facts, err := parseVisualFacts(`prefix {"schema":"visual_facts.v1","images":[{"index":2,"summary":"second"},{"index":1,"summary":"first","visible_text":"OCR"}]} suffix`, 2)
	require.NoError(t, err)
	require.Len(t, facts, 2)
	assert.Equal(t, "first", facts[0].Summary)
	assert.Equal(t, "OCR", facts[0].VisibleText)
	assert.Equal(t, "second", facts[1].Summary)

	_, err = parseVisualFacts(`{"schema":"visual_facts.v1","images":[{"index":1,"summary":"one"},{"index":1,"summary":"duplicate"}]}`, 2)
	require.Error(t, err)
	_, err = parseVisualFacts(`{"schema":"visual_facts.v0","images":[{"index":1,"summary":"one"}]}`, 1)
	require.Error(t, err)

	facts, err = parseVisualFacts(`{"schema":"visual_facts.v1","images":[{"index":1,"summary":"settings","visible_text":"文件夹区域加锁; "文件夹区域" 是由 "我的文件夹" 组成"}]}`, 1)
	require.NoError(t, err)
	assert.Equal(t, `文件夹区域加锁; "文件夹区域" 是由 "我的文件夹" 组成`, facts[0].VisibleText)
}

func TestCanonicalVisualFactIsStable(t *testing.T) {
	first, err := canonicalVisualFact(`{"visible_text":"abc","summary":"screen","index":9,"objects":["window"]}`)
	require.NoError(t, err)
	second, err := canonicalVisualFact(first)
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.NotContains(t, first, "index")
}

func TestDoVisionRequestWithRetry(t *testing.T) {
	tests := []struct {
		name           string
		statuses       []int
		firstError     error
		cancelContext  bool
		expiredContext bool
		wantAttempts   int
		wantStatus     int
		wantError      bool
	}{
		{name: "retries transport error", firstError: errors.New("transport failed"), statuses: []int{http.StatusOK}, wantAttempts: 2, wantStatus: http.StatusOK},
		{name: "does not retry canceled error", firstError: context.Canceled, wantAttempts: 1, wantError: true},
		{name: "does not retry deadline error", firstError: context.DeadlineExceeded, wantAttempts: 1, wantError: true},
		{name: "retries 429", statuses: []int{http.StatusTooManyRequests, http.StatusOK}, wantAttempts: 2, wantStatus: http.StatusOK},
		{name: "retries 500", statuses: []int{http.StatusInternalServerError, http.StatusOK}, wantAttempts: 2, wantStatus: http.StatusOK},
		{name: "retries 529", statuses: []int{529, http.StatusOK}, wantAttempts: 2, wantStatus: http.StatusOK},
		{name: "does not retry 400", statuses: []int{http.StatusBadRequest}, wantAttempts: 1, wantStatus: http.StatusBadRequest},
		{name: "does not retry 401", statuses: []int{http.StatusUnauthorized}, wantAttempts: 1, wantStatus: http.StatusUnauthorized},
		{name: "does not retry 403", statuses: []int{http.StatusForbidden}, wantAttempts: 1, wantStatus: http.StatusForbidden},
		{name: "does not start canceled request", cancelContext: true, wantAttempts: 0, wantError: true},
		{name: "does not start expired request", expiredContext: true, wantAttempts: 0, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if tt.expiredContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-time.Second))
				defer cancel()
			}
			var attempts int
			var requestBodies []*trackedVisionBody
			var responseBodies []*trackedVisionBody
			response, err := doVisionRequestWithRetry(
				ctx,
				func() (io.Reader, io.Closer, error) {
					body := &trackedVisionBody{Reader: strings.NewReader("request")}
					requestBodies = append(requestBodies, body)
					return body, body, nil
				},
				func(io.Reader) (any, error) {
					attempts++
					if attempts == 1 && tt.firstError != nil {
						return nil, tt.firstError
					}
					statusIndex := attempts - 1
					if tt.firstError != nil {
						statusIndex--
					}
					body := &trackedVisionBody{Reader: strings.NewReader("response")}
					responseBodies = append(responseBodies, body)
					return &http.Response{StatusCode: tt.statuses[statusIndex], Body: body}, nil
				},
			)

			assert.Equal(t, tt.wantAttempts, attempts)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, response)
				assert.Equal(t, tt.wantStatus, response.StatusCode)
			}
			for _, body := range requestBodies {
				assert.True(t, body.closed)
			}
			if len(responseBodies) > 1 {
				assert.True(t, responseBodies[0].closed)
			}
		})
	}
}
