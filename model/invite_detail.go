package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	InviteDetailTypeInvite = "invite"
	InviteDetailTypeRebate = "rebate"
)

type InviteDetail struct {
	Id                  int     `json:"id"`
	DetailKey           string  `json:"detail_key" gorm:"type:varchar(64);uniqueIndex"`
	DetailType          string  `json:"detail_type" gorm:"type:varchar(16);index"`
	InviterId           int     `json:"inviter_id" gorm:"index"`
	InviteeId           int     `json:"invitee_id" gorm:"index"`
	InviteeUsername     string  `json:"invitee_username" gorm:"type:varchar(64);default:''"`
	InviteeDisplayName  string  `json:"invitee_display_name" gorm:"type:varchar(64);default:''"`
	InviteTime          int64   `json:"invite_time" gorm:"bigint;default:0;index"`
	TopUpId             int     `json:"top_up_id" gorm:"default:0;index"`
	TopUpTradeNo        string  `json:"top_up_trade_no" gorm:"type:varchar(255);default:'';index"`
	TopUpAmount         int64   `json:"top_up_amount" gorm:"default:0"`
	TopUpMoney          float64 `json:"top_up_money" gorm:"default:0"`
	GrantedQuota        int     `json:"granted_quota" gorm:"default:0"`
	PaymentMethod       string  `json:"payment_method" gorm:"type:varchar(50);default:''"`
	RebateQuota         int     `json:"rebate_quota" gorm:"default:0"`
	RebateRefundedQuota int     `json:"rebate_refunded_quota" gorm:"default:0"`
	RebateTime          int64   `json:"rebate_time" gorm:"bigint;default:0;index"`
	CreatedAt           int64   `json:"created_at" gorm:"bigint;default:0"`
	UpdatedAt           int64   `json:"updated_at" gorm:"bigint;default:0"`
}

type inviteTimeRow struct {
	UserId     int   `gorm:"column:user_id"`
	InviteTime int64 `gorm:"column:invite_time"`
}

func getInviteDetailInviteTimeMap(tx *gorm.DB, inviteeIds []int) (map[int]int64, error) {
	if len(inviteeIds) == 0 {
		return map[int]int64{}, nil
	}
	var rows []inviteTimeRow
	err := tx.Model(&Log{}).
		Select("user_id, MIN(created_at) AS invite_time").
		Where("user_id IN ? AND type = ? AND content LIKE ?", inviteeIds, LogTypeSystem, "使用邀请码赠送 %").
		Group("user_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int]int64, len(rows))
	for _, row := range rows {
		result[row.UserId] = row.InviteTime
	}
	return result, nil
}

func upsertInviteDetailTx(tx *gorm.DB, detail *InviteDetail) error {
	if tx == nil || detail == nil {
		return errors.New("invite detail is nil")
	}
	now := common.GetTimestamp()
	var existing InviteDetail
	err := tx.Where("detail_key = ?", detail.DetailKey).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if detail.CreatedAt == 0 {
				detail.CreatedAt = now
			}
			detail.UpdatedAt = now
			return tx.Create(detail).Error
		}
		return err
	}
	if detail.CreatedAt == 0 {
		detail.CreatedAt = existing.CreatedAt
	}
	detail.UpdatedAt = now
	return tx.Model(&existing).Updates(map[string]any{
		"detail_type":           detail.DetailType,
		"inviter_id":            detail.InviterId,
		"invitee_id":            detail.InviteeId,
		"invitee_username":      detail.InviteeUsername,
		"invitee_display_name":  detail.InviteeDisplayName,
		"invite_time":           detail.InviteTime,
		"top_up_id":             detail.TopUpId,
		"top_up_trade_no":       detail.TopUpTradeNo,
		"top_up_amount":         detail.TopUpAmount,
		"top_up_money":          detail.TopUpMoney,
		"granted_quota":         detail.GrantedQuota,
		"payment_method":        detail.PaymentMethod,
		"rebate_quota":          detail.RebateQuota,
		"rebate_refunded_quota": detail.RebateRefundedQuota,
		"rebate_time":           detail.RebateTime,
		"updated_at":            detail.UpdatedAt,
	}).Error
}

func SyncInviteRegistrationDetail(inviterId int, invitee *User, inviteTime int64) error {
	return syncInviteRegistrationDetailTx(DB, inviterId, invitee, inviteTime)
}

func syncInviteRegistrationDetailTx(tx *gorm.DB, inviterId int, invitee *User, inviteTime int64) error {
	if inviterId == 0 || invitee == nil || invitee.Id == 0 {
		return nil
	}
	if inviteTime == 0 {
		inviteTime = common.GetTimestamp()
	}
	return upsertInviteDetailTx(tx, &InviteDetail{
		DetailKey:          fmt.Sprintf("invite:%d", invitee.Id),
		DetailType:         InviteDetailTypeInvite,
		InviterId:          inviterId,
		InviteeId:          invitee.Id,
		InviteeUsername:    invitee.Username,
		InviteeDisplayName: invitee.DisplayName,
		InviteTime:         inviteTime,
	})
}

func syncInviteRebateDetailTx(tx *gorm.DB, topUp *TopUp, invitee *User) error {
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
	detail := &InviteDetail{
		DetailKey:           fmt.Sprintf("rebate:%d", topUp.Id),
		DetailType:          InviteDetailTypeRebate,
		InviterId:           topUp.InviteRebateUserId,
		TopUpId:             topUp.Id,
		TopUpTradeNo:        topUp.TradeNo,
		TopUpAmount:         topUp.Amount,
		TopUpMoney:          topUp.Money,
		GrantedQuota:        getTopUpGrantedQuota(topUp),
		PaymentMethod:       topUp.PaymentMethod,
		RebateQuota:         topUp.InviteRebateQuota,
		RebateRefundedQuota: topUp.InviteRebateRefundedQuota,
		RebateTime:          topUp.InviteRebateTime,
	}
	if invitee != nil {
		detail.InviteeId = invitee.Id
		detail.InviteeUsername = invitee.Username
		detail.InviteeDisplayName = invitee.DisplayName
	}
	return upsertInviteDetailTx(tx, detail)
}

func backfillInviteRegistrationDetails() error {
	var invitees []User
	return DB.Model(&User{}).
		Select("id", "username", "display_name", "inviter_id").
		Where("inviter_id <> 0").
		Order("id ASC").
		FindInBatches(&invitees, 200, func(tx *gorm.DB, _ int) error {
			if len(invitees) == 0 {
				return nil
			}
			inviteeIds := make([]int, 0, len(invitees))
			for _, invitee := range invitees {
				inviteeIds = append(inviteeIds, invitee.Id)
			}
			inviteTimeMap, err := getInviteDetailInviteTimeMap(tx, inviteeIds)
			if err != nil {
				return err
			}
			for i := range invitees {
				if err := syncInviteRegistrationDetailTx(tx, invitees[i].InviterId, &invitees[i], inviteTimeMap[invitees[i].Id]); err != nil {
					return err
				}
			}
			return nil
		}).Error
}

func backfillInviteRebateDetails() error {
	var topUps []TopUp
	return DB.Model(&TopUp{}).
		Where("invite_rebate_user_id <> 0 AND invite_rebate_quota > 0").
		Order("id ASC").
		FindInBatches(&topUps, 200, func(tx *gorm.DB, _ int) error {
			if len(topUps) == 0 {
				return nil
			}
			userIds := make([]int, 0, len(topUps))
			for _, topUp := range topUps {
				userIds = append(userIds, topUp.UserId)
			}
			var users []User
			if err := tx.Select("id", "username", "display_name").Where("id IN ?", userIds).Find(&users).Error; err != nil {
				return err
			}
			userMap := make(map[int]*User, len(users))
			for i := range users {
				userMap[users[i].Id] = &users[i]
			}
			for i := range topUps {
				if err := syncInviteRebateDetailTx(tx, &topUps[i], userMap[topUps[i].UserId]); err != nil {
					return err
				}
			}
			return nil
		}).Error
}

func BackfillInviteDetails() error {
	common.SysLog("invite detail backfill started")
	if err := backfillInviteRegistrationDetails(); err != nil {
		return err
	}
	if err := backfillInviteRebateDetails(); err != nil {
		return err
	}
	common.SysLog("invite detail backfill completed")
	return nil
}

func GetInviteRecordsByInviterId(inviterId int) ([]*InviteDetail, error) {
	var details []*InviteDetail
	err := DB.Where("inviter_id = ? AND detail_type = ?", inviterId, InviteDetailTypeInvite).
		Order("invite_time DESC, id DESC").
		Find(&details).Error
	return details, err
}

func GetInviteRebateRecordsByInviterId(inviterId int) ([]*InviteDetail, error) {
	var details []*InviteDetail
	err := DB.Where("inviter_id = ? AND detail_type = ?", inviterId, InviteDetailTypeRebate).
		Order("rebate_time DESC, id DESC").
		Find(&details).Error
	return details, err
}
