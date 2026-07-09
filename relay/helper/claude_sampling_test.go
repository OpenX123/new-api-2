package helper

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func TestNormalizeClaudeSamplingForModel(t *testing.T) {
	t.Run("opus-4-8 strips sampling params", func(t *testing.T) {
		req := &dto.ClaudeRequest{
			Model:       "claude-opus-4-8",
			Temperature: common.GetPointer[float64](0.7),
			TopP:        common.GetPointer[float64](0.9),
			TopK:        common.GetPointer[int](40),
		}
		NormalizeClaudeSamplingForModel(req)
		if req.Temperature != nil || req.TopP != nil || req.TopK != nil {
			t.Fatalf("sampling params not stripped: %+v", req)
		}
	})

	t.Run("opus-4-8 converts enabled thinking to adaptive", func(t *testing.T) {
		req := &dto.ClaudeRequest{
			Model: "claude-opus-4-8",
			Thinking: &dto.Thinking{
				Type:         "enabled",
				BudgetTokens: common.GetPointer[int](2048),
			},
		}
		NormalizeClaudeSamplingForModel(req)
		if req.Thinking == nil || req.Thinking.Type != "adaptive" {
			t.Fatalf("thinking not converted to adaptive: %+v", req.Thinking)
		}
		if req.Thinking.BudgetTokens != nil {
			t.Fatalf("budget_tokens should be dropped")
		}
		if req.Thinking.Display != "summarized" {
			t.Fatalf("display should default to summarized, got %q", req.Thinking.Display)
		}
	})

	t.Run("adaptive thinking untouched", func(t *testing.T) {
		req := &dto.ClaudeRequest{
			Model:    "claude-opus-4-7",
			Thinking: &dto.Thinking{Type: "adaptive", Display: "omitted"},
		}
		NormalizeClaudeSamplingForModel(req)
		if req.Thinking.Type != "adaptive" || req.Thinking.Display != "omitted" {
			t.Fatalf("adaptive thinking should pass through: %+v", req.Thinking)
		}
	})

	t.Run("older models untouched", func(t *testing.T) {
		req := &dto.ClaudeRequest{
			Model:       "claude-opus-4-6",
			Temperature: common.GetPointer[float64](0.7),
			Thinking: &dto.Thinking{
				Type:         "enabled",
				BudgetTokens: common.GetPointer[int](2048),
			},
		}
		NormalizeClaudeSamplingForModel(req)
		if req.Temperature == nil || req.Thinking.Type != "enabled" {
			t.Fatalf("opus-4-6 request should not be modified: %+v", req)
		}
	})

	t.Run("nil request is a no-op", func(t *testing.T) {
		NormalizeClaudeSamplingForModel(nil)
	})

	t.Run("opus-4-6 with both temperature and top_p drops top_p", func(t *testing.T) {
		req := &dto.ClaudeRequest{
			Model:       "claude-opus-4-6",
			Temperature: common.GetPointer[float64](0.7),
			TopP:        common.GetPointer[float64](0.9),
		}
		NormalizeClaudeSamplingForModel(req)
		if req.Temperature == nil || *req.Temperature != 0.7 {
			t.Fatalf("temperature should be preserved, got %+v", req.Temperature)
		}
		if req.TopP != nil {
			t.Fatalf("top_p should be dropped when both are set, got %+v", req.TopP)
		}
	})

	t.Run("opus-4-6 with only temperature is untouched", func(t *testing.T) {
		req := &dto.ClaudeRequest{
			Model:       "claude-opus-4-6",
			Temperature: common.GetPointer[float64](0.7),
		}
		NormalizeClaudeSamplingForModel(req)
		if req.Temperature == nil || *req.Temperature != 0.7 || req.TopP != nil {
			t.Fatalf("only-temperature request should be unchanged: %+v", req)
		}
	})

	t.Run("opus-4-6 with only top_p is untouched", func(t *testing.T) {
		req := &dto.ClaudeRequest{
			Model: "claude-opus-4-6",
			TopP:  common.GetPointer[float64](0.9),
		}
		NormalizeClaudeSamplingForModel(req)
		if req.TopP == nil || *req.TopP != 0.9 || req.Temperature != nil {
			t.Fatalf("only-top_p request should be unchanged: %+v", req)
		}
	})

	t.Run("opus-4-8 with both still strips all (no regression)", func(t *testing.T) {
		req := &dto.ClaudeRequest{
			Model:       "claude-opus-4-8",
			Temperature: common.GetPointer[float64](0.7),
			TopP:        common.GetPointer[float64](0.9),
		}
		NormalizeClaudeSamplingForModel(req)
		if req.Temperature != nil || req.TopP != nil {
			t.Fatalf("4.8 should strip both temperature and top_p: %+v", req)
		}
	})

	t.Run("sonnet-5 strips sampling params", func(t *testing.T) {
		req := &dto.ClaudeRequest{
			Model:       "claude-sonnet-5",
			Temperature: common.GetPointer[float64](0.7),
			TopP:        common.GetPointer[float64](0.9),
			TopK:        common.GetPointer[int](40),
		}
		NormalizeClaudeSamplingForModel(req)
		if req.Temperature != nil || req.TopP != nil || req.TopK != nil {
			t.Fatalf("sonnet-5 sampling params not stripped: %+v", req)
		}
	})

	t.Run("mythos-5 strips sampling params", func(t *testing.T) {
		req := &dto.ClaudeRequest{
			Model:       "claude-mythos-5",
			Temperature: common.GetPointer[float64](0.7),
		}
		NormalizeClaudeSamplingForModel(req)
		if req.Temperature != nil {
			t.Fatalf("mythos-5 temperature not stripped: %+v", req)
		}
	})

	t.Run("sonnet-5 converts enabled thinking to adaptive", func(t *testing.T) {
		req := &dto.ClaudeRequest{
			Model: "claude-sonnet-5",
			Thinking: &dto.Thinking{
				Type:         "enabled",
				BudgetTokens: common.GetPointer[int](2048),
			},
		}
		NormalizeClaudeSamplingForModel(req)
		if req.Thinking == nil || req.Thinking.Type != "adaptive" {
			t.Fatalf("sonnet-5 thinking not converted to adaptive: %+v", req.Thinking)
		}
	})

	t.Run("generic claude model with both drops top_p", func(t *testing.T) {
		req := &dto.ClaudeRequest{
			Model:       "claude-3-5-sonnet",
			Temperature: common.GetPointer[float64](0.5),
			TopP:        common.GetPointer[float64](0.8),
		}
		NormalizeClaudeSamplingForModel(req)
		if req.Temperature == nil || *req.Temperature != 0.5 {
			t.Fatalf("temperature should be preserved for generic model, got %+v", req.Temperature)
		}
		if req.TopP != nil {
			t.Fatalf("top_p should be dropped for generic model when both set, got %+v", req.TopP)
		}
	})
}
