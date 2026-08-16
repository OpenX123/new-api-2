package middleware

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// InspectMultimodalRequest reports images that need the configured vision
// preprocessor. It never changes the requested model or selects a channel.
//
// Only the Claude Messages shape is supported by the vision preprocessor.
// Configured aliases using Chat Completions or Responses fail explicitly so an
// image is never forwarded to a text-only model by accident.
func InspectMultimodalRequest(path string, body []byte, visionModel string) (string, int, error) {
	visionModel = strings.TrimSpace(visionModel)
	if visionModel == "" {
		return "", 0, nil
	}

	var root map[string]any
	if err := common.Unmarshal(body, &root); err != nil {
		return "", 0, err
	}

	switch {
	case strings.HasSuffix(path, "/messages"):
		imageCount := 0
		if countClaudeImages(root["system"]) > 0 {
			return "", 0, fmt.Errorf("images in Claude system content are unsupported")
		}
		messages, _ := root["messages"].([]any)
		for _, rawMessage := range messages {
			message, ok := rawMessage.(map[string]any)
			if !ok {
				continue
			}
			count := countClaudeImages(message["content"])
			if count == 0 {
				continue
			}
			if strings.TrimSpace(common.Interface2String(message["role"])) != "user" {
				return "", 0, fmt.Errorf("Claude images are only supported in user messages")
			}
			imageCount += count
		}
		if imageCount == 0 {
			return "", 0, nil
		}
		return visionModel, imageCount, nil

	case strings.HasSuffix(path, "/chat/completions"):
		if countMessageImages(root["messages"], countChatImages) > 0 {
			return "", 0, fmt.Errorf("image input for configured multimodal aliases is unsupported on Chat Completions; use Claude Messages")
		}

	case strings.HasSuffix(path, "/responses"):
		if countResponsesImages(root["input"]) > 0 {
			return "", 0, fmt.Errorf("image input for configured multimodal aliases is unsupported on Responses; use Claude Messages")
		}
	}

	return "", 0, nil
}

func countMessageImages(value any, countContent func(any) int) int {
	messages, ok := value.([]any)
	if !ok {
		return 0
	}
	count := 0
	for _, rawMessage := range messages {
		message, ok := rawMessage.(map[string]any)
		if ok {
			count += countContent(message["content"])
		}
	}
	return count
}

func countClaudeImages(value any) int {
	parts, ok := value.([]any)
	if !ok {
		return 0
	}
	count := 0
	for _, rawPart := range parts {
		part, ok := rawPart.(map[string]any)
		if !ok {
			continue
		}
		switch strings.TrimSpace(common.Interface2String(part["type"])) {
		case "image":
			count++
		case "tool_result":
			count += countClaudeImages(part["content"])
		}
	}
	return count
}

func countChatImages(value any) int {
	parts, ok := value.([]any)
	if !ok {
		return 0
	}
	count := 0
	for _, rawPart := range parts {
		part, ok := rawPart.(map[string]any)
		if ok && strings.TrimSpace(common.Interface2String(part["type"])) == "image_url" {
			count++
		}
	}
	return count
}

func countResponsesImages(value any) int {
	switch typed := value.(type) {
	case []any:
		count := 0
		for _, item := range typed {
			count += countResponsesImages(item)
		}
		return count
	case map[string]any:
		count := 0
		if strings.TrimSpace(common.Interface2String(typed["type"])) == "input_image" {
			count++
		}
		count += countResponsesImages(typed["content"])
		count += countResponsesImages(typed["output"])
		return count
	}
	return 0
}
