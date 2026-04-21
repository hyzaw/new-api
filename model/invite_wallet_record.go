package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	InviteWalletChangeTypeInviteReward           = "invite_reward"
	InviteWalletChangeTypeTopUpRebate            = "topup_rebate"
	InviteWalletChangeTypeTopUpRebateRefund      = "topup_rebate_refund"
	InviteWalletChangeTypeTransferOut            = "transfer_out"
	InviteWalletChangeTypeWithdrawalApply        = "withdrawal_apply"
	InviteWalletChangeTypeWithdrawalRejectReturn = "withdrawal_reject_return"
	InviteWalletChangeTypeAdminAdd               = "admin_add"
	InviteWalletChangeTypeAdminSubtract          = "admin_subtract"
	InviteWalletChangeTypeAdminOverride          = "admin_override"
)

type InviteWalletRecord struct {
	Id                 int     `json:"id"`
	RecordKey          string  `json:"record_key" gorm:"type:varchar(128);uniqueIndex"`
	UserId             int     `json:"user_id" gorm:"index"`
	Username           string  `json:"username" gorm:"type:varchar(64);default:'';index"`
	ChangeType         string  `json:"change_type" gorm:"type:varchar(32);index"`
	AffQuotaDelta      int     `json:"aff_quota_delta"`
	QuotaDelta         int     `json:"quota_delta"`
	AffBalanceAfter    int     `json:"aff_balance_after"`
	QuotaBalanceAfter  int     `json:"quota_balance_after"`
	InviteeId          int     `json:"invitee_id" gorm:"index"`
	InviteeUsername    string  `json:"invitee_username" gorm:"type:varchar(64);default:''"`
	InviteeDisplayName string  `json:"invitee_display_name" gorm:"type:varchar(64);default:''"`
	TopUpId            int     `json:"top_up_id" gorm:"index"`
	TopUpTradeNo       string  `json:"top_up_trade_no" gorm:"type:varchar(255);default:'';index"`
	TopUpAmount        int64   `json:"top_up_amount"`
	TopUpMoney         float64 `json:"top_up_money"`
	GrantedQuota       int     `json:"granted_quota"`
	RebateQuota        int     `json:"rebate_quota"`
	PaymentMethod      string  `json:"payment_method" gorm:"type:varchar(50);default:''"`
	TopUpRefundId      int     `json:"top_up_refund_id" gorm:"index"`
	RefundAmount       float64 `json:"refund_amount"`
	WithdrawalId       int     `json:"withdrawal_id" gorm:"index"`
	WithdrawalAmount   float64 `json:"withdrawal_amount"`
	WithdrawalStatus   string  `json:"withdrawal_status" gorm:"type:varchar(32);default:''"`
	OperatorId         int     `json:"operator_id" gorm:"index"`
	OperatorName       string  `json:"operator_name" gorm:"type:varchar(64);default:''"`
	Remark             string  `json:"remark" gorm:"type:text"`
	CreatedAt          int64   `json:"created_at" gorm:"bigint;index"`
	UpdatedAt          int64   `json:"updated_at" gorm:"bigint"`
}

type InviteWalletOverviewSummary struct {
	Id              int    `json:"id"`
	Username        string `json:"username"`
	DisplayName     string `json:"display_name"`
	AffCount        int    `json:"aff_count"`
	AffQuota        int    `json:"aff_quota"`
	AffHistoryQuota int    `json:"aff_history_quota"`
}

type InviteWalletOverview struct {
	Summary       *InviteWalletOverviewSummary `json:"summary"`
	InviteRecords []*InviteDetail              `json:"invite_records"`
	RebateRecords []*InviteDetail              `json:"rebate_records"`
	WalletRecords []*InviteWalletRecord        `json:"wallet_records"`
	Withdrawals   []*InviteWithdrawal          `json:"withdrawals"`
	TopUpOrders   []*AdminTopUpItem            `json:"topup_orders"`
}

type inviteWalletBackfillLog struct {
	Id        int
	UserId    int
	Content   string
	Other     string
	CreatedAt int64
}

var inviteWalletQuotaRegexp = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)

func createInviteWalletRecordTx(tx *gorm.DB, record *InviteWalletRecord) error {
	if tx == nil || record == nil {
		return errors.New("invite wallet record is nil")
	}
	record.RecordKey = strings.TrimSpace(record.RecordKey)
	if record.RecordKey == "" {
		record.RecordKey = "invite_wallet:" + common.GetUUID()
	}
	now := common.GetTimestamp()
	if record.CreatedAt == 0 {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	return tx.Create(record).Error
}

func inviteWalletRecordExistsTx(tx *gorm.DB, recordKey string) (bool, error) {
	if tx == nil || strings.TrimSpace(recordKey) == "" {
		return false, errors.New("invite wallet record key is empty")
	}
	var count int64
	err := tx.Model(&InviteWalletRecord{}).Where("record_key = ?", recordKey).Count(&count).Error
	return count > 0, err
}

func createInviteWalletRecordIfAbsentTx(tx *gorm.DB, record *InviteWalletRecord) error {
	exists, err := inviteWalletRecordExistsTx(tx, record.RecordKey)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return createInviteWalletRecordTx(tx, record)
}

func parseInviteWalletLoggedQuotas(text string) []int {
	matches := inviteWalletQuotaRegexp.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}
	quotas := make([]int, 0, len(matches))
	for _, match := range matches {
		value, err := decimal.NewFromString(match)
		if err != nil {
			continue
		}
		switch operation_setting.GetQuotaDisplayType() {
		case operation_setting.QuotaDisplayTypeTokens:
			quotas = append(quotas, int(value.Round(0).IntPart()))
		case operation_setting.QuotaDisplayTypeCNY:
			rate := operation_setting.USDExchangeRate
			if rate <= 0 {
				rate = 1
			}
			quotas = append(quotas, int(value.Div(decimal.NewFromFloat(rate)).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Round(0).IntPart()))
		case operation_setting.QuotaDisplayTypeCustom:
			rate := operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate
			if rate <= 0 {
				rate = 1
			}
			quotas = append(quotas, int(value.Div(decimal.NewFromFloat(rate)).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Round(0).IntPart()))
		default:
			quotas = append(quotas, int(value.Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Round(0).IntPart()))
		}
	}
	return quotas
}

func syncInviteRewardWalletRecordTx(tx *gorm.DB, inviterId int, invitee *User, rewardQuota int, inviteTime int64) error {
	if tx == nil || inviterId == 0 || invitee == nil || invitee.Id == 0 || rewardQuota <= 0 {
		return nil
	}
	if inviteTime == 0 {
		inviteTime = common.GetTimestamp()
	}
	var inviter User
	if err := tx.Select("id", "username", "aff_quota", "quota").Where("id = ?", inviterId).First(&inviter).Error; err != nil {
		return err
	}
	return createInviteWalletRecordIfAbsentTx(tx, &InviteWalletRecord{
		RecordKey:          fmt.Sprintf("invite_reward:%d", invitee.Id),
		UserId:             inviterId,
		Username:           inviter.Username,
		ChangeType:         InviteWalletChangeTypeInviteReward,
		AffQuotaDelta:      rewardQuota,
		AffBalanceAfter:    inviter.AffQuota,
		QuotaBalanceAfter:  inviter.Quota,
		InviteeId:          invitee.Id,
		InviteeUsername:    invitee.Username,
		InviteeDisplayName: invitee.DisplayName,
		CreatedAt:          inviteTime,
	})
}

func syncTopUpInviteRebateWalletRecordTx(tx *gorm.DB, topUp *TopUp, invitee *User, inviter *User) error {
	if tx == nil || topUp == nil || topUp.InviteRebateUserId == 0 || topUp.InviteRebateQuota <= 0 {
		return nil
	}
	if invitee == nil && topUp.UserId != 0 {
		invitee = &User{}
		if err := tx.Select("id", "username", "display_name").Where("id = ?", topUp.UserId).First(invitee).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			invitee = nil
		}
	}
	if inviter == nil {
		inviter = &User{}
		if err := tx.Select("id", "username", "aff_quota", "quota").Where("id = ?", topUp.InviteRebateUserId).First(inviter).Error; err != nil {
			return err
		}
	}
	record := &InviteWalletRecord{
		RecordKey:         fmt.Sprintf("topup_rebate:%d", topUp.Id),
		UserId:            topUp.InviteRebateUserId,
		Username:          inviter.Username,
		ChangeType:        InviteWalletChangeTypeTopUpRebate,
		AffQuotaDelta:     topUp.InviteRebateQuota,
		AffBalanceAfter:   inviter.AffQuota,
		QuotaBalanceAfter: inviter.Quota,
		TopUpId:           topUp.Id,
		TopUpTradeNo:      topUp.TradeNo,
		TopUpAmount:       topUp.Amount,
		TopUpMoney:        topUp.Money,
		GrantedQuota:      getTopUpGrantedQuota(topUp),
		RebateQuota:       topUp.InviteRebateQuota,
		PaymentMethod:     topUp.PaymentMethod,
		CreatedAt:         topUp.InviteRebateTime,
	}
	if invitee != nil {
		record.InviteeId = invitee.Id
		record.InviteeUsername = invitee.Username
		record.InviteeDisplayName = invitee.DisplayName
	}
	return createInviteWalletRecordIfAbsentTx(tx, record)
}

func syncTopUpInviteRebateRefundWalletRecordTx(tx *gorm.DB, topUp *TopUp, invitee *User, refund *TopUpRefund, inviter *User, deductFromAff int, deductFromQuota int) error {
	if tx == nil || topUp == nil || refund == nil || refund.InviteRebateDelta <= 0 || topUp.InviteRebateUserId == 0 {
		return nil
	}
	if invitee == nil && topUp.UserId != 0 {
		invitee = &User{}
		if err := tx.Select("id", "username", "display_name").Where("id = ?", topUp.UserId).First(invitee).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			invitee = nil
		}
	}
	record := &InviteWalletRecord{
		RecordKey:         fmt.Sprintf("topup_rebate_refund:%d", refund.Id),
		UserId:            topUp.InviteRebateUserId,
		ChangeType:        InviteWalletChangeTypeTopUpRebateRefund,
		AffQuotaDelta:     -deductFromAff,
		QuotaDelta:        -deductFromQuota,
		TopUpId:           topUp.Id,
		TopUpTradeNo:      topUp.TradeNo,
		TopUpAmount:       topUp.Amount,
		TopUpMoney:        topUp.Money,
		GrantedQuota:      getTopUpGrantedQuota(topUp),
		RebateQuota:       refund.InviteRebateDelta,
		PaymentMethod:     topUp.PaymentMethod,
		TopUpRefundId:     refund.Id,
		RefundAmount:      refund.RefundAmount,
		AffBalanceAfter:   inviter.AffQuota,
		QuotaBalanceAfter: inviter.Quota,
		CreatedAt:         refund.CompleteTime,
	}
	if record.CreatedAt == 0 {
		record.CreatedAt = refund.UpdateTime
	}
	if record.CreatedAt == 0 {
		record.CreatedAt = refund.CreateTime
	}
	if inviter != nil {
		record.Username = inviter.Username
	}
	if invitee != nil {
		record.InviteeId = invitee.Id
		record.InviteeUsername = invitee.Username
		record.InviteeDisplayName = invitee.DisplayName
	}
	if deductFromQuota > 0 {
		record.Remark = "邀请余额不足，主余额同步扣减"
	}
	return createInviteWalletRecordIfAbsentTx(tx, record)
}

func createTransferInviteWalletRecordTx(tx *gorm.DB, user *User, transferQuota int, createdAt int64) error {
	if tx == nil || user == nil || user.Id == 0 || transferQuota <= 0 {
		return nil
	}
	if createdAt == 0 {
		createdAt = common.GetTimestamp()
	}
	return createInviteWalletRecordTx(tx, &InviteWalletRecord{
		RecordKey:         "transfer_out:" + common.GetUUID(),
		UserId:            user.Id,
		Username:          user.Username,
		ChangeType:        InviteWalletChangeTypeTransferOut,
		AffQuotaDelta:     -transferQuota,
		QuotaDelta:        transferQuota,
		AffBalanceAfter:   user.AffQuota,
		QuotaBalanceAfter: user.Quota,
		CreatedAt:         createdAt,
	})
}

func createInviteWithdrawalApplyWalletRecordTx(tx *gorm.DB, user *User, withdrawal *InviteWithdrawal) error {
	if tx == nil || user == nil || user.Id == 0 || withdrawal == nil || withdrawal.Id == 0 {
		return nil
	}
	return createInviteWalletRecordIfAbsentTx(tx, &InviteWalletRecord{
		RecordKey:         fmt.Sprintf("withdrawal_apply:%d", withdrawal.Id),
		UserId:            user.Id,
		Username:          user.Username,
		ChangeType:        InviteWalletChangeTypeWithdrawalApply,
		AffQuotaDelta:     -withdrawal.Quota,
		AffBalanceAfter:   user.AffQuota,
		QuotaBalanceAfter: user.Quota,
		WithdrawalId:      withdrawal.Id,
		WithdrawalAmount:  withdrawal.Amount,
		WithdrawalStatus:  withdrawal.Status,
		CreatedAt:         withdrawal.CreatedAt,
		Remark:            strings.TrimSpace(withdrawal.UserRemark),
	})
}

func createInviteWithdrawalRejectWalletRecordTx(tx *gorm.DB, user *User, withdrawal *InviteWithdrawal) error {
	if tx == nil || user == nil || user.Id == 0 || withdrawal == nil || withdrawal.Id == 0 {
		return nil
	}
	return createInviteWalletRecordIfAbsentTx(tx, &InviteWalletRecord{
		RecordKey:         fmt.Sprintf("withdrawal_reject_return:%d", withdrawal.Id),
		UserId:            user.Id,
		Username:          user.Username,
		ChangeType:        InviteWalletChangeTypeWithdrawalRejectReturn,
		AffQuotaDelta:     withdrawal.Quota,
		AffBalanceAfter:   user.AffQuota,
		QuotaBalanceAfter: user.Quota,
		WithdrawalId:      withdrawal.Id,
		WithdrawalAmount:  withdrawal.Amount,
		WithdrawalStatus:  withdrawal.Status,
		OperatorId:        withdrawal.OperatorId,
		OperatorName:      withdrawal.OperatorName,
		CreatedAt:         withdrawal.ProcessedAt,
		Remark:            strings.TrimSpace(withdrawal.AdminRemark),
	})
}

func createAdminInviteWalletAdjustmentRecordTx(tx *gorm.DB, user *User, changeType string, oldAffQuota int, operatorId int, operatorName string, remark string, createdAt int64) error {
	if tx == nil || user == nil || user.Id == 0 {
		return nil
	}
	if createdAt == 0 {
		createdAt = common.GetTimestamp()
	}
	return createInviteWalletRecordTx(tx, &InviteWalletRecord{
		RecordKey:         "admin_adjust:" + common.GetUUID(),
		UserId:            user.Id,
		Username:          user.Username,
		ChangeType:        changeType,
		AffQuotaDelta:     user.AffQuota - oldAffQuota,
		AffBalanceAfter:   user.AffQuota,
		QuotaBalanceAfter: user.Quota,
		OperatorId:        operatorId,
		OperatorName:      operatorName,
		Remark:            strings.TrimSpace(remark),
		CreatedAt:         createdAt,
	})
}

func backfillInviteRewardWalletRecords() error {
	var inviteRecords []InviteDetail
	if err := DB.Where("detail_type = ?", InviteDetailTypeInvite).Order("inviter_id ASC, invite_time ASC, id ASC").Find(&inviteRecords).Error; err != nil {
		return err
	}
	if len(inviteRecords) == 0 {
		return nil
	}

	var inviterIds []int
	for _, record := range inviteRecords {
		if record.InviterId != 0 {
			inviterIds = append(inviterIds, record.InviterId)
		}
	}
	var inviters []User
	if err := DB.Select("id", "username").Where("id IN ?", inviterIds).Find(&inviters).Error; err != nil {
		return err
	}
	inviterMap := make(map[int]User, len(inviters))
	for _, inviter := range inviters {
		inviterMap[inviter.Id] = inviter
	}

	var logs []inviteWalletBackfillLog
	if err := DB.Model(&Log{}).
		Select("id", "user_id", "content", "other", "created_at").
		Where("type = ? AND content LIKE ?", LogTypeSystem, "邀请用户赠送 %").
		Order("user_id ASC, created_at ASC, id ASC").
		Find(&logs).Error; err != nil {
		return err
	}
	logQuotaMap := make(map[int][]int, len(inviters))
	for _, logItem := range logs {
		parsed := parseInviteWalletLoggedQuotas(logItem.Content)
		if len(parsed) == 0 {
			continue
		}
		logQuotaMap[logItem.UserId] = append(logQuotaMap[logItem.UserId], parsed[0])
	}
	logOffsetMap := make(map[int]int, len(logQuotaMap))

	return DB.Transaction(func(tx *gorm.DB) error {
		for _, record := range inviteRecords {
			quota := 0
			if values := logQuotaMap[record.InviterId]; len(values) > logOffsetMap[record.InviterId] {
				quota = values[logOffsetMap[record.InviterId]]
				logOffsetMap[record.InviterId]++
			} else if common.QuotaForInviter > 0 {
				quota = common.QuotaForInviter
			}
			if quota <= 0 {
				continue
			}
			inviter := inviterMap[record.InviterId]
			if err := createInviteWalletRecordIfAbsentTx(tx, &InviteWalletRecord{
				RecordKey:          fmt.Sprintf("invite_reward:%d", record.InviteeId),
				UserId:             record.InviterId,
				Username:           inviter.Username,
				ChangeType:         InviteWalletChangeTypeInviteReward,
				AffQuotaDelta:      quota,
				InviteeId:          record.InviteeId,
				InviteeUsername:    record.InviteeUsername,
				InviteeDisplayName: record.InviteeDisplayName,
				CreatedAt:          record.InviteTime,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func backfillTopUpRebateWalletRecords() error {
	var topUps []TopUp
	if err := DB.Where("invite_rebate_user_id <> 0 AND invite_rebate_quota > 0").Order("id ASC").Find(&topUps).Error; err != nil {
		return err
	}
	if len(topUps) == 0 {
		return nil
	}

	userIds := make([]int, 0, len(topUps)*2)
	for _, topUp := range topUps {
		userIds = append(userIds, topUp.UserId, topUp.InviteRebateUserId)
	}
	var users []User
	if err := DB.Select("id", "username", "display_name").Where("id IN ?", userIds).Find(&users).Error; err != nil {
		return err
	}
	userMap := make(map[int]User, len(users))
	for _, user := range users {
		userMap[user.Id] = user
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		for _, topUp := range topUps {
			record := &InviteWalletRecord{
				RecordKey:          fmt.Sprintf("topup_rebate:%d", topUp.Id),
				UserId:             topUp.InviteRebateUserId,
				Username:           userMap[topUp.InviteRebateUserId].Username,
				ChangeType:         InviteWalletChangeTypeTopUpRebate,
				AffQuotaDelta:      topUp.InviteRebateQuota,
				TopUpId:            topUp.Id,
				TopUpTradeNo:       topUp.TradeNo,
				TopUpAmount:        topUp.Amount,
				TopUpMoney:         topUp.Money,
				GrantedQuota:       getTopUpGrantedQuota(&topUp),
				RebateQuota:        topUp.InviteRebateQuota,
				PaymentMethod:      topUp.PaymentMethod,
				CreatedAt:          topUp.InviteRebateTime,
				InviteeId:          topUp.UserId,
				InviteeUsername:    userMap[topUp.UserId].Username,
				InviteeDisplayName: userMap[topUp.UserId].DisplayName,
			}
			if err := createInviteWalletRecordIfAbsentTx(tx, record); err != nil {
				return err
			}
		}
		return nil
	})
}

func backfillTopUpRebateRefundWalletRecords() error {
	var refunds []TopUpRefund
	if err := DB.Where("status = ? AND invite_rebate_delta > 0", TopUpRefundStatusSuccess).Order("id ASC").Find(&refunds).Error; err != nil {
		return err
	}
	if len(refunds) == 0 {
		return nil
	}

	topUpIds := make([]int, 0, len(refunds))
	for _, refund := range refunds {
		topUpIds = append(topUpIds, refund.TopUpId)
	}
	var topUps []TopUp
	if err := DB.Where("id IN ?", topUpIds).Find(&topUps).Error; err != nil {
		return err
	}
	topUpMap := make(map[int]TopUp, len(topUps))
	userIds := make([]int, 0, len(topUps)*2)
	for _, topUp := range topUps {
		topUpMap[topUp.Id] = topUp
		userIds = append(userIds, topUp.UserId, topUp.InviteRebateUserId)
	}
	var users []User
	if err := DB.Select("id", "username", "display_name").Where("id IN ?", userIds).Find(&users).Error; err != nil {
		return err
	}
	userMap := make(map[int]User, len(users))
	for _, user := range users {
		userMap[user.Id] = user
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		for _, refund := range refunds {
			topUp, ok := topUpMap[refund.TopUpId]
			if !ok || topUp.InviteRebateUserId == 0 {
				continue
			}
			record := &InviteWalletRecord{
				RecordKey:          fmt.Sprintf("topup_rebate_refund:%d", refund.Id),
				UserId:             topUp.InviteRebateUserId,
				Username:           userMap[topUp.InviteRebateUserId].Username,
				ChangeType:         InviteWalletChangeTypeTopUpRebateRefund,
				AffQuotaDelta:      -refund.InviteRebateDelta,
				TopUpId:            topUp.Id,
				TopUpTradeNo:       topUp.TradeNo,
				TopUpAmount:        topUp.Amount,
				TopUpMoney:         topUp.Money,
				GrantedQuota:       getTopUpGrantedQuota(&topUp),
				RebateQuota:        refund.InviteRebateDelta,
				PaymentMethod:      topUp.PaymentMethod,
				TopUpRefundId:      refund.Id,
				RefundAmount:       refund.RefundAmount,
				InviteeId:          topUp.UserId,
				InviteeUsername:    userMap[topUp.UserId].Username,
				InviteeDisplayName: userMap[topUp.UserId].DisplayName,
				CreatedAt:          refund.CompleteTime,
				Remark:             "历史退款回填，若当时邀请余额不足，主余额扣减部分可能未精确还原",
			}
			if record.CreatedAt == 0 {
				record.CreatedAt = refund.UpdateTime
			}
			if record.CreatedAt == 0 {
				record.CreatedAt = refund.CreateTime
			}
			if err := createInviteWalletRecordIfAbsentTx(tx, record); err != nil {
				return err
			}
		}
		return nil
	})
}

func backfillInviteWithdrawalWalletRecords() error {
	var withdrawals []InviteWithdrawal
	if err := DB.Order("id ASC").Find(&withdrawals).Error; err != nil {
		return err
	}
	if len(withdrawals) == 0 {
		return nil
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		for _, withdrawal := range withdrawals {
			applyRecord := &InviteWalletRecord{
				RecordKey:        fmt.Sprintf("withdrawal_apply:%d", withdrawal.Id),
				UserId:           withdrawal.UserId,
				Username:         withdrawal.Username,
				ChangeType:       InviteWalletChangeTypeWithdrawalApply,
				AffQuotaDelta:    -withdrawal.Quota,
				WithdrawalId:     withdrawal.Id,
				WithdrawalAmount: withdrawal.Amount,
				WithdrawalStatus: withdrawal.Status,
				CreatedAt:        withdrawal.CreatedAt,
				Remark:           strings.TrimSpace(withdrawal.UserRemark),
			}
			if err := createInviteWalletRecordIfAbsentTx(tx, applyRecord); err != nil {
				return err
			}
			if withdrawal.Status != InviteWithdrawalStatusRejected {
				continue
			}
			rejectRecord := &InviteWalletRecord{
				RecordKey:        fmt.Sprintf("withdrawal_reject_return:%d", withdrawal.Id),
				UserId:           withdrawal.UserId,
				Username:         withdrawal.Username,
				ChangeType:       InviteWalletChangeTypeWithdrawalRejectReturn,
				AffQuotaDelta:    withdrawal.Quota,
				WithdrawalId:     withdrawal.Id,
				WithdrawalAmount: withdrawal.Amount,
				WithdrawalStatus: withdrawal.Status,
				OperatorId:       withdrawal.OperatorId,
				OperatorName:     withdrawal.OperatorName,
				CreatedAt:        withdrawal.ProcessedAt,
				Remark:           strings.TrimSpace(withdrawal.AdminRemark),
			}
			if err := createInviteWalletRecordIfAbsentTx(tx, rejectRecord); err != nil {
				return err
			}
		}
		return nil
	})
}

func backfillInviteWalletManageLogs() error {
	var logs []inviteWalletBackfillLog
	if err := DB.Model(&Log{}).
		Select("id", "user_id", "content", "other", "created_at").
		Where("type = ? AND (content LIKE ? OR content LIKE ? OR content LIKE ? OR content LIKE ?)", LogTypeManage, "管理员增加邀请余额 %", "管理员减少邀请余额 %", "管理员覆盖邀请余额从 %", "划转邀请余额 %").
		Order("id ASC").
		Find(&logs).Error; err != nil {
		return err
	}
	if len(logs) == 0 {
		return nil
	}

	userIds := make([]int, 0, len(logs))
	for _, logItem := range logs {
		userIds = append(userIds, logItem.UserId)
	}
	var users []User
	if err := DB.Select("id", "username").Where("id IN ?", userIds).Find(&users).Error; err != nil {
		return err
	}
	userMap := make(map[int]User, len(users))
	for _, user := range users {
		userMap[user.Id] = user
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		for _, logItem := range logs {
			record := &InviteWalletRecord{
				RecordKey: "manage_log:" + fmt.Sprintf("%d", logItem.Id),
				UserId:    logItem.UserId,
				Username:  userMap[logItem.UserId].Username,
				CreatedAt: logItem.CreatedAt,
				Remark:    "历史日志回填",
			}
			parsedQuotas := parseInviteWalletLoggedQuotas(logItem.Content)
			switch {
			case strings.HasPrefix(logItem.Content, "管理员增加邀请余额"):
				if len(parsedQuotas) == 0 {
					continue
				}
				record.ChangeType = InviteWalletChangeTypeAdminAdd
				record.AffQuotaDelta = parsedQuotas[0]
			case strings.HasPrefix(logItem.Content, "管理员减少邀请余额"):
				if len(parsedQuotas) == 0 {
					continue
				}
				record.ChangeType = InviteWalletChangeTypeAdminSubtract
				record.AffQuotaDelta = -parsedQuotas[0]
			case strings.HasPrefix(logItem.Content, "管理员覆盖邀请余额从"):
				if len(parsedQuotas) < 2 {
					continue
				}
				record.ChangeType = InviteWalletChangeTypeAdminOverride
				record.AffQuotaDelta = parsedQuotas[1] - parsedQuotas[0]
				record.AffBalanceAfter = parsedQuotas[1]
			case strings.HasPrefix(logItem.Content, "划转邀请余额"):
				if len(parsedQuotas) == 0 {
					continue
				}
				record.ChangeType = InviteWalletChangeTypeTransferOut
				record.AffQuotaDelta = -parsedQuotas[0]
				record.QuotaDelta = parsedQuotas[0]
			default:
				continue
			}

			otherMap, _ := common.StrToMap(logItem.Other)
			if adminInfo, ok := otherMap["admin_info"].(map[string]interface{}); ok {
				if operatorId, ok := adminInfo["admin_id"].(float64); ok {
					record.OperatorId = int(operatorId)
				}
				if operatorName, ok := adminInfo["admin_username"].(string); ok {
					record.OperatorName = operatorName
				}
			}
			if err := createInviteWalletRecordIfAbsentTx(tx, record); err != nil {
				return err
			}
		}
		return nil
	})
}

func BackfillInviteWalletRecords() error {
	common.SysLog("invite wallet record backfill started")
	if err := backfillInviteRewardWalletRecords(); err != nil {
		return err
	}
	if err := backfillTopUpRebateWalletRecords(); err != nil {
		return err
	}
	if err := backfillTopUpRebateRefundWalletRecords(); err != nil {
		return err
	}
	if err := backfillInviteWithdrawalWalletRecords(); err != nil {
		return err
	}
	if err := backfillInviteWalletManageLogs(); err != nil {
		return err
	}
	common.SysLog("invite wallet record backfill completed")
	return nil
}

func GetInviteWalletRecordsByUserId(userId int) ([]*InviteWalletRecord, error) {
	var records []*InviteWalletRecord
	err := DB.Where("user_id = ?", userId).Order("created_at DESC, id DESC").Find(&records).Error
	return records, err
}

func GetInviteRebateTopUpsByInviterId(inviterId int) ([]*AdminTopUpItem, error) {
	var topUps []*TopUp
	if err := DB.Where("invite_rebate_user_id = ?", inviterId).Order("id DESC").Find(&topUps).Error; err != nil {
		return nil, err
	}
	return BuildAdminTopUpItems(topUps)
}

func GetInviteWalletOverviewByUserId(userId int) (*InviteWalletOverview, error) {
	user := &User{}
	if err := DB.Select("id", "username", "display_name", "aff_count", "aff_quota", "aff_history").Where("id = ?", userId).First(user).Error; err != nil {
		return nil, err
	}
	inviteRecords, err := GetInviteRecordsByInviterId(userId)
	if err != nil {
		return nil, err
	}
	rebateRecords, err := GetInviteRebateRecordsByInviterId(userId)
	if err != nil {
		return nil, err
	}
	walletRecords, err := GetInviteWalletRecordsByUserId(userId)
	if err != nil {
		return nil, err
	}
	withdrawals, err := GetInviteWithdrawalsByUserId(userId)
	if err != nil {
		return nil, err
	}
	topUpOrders, err := GetInviteRebateTopUpsByInviterId(userId)
	if err != nil {
		return nil, err
	}
	return &InviteWalletOverview{
		Summary: &InviteWalletOverviewSummary{
			Id:              user.Id,
			Username:        user.Username,
			DisplayName:     user.DisplayName,
			AffCount:        user.AffCount,
			AffQuota:        user.AffQuota,
			AffHistoryQuota: user.AffHistoryQuota,
		},
		InviteRecords: inviteRecords,
		RebateRecords: rebateRecords,
		WalletRecords: walletRecords,
		Withdrawals:   withdrawals,
		TopUpOrders:   topUpOrders,
	}, nil
}

func AdjustInviteQuotaByAdmin(userId int, mode string, value int, operatorId int, operatorName string) (*User, error) {
	if userId == 0 {
		return nil, errors.New("用户不存在")
	}
	if value < 0 {
		return nil, errors.New("调整额度不能为负数")
	}
	operatorName = strings.TrimSpace(operatorName)
	mode = strings.TrimSpace(mode)

	var updatedUser User
	now := common.GetTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Select("id", "username", "quota", "aff_quota", "aff_history").
			Where("id = ?", userId).
			First(&updatedUser).Error; err != nil {
			return err
		}
		oldAffQuota := updatedUser.AffQuota
		changeType := ""
		remark := ""

		switch mode {
		case "add":
			if value <= 0 {
				return errors.New("额度不能为 0")
			}
			updatedUser.AffQuota += value
			updatedUser.AffHistoryQuota += value
			changeType = InviteWalletChangeTypeAdminAdd
			remark = "管理员增加邀请余额"
		case "subtract":
			if value <= 0 {
				return errors.New("额度不能为 0")
			}
			if updatedUser.AffQuota < value {
				return errors.New("邀请余额不足，无法减少")
			}
			updatedUser.AffQuota -= value
			changeType = InviteWalletChangeTypeAdminSubtract
			remark = "管理员减少邀请余额"
		case "override":
			if value > updatedUser.AffHistoryQuota {
				updatedUser.AffHistoryQuota = value
			}
			updatedUser.AffQuota = value
			changeType = InviteWalletChangeTypeAdminOverride
			remark = "管理员覆盖邀请余额"
		default:
			return errors.New("无效的操作模式")
		}

		if err := tx.Model(&User{}).Where("id = ?", updatedUser.Id).Updates(map[string]any{
			"aff_quota":   updatedUser.AffQuota,
			"aff_history": updatedUser.AffHistoryQuota,
		}).Error; err != nil {
			return err
		}
		return createAdminInviteWalletAdjustmentRecordTx(tx, &updatedUser, changeType, oldAffQuota, operatorId, operatorName, remark, now)
	})
	if err != nil {
		return nil, err
	}
	return &updatedUser, nil
}
