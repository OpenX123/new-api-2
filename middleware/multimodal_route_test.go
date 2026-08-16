package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInspectMultimodalRequest(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		model      string
		wantVision string
		wantImages int
		wantError  string
	}{
		{
			name:       "Claude user image and nested tool result",
			path:       "/v1/messages",
			model:      "MiniMax-M3",
			body:       `{"messages":[{"role":"user","content":[{"type":"image","source":{"type":"url","url":"https://example.com/a.png"}},{"type":"tool_result","tool_use_id":"x","content":[{"type":"image","source":{"type":"url","url":"https://example.com/b.png"}}]}]}]}`,
			wantVision: "MiniMax-M3",
			wantImages: 2,
		},
		{
			name:      "Claude assistant image is invalid",
			path:      "/v1/messages",
			model:     "MiniMax-M3",
			body:      `{"messages":[{"role":"assistant","content":[{"type":"image","source":{"type":"url","url":"https://example.com/a.png"}}]}]}`,
			wantError: "only supported in user messages",
		},
		{
			name:      "Claude system image is invalid",
			path:      "/v1/messages",
			model:     "MiniMax-M3",
			body:      `{"system":[{"type":"image","source":{"type":"url","url":"https://example.com/a.png"}}],"messages":[{"role":"user","content":"hi"}]}`,
			wantError: "system content",
		},
		{
			name:      "Chat image is explicitly unsupported",
			path:      "/v1/chat/completions",
			model:     "MiniMax-M3",
			body:      `{"messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"https://example.com/a.png"}}]}]}`,
			wantError: "unsupported on Chat Completions",
		},
		{
			name:      "Responses image is explicitly unsupported",
			path:      "/v1/responses",
			model:     "MiniMax-M3",
			body:      `{"input":[{"role":"user","content":[{"type":"input_image","image_url":"https://example.com/a.png"}]}]}`,
			wantError: "unsupported on Responses",
		},
		{
			name:      "Responses tool output image is explicitly unsupported",
			path:      "/v1/responses",
			model:     "MiniMax-M3",
			body:      `{"input":[{"type":"function_call_output","call_id":"call_1","output":[{"type":"input_image","image_url":"https://example.com/a.png"}]}]}`,
			wantError: "unsupported on Responses",
		},
		{
			name:  "configured text-only request",
			path:  "/v1/messages",
			model: "MiniMax-M3",
			body:  `{"messages":[{"role":"user","content":"hi"}]}`,
		},
		{
			name:  "unconfigured model bypasses inspection",
			path:  "/v1/messages",
			model: "",
			body:  `{not-json`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vision, images, err := InspectMultimodalRequest(test.path, []byte(test.body), test.model)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				assert.Empty(t, vision)
				assert.Zero(t, images)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantVision, vision)
			assert.Equal(t, test.wantImages, images)
		})
	}
}
