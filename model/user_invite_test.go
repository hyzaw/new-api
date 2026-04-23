package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertIncreasesAffCountWhenInviterRewardIsZero(t *testing.T) {
	truncateTables(t)

	originalQuotaForNewUser := common.QuotaForNewUser
	originalQuotaForInvitee := common.QuotaForInvitee
	originalQuotaForInviter := common.QuotaForInviter
	t.Cleanup(func() {
		common.QuotaForNewUser = originalQuotaForNewUser
		common.QuotaForInvitee = originalQuotaForInvitee
		common.QuotaForInviter = originalQuotaForInviter
	})

	common.QuotaForNewUser = 0
	common.QuotaForInvitee = 0
	common.QuotaForInviter = 0

	inviter := &User{
		Username:    "inviter_count_only",
		Password:    "password123",
		DisplayName: "inviter_count_only",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "aff_count_only",
	}
	require.NoError(t, DB.Create(inviter).Error)

	invitee := &User{
		Username:    "invitee_count_only",
		Password:    "password123",
		DisplayName: "invitee_count_only",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, invitee.Insert(inviter.Id))

	var reloadedInviter User
	require.NoError(t, DB.First(&reloadedInviter, inviter.Id).Error)
	assert.Equal(t, 1, reloadedInviter.AffCount)
	assert.Equal(t, 0, reloadedInviter.AffQuota)
	assert.Equal(t, 0, reloadedInviter.AffHistoryQuota)

	records, err := GetInviteWalletRecordsByUserId(inviter.Id)
	require.NoError(t, err)
	assert.Len(t, records, 0)
}

func TestInsertGrantsNewUserQuotaAsGiftQuota(t *testing.T) {
	truncateTables(t)

	originalQuotaForNewUser := common.QuotaForNewUser
	originalQuotaForInvitee := common.QuotaForInvitee
	originalQuotaForInviter := common.QuotaForInviter
	t.Cleanup(func() {
		common.QuotaForNewUser = originalQuotaForNewUser
		common.QuotaForInvitee = originalQuotaForInvitee
		common.QuotaForInviter = originalQuotaForInviter
	})

	common.QuotaForNewUser = 100
	common.QuotaForInvitee = 0
	common.QuotaForInviter = 0

	user := &User{
		Username:    "new_user_gift_quota",
		Password:    "password123",
		DisplayName: "new_user_gift_quota",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, user.Insert(0))

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	assert.Equal(t, 0, reloaded.Quota)
	assert.Equal(t, 100, reloaded.GiftQuota)
}

func TestInsertGrantsInviteeQuotaAsGiftQuota(t *testing.T) {
	truncateTables(t)

	originalQuotaForNewUser := common.QuotaForNewUser
	originalQuotaForInvitee := common.QuotaForInvitee
	originalQuotaForInviter := common.QuotaForInviter
	t.Cleanup(func() {
		common.QuotaForNewUser = originalQuotaForNewUser
		common.QuotaForInvitee = originalQuotaForInvitee
		common.QuotaForInviter = originalQuotaForInviter
	})

	common.QuotaForNewUser = 100
	common.QuotaForInvitee = 50
	common.QuotaForInviter = 0

	inviter := &User{
		Username:    "gift_inviter",
		Password:    "password123",
		DisplayName: "gift_inviter",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "gift_aff",
	}
	require.NoError(t, DB.Create(inviter).Error)

	invitee := &User{
		Username:    "gift_invitee",
		Password:    "password123",
		DisplayName: "gift_invitee",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, invitee.Insert(inviter.Id))

	var reloadedInvitee User
	require.NoError(t, DB.First(&reloadedInvitee, invitee.Id).Error)
	assert.Equal(t, 0, reloadedInvitee.Quota)
	assert.Equal(t, 150, reloadedInvitee.GiftQuota)
}

func TestInsertGrantsInviterRewardAsGiftQuota(t *testing.T) {
	truncateTables(t)

	originalQuotaForNewUser := common.QuotaForNewUser
	originalQuotaForInvitee := common.QuotaForInvitee
	originalQuotaForInviter := common.QuotaForInviter
	t.Cleanup(func() {
		common.QuotaForNewUser = originalQuotaForNewUser
		common.QuotaForInvitee = originalQuotaForInvitee
		common.QuotaForInviter = originalQuotaForInviter
	})

	common.QuotaForNewUser = 0
	common.QuotaForInvitee = 0
	common.QuotaForInviter = 75

	inviter := &User{
		Username:    "inviter_gift_reward",
		Password:    "password123",
		DisplayName: "inviter_gift_reward",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "inviter_gift_reward_aff",
	}
	require.NoError(t, DB.Create(inviter).Error)

	invitee := &User{
		Username:    "invitee_for_inviter_gift",
		Password:    "password123",
		DisplayName: "invitee_for_inviter_gift",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, invitee.Insert(inviter.Id))

	var reloadedInviter User
	require.NoError(t, DB.First(&reloadedInviter, inviter.Id).Error)
	assert.Equal(t, 1, reloadedInviter.AffCount)
	assert.Equal(t, 0, reloadedInviter.AffQuota)
	assert.Equal(t, 0, reloadedInviter.AffHistoryQuota)
	assert.Equal(t, 75, reloadedInviter.GiftQuota)

	records, err := GetInviteWalletRecordsByUserId(inviter.Id)
	require.NoError(t, err)
	assert.Len(t, records, 0)
}

func TestFinalizeOAuthUserCreationIncreasesAffCountWhenInviterRewardIsZero(t *testing.T) {
	truncateTables(t)

	originalQuotaForInviter := common.QuotaForInviter
	t.Cleanup(func() {
		common.QuotaForInviter = originalQuotaForInviter
	})
	common.QuotaForInviter = 0

	inviter := &User{
		Username:    "oauth_inviter_count_only",
		Password:    "password123",
		DisplayName: "oauth_inviter_count_only",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "oauth_aff_count_only",
	}
	require.NoError(t, DB.Create(inviter).Error)

	user := &User{
		Username:    "oauth_invitee_count_only",
		DisplayName: "oauth_invitee_count_only",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, user.InsertWithTx(DB, inviter.Id))

	user.FinalizeOAuthUserCreation(inviter.Id)

	var reloadedInviter User
	require.NoError(t, DB.First(&reloadedInviter, inviter.Id).Error)
	assert.Equal(t, 1, reloadedInviter.AffCount)
	assert.Equal(t, 0, reloadedInviter.AffQuota)
	assert.Equal(t, 0, reloadedInviter.AffHistoryQuota)

	records, err := GetInviteWalletRecordsByUserId(inviter.Id)
	require.NoError(t, err)
	assert.Len(t, records, 0)
}

func TestFinalizeOAuthUserCreationGrantsInviteeQuotaAsGiftQuota(t *testing.T) {
	truncateTables(t)

	originalQuotaForNewUser := common.QuotaForNewUser
	originalQuotaForInvitee := common.QuotaForInvitee
	originalQuotaForInviter := common.QuotaForInviter
	t.Cleanup(func() {
		common.QuotaForNewUser = originalQuotaForNewUser
		common.QuotaForInvitee = originalQuotaForInvitee
		common.QuotaForInviter = originalQuotaForInviter
	})

	common.QuotaForNewUser = 100
	common.QuotaForInvitee = 50
	common.QuotaForInviter = 0

	inviter := &User{
		Username:    "oauth_gift_inviter",
		Password:    "password123",
		DisplayName: "oauth_gift_inviter",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "oauth_gift_aff",
	}
	require.NoError(t, DB.Create(inviter).Error)

	user := &User{
		Username:    "oauth_gift_invitee",
		DisplayName: "oauth_gift_invitee",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, user.InsertWithTx(DB, inviter.Id))

	user.FinalizeOAuthUserCreation(inviter.Id)

	var reloadedInvitee User
	require.NoError(t, DB.First(&reloadedInvitee, user.Id).Error)
	assert.Equal(t, 0, reloadedInvitee.Quota)
	assert.Equal(t, 150, reloadedInvitee.GiftQuota)
}
