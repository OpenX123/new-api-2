package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type CCSwitchSetting struct {
	ClaudeName        string `json:"claude_name"`
	ClaudeModel       string `json:"claude_model"`
	ClaudeHaikuModel  string `json:"claude_haiku_model"`
	ClaudeSonnetModel string `json:"claude_sonnet_model"`
	ClaudeOpusModel   string `json:"claude_opus_model"`
	CodexName         string `json:"codex_name"`
	CodexModel        string `json:"codex_model"`
	GeminiName        string `json:"gemini_name"`
	GeminiModel       string `json:"gemini_model"`
}

var ccSwitchSetting = CCSwitchSetting{
	ClaudeName: "My Claude",
	CodexName:  "My Codex",
	GeminiName: "My Gemini",
}

func init() {
	config.GlobalConfig.Register("cc_switch_setting", &ccSwitchSetting)
}

func GetCCSwitchSetting() *CCSwitchSetting {
	return &ccSwitchSetting
}
