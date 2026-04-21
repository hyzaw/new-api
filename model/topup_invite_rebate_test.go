package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManualCompleteTopUpAppliesInviteRebate(t *testing.T) {
	truncateTables(t)

	originalRatio := common.TopUpAffRatio
	t.Cleanup(func() {
		common.TopUpAffRatio = originalRatio
	})
	common.TopUpAffRatio = 10

	inviter := &User{Username: "inviter", Password: "password123", DisplayName: "inviter", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "aff_inviter_1"}
	invitee := &User{Username: "invitee", Password: "password123", DisplayName: "invitee", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, InviterId: 1, AffCode: "aff_invitee_1"}
	require.NoError(t, DB.Create(inviter).Error)
	invitee.InviterId = inviter.Id
	require.NoError(t, DB.Create(invitee).Error)

	topUp := &TopUp{
		UserId:        invitee.Id,
		Amount:        10,
		Money:         10,
		TradeNo:       "invite_topup_complete",
		PaymentMethod: "alipay_f2f",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(topUp).Error)

	completed, err := ManualCompleteTopUp(topUp.TradeNo, "127.0.0.1")
	require.NoError(t, err)
	require.True(t, completed)

	reloadedTopUp := GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, reloadedTopUp)
	expectedQuota := getTopUpGrantedQuota(reloadedTopUp)
	expectedRebate := calculateInviteTopUpRebateQuota(expectedQuota)

	var reloadedInvitee User
	require.NoError(t, DB.First(&reloadedInvitee, invitee.Id).Error)
	assert.Equal(t, expectedQuota, reloadedInvitee.Quota)

	var reloadedInviter User
	require.NoError(t, DB.First(&reloadedInviter, inviter.Id).Error)
	assert.Equal(t, expectedRebate, reloadedInviter.AffQuota)
	assert.Equal(t, expectedRebate, reloadedInviter.AffHistoryQuota)
	assert.Equal(t, inviter.Id, reloadedTopUp.InviteRebateUserId)
	assert.Equal(t, expectedRebate, reloadedTopUp.InviteRebateQuota)
	assert.Equal(t, 10.0, reloadedTopUp.InviteRebateRatio)
	assert.NotZero(t, reloadedTopUp.InviteRebateTime)

	completedAgain, err := ManualCompleteTopUp(topUp.TradeNo, "127.0.0.1")
	require.NoError(t, err)
	assert.False(t, completedAgain)

	var inviterAfterSecondCall User
	require.NoError(t, DB.First(&inviterAfterSecondCall, inviter.Id).Error)
	assert.Equal(t, expectedRebate, inviterAfterSecondCall.AffQuota)
}

func TestManualRefundRollsBackInviteRebate(t *testing.T) {
	truncateTables(t)

	originalRatio := common.TopUpAffRatio
	t.Cleanup(func() {
		common.TopUpAffRatio = originalRatio
	})
	common.TopUpAffRatio = 10

	inviter := &User{Username: "refund_inviter", Password: "password123", DisplayName: "refund_inviter", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "aff_inviter_2"}
	require.NoError(t, DB.Create(inviter).Error)

	invitee := &User{Username: "refund_invitee", Password: "password123", DisplayName: "refund_invitee", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, InviterId: inviter.Id, AffCode: "aff_invitee_2"}
	require.NoError(t, DB.Create(invitee).Error)

	topUp := &TopUp{
		UserId:        invitee.Id,
		Amount:        10,
		Money:         10,
		TradeNo:       "invite_topup_refund",
		PaymentMethod: "alipay_f2f",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(topUp).Error)

	completed, err := ManualCompleteTopUp(topUp.TradeNo, "127.0.0.1")
	require.NoError(t, err)
	require.True(t, completed)

	reloadedTopUp := GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, reloadedTopUp)
	totalQuota := getTopUpGrantedQuota(reloadedTopUp)
	totalRebate := reloadedTopUp.InviteRebateQuota

	refund, err := MarkTopUpRefundManual(reloadedTopUp.Id, "5.00", "test refund", 1, "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, refund)

	var reloadedInvitee User
	require.NoError(t, DB.First(&reloadedInvitee, invitee.Id).Error)
	assert.Equal(t, totalQuota-refund.QuotaDelta, reloadedInvitee.Quota)

	var reloadedInviter User
	require.NoError(t, DB.First(&reloadedInviter, inviter.Id).Error)
	assert.Equal(t, totalRebate-refund.InviteRebateDelta, reloadedInviter.AffQuota)
	assert.Equal(t, totalRebate-refund.InviteRebateDelta, reloadedInviter.AffHistoryQuota)

	reloadedTopUp = GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, reloadedTopUp)
	assert.Equal(t, refund.InviteRebateDelta, reloadedTopUp.InviteRebateRefundedQuota)
	assert.Equal(t, totalRebate/2, refund.InviteRebateDelta)
}

func TestManualRefundDeductsMainQuotaWhenInviteQuotaTransferred(t *testing.T) {
	truncateTables(t)

	originalRatio := common.TopUpAffRatio
	t.Cleanup(func() {
		common.TopUpAffRatio = originalRatio
	})
	common.TopUpAffRatio = 10

	inviter := &User{Username: "transfer_inviter", Password: "password123", DisplayName: "transfer_inviter", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "aff_inviter_3"}
	require.NoError(t, DB.Create(inviter).Error)

	invitee := &User{Username: "transfer_invitee", Password: "password123", DisplayName: "transfer_invitee", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, InviterId: inviter.Id, AffCode: "aff_invitee_3"}
	require.NoError(t, DB.Create(invitee).Error)

	topUp := &TopUp{
		UserId:        invitee.Id,
		Amount:        10,
		Money:         10,
		TradeNo:       "invite_topup_transfer_refund",
		PaymentMethod: "alipay_f2f",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(topUp).Error)

	completed, err := ManualCompleteTopUp(topUp.TradeNo, "127.0.0.1")
	require.NoError(t, err)
	require.True(t, completed)

	reloadedTopUp := GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, reloadedTopUp)
	totalRebate := reloadedTopUp.InviteRebateQuota

	var reloadedInviter User
	require.NoError(t, DB.First(&reloadedInviter, inviter.Id).Error)
	require.NoError(t, reloadedInviter.TransferAffQuotaToQuota(totalRebate))

	refund, err := MarkTopUpRefundManual(reloadedTopUp.Id, "5.00", "test refund after transfer", 1, "127.0.0.1")
	require.NoError(t, err)

	require.NoError(t, DB.First(&reloadedInviter, inviter.Id).Error)
	assert.Equal(t, 0, reloadedInviter.AffQuota)
	assert.Equal(t, totalRebate-refund.InviteRebateDelta, reloadedInviter.Quota)
	assert.Equal(t, totalRebate-refund.InviteRebateDelta, reloadedInviter.AffHistoryQuota)
}
