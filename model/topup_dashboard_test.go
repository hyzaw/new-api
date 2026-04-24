package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAdminTopUpDashboardStatsValuableUsersSortedByCumulativeTopUp(t *testing.T) {
	truncateTables(t)

	userA := &User{
		Username:    "valuable_user_a",
		Password:    "password123",
		DisplayName: "用户A",
		AffCode:     "topup_dash_a",
		Quota:       500,
		GiftQuota:   120,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	userB := &User{
		Username:    "valuable_user_b",
		Password:    "password123",
		DisplayName: "用户B",
		AffCode:     "topup_dash_b",
		Quota:       800,
		GiftQuota:   60,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	userC := &User{
		Username:    "valuable_user_c",
		Password:    "password123",
		DisplayName: "用户C",
		AffCode:     "topup_dash_c",
		Quota:       100,
		GiftQuota:   20,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, DB.Create(userA).Error)
	require.NoError(t, DB.Create(userB).Error)
	require.NoError(t, DB.Create(userC).Error)

	topUps := []*TopUp{
		{
			UserId:        userA.Id,
			Money:         120,
			TradeNo:       "dashboard_topup_a_1",
			PaymentMethod: PaymentMethodStripe,
			CreateTime:    1713800000,
			CompleteTime:  1713800100,
			Status:        common.TopUpStatusSuccess,
		},
		{
			UserId:        userA.Id,
			Money:         30,
			TradeNo:       "dashboard_topup_a_2",
			PaymentMethod: PaymentMethodStripe,
			CreateTime:    1713800200,
			CompleteTime:  1713800300,
			Status:        common.TopUpStatusSuccess,
		},
		{
			UserId:        userB.Id,
			Money:         260,
			TradeNo:       "dashboard_topup_b_1",
			PaymentMethod: PaymentMethodStripe,
			CreateTime:    1713800400,
			CompleteTime:  1713800500,
			Status:        common.TopUpStatusSuccess,
		},
		{
			UserId:        userB.Id,
			Money:         999,
			TradeNo:       "dashboard_topup_b_pending",
			PaymentMethod: PaymentMethodStripe,
			CreateTime:    1713800600,
			Status:        common.TopUpStatusPending,
		},
		{
			UserId:        userC.Id,
			Money:         80,
			TradeNo:       "dashboard_topup_c_1",
			PaymentMethod: PaymentMethodStripe,
			CreateTime:    1713800700,
			CompleteTime:  1713800800,
			Status:        common.TopUpStatusSuccess,
		},
	}
	for _, topUp := range topUps {
		require.NoError(t, DB.Create(topUp).Error)
	}

	stats, err := GetAdminTopUpDashboardStats(30)
	require.NoError(t, err)
	require.Len(t, stats.ValuableUsers, 3)
	assert.EqualValues(t, 3, stats.Overview.PaidUserCount)
	assert.EqualValues(t, 3, stats.Overview.TotalUserCount)
	assert.EqualValues(t, 1400, stats.Overview.TotalUserQuota)
	assert.EqualValues(t, 200, stats.Overview.TotalUserGiftQuota)

	assert.Equal(t, userB.Id, stats.ValuableUsers[0].UserId)
	assert.Equal(t, "valuable_user_b", stats.ValuableUsers[0].Username)
	assert.Equal(t, "用户B", stats.ValuableUsers[0].DisplayName)
	assert.Equal(t, 260.0, stats.ValuableUsers[0].TotalMoney)
	assert.EqualValues(t, 1, stats.ValuableUsers[0].SuccessOrders)
	assert.EqualValues(t, 1713800500, stats.ValuableUsers[0].LastTopUpTime)

	assert.Equal(t, userA.Id, stats.ValuableUsers[1].UserId)
	assert.Equal(t, 150.0, stats.ValuableUsers[1].TotalMoney)
	assert.EqualValues(t, 2, stats.ValuableUsers[1].SuccessOrders)

	assert.Equal(t, userC.Id, stats.ValuableUsers[2].UserId)
	assert.Equal(t, 80.0, stats.ValuableUsers[2].TotalMoney)
}
