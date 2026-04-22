package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type BatchUpdateUserGroupResult struct {
	TargetGroup string `json:"target_group"`
	UserCount   int64  `json:"user_count"`
}

type MigrateUserGroupResult struct {
	SourceGroup string `json:"source_group"`
	TargetGroup string `json:"target_group"`
	UserCount   int64  `json:"user_count"`
}

func NormalizeUserIDs(userIDs []int) []int {
	seen := make(map[int]struct{}, len(userIDs))
	result := make([]int, 0, len(userIDs))
	for _, userID := range userIDs {
		if userID <= 0 {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		result = append(result, userID)
	}
	return result
}

func GetActiveUsersByIDs(userIDs []int) ([]User, error) {
	normalizedIDs := NormalizeUserIDs(userIDs)
	if len(normalizedIDs) == 0 {
		return nil, errors.New("用户不能为空")
	}

	users := make([]User, 0, len(normalizedIDs))
	if err := DB.Select("id", "username", "role", commonGroupCol).
		Where("id IN ?", normalizedIDs).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func GetActiveUsersByGroup(group string) ([]User, error) {
	group = strings.TrimSpace(group)
	if group == "" {
		return nil, errors.New("来源分组不能为空")
	}

	users := make([]User, 0)
	if err := DB.Select("id", "username", "role", commonGroupCol).
		Where(commonGroupCol+" = ?", group).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func BatchUpdateUserGroup(userIDs []int, targetGroup string) (*BatchUpdateUserGroupResult, error) {
	normalizedIDs := NormalizeUserIDs(userIDs)
	targetGroup = strings.TrimSpace(targetGroup)

	if len(normalizedIDs) == 0 {
		return nil, errors.New("请至少选择一个用户")
	}
	if targetGroup == "" {
		return nil, errors.New("目标分组不能为空")
	}

	updateResult := DB.Model(&User{}).
		Where("id IN ?", normalizedIDs).
		Update("group", targetGroup)
	if updateResult.Error != nil {
		return nil, updateResult.Error
	}

	for _, userID := range normalizedIDs {
		if err := InvalidateUserCache(userID); err != nil {
			common.SysLog("failed to invalidate user cache after batch group update: " + err.Error())
		}
	}

	return &BatchUpdateUserGroupResult{
		TargetGroup: targetGroup,
		UserCount:   updateResult.RowsAffected,
	}, nil
}

func MigrateUserGroup(sourceGroup string, targetGroup string) (*MigrateUserGroupResult, error) {
	sourceGroup = strings.TrimSpace(sourceGroup)
	targetGroup = strings.TrimSpace(targetGroup)

	if sourceGroup == "" || targetGroup == "" {
		return nil, errors.New("来源分组和目标分组不能为空")
	}
	if sourceGroup == targetGroup {
		return nil, errors.New("来源分组和目标分组不能相同")
	}

	users, err := GetActiveUsersByGroup(sourceGroup)
	if err != nil {
		return nil, err
	}

	result := &MigrateUserGroupResult{
		SourceGroup: sourceGroup,
		TargetGroup: targetGroup,
		UserCount:   int64(len(users)),
	}
	if len(users) == 0 {
		return result, nil
	}

	updateResult := DB.Model(&User{}).
		Where(commonGroupCol+" = ?", sourceGroup).
		Update("group", targetGroup)
	if updateResult.Error != nil {
		return nil, updateResult.Error
	}

	for _, user := range users {
		if err := InvalidateUserCache(user.Id); err != nil {
			common.SysLog("failed to invalidate user cache after group migration: " + err.Error())
		}
	}

	result.UserCount = updateResult.RowsAffected
	return result, nil
}
