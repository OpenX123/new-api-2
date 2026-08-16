package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

type ClaudeVisionImage struct {
	Source dto.ClaudeMessageSource
	Hash   string
}

// CollectClaudeVisionImages returns user images in message/block order,
// including images nested in tool_result content.
func CollectClaudeVisionImages(request *dto.ClaudeRequest) ([]ClaudeVisionImage, error) {
	if request == nil {
		return nil, fmt.Errorf("Claude request is nil")
	}
	images := make([]ClaudeVisionImage, 0)
	for _, message := range request.Messages {
		if strings.TrimSpace(message.Role) != "user" || message.IsStringContent() {
			continue
		}
		content, err := message.ParseContent()
		if err != nil {
			return nil, err
		}
		if err := collectClaudeVisionContent(content, &images); err != nil {
			return nil, err
		}
	}
	return images, nil
}

// ReplaceClaudeVisionImages replaces every user image with the text fact
// keyed by its source hash. The request is changed only after all facts pass.
func ReplaceClaudeVisionImages(request *dto.ClaudeRequest, factsByHash map[string]string) error {
	if request == nil {
		return fmt.Errorf("Claude request is nil")
	}
	data, err := common.Marshal(request)
	if err != nil {
		return err
	}
	cloned := &dto.ClaudeRequest{}
	if err := common.Unmarshal(data, cloned); err != nil {
		return err
	}

	for i := range cloned.Messages {
		message := &cloned.Messages[i]
		if strings.TrimSpace(message.Role) != "user" || message.IsStringContent() {
			continue
		}
		content, err := message.ParseContent()
		if err != nil {
			return err
		}
		replaced, err := replaceClaudeVisionContent(content, factsByHash)
		if err != nil {
			return err
		}
		message.Content = replaced
	}
	*request = *cloned
	return nil
}

func collectClaudeVisionContent(content []dto.ClaudeMediaMessage, images *[]ClaudeVisionImage) error {
	for _, block := range content {
		switch strings.TrimSpace(block.Type) {
		case "image":
			image, err := claudeVisionImage(block.Source)
			if err != nil {
				return err
			}
			*images = append(*images, image)
		case "tool_result":
			nested, ok, err := claudeToolResultContent(block.Content)
			if err != nil {
				return err
			}
			if ok {
				if err := collectClaudeVisionContent(nested, images); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func replaceClaudeVisionContent(content []dto.ClaudeMediaMessage, factsByHash map[string]string) ([]dto.ClaudeMediaMessage, error) {
	replaced := make([]dto.ClaudeMediaMessage, len(content))
	copy(replaced, content)
	for i := range replaced {
		block := &replaced[i]
		switch strings.TrimSpace(block.Type) {
		case "image":
			image, err := claudeVisionImage(block.Source)
			if err != nil {
				return nil, err
			}
			fact, ok := factsByHash[image.Hash]
			if !ok || strings.TrimSpace(fact) == "" {
				return nil, fmt.Errorf("missing vision fact for source hash %s", image.Hash)
			}
			cacheControl := block.CacheControl
			*block = dto.ClaudeMediaMessage{Type: "text", Text: common.GetPointer(fact), CacheControl: cacheControl}
		case "tool_result":
			nested, ok, err := claudeToolResultContent(block.Content)
			if err != nil {
				return nil, err
			}
			if ok {
				block.Content, err = replaceClaudeVisionContent(nested, factsByHash)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return replaced, nil
}

func claudeToolResultContent(content any) ([]dto.ClaudeMediaMessage, bool, error) {
	switch content.(type) {
	case nil, string:
		return nil, false, nil
	}
	nested, err := common.Any2Type[[]dto.ClaudeMediaMessage](content)
	if err != nil {
		return nil, false, err
	}
	return nested, true, nil
}

func claudeVisionImage(source *dto.ClaudeMessageSource) (ClaudeVisionImage, error) {
	if source == nil {
		return ClaudeVisionImage{}, fmt.Errorf("Claude image source is missing")
	}
	rawSource := source.Url
	if rawSource == "" {
		switch data := source.Data.(type) {
		case string:
			rawSource = data
		case nil:
		default:
			encoded, err := common.Marshal(data)
			if err != nil {
				return ClaudeVisionImage{}, err
			}
			rawSource = string(encoded)
		}
	}
	if rawSource == "" {
		return ClaudeVisionImage{}, fmt.Errorf("Claude image source is empty")
	}
	hash := sha256.Sum256([]byte(source.Type + "\x00" + source.MediaType + "\x00" + rawSource))
	return ClaudeVisionImage{
		Source: *source,
		Hash:   hex.EncodeToString(hash[:]),
	}, nil
}
