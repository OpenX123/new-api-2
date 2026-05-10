package service

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newCtxWithBody(method, path, contentType string, body []byte) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(method, path, io.NopCloser(bytes.NewReader(body)))
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = int64(len(body))
	c.Request = req
	return c
}

func TestDetectVisionInRequest_OpenAIChat(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "text only",
			body: `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`,
			want: false,
		},
		{
			name: "image_url present",
			body: `{"model":"gpt-4o","messages":[{"role":"user","content":[{"type":"text","text":"what is this"},{"type":"image_url","image_url":{"url":"https://example.com/a.png"}}]}]}`,
			want: true,
		},
		{
			name: "audio only - not detected as vision",
			body: `{"model":"gpt-4o","messages":[{"role":"user","content":[{"type":"input_audio","input_audio":{"data":"xx","format":"wav"}}]}]}`,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCtxWithBody("POST", "/v1/chat/completions", "application/json", []byte(tt.body))
			got := DetectVisionInRequest(c, c.Request.URL.Path)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectVisionInRequest_Claude(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "string content",
			body: `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"hi"}]}`,
			want: false,
		},
		{
			name: "image in content array",
			body: `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":[{"type":"text","text":"see this"},{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBOR"}}]}]}`,
			want: true,
		},
		{
			name: "image in system",
			body: `{"model":"claude-3-5-sonnet","system":[{"type":"image","source":{"type":"url","url":"https://x/y.png","media_type":"image/png"}}],"messages":[{"role":"user","content":"hi"}]}`,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCtxWithBody("POST", "/v1/messages", "application/json", []byte(tt.body))
			got := DetectVisionInRequest(c, c.Request.URL.Path)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectVisionInRequest_Gemini(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "text only",
			body: `{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`,
			want: false,
		},
		{
			name: "inlineData image",
			body: `{"contents":[{"role":"user","parts":[{"inlineData":{"mimeType":"image/jpeg","data":"AAAA"}}]}]}`,
			want: true,
		},
		{
			name: "inline_data snake_case image",
			body: `{"contents":[{"role":"user","parts":[{"inline_data":{"mimeType":"image/png","data":"AAAA"}}]}]}`,
			want: true,
		},
		{
			name: "fileData with image mime",
			body: `{"contents":[{"role":"user","parts":[{"fileData":{"mimeType":"image/webp","fileUri":"gs://x/y"}}]}]}`,
			want: true,
		},
		{
			name: "fileData with non-image mime",
			body: `{"contents":[{"role":"user","parts":[{"fileData":{"mimeType":"application/pdf","fileUri":"gs://x/y"}}]}]}`,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCtxWithBody("POST", "/v1beta/models/gemini-2.0-flash:generateContent", "application/json", []byte(tt.body))
			got := DetectVisionInRequest(c, c.Request.URL.Path)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectVisionInRequest_Responses(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "string input",
			body: `{"model":"gpt-5","input":"hello"}`,
			want: false,
		},
		{
			name: "input_image present",
			body: `{"model":"gpt-5","input":[{"role":"user","content":[{"type":"input_text","text":"see"},{"type":"input_image","image_url":"https://x/y.png"}]}]}`,
			want: true,
		},
		{
			name: "input_file only - not vision",
			body: `{"model":"gpt-5","input":[{"role":"user","content":[{"type":"input_file","file_url":"https://x/y.pdf"}]}]}`,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCtxWithBody("POST", "/v1/responses", "application/json", []byte(tt.body))
			got := DetectVisionInRequest(c, c.Request.URL.Path)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectVisionInRequest_UnknownPath(t *testing.T) {
	c := newCtxWithBody("POST", "/v1/embeddings", "application/json", []byte(`{"model":"text-embedding-3-small","input":"hi"}`))
	if got := DetectVisionInRequest(c, c.Request.URL.Path); got != false {
		t.Errorf("unknown path should return false, got %v", got)
	}
}

func TestDetectVisionInRequest_BodyUnchangedAfterPeek(t *testing.T) {
	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"https://x.png"}}]}]}`)
	c := newCtxWithBody("POST", "/v1/chat/completions", "application/json", body)
	if !DetectVisionInRequest(c, c.Request.URL.Path) {
		t.Fatal("expected vision detected")
	}
	// Re-read body — should still return original bytes
	got, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("re-read body failed: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("body mismatch after detection.\nwant: %s\ngot:  %s", body, got)
	}
}
