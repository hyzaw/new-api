package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type AdminRefundTopUpRequest struct {
	TopUpId       int    `json:"top_up_id"`
	RefundAmount  string `json:"refund_amount"`
	RefundReason  string `json:"refund_reason"`
	OutRequestNo  string `json:"out_request_no"`
}

func GetTopUpDashboardStats(c *gin.Context) {
	days := common.String2Int(c.DefaultQuery("days", "30"))
	stats, err := model.GetAdminTopUpDashboardStats(days)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, stats)
}

func GetTopUpRefundsByAdmin(c *gin.Context) {
	id := common.String2Int(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}

	refunds, err := model.GetTopUpRefundsByTopUpId(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": refunds})
}

func AdminRefundTopUp(c *gin.Context) {
	var req AdminRefundTopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.TopUpId <= 0 {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	if strings.TrimSpace(req.RefundAmount) == "" {
		common.ApiErrorMsg(c, "退款金额不能为空")
		return
	}

	topUp, refund, _, err := model.PrepareTopUpRefund(req.TopUpId, req.RefundAmount, req.RefundReason, req.OutRequestNo, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if refund.Status == model.TopUpRefundStatusSuccess {
		common.ApiSuccess(c, gin.H{
			"message": "退款成功",
			"refund":  refund,
		})
		return
	}

	LockOrder(topUp.TradeNo)
	defer UnlockOrder(topUp.TradeNo)

	resp, err := refundAlipayTrade(refund)
	if err != nil {
		_, finalizeErr := model.FinalizeTopUpRefund(refund.Id, model.TopUpRefundFinalizePayload{
			Msg: err.Error(),
		}, c.ClientIP())
		if finalizeErr != nil {
			common.ApiError(c, finalizeErr)
			return
		}
		common.ApiErrorMsg(c, err.Error())
		return
	}

	finalized, err := model.FinalizeTopUpRefund(refund.Id, model.TopUpRefundFinalizePayload{
		Code:         resp.Code,
		Msg:          resp.Msg,
		SubCode:      resp.SubCode,
		SubMsg:       resp.SubMsg,
		TradeNo:      resp.TradeNo,
		OutTradeNo:   resp.OutTradeNo,
		BuyerLogonID: resp.BuyerLogonID,
		RefundFee:    resp.RefundFee,
		SendBackFee:  resp.SendBackFee,
		FundChange:   resp.FundChange,
	}, c.ClientIP())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if resp.Code != "10000" {
		message := resp.SubMsg
		if message == "" {
			message = resp.Msg
		}
		if message == "" {
			message = "退款失败"
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": message,
			"data":    finalized,
		})
		return
	}

	message := "退款请求已提交"
	if strings.EqualFold(resp.FundChange, "Y") {
		message = "退款成功"
	}
	common.ApiSuccess(c, gin.H{
		"message": message,
		"refund":  finalized,
	})
}
