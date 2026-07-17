package model

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupLogIPTestDB(t *testing.T) {
	t.Helper()

	previousDB := DB
	previousLogDB := LOG_DB
	previousLogConsumeEnabled := common.LogConsumeEnabled
	previousLogRecordIPEnabled := common.LogRecordIPEnabled
	previousDataExportEnabled := common.DataExportEnabled

	dsn := fmt.Sprintf("file:log_ip_%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &Log{}))
	DB = db
	LOG_DB = db
	common.LogConsumeEnabled = true
	common.LogRecordIPEnabled = false
	common.DataExportEnabled = false

	t.Cleanup(func() {
		DB = previousDB
		LOG_DB = previousLogDB
		common.LogConsumeEnabled = previousLogConsumeEnabled
		common.LogRecordIPEnabled = previousLogRecordIPEnabled
		common.DataExportEnabled = previousDataExportEnabled
	})
}

func newLogIPTestContext(ip string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.RemoteAddr = ip + ":12345"
	return c
}

func TestRecordUsageLogsAlwaysStoreClientIP(t *testing.T) {
	setupLogIPTestDB(t)
	c := newLogIPTestContext("198.51.100.7")

	RecordErrorLog(c, 1, 2, "test-model", "test-token", "upstream error", 0, 1, false, "default", nil)
	RecordConsumeLog(c, 1, RecordConsumeLogParams{
		ChannelId: 2,
		ModelName: "test-model",
		Group:     "default",
	})

	var logs []Log
	require.NoError(t, LOG_DB.Order("type").Find(&logs).Error)
	require.Len(t, logs, 2)
	assert.Equal(t, "198.51.100.7", logs[0].Ip)
	assert.Equal(t, "198.51.100.7", logs[1].Ip)
}
