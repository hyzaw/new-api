package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInviteWithdrawalAndRejectRestoresAffQuota(t *testing.T) {
	truncateTables(t)

	user := &User{
		Username:    "withdraw_user",
		Password:    "password123",
		DisplayName: "withdraw_user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "withdraw_aff_code",
		AffQuota:    int(common.QuotaPerUnit * 100),
	}
	require.NoError(t, DB.Create(user).Error)

	withdrawal, err := CreateInviteWithdrawal(user.Id, "20.00", "data:image/png;base64,ZmFrZQ==", "test")
	require.NoError(t, err)
	require.NotNil(t, withdrawal)
	assert.Equal(t, InviteWithdrawalStatusPending, withdrawal.Status)

	var reloadedUser User
	require.NoError(t, DB.First(&reloadedUser, user.Id).Error)
	assert.Equal(t, user.AffQuota-withdrawal.Quota, reloadedUser.AffQuota)

	reviewed, err := ReviewInviteWithdrawal(withdrawal.Id, InviteWithdrawalStatusRejected, "reject", 1, "admin")
	require.NoError(t, err)
	assert.Equal(t, InviteWithdrawalStatusRejected, reviewed.Status)
	assert.Equal(t, "admin", reviewed.OperatorName)

	require.NoError(t, DB.First(&reloadedUser, user.Id).Error)
	assert.Equal(t, user.AffQuota, reloadedUser.AffQuota)

	records, err := GetInviteWalletRecordsByUserId(user.Id)
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, InviteWalletChangeTypeWithdrawalRejectReturn, records[0].ChangeType)
	assert.Equal(t, withdrawal.Quota, records[0].AffQuotaDelta)
	assert.Equal(t, InviteWalletChangeTypeWithdrawalApply, records[1].ChangeType)
}

func TestCreateInviteWithdrawalAndPayKeepsAffQuotaDeducted(t *testing.T) {
	truncateTables(t)

	user := &User{
		Username:    "withdraw_paid_user",
		Password:    "password123",
		DisplayName: "withdraw_paid_user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "withdraw_paid_aff_code",
		AffQuota:    int(common.QuotaPerUnit * 100),
	}
	require.NoError(t, DB.Create(user).Error)

	withdrawal, err := CreateInviteWithdrawal(user.Id, "20.00", "data:image/png;base64,ZmFrZQ==", "")
	require.NoError(t, err)
	require.NotNil(t, withdrawal)

	reviewed, err := ReviewInviteWithdrawal(withdrawal.Id, InviteWithdrawalStatusPaid, "paid", 1, "admin")
	require.NoError(t, err)
	assert.Equal(t, InviteWithdrawalStatusPaid, reviewed.Status)

	var reloadedUser User
	require.NoError(t, DB.First(&reloadedUser, user.Id).Error)
	assert.Equal(t, user.AffQuota-withdrawal.Quota, reloadedUser.AffQuota)

	records, err := GetInviteWalletRecordsByUserId(user.Id)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, InviteWalletChangeTypeWithdrawalApply, records[0].ChangeType)
	assert.Equal(t, -withdrawal.Quota, records[0].AffQuotaDelta)
}
