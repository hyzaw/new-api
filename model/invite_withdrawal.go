package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	InviteWithdrawalStatusPending  = "pending"
	InviteWithdrawalStatusPaid     = "paid"
	InviteWithdrawalStatusRejected = "rejected"
)

const (
	minInviteWithdrawalAmount      = 20
	maxInviteWithdrawalReceiptSize = 512 * 1024
)

type InviteWithdrawal struct {
	Id          int     `json:"id"`
	UserId      int     `json:"user_id" gorm:"index"`
	Username    string  `json:"username" gorm:"type:varchar(64);index;default:''"`
	Amount      float64 `json:"amount"`
	Quota       int     `json:"quota"`
	ReceiptCode string  `json:"receipt_code" gorm:"type:text"`
	UserRemark  string  `json:"user_remark" gorm:"type:varchar(255);default:''"`
	AdminRemark string  `json:"admin_remark" gorm:"type:varchar(255);default:''"`
	Status      string  `json:"status" gorm:"type:varchar(32);index"`
	OperatorId  int     `json:"operator_id" gorm:"index"`
	ProcessedAt int64   `json:"processed_at" gorm:"bigint;default:0;index"`
	CreatedAt   int64   `json:"created_at" gorm:"bigint;index"`
	UpdatedAt   int64   `json:"updated_at" gorm:"bigint"`
}

func validateInviteWithdrawalReceiptCode(receiptCode string) (string, error) {
	normalized := strings.TrimSpace(receiptCode)
	if normalized == "" {
		return "", errors.New("请上传收款码")
	}
	if !strings.HasPrefix(normalized, "data:image/") || !strings.Contains(normalized, ";base64,") {
		return "", errors.New("收款码格式无效，请重新上传图片")
	}
	if len(normalized) > maxInviteWithdrawalReceiptSize {
		return "", errors.New("收款码图片过大，请压缩后重试")
	}
	return normalized, nil
}

func displayAmountDecimalToQuota(amount decimal.Decimal) (int, error) {
	if !amount.IsPositive() {
		return 0, errors.New("提现金额必须大于 0")
	}

	switch operation_setting.GetQuotaDisplayType() {
	case operation_setting.QuotaDisplayTypeTokens:
		rounded := amount.Round(0)
		if !rounded.IsPositive() {
			return 0, errors.New("提现金额无效")
		}
		return int(rounded.IntPart()), nil
	default:
		usdAmount := amount
		if operation_setting.GetQuotaDisplayType() != operation_setting.QuotaDisplayTypeUSD {
			rate := operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
			if rate <= 0 {
				rate = 1
			}
			usdAmount = amount.Div(decimal.NewFromFloat(rate))
		}
		quota := usdAmount.Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Round(0)
		if !quota.IsPositive() {
			return 0, errors.New("提现金额无效")
		}
		return int(quota.IntPart()), nil
	}
}

func CreateInviteWithdrawal(userId int, amount string, receiptCode string, userRemark string) (*InviteWithdrawal, error) {
	amountDecimal, err := parseMoneyDecimal(amount)
	if err != nil {
		return nil, err
	}
	if amountDecimal.LessThan(decimal.NewFromInt(minInviteWithdrawalAmount)) {
		return nil, fmt.Errorf("最低提现金额为 %d", minInviteWithdrawalAmount)
	}

	normalizedReceiptCode, err := validateInviteWithdrawalReceiptCode(receiptCode)
	if err != nil {
		return nil, err
	}

	quota, err := displayAmountDecimalToQuota(amountDecimal)
	if err != nil {
		return nil, err
	}

	withdrawal := &InviteWithdrawal{}
	now := common.GetTimestamp()
	userRemark = strings.TrimSpace(userRemark)

	err = DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Select("id", "username", "aff_quota").
			Where("id = ?", userId).
			First(&user).Error; err != nil {
			return err
		}
		if user.AffQuota < quota {
			return errors.New("邀请余额不足，无法申请提现")
		}
		if err := tx.Model(&User{}).
			Where("id = ?", user.Id).
			Update("aff_quota", gorm.Expr("aff_quota - ?", quota)).Error; err != nil {
			return err
		}

		withdrawal = &InviteWithdrawal{
			UserId:      user.Id,
			Username:    user.Username,
			Amount:      amountDecimal.InexactFloat64(),
			Quota:       quota,
			ReceiptCode: normalizedReceiptCode,
			UserRemark:  userRemark,
			Status:      InviteWithdrawalStatusPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		return tx.Create(withdrawal).Error
	})
	if err != nil {
		return nil, err
	}

	RecordLog(userId, LogTypeSystem, fmt.Sprintf("申请邀请提现 %.2f，预扣邀请额度 %s", withdrawal.Amount, logger.LogQuota(withdrawal.Quota)))
	return withdrawal, nil
}

func GetInviteWithdrawalsByUserId(userId int) ([]*InviteWithdrawal, error) {
	var withdrawals []*InviteWithdrawal
	err := DB.Where("user_id = ?", userId).Order("id DESC").Find(&withdrawals).Error
	return withdrawals, err
}

func GetInviteWithdrawals(pageInfo *common.PageInfo, keyword string, status string) ([]*InviteWithdrawal, int64, error) {
	var (
		withdrawals []*InviteWithdrawal
		total       int64
	)

	tx := DB.Model(&InviteWithdrawal{})
	keyword = strings.TrimSpace(keyword)
	status = strings.TrimSpace(status)

	if keyword != "" {
		likeKeyword := "%" + keyword + "%"
		if userId, err := strconv.Atoi(keyword); err == nil {
			tx = tx.Where("user_id = ? OR username LIKE ?", userId, likeKeyword)
		} else {
			tx = tx.Where("username LIKE ?", likeKeyword)
		}
	}
	if status != "" {
		tx = tx.Where("status = ?", status)
	}

	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Order("id DESC").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&withdrawals).Error; err != nil {
		return nil, 0, err
	}
	return withdrawals, total, nil
}

func ReviewInviteWithdrawal(id int, action string, adminRemark string, operatorId int, operatorName string) (*InviteWithdrawal, error) {
	adminRemark = strings.TrimSpace(adminRemark)
	action = strings.TrimSpace(action)

	if action != InviteWithdrawalStatusPaid && action != InviteWithdrawalStatusRejected {
		return nil, errors.New("无效的提现处理操作")
	}

	var withdrawal InviteWithdrawal
	now := common.GetTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", id).First(&withdrawal).Error; err != nil {
			return errors.New("提现申请不存在")
		}
		if withdrawal.Status != InviteWithdrawalStatusPending {
			return errors.New("该提现申请已处理，请勿重复操作")
		}

		withdrawal.Status = action
		withdrawal.AdminRemark = adminRemark
		withdrawal.OperatorId = operatorId
		withdrawal.ProcessedAt = now
		withdrawal.UpdatedAt = now

		if action == InviteWithdrawalStatusRejected {
			if err := tx.Model(&User{}).
				Where("id = ?", withdrawal.UserId).
				Update("aff_quota", gorm.Expr("aff_quota + ?", withdrawal.Quota)).Error; err != nil {
				return err
			}
		}

		return tx.Save(&withdrawal).Error
	})
	if err != nil {
		return nil, err
	}

	adminInfo := map[string]interface{}{
		"admin_id":       operatorId,
		"admin_username": operatorName,
	}
	switch action {
	case InviteWithdrawalStatusPaid:
		RecordLogWithAdminInfo(withdrawal.UserId, LogTypeManage, fmt.Sprintf("管理员已处理邀请提现申请 %.2f", withdrawal.Amount), adminInfo)
	case InviteWithdrawalStatusRejected:
		RecordLogWithAdminInfo(withdrawal.UserId, LogTypeManage, fmt.Sprintf("管理员已驳回邀请提现申请 %.2f，并退回邀请额度 %s", withdrawal.Amount, logger.LogQuota(withdrawal.Quota)), adminInfo)
	}

	return &withdrawal, nil
}
