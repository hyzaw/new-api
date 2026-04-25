package model

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"gorm.io/gorm"
)

var ErrInvalidRedemptionQuota = errors.New("通用余额和赠送余额不能同时为 0")
var ErrInvalidLotteryRedemptionQuota = errors.New("抽奖兑换码额度配置无效")
var ErrLotteryRedemptionAlreadyRedeemed = errors.New("该抽奖兑换码每个用户只能兑换一次")
var ErrLotteryRedemptionExhausted = errors.New("该抽奖兑换码已领取完")

const (
	RedemptionTypeNormal  = "normal"
	RedemptionTypeLottery = "lottery"

	RedemptionLotteryModeRange   = "range"
	RedemptionLotteryModeChoices = "choices"

	RedemptionLotteryBalanceQuota = "quota"
	RedemptionLotteryBalanceGift  = "gift_quota"
)

type Redemption struct {
	Id                  int            `json:"id"`
	UserId              int            `json:"user_id"`
	Key                 string         `json:"key" gorm:"type:varchar(64);uniqueIndex"`
	Status              int            `json:"status" gorm:"default:1"`
	Name                string         `json:"name" gorm:"index"`
	Quota               int            `json:"quota" gorm:"default:100"`
	GiftQuota           int            `json:"gift_quota" gorm:"default:0;column:gift_quota"`
	CreatedTime         int64          `json:"created_time" gorm:"bigint"`
	RedeemedTime        int64          `json:"redeemed_time" gorm:"bigint"`
	Count               int            `json:"count" gorm:"-:all"` // only for api request
	UsedUserId          int            `json:"used_user_id"`
	DeletedAt           gorm.DeletedAt `gorm:"index"`
	ExpiredTime         int64          `json:"expired_time" gorm:"bigint"` // 过期时间，0 表示不过期
	Type                string         `json:"type" gorm:"type:varchar(20);default:'normal';index"`
	LotteryMode         string         `json:"lottery_mode" gorm:"type:varchar(20);default:''"`
	LotteryQuotaMin     int            `json:"lottery_quota_min" gorm:"default:0"`
	LotteryQuotaMax     int            `json:"lottery_quota_max" gorm:"default:0"`
	LotteryQuotaChoices string         `json:"lottery_quota_choices" gorm:"type:text"`
	LotteryBalanceType  string         `json:"lottery_balance_type" gorm:"type:varchar(20);default:'quota'"`
	MaxRedeemCount      int            `json:"max_redeem_count" gorm:"default:0"`
	RedeemedCount       int            `json:"redeemed_count" gorm:"default:0"`
}

type RedemptionRecord struct {
	Id           int   `json:"id"`
	RedemptionId int   `json:"redemption_id" gorm:"index:idx_redemption_records_redemption_user,unique,priority:1;index"`
	UserId       int   `json:"user_id" gorm:"index:idx_redemption_records_redemption_user,unique,priority:2;index"`
	Quota        int   `json:"quota" gorm:"default:0"`
	GiftQuota    int   `json:"gift_quota" gorm:"default:0;column:gift_quota"`
	CreatedTime  int64 `json:"created_time" gorm:"bigint"`
}

type LotteryQuotaChoice struct {
	Quota  int
	Weight int
}

type RedeemResult struct {
	Quota      int `json:"quota"`
	GiftQuota  int `json:"gift_quota"`
	TotalQuota int `json:"total_quota"`
}

func GetAllRedemptions(startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	// 开始事务
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取总数
	err = tx.Model(&Redemption{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 获取分页数据
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 提交事务
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func SearchRedemptions(keyword string, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Build query based on keyword type
	query := tx.Model(&Redemption{})

	// Only try to convert to ID if the string represents a valid integer
	if id, err := strconv.Atoi(keyword); err == nil {
		query = query.Where("id = ? OR name LIKE ?", id, keyword+"%")
	} else {
		query = query.Where("name LIKE ?", keyword+"%")
	}

	// Get total count
	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated data
	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func GetRedemptionById(id int) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	var err error = nil
	err = DB.First(&redemption, "id = ?", id).Error
	return &redemption, err
}

func (redemption *Redemption) IsLottery() bool {
	return redemption.Type == RedemptionTypeLottery
}

func (redemption *Redemption) NormalizeType() {
	if redemption.Type == "" {
		redemption.Type = RedemptionTypeNormal
	}
	if redemption.LotteryBalanceType == "" {
		redemption.LotteryBalanceType = RedemptionLotteryBalanceQuota
	}
}

func ParseLotteryQuotaChoices(raw string) ([]LotteryQuotaChoice, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	choices := make([]LotteryQuotaChoice, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		quotaPart := part
		weight := 1
		if strings.Contains(part, ":") {
			segments := strings.SplitN(part, ":", 2)
			quotaPart = strings.TrimSpace(segments[0])
			parsedWeight, err := strconv.Atoi(strings.TrimSpace(segments[1]))
			if err != nil || parsedWeight <= 0 {
				return nil, ErrInvalidLotteryRedemptionQuota
			}
			weight = parsedWeight
		}
		quota, err := strconv.Atoi(quotaPart)
		if err != nil || quota <= 0 {
			return nil, ErrInvalidLotteryRedemptionQuota
		}
		choices = append(choices, LotteryQuotaChoice{Quota: quota, Weight: weight})
	}
	if len(choices) == 0 {
		return nil, ErrInvalidLotteryRedemptionQuota
	}
	return choices, nil
}

func ValidateLotteryRedemption(redemption *Redemption) error {
	redemption.NormalizeType()
	if !redemption.IsLottery() {
		if redemption.Quota <= 0 && redemption.GiftQuota <= 0 {
			return ErrInvalidRedemptionQuota
		}
		return nil
	}
	switch redemption.LotteryMode {
	case RedemptionLotteryModeRange:
		if redemption.LotteryQuotaMin <= 0 || redemption.LotteryQuotaMax < redemption.LotteryQuotaMin {
			return ErrInvalidLotteryRedemptionQuota
		}
	case RedemptionLotteryModeChoices:
		if _, err := ParseLotteryQuotaChoices(redemption.LotteryQuotaChoices); err != nil {
			return err
		}
	default:
		return ErrInvalidLotteryRedemptionQuota
	}
	if redemption.MaxRedeemCount < 0 {
		return ErrInvalidLotteryRedemptionQuota
	}
	if redemption.LotteryBalanceType != RedemptionLotteryBalanceQuota && redemption.LotteryBalanceType != RedemptionLotteryBalanceGift {
		return ErrInvalidLotteryRedemptionQuota
	}
	return nil
}

func randomIntInclusive(min int, max int) (int, error) {
	if min < 0 || max < min {
		return 0, ErrInvalidLotteryRedemptionQuota
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return 0, err
	}
	return min + int(n.Int64()), nil
}

func (redemption *Redemption) DrawLotteryQuota() (int, error) {
	switch redemption.LotteryMode {
	case RedemptionLotteryModeRange:
		return randomIntInclusive(redemption.LotteryQuotaMin, redemption.LotteryQuotaMax)
	case RedemptionLotteryModeChoices:
		choices, err := ParseLotteryQuotaChoices(redemption.LotteryQuotaChoices)
		if err != nil {
			return 0, err
		}
		totalWeight := 0
		for _, choice := range choices {
			totalWeight += choice.Weight
		}
		idx, err := randomIntInclusive(1, totalWeight)
		if err != nil {
			return 0, err
		}
		current := 0
		for _, choice := range choices {
			current += choice.Weight
			if idx <= current {
				return choice.Quota, nil
			}
		}
		return 0, ErrInvalidLotteryRedemptionQuota
	default:
		return 0, ErrInvalidLotteryRedemptionQuota
	}
}

func applyRedemptionQuota(tx *gorm.DB, userId int, quota int, giftQuota int) error {
	updates := map[string]interface{}{}
	if quota > 0 {
		updates["quota"] = gorm.Expr("quota + ?", quota)
	}
	if giftQuota > 0 {
		updates["gift_quota"] = gorm.Expr("gift_quota + ?", giftQuota)
	}
	if len(updates) == 0 {
		return errors.New("兑换码额度无效")
	}
	return tx.Model(&User{}).Where("id = ?", userId).Updates(updates).Error
}

func redeemLottery(tx *gorm.DB, redemption *Redemption, userId int) (*RedeemResult, error) {
	var count int64
	err := tx.Model(&RedemptionRecord{}).Where("redemption_id = ? AND user_id = ?", redemption.Id, userId).Count(&count).Error
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrLotteryRedemptionAlreadyRedeemed
	}
	if redemption.MaxRedeemCount > 0 && redemption.RedeemedCount >= redemption.MaxRedeemCount {
		return nil, ErrLotteryRedemptionExhausted
	}
	quota, err := redemption.DrawLotteryQuota()
	if err != nil {
		return nil, err
	}
	giftQuota := 0
	if redemption.LotteryBalanceType == RedemptionLotteryBalanceGift {
		giftQuota = quota
		quota = 0
	}
	record := RedemptionRecord{
		RedemptionId: redemption.Id,
		UserId:       userId,
		Quota:        quota,
		GiftQuota:    giftQuota,
		CreatedTime:  common.GetTimestamp(),
	}
	if err = tx.Create(&record).Error; err != nil {
		return nil, ErrLotteryRedemptionAlreadyRedeemed
	}
	if err = applyRedemptionQuota(tx, userId, quota, giftQuota); err != nil {
		return nil, err
	}
	redemption.RedeemedTime = common.GetTimestamp()
	redemption.UsedUserId = userId
	if err = tx.Model(redemption).Select("redeemed_time", "used_user_id", "redeemed_count").Updates(map[string]interface{}{
		"redeemed_time":  redemption.RedeemedTime,
		"used_user_id":   userId,
		"redeemed_count": gorm.Expr("redeemed_count + ?", 1),
	}).Error; err != nil {
		return nil, err
	}
	redemption.Quota = quota
	redemption.GiftQuota = giftQuota
	redemption.RedeemedCount++
	return &RedeemResult{
		Quota:      quota,
		GiftQuota:  giftQuota,
		TotalQuota: quota + giftQuota,
	}, nil
}

func Redeem(key string, userId int) (result *RedeemResult, err error) {
	if key == "" {
		return nil, errors.New("未提供兑换码")
	}
	if userId == 0 {
		return nil, errors.New("无效的 user id")
	}
	redemption := &Redemption{}

	keyCol := "`key`"
	if common.UsingPostgreSQL {
		keyCol = `"key"`
	}
	common.RandomSleep()
	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(keyCol+" = ?", key).First(redemption).Error
		if err != nil {
			return errors.New("无效的兑换码")
		}
		redemption.NormalizeType()
		if redemption.Status != common.RedemptionCodeStatusEnabled {
			if redemption.IsLottery() {
				return errors.New("该兑换码已被禁用")
			}
			return errors.New("该兑换码已被使用")
		}
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
			return errors.New("该兑换码已过期")
		}
		if redemption.IsLottery() {
			result, err = redeemLottery(tx, redemption, userId)
			return err
		}
		if err = applyRedemptionQuota(tx, userId, redemption.Quota, redemption.GiftQuota); err != nil {
			return err
		}
		redemption.RedeemedTime = common.GetTimestamp()
		redemption.Status = common.RedemptionCodeStatusUsed
		redemption.UsedUserId = userId
		if err = tx.Save(redemption).Error; err != nil {
			return err
		}
		result = &RedeemResult{
			Quota:      redemption.Quota,
			GiftQuota:  redemption.GiftQuota,
			TotalQuota: redemption.Quota + redemption.GiftQuota,
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrLotteryRedemptionAlreadyRedeemed) {
			return nil, err
		}
		if errors.Is(err, ErrLotteryRedemptionExhausted) {
			return nil, err
		}
		common.SysError("redemption failed: " + err.Error())
		return nil, ErrRedeemFailed
	}
	if err := InvalidateUserCache(userId); err != nil {
		common.SysLog("failed to invalidate user cache after redeeming code: " + err.Error())
	}
	logParts := make([]string, 0, 2)
	if redemption.Quota > 0 {
		logParts = append(logParts, fmt.Sprintf("通用余额 %s", logger.LogQuota(redemption.Quota)))
	}
	if redemption.GiftQuota > 0 {
		logParts = append(logParts, fmt.Sprintf("赠送余额 %s", logger.LogQuota(redemption.GiftQuota)))
	}
	RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码充值 %s，兑换码ID %d", strings.Join(logParts, "，"), redemption.Id))
	return result, nil
}

func (redemption *Redemption) Insert() error {
	redemption.NormalizeType()
	var err error
	err = DB.Create(redemption).Error
	return err
}

func (redemption *Redemption) SelectUpdate() error {
	// This can update zero values
	return DB.Model(redemption).Select("redeemed_time", "status").Updates(redemption).Error
}

// Update Make sure your token's fields is completed, because this will update non-zero values
func (redemption *Redemption) Update() error {
	redemption.NormalizeType()
	var err error
	err = DB.Model(redemption).Select("name", "status", "quota", "gift_quota", "redeemed_time", "expired_time", "type", "lottery_mode", "lottery_quota_min", "lottery_quota_max", "lottery_quota_choices", "lottery_balance_type", "max_redeem_count").Updates(redemption).Error
	return err
}

func (redemption *Redemption) Delete() error {
	var err error
	err = DB.Delete(redemption).Error
	return err
}

func DeleteRedemptionById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	err = DB.Where(redemption).First(&redemption).Error
	if err != nil {
		return err
	}
	return redemption.Delete()
}

func DeleteInvalidRedemptions() (int64, error) {
	now := common.GetTimestamp()
	result := DB.Where("status IN ? OR (status = ? AND expired_time != 0 AND expired_time < ?)", []int{common.RedemptionCodeStatusUsed, common.RedemptionCodeStatusDisabled}, common.RedemptionCodeStatusEnabled, now).Delete(&Redemption{})
	return result.RowsAffected, result.Error
}
