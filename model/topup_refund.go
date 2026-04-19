package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	TopUpRefundStatusPending = "pending"
	TopUpRefundStatusSuccess = "success"
	TopUpRefundStatusFailed  = "failed"
)

type TopUpRefund struct {
	Id              int     `json:"id"`
	TopUpId         int     `json:"top_up_id" gorm:"index"`
	UserId          int     `json:"user_id" gorm:"index"`
	TradeNo         string  `json:"trade_no" gorm:"type:varchar(255);index"`
	RefundNo        string  `json:"refund_no" gorm:"type:varchar(64);uniqueIndex"`
	RefundAmount    float64 `json:"refund_amount"`
	RefundReason    string  `json:"refund_reason" gorm:"type:varchar(255)"`
	Status          string  `json:"status" gorm:"type:varchar(32);index"`
	FundChange      string  `json:"fund_change" gorm:"type:varchar(8)"`
	AlipayTradeNo   string  `json:"alipay_trade_no" gorm:"type:varchar(64);index"`
	BuyerLogonID    string  `json:"buyer_logon_id" gorm:"type:varchar(100)"`
	ResponseCode    string  `json:"response_code" gorm:"type:varchar(32)"`
	ResponseMsg     string  `json:"response_msg" gorm:"type:varchar(255)"`
	ResponseSubCode string  `json:"response_sub_code" gorm:"type:varchar(64)"`
	ResponseSubMsg  string  `json:"response_sub_msg" gorm:"type:varchar(255)"`
	RefundFee       float64 `json:"refund_fee"`
	SendBackFee     float64 `json:"send_back_fee"`
	QuotaDelta      int     `json:"quota_delta"`
	OperatorId      int     `json:"operator_id" gorm:"index"`
	CreateTime      int64   `json:"create_time" gorm:"index"`
	UpdateTime      int64   `json:"update_time"`
	CompleteTime    int64   `json:"complete_time"`
}

type TopUpRefundSummary struct {
	RequestedAmount  float64 `json:"requested_amount"`
	SuccessfulAmount float64 `json:"successful_amount"`
	PendingAmount    float64 `json:"pending_amount"`
	RefundCount      int     `json:"refund_count"`
}

type AdminTopUpItem struct {
	Id                     int     `json:"id"`
	UserId                 int     `json:"user_id"`
	Username               string  `json:"username"`
	Amount                 int64   `json:"amount"`
	Money                  float64 `json:"money"`
	TradeNo                string  `json:"trade_no"`
	PaymentMethod          string  `json:"payment_method"`
	CreateTime             int64   `json:"create_time"`
	CompleteTime           int64   `json:"complete_time"`
	Status                 string  `json:"status"`
	RefundCount            int     `json:"refund_count"`
	RequestedRefundAmount  float64 `json:"requested_refund_amount"`
	SuccessfulRefundAmount float64 `json:"successful_refund_amount"`
	PendingRefundAmount    float64 `json:"pending_refund_amount"`
	RefundableAmount       float64 `json:"refundable_amount"`
	RefundStatus           string  `json:"refund_status"`
	CanRefund              bool    `json:"can_refund"`
	CanManualRefund        bool    `json:"can_manual_refund"`
}

type TopUpRefundFinalizePayload struct {
	Code          string
	Msg           string
	SubCode       string
	SubMsg        string
	TradeNo       string
	OutTradeNo    string
	BuyerLogonID  string
	RefundFee     string
	SendBackFee   string
	FundChange    string
}

func getTopUpQuota(topUp *TopUp) int {
	if topUp == nil {
		return 0
	}
	dAmount := decimal.NewFromInt(topUp.Amount)
	dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
	return int(dAmount.Mul(dQuotaPerUnit).IntPart())
}

func parseMoneyDecimal(raw string) (decimal.Decimal, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return decimal.Zero, errors.New("金额不能为空")
	}
	d, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, errors.New("金额格式错误")
	}
	d = d.Round(2)
	if !d.IsPositive() {
		return decimal.Zero, errors.New("退款金额必须大于 0")
	}
	return d, nil
}

func decimalFromFloatMoney(value float64) decimal.Decimal {
	return decimal.NewFromFloat(value).Round(2)
}

func applySuccessfulTopUpRefundQuota(tx *gorm.DB, topUp *TopUp, refund *TopUpRefund) error {
	if tx == nil || topUp == nil || refund == nil {
		return errors.New("退款参数错误")
	}

	totalQuota := getTopUpQuota(topUp)
	totalMoney := decimalFromFloatMoney(topUp.Money)

	var successfulRefunds []*TopUpRefund
	if err := tx.Where("top_up_id = ? AND status = ? AND id <> ?", topUp.Id, TopUpRefundStatusSuccess, refund.Id).Find(&successfulRefunds).Error; err != nil {
		return err
	}

	successAmountBefore := decimal.Zero
	quotaDeductedBefore := 0
	for _, item := range successfulRefunds {
		successAmountBefore = successAmountBefore.Add(decimalFromFloatMoney(item.RefundAmount))
		quotaDeductedBefore += item.QuotaDelta
	}

	successAmountAfter := successAmountBefore.Add(decimalFromFloatMoney(refund.RefundAmount))
	targetQuotaDeducted := quotaDeductedBefore
	if !totalMoney.IsZero() && totalQuota > 0 {
		if successAmountAfter.GreaterThanOrEqual(totalMoney) {
			targetQuotaDeducted = totalQuota
		} else {
			targetQuotaDeducted = int(decimal.NewFromInt(int64(totalQuota)).
				Mul(successAmountAfter).
				Div(totalMoney).
				IntPart())
		}
	}
	quotaDelta := targetQuotaDeducted - quotaDeductedBefore
	if quotaDelta < 0 {
		quotaDelta = 0
	}

	if quotaDelta > 0 {
		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota - ?", quotaDelta)).Error; err != nil {
			return err
		}
	}

	refund.QuotaDelta = quotaDelta
	refund.CompleteTime = common.GetTimestamp()
	refund.Status = TopUpRefundStatusSuccess
	return nil
}

func generateTopUpRefundNo(topUpId int) string {
	return fmt.Sprintf("ALIRF_%d_%d", topUpId, time.Now().UnixNano())
}

func generateManualTopUpRefundNo(topUpId int) string {
	return fmt.Sprintf("MANUALRF_%d_%d", topUpId, time.Now().UnixNano())
}

func GetTopUpRefundsByTopUpId(topUpId int) ([]*TopUpRefund, error) {
	var refunds []*TopUpRefund
	err := DB.Where("top_up_id = ?", topUpId).Order("id desc").Find(&refunds).Error
	return refunds, err
}

func getTopUpRefundSummariesByTradeNos(tradeNos []string) (map[string]TopUpRefundSummary, error) {
	summaries := make(map[string]TopUpRefundSummary, len(tradeNos))
	if len(tradeNos) == 0 {
		return summaries, nil
	}

	var refunds []*TopUpRefund
	if err := DB.Where("trade_no IN ?", tradeNos).Find(&refunds).Error; err != nil {
		return nil, err
	}

	for _, refund := range refunds {
		summary := summaries[refund.TradeNo]
		summary.RefundCount++
		switch refund.Status {
		case TopUpRefundStatusSuccess:
			summary.SuccessfulAmount += refund.RefundAmount
			summary.RequestedAmount += refund.RefundAmount
		case TopUpRefundStatusPending:
			summary.PendingAmount += refund.RefundAmount
			summary.RequestedAmount += refund.RefundAmount
		}
		summaries[refund.TradeNo] = summary
	}
	return summaries, nil
}

func BuildAdminTopUpItems(topups []*TopUp) ([]*AdminTopUpItem, error) {
	items := make([]*AdminTopUpItem, 0, len(topups))
	if len(topups) == 0 {
		return items, nil
	}

	tradeNos := make([]string, 0, len(topups))
	for _, topUp := range topups {
		if topUp == nil {
			continue
		}
		tradeNos = append(tradeNos, topUp.TradeNo)
	}

	summaries, err := getTopUpRefundSummariesByTradeNos(tradeNos)
	if err != nil {
		return nil, err
	}

	for _, topUp := range topups {
		if topUp == nil {
			continue
		}
		username, _ := GetUsernameById(topUp.UserId, false)
		summary := summaries[topUp.TradeNo]
		refundableAmount := decimal.Zero
		if topUp.Status == common.TopUpStatusSuccess {
			refundableAmount = decimalFromFloatMoney(topUp.Money).
				Sub(decimalFromFloatMoney(summary.RequestedAmount)).
				Round(2)
			if refundableAmount.IsNegative() {
				refundableAmount = decimal.Zero
			}
		}

		refundStatus := "none"
		if summary.PendingAmount > 0 {
			refundStatus = "pending"
		} else if refundableAmount.IsZero() && summary.SuccessfulAmount > 0 {
			refundStatus = "full"
		} else if summary.SuccessfulAmount > 0 {
			refundStatus = "partial"
		}

		items = append(items, &AdminTopUpItem{
			Id:                     topUp.Id,
			UserId:                 topUp.UserId,
			Username:               username,
			Amount:                 topUp.Amount,
			Money:                  topUp.Money,
			TradeNo:                topUp.TradeNo,
			PaymentMethod:          topUp.PaymentMethod,
			CreateTime:             topUp.CreateTime,
			CompleteTime:           topUp.CompleteTime,
			Status:                 topUp.Status,
			RefundCount:            summary.RefundCount,
			RequestedRefundAmount:  decimalFromFloatMoney(summary.RequestedAmount).InexactFloat64(),
			SuccessfulRefundAmount: decimalFromFloatMoney(summary.SuccessfulAmount).InexactFloat64(),
			PendingRefundAmount:    decimalFromFloatMoney(summary.PendingAmount).InexactFloat64(),
			RefundableAmount:       refundableAmount.InexactFloat64(),
			RefundStatus:           refundStatus,
			CanRefund: topUp.PaymentMethod == "alipay_f2f" &&
				topUp.Status == common.TopUpStatusSuccess &&
				refundableAmount.GreaterThan(decimal.Zero),
			CanManualRefund: topUp.Status == common.TopUpStatusSuccess &&
				refundableAmount.GreaterThan(decimal.Zero),
		})
	}

	return items, nil
}

func PrepareTopUpRefund(topUpId int, refundAmount string, refundReason string, outRequestNo string, operatorId int) (*TopUp, *TopUpRefund, bool, error) {
	amountDecimal, err := parseMoneyDecimal(refundAmount)
	if err != nil {
		return nil, nil, false, err
	}

	var topUp TopUp
	var refund TopUpRefund
	isRetry := false
	now := common.GetTimestamp()

	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", topUpId).First(&topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}
		if topUp.PaymentMethod != "alipay_f2f" {
			return errors.New("仅支持支付宝当面付订单退款")
		}
		if topUp.Status != common.TopUpStatusSuccess {
			return errors.New("仅支付成功的订单支持退款")
		}

		totalMoney := decimalFromFloatMoney(topUp.Money)
		if amountDecimal.GreaterThan(totalMoney) {
			return errors.New("退款金额不能大于原支付金额")
		}

		if outRequestNo == "" {
			outRequestNo = generateTopUpRefundNo(topUp.Id)
		}

		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("refund_no = ?", outRequestNo).First(&refund).Error; err == nil {
			isRetry = true
			if refund.TopUpId != topUp.Id {
				return errors.New("退款请求号已用于其他订单")
			}
			if !decimalFromFloatMoney(refund.RefundAmount).Equal(amountDecimal) {
				return errors.New("相同退款请求号的退款金额必须保持一致")
			}
			existingReason := strings.TrimSpace(refund.RefundReason)
			incomingReason := strings.TrimSpace(refundReason)
			if existingReason != incomingReason {
				return errors.New("相同退款请求号的退款原因必须保持一致")
			}
			refund.OperatorId = operatorId
			refund.UpdateTime = now
			return tx.Save(&refund).Error
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var latest TopUpRefund
		if err := tx.Order("id desc").Where("top_up_id = ?", topUp.Id).First(&latest).Error; err == nil {
			if now-latest.CreateTime < 3 {
				return errors.New("同一笔交易退款至少间隔 3 秒")
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var refunds []*TopUpRefund
		if err := tx.Where("top_up_id = ?", topUp.Id).Find(&refunds).Error; err != nil {
			return err
		}

		reservedAmount := decimal.Zero
		for _, item := range refunds {
			if item.Status == TopUpRefundStatusFailed {
				continue
			}
			reservedAmount = reservedAmount.Add(decimalFromFloatMoney(item.RefundAmount))
		}
		if reservedAmount.Add(amountDecimal).GreaterThan(totalMoney) {
			return errors.New("累计退款金额不能大于原支付金额")
		}

		refund = TopUpRefund{
			TopUpId:      topUp.Id,
			UserId:       topUp.UserId,
			TradeNo:      topUp.TradeNo,
			RefundNo:     outRequestNo,
			RefundAmount: amountDecimal.InexactFloat64(),
			RefundReason: strings.TrimSpace(refundReason),
			Status:       TopUpRefundStatusPending,
			OperatorId:   operatorId,
			CreateTime:   now,
			UpdateTime:   now,
		}
		return tx.Create(&refund).Error
	})
	if err != nil {
		return nil, nil, false, err
	}
	return &topUp, &refund, isRetry, nil
}

func FinalizeTopUpRefund(refundId int, payload TopUpRefundFinalizePayload, callerIp string) (*TopUpRefund, error) {
	var refund TopUpRefund

	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", refundId).First(&refund).Error; err != nil {
			return errors.New("退款记录不存在")
		}

		var topUp TopUp
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", refund.TopUpId).First(&topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}

		now := common.GetTimestamp()
		refund.UpdateTime = now
		refund.ResponseCode = payload.Code
		refund.ResponseMsg = payload.Msg
		refund.ResponseSubCode = payload.SubCode
		refund.ResponseSubMsg = payload.SubMsg
		refund.FundChange = payload.FundChange
		refund.BuyerLogonID = payload.BuyerLogonID
		refund.AlipayTradeNo = payload.TradeNo

		if payload.RefundFee != "" {
			if d, err := parseMoneyDecimal(payload.RefundFee); err == nil {
				refund.RefundFee = d.InexactFloat64()
			}
		}
		if payload.SendBackFee != "" {
			if d, err := parseMoneyDecimal(payload.SendBackFee); err == nil {
				refund.SendBackFee = d.InexactFloat64()
			}
		}

		nextStatus := TopUpRefundStatusFailed
		if payload.Code == "10000" && strings.EqualFold(payload.FundChange, "Y") {
			nextStatus = TopUpRefundStatusSuccess
		} else if payload.Code == "10000" || payload.SubCode == "ACQ.SYSTEM_ERROR" {
			nextStatus = TopUpRefundStatusPending
		}

		if nextStatus == TopUpRefundStatusSuccess && refund.Status != TopUpRefundStatusSuccess {
			if err := applySuccessfulTopUpRefundQuota(tx, &topUp, &refund); err != nil {
				return err
			}
		} else {
			refund.Status = nextStatus
			if nextStatus == TopUpRefundStatusFailed {
				refund.CompleteTime = now
			}
		}

		if err := tx.Save(&refund).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if refund.Status == TopUpRefundStatusSuccess {
		content := fmt.Sprintf("支付宝充值退款成功，订单号: %s，退款金额: %.2f，扣回额度: %s", refund.TradeNo, refund.RefundAmount, logger.FormatQuota(refund.QuotaDelta))
		RecordRefundLog(refund.UserId, content, callerIp, "alipay_f2f", refund.OperatorId)
	}

	return &refund, nil
}

func MarkTopUpRefundManual(topUpId int, refundAmount string, refundReason string, operatorId int, callerIp string) (*TopUpRefund, error) {
	amountDecimal, err := parseMoneyDecimal(refundAmount)
	if err != nil {
		return nil, err
	}

	var refund TopUpRefund
	now := common.GetTimestamp()

	err = DB.Transaction(func(tx *gorm.DB) error {
		var topUp TopUp
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", topUpId).First(&topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}
		if topUp.Status != common.TopUpStatusSuccess {
			return errors.New("仅支付成功的订单可以手动标记退款")
		}

		totalMoney := decimalFromFloatMoney(topUp.Money)
		if amountDecimal.GreaterThan(totalMoney) {
			return errors.New("退款金额不能大于原支付金额")
		}

		var refunds []*TopUpRefund
		if err := tx.Where("top_up_id = ?", topUp.Id).Find(&refunds).Error; err != nil {
			return err
		}

		reservedAmount := decimal.Zero
		for _, item := range refunds {
			if item.Status == TopUpRefundStatusFailed {
				continue
			}
			reservedAmount = reservedAmount.Add(decimalFromFloatMoney(item.RefundAmount))
		}
		if reservedAmount.Add(amountDecimal).GreaterThan(totalMoney) {
			return errors.New("累计退款金额不能大于原支付金额")
		}

		refund = TopUpRefund{
			TopUpId:         topUp.Id,
			UserId:          topUp.UserId,
			TradeNo:         topUp.TradeNo,
			RefundNo:        generateManualTopUpRefundNo(topUp.Id),
			RefundAmount:    amountDecimal.InexactFloat64(),
			RefundReason:    strings.TrimSpace(refundReason),
			Status:          TopUpRefundStatusPending,
			FundChange:      "MANUAL",
			ResponseCode:    "MANUAL",
			ResponseMsg:     "管理员手动标记退款",
			ResponseSubCode: "MANUAL_REFUND",
			ResponseSubMsg:  "管理员手动标记退款",
			OperatorId:      operatorId,
			CreateTime:      now,
			UpdateTime:      now,
		}
		if err := tx.Create(&refund).Error; err != nil {
			return err
		}

		return applySuccessfulTopUpRefundQuota(tx, &topUp, &refund)
	})
	if err != nil {
		return nil, err
	}

	content := fmt.Sprintf("管理员手动标记充值退款成功，订单号: %s，退款金额: %.2f，扣回额度: %s", refund.TradeNo, refund.RefundAmount, logger.FormatQuota(refund.QuotaDelta))
	RecordRefundLog(refund.UserId, content, callerIp, "manual", refund.OperatorId)
	return &refund, nil
}
