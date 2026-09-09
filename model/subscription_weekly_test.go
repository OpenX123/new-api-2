package model

import (
	"math"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupWeeklySubscription(t *testing.T, weekly int64) (*gorm.DB, UserSubscription) {
	t.Helper()
	db := setupAdminSubscriptionResetTestDB(t)
	previousType := common.MainDatabaseType()
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() { common.SetMainDatabaseType(previousType) })
	require.NoError(t, db.AutoMigrate(&SubscriptionPlan{}, &SubscriptionPreConsumeRecord{}))
	plan := SubscriptionPlan{Title: t.Name(), TotalAmount: 100, WeeklyAmount: weekly, QuotaResetPeriod: SubscriptionResetCustom, QuotaResetCustomSeconds: 18000}
	require.NoError(t, db.Create(&plan).Error)
	InvalidateSubscriptionPlanCache(plan.Id)
	t.Cleanup(func() { InvalidateSubscriptionPlanCache(plan.Id) })
	now := time.Now().Unix()
	sub := UserSubscription{UserId: 1, PlanId: plan.Id, AmountTotal: 100, WeeklyAmount: weekly,
		StartTime: now - 3600, EndTime: now + 30*86400, Status: "active", LastResetTime: now - 3600, NextResetTime: now + 14400,
		WeeklyResetTime: now - 3600 + 7*86400}
	require.NoError(t, db.Create(&sub).Error)
	return db, sub
}

func TestWeeklySubscriptionSurvivesShortReset(t *testing.T) {
	db, sub := setupWeeklySubscription(t, 150)
	_, err := PreConsumeUserSubscription("first", 1, "", 0, 100)
	require.NoError(t, err)
	now := time.Now().Unix()
	require.NoError(t, db.Model(&sub).Updates(map[string]interface{}{"last_reset_time": now - 18001, "next_reset_time": now - 1}).Error)
	_, err = PreConsumeUserSubscription("second", 1, "", 0, 50)
	require.NoError(t, err)
	_, err = PreConsumeUserSubscription("blocked", 1, "", 0, 1)
	require.ErrorContains(t, err, "quota insufficient")
	require.NoError(t, db.First(&sub, sub.Id).Error)
	assert.Equal(t, int64(50), sub.AmountUsed)
	assert.Equal(t, int64(150), sub.WeeklyUsed)
	var count int64
	require.NoError(t, db.Model(&SubscriptionPreConsumeRecord{}).Where("request_id = ?", "blocked").Count(&count).Error)
	assert.Zero(t, count)
}

func TestWeeklySubscriptionAdjustAndRefundAreAtomic(t *testing.T) {
	db, sub := setupWeeklySubscription(t, 60)
	_, err := PreConsumeUserSubscription("request", 1, "", 0, 40)
	require.NoError(t, err)
	_, err = PreConsumeUserSubscription("request", 1, "", 0, 40)
	require.NoError(t, err)
	require.NoError(t, AdjustSubscriptionPreConsume("request", 20))
	require.ErrorContains(t, AdjustSubscriptionPreConsume("request", 1), "weekly quota insufficient")
	require.NoError(t, db.First(&sub, sub.Id).Error)
	assert.Equal(t, int64(60), sub.AmountUsed)
	assert.Equal(t, int64(60), sub.WeeklyUsed)
	require.NoError(t, RefundSubscriptionPreConsume("request"))
	require.NoError(t, RefundSubscriptionPreConsume("request"))
	require.NoError(t, db.First(&sub, sub.Id).Error)
	assert.Zero(t, sub.AmountUsed)
	assert.Zero(t, sub.WeeklyUsed)
}

func TestWeeklySubscriptionSettlementRefundsDifference(t *testing.T) {
	db, sub := setupWeeklySubscription(t, 60)
	result, err := PreConsumeUserSubscription("request", 1, "", 0, 40)
	require.NoError(t, err)
	require.NoError(t, PostConsumeUserSubscriptionDelta(sub.Id, -10, result.UsageTime))
	require.NoError(t, db.First(&sub, sub.Id).Error)
	assert.Equal(t, int64(30), sub.AmountUsed)
	assert.Equal(t, int64(30), sub.WeeklyUsed)
}

func TestWeeklySubscriptionLateRefundDoesNotCreditNewWeek(t *testing.T) {
	db, sub := setupWeeklySubscription(t, 60)
	now := time.Now().Unix()
	oldTime := now - 7*86400 - 60
	require.NoError(t, db.Model(&sub).Updates(map[string]interface{}{
		"start_time": oldTime, "weekly_reset_time": now - 60, "weekly_used": 50, "amount_used": 50,
	}).Error)
	record := SubscriptionPreConsumeRecord{RequestId: "old", UserId: 1, UserSubscriptionId: sub.Id, PreConsumed: 50, Status: "consumed", CreatedAt: oldTime}
	require.NoError(t, db.Create(&record).Error)
	_, err := PreConsumeUserSubscription("new", 1, "", 0, 20)
	require.NoError(t, err)
	require.NoError(t, RefundSubscriptionPreConsume("old"))
	require.NoError(t, PostConsumeUserSubscriptionDelta(sub.Id, 5, oldTime))
	require.NoError(t, db.First(&sub, sub.Id).Error)
	assert.Equal(t, int64(20), sub.WeeklyUsed)
	assert.Equal(t, oldTime+14*86400, sub.WeeklyResetTime)
}

func TestWeeklySubscriptionZeroPreservesLegacyQuota(t *testing.T) {
	db, sub := setupWeeklySubscription(t, 0)
	_, err := PreConsumeUserSubscription("legacy", 1, "", 0, 100)
	require.NoError(t, err)
	_, err = PreConsumeUserSubscription("blocked", 1, "", 0, 1)
	require.Error(t, err)
	require.NoError(t, db.First(&sub, sub.Id).Error)
	assert.Equal(t, int64(100), sub.AmountUsed)
	assert.Zero(t, sub.WeeklyUsed)
}

func TestWeeklySubscriptionResetJobWorksWithoutShortReset(t *testing.T) {
	db, sub := setupWeeklySubscription(t, 60)
	now := time.Now().Unix()
	require.NoError(t, db.Model(&SubscriptionPlan{}).Where("id = ?", sub.PlanId).Update("quota_reset_period", SubscriptionResetNever).Error)
	InvalidateSubscriptionPlanCache(sub.PlanId)
	require.NoError(t, db.Model(&sub).Updates(map[string]interface{}{
		"start_time": now - 14*86400, "next_reset_time": 0, "weekly_reset_time": now - 7*86400, "weekly_used": 60, "amount_used": 60,
	}).Error)
	count, err := ResetDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.NoError(t, db.First(&sub, sub.Id).Error)
	assert.Zero(t, sub.WeeklyUsed)
	assert.Equal(t, int64(60), sub.AmountUsed)
	assert.Equal(t, now+7*86400, sub.WeeklyResetTime)
}

func TestWeeklySubscriptionRejectsOverflow(t *testing.T) {
	db, sub := setupWeeklySubscription(t, math.MaxInt64)
	require.NoError(t, db.Model(&sub).Updates(map[string]interface{}{"amount_total": 0, "amount_used": math.MaxInt64 - 1, "weekly_used": math.MaxInt64 - 1}).Error)
	_, err := PreConsumeUserSubscription("overflow", 1, "", 0, 2)
	require.Error(t, err)
	require.Error(t, PostConsumeUserSubscriptionDelta(sub.Id, 2))
	require.NoError(t, db.First(&sub, sub.Id).Error)
	assert.Equal(t, int64(math.MaxInt64-1), sub.AmountUsed)
	assert.Equal(t, int64(math.MaxInt64-1), sub.WeeklyUsed)
}

func TestWeeklyPlanSQLiteMigrationPreservesExistingPlan(t *testing.T) {
	db := setupAdminSubscriptionResetTestDB(t)
	previousType := common.MainDatabaseType()
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() { common.SetMainDatabaseType(previousType) })
	require.NoError(t, db.Exec("CREATE TABLE subscription_plans (id integer PRIMARY KEY, title varchar(128) NOT NULL, price_amount decimal(10,6) NOT NULL)").Error)
	require.NoError(t, db.Exec("INSERT INTO subscription_plans (id, title, price_amount) VALUES (1, 'existing', 10)").Error)
	require.NoError(t, ensureSubscriptionPlanTableSQLite())
	require.NoError(t, ensureSubscriptionPlanTableSQLite())
	var plan SubscriptionPlan
	require.NoError(t, db.First(&plan, 1).Error)
	assert.Equal(t, "existing", plan.Title)
	assert.Zero(t, plan.WeeklyAmount)
	plan.WeeklyAmount = 250
	require.NoError(t, db.Save(&plan).Error)
	require.NoError(t, db.First(&plan, 1).Error)
	assert.Equal(t, int64(250), plan.WeeklyAmount)
}

func TestWeeklySubscriptionSnapshotsPlanAtActivation(t *testing.T) {
	db, existing := setupWeeklySubscription(t, 150)
	var plan SubscriptionPlan
	require.NoError(t, db.First(&plan, existing.PlanId).Error)
	plan.DurationUnit = SubscriptionDurationMonth
	plan.DurationValue = 1
	plan.WeeklyAmount = 300
	require.NoError(t, db.Save(&plan).Error)
	var created *UserSubscription
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		created, err = CreateUserSubscriptionFromPlanTx(tx, 2, &plan, "admin")
		return err
	}))
	assert.Equal(t, int64(300), created.WeeklyAmount)
	assert.Zero(t, created.WeeklyUsed)
	assert.Equal(t, created.StartTime+7*86400, created.WeeklyResetTime)
	assert.Equal(t, created.StartTime+18000, created.NextResetTime)
	assert.Equal(t, time.Unix(created.StartTime, 0).AddDate(0, 1, 0).Unix(), created.EndTime)
	require.NoError(t, db.First(&existing, existing.Id).Error)
	assert.Equal(t, int64(150), existing.WeeklyAmount)
}
