package helper

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVisionTTFTGateReplaysPreludeAfterContent(t *testing.T) {
	stream := "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n" +
		"data: [DONE]\n\n"
	released := false
	body := NewVisionTTFTGateBody(io.NopCloser(strings.NewReader(stream)), func() {}, func() error { return nil }, func() bool { released = true; return true })

	got, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, stream, string(got))
	assert.True(t, released)
}

func TestVisionTTFTGateReturnsDeadlineBeforeContent(t *testing.T) {
	body := NewVisionTTFTGateBody(
		io.NopCloser(strings.NewReader("data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n")),
		func() {},
		func() error { return ErrVisionTTFTTimeout },
		func() bool { return true },
	)

	_, err := io.ReadAll(body)
	assert.ErrorIs(t, err, ErrVisionTTFTTimeout)
	assert.NotErrorIs(t, err, context.Canceled)
}

func TestVisionTTFTGateRejectsContentWhenDeadlineAlreadyWon(t *testing.T) {
	body := NewVisionTTFTGateBody(
		io.NopCloser(strings.NewReader("data: {\"choices\":[{\"delta\":{\"content\":\"too late\"}}]}\n\n")),
		func() {},
		func() error { return ErrVisionTTFTTimeout },
		func() bool { return false },
	)

	got, err := io.ReadAll(body)
	assert.Empty(t, got)
	assert.ErrorIs(t, err, ErrVisionTTFTTimeout)
}

func TestVisionTTFTGateRejectsStreamEndingBeforeContent(t *testing.T) {
	body := NewVisionTTFTGateBody(
		io.NopCloser(strings.NewReader("data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\ndata: [DONE]\n\n")),
		func() {},
		func() error { return nil },
		func() bool { return true },
	)

	got, err := io.ReadAll(body)
	assert.Empty(t, got)
	assert.ErrorIs(t, err, ErrVisionTTFTNoContent)
}

func TestMeaningfulSSELineRecognizesVisibleProtocolDeltas(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{name: "Chat role prelude", line: `data: {"choices":[{"delta":{"role":"assistant"}}]}`, want: false},
		{name: "Chat text", line: `data: {"choices":[{"delta":{"content":"hello"}}]}`, want: true},
		{name: "Chat refusal text", line: `data: {"choices":[{"delta":{"refusal":"cannot comply"}}]}`, want: true},
		{name: "Chat tool call", line: `data: {"choices":[{"delta":{"tool_calls":[{"index":0}]}}]}`, want: true},
		{name: "Claude message start", line: `data: {"type":"message_start","message":{"content":[]}}`, want: false},
		{name: "Claude thinking", line: `data: {"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"working"}}`, want: true},
		{name: "Claude text", line: `data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"hello"}}`, want: true},
		{name: "Claude tool use", line: `data: {"type":"content_block_start","content_block":{"type":"tool_use","id":"tool_1"}}`, want: true},
		{name: "Responses created", line: `data: {"type":"response.created","response":{"output":[]}}`, want: false},
		{name: "Responses text", line: `data: {"type":"response.output_text.delta","delta":"hello"}`, want: true},
		{name: "Responses function call", line: `data: {"type":"response.output_item.added","item":{"type":"function_call","call_id":"call_1"}}`, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, meaningfulSSELine(test.line))
		})
	}
}

func TestVisionTTFTGateAllowsLargeFirstMeaningfulDelta(t *testing.T) {
	thinking := strings.Repeat("x", maxVisionTTFTPreludeBytes+1)
	stream := `data: {"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"` + thinking + `"}}` + "\n\n"
	released := false
	body := NewVisionTTFTGateBody(io.NopCloser(strings.NewReader(stream)), func() {}, func() error { return nil }, func() bool { released = true; return true })

	got, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, stream, string(got))
	assert.True(t, released)
}

func TestVisionTTFTErrorIsGatewayTimeout(t *testing.T) {
	for _, err := range []error{ErrVisionTTFTTimeout, ErrVisionTTFTPreludeOverflow, ErrVisionTTFTNoContent} {
		apiErr := VisionTTFTError(err)
		require.NotNil(t, apiErr)
		assert.Equal(t, 504, apiErr.StatusCode)
		assert.False(t, types.IsSkipRetryError(apiErr))
	}
	assert.Nil(t, VisionTTFTError(io.EOF))
}
