package router

import (
	"bytes"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

const defaultIndexFixture = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/png" href="/logo.png" />
    <title>New API</title>
    <meta name="title" content="New API" />
  </head>
  <body><div id="root"></div></body>
</html>`

const classicIndexFixture = `<!doctype html>
<html lang="zh">
  <head>
    <link rel="icon" href="/logo.png" />
    <title>New API</title>
  </head>
  <body><div id="root"></div></body>
</html>`

func resetInjector() {
	injector = &indexInjector{}
}

func withSystemBranding(t *testing.T, name, logo string, fn func()) {
	t.Helper()
	prevName, prevLogo := common.SystemName, common.Logo
	common.SystemName, common.Logo = name, logo
	t.Cleanup(func() {
		common.SystemName, common.Logo = prevName, prevLogo
	})
	fn()
}

func TestRenderIndex_DefaultBranding_NoOp(t *testing.T) {
	resetInjector()
	withSystemBranding(t, "New API", "", func() {
		out := injector.renderIndex("default", []byte(defaultIndexFixture))
		if string(out) != defaultIndexFixture {
			t.Fatalf("expected unchanged bytes when SystemName is default; got:\n%s", out)
		}
	})
}

func TestRenderIndex_CustomSystemName_ReplacesTitleAndMeta(t *testing.T) {
	resetInjector()
	withSystemBranding(t, "云链API", "", func() {
		out := injector.renderIndex("default", []byte(defaultIndexFixture))
		s := string(out)
		if !strings.Contains(s, "<title>云链API</title>") {
			t.Errorf("title not replaced; got:\n%s", s)
		}
		if !strings.Contains(s, `<meta name="title" content="云链API" />`) {
			t.Errorf("meta title not replaced; got:\n%s", s)
		}
		if strings.Contains(s, "New API") {
			t.Errorf("stale 'New API' literal remains; got:\n%s", s)
		}
	})
}

func TestRenderIndex_HTMLEscapeSystemName(t *testing.T) {
	resetInjector()
	withSystemBranding(t, `<script>alert(1)</script>`, "", func() {
		out := injector.renderIndex("default", []byte(defaultIndexFixture))
		s := string(out)
		if strings.Contains(s, "<script>alert(1)</script>") {
			t.Errorf("XSS payload not escaped in title; got:\n%s", s)
		}
		if !strings.Contains(s, "&lt;script&gt;alert(1)&lt;/script&gt;") {
			t.Errorf("expected escaped payload in output; got:\n%s", s)
		}
	})
}

func TestRenderIndex_CustomLogo_ReplacesFavicon(t *testing.T) {
	resetInjector()
	withSystemBranding(t, "云链API", "https://cdn.example.com/yunlink.png", func() {
		out := injector.renderIndex("default", []byte(defaultIndexFixture))
		s := string(out)
		if !strings.Contains(s, `href="https://cdn.example.com/yunlink.png"`) {
			t.Errorf("favicon href not replaced; got:\n%s", s)
		}
		if strings.Contains(s, `href="/logo.png"`) {
			t.Errorf("stale /logo.png remains; got:\n%s", s)
		}
	})
}

func TestRenderIndex_InlineSvgLogo_KeepsDefaultFavicon(t *testing.T) {
	resetInjector()
	withSystemBranding(t, "YiYongAI", `<svg viewBox="0 0 24 24"><path d="M0 0h24v24H0z" /></svg>`, func() {
		out := injector.renderIndex("default", []byte(defaultIndexFixture))

		require.Contains(t, string(out), `href="/logo.png"`)
		require.NotContains(t, string(out), `<svg`)
	})
}

func TestRenderIndex_ClassicTheme(t *testing.T) {
	resetInjector()
	withSystemBranding(t, "云链API", "", func() {
		out := injector.renderIndex("classic", []byte(classicIndexFixture))
		s := string(out)
		if !strings.Contains(s, "<title>云链API</title>") {
			t.Errorf("classic title not replaced; got:\n%s", s)
		}
	})
}

func TestRenderIndex_CacheHit(t *testing.T) {
	resetInjector()
	withSystemBranding(t, "云链API", "", func() {
		_ = injector.renderIndex("default", []byte(defaultIndexFixture))
		_ = injector.renderIndex("default", []byte(defaultIndexFixture))
		_ = injector.renderIndex("default", []byte(defaultIndexFixture))
		// 3 calls, 2 hits expected (1st = miss + store, 2nd/3rd = hit)
		if hits := injector.hitCount.Load(); hits != 2 {
			t.Errorf("expected 2 cache hits across 3 calls, got %d", hits)
		}
	})
}

func TestRenderIndex_CacheInvalidatesOnNameChange(t *testing.T) {
	resetInjector()
	withSystemBranding(t, "云链API", "", func() {
		first := injector.renderIndex("default", []byte(defaultIndexFixture))
		if !bytes.Contains(first, []byte("<title>云链API</title>")) {
			t.Fatalf("first render missing 云链API title")
		}
		// Simulate admin renaming the site at runtime.
		common.SystemName = "MyGateway"
		second := injector.renderIndex("default", []byte(defaultIndexFixture))
		if !bytes.Contains(second, []byte("<title>MyGateway</title>")) {
			t.Errorf("cache not invalidated on SystemName change; got:\n%s", second)
		}
	})
}
