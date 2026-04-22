package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func GetGroups(c *gin.Context) {
	groupNames := make([]string, 0)
	for groupName := range ratio_setting.GetGroupRatioCopy() {
		groupNames = append(groupNames, groupName)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    groupNames,
	})
}

func GetUserGroups(c *gin.Context) {
	usableGroups := make(map[string]map[string]interface{})
	userGroup := ""
	userId := c.GetInt("id")
	userGroup, _ = model.GetUserGroup(userId, false)
	userUsableGroups := service.GetUserUsableGroups(userGroup)
	for groupName, _ := range ratio_setting.GetGroupRatioCopy() {
		// UserUsableGroups contains the groups that the user can use
		if desc, ok := userUsableGroups[groupName]; ok {
			usableGroups[groupName] = map[string]interface{}{
				"ratio": service.GetUserGroupRatio(userGroup, groupName),
				"desc":  desc,
			}
		}
	}
	if _, ok := userUsableGroups["auto"]; ok {
		usableGroups["auto"] = map[string]interface{}{
			"ratio": "自动",
			"desc":  setting.GetUsableGroupDescription("auto"),
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usableGroups,
	})
}

type GroupMigrationRequest struct {
	SourceGroup string `json:"source_group"`
	TargetGroup string `json:"target_group"`
}

func MigrateGroupUsers(c *gin.Context) {
	var req GroupMigrationRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("无效的参数"))
		return
	}

	req.SourceGroup = strings.TrimSpace(req.SourceGroup)
	req.TargetGroup = strings.TrimSpace(req.TargetGroup)

	if req.SourceGroup == "" || req.TargetGroup == "" {
		common.ApiError(c, errors.New("来源分组和目标分组不能为空"))
		return
	}
	if req.SourceGroup == req.TargetGroup {
		common.ApiError(c, errors.New("来源分组和目标分组不能相同"))
		return
	}
	if !ratio_setting.ContainsGroupRatio(req.SourceGroup) {
		common.ApiError(c, errors.New("来源分组不存在"))
		return
	}
	if !ratio_setting.ContainsGroupRatio(req.TargetGroup) {
		common.ApiError(c, errors.New("目标分组不存在"))
		return
	}

	result, err := model.MigrateUsersAndTokensGroup(req.SourceGroup, req.TargetGroup)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	model.RecordLog(
		c.GetInt("id"),
		model.LogTypeManage,
		fmt.Sprintf(
			"批量切换分组：%s -> %s，影响用户 %d 个，令牌 %d 个",
			req.SourceGroup,
			req.TargetGroup,
			result.UserCount,
			result.TokenCount,
		),
	)

	common.ApiSuccess(c, result)
}
