package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"

	"github.com/gin-gonic/gin"
)

// DetectVisionInRequest 检查请求体中是否包含图像类多模态内容。
//
// 检测覆盖以下入口格式：
//   - OpenAI Chat Completions (/v1/chat/completions, /pg/chat/completions)
//   - Claude Messages (/v1/messages)
//   - Gemini generateContent (/v1beta/models/...:..., /v1/models/...:...)
//   - OpenAI Responses API (/v1/responses)
//
// 仅检测「图像」类内容（image_url / image / inline_data 中的 image MIME / input_image），
// 不检测 audio/video/file。
//
// 返回 false 不会阻断请求 —— 调用方据此决定是否在路由层加上 vision 渠道过滤。
func DetectVisionInRequest(c *gin.Context, path string) bool {
	if c == nil || c.Request == nil {
		return false
	}
	switch {
	case strings.Contains(path, "/v1/chat/completions"),
		strings.Contains(path, "/pg/chat/completions"):
		return detectVisionOpenAIChat(c)
	case strings.Contains(path, "/v1/messages"):
		return detectVisionClaude(c)
	case strings.HasPrefix(path, "/v1beta/models/"),
		strings.HasPrefix(path, "/v1/models/"):
		return detectVisionGemini(c)
	case strings.HasPrefix(path, "/v1/responses"):
		return detectVisionResponses(c)
	}
	return false
}

func detectVisionOpenAIChat(c *gin.Context) bool {
	var req dto.GeneralOpenAIRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return false
	}
	for i := range req.Messages {
		for _, media := range req.Messages[i].ParseContent() {
			if media.Type == dto.ContentTypeImageURL {
				return true
			}
		}
	}
	return false
}

func detectVisionClaude(c *gin.Context) bool {
	var req dto.ClaudeRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return false
	}
	if req.System != nil && !req.IsStringSystem() {
		for _, media := range req.ParseSystem() {
			if media.Type == "image" {
				return true
			}
		}
	}
	for i := range req.Messages {
		if req.Messages[i].IsStringContent() {
			continue
		}
		mediaList, err := req.Messages[i].ParseContent()
		if err != nil {
			continue
		}
		for _, media := range mediaList {
			if media.Type == "image" {
				return true
			}
		}
	}
	return false
}

func detectVisionGemini(c *gin.Context) bool {
	var req dto.GeminiChatRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return false
	}
	for _, content := range req.Contents {
		for _, part := range content.Parts {
			if part.InlineData != nil && strings.HasPrefix(part.InlineData.MimeType, "image/") {
				return true
			}
			if part.FileData != nil && strings.HasPrefix(part.FileData.MimeType, "image/") {
				return true
			}
		}
	}
	if req.SystemInstructions != nil {
		for _, part := range req.SystemInstructions.Parts {
			if part.InlineData != nil && strings.HasPrefix(part.InlineData.MimeType, "image/") {
				return true
			}
			if part.FileData != nil && strings.HasPrefix(part.FileData.MimeType, "image/") {
				return true
			}
		}
	}
	return false
}

func detectVisionResponses(c *gin.Context) bool {
	var req dto.OpenAIResponsesRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return false
	}
	for _, input := range req.ParseInput() {
		if input.Type == "input_image" {
			return true
		}
	}
	return false
}
