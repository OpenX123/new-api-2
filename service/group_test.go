package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// setBaseGroups 重置全局基础可用分组,隔离测试。
func setBaseGroups(t *testing.T, json string) {
	t.Helper()
	if err := setting.UpdateUserUsableGroupsByJSONString(json); err != nil {
		t.Fatalf("failed to set base groups: %v", err)
	}
}

func TestGetUserUsableGroups_RemoveSelf(t *testing.T) {
	// 基础分组只有 default,不含自定义组
	setBaseGroups(t, `{"default":"默认分组"}`)

	t.Run("removing own group via -: keeps it out of the list", func(t *testing.T) {
		ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.Set("0.6custom", map[string]string{
			"稳定claude":     "低价渠道",
			"官方claude":     "满血CC MAX",
			"-:0.6custom": "remove",
		})
		groups := GetUserUsableGroups("0.6custom")
		if _, ok := groups["0.6custom"]; ok {
			t.Fatalf("group '0.6custom' should be removed but is still present: %+v", groups)
		}
		// 被 append 的分组仍应存在
		if _, ok := groups["稳定claude"]; !ok {
			t.Fatalf("appended group '稳定claude' missing: %+v", groups)
		}
	})

	t.Run("group without remove rule is auto-added as fallback", func(t *testing.T) {
		ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.Set("plaingroup", map[string]string{
			"官方claude": "满血CC MAX",
		})
		groups := GetUserUsableGroups("plaingroup")
		if _, ok := groups["plaingroup"]; !ok {
			t.Fatalf("group 'plaingroup' should be auto-added as fallback: %+v", groups)
		}
	})

	t.Run("group with no rules at all is auto-added as fallback", func(t *testing.T) {
		groups := GetUserUsableGroups("norules")
		if _, ok := groups["norules"]; !ok {
			t.Fatalf("group 'norules' should be auto-added as fallback: %+v", groups)
		}
	})
}
