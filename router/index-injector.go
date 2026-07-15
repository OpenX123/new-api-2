package router

import (
	"bytes"
	"html"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// indexInjector renders the embedded index.html with the live SystemName / Logo
// applied to <title>, <meta name="title">, and the favicon <link>. This removes
// the title-flicker that otherwise happens because the SPA only updates the
// document title in JS after `/api/status` returns.
//
// The rendered output is cached and invalidated whenever SystemName or Logo
// change at runtime (admin updates), so the per-request cost is just a map
// lookup plus a small bytes copy via gin's c.Data.

type indexCacheKey struct {
	theme      string
	systemName string
	logo       string
}

type indexInjector struct {
	cache    sync.Map // indexCacheKey -> []byte
	hitCount atomic.Uint64
}

var injector = &indexInjector{}

// defaultTitleNeedle / classicTitleNeedle: both themes ship the literal
// "<title>New API</title>" string in their built index.html. We do a direct
// byte replace rather than introducing a template placeholder so the upstream
// index.html stays untouched.
var (
	titleNeedle       = []byte("<title>New API</title>")
	metaTitleNeedle   = []byte(`<meta name="title" content="New API" />`)
	faviconNeedlePng  = []byte(`href="/logo.png"`)
	faviconAltNeedle1 = []byte(`<link rel="icon" type="image/png" href="/logo.png" />`)
	faviconAltNeedle2 = []byte(`<link rel="icon" href="/logo.png" />`)
)

func isInlineSVGLogo(logo string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(logo)), "<svg")
}

// renderIndex returns the index.html bytes for the given theme with the
// current SystemName / Logo substituted in. The result is cached per
// (theme, systemName, logo) triple.
func (ij *indexInjector) renderIndex(theme string, base []byte) []byte {
	systemName := common.SystemName
	logo := common.Logo

	key := indexCacheKey{theme: theme, systemName: systemName, logo: logo}
	if cached, ok := ij.cache.Load(key); ok {
		ij.hitCount.Add(1)
		return cached.([]byte)
	}

	out := make([]byte, len(base))
	copy(out, base)

	// Skip title replacement when the admin hasn't customized SystemName —
	// keep the upstream literal so behaviour is identical to vanilla new-api.
	if systemName != "" && systemName != "New API" {
		escaped := html.EscapeString(systemName)
		out = bytes.ReplaceAll(out, titleNeedle,
			[]byte("<title>"+escaped+"</title>"))
		out = bytes.ReplaceAll(out, metaTitleNeedle,
			[]byte(`<meta name="title" content="`+escaped+`" />`))
	}

	// Logo: if admin uploaded a custom logo URL, swap the favicon href.
	if logo != "" && !isInlineSVGLogo(logo) && !strings.Contains(logo, "/logo.png") {
		// Replace any occurrence of href="/logo.png" with the new URL.
		// Quote the URL safely (admin-controlled, expected to be a clean URL,
		// but escape just in case).
		safeLogo := strings.ReplaceAll(logo, `"`, "")
		repl := []byte(`href="` + safeLogo + `"`)
		out = bytes.ReplaceAll(out, faviconNeedlePng, repl)
		// Also handle the longer literal forms in case the short needle was
		// already replaced by analytics injection or future edits.
		_ = faviconAltNeedle1
		_ = faviconAltNeedle2
	}

	ij.cache.Store(key, out)
	return out
}

// serveIndex writes the rendered index.html for the active theme.
func serveIndex(c *gin.Context, assets ThemeAssets) {
	c.Header("Cache-Control", "no-cache")
	var (
		theme string
		base  []byte
	)
	if common.GetTheme() == "classic" {
		theme = "classic"
		base = assets.ClassicIndexPage
	} else {
		theme = "default"
		base = assets.DefaultIndexPage
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8",
		injector.renderIndex(theme, base))
}
