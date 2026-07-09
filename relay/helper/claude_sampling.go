package helper

import (
	"strings"

	"github.com/QuantumNous/new-api/dto"
)

// claudeModelRejectsSampling 判断模型是否已移除采样参数。
// Opus 4.7/4.8 及 Claude 5 家族(Fable/Sonnet/Mythos)起
// temperature/top_p/top_k 一律返回 400
// ("`temperature` is deprecated for this model"),
// 且 thinking.type="enabled"(budget_tokens) 也已移除,仅支持 adaptive。
func claudeModelRejectsSampling(model string) bool {
	return strings.HasPrefix(model, "claude-opus-4-7") ||
		strings.HasPrefix(model, "claude-opus-4-8") ||
		strings.HasPrefix(model, "claude-fable") ||
		strings.HasPrefix(model, "claude-sonnet-5") ||
		strings.HasPrefix(model, "claude-mythos")
}

// NormalizeClaudeSamplingForModel 按上游模型的实际限制清理请求参数:
// 对已移除采样参数的模型剥掉 temperature/top_p/top_k,
// 并把 enabled(budget_tokens) thinking 转为 adaptive,避免 Anthropic 400。
// 对其他模型仅做 temperature/top_p 互斥处理。
func NormalizeClaudeSamplingForModel(req *dto.ClaudeRequest) {
	if req == nil {
		return
	}
	if claudeModelRejectsSampling(req.Model) {
		req.Temperature = nil
		req.TopP = nil
		req.TopK = nil
		if req.Thinking != nil && req.Thinking.Type == "enabled" {
			display := req.Thinking.Display
			if display == "" {
				// enabled thinking 在旧模型上默认可见,转 adaptive 后保持可见摘要
				display = "summarized"
			}
			req.Thinking = &dto.Thinking{Type: "adaptive", Display: display}
		}
		return
	}
	// Claude 不允许 temperature 与 top_p 同时指定,否则返回
	// "`temperature` and `top_p` cannot both be specified for this model"。
	// 二者都非空时保留 temperature、剥掉 top_p。
	if req.Temperature != nil && req.TopP != nil {
		req.TopP = nil
	}
}
