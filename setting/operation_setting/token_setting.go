package operation_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

// TokenSetting 令牌相关配置
type TokenSetting struct {
	MaxUserTokens            int    `json:"max_user_tokens"`            // 每用户最大令牌数量
	ClientRestrictionEnabled bool   `json:"client_restriction_enabled"` // 是否仅允许指定客户端访问中继接口
	AllowedClientUserAgents  string `json:"allowed_client_user_agents"` // User-Agent 关键词，逗号或换行分隔
}

// 默认配置
var tokenSetting = TokenSetting{
	MaxUserTokens:           1000, // 默认每用户最多 1000 个令牌
	AllowedClientUserAgents: "claude-cli,claude-code,codex_cli_rs,codex-cli,opencode,cline,roo-code,roo-cline,continue,aider,cursor,windsurf,zed",
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("token_setting", &tokenSetting)
}

// GetTokenSetting 获取令牌配置
func GetTokenSetting() *TokenSetting {
	return &tokenSetting
}

// GetMaxUserTokens 获取每用户最大令牌数量
func GetMaxUserTokens() int {
	return GetTokenSetting().MaxUserTokens
}

// IsAllowedRelayClient reports whether a relay request's User-Agent matches
// one of the configured, case-insensitive keywords. An empty allowlist denies
// every request when client restriction is enabled.
func IsAllowedRelayClient(userAgent string) bool {
	setting := GetTokenSetting()
	if !setting.ClientRestrictionEnabled {
		return true
	}

	userAgent = strings.ToLower(strings.TrimSpace(userAgent))
	if userAgent == "" {
		return false
	}

	for _, keyword := range strings.FieldsFunc(setting.AllowedClientUserAgents, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	}) {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" && strings.Contains(userAgent, keyword) {
			return true
		}
	}
	return false
}
