package model

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const defaultPendingTopUpExpireAfterSeconds int64 = 2 * 60 * 60

type TopUp struct {
	Id                        int     `json:"id"`
	UserId                    int     `json:"user_id" gorm:"index"`
	Amount                    int64   `json:"amount"`
	Money                     float64 `json:"money"`
	TradeNo                   string  `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod             string  `json:"payment_method" gorm:"type:varchar(50)"`
	CreateTime                int64   `json:"create_time"`
	CompleteTime              int64   `json:"complete_time"`
	Status                    string  `json:"status"`
	InviteRebateUserId        int     `json:"invite_rebate_user_id" gorm:"index"`
	InviteRebateRatio         float64 `json:"invite_rebate_ratio"`
	InviteRebateQuota         int     `json:"invite_rebate_quota"`
	InviteRebateRefundedQuota int     `json:"invite_rebate_refunded_quota"`
	InviteRebateTime          int64   `json:"invite_rebate_time"`
}

var ErrPaymentMethodMismatch = errors.New("payment method mismatch")

func (topUp *TopUp) Insert() error {
	var err error
	err = DB.Create(topUp).Error
	return err
}

func (topUp *TopUp) Update() error {
	var err error
	err = DB.Save(topUp).Error
	return err
}

func GetTopUpById(id int) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("id = ?", id).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func GetTopUpByTradeNo(tradeNo string) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("trade_no = ?", tradeNo).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func getTopUpGrantedQuota(topUp *TopUp) int {
	if topUp == nil {
		return 0
	}

	switch topUp.PaymentMethod {
	case "stripe":
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		return int(decimal.NewFromFloat(topUp.Money).Mul(dQuotaPerUnit).IntPart())
	case "creem":
		if topUp.Amount <= 0 {
			return 0
		}
		return int(topUp.Amount)
	default:
		if topUp.Amount <= 0 {
			return 0
		}
		dAmount := decimal.NewFromInt(topUp.Amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		return int(dAmount.Mul(dQuotaPerUnit).IntPart())
	}
}

func calculateInviteTopUpRebateQuota(totalQuota int) int {
	if totalQuota <= 0 || common.TopUpAffRatio <= 0 {
		return 0
	}
	return int(decimal.NewFromInt(int64(totalQuota)).
		Mul(decimal.NewFromFloat(common.TopUpAffRatio)).
		Div(decimal.NewFromInt(100)).
		IntPart())
}

func calculateTopUpInviteRebateQuota(topUp *TopUp) int {
	return calculateInviteTopUpRebateQuota(getTopUpGrantedQuota(topUp))
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func applyTopUpInviteRebateTx(tx *gorm.DB, topUp *TopUp) error {
	if tx == nil || topUp == nil {
		return errors.New("邀请返利参数错误")
	}
	if topUp.InviteRebateTime > 0 || topUp.InviteRebateQuota > 0 || topUp.InviteRebateUserId > 0 {
		return nil
	}
	if common.TopUpAffRatio <= 0 {
		return nil
	}

	var user User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Select("id", "inviter_id", "username", "display_name").Where("id = ?", topUp.UserId).First(&user).Error; err != nil {
		return err
	}
	if user.InviterId == 0 {
		return nil
	}

	rebateQuota := calculateTopUpInviteRebateQuota(topUp)
	if rebateQuota <= 0 {
		return nil
	}

	var inviter User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").
		Select("id", "username", "quota", "aff_quota", "aff_history").
		Where("id = ?", user.InviterId).
		First(&inviter).Error; err != nil {
		return err
	}
	inviter.AffQuota += rebateQuota
	inviter.AffHistoryQuota += rebateQuota
	if err := tx.Model(&User{}).Where("id = ?", inviter.Id).Updates(map[string]any{
		"aff_quota":   inviter.AffQuota,
		"aff_history": inviter.AffHistoryQuota,
	}).Error; err != nil {
		return err
	}

	topUp.InviteRebateUserId = user.InviterId
	topUp.InviteRebateRatio = common.TopUpAffRatio
	topUp.InviteRebateQuota = rebateQuota
	topUp.InviteRebateRefundedQuota = 0
	topUp.InviteRebateTime = common.GetTimestamp()

	if err := tx.Model(topUp).Select(
		"invite_rebate_user_id",
		"invite_rebate_ratio",
		"invite_rebate_quota",
		"invite_rebate_refunded_quota",
		"invite_rebate_time",
	).Updates(topUp).Error; err != nil {
		return err
	}
	if err := syncInviteRebateDetailTx(tx, topUp, &user); err != nil {
		return err
	}
	return syncTopUpInviteRebateWalletRecordTx(tx, topUp, &user, &inviter)
}

func applyTopUpInviteRebateRefundTx(tx *gorm.DB, topUp *TopUp, successfulRefundAmount decimal.Decimal, refund *TopUpRefund) error {
	if tx == nil || topUp == nil {
		return errors.New("邀请返利回退参数错误")
	}
	if refund != nil {
		refund.InviteRebateDelta = 0
	}
	if topUp.InviteRebateQuota <= 0 || topUp.InviteRebateUserId <= 0 {
		return nil
	}

	totalMoney := decimal.NewFromFloat(topUp.Money).Round(2)
	if !successfulRefundAmount.IsPositive() || !totalMoney.IsPositive() {
		return nil
	}

	targetRefunded := topUp.InviteRebateRefundedQuota
	if successfulRefundAmount.GreaterThanOrEqual(totalMoney) {
		targetRefunded = topUp.InviteRebateQuota
	} else {
		targetRefunded = int(decimal.NewFromInt(int64(topUp.InviteRebateQuota)).
			Mul(successfulRefundAmount).
			Div(totalMoney).
			IntPart())
	}
	if targetRefunded < topUp.InviteRebateRefundedQuota {
		targetRefunded = topUp.InviteRebateRefundedQuota
	}

	rebateDelta := targetRefunded - topUp.InviteRebateRefundedQuota
	if refund != nil {
		refund.InviteRebateDelta = rebateDelta
	}
	if rebateDelta <= 0 {
		return nil
	}

	var inviter User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", topUp.InviteRebateUserId).First(&inviter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			topUp.InviteRebateRefundedQuota = targetRefunded
			return tx.Model(topUp).Update("invite_rebate_refunded_quota", targetRefunded).Error
		}
		return err
	}

	deductFromAff := minInt(inviter.AffQuota, rebateDelta)
	deductFromQuota := rebateDelta - deductFromAff
	inviter.AffQuota -= deductFromAff
	if inviter.AffHistoryQuota >= rebateDelta {
		inviter.AffHistoryQuota -= rebateDelta
	} else {
		inviter.AffHistoryQuota = 0
	}
	inviter.Quota -= deductFromQuota
	if err := tx.Model(&User{}).Where("id = ?", inviter.Id).Updates(map[string]any{
		"aff_quota":   inviter.AffQuota,
		"aff_history": inviter.AffHistoryQuota,
		"quota":       inviter.Quota,
	}).Error; err != nil {
		return err
	}

	topUp.InviteRebateRefundedQuota = targetRefunded
	if err := tx.Model(topUp).Update("invite_rebate_refunded_quota", targetRefunded).Error; err != nil {
		return err
	}
	if err := syncInviteRebateDetailTx(tx, topUp, nil); err != nil {
		return err
	}
	return syncTopUpInviteRebateRefundWalletRecordTx(tx, topUp, nil, refund, &inviter, deductFromAff, deductFromQuota)
}

func finalizeSuccessfulTopUpTx(tx *gorm.DB, topUp *TopUp, extraUserUpdates map[string]any) (int, error) {
	if tx == nil || topUp == nil {
		return 0, errors.New("充值参数错误")
	}
	if topUp.Status == common.TopUpStatusSuccess {
		return 0, nil
	}
	if topUp.Status != common.TopUpStatusPending {
		return 0, errors.New("充值订单状态错误")
	}

	quotaToAdd := getTopUpGrantedQuota(topUp)
	if quotaToAdd <= 0 {
		return 0, errors.New("无效的充值额度")
	}

	topUp.CompleteTime = common.GetTimestamp()
	topUp.Status = common.TopUpStatusSuccess
	if err := tx.Save(topUp).Error; err != nil {
		return 0, err
	}

	updates := map[string]any{
		"quota": gorm.Expr("quota + ?", quotaToAdd),
	}
	for key, value := range extraUserUpdates {
		updates[key] = value
	}
	if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Updates(updates).Error; err != nil {
		return 0, err
	}

	if err := applyTopUpInviteRebateTx(tx, topUp); err != nil {
		return 0, err
	}

	return quotaToAdd, nil
}

func ExpireTopUpOrder(tradeNo string) error {
	if tradeNo == "" {
		return errors.New("tradeNo is empty")
	}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}
		if topUp.Status != common.TopUpStatusPending {
			return nil
		}
		topUp.Status = common.TopUpStatusExpired
		topUp.CompleteTime = common.GetTimestamp()
		return tx.Save(topUp).Error
	})
}

func ExpirePendingTopUps(batchSize int, maxPendingSeconds int64) (int, error) {
	if batchSize <= 0 {
		batchSize = 200
	}
	if maxPendingSeconds <= 0 {
		maxPendingSeconds = defaultPendingTopUpExpireAfterSeconds
	}

	cutoff := common.GetTimestamp() - maxPendingSeconds
	var ids []int
	if err := DB.Model(&TopUp{}).
		Where("status = ? AND create_time <= ?", common.TopUpStatusPending, cutoff).
		Order("id asc").
		Limit(batchSize).
		Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}

	now := common.GetTimestamp()
	res := DB.Model(&TopUp{}).
		Where("id IN ? AND status = ?", ids, common.TopUpStatusPending).
		Updates(map[string]any{
			"status":        common.TopUpStatusExpired,
			"complete_time": now,
		})
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

func Recharge(referenceId string, customerId string, callerIp string) (completed bool, err error) {
	if referenceId == "" {
		return false, errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}
	completed = false

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentMethod != "stripe" {
			return ErrPaymentMethodMismatch
		}

		quotaToAdd, err = finalizeSuccessfulTopUpTx(tx, topUp, map[string]any{
			"stripe_customer": customerId,
		})
		if err != nil {
			return err
		}
		completed = quotaToAdd > 0

		return nil
	})

	if err != nil {
		common.SysError("topup failed: " + err.Error())
		return false, errors.New("充值失败，请稍后重试")
	}

	if completed {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%d", logger.FormatQuota(quotaToAdd), topUp.Amount), callerIp, topUp.PaymentMethod, "stripe")
		if topUp.InviteRebateUserId > 0 && topUp.InviteRebateQuota > 0 {
			RecordLog(topUp.InviteRebateUserId, LogTypeSystem, fmt.Sprintf("邀请用户充值返利 %s，订单号: %s", logger.LogQuota(topUp.InviteRebateQuota), topUp.TradeNo))
		}
	}

	return completed, nil
}

// topUpQueryWindowSeconds 限制充值记录查询的时间窗口（秒）。
const topUpQueryWindowSeconds int64 = 30 * 24 * 60 * 60

// topUpQueryCutoff 返回允许查询的最早 create_time（秒级 Unix 时间戳）。
func topUpQueryCutoff() int64 {
	return common.GetTimestamp() - topUpQueryWindowSeconds
}

func GetUserTopUps(userId int, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	// Start transaction
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	cutoff := topUpQueryCutoff()

	// Get total count within transaction
	err = tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, cutoff).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated topups within same transaction
	err = tx.Where("user_id = ? AND create_time >= ?", userId, cutoff).Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// GetAllTopUps 获取全平台的充值记录（管理员使用，不限制时间窗口）
func GetAllTopUps(pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err = tx.Model(&TopUp{}).Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// searchTopUpCountHardLimit 搜索充值记录时 COUNT 的安全上限，
// 防止对超大表执行无界 COUNT 触发 DoS。
const searchTopUpCountHardLimit = 10000

// SearchUserTopUps 按订单号搜索某用户的充值记录
func SearchUserTopUps(userId int, keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, topUpQueryCutoff())
	if keyword != "" {
		pattern, perr := sanitizeLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// SearchAllTopUps 按订单号、用户名、显示名、邮箱或用户 ID 搜索全平台充值记录（管理员使用，不限制时间窗口）
func SearchAllTopUps(keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{})
	if keyword != "" {
		pattern, perr := sanitizeLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		query = query.Joins("LEFT JOIN users ON users.id = top_ups.user_id")

		searchExpr := "(top_ups.trade_no LIKE ? ESCAPE '!' OR users.username LIKE ? ESCAPE '!' OR users.display_name LIKE ? ESCAPE '!' OR users.email LIKE ? ESCAPE '!')"
		searchArgs := []interface{}{pattern, pattern, pattern, pattern}

		if keywordInt, convErr := strconv.Atoi(keyword); convErr == nil {
			searchExpr = "(top_ups.user_id = ? OR users.id = ? OR " + searchExpr[1:]
			searchArgs = append([]interface{}{keywordInt, keywordInt}, searchArgs...)
		}

		query = query.Where(searchExpr, searchArgs...)
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// ManualCompleteTopUp 管理员手动完成订单并给用户充值
func ManualCompleteTopUp(tradeNo string, callerIp string) (bool, error) {
	if tradeNo == "" {
		return false, errors.New("未提供订单号")
	}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	var userId int
	var quotaToAdd int
	var payMoney float64
	var paymentMethod string
	var inviterRebateUserId int
	var inviterRebateQuota int
	completed := false

	err := DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		// 行级锁，避免并发补单
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}

		// 幂等处理：已成功直接返回
		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("订单状态不是待支付，无法补单")
		}

		quotaToAdd, err := finalizeSuccessfulTopUpTx(tx, topUp, nil)
		if err != nil {
			return err
		}
		completed = quotaToAdd > 0

		userId = topUp.UserId
		payMoney = topUp.Money
		paymentMethod = topUp.PaymentMethod
		inviterRebateUserId = topUp.InviteRebateUserId
		inviterRebateQuota = topUp.InviteRebateQuota
		return nil
	})

	if err != nil {
		return false, err
	}

	// 事务外记录日志，避免阻塞
	if completed {
		RecordTopupLog(userId, fmt.Sprintf("管理员补单成功，充值金额: %v，支付金额：%f", logger.FormatQuota(quotaToAdd), payMoney), callerIp, paymentMethod, "admin")
		if inviterRebateUserId > 0 && inviterRebateQuota > 0 {
			RecordLog(inviterRebateUserId, LogTypeSystem, fmt.Sprintf("邀请用户充值返利 %s，订单号: %s", logger.LogQuota(inviterRebateQuota), tradeNo))
		}
	}
	return completed, nil
}
func RechargeEpay(referenceId string, callerIp string) (completed bool, err error) {
	if referenceId == "" {
		return false, errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}
	completed = false

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentMethod == "stripe" || topUp.PaymentMethod == "creem" || topUp.PaymentMethod == "waffo" || topUp.PaymentMethod == "alipay_f2f" {
			return ErrPaymentMethodMismatch
		}

		quotaToAdd, err = finalizeSuccessfulTopUpTx(tx, topUp, nil)
		if err != nil {
			return err
		}
		completed = quotaToAdd > 0
		return nil
	})

	if err != nil {
		common.SysError("epay topup failed: " + err.Error())
		return false, errors.New("充值失败，请稍后重试")
	}

	if completed {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%f", logger.LogQuota(quotaToAdd), topUp.Money), callerIp, topUp.PaymentMethod, "epay")
		if topUp.InviteRebateUserId > 0 && topUp.InviteRebateQuota > 0 {
			RecordLog(topUp.InviteRebateUserId, LogTypeSystem, fmt.Sprintf("邀请用户充值返利 %s，订单号: %s", logger.LogQuota(topUp.InviteRebateQuota), topUp.TradeNo))
		}
	}

	return completed, nil
}

func RechargeCreem(referenceId string, customerEmail string, customerName string, callerIp string) (completed bool, err error) {
	if referenceId == "" {
		return false, errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}
	completed = false

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentMethod != "creem" {
			return ErrPaymentMethodMismatch
		}

		// 构建更新字段，优先使用邮箱，如果邮箱为空则使用用户名
		updateFields := map[string]any{}

		// 如果有客户邮箱，尝试更新用户邮箱（仅当用户邮箱为空时）
		if customerEmail != "" {
			// 先检查用户当前邮箱是否为空
			var user User
			err = tx.Where("id = ?", topUp.UserId).First(&user).Error
			if err != nil {
				return err
			}

			// 如果用户邮箱为空，则更新为支付时使用的邮箱
			if user.Email == "" {
				updateFields["email"] = customerEmail
			}
		}

		quotaToAdd, err = finalizeSuccessfulTopUpTx(tx, topUp, updateFields)
		if err != nil {
			return err
		}
		completed = quotaToAdd > 0

		return nil
	})

	if err != nil {
		common.SysError("creem topup failed: " + err.Error())
		return false, errors.New("充值失败，请稍后重试")
	}

	if completed {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("使用Creem充值成功，充值额度: %v，支付金额：%.2f", quotaToAdd, topUp.Money), callerIp, topUp.PaymentMethod, "creem")
		if topUp.InviteRebateUserId > 0 && topUp.InviteRebateQuota > 0 {
			RecordLog(topUp.InviteRebateUserId, LogTypeSystem, fmt.Sprintf("邀请用户充值返利 %s，订单号: %s", logger.LogQuota(topUp.InviteRebateQuota), topUp.TradeNo))
		}
	}

	return completed, nil
}

func RechargeWaffo(tradeNo string, callerIp string) (completed bool, err error) {
	if tradeNo == "" {
		return false, errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}
	completed = false

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentMethod != "waffo" {
			return ErrPaymentMethodMismatch
		}

		quotaToAdd, err = finalizeSuccessfulTopUpTx(tx, topUp, nil)
		if err != nil {
			return err
		}
		completed = quotaToAdd > 0

		return nil
	})

	if err != nil {
		common.SysError("waffo topup failed: " + err.Error())
		return false, errors.New("充值失败，请稍后重试")
	}

	if completed && quotaToAdd > 0 {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("Waffo充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(quotaToAdd), topUp.Money), callerIp, topUp.PaymentMethod, "waffo")
		if topUp.InviteRebateUserId > 0 && topUp.InviteRebateQuota > 0 {
			RecordLog(topUp.InviteRebateUserId, LogTypeSystem, fmt.Sprintf("邀请用户充值返利 %s，订单号: %s", logger.LogQuota(topUp.InviteRebateQuota), topUp.TradeNo))
		}
	}

	return completed, nil
}

func RechargeAlipayF2F(tradeNo string, callerIp string) (completed bool, err error) {
	if tradeNo == "" {
		return false, errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}
	completed = false

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentMethod != "alipay_f2f" {
			return ErrPaymentMethodMismatch
		}

		quotaToAdd, err = finalizeSuccessfulTopUpTx(tx, topUp, nil)
		if err != nil {
			return err
		}
		completed = quotaToAdd > 0

		return nil
	})

	if err != nil {
		common.SysError("alipay f2f topup failed: " + err.Error())
		return false, errors.New("充值失败，请稍后重试")
	}

	if completed && quotaToAdd > 0 {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("支付宝当面付充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(quotaToAdd), topUp.Money), callerIp, topUp.PaymentMethod, "alipay_f2f")
		if topUp.InviteRebateUserId > 0 && topUp.InviteRebateQuota > 0 {
			RecordLog(topUp.InviteRebateUserId, LogTypeSystem, fmt.Sprintf("邀请用户充值返利 %s，订单号: %s", logger.LogQuota(topUp.InviteRebateQuota), topUp.TradeNo))
		}
	}

	return completed, nil
}
