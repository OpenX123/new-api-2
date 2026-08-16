package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureClaudeRequestHasNoImages(t *testing.T) {
	require.NoError(t, EnsureClaudeRequestHasNoImages([]byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"fact"}]}]}`)))
	require.Error(t, EnsureClaudeRequestHasNoImages([]byte(`{"messages":[{"role":"user","content":[{"type":"tool_result","content":[{"type":"image"}]}]}]}`)))
}
