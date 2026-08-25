package model

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAdminSubscriptionResetTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := DB
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&UserSubscription{}))
	DB = db
	t.Cleanup(func() {
		DB = previousDB
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestAdminResetUserSubscriptionQuotaClearsUsageOnly(t *testing.T) {
	db := setupAdminSubscriptionResetTestDB(t)
	now := time.Now().Unix()
	sub := UserSubscription{
		Id:             11,
		UserId:         22,
		PlanId:         33,
		AmountTotal:    1_000,
		AmountUsed:     450,
		StartTime:      now - 3600,
		EndTime:        now + 3600,
		Status:         "active",
		LastResetTime:  now - 1800,
		NextResetTime:  now + 1800,
		UpgradeGroup:   "vip",
		DowngradeGroup: "basic",
	}
	require.NoError(t, db.Create(&sub).Error)

	result, err := AdminResetUserSubscriptionQuota(sub.Id)
	require.NoError(t, err)
	assert.Equal(t, sub.UserId, result.UserId)
	assert.Equal(t, sub.PlanId, result.PlanId)
	assert.Equal(t, sub.Id, result.SubscriptionId)
	assert.Equal(t, int64(450), result.AmountUsedBefore)
	assert.Zero(t, result.AmountUsedAfter)

	var stored UserSubscription
	require.NoError(t, db.First(&stored, sub.Id).Error)
	assert.Zero(t, stored.AmountUsed)
	assert.Equal(t, sub.StartTime, stored.StartTime)
	assert.Equal(t, sub.EndTime, stored.EndTime)
	assert.Equal(t, sub.LastResetTime, stored.LastResetTime)
	assert.Equal(t, sub.NextResetTime, stored.NextResetTime)
	assert.Equal(t, "active", stored.Status)
	assert.Equal(t, "vip", stored.UpgradeGroup)
	assert.Equal(t, "basic", stored.DowngradeGroup)
}

func TestAdminClearUserSubscriptionQuotaClearsRemainingQuotaOnly(t *testing.T) {
	db := setupAdminSubscriptionResetTestDB(t)
	now := time.Now().Unix()
	sub := UserSubscription{
		Id:            12,
		UserId:        23,
		PlanId:        34,
		AmountTotal:   1_000,
		AmountUsed:    450,
		StartTime:     now - 3600,
		EndTime:       now + 3600,
		Status:        "active",
		LastResetTime: now - 1800,
		NextResetTime: now + 1800,
	}
	require.NoError(t, db.Create(&sub).Error)

	result, err := AdminClearUserSubscriptionQuota(sub.Id)
	require.NoError(t, err)
	assert.Equal(t, int64(450), result.AmountUsedBefore)
	assert.Equal(t, int64(1_000), result.AmountUsedAfter)

	var stored UserSubscription
	require.NoError(t, db.First(&stored, sub.Id).Error)
	assert.Equal(t, sub.AmountTotal, stored.AmountUsed)
	assert.Equal(t, sub.StartTime, stored.StartTime)
	assert.Equal(t, sub.EndTime, stored.EndTime)
	assert.Equal(t, sub.LastResetTime, stored.LastResetTime)
	assert.Equal(t, sub.NextResetTime, stored.NextResetTime)
	assert.Equal(t, "active", stored.Status)
}

func TestAdminResetUserSubscriptionQuotaRejectsInactiveSubscription(t *testing.T) {
	db := setupAdminSubscriptionResetTestDB(t)
	now := time.Now().Unix()
	subs := []UserSubscription{
		{Id: 1, UserId: 2, PlanId: 3, AmountTotal: 1_000, AmountUsed: 400, StartTime: now - 7200, EndTime: now - 3600, Status: "active"},
		{Id: 2, UserId: 2, PlanId: 3, AmountTotal: 1_000, AmountUsed: 500, StartTime: now - 3600, EndTime: now + 3600, Status: "cancelled"},
	}
	require.NoError(t, db.Create(&subs).Error)

	operations := []func(int) (*AdminUserSubscriptionQuotaResult, error){
		AdminResetUserSubscriptionQuota,
		AdminClearUserSubscriptionQuota,
	}
	for _, operation := range operations {
		for _, sub := range subs {
			_, err := operation(sub.Id)
			require.ErrorContains(t, err, "subscription is not active")

			var stored UserSubscription
			require.NoError(t, db.First(&stored, sub.Id).Error)
			assert.Equal(t, sub.AmountUsed, stored.AmountUsed)
		}
	}
}
