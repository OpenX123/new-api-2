package helper

import (
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleGroupRatioAppliesUserRatio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1.5}`))
	require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(`{"svip":{"vip":0.5}}`))

	tests := []struct {
		name             string
		userGroup        string
		usingGroup       string
		userRatio        float64
		wantGroupRatio   float64
		wantSpecial      bool
		wantSpecialRatio float64
	}{
		{"user ratio 1 keeps group ratio", "default", "vip", 1.0, 1.5, false, -1},
		{"user ratio multiplies group ratio", "default", "vip", 0.8, 1.2, false, -1},
		{"zero-value user ratio treated as 1", "default", "vip", 0, 1.5, false, -1},
		{"user ratio multiplies group-group special ratio", "svip", "vip", 0.8, 0.4, true, 0.4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
			info := &relaycommon.RelayInfo{
				UserGroup:  tt.userGroup,
				UsingGroup: tt.usingGroup,
				UserRatio:  tt.userRatio,
			}
			got := HandleGroupRatio(c, info)
			assert.InDelta(t, tt.wantGroupRatio, got.GroupRatio, 1e-9)
			assert.Equal(t, tt.wantSpecial, got.HasSpecialRatio)
			if tt.wantSpecial {
				assert.InDelta(t, tt.wantSpecialRatio, got.GroupSpecialRatio, 1e-9)
			}
		})
	}
}
