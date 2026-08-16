package service

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingBillingSettler struct {
	actualQuota int
}

func (s *failingBillingSettler) Settle(actualQuota int) error {
	s.actualQuota = actualQuota
	return errors.New("forced settlement failure")
}

func (*failingBillingSettler) Refund(*gin.Context)      {}
func (*failingBillingSettler) NeedsRefund() bool        { return false }
func (*failingBillingSettler) GetPreConsumedQuota() int { return 100 }
func (*failingBillingSettler) Reserve(int) error        { return nil }

// TestDecimalToQuotaSaturation guards the billing invariant that an oversized
// quota product (e.g. per-call price multiplied by a huge image n ratio) must
// saturate instead of wrapping into a negative charge (credit).
func TestDecimalToQuotaSaturation(t *testing.T) {
	// 2000 quota per call * n=18446744073686646784 overflows int64.
	overflowing := decimal.NewFromInt(2000).Mul(decimal.NewFromFloat(1.8446744073686647e19))
	require.Equal(t, math.MaxInt32, decimalToQuota(overflowing))

	require.Equal(t, math.MinInt32, decimalToQuota(overflowing.Neg()))
	require.Equal(t, 42, decimalToQuota(decimal.NewFromFloat(41.7)))
}

func TestCalculateVisionActualQuotaRatioPricesUpstreamCache(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 10,
		UsageSemantic:    "anthropic",
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         80,
			CachedCreationTokens: 30,
		},
		ClaudeCacheCreation5mTokens: 10,
		ClaudeCacheCreation1hTokens: 20,
	}
	component := &relaycommon.VisionBillingComponent{
		Usage: usage,
		PriceData: types.PriceData{
			ModelRatio:           0.3,
			CompletionRatio:      4,
			CacheRatio:           0.1,
			CacheCreationRatio:   1.25,
			CacheCreation5mRatio: 1.25,
			CacheCreation1hRatio: 2,
			GroupRatioInfo:       types.GroupRatioInfo{GroupRatio: 1},
		},
	}

	quota, result, err := CalculateVisionActualQuota(component)

	require.NoError(t, err)
	require.Nil(t, result)
	require.Equal(t, 60, quota)
	require.Equal(t, 80, usage.PromptTokensDetails.CachedTokens)
	require.Equal(t, 30, usage.PromptTokensDetails.CachedCreationTokens)
}

func TestCalculateVisionActualQuotaSaturatesUntrustedUsage(t *testing.T) {
	component := &relaycommon.VisionBillingComponent{
		Usage: &dto.Usage{
			PromptTokens:  1,
			UsageSemantic: "anthropic",
			PromptTokensDetails: dto.InputTokenDetails{
				CachedCreationTokens: math.MaxInt,
			},
			ClaudeCacheCreation5mTokens: math.MaxInt,
			ClaudeCacheCreation1hTokens: math.MaxInt,
		},
		PriceData: types.PriceData{
			ModelRatio:           1,
			CompletionRatio:      1,
			CacheCreationRatio:   1,
			CacheCreation5mRatio: 0.1,
			CacheCreation1hRatio: 0.1,
			GroupRatioInfo:       types.GroupRatioInfo{GroupRatio: 1},
		},
	}

	quota, result, err := CalculateVisionActualQuota(component)

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.Equal(t, math.MaxInt32, quota)
}

func TestCalculateVisionActualQuotaTieredPricesUpstreamCache(t *testing.T) {
	expr := `tier("vision", p * 2 + c * 4 + cr * 0.2 + cc * 2.5 + cc1h * 4)`
	component := &relaycommon.VisionBillingComponent{
		Usage: &dto.Usage{
			PromptTokens:     1_000_000,
			CompletionTokens: 100_000,
			UsageSemantic:    "anthropic",
			PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens:         500_000,
				CachedCreationTokens: 150_000,
			},
			ClaudeCacheCreation5mTokens: 100_000,
			ClaudeCacheCreation1hTokens: 50_000,
		},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:  "tiered_expr",
			ExprString:   expr,
			ExprHash:     billingexpr.ExprHashString(expr),
			GroupRatio:   1,
			QuotaPerUnit: common.QuotaPerUnit,
			ExprVersion:  billingexpr.ExprVersion(expr),
		},
	}

	quota, result, err := CalculateVisionActualQuota(component)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1_475_000, quota)
}

func TestCalculateVisionActualQuotaSumsBatchesIndependently(t *testing.T) {
	expr := `p <= 100 ? tier("small", p) : tier("large", p * 10)`
	components := make([]*relaycommon.VisionBillingComponent, 2)
	for i := range components {
		components[i] = &relaycommon.VisionBillingComponent{
			Usage: &dto.Usage{PromptTokens: 60},
			TieredBillingSnapshot: &billingexpr.BillingSnapshot{
				BillingMode:  "tiered_expr",
				ExprString:   expr,
				ExprHash:     billingexpr.ExprHashString(expr),
				GroupRatio:   1,
				QuotaPerUnit: common.QuotaPerUnit,
				ExprVersion:  billingexpr.ExprVersion(expr),
			},
		}
	}

	quota, result, err := CalculateVisionActualQuota(&relaycommon.VisionBillingComponent{Components: components})

	require.NoError(t, err)
	require.Nil(t, result)
	require.Equal(t, 60, quota)
}

func TestCalculateVisionActualQuotaCacheHitIsFree(t *testing.T) {
	component := &relaycommon.VisionBillingComponent{
		CacheHit:       true,
		EstimatedQuota: 123,
		Usage:          &dto.Usage{PromptTokens: 1000, CompletionTokens: 100},
		PriceData: types.PriceData{
			ModelRatio:      1,
			CompletionRatio: 2,
			GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
		},
	}

	quota, result, err := CalculateVisionActualQuota(component)

	require.NoError(t, err)
	require.Nil(t, result)
	require.Zero(t, quota)
}

func TestCalculateVisionActualQuotaTieredFailureFallsBackToEstimate(t *testing.T) {
	component := &relaycommon.VisionBillingComponent{
		EstimatedQuota: 321,
		Usage:          &dto.Usage{PromptTokens: 1000},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:  "tiered_expr",
			ExprString:   "invalid(",
			ExprHash:     billingexpr.ExprHashString("invalid("),
			GroupRatio:   1,
			QuotaPerUnit: common.QuotaPerUnit,
		},
	}

	quota, result, err := CalculateVisionActualQuota(component)

	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, 321, quota)
}

func TestCalculateVisionActualQuotaTieredUsesFrozenRequestInput(t *testing.T) {
	expr := `has(header("x-vision-tier"), "priority") ? tier("vision", (p * 2 + c * 4) * 2) : tier("vision", p * 2 + c * 4)`
	component := &relaycommon.VisionBillingComponent{
		Usage: &dto.Usage{
			PromptTokens:     1_000_000,
			CompletionTokens: 100_000,
			PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens: 500_000,
			},
		},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:  "tiered_expr",
			ExprString:   expr,
			ExprHash:     billingexpr.ExprHashString(expr),
			GroupRatio:   1,
			QuotaPerUnit: common.QuotaPerUnit,
			ExprVersion:  billingexpr.ExprVersion(expr),
		},
		BillingRequestInput: &billingexpr.RequestInput{
			Headers: map[string]string{"x-vision-tier": "priority"},
		},
	}

	quota, result, err := CalculateVisionActualQuota(component)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "vision", result.MatchedTier)
	require.Equal(t, 2_400_000, quota)
}

func TestMergedTextVisionQuotaSaturates(t *testing.T) {
	require.Equal(t, math.MaxInt32, mergedTextVisionQuota(math.MaxInt32, 1))
	require.Equal(t, 30, mergedTextVisionQuota(10, 20))
}

func TestTieredFallbackDoesNotDoubleCountVisionReserve(t *testing.T) {
	const mainEstimate = 100
	const visionActual = 20
	info := &relaycommon.RelayInfo{
		FinalPreConsumedQuota: mainEstimate + visionActual,
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:              "tiered_expr",
			ExprString:               "invalid(",
			ExprHash:                 billingexpr.ExprHashString("invalid("),
			EstimatedQuotaAfterGroup: mainEstimate,
		},
	}

	ok, mainQuota, result := TryTieredSettle(info, billingexpr.TokenParams{P: 10})

	require.True(t, ok)
	require.Nil(t, result)
	require.Equal(t, mainEstimate, mainQuota)
	require.Equal(t, mainEstimate+visionActual, mergedTextVisionQuota(mainQuota, visionActual))
}

func TestPostTextConsumeQuotaAttributesVisionBatchesToActualChannels(t *testing.T) {
	const userID, mainChannelID, visionChannelID, backupVisionChannelID = 9101, 9102, 9103, 9104
	seedUser(t, userID, 100_000)
	seedChannel(t, mainChannelID)
	seedChannel(t, visionChannelID)
	seedChannel(t, backupVisionChannelID)
	t.Cleanup(func() {
		model.DB.Delete(&model.User{}, userID)
		model.DB.Delete(&model.Channel{}, []int{mainChannelID, visionChannelID, backupVisionChannelID})
		model.LOG_DB.Where("request_id = ?", "vision-billing-test").Delete(&model.Log{})
	})

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	ctx.Set(common.RequestIdKey, "vision-billing-test")
	ctx.Set("token_name", "test-token")
	mainUsage := &dto.Usage{PromptTokens: 100, CompletionTokens: 10}
	relayInfo := &relaycommon.RelayInfo{
		UserId:                userID,
		UsingGroup:            "default",
		OriginModelName:       "deepseek-main",
		StartTime:             time.Now(),
		FirstResponseTime:     time.Now(),
		FinalPreConsumedQuota: 150,
		ChannelMeta:           &relaycommon.ChannelMeta{ChannelId: mainChannelID},
		PriceData: types.PriceData{
			ModelRatio:      1,
			CompletionRatio: 2,
			GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
		},
		VisionBilling: &relaycommon.VisionBillingComponent{
			ChannelId:      visionChannelID,
			ModelName:      "minimax-m3",
			VisionAlias:    "deepseek-vision",
			EstimatedQuota: 30,
			LatencyMs:      800,
			ImageCount:     2,
			Components: []*relaycommon.VisionBillingComponent{
				{
					ChannelId: visionChannelID,
					Usage:     &dto.Usage{PromptTokens: 20, CompletionTokens: 5},
					PriceData: types.PriceData{ModelRatio: 0.5, CompletionRatio: 2,
						GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1}},
				},
				{
					ChannelId: backupVisionChannelID,
					Usage:     &dto.Usage{PromptTokens: 20, CompletionTokens: 5},
					PriceData: types.PriceData{ModelRatio: 0.5, CompletionRatio: 2,
						GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1}},
				},
			},
		},
	}

	PostTextConsumeQuota(ctx, relayInfo, mainUsage, nil)

	var user model.User
	require.NoError(t, model.DB.Select("used_quota", "request_count").First(&user, userID).Error)
	assert.Equal(t, 150, user.UsedQuota)
	assert.Equal(t, 1, user.RequestCount)

	var mainChannel, visionChannel, backupVisionChannel model.Channel
	require.NoError(t, model.DB.Select("used_quota").First(&mainChannel, mainChannelID).Error)
	require.NoError(t, model.DB.Select("used_quota").First(&visionChannel, visionChannelID).Error)
	require.NoError(t, model.DB.Select("used_quota").First(&backupVisionChannel, backupVisionChannelID).Error)
	assert.Equal(t, int64(120), mainChannel.UsedQuota)
	assert.Equal(t, int64(15), visionChannel.UsedQuota)
	assert.Equal(t, int64(15), backupVisionChannel.UsedQuota)

	var logs []model.Log
	require.NoError(t, model.LOG_DB.Where("request_id = ?", "vision-billing-test").Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, 150, logs[0].Quota)
	other := map[string]interface{}{}
	require.NoError(t, common.UnmarshalJsonStr(logs[0].Other, &other))
	vision, ok := other["vision"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "minimax-m3", vision["model"])
	assert.Equal(t, "deepseek-vision", vision["alias"])
	assert.Equal(t, float64(30), vision["actual_quota"])
	assert.Equal(t, false, vision["cache_hit"])

	assert.Equal(t, 100, mainUsage.PromptTokens)
}

func TestPostTextConsumeQuotaSettlementFailureDoesNotRecordConsumption(t *testing.T) {
	const userID, channelID = 9111, 9112
	require.NoError(t, model.DB.Create(&model.User{Id: userID, Username: "billing_settlement_failure", Quota: 100_000, Status: common.UserStatusEnabled, AffCode: "billing-settlement-failure"}).Error)
	seedChannel(t, channelID)
	previousErrorLogEnabled := constant.ErrorLogEnabled
	constant.ErrorLogEnabled = true
	t.Cleanup(func() {
		constant.ErrorLogEnabled = previousErrorLogEnabled
		model.DB.Unscoped().Delete(&model.User{}, userID)
		model.DB.Delete(&model.Channel{}, channelID)
		model.LOG_DB.Where("request_id = ?", "billing-settlement-failure-test").Delete(&model.Log{})
	})

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	ctx.Set(common.RequestIdKey, "billing-settlement-failure-test")
	ctx.Set("token_name", "test-token")
	settler := &failingBillingSettler{}
	relayInfo := &relaycommon.RelayInfo{
		UserId:                userID,
		UsingGroup:            "default",
		OriginModelName:       "deepseek-main",
		StartTime:             time.Now(),
		FinalPreConsumedQuota: 100,
		Billing:               settler,
		ChannelMeta:           &relaycommon.ChannelMeta{ChannelId: channelID},
		PriceData: types.PriceData{
			ModelRatio:      1,
			CompletionRatio: 2,
			GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
		},
	}

	PostTextConsumeQuota(ctx, relayInfo, &dto.Usage{PromptTokens: 100, CompletionTokens: 10}, nil)

	require.Equal(t, 120, settler.actualQuota)
	var user model.User
	require.NoError(t, model.DB.Select("used_quota", "request_count").First(&user, userID).Error)
	require.Zero(t, user.UsedQuota)
	require.Zero(t, user.RequestCount)
	var channel model.Channel
	require.NoError(t, model.DB.Select("used_quota").First(&channel, channelID).Error)
	require.Zero(t, channel.UsedQuota)

	var logs []model.Log
	require.NoError(t, model.LOG_DB.Where("request_id = ?", "billing-settlement-failure-test").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, model.LogTypeError, logs[0].Type)
	require.Zero(t, logs[0].Quota)
	require.Contains(t, logs[0].Content, "billing settlement failed")
}

func TestRecordFailedVisionCostUsesErrorLogWithoutUserConsumption(t *testing.T) {
	const userID, mainChannelID, visionChannelID = 9201, 9202, 9203
	seedChannel(t, mainChannelID)
	seedChannel(t, visionChannelID)
	previousErrorLogEnabled := constant.ErrorLogEnabled
	constant.ErrorLogEnabled = true
	t.Cleanup(func() {
		constant.ErrorLogEnabled = previousErrorLogEnabled
		model.DB.Delete(&model.Channel{}, []int{mainChannelID, visionChannelID})
		model.LOG_DB.Where("request_id = ?", "vision-failed-main-test").Delete(&model.Log{})
	})

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	ctx.Set(common.RequestIdKey, "vision-failed-main-test")
	ctx.Set("token_name", "test-token")
	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         77,
		UsingGroup:      "default",
		OriginModelName: "deepseek-main",
		StartTime:       time.Now(),
		IsStream:        true,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: mainChannelID},
		VisionBilling: &relaycommon.VisionBillingComponent{
			ChannelId:      visionChannelID,
			ModelName:      "minimax-m3",
			VisionAlias:    "deepseek-vision",
			Usage:          &dto.Usage{PromptTokens: 40, CompletionTokens: 10},
			EstimatedQuota: 30,
			LatencyMs:      800,
			ImageCount:     1,
			PriceData: types.PriceData{
				ModelRatio:      0.5,
				CompletionRatio: 2,
				GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
			},
		},
	}
	apiErr := types.NewErrorWithStatusCode(
		fmt.Errorf("main upstream failed"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusBadGateway,
	)

	RecordFailedVisionCost(ctx, relayInfo, apiErr)

	var mainChannel, visionChannel model.Channel
	require.NoError(t, model.DB.Select("used_quota").First(&mainChannel, mainChannelID).Error)
	require.NoError(t, model.DB.Select("used_quota").First(&visionChannel, visionChannelID).Error)
	require.Zero(t, mainChannel.UsedQuota)
	require.Equal(t, int64(30), visionChannel.UsedQuota)

	var logs []model.Log
	require.NoError(t, model.LOG_DB.Where("request_id = ?", "vision-failed-main-test").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, model.LogTypeError, logs[0].Type)
	require.Zero(t, logs[0].Quota)
	require.Equal(t, visionChannelID, logs[0].ChannelId)

	other := map[string]interface{}{}
	require.NoError(t, common.UnmarshalJsonStr(logs[0].Other, &other))
	require.Equal(t, true, other["vision_cost_only"])
	require.Equal(t, float64(0), other["user_quota"])
	vision, ok := other["vision"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, float64(30), vision["actual_quota"])

	var consumeLogs int64
	require.NoError(t, model.LOG_DB.Model(&model.Log{}).
		Where("request_id = ? AND type = ?", "vision-failed-main-test", model.LogTypeConsume).
		Count(&consumeLogs).Error)
	require.Zero(t, consumeLogs)
}

func TestCalculateTextQuotaSummaryUnifiedForClaudeSemantic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 200,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         100,
			CachedCreationTokens: 50,
		},
		ClaudeCacheCreation5mTokens: 10,
		ClaudeCacheCreation1hTokens: 20,
	}

	priceData := types.PriceData{
		ModelRatio:           1,
		CompletionRatio:      2,
		CacheRatio:           0.1,
		CacheCreationRatio:   1.25,
		CacheCreation5mRatio: 1.25,
		CacheCreation1hRatio: 2,
		GroupRatioInfo: types.GroupRatioInfo{
			GroupRatio: 1,
		},
	}

	chatRelayInfo := &relaycommon.RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		FinalRequestRelayFormat: types.RelayFormatClaude,
		OriginModelName:         "claude-3-7-sonnet",
		PriceData:               priceData,
		StartTime:               time.Now(),
	}
	messageRelayInfo := &relaycommon.RelayInfo{
		RelayFormat:             types.RelayFormatClaude,
		FinalRequestRelayFormat: types.RelayFormatClaude,
		OriginModelName:         "claude-3-7-sonnet",
		PriceData:               priceData,
		StartTime:               time.Now(),
	}

	chatSummary := calculateTextQuotaSummary(ctx, chatRelayInfo, usage)
	messageSummary := calculateTextQuotaSummary(ctx, messageRelayInfo, usage)

	require.Equal(t, messageSummary.Quota, chatSummary.Quota)
	require.Equal(t, messageSummary.CacheCreationTokens5m, chatSummary.CacheCreationTokens5m)
	require.Equal(t, messageSummary.CacheCreationTokens1h, chatSummary.CacheCreationTokens1h)
	require.True(t, chatSummary.IsClaudeUsageSemantic)
	require.Equal(t, 1488, chatSummary.Quota)
}

func TestCalculateTextQuotaSummaryUsesSplitClaudeCacheCreationRatios(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		FinalRequestRelayFormat: types.RelayFormatClaude,
		OriginModelName:         "claude-3-7-sonnet",
		PriceData: types.PriceData{
			ModelRatio:           1,
			CompletionRatio:      1,
			CacheRatio:           0,
			CacheCreationRatio:   1,
			CacheCreation5mRatio: 2,
			CacheCreation1hRatio: 3,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 0,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedCreationTokens: 10,
		},
		ClaudeCacheCreation5mTokens: 2,
		ClaudeCacheCreation1hTokens: 3,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// 100 + remaining(5)*1 + 2*2 + 3*3 = 118
	require.Equal(t, 118, summary.Quota)
}

func TestCalculateTextQuotaSummaryUsesAnthropicUsageSemanticFromUpstreamUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		RelayFormat:     types.RelayFormatOpenAI,
		OriginModelName: "claude-3-7-sonnet",
		PriceData: types.PriceData{
			ModelRatio:           1,
			CompletionRatio:      2,
			CacheRatio:           0.1,
			CacheCreationRatio:   1.25,
			CacheCreation5mRatio: 1.25,
			CacheCreation1hRatio: 2,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 200,
		UsageSemantic:    "anthropic",
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         100,
			CachedCreationTokens: 50,
		},
		ClaudeCacheCreation5mTokens: 10,
		ClaudeCacheCreation1hTokens: 20,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	require.True(t, summary.IsClaudeUsageSemantic)
	require.Equal(t, "anthropic", summary.UsageSemantic)
	require.Equal(t, 1488, summary.Quota)
}

func TestCacheWriteTokensTotal(t *testing.T) {
	t.Run("split cache creation", func(t *testing.T) {
		summary := textQuotaSummary{
			CacheCreationTokens:   50,
			CacheCreationTokens5m: 10,
			CacheCreationTokens1h: 20,
		}
		require.Equal(t, 50, cacheWriteTokensTotal(summary))
	})

	t.Run("legacy cache creation", func(t *testing.T) {
		summary := textQuotaSummary{CacheCreationTokens: 50}
		require.Equal(t, 50, cacheWriteTokensTotal(summary))
	})

	t.Run("split cache creation without aggregate remainder", func(t *testing.T) {
		summary := textQuotaSummary{
			CacheCreationTokens5m: 10,
			CacheCreationTokens1h: 20,
		}
		require.Equal(t, 30, cacheWriteTokensTotal(summary))
	})
}

func TestCalculateTextQuotaSummaryHandlesLegacyClaudeDerivedOpenAIUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		RelayFormat:     types.RelayFormatOpenAI,
		OriginModelName: "claude-3-7-sonnet",
		PriceData: types.PriceData{
			ModelRatio:           1,
			CompletionRatio:      5,
			CacheRatio:           0.1,
			CacheCreationRatio:   1.25,
			CacheCreation5mRatio: 1.25,
			CacheCreation1hRatio: 2,
			GroupRatioInfo:       types.GroupRatioInfo{GroupRatio: 1},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     62,
		CompletionTokens: 95,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 3544,
		},
		ClaudeCacheCreation5mTokens: 586,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// 62 + 3544*0.1 + 586*1.25 + 95*5 = 1624.9 => 1624
	require.Equal(t, 1624, summary.Quota)
}

func TestCalculateTextQuotaSummarySeparatesOpenRouterCacheReadFromPromptBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "openai/gpt-4.1",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenRouter,
		},
		PriceData: types.PriceData{
			ModelRatio:         1,
			CompletionRatio:    1,
			CacheRatio:         0.1,
			CacheCreationRatio: 1.25,
			GroupRatioInfo:     types.GroupRatioInfo{GroupRatio: 1},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     2604,
		CompletionTokens: 383,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 2432,
		},
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// OpenRouter OpenAI-format display keeps prompt_tokens as total input,
	// but billing still separates normal input from cache read tokens.
	// quota = (2604 - 2432) + 2432*0.1 + 383 = 798.2 => 798
	require.Equal(t, 2604, summary.PromptTokens)
	require.Equal(t, 798, summary.Quota)
}

func TestCalculateTextQuotaSummarySeparatesOpenRouterCacheCreationFromPromptBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "openai/gpt-4.1",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenRouter,
		},
		PriceData: types.PriceData{
			ModelRatio:         1,
			CompletionRatio:    1,
			CacheCreationRatio: 1.25,
			GroupRatioInfo:     types.GroupRatioInfo{GroupRatio: 1},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     2604,
		CompletionTokens: 383,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedCreationTokens: 100,
		},
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// prompt_tokens is still logged as total input, but cache creation is billed separately.
	// quota = (2604 - 100) + 100*1.25 + 383 = 3012
	require.Equal(t, 2604, summary.PromptTokens)
	require.Equal(t, 3012, summary.Quota)
}

func TestCalculateTextQuotaSummaryKeepsPrePRClaudeOpenRouterBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		FinalRequestRelayFormat: types.RelayFormatClaude,
		OriginModelName:         "anthropic/claude-3.7-sonnet",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenRouter,
		},
		PriceData: types.PriceData{
			ModelRatio:         1,
			CompletionRatio:    1,
			CacheRatio:         0.1,
			CacheCreationRatio: 1.25,
			GroupRatioInfo:     types.GroupRatioInfo{GroupRatio: 1},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     2604,
		CompletionTokens: 383,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 2432,
		},
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// Pre-PR PostClaudeConsumeQuota behavior for OpenRouter:
	// prompt = 2604 - 2432 = 172
	// quota = 172 + 2432*0.1 + 383 = 798.2 => 798
	require.True(t, summary.IsClaudeUsageSemantic)
	require.Equal(t, 172, summary.PromptTokens)
	require.Equal(t, 798, summary.Quota)
}

func TestComposeTieredTextQuotaKeepsToolCallSurcharges(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set("image_generation_call", true)
	ctx.Set("image_generation_call_quality", "low")
	ctx.Set("image_generation_call_size", "1024x1024")

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "o1",
		PriceData: types.PriceData{
			ModelRatio:      1,
			CompletionRatio: 1,
			GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
		},
		ResponsesUsageInfo: &relaycommon.ResponsesUsageInfo{
			BuiltInTools: map[string]*relaycommon.BuildInToolInfo{
				dto.BuildInToolWebSearchPreview: &relaycommon.BuildInToolInfo{
					CallCount: 1,
				},
				dto.BuildInToolFileSearch: &relaycommon.BuildInToolInfo{
					CallCount: 2,
				},
			},
		},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:               "tiered_expr",
			GroupRatio:                1,
			EstimatedQuotaBeforeGroup: 1000,
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)
	quota := composeTieredTextQuota(relayInfo, summary, 1000, &billingexpr.TieredResult{
		ActualQuotaBeforeGroup: 1000,
		ActualQuotaAfterGroup:  1000,
	})

	require.Equal(t, int64(13000), summary.ToolCallSurchargeQuota.Round(0).IntPart())
	require.Equal(t, 14000, quota)
}

func TestComposeTieredTextQuotaFallbackKeepsToolCallSurcharges(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set("claude_web_search_requests", 2)

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "claude-3-7-sonnet",
		PriceData: types.PriceData{
			ModelRatio:      1,
			CompletionRatio: 1,
			GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1.25},
		},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:               "tiered_expr",
			GroupRatio:                1.25,
			EstimatedQuotaBeforeGroup: 1000,
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)
	quota := composeTieredTextQuota(relayInfo, summary, 1250, nil)

	require.Equal(t, int64(12500), summary.ToolCallSurchargeQuota.Round(0).IntPart())
	require.Equal(t, 13750, quota)
}

func TestComposeTieredTextQuotaErrorFallbackAddsToolSurcharge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set("claude_web_search_requests", 2)

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "claude-3-7-sonnet",
		PriceData: types.PriceData{
			ModelRatio:      1,
			CompletionRatio: 1,
			GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1.25},
		},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:               "tiered_expr",
			GroupRatio:                1.25,
			EstimatedQuotaBeforeGroup: 1000,
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// tieredResult=nil simulates a settlement error with a frozen estimate.
	preConsumedFallback := 2000
	quota := composeTieredTextQuota(relayInfo, summary, preConsumedFallback, nil)

	require.Equal(t, int64(12500), summary.ToolCallSurchargeQuota.Round(0).IntPart())
	require.Equal(t, 14500, quota)
}
