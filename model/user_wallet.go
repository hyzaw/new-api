package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"gorm.io/gorm"
)

type WalletQuotaAllocation struct {
	BaseQuota         int  `json:"base_quota,omitempty"`
	BaseGiftQuota     int  `json:"base_gift_quota,omitempty"`
	ConsumedQuota     int  `json:"consumed_quota,omitempty"`
	ConsumedGiftQuota int  `json:"consumed_gift_quota,omitempty"`
	GiftEligible      bool `json:"gift_eligible,omitempty"`
}

type UserWalletSummary struct {
	Quota            int  `json:"quota"`
	GiftQuota        int  `json:"gift_quota"`
	AvailableQuota   int  `json:"available_quota"`
	GiftEligible     bool `json:"gift_eligible"`
	AvailableGiftUse int  `json:"available_gift_use"`
}

func (allocation *WalletQuotaAllocation) normalize() {
	if allocation.BaseQuota < 0 {
		allocation.BaseQuota = 0
	}
	if allocation.BaseGiftQuota < 0 {
		allocation.BaseGiftQuota = 0
	}
	if allocation.ConsumedQuota < 0 {
		allocation.ConsumedQuota = 0
	}
	if allocation.ConsumedGiftQuota < 0 {
		allocation.ConsumedGiftQuota = 0
	}
}

func (allocation *WalletQuotaAllocation) TotalConsumed() int {
	if allocation == nil {
		return 0
	}
	allocation.normalize()
	return allocation.ConsumedQuota + allocation.ConsumedGiftQuota
}

func (allocation *WalletQuotaAllocation) splitForTotal(total int) (int, int) {
	if allocation == nil {
		return total, 0
	}
	allocation.normalize()
	if total < 0 {
		total = 0
	}
	if !allocation.GiftEligible {
		return total, 0
	}
	giftConsumed := total
	if giftConsumed > allocation.BaseGiftQuota {
		giftConsumed = allocation.BaseGiftQuota
	}
	return total - giftConsumed, giftConsumed
}

func (allocation *WalletQuotaAllocation) ApplyTarget(total int) {
	if allocation == nil {
		return
	}
	quotaConsumed, giftConsumed := allocation.splitForTotal(total)
	allocation.ConsumedQuota = quotaConsumed
	allocation.ConsumedGiftQuota = giftConsumed
}

func (allocation *WalletQuotaAllocation) DeltasForTarget(total int) (int, int) {
	if allocation == nil {
		return total, 0
	}
	targetQuota, targetGift := allocation.splitForTotal(total)
	return targetQuota - allocation.ConsumedQuota, targetGift - allocation.ConsumedGiftQuota
}

func GetUserWalletSummary(userId int, group string, modelName string) (*UserWalletSummary, error) {
	user, err := GetUserById(userId, false)
	if err != nil {
		return nil, err
	}
	giftEligible := operation_setting.IsGiftQuotaAllowed(group, modelName)
	availableQuota := user.Quota
	availableGiftUse := 0
	if giftEligible {
		availableGiftUse = user.GiftQuota
		availableQuota += user.GiftQuota
	}
	return &UserWalletSummary{
		Quota:            user.Quota,
		GiftQuota:        user.GiftQuota,
		AvailableQuota:   availableQuota,
		GiftEligible:     giftEligible,
		AvailableGiftUse: availableGiftUse,
	}, nil
}

func GetUserWalletAllocation(userId int, group string, modelName string) (*WalletQuotaAllocation, error) {
	summary, err := GetUserWalletSummary(userId, group, modelName)
	if err != nil {
		return nil, err
	}
	return &WalletQuotaAllocation{
		BaseQuota:     summary.Quota,
		BaseGiftQuota: summary.GiftQuota,
		GiftEligible:  summary.GiftEligible,
	}, nil
}

func ApplyUserWalletTarget(userId int, allocation *WalletQuotaAllocation, targetTotal int) error {
	if allocation == nil {
		return errors.New("wallet allocation is nil")
	}
	allocation.normalize()
	if targetTotal < 0 {
		targetTotal = 0
	}
	quotaDelta, giftDelta := allocation.DeltasForTarget(targetTotal)
	if quotaDelta == 0 && giftDelta == 0 {
		return nil
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		user := &User{}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Select("id", "quota", "gift_quota").First(user, "id = ?", userId).Error; err != nil {
			return err
		}
		if quotaDelta > 0 && user.Quota < quotaDelta {
			return errors.New("账户余额不足")
		}
		if giftDelta > 0 && user.GiftQuota < giftDelta {
			return errors.New("赠送余额不足")
		}
		nextQuota := user.Quota - quotaDelta
		nextGiftQuota := user.GiftQuota - giftDelta
		return tx.Model(&User{}).Where("id = ?", userId).Updates(map[string]interface{}{
			"quota":      nextQuota,
			"gift_quota": nextGiftQuota,
		}).Error
	})
	if err != nil {
		return err
	}

	allocation.ApplyTarget(targetTotal)
	if err := InvalidateUserCache(userId); err != nil {
		common.SysLog("failed to invalidate user cache after wallet target adjustment: " + err.Error())
	}
	return nil
}

func IncreaseUserGiftQuota(userId int, quota int) error {
	if quota < 0 {
		return errors.New("gift quota 不能为负数！")
	}
	if quota == 0 {
		return nil
	}
	if err := DB.Model(&User{}).Where("id = ?", userId).Update("gift_quota", gorm.Expr("gift_quota + ?", quota)).Error; err != nil {
		return err
	}
	if err := InvalidateUserCache(userId); err != nil {
		common.SysLog("failed to invalidate user cache after increasing gift quota: " + err.Error())
	}
	return nil
}

func DecreaseUserGiftQuota(userId int, quota int) error {
	if quota < 0 {
		return errors.New("gift quota 不能为负数！")
	}
	if quota == 0 {
		return nil
	}
	if err := DB.Model(&User{}).Where("id = ? AND gift_quota >= ?", userId, quota).Update("gift_quota", gorm.Expr("gift_quota - ?", quota)).Error; err != nil {
		return err
	}
	if err := InvalidateUserCache(userId); err != nil {
		common.SysLog("failed to invalidate user cache after decreasing gift quota: " + err.Error())
	}
	return nil
}
