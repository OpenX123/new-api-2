package service

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type textQuotaSummary struct {
	PromptTokens             int
	CompletionTokens         int
	TotalTokens              int
	CacheTokens              int
	CacheCreationTokens      int
	CacheCreationTokens5m    int
	CacheCreationTokens1h    int
	ImageTokens              int
	AudioTokens              int
	ModelName                string
	TokenName                string
	UseTimeSeconds           int64
	CompletionRatio          float64
	CacheRatio               float64
	ImageRatio               float64
	ModelRatio               float64
	GroupRatio               float64
	ModelPrice               float64
	CacheCreationRatio       float64
	CacheCreationRatio5m     float64
	CacheCreationRatio1h     float64
	Quota                    int
	IsClaudeUsageSemantic    bool
	UsageSemantic            string
	WebSearchPrice           float64
	WebSearchCallCount       int
	ClaudeWebSearchPrice     float64
	ClaudeWebSearchCallCount int
	FileSearchPrice          float64
	FileSearchCallCount      int
	AudioInputPrice          float64
	ImageGenerationCallPrice float64
	ToolCallSurchargeQuota   decimal.Decimal
}

func cacheWriteTokensTotal(summary textQuotaSummary) int {
	if summary.CacheCreationTokens5m > 0 || summary.CacheCreationTokens1h > 0 {
		splitCacheWriteTokens := summary.CacheCreationTokens5m + summary.CacheCreationTokens1h
		if summary.CacheCreationTokens > splitCacheWriteTokens {
			return summary.CacheCreationTokens
		}
		return splitCacheWriteTokens
	}
	return summary.CacheCreationTokens
}

func isLegacyClaudeDerivedOpenAIUsage(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) bool {
	if relayInfo == nil || usage == nil {
		return false
	}
	if relayInfo.GetFinalRequestRelayFormat() == types.RelayFormatClaude {
		return false
	}
	if usage.UsageSource != "" || usage.UsageSemantic != "" {
		return false
	}
	return usage.ClaudeCacheCreation5mTokens > 0 || usage.ClaudeCacheCreation1hTokens > 0
}

func calculateTextToolCallSurcharge(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, summary *textQuotaSummary) decimal.Decimal {
	dGroupRatio := decimal.NewFromFloat(summary.GroupRatio)
	dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)

	var surcharge decimal.Decimal

	if relayInfo.ResponsesUsageInfo != nil {
		if webSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearchPreview]; exists && webSearchTool.CallCount > 0 {
			summary.WebSearchCallCount = webSearchTool.CallCount
			summary.WebSearchPrice = operation_setting.GetToolPriceForModel("web_search_preview", summary.ModelName)
			surcharge = surcharge.Add(decimal.NewFromFloat(summary.WebSearchPrice).
				Mul(decimal.NewFromInt(int64(webSearchTool.CallCount))).
				Div(decimal.NewFromInt(1000)).
				Mul(dGroupRatio).
				Mul(dQuotaPerUnit))
		}
	} else if strings.HasSuffix(summary.ModelName, "search-preview") {
		summary.WebSearchCallCount = 1
		summary.WebSearchPrice = operation_setting.GetToolPriceForModel("web_search_preview", summary.ModelName)
		surcharge = surcharge.Add(decimal.NewFromFloat(summary.WebSearchPrice).
			Div(decimal.NewFromInt(1000)).
			Mul(dGroupRatio).
			Mul(dQuotaPerUnit))
	}

	summary.ClaudeWebSearchCallCount = ctx.GetInt("claude_web_search_requests")
	if summary.ClaudeWebSearchCallCount > 0 {
		summary.ClaudeWebSearchPrice = operation_setting.GetToolPrice("web_search")
		surcharge = surcharge.Add(decimal.NewFromFloat(summary.ClaudeWebSearchPrice).
			Div(decimal.NewFromInt(1000)).
			Mul(dGroupRatio).
			Mul(dQuotaPerUnit).
			Mul(decimal.NewFromInt(int64(summary.ClaudeWebSearchCallCount))))
	}

	if relayInfo.ResponsesUsageInfo != nil {
		if fileSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolFileSearch]; exists && fileSearchTool.CallCount > 0 {
			summary.FileSearchCallCount = fileSearchTool.CallCount
			summary.FileSearchPrice = operation_setting.GetToolPrice("file_search")
			surcharge = surcharge.Add(decimal.NewFromFloat(summary.FileSearchPrice).
				Mul(decimal.NewFromInt(int64(fileSearchTool.CallCount))).
				Div(decimal.NewFromInt(1000)).
				Mul(dGroupRatio).
				Mul(dQuotaPerUnit))
		}
	}

	if ctx.GetBool("image_generation_call") {
		summary.ImageGenerationCallPrice = operation_setting.GetGPTImage1PriceOnceCall(ctx.GetString("image_generation_call_quality"), ctx.GetString("image_generation_call_size"))
		surcharge = surcharge.Add(decimal.NewFromFloat(summary.ImageGenerationCallPrice).
			Mul(dGroupRatio).
			Mul(dQuotaPerUnit))
	}

	return surcharge
}

func composeTieredTextQuota(relayInfo *relaycommon.RelayInfo, summary textQuotaSummary, tieredQuota int, tieredResult *billingexpr.TieredResult) int {
	if summary.ToolCallSurchargeQuota.IsZero() {
		return tieredQuota
	}

	if tieredResult != nil {
		if snap := relayInfo.TieredBillingSnapshot; snap != nil {
			return decimalToQuota(decimal.NewFromFloat(tieredResult.ActualQuotaBeforeGroup).
				Mul(decimal.NewFromFloat(snap.GroupRatio)).
				Add(summary.ToolCallSurchargeQuota))
		}
	}

	return tieredQuota + decimalToQuota(summary.ToolCallSurchargeQuota)
}

func calculateTextQuotaSummary(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage) textQuotaSummary {
	summary := textQuotaSummary{
		ModelName:            relayInfo.OriginModelName,
		TokenName:            ctx.GetString("token_name"),
		UseTimeSeconds:       time.Now().Unix() - relayInfo.StartTime.Unix(),
		CompletionRatio:      relayInfo.PriceData.CompletionRatio,
		CacheRatio:           relayInfo.PriceData.CacheRatio,
		ImageRatio:           relayInfo.PriceData.ImageRatio,
		ModelRatio:           relayInfo.PriceData.ModelRatio,
		GroupRatio:           relayInfo.PriceData.GroupRatioInfo.GroupRatio,
		ModelPrice:           relayInfo.PriceData.ModelPrice,
		CacheCreationRatio:   relayInfo.PriceData.CacheCreationRatio,
		CacheCreationRatio5m: relayInfo.PriceData.CacheCreation5mRatio,
		CacheCreationRatio1h: relayInfo.PriceData.CacheCreation1hRatio,
		UsageSemantic:        usageSemanticFromUsage(relayInfo, usage),
	}
	summary.IsClaudeUsageSemantic = summary.UsageSemantic == "anthropic"

	if usage == nil {
		usage = &dto.Usage{
			PromptTokens:     relayInfo.GetEstimatePromptTokens(),
			CompletionTokens: 0,
			TotalTokens:      relayInfo.GetEstimatePromptTokens(),
		}
	}

	summary.PromptTokens = usage.PromptTokens
	summary.CompletionTokens = usage.CompletionTokens
	summary.TotalTokens = common.QuotaFromFloat(float64(usage.PromptTokens) + float64(usage.CompletionTokens))
	summary.CacheTokens = usage.PromptTokensDetails.CachedTokens
	summary.CacheCreationTokens = usage.PromptTokensDetails.CachedCreationTokens
	summary.CacheCreationTokens5m = usage.ClaudeCacheCreation5mTokens
	summary.CacheCreationTokens1h = usage.ClaudeCacheCreation1hTokens
	summary.ImageTokens = usage.PromptTokensDetails.ImageTokens
	summary.AudioTokens = usage.PromptTokensDetails.AudioTokens
	legacyClaudeDerived := isLegacyClaudeDerivedOpenAIUsage(relayInfo, usage)
	isOpenRouterClaudeBilling := relayInfo.ChannelMeta != nil &&
		relayInfo.ChannelType == constant.ChannelTypeOpenRouter &&
		summary.IsClaudeUsageSemantic

	if isOpenRouterClaudeBilling {
		summary.PromptTokens -= summary.CacheTokens
		isUsingCustomSettings := relayInfo.PriceData.UsePrice || hasCustomModelRatio(summary.ModelName, relayInfo.PriceData.ModelRatio)
		if summary.CacheCreationTokens == 0 && relayInfo.PriceData.CacheCreationRatio != 1 && usage.Cost != 0 && !isUsingCustomSettings {
			maybeCacheCreationTokens := CalcOpenRouterCacheCreateTokens(*usage, relayInfo.PriceData)
			if maybeCacheCreationTokens >= 0 && summary.PromptTokens >= maybeCacheCreationTokens {
				summary.CacheCreationTokens = maybeCacheCreationTokens
			}
		}
		summary.PromptTokens -= summary.CacheCreationTokens
	}

	dPromptTokens := decimal.NewFromInt(int64(summary.PromptTokens))
	dCacheTokens := decimal.NewFromInt(int64(summary.CacheTokens))
	dImageTokens := decimal.NewFromInt(int64(summary.ImageTokens))
	dAudioTokens := decimal.NewFromInt(int64(summary.AudioTokens))
	dCompletionTokens := decimal.NewFromInt(int64(summary.CompletionTokens))
	dCachedCreationTokens := decimal.NewFromInt(int64(summary.CacheCreationTokens))
	dCompletionRatio := decimal.NewFromFloat(summary.CompletionRatio)
	dCacheRatio := decimal.NewFromFloat(summary.CacheRatio)
	dImageRatio := decimal.NewFromFloat(summary.ImageRatio)
	dModelRatio := decimal.NewFromFloat(summary.ModelRatio)
	dGroupRatio := decimal.NewFromFloat(summary.GroupRatio)
	dModelPrice := decimal.NewFromFloat(summary.ModelPrice)
	dCacheCreationRatio := decimal.NewFromFloat(summary.CacheCreationRatio)
	dCacheCreationRatio5m := decimal.NewFromFloat(summary.CacheCreationRatio5m)
	dCacheCreationRatio1h := decimal.NewFromFloat(summary.CacheCreationRatio1h)
	dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)

	ratio := dModelRatio.Mul(dGroupRatio)
	summary.ToolCallSurchargeQuota = calculateTextToolCallSurcharge(ctx, relayInfo, &summary)

	var audioInputQuota decimal.Decimal
	if !relayInfo.PriceData.UsePrice {
		baseTokens := dPromptTokens

		var cachedTokensWithRatio decimal.Decimal
		if !dCacheTokens.IsZero() {
			if !summary.IsClaudeUsageSemantic && !legacyClaudeDerived {
				baseTokens = baseTokens.Sub(dCacheTokens)
			}
			cachedTokensWithRatio = dCacheTokens.Mul(dCacheRatio)
		}

		var cachedCreationTokensWithRatio decimal.Decimal
		hasSplitCacheCreationTokens := summary.CacheCreationTokens5m > 0 || summary.CacheCreationTokens1h > 0
		if !dCachedCreationTokens.IsZero() || hasSplitCacheCreationTokens {
			if !summary.IsClaudeUsageSemantic && !legacyClaudeDerived {
				baseTokens = baseTokens.Sub(dCachedCreationTokens)
				cachedCreationTokensWithRatio = dCachedCreationTokens.Mul(dCacheCreationRatio)
			} else {
				remaining := summary.CacheCreationTokens - summary.CacheCreationTokens5m - summary.CacheCreationTokens1h
				if remaining < 0 {
					remaining = 0
				}
				cachedCreationTokensWithRatio = decimal.NewFromInt(int64(remaining)).Mul(dCacheCreationRatio)
				cachedCreationTokensWithRatio = cachedCreationTokensWithRatio.Add(decimal.NewFromInt(int64(summary.CacheCreationTokens5m)).Mul(dCacheCreationRatio5m))
				cachedCreationTokensWithRatio = cachedCreationTokensWithRatio.Add(decimal.NewFromInt(int64(summary.CacheCreationTokens1h)).Mul(dCacheCreationRatio1h))
			}
		}

		var imageTokensWithRatio decimal.Decimal
		if !dImageTokens.IsZero() {
			baseTokens = baseTokens.Sub(dImageTokens)
			imageTokensWithRatio = dImageTokens.Mul(dImageRatio)
		}

		if !dAudioTokens.IsZero() {
			summary.AudioInputPrice = operation_setting.GetGeminiInputAudioPricePerMillionTokens(summary.ModelName)
			if summary.AudioInputPrice > 0 {
				baseTokens = baseTokens.Sub(dAudioTokens)
				audioInputQuota = decimal.NewFromFloat(summary.AudioInputPrice).
					Div(decimal.NewFromInt(1000000)).Mul(dAudioTokens).Mul(dGroupRatio).Mul(dQuotaPerUnit)
			}
		}

		promptQuota := baseTokens.Add(cachedTokensWithRatio).Add(imageTokensWithRatio).Add(cachedCreationTokensWithRatio)
		completionQuota := dCompletionTokens.Mul(dCompletionRatio)
		quotaCalculateDecimal := promptQuota.Add(completionQuota).Mul(ratio)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(summary.ToolCallSurchargeQuota)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(audioInputQuota)

		if len(relayInfo.PriceData.OtherRatios) > 0 {
			for _, otherRatio := range relayInfo.PriceData.OtherRatios {
				quotaCalculateDecimal = quotaCalculateDecimal.Mul(decimal.NewFromFloat(otherRatio))
			}
		}

		if !ratio.IsZero() && quotaCalculateDecimal.LessThanOrEqual(decimal.Zero) {
			quotaCalculateDecimal = decimal.NewFromInt(1)
		}
		summary.Quota = decimalToQuota(quotaCalculateDecimal)
	} else {
		quotaCalculateDecimal := dModelPrice.Mul(dQuotaPerUnit).Mul(dGroupRatio)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(summary.ToolCallSurchargeQuota)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(audioInputQuota)
		if len(relayInfo.PriceData.OtherRatios) > 0 {
			for _, otherRatio := range relayInfo.PriceData.OtherRatios {
				quotaCalculateDecimal = quotaCalculateDecimal.Mul(decimal.NewFromFloat(otherRatio))
			}
		}
		summary.Quota = decimalToQuota(quotaCalculateDecimal)
	}

	if summary.TotalTokens == 0 {
		summary.Quota = 0
	} else if !ratio.IsZero() && summary.Quota == 0 {
		summary.Quota = 1
	}

	return summary
}

// decimalToQuota converts a computed quota decimal to int with saturation
// (see common.QuotaFromFloat). Oversized multipliers (e.g. an absurd image
// generation count) must never wrap around and turn a charge into a credit.
func decimalToQuota(d decimal.Decimal) int {
	f, _ := d.Round(0).Float64()
	return common.QuotaFromFloat(f)
}

func usageSemanticFromUsage(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) string {
	if usage != nil && usage.UsageSemantic != "" {
		return usage.UsageSemantic
	}
	if relayInfo != nil && relayInfo.GetFinalRequestRelayFormat() == types.RelayFormatClaude {
		return "anthropic"
	}
	return "openai"
}

// CalculateVisionActualQuota computes the vision preflight charge without
// mutating its Usage or any database state. An application-level facts cache
// hit is free; cache usage reported by the vision upstream follows its price.
func CalculateVisionActualQuota(component *relaycommon.VisionBillingComponent) (int, *billingexpr.TieredResult, error) {
	if component == nil {
		return 0, nil, nil
	}
	if len(component.Components) > 0 {
		total := 0
		var batchErr error
		for _, child := range component.Components {
			quota, _, err := CalculateVisionActualQuota(child)
			total = common.QuotaFromFloat(float64(total) + float64(quota))
			if err != nil && batchErr == nil {
				batchErr = err
			}
		}
		return total, nil, batchErr
	}
	if component.CacheHit {
		return 0, nil, nil
	}
	fallbackQuota := component.EstimatedQuota
	if fallbackQuota < 0 {
		fallbackQuota = 0
	}
	if component.Usage == nil {
		return fallbackQuota, nil, nil
	}

	usage := *component.Usage

	if snap := component.TieredBillingSnapshot; snap != nil && snap.BillingMode == "tiered_expr" {
		requestInput := billingexpr.RequestInput{}
		if component.BillingRequestInput != nil {
			requestInput = *component.BillingRequestInput
		}
		usedVars := billingexpr.UsedVars(snap.ExprString)
		params := BuildTieredTokenParams(&usage, usage.UsageSemantic == "anthropic", usedVars)
		result, err := billingexpr.ComputeTieredQuotaWithRequest(snap, params, requestInput)
		if err != nil {
			return fallbackQuota, nil, err
		}
		if result.ActualQuotaAfterGroup < 0 {
			result.ActualQuotaAfterGroup = 0
		}
		return result.ActualQuotaAfterGroup, &result, nil
	}

	promptTokens := usage.PromptTokens
	completionTokens := usage.CompletionTokens
	imageTokens := usage.PromptTokensDetails.ImageTokens
	cacheTokens := usage.PromptTokensDetails.CachedTokens
	if cacheTokens == 0 && usage.PromptCacheHitTokens > 0 {
		cacheTokens = usage.PromptCacheHitTokens
	}
	cacheCreationTokens := usage.PromptTokensDetails.CachedCreationTokens
	cacheCreation5mTokens := usage.ClaudeCacheCreation5mTokens
	cacheCreation1hTokens := usage.ClaudeCacheCreation1hTokens
	if promptTokens < 0 {
		promptTokens = 0
	}
	if completionTokens < 0 {
		completionTokens = 0
	}
	if imageTokens < 0 {
		imageTokens = 0
	}
	if cacheTokens < 0 {
		cacheTokens = 0
	}
	if cacheCreationTokens < 0 {
		cacheCreationTokens = 0
	}
	if cacheCreation5mTokens < 0 {
		cacheCreation5mTokens = 0
	}
	if cacheCreation1hTokens < 0 {
		cacheCreation1hTokens = 0
	}
	if promptTokens == 0 && completionTokens == 0 {
		return 0, nil, nil
	}
	if component.PriceData.FreeModel {
		return 0, nil, nil
	}

	priceData := component.PriceData
	groupRatio := decimal.NewFromFloat(priceData.GroupRatioInfo.GroupRatio)
	var quota decimal.Decimal
	if priceData.UsePrice {
		quota = decimal.NewFromFloat(priceData.ModelPrice).
			Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
			Mul(groupRatio)
	} else {
		promptTokenCount := decimal.NewFromInt(int64(promptTokens))
		imageTokenCount := decimal.NewFromInt(int64(imageTokens))
		cacheTokenCount := decimal.NewFromInt(int64(cacheTokens))
		cacheCreationTokenCount := decimal.NewFromInt(int64(cacheCreationTokens))
		cacheCreation5mTokenCount := decimal.NewFromInt(int64(cacheCreation5mTokens))
		cacheCreation1hTokenCount := decimal.NewFromInt(int64(cacheCreation1hTokens))
		basePromptTokens := promptTokenCount.Sub(imageTokenCount)
		if usage.UsageSemantic != "anthropic" {
			basePromptTokens = basePromptTokens.Sub(cacheTokenCount).Sub(cacheCreationTokenCount)
		}
		if basePromptTokens.IsNegative() {
			basePromptTokens = decimal.Zero
		}
		promptQuota := basePromptTokens.
			Add(imageTokenCount.Mul(decimal.NewFromFloat(priceData.ImageRatio))).
			Add(cacheTokenCount.Mul(decimal.NewFromFloat(priceData.CacheRatio)))
		if usage.UsageSemantic == "anthropic" {
			splitCacheCreationTokens := cacheCreation5mTokenCount.Add(cacheCreation1hTokenCount)
			if cacheCreationTokenCount.LessThan(splitCacheCreationTokens) {
				cacheCreationTokenCount = splitCacheCreationTokens
			}
			promptQuota = promptQuota.
				Add(cacheCreationTokenCount.Sub(splitCacheCreationTokens).Mul(decimal.NewFromFloat(priceData.CacheCreationRatio))).
				Add(cacheCreation5mTokenCount.Mul(decimal.NewFromFloat(priceData.CacheCreation5mRatio))).
				Add(cacheCreation1hTokenCount.Mul(decimal.NewFromFloat(priceData.CacheCreation1hRatio)))
		} else {
			promptQuota = promptQuota.Add(cacheCreationTokenCount.Mul(decimal.NewFromFloat(priceData.CacheCreationRatio)))
		}
		completionQuota := decimal.NewFromInt(int64(completionTokens)).
			Mul(decimal.NewFromFloat(priceData.CompletionRatio))
		quota = promptQuota.Add(completionQuota).
			Mul(decimal.NewFromFloat(priceData.ModelRatio)).
			Mul(groupRatio)
	}
	for _, otherRatio := range priceData.OtherRatios {
		quota = quota.Mul(decimal.NewFromFloat(otherRatio))
	}
	actualQuota := decimalToQuota(quota)
	if actualQuota < 0 {
		actualQuota = 0
	}
	if !priceData.UsePrice && priceData.ModelRatio*priceData.GroupRatioInfo.GroupRatio != 0 && actualQuota == 0 {
		actualQuota = 1
	}
	return actualQuota, nil, nil
}

func mergedTextVisionQuota(textQuota, visionQuota int) int {
	if textQuota < 0 {
		textQuota = 0
	}
	if visionQuota < 0 {
		visionQuota = 0
	}
	return common.QuotaFromFloat(float64(textQuota) + float64(visionQuota))
}

func appendVisionBillingInfo(other map[string]interface{}, component *relaycommon.VisionBillingComponent, tieredResult *billingexpr.TieredResult) {
	if component == nil || other == nil {
		return
	}
	usage := component.Usage
	vision := map[string]interface{}{
		"channel_id":      component.ChannelId,
		"failover_count":  component.FailoverCount,
		"model":           component.ModelName,
		"alias":           component.VisionAlias,
		"estimated_quota": component.EstimatedQuota,
		"actual_quota":    component.ActualQuota,
		"latency_ms":      component.LatencyMs,
		"cache_hit":       component.CacheHit,
		"image_count":     component.ImageCount,
		"price": map[string]interface{}{
			"use_price":        component.PriceData.UsePrice,
			"model_price":      component.PriceData.ModelPrice,
			"model_ratio":      component.PriceData.ModelRatio,
			"completion_ratio": component.PriceData.CompletionRatio,
			"group_ratio":      component.PriceData.GroupRatioInfo.GroupRatio,
		},
	}
	if len(component.AttemptedChannelIds) > 0 {
		vision["attempted_channel_ids"] = component.AttemptedChannelIds
	}
	if usage != nil {
		vision["prompt_tokens"] = usage.PromptTokens
		vision["completion_tokens"] = usage.CompletionTokens
		vision["total_tokens"] = common.QuotaFromFloat(float64(usage.PromptTokens) + float64(usage.CompletionTokens))
	} else if len(component.Components) > 0 {
		promptTokens := 0
		completionTokens := 0
		for _, child := range component.Components {
			if child != nil && child.Usage != nil {
				promptTokens = common.QuotaFromFloat(float64(promptTokens) + float64(child.Usage.PromptTokens))
				completionTokens = common.QuotaFromFloat(float64(completionTokens) + float64(child.Usage.CompletionTokens))
			}
		}
		vision["batch_count"] = len(component.Components)
		vision["prompt_tokens"] = promptTokens
		vision["completion_tokens"] = completionTokens
		vision["total_tokens"] = common.QuotaFromFloat(float64(promptTokens) + float64(completionTokens))
	}
	if snap := component.TieredBillingSnapshot; snap != nil {
		vision["billing_mode"] = snap.BillingMode
		vision["expr_b64"] = base64.StdEncoding.EncodeToString([]byte(snap.ExprString))
		if tieredResult != nil {
			vision["matched_tier"] = tieredResult.MatchedTier
		}
	}
	other["vision"] = vision
}

func updateVisionChannelUsedQuota(ctx *gin.Context, component *relaycommon.VisionBillingComponent) {
	if component == nil || component.CacheHit {
		return
	}
	if len(component.Components) > 0 {
		for _, child := range component.Components {
			updateVisionChannelUsedQuota(ctx, child)
		}
		return
	}
	if component.ChannelId == 0 {
		return
	}
	quota, _, err := CalculateVisionActualQuota(component)
	component.ActualQuota = quota
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("error calculating vision channel %d used quota: %s", component.ChannelId, err.Error()))
	}
	if quota > 0 {
		model.UpdateChannelUsedQuota(component.ChannelId, quota)
	}
}

// RecordFailedVisionCost keeps the real vision-channel cost when the main
// request fails. The user-facing billing session is refunded by the caller;
// this records only channel cost and an error log with zero user quota.
func RecordFailedVisionCost(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, apiErr *types.NewAPIError) {
	if ctx == nil || relayInfo == nil || apiErr == nil {
		return
	}
	component := relayInfo.VisionBilling
	if component == nil || component.CacheHit || component.ChannelId == 0 || (component.Usage == nil && len(component.Components) == 0) {
		return
	}

	quota, tieredResult, err := CalculateVisionActualQuota(component)
	component.ActualQuota = quota
	if err != nil {
		logger.LogError(ctx, "error calculating failed-request vision cost, using estimated quota: "+err.Error())
	}
	if quota <= 0 {
		return
	}

	updateVisionChannelUsedQuota(ctx, component)
	if !constant.ErrorLogEnabled {
		return
	}

	other := map[string]interface{}{
		"vision_cost_only": true,
		"user_quota":       0,
		"main_error_type":  apiErr.GetErrorType(),
		"main_error_code":  apiErr.GetErrorCode(),
		"main_status_code": apiErr.StatusCode,
	}
	appendVisionBillingInfo(other, component, tieredResult)
	model.RecordErrorLog(
		ctx,
		relayInfo.UserId,
		component.ChannelId,
		component.ModelName,
		ctx.GetString("token_name"),
		"vision channel cost recorded after main request failure",
		relayInfo.TokenId,
		int(time.Since(relayInfo.StartTime).Seconds()),
		relayInfo.IsStream,
		relayInfo.UsingGroup,
		other,
	)
}

func PostTextConsumeQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage, extraContent []string) {
	originUsage := usage
	if usage == nil {
		extraContent = append(extraContent, "上游无计费信息")
	}
	if originUsage != nil {
		ObserveChannelAffinityUsageCacheByRelayFormat(ctx, usage, relayInfo.GetFinalRequestRelayFormat())
	}

	adminRejectReason := common.GetContextKeyString(ctx, constant.ContextKeyAdminRejectReason)
	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	var tieredResult *billingexpr.TieredResult
	tieredBillingApplied := false
	if originUsage != nil {
		var tieredUsedVars map[string]bool
		if snap := relayInfo.TieredBillingSnapshot; snap != nil {
			tieredUsedVars = billingexpr.UsedVars(snap.ExprString)
		}
		tieredOk, tieredQuota, tieredRes := TryTieredSettle(relayInfo, BuildTieredTokenParams(usage, summary.IsClaudeUsageSemantic, tieredUsedVars))
		if tieredOk {
			tieredBillingApplied = true
			tieredResult = tieredRes
			summary.Quota = composeTieredTextQuota(relayInfo, summary, tieredQuota, tieredRes)
		}
	}

	visionQuota, visionTieredResult, visionErr := CalculateVisionActualQuota(relayInfo.VisionBilling)
	if relayInfo.VisionBilling != nil {
		relayInfo.VisionBilling.ActualQuota = visionQuota
	}
	if visionErr != nil {
		logger.LogError(ctx, "error calculating vision billing, using estimated quota: "+visionErr.Error())
		extraContent = append(extraContent, "视觉计费结算失败，按预估额度扣费")
	}
	totalQuota := mergedTextVisionQuota(summary.Quota, visionQuota)

	if summary.WebSearchCallCount > 0 {
		extraContent = append(extraContent, fmt.Sprintf("Web Search 调用 %d 次，调用花费 %s", summary.WebSearchCallCount, decimal.NewFromFloat(summary.WebSearchPrice).Mul(decimal.NewFromInt(int64(summary.WebSearchCallCount))).Div(decimal.NewFromInt(1000)).Mul(decimal.NewFromFloat(summary.GroupRatio)).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).String()))
	}
	if summary.ClaudeWebSearchCallCount > 0 {
		extraContent = append(extraContent, fmt.Sprintf("Claude Web Search 调用 %d 次，调用花费 %s", summary.ClaudeWebSearchCallCount, decimal.NewFromFloat(summary.ClaudeWebSearchPrice).Div(decimal.NewFromInt(1000)).Mul(decimal.NewFromFloat(summary.GroupRatio)).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Mul(decimal.NewFromInt(int64(summary.ClaudeWebSearchCallCount))).String()))
	}
	if summary.FileSearchCallCount > 0 {
		extraContent = append(extraContent, fmt.Sprintf("File Search 调用 %d 次，调用花费 %s", summary.FileSearchCallCount, decimal.NewFromFloat(summary.FileSearchPrice).Mul(decimal.NewFromInt(int64(summary.FileSearchCallCount))).Div(decimal.NewFromInt(1000)).Mul(decimal.NewFromFloat(summary.GroupRatio)).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).String()))
	}
	if summary.AudioInputPrice > 0 && summary.AudioTokens > 0 {
		extraContent = append(extraContent, fmt.Sprintf("Audio Input 花费 %s", decimal.NewFromFloat(summary.AudioInputPrice).Div(decimal.NewFromInt(1000000)).Mul(decimal.NewFromInt(int64(summary.AudioTokens))).Mul(decimal.NewFromFloat(summary.GroupRatio)).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).String()))
	}
	if summary.ImageGenerationCallPrice > 0 {
		extraContent = append(extraContent, fmt.Sprintf("Image Generation Call 花费 %s", decimal.NewFromFloat(summary.ImageGenerationCallPrice).Mul(decimal.NewFromFloat(summary.GroupRatio)).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).String()))
	}

	mainHasUsage := summary.TotalTokens != 0
	visionHasUsage := relayInfo.VisionBilling != nil && (relayInfo.VisionBilling.Usage != nil || visionQuota > 0)
	if !mainHasUsage {
		extraContent = append(extraContent, "上游没有返回计费信息，无法扣费（可能是上游超时）")
		logger.LogError(ctx, fmt.Sprintf("total tokens is 0, cannot consume quota, userId %d, channelId %d, tokenId %d, model %s， pre-consumed quota %d", relayInfo.UserId, relayInfo.ChannelId, relayInfo.TokenId, summary.ModelName, relayInfo.FinalPreConsumedQuota))
	}
	if err := SettleBilling(ctx, relayInfo, totalQuota); err != nil {
		logger.LogError(ctx, "error settling billing: "+err.Error())
		if constant.ErrorLogEnabled {
			model.RecordErrorLog(ctx, relayInfo.UserId, relayInfo.ChannelId, summary.ModelName, summary.TokenName,
				"billing settlement failed: "+err.Error(), relayInfo.TokenId, int(summary.UseTimeSeconds), relayInfo.IsStream, relayInfo.UsingGroup,
				map[string]interface{}{
					"billing_settlement_failed": true,
					"actual_quota":              totalQuota,
					"pre_consumed_quota":        relayInfo.FinalPreConsumedQuota,
				})
		}
		return
	}
	if mainHasUsage || visionHasUsage {
		model.UpdateUserUsedQuotaAndRequestCount(relayInfo.UserId, totalQuota)
	}
	if mainHasUsage {
		model.UpdateChannelUsedQuota(relayInfo.ChannelId, summary.Quota)
	}
	if relayInfo.VisionBilling != nil && visionHasUsage {
		updateVisionChannelUsedQuota(ctx, relayInfo.VisionBilling)
	}

	logModel := summary.ModelName
	if strings.HasPrefix(logModel, "gpt-4-gizmo") {
		logModel = "gpt-4-gizmo-*"
		extraContent = append(extraContent, fmt.Sprintf("模型 %s", summary.ModelName))
	}
	if strings.HasPrefix(logModel, "gpt-4o-gizmo") {
		logModel = "gpt-4o-gizmo-*"
		extraContent = append(extraContent, fmt.Sprintf("模型 %s", summary.ModelName))
	}

	logContent := strings.Join(extraContent, ", ")
	var other map[string]interface{}
	if summary.IsClaudeUsageSemantic {
		other = GenerateClaudeOtherInfo(ctx, relayInfo,
			summary.ModelRatio, summary.GroupRatio, summary.CompletionRatio,
			summary.CacheTokens, summary.CacheRatio,
			summary.CacheCreationTokens, summary.CacheCreationRatio,
			summary.CacheCreationTokens5m, summary.CacheCreationRatio5m,
			summary.CacheCreationTokens1h, summary.CacheCreationRatio1h,
			summary.ModelPrice, relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio)
		other["usage_semantic"] = "anthropic"
	} else {
		other = GenerateTextOtherInfo(ctx, relayInfo, summary.ModelRatio, summary.GroupRatio, summary.CompletionRatio, summary.CacheTokens, summary.CacheRatio, summary.ModelPrice, relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio)
	}
	if adminRejectReason != "" {
		other["reject_reason"] = adminRejectReason
	}
	if summary.ImageTokens != 0 {
		other["image"] = true
		other["image_ratio"] = summary.ImageRatio
		other["image_output"] = summary.ImageTokens
	}
	if summary.WebSearchCallCount > 0 {
		other["web_search"] = true
		other["web_search_call_count"] = summary.WebSearchCallCount
		other["web_search_price"] = summary.WebSearchPrice
	} else if summary.ClaudeWebSearchCallCount > 0 {
		other["web_search"] = true
		other["web_search_call_count"] = summary.ClaudeWebSearchCallCount
		other["web_search_price"] = summary.ClaudeWebSearchPrice
	}
	if summary.FileSearchCallCount > 0 {
		other["file_search"] = true
		other["file_search_call_count"] = summary.FileSearchCallCount
		other["file_search_price"] = summary.FileSearchPrice
	}
	if summary.AudioInputPrice > 0 && summary.AudioTokens > 0 {
		other["audio_input_seperate_price"] = true
		other["audio_input_token_count"] = summary.AudioTokens
		other["audio_input_price"] = summary.AudioInputPrice
	}
	if summary.ImageGenerationCallPrice > 0 {
		other["image_generation_call"] = true
		other["image_generation_call_price"] = summary.ImageGenerationCallPrice
	}
	if summary.CacheCreationTokens > 0 {
		other["cache_creation_tokens"] = summary.CacheCreationTokens
		other["cache_creation_ratio"] = summary.CacheCreationRatio
	}
	if summary.CacheCreationTokens5m > 0 {
		other["cache_creation_tokens_5m"] = summary.CacheCreationTokens5m
		other["cache_creation_ratio_5m"] = summary.CacheCreationRatio5m
	}
	if summary.CacheCreationTokens1h > 0 {
		other["cache_creation_tokens_1h"] = summary.CacheCreationTokens1h
		other["cache_creation_ratio_1h"] = summary.CacheCreationRatio1h
	}
	cacheWriteTokens := cacheWriteTokensTotal(summary)
	if cacheWriteTokens > 0 {
		// cache_write_tokens: normalized cache creation total for UI display.
		// If split 5m/1h values are present, this is their sum; otherwise it falls back
		// to cache_creation_tokens.
		other["cache_write_tokens"] = cacheWriteTokens
	}
	if relayInfo.GetFinalRequestRelayFormat() != types.RelayFormatClaude && usage != nil && usage.UsageSource != "" && usage.InputTokens > 0 {
		// input_tokens_total: explicit normalized total input used by the usage log UI.
		// Only write this field when upstream/current conversion has already provided a
		// reliable total input value and tagged the usage source. Do not infer it from
		// prompt/cache fields here, otherwise old upstream payloads may be double-counted.
		other["input_tokens_total"] = usage.InputTokens
	}
	if tieredBillingApplied {
		InjectTieredBillingInfo(other, relayInfo, tieredResult)
	}
	appendVisionBillingInfo(other, relayInfo.VisionBilling, visionTieredResult)

	model.RecordConsumeLog(ctx, relayInfo.UserId, model.RecordConsumeLogParams{
		ChannelId:        relayInfo.ChannelId,
		PromptTokens:     summary.PromptTokens,
		CompletionTokens: summary.CompletionTokens,
		ModelName:        logModel,
		TokenName:        summary.TokenName,
		Quota:            totalQuota,
		Content:          logContent,
		TokenId:          relayInfo.TokenId,
		UseTimeSeconds:   int(summary.UseTimeSeconds),
		IsStream:         relayInfo.IsStream,
		Group:            relayInfo.UsingGroup,
		Other:            other,
	})
	gopool.Go(func() {
		perfmetrics.RecordRelaySample(relayInfo, true, int64(summary.CompletionTokens))
	})
}
