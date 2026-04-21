package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackfillInviteDetails(t *testing.T) {
	truncateTables(t)

	inviter := &User{
		Username:    "detail_inviter",
		Password:    "password123",
		DisplayName: "detail_inviter",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "detail_aff_code",
	}
	require.NoError(t, DB.Create(inviter).Error)

	invitee := &User{
		Username:    "detail_invitee",
		Password:    "password123",
		DisplayName: "detail_invitee",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		InviterId:   inviter.Id,
		AffCode:     "detail_aff_code_invitee",
	}
	require.NoError(t, DB.Create(invitee).Error)

	RecordLog(invitee.Id, LogTypeSystem, "使用邀请码赠送 100")

	topUp := &TopUp{
		UserId:                    invitee.Id,
		Amount:                    20,
		Money:                     20,
		TradeNo:                   "detail_rebate_trade",
		PaymentMethod:             "alipay_f2f",
		CreateTime:                common.GetTimestamp(),
		Status:                    common.TopUpStatusSuccess,
		InviteRebateUserId:        inviter.Id,
		InviteRebateRatio:         10,
		InviteRebateQuota:         200000,
		InviteRebateRefundedQuota: 50000,
		InviteRebateTime:          common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(topUp).Error)

	require.NoError(t, BackfillInviteDetails())

	inviteRecords, err := GetInviteRecordsByInviterId(inviter.Id)
	require.NoError(t, err)
	require.Len(t, inviteRecords, 1)
	assert.Equal(t, invitee.Id, inviteRecords[0].InviteeId)
	assert.Equal(t, invitee.Username, inviteRecords[0].InviteeUsername)
	assert.NotZero(t, inviteRecords[0].InviteTime)

	rebateRecords, err := GetInviteRebateRecordsByInviterId(inviter.Id)
	require.NoError(t, err)
	require.Len(t, rebateRecords, 1)
	assert.Equal(t, topUp.TradeNo, rebateRecords[0].TopUpTradeNo)
	assert.Equal(t, topUp.InviteRebateQuota, rebateRecords[0].RebateQuota)
	assert.Equal(t, topUp.InviteRebateRefundedQuota, rebateRecords[0].RebateRefundedQuota)
	assert.Equal(t, invitee.Username, rebateRecords[0].InviteeUsername)
}
