package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
)

type GroupMigrationResult struct {
	SourceGroup string `json:"source_group"`
	TargetGroup string `json:"target_group"`
	UserCount   int64  `json:"user_count"`
	TokenCount  int64  `json:"token_count"`
}

func MigrateTokenGroup(sourceGroup, targetGroup string) (*GroupMigrationResult, error) {
	if sourceGroup == "" || targetGroup == "" {
		return nil, errors.New("分组不能为空")
	}
	if sourceGroup == targetGroup {
		return nil, errors.New("来源分组和目标分组不能相同")
	}

	result := &GroupMigrationResult{
		SourceGroup: sourceGroup,
		TargetGroup: targetGroup,
	}

	var tokenKeys []string

	tx := DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&Token{}).
		Where(commonGroupCol+" = ?", sourceGroup).
		Pluck(commonKeyCol, &tokenKeys).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if len(tokenKeys) == 0 {
		if err := tx.Rollback().Error; err != nil {
			return nil, err
		}
		return result, nil
	}

	tokenUpdate := tx.Model(&Token{}).
		Where(commonGroupCol+" = ?", sourceGroup).
		Update("group", targetGroup)
	if tokenUpdate.Error != nil {
		tx.Rollback()
		return nil, tokenUpdate.Error
	}
	result.TokenCount = tokenUpdate.RowsAffected

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	if err := InvalidateTokensCacheByKeys(tokenKeys); err != nil {
		common.SysLog("failed to invalidate migrated token cache: " + err.Error())
	}

	return result, nil
}
