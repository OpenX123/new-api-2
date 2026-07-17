package controller

import (
	"fmt"
	"math"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type batchAddQuotaResponse struct {
	Success bool `json:"success"`
	Data    int  `json:"data"`
}

func performBatchAddQuotaRequest(t *testing.T, body string, role int) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("POST", "/api/user/batch/quota", strings.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("id", 100)
	context.Set("username", "operator")
	context.Set("role", role)

	BatchAddUserQuota(context)
	return recorder
}

func TestBatchAddUserQuotaRejectsInvalidRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tooManyIds := make([]int, maxBatchQuotaUsers+1)
	for index := range tooManyIds {
		tooManyIds[index] = index + 1
	}
	tooManyBody, err := common.Marshal(UserQuotaBatch{Ids: tooManyIds, Value: 300})
	require.NoError(t, err)

	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: "{"},
		{name: "empty IDs", body: `{"ids":[],"value":300}`},
		{name: "too many IDs", body: string(tooManyBody)},
		{name: "zero value", body: `{"ids":[1],"value":0}`},
		{name: "negative value", body: `{"ids":[1],"value":-1}`},
		{name: "value above quota storage limit", body: fmt.Sprintf(`{"ids":[1],"value":%d}`, int64(math.MaxInt32)+1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performBatchAddQuotaRequest(t, test.body, common.RoleRootUser)
			assert.Equal(t, 200, response.Code)

			var payload batchAddQuotaResponse
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
			assert.False(t, payload.Success)
		})
	}
}

func TestBatchAddUserQuotaUpdatesUsersAndRecordsAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousMainType := common.MainDatabaseType()
	previousLogType := common.LogDatabaseType()
	previousRedisEnabled := common.RedisEnabled

	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}))

	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		common.RedisEnabled = previousRedisEnabled
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	users := []model.User{
		{Id: 100, Username: "operator", Role: common.RoleRootUser, Quota: 1000, AffCode: "operator"},
		{Id: 1, Username: "target", Role: common.RoleCommonUser, Quota: 50, AffCode: "target"},
	}
	require.NoError(t, db.Create(&users).Error)

	response := performBatchAddQuotaRequest(t, `{"ids":[1],"value":300}`, common.RoleRootUser)
	assert.Equal(t, 200, response.Code)

	var payload batchAddQuotaResponse
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	assert.True(t, payload.Success)
	assert.Equal(t, 1, payload.Data)

	var target model.User
	require.NoError(t, db.First(&target, 1).Error)
	assert.Equal(t, 350, target.Quota)

	var auditLogs []model.Log
	require.NoError(t, db.Where("type = ?", model.LogTypeManage).Find(&auditLogs).Error)
	require.Len(t, auditLogs, 1)
	assert.Equal(t, 100, auditLogs[0].UserId)
	assert.Contains(t, auditLogs[0].Content, "Increased user quota by")
	assert.Contains(t, auditLogs[0].Other, `"action":"user.quota_add"`)
	assert.Contains(t, auditLogs[0].Other, `"target_user_id":1`)
}
