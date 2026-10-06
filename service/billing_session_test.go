package service

import (
	"errors"
	"fmt"
	"math"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	billingSubscriptionTablesOnce sync.Once
	billingSubscriptionTablesErr  error
)

func TestBillingSessionSubscriptionEstimateExceedsAvailableQuota(t *testing.T) {
	for _, weekly := range []bool{false, true} {
		t.Run(fmt.Sprint("weekly=", weekly), func(t *testing.T) {
			truncate(t)
			ensureBillingSubscriptionTables(t)
			t.Cleanup(func() {
				require.NoError(t, model.DB.Exec("DELETE FROM subscription_pre_consume_records").Error)
				require.NoError(t, model.DB.Exec("DELETE FROM subscription_plans").Error)
			})
			const userID, tokenID, planID, subscriptionID = 9401, 9402, 9403, 9404
			const available, estimate, actual = 28_900_632, 29_001_985, 126_490
			seedUser(t, userID, 1_593_829)
			seedToken(t, tokenID, userID, "partial-sub-token", estimate*2)
			seedBillingSubscription(t, planID, subscriptionID, userID, available, true)
			if weekly {
				require.NoError(t, model.DB.Model(&model.UserSubscription{}).Where("id = ?", subscriptionID).Updates(map[string]interface{}{
					"amount_total": 50_000_000, "weekly_amount": available, "weekly_reset_time": time.Now().Add(24 * time.Hour).Unix(),
				}).Error)
			}
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			info := billingSessionRelayInfo(userID, tokenID, "partial-sub-token", "subscription_first", "partial-sub-request")
			info.ForcePreConsume = false
			info.IsPlayground = false
			session, apiErr := NewBillingSession(ctx, info, estimate)
			require.Nil(t, apiErr)
			assert.Equal(t, BillingSourceSubscription, info.BillingSource)
			assert.Equal(t, available, session.GetPreConsumedQuota())
			assert.Equal(t, int64(available), getSubscriptionUsed(t, subscriptionID))
			assert.Equal(t, estimate*2-available, getTokenRemainQuota(t, tokenID))
			assert.Equal(t, 1_593_829, getUserQuota(t, userID))
			// A second in-flight request cannot spend the first request's reservation.
			second := billingSessionRelayInfo(userID, tokenID, "partial-sub-token", "subscription_only", "partial-sub-second")
			second.ForcePreConsume = false
			_, secondErr := NewBillingSession(ctx, second, estimate)
			require.NotNil(t, secondErr)
			require.NoError(t, session.Settle(actual))
			assert.Equal(t, int64(actual), getSubscriptionUsed(t, subscriptionID))
			assert.Equal(t, estimate*2-actual, getTokenRemainQuota(t, tokenID))
			assert.Equal(t, int64(actual-available), info.SubscriptionPostDelta)
			if weekly {
				var sub model.UserSubscription
				require.NoError(t, model.DB.First(&sub, subscriptionID).Error)
				assert.Equal(t, int64(actual), sub.WeeklyUsed)
			}
			// The returned quota is reusable; a failed next request refunds exactly
			// its capped reservation, including on a repeated refund attempt.
			next, nextErr := NewBillingSession(ctx, second, estimate)
			require.Nil(t, nextErr)
			assert.Equal(t, available-actual, next.GetPreConsumedQuota())
			require.Error(t, next.Settle(available))
			funding := next.funding.(*SubscriptionFunding)
			require.NoError(t, funding.Refund())
			require.NoError(t, funding.Refund())
			assert.Equal(t, int64(actual), getSubscriptionUsed(t, subscriptionID))
		})
	}
}

func TestBillingSessionSubscriptionReservationBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		force, fixed         bool
		estimate, tokenQuota int
		wantCode             types.ErrorCode
	}{
		{name: "async requires full cost", force: true, estimate: 80, tokenQuota: 100, wantCode: types.ErrorCodeInsufficientUserQuota},
		{name: "fixed price requires full cost", fixed: true, estimate: 80, tokenQuota: 100, wantCode: types.ErrorCodeInsufficientUserQuota},
		{name: "saturated estimate cannot bypass quota", estimate: math.MaxInt32, tokenQuota: math.MaxInt32, wantCode: types.ErrorCodeInsufficientUserQuota},
		{name: "token rejection refunds subscription", estimate: 80, tokenQuota: 20, wantCode: types.ErrorCodePreConsumeTokenQuotaFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			truncate(t)
			ensureBillingSubscriptionTables(t)
			t.Cleanup(func() {
				require.NoError(t, model.DB.Exec("DELETE FROM subscription_pre_consume_records").Error)
				require.NoError(t, model.DB.Exec("DELETE FROM subscription_plans").Error)
			})
			seedUser(t, 9411, 0)
			seedToken(t, 9412, 9411, "boundary-sub-token", tc.tokenQuota)
			seedBillingSubscription(t, 9413, 9414, 9411, 50, false)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			info := billingSessionRelayInfo(9411, 9412, "boundary-sub-token", "subscription_only", "boundary-request")
			info.ForcePreConsume = tc.force
			info.PriceData.UsePrice = tc.fixed
			info.IsPlayground = false
			_, apiErr := NewBillingSession(ctx, info, tc.estimate)
			require.NotNil(t, apiErr)
			assert.Equal(t, tc.wantCode, apiErr.GetErrorCode())
			assert.Zero(t, getSubscriptionUsed(t, 9414))
			assert.Equal(t, tc.tokenQuota, getTokenRemainQuota(t, 9412))
			if tc.wantCode == types.ErrorCodePreConsumeTokenQuotaFailed {
				require.NoError(t, model.RefundSubscriptionPreConsume("boundary-request"))
				assert.Zero(t, getSubscriptionUsed(t, 9414))
			}
		})
	}
}

func TestBillingSessionReserveWalletCannotOverdraw(t *testing.T) {
	truncate(t)
	const userID, tokenID = 9301, 9302
	seedUser(t, userID, 100)
	seedToken(t, tokenID, userID, "billing-wallet-token", 1_000)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := billingSessionRelayInfo(userID, tokenID, "billing-wallet-token", "wallet_only", "wallet-reserve")

	session, apiErr := NewBillingSession(ctx, info, 30)
	require.Nil(t, apiErr)
	require.Equal(t, 70, getUserQuota(t, userID))
	require.Equal(t, 1_000, getTokenRemainQuota(t, tokenID))

	err := session.Reserve(110)

	require.Error(t, err)
	var reserveErr *types.NewAPIError
	require.ErrorAs(t, err, &reserveErr)
	require.Equal(t, types.ErrorCodeInsufficientUserQuota, reserveErr.GetErrorCode())
	require.Equal(t, 30, session.GetPreConsumedQuota())
	require.Equal(t, 70, getUserQuota(t, userID))
	require.Equal(t, 1_000, getTokenRemainQuota(t, tokenID))
}

func TestBillingSessionReserveSubscriptionFirstFallsBackToWallet(t *testing.T) {
	truncate(t)
	ensureBillingSubscriptionTables(t)
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM subscription_pre_consume_records")
		model.DB.Exec("DELETE FROM subscription_plans")
	})
	const userID, tokenID, planID, subscriptionID = 9311, 9312, 9313, 9314
	seedUser(t, userID, 100)
	seedToken(t, tokenID, userID, "billing-sub-token", 1_000)
	seedBillingSubscription(t, planID, subscriptionID, userID, 50, true)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := billingSessionRelayInfo(userID, tokenID, "billing-sub-token", "subscription_first", "subscription-wallet-fallback")

	session, apiErr := NewBillingSession(ctx, info, 30)
	require.Nil(t, apiErr)
	require.Equal(t, int64(30), getSubscriptionUsed(t, subscriptionID))
	require.Equal(t, 100, getUserQuota(t, userID))

	require.NoError(t, session.Reserve(80))

	require.Equal(t, BillingSourceWallet, info.BillingSource)
	require.Equal(t, 80, session.GetPreConsumedQuota())
	require.Equal(t, int64(0), getSubscriptionUsed(t, subscriptionID))
	require.Equal(t, 20, getUserQuota(t, userID))
	require.Equal(t, 1_000, getTokenRemainQuota(t, tokenID))
	require.NoError(t, session.Settle(70))
	require.Equal(t, 30, getUserQuota(t, userID))
	require.Equal(t, 1_000, getTokenRemainQuota(t, tokenID))
}

func TestBillingSessionReserveSubscriptionFirstHonorsStrictPlan(t *testing.T) {
	truncate(t)
	ensureBillingSubscriptionTables(t)
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM subscription_pre_consume_records")
		model.DB.Exec("DELETE FROM subscription_plans")
	})
	const userID, tokenID, planID, subscriptionID = 9331, 9332, 9333, 9334
	seedUser(t, userID, 100)
	seedToken(t, tokenID, userID, "billing-strict-sub-token", 1_000)
	seedBillingSubscription(t, planID, subscriptionID, userID, 50, false)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := billingSessionRelayInfo(userID, tokenID, "billing-strict-sub-token", "subscription_first", "strict-subscription")

	session, apiErr := NewBillingSession(ctx, info, 30)
	require.Nil(t, apiErr)

	err := session.Reserve(80)

	require.Error(t, err)
	var reserveErr *types.NewAPIError
	require.ErrorAs(t, err, &reserveErr)
	require.Equal(t, types.ErrorCodeInsufficientUserQuota, reserveErr.GetErrorCode())
	require.Equal(t, BillingSourceSubscription, info.BillingSource)
	require.Equal(t, 30, session.GetPreConsumedQuota())
	require.Equal(t, int64(30), getSubscriptionUsed(t, subscriptionID))
	require.Equal(t, 100, getUserQuota(t, userID))
}

func TestBillingSessionSubscriptionReserveRefundsTotalIdempotently(t *testing.T) {
	truncate(t)
	ensureBillingSubscriptionTables(t)
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM subscription_pre_consume_records")
		model.DB.Exec("DELETE FROM subscription_plans")
	})
	const userID, tokenID, planID, subscriptionID = 9341, 9342, 9343, 9344
	const requestID = "subscription-total-refund"
	seedUser(t, userID, 100)
	seedToken(t, tokenID, userID, "billing-refund-sub-token", 1_000)
	seedBillingSubscription(t, planID, subscriptionID, userID, 100, false)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := billingSessionRelayInfo(userID, tokenID, "billing-refund-sub-token", "subscription_only", requestID)

	session, apiErr := NewBillingSession(ctx, info, 30)
	require.Nil(t, apiErr)
	require.NoError(t, session.Reserve(70))
	require.Equal(t, int64(70), getSubscriptionUsed(t, subscriptionID))

	var record model.SubscriptionPreConsumeRecord
	require.NoError(t, model.DB.Where("request_id = ?", requestID).First(&record).Error)
	require.Equal(t, int64(70), record.PreConsumed)

	funding := session.funding.(*SubscriptionFunding)
	require.NoError(t, funding.Refund())
	require.NoError(t, funding.Refund())
	require.Equal(t, int64(0), getSubscriptionUsed(t, subscriptionID))
	require.NoError(t, model.DB.Where("request_id = ?", requestID).First(&record).Error)
	require.Equal(t, "refunded", record.Status)
}

func TestBillingSessionTrustedWalletSettlementCannotOverdraw(t *testing.T) {
	truncate(t)
	const userID = 9321
	walletQuota := common.GetTrustQuota() + 10
	seedUser(t, userID, walletQuota)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenUnlimited:  true,
		OriginModelName: "deepseek-main",
		RequestId:       "trusted-wallet",
		UserSetting:     dto.UserSetting{BillingPreference: "wallet_only"},
	}

	session, apiErr := NewBillingSession(ctx, info, 30)
	require.Nil(t, apiErr)
	require.Zero(t, session.GetPreConsumedQuota())
	require.NoError(t, session.Reserve(walletQuota+1))
	require.Equal(t, walletQuota, getUserQuota(t, userID))

	err := session.Settle(walletQuota + 1)

	require.Error(t, err)
	require.True(t, errors.Is(err, model.ErrInsufficientUserQuota))
	require.Equal(t, walletQuota, getUserQuota(t, userID))
}

func billingSessionRelayInfo(userID, tokenID int, tokenKey, preference, requestID string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        tokenKey,
		OriginModelName: "deepseek-main",
		RequestId:       requestID,
		ForcePreConsume: true,
		IsPlayground:    true,
		UserSetting:     dto.UserSetting{BillingPreference: preference},
	}
}

func seedBillingSubscription(t *testing.T, planID, subscriptionID, userID int, total int64, allowWalletOverflow bool) {
	t.Helper()
	plan := &model.SubscriptionPlan{
		Id:                  planID,
		Title:               "billing-test-plan",
		Enabled:             true,
		QuotaResetPeriod:    model.SubscriptionResetNever,
		AllowWalletOverflow: common.GetPointer(allowWalletOverflow),
	}
	require.NoError(t, model.DB.Create(plan).Error)
	subscription := &model.UserSubscription{
		Id:                  subscriptionID,
		UserId:              userID,
		PlanId:              planID,
		AmountTotal:         total,
		Status:              "active",
		StartTime:           time.Now().Add(-time.Hour).Unix(),
		EndTime:             time.Now().Add(24 * time.Hour).Unix(),
		AllowWalletOverflow: allowWalletOverflow,
	}
	require.NoError(t, model.DB.Create(subscription).Error)
}

func ensureBillingSubscriptionTables(t *testing.T) {
	t.Helper()
	billingSubscriptionTablesOnce.Do(func() {
		billingSubscriptionTablesErr = model.DB.AutoMigrate(&model.SubscriptionPlan{}, &model.SubscriptionPreConsumeRecord{})
	})
	require.NoError(t, billingSubscriptionTablesErr)
}
