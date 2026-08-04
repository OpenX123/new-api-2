package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAllowedRelayClient(t *testing.T) {
	setting := GetTokenSetting()
	originalEnabled := setting.ClientRestrictionEnabled
	originalUserAgents := setting.AllowedClientUserAgents
	t.Cleanup(func() {
		setting.ClientRestrictionEnabled = originalEnabled
		setting.AllowedClientUserAgents = originalUserAgents
	})

	setting.ClientRestrictionEnabled = true
	setting.AllowedClientUserAgents = "claude-cli, codex_cli_rs, opencode\ncustom-client"
	require.NotEmpty(t, setting.AllowedClientUserAgents)

	tests := []struct {
		name      string
		userAgent string
		allowed   bool
	}{
		{name: "claude cli", userAgent: "claude-cli/1.2.3 (external, cli)", allowed: true},
		{name: "codex cli case insensitive", userAgent: "Codex_CLI_RS/0.9.0", allowed: true},
		{name: "opencode", userAgent: "opencode/1.0.120", allowed: true},
		{name: "custom client", userAgent: "custom-client/1.0", allowed: true},
		{name: "curl", userAgent: "curl/8.0", allowed: false},
		{name: "missing user agent", userAgent: "", allowed: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.allowed, IsAllowedRelayClient(test.userAgent))
		})
	}
}

func TestIsAllowedRelayClientDisabled(t *testing.T) {
	setting := GetTokenSetting()
	originalEnabled := setting.ClientRestrictionEnabled
	t.Cleanup(func() { setting.ClientRestrictionEnabled = originalEnabled })

	setting.ClientRestrictionEnabled = false
	assert.True(t, IsAllowedRelayClient(""))
}
