package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type CreateInviteWithdrawalRequest struct {
	Amount      string `json:"amount"`
	ReceiptCode string `json:"receipt_code"`
	UserRemark  string `json:"user_remark"`
}

type ReviewInviteWithdrawalRequest struct {
	Action      string `json:"action"`
	AdminRemark string `json:"admin_remark"`
}

func CreateInviteWithdrawal(c *gin.Context) {
	var req CreateInviteWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	withdrawal, err := model.CreateInviteWithdrawal(c.GetInt("id"), req.Amount, req.ReceiptCode, req.UserRemark)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, withdrawal)
}

func GetUserInviteWithdrawals(c *gin.Context) {
	withdrawals, err := model.GetInviteWithdrawalsByUserId(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"items": withdrawals,
	})
}

func GetAllInviteWithdrawals(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.GetInviteWithdrawals(pageInfo, c.Query("keyword"), c.Query("status"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func ReviewInviteWithdrawal(c *gin.Context) {
	var req ReviewInviteWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	id := common.String2Int(strings.TrimSpace(c.Param("id")))
	if id == 0 {
		common.ApiErrorMsg(c, "提现申请不存在")
		return
	}

	withdrawal, err := model.ReviewInviteWithdrawal(id, req.Action, req.AdminRemark, c.GetInt("id"), c.GetString("username"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, withdrawal)
}
