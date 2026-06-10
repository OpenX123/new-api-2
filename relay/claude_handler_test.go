package relay

import (
	"encoding/json"
	"testing"
)

func TestRemoveClearThinkingEdits(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // "" 表示期望整个 context_management 被去掉
	}{
		{
			name: "only clear_thinking edit drops whole context_management",
			in:   `{"edits":[{"type":"clear_thinking_20251015"}]}`,
			want: "",
		},
		{
			name: "mixed edits keeps others",
			in:   `{"edits":[{"type":"clear_thinking_20251015"},{"type":"clear_tool_uses_20250919"}]}`,
			want: `{"edits":[{"type":"clear_tool_uses_20250919"}]}`,
		},
		{
			name: "no clear_thinking passes through unchanged",
			in:   `{"edits":[{"type":"clear_tool_uses_20250919"}]}`,
			want: `{"edits":[{"type":"clear_tool_uses_20250919"}]}`,
		},
		{
			name: "edits empty after filter but sibling fields kept",
			in:   `{"edits":[{"type":"clear_thinking_20251015","keep":{"type":"thinking_turns","value":2}}],"foo":1}`,
			want: `{"foo":1}`,
		},
		{
			name: "invalid json passes through",
			in:   `not-json`,
			want: `not-json`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := removeClearThinkingEdits(json.RawMessage(tc.in))
			if tc.want == "" {
				if got != nil {
					t.Fatalf("expected nil, got %s", got)
				}
				return
			}
			if !jsonEqualOrRawEqual(t, got, tc.want) {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}

	if got := removeClearThinkingEdits(nil); got != nil {
		t.Fatalf("nil input should return nil, got %s", got)
	}
}

// jsonEqualOrRawEqual 比较两段 JSON 语义相等;非法 JSON 退化为字符串比较。
func jsonEqualOrRawEqual(t *testing.T, got json.RawMessage, want string) bool {
	t.Helper()
	var g, w any
	if json.Unmarshal(got, &g) != nil || json.Unmarshal([]byte(want), &w) != nil {
		return string(got) == want
	}
	gb, _ := json.Marshal(g)
	wb, _ := json.Marshal(w)
	return string(gb) == string(wb)
}
