package common

import (
	"errors"
	"strings"

	appcommon "github.com/QuantumNous/new-api/common"
)

// EnsureClaudeRequestHasNoImages prevents channel overrides from restoring an
// image after the vision bridge has converted the request for a text model.
func EnsureClaudeRequestHasNoImages(data []byte) error {
	var request map[string]any
	if err := appcommon.Unmarshal(data, &request); err != nil {
		return err
	}
	if containsClaudeImage(request["system"]) || containsClaudeImage(request["messages"]) {
		return errors.New("vision preprocessing left image content in the text-model request")
	}
	return nil
}

func containsClaudeImage(value any) bool {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if containsClaudeImage(item) {
				return true
			}
		}
	case map[string]any:
		if strings.TrimSpace(appcommon.Interface2String(typed["type"])) == "image" {
			return true
		}
		for _, field := range []string{"content", "messages"} {
			if containsClaudeImage(typed[field]) {
				return true
			}
		}
	}
	return false
}
