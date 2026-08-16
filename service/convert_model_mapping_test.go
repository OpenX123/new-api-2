package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestOpenAIToClaudeResponseKeepsMappedPublicModel(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName:   "claude-opus-5",
		SendResponseCount: 1,
		ChannelMeta: &relaycommon.ChannelMeta{
			IsModelMapped: true,
		},
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{},
	}

	stream := StreamResponseOpenAI2Claude(&dto.ChatCompletionsStreamResponse{Model: "deepseek-v4-flash"}, info)
	require.NotEmpty(t, stream)
	require.Equal(t, "claude-opus-5", stream[0].Message.Model)

	nonStream := ResponseOpenAI2Claude(&dto.OpenAITextResponse{Model: "deepseek-v4-flash"}, info)
	require.Equal(t, "claude-opus-5", nonStream.Model)
}
