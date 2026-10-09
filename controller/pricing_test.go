package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPricingHonorsBlockedGroups(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	originalGroups := setting.UserUsableGroups2JSONString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	special := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
	originalSpecial := special.ReadAll()
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
		special.Clear()
		special.AddAll(originalSpecial)
		model.InvalidatePricingCache()
	})
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","hidden":"Hidden"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"hidden":0.8}`))
	special.Set("default", map[string]string{"-:hidden": ""})
	require.NoError(t, db.Create(&model.User{Id: 9701, Username: "pricing-user", Group: "default", Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "shared-model", ChannelId: 1, Enabled: true},
		{Group: "hidden", Model: "shared-model", ChannelId: 1, Enabled: true},
		{Group: "hidden", Model: "hidden-model", ChannelId: 1, Enabled: true},
	}).Error)
	model.InvalidatePricingCache()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 9701)
	GetPricing(ctx)
	var response struct {
		Success bool               `json:"success"`
		Data    []model.Pricing    `json:"data"`
		Groups  map[string]string  `json:"usable_group"`
		Ratios  map[string]float64 `json:"group_ratio"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, map[string]string{"default": "Default"}, response.Groups)
	assert.Equal(t, map[string]float64{"default": 1}, response.Ratios)
	require.Len(t, response.Data, 1)
	assert.Equal(t, "shared-model", response.Data[0].ModelName)
	assert.Equal(t, []string{"default"}, response.Data[0].EnableGroup)
	// Public visitors must not reveal groups blocked for default users either.
	publicRecorder := httptest.NewRecorder()
	publicCtx, _ := gin.CreateTestContext(publicRecorder)
	GetPricing(publicCtx)
	require.NoError(t, common.Unmarshal(publicRecorder.Body.Bytes(), &response))
	assert.Equal(t, map[string]string{"default": "Default"}, response.Groups)
	assert.Equal(t, map[string]float64{"default": 1}, response.Ratios)
	require.Len(t, response.Data, 1)
	assert.Equal(t, []string{"default"}, response.Data[0].EnableGroup)
	// Per-user filtering must never mutate the shared pricing cache.
	for _, item := range model.GetPricing() {
		if item.ModelName == "shared-model" {
			assert.ElementsMatch(t, []string{"default", "hidden"}, item.EnableGroup)
		}
	}
}

func TestFilterPricingGroupsAllAndEmpty(t *testing.T) {
	pricing := []model.Pricing{{ModelName: "universal", EnableGroup: []string{"all", "hidden"}}}
	filtered := filterPricingByUsableGroups(pricing, map[string]string{"default": "Default"})
	require.Len(t, filtered, 1)
	assert.Equal(t, []string{"default"}, filtered[0].EnableGroup)
	assert.Empty(t, filterPricingByUsableGroups(pricing, map[string]string{}))
	assert.Equal(t, []string{"all", "hidden"}, pricing[0].EnableGroup)
}
