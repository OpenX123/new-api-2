package helper

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

var (
	ErrVisionTTFTTimeout         = errors.New("vision-augmented request did not produce content before the TTFT deadline")
	ErrVisionTTFTPreludeOverflow = errors.New("vision-augmented TTFT prelude exceeded 64 KiB")
	ErrVisionTTFTNoContent       = errors.New("vision-augmented upstream stream ended before producing content")
)

const maxVisionTTFTPreludeBytes = 64 << 10

const VisionEndToEndTTFTTimeout = 60 * time.Second

func VisionTTFTError(err error) *types.NewAPIError {
	if !IsVisionTTFTError(err) {
		return nil
	}
	return types.NewOpenAIError(err, types.ErrorCodeChannelResponseTimeExceeded, http.StatusGatewayTimeout, types.ErrOptionWithLocalDeadline())
}

func IsVisionTTFTError(err error) bool {
	return errors.Is(err, ErrVisionTTFTTimeout) || errors.Is(err, ErrVisionTTFTPreludeOverflow) || errors.Is(err, ErrVisionTTFTNoContent)
}

type visionTTFTGateBody struct {
	reader  *bufio.Reader
	body    io.ReadCloser
	buffer  bytes.Buffer
	replay  *bytes.Reader
	cancel  context.CancelFunc
	cause   func() error
	release func() bool
	opened  bool
	mu      sync.Mutex
}

func NewVisionTTFTGateBody(body io.ReadCloser, cancel context.CancelFunc, cause func() error, release func() bool) io.ReadCloser {
	return &visionTTFTGateBody{
		reader:  bufio.NewReader(body),
		body:    body,
		cancel:  cancel,
		cause:   cause,
		release: release,
	}
}

func (b *visionTTFTGateBody) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.replay != nil {
		n, err := b.replay.Read(p)
		if err != io.EOF || n > 0 {
			return n, err
		}
		b.replay = nil
	}
	for !b.opened {
		line, err := b.reader.ReadString('\n')
		if line != "" {
			meaningful := meaningfulSSELine(line)
			if !meaningful && b.buffer.Len()+len(line) > maxVisionTTFTPreludeBytes {
				return 0, ErrVisionTTFTPreludeOverflow
			}
			b.buffer.WriteString(line)
			if meaningful {
				if b.release != nil && !b.release() {
					if b.cause != nil && errors.Is(b.cause(), ErrVisionTTFTTimeout) {
						return 0, ErrVisionTTFTTimeout
					}
					return 0, context.Canceled
				}
				b.opened = true
				b.replay = bytes.NewReader(b.buffer.Bytes())
				return b.replay.Read(p)
			}
		}
		if err != nil {
			if b.cause != nil && errors.Is(b.cause(), ErrVisionTTFTTimeout) {
				return 0, ErrVisionTTFTTimeout
			}
			if errors.Is(err, io.EOF) {
				return 0, ErrVisionTTFTNoContent
			}
			return 0, err
		}
	}
	return b.reader.Read(p)
}

func (b *visionTTFTGateBody) Close() error {
	if b.cancel != nil {
		b.cancel()
	}
	return b.body.Close()
}

func meaningfulSSELine(line string) bool {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "data:") {
		return false
	}
	data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if data == "" || data == "[DONE]" {
		return false
	}
	var value any
	if common.Unmarshal([]byte(data), &value) != nil {
		return false
	}
	return containsMeaningfulDelta(value)
}

func containsMeaningfulDelta(value any) bool {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if containsMeaningfulDelta(item) {
				return true
			}
		}
	case map[string]any:
		if _, ok := typed["error"]; ok {
			return true
		}
		eventType := strings.TrimSpace(common.Interface2String(typed["type"]))
		switch eventType {
		case "response.output_text.delta", "response.function_call_arguments.delta":
			if strings.TrimSpace(common.Interface2String(typed["delta"])) != "" {
				return true
			}
		case "response.output_item.added":
			if item, ok := typed["item"].(map[string]any); ok {
				switch strings.TrimSpace(common.Interface2String(item["type"])) {
				case "function_call", "custom_tool_call", "computer_call":
					return true
				}
			}
		}
		if delta, ok := typed["delta"].(map[string]any); ok {
			for _, key := range []string{"content", "text", "thinking", "refusal", "partial_json"} {
				if strings.TrimSpace(common.Interface2String(delta[key])) != "" {
					return true
				}
			}
			if toolCalls, ok := delta["tool_calls"].([]any); ok && len(toolCalls) > 0 {
				return true
			}
		}
		if block, ok := typed["content_block"].(map[string]any); ok && common.Interface2String(block["type"]) == "tool_use" {
			return true
		}
		for key, child := range typed {
			if key != "usage" && key != "metadata" && containsMeaningfulDelta(child) {
				return true
			}
		}
	}
	return false
}
