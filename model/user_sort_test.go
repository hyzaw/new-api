package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllUsersSortsByRemainingAndTotalBalance(t *testing.T) {
	truncateTables(t)

	users := []*User{
		{
			Username:    "balance_sort_a",
			Password:    "password123",
			Quota:       100,
			GiftQuota:   0,
			UsedQuota:   100,
			Role:        common.RoleCommonUser,
			Status:      common.UserStatusEnabled,
			AffCode:     "balance_sort_a",
			InviterId:   0,
			DisplayName: "balance_sort_a",
		},
		{
			Username:    "balance_sort_b",
			Password:    "password123",
			Quota:       20,
			GiftQuota:   200,
			UsedQuota:   10,
			Role:        common.RoleCommonUser,
			Status:      common.UserStatusEnabled,
			AffCode:     "balance_sort_b",
			InviterId:   0,
			DisplayName: "balance_sort_b",
		},
		{
			Username:    "balance_sort_c",
			Password:    "password123",
			Quota:       300,
			GiftQuota:   0,
			UsedQuota:   0,
			Role:        common.RoleCommonUser,
			Status:      common.UserStatusEnabled,
			AffCode:     "balance_sort_c",
			InviterId:   0,
			DisplayName: "balance_sort_c",
		},
	}
	for _, user := range users {
		require.NoError(t, DB.Create(user).Error)
	}

	pageInfo := &common.PageInfo{Page: 1, PageSize: 20}

	remainingDesc, _, err := GetAllUsers(pageInfo, "remaining_balance_desc")
	require.NoError(t, err)
	require.Len(t, remainingDesc, 3)
	assert.Equal(t, "balance_sort_c", remainingDesc[0].Username)
	assert.Equal(t, "balance_sort_b", remainingDesc[1].Username)
	assert.Equal(t, "balance_sort_a", remainingDesc[2].Username)

	totalAsc, _, err := GetAllUsers(pageInfo, "total_balance_asc")
	require.NoError(t, err)
	require.Len(t, totalAsc, 3)
	assert.Equal(t, "balance_sort_a", totalAsc[0].Username)
	assert.Equal(t, "balance_sort_b", totalAsc[1].Username)
	assert.Equal(t, "balance_sort_c", totalAsc[2].Username)
}
