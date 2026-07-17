package model

import (
	"fmt"
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupBatchQuotaTestDB(t *testing.T) {
	t.Helper()

	previousDB := DB
	dsn := fmt.Sprintf("file:batch_quota_%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}))
	DB = db
	t.Cleanup(func() { DB = previousDB })
}

func TestBatchIncreaseUserQuotaUpdatesOnlyManageableUsers(t *testing.T) {
	setupBatchQuotaTestDB(t)

	users := []User{
		{Id: 1, Username: "common", Role: common.RoleCommonUser, Quota: 100, AffCode: "common"},
		{Id: 2, Username: "admin", Role: common.RoleAdminUser, Quota: 200, AffCode: "admin"},
		{Id: 3, Username: "root", Role: common.RoleRootUser, Quota: 300, AffCode: "root"},
	}
	require.NoError(t, DB.Create(&users).Error)

	updatedIds, err := BatchIncreaseUserQuota(
		[]int{1, 1, 2, 3},
		50,
		common.RoleRootUser,
	)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{1, 2}, updatedIds)

	var updatedUsers []User
	require.NoError(t, DB.Order("id").Find(&updatedUsers).Error)
	require.Len(t, updatedUsers, 3)
	assert.Equal(t, 150, updatedUsers[0].Quota)
	assert.Equal(t, 250, updatedUsers[1].Quota)
	assert.Equal(t, 300, updatedUsers[2].Quota)
}

func TestBatchIncreaseUserQuotaIsAtomicOnOverflow(t *testing.T) {
	setupBatchQuotaTestDB(t)

	users := []User{
		{Id: 1, Username: "safe", Role: common.RoleCommonUser, Quota: 100, AffCode: "safe"},
		{Id: 2, Username: "overflow", Role: common.RoleCommonUser, Quota: math.MaxInt32 - 10, AffCode: "overflow"},
	}
	require.NoError(t, DB.Create(&users).Error)

	_, err := BatchIncreaseUserQuota([]int{1, 2}, 20, common.RoleRootUser)
	require.Error(t, err)

	var safe User
	require.NoError(t, DB.First(&safe, 1).Error)
	assert.Equal(t, 100, safe.Quota)
}

func TestBatchIncreaseUserQuotaRespectsAdminRoleBoundary(t *testing.T) {
	setupBatchQuotaTestDB(t)

	users := []User{
		{Id: 1, Username: "common", Role: common.RoleCommonUser, Quota: 100, AffCode: "common"},
		{Id: 2, Username: "admin", Role: common.RoleAdminUser, Quota: 200, AffCode: "admin"},
		{Id: 3, Username: "root", Role: common.RoleRootUser, Quota: 300, AffCode: "root"},
	}
	require.NoError(t, DB.Create(&users).Error)

	updatedIds, err := BatchIncreaseUserQuota(
		[]int{1, 2, 3},
		50,
		common.RoleAdminUser,
	)
	require.NoError(t, err)
	assert.Equal(t, []int{1}, updatedIds)

	var updatedUsers []User
	require.NoError(t, DB.Order("id").Find(&updatedUsers).Error)
	require.Len(t, updatedUsers, 3)
	assert.Equal(t, 150, updatedUsers[0].Quota)
	assert.Equal(t, 200, updatedUsers[1].Quota)
	assert.Equal(t, 300, updatedUsers[2].Quota)
}
