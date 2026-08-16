package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const claudeVisionTransformInput = `{
  "model":"deepseek-vision",
  "messages":[
    {
      "role":"user",
      "content":[
        {"type":"text","text":"before","cache_control":{"type":"ephemeral"}},
        {"type":"image","source":{"type":"url","media_type":"image/jpeg","url":"https://example.com/a.png"},"cache_control":{"type":"ephemeral"}},
        {"type":"tool_result","tool_use_id":"tool-1","is_error":true,"cache_control":{"type":"ephemeral"},"content":[
          {"type":"text","text":"tool-before"},
          {"type":"image","source":{"type":"base64","media_type":"image/png","data":"YWJj"},"cache_control":{"type":"ephemeral"}},
          {"type":"text","text":"tool-after"}
        ]},
        {"type":"text","text":"after"}
      ]
    },
    {"role":"assistant","content":[{"type":"text","text":"assistant"}]}
  ]
}`

func TestCollectClaudeVisionImages(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		want      []ClaudeVisionImage
		wantError string
	}{
		{
			name: "direct and tool result images keep deterministic order",
			body: claudeVisionTransformInput,
			want: []ClaudeVisionImage{
				{
					Source: dto.ClaudeMessageSource{
						Type:      "url",
						MediaType: "image/jpeg",
						Url:       "https://example.com/a.png",
					},
					Hash: "d304098965cd71d2e647378a1feeb35355b7b3fce6f1d8bc3ffab223a197655e",
				},
				{
					Source: dto.ClaudeMessageSource{
						Type:      "base64",
						MediaType: "image/png",
						Data:      "YWJj",
					},
					Hash: "cdab4469e47f1c8f91bf9327567a7a70c02ba2b5157da798409acc3a05816146",
				},
			},
		},
		{
			name: "string user content has no images",
			body: `{"model":"deepseek-vision","messages":[{"role":"user","content":"hello"}]}`,
			want: []ClaudeVisionImage{},
		},
		{
			name:      "image without source is rejected",
			body:      `{"model":"deepseek-vision","messages":[{"role":"user","content":[{"type":"image"}]}]}`,
			wantError: "source is missing",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &dto.ClaudeRequest{}
			require.NoError(t, common.Unmarshal([]byte(test.body), request))

			got, err := CollectClaudeVisionImages(request)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestReplaceClaudeVisionImages(t *testing.T) {
	const expected = `{
      "model":"deepseek-vision",
      "messages":[
        {
          "role":"user",
          "content":[
            {"type":"text","text":"before","cache_control":{"type":"ephemeral"}},
            {"type":"text","text":"url fact","cache_control":{"type":"ephemeral"}},
            {"type":"tool_result","tool_use_id":"tool-1","is_error":true,"cache_control":{"type":"ephemeral"},"content":[
              {"type":"text","text":"tool-before"},
              {"type":"text","text":"base64 fact","cache_control":{"type":"ephemeral"}},
              {"type":"text","text":"tool-after"}
            ]},
            {"type":"text","text":"after"}
          ]
        },
        {"role":"assistant","content":[{"type":"text","text":"assistant"}]}
      ]
    }`
	tests := []struct {
		name      string
		facts     map[string]string
		want      string
		wantError string
	}{
		{
			name: "replace at original positions and preserve tool metadata",
			facts: map[string]string{
				"d304098965cd71d2e647378a1feeb35355b7b3fce6f1d8bc3ffab223a197655e": "url fact",
				"cdab4469e47f1c8f91bf9327567a7a70c02ba2b5157da798409acc3a05816146": "base64 fact",
			},
			want: expected,
		},
		{
			name: "missing fact rejects the whole transform",
			facts: map[string]string{
				"d304098965cd71d2e647378a1feeb35355b7b3fce6f1d8bc3ffab223a197655e": "url fact",
			},
			wantError: "missing vision fact",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &dto.ClaudeRequest{}
			require.NoError(t, common.Unmarshal([]byte(claudeVisionTransformInput), request))

			err := ReplaceClaudeVisionImages(request, test.facts)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				originalImages, collectErr := CollectClaudeVisionImages(request)
				require.NoError(t, collectErr)
				require.Len(t, originalImages, 2)
				return
			}
			require.NoError(t, err)

			actualJSON, err := common.Marshal(request)
			require.NoError(t, err)
			var actual, want any
			require.NoError(t, common.Unmarshal(actualJSON, &actual))
			require.NoError(t, common.Unmarshal([]byte(test.want), &want))
			assert.Equal(t, want, actual)

			originalImages, collectErr := CollectClaudeVisionImages(request)
			require.NoError(t, collectErr)
			require.Empty(t, originalImages)
		})
	}
}
