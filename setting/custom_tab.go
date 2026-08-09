package setting

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// CustomTab 描述一个由管理员配置的自定义标签页，在控制台侧边栏中渲染为 iframe 入口。
// URL 支持两个占位符：{key} 替换为用户当前可用的 API Key，{address} 替换为站点地址。
type CustomTab struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

var CustomTabs = make([]CustomTab, 0)

func UpdateCustomTabsByJsonString(jsonString string) error {
	CustomTabs = make([]CustomTab, 0)
	if strings.TrimSpace(jsonString) == "" {
		return nil
	}
	return common.UnmarshalJsonStr(jsonString, &CustomTabs)
}

func CustomTabs2JsonString() string {
	jsonBytes, err := common.Marshal(CustomTabs)
	if err != nil {
		common.SysLog("error marshalling custom tabs: " + err.Error())
		return "[]"
	}
	return string(jsonBytes)
}
