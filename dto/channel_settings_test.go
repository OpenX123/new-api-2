package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvancedCustomValidateResponsesToChatConverterPath(t *testing.T) {
	valid := &AdvancedCustomConfig{
		Routes: []AdvancedCustomRoute{
			{
				IncomingPath: "/v1/responses",
				UpstreamPath: "/v1/chat/completions",
				Converter:    AdvancedCustomConverterOpenAIResponsesToOpenAIChatCompletions,
			},
		},
	}
	require.NoError(t, valid.Validate())

	tests := []struct {
		name         string
		incomingPath string
	}{
		{name: "chat completions", incomingPath: "/v1/chat/completions"},
		{name: "responses compact", incomingPath: "/v1/responses/compact"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AdvancedCustomConfig{
				Routes: []AdvancedCustomRoute{
					{
						IncomingPath: tt.incomingPath,
						UpstreamPath: "/v1/chat/completions",
						Converter:    AdvancedCustomConverterOpenAIResponsesToOpenAIChatCompletions,
					},
				},
			}
			err := config.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "converter does not match incoming_path")
		})
	}
}

func TestVisionBridgeConfig(t *testing.T) {
	config := &VisionBridgeConfig{
		ModelMap:           map[string]string{"claude-opus-5": "vision-alias"},
		ChannelId:          2,
		FallbackChannelIds: []int{3},
		TTFTTimeoutMs:      10000,
		AttemptTimeoutMs:   6000,
		ServiceTier:        "priority",
	}
	require.NoError(t, config.Validate())
	assert.Equal(t, []int{2, 3}, config.ChannelIds())
	assert.Equal(t, "priority", config.ServiceTier)
	assert.Equal(t, 10000, config.TimeoutMs())
	model, ok := config.Resolve(" claude-opus-5 ")
	assert.True(t, ok)
	assert.Equal(t, "vision-alias", model)

	missingFailoverTimeouts := &VisionBridgeConfig{
		ModelMap:           map[string]string{"claude-opus-5": "vision-alias"},
		ChannelId:          2,
		FallbackChannelIds: []int{3},
	}
	require.ErrorContains(t, missingFailoverTimeouts.Validate(), "attempt_timeout_ms")

	config.ChannelId = 0
	require.ErrorContains(t, config.Validate(), "channel_id")

	config.ChannelId = 2
	config.FallbackChannelIds = []int{3, 4}
	require.ErrorContains(t, config.Validate(), "fallback_channel_ids")

	config.FallbackChannelIds = []int{3, 2}
	require.ErrorContains(t, config.Validate(), "fallback_channel_ids")

	config.FallbackChannelIds = []int{3}
	config.AttemptTimeoutMs = config.TTFTTimeoutMs
	require.ErrorContains(t, config.Validate(), "attempt_timeout_ms")
}
