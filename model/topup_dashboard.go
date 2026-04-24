package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type TopUpDashboardOverview struct {
	TotalOrders         int64   `json:"total_orders"`
	SuccessOrders       int64   `json:"success_orders"`
	PendingOrders       int64   `json:"pending_orders"`
	PaidUserCount       int64   `json:"paid_user_count"`
	TotalUserCount      int64   `json:"total_user_count"`
	TotalMoney          float64 `json:"total_money"`
	SuccessMoney        float64 `json:"success_money"`
	RefundedMoney       float64 `json:"refunded_money"`
	PendingRefundMoney  float64 `json:"pending_refund_money"`
	RefundCount         int64   `json:"refund_count"`
	NetMoney            float64 `json:"net_money"`
	TotalUserQuota      int64   `json:"total_user_quota"`
	TotalUserGiftQuota  int64   `json:"total_user_gift_quota"`
	TotalConsumedQuota  int64   `json:"total_consumed_quota"`
	TotalConsumedTokens int64   `json:"total_consumed_tokens"`
}

type topUpDashboardConsumeTotals struct {
	Quota  int64 `gorm:"column:quota"`
	Tokens int64 `gorm:"column:tokens"`
}

const (
	topUpDashboardConsumeCachePrefix = "new-api:topup_dashboard:consume:v1"
	topUpDashboardTodayCacheTTL      = 10 * time.Minute
	topUpDashboardHistoryCacheTTL    = 365 * 24 * time.Hour
)

type TopUpDashboardTrendItem struct {
	Date          string  `json:"date"`
	TotalMoney    float64 `json:"total_money"`
	SuccessMoney  float64 `json:"success_money"`
	RefundedMoney float64 `json:"refunded_money"`
	OrderCount    int     `json:"order_count"`
	SuccessCount  int     `json:"success_count"`
	RefundCount   int     `json:"refund_count"`
}

type TopUpDashboardDistributionItem struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

type TopUpDashboardValuableUserItem struct {
	UserId        int     `json:"user_id"`
	Username      string  `json:"username"`
	DisplayName   string  `json:"display_name"`
	TotalMoney    float64 `json:"total_money"`
	SuccessOrders int64   `json:"success_orders"`
	LastTopUpTime int64   `json:"last_topup_time"`
}

type TopUpDashboardStats struct {
	Days           int                               `json:"days"`
	Overview       TopUpDashboardOverview            `json:"overview"`
	DailyTrend     []*TopUpDashboardTrendItem        `json:"daily_trend"`
	PaymentMethods []*TopUpDashboardDistributionItem `json:"payment_methods"`
	OrderStatuses  []*TopUpDashboardDistributionItem `json:"order_statuses"`
	RefundStatuses []*TopUpDashboardDistributionItem `json:"refund_statuses"`
	ValuableUsers  []*TopUpDashboardValuableUserItem `json:"valuable_users"`
}

func getDayBucket(ts int64) string {
	return time.Unix(ts, 0).In(time.Local).Format("2006-01-02")
}

func getDayStart(ts int64) time.Time {
	t := time.Unix(ts, 0).In(time.Local)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func topUpDashboardConsumeCacheKey(day string) string {
	return fmt.Sprintf("%s:%s", topUpDashboardConsumeCachePrefix, day)
}

func encodeTopUpDashboardConsumeTotals(totals topUpDashboardConsumeTotals) string {
	return fmt.Sprintf("%d:%d", totals.Quota, totals.Tokens)
}

func decodeTopUpDashboardConsumeTotals(raw string) (topUpDashboardConsumeTotals, bool) {
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return topUpDashboardConsumeTotals{}, false
	}
	quota, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return topUpDashboardConsumeTotals{}, false
	}
	tokens, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return topUpDashboardConsumeTotals{}, false
	}
	return topUpDashboardConsumeTotals{Quota: quota, Tokens: tokens}, true
}

func getTopUpDashboardConsumeTotalsForDay(dayStart time.Time, nowDayStart time.Time) (topUpDashboardConsumeTotals, error) {
	day := dayStart.Format("2006-01-02")
	cacheKey := topUpDashboardConsumeCacheKey(day)
	if common.RedisEnabled && common.RDB != nil {
		raw, err := common.RedisGet(cacheKey)
		if err == nil {
			if totals, ok := decodeTopUpDashboardConsumeTotals(raw); ok {
				return totals, nil
			}
		} else if !errors.Is(err, redis.Nil) {
			common.SysLog(fmt.Sprintf("failed to get topup dashboard consume cache %s: %s", cacheKey, err.Error()))
		}
	}

	var totals topUpDashboardConsumeTotals
	start := dayStart.Unix()
	end := dayStart.AddDate(0, 0, 1).Unix()
	err := LOG_DB.Model(&Log{}).
		Select("COALESCE(SUM(quota), 0) AS quota, COALESCE(SUM(prompt_tokens + completion_tokens), 0) AS tokens").
		Where("type = ? AND created_at >= ? AND created_at < ?", LogTypeConsume, start, end).
		Scan(&totals).Error
	if err != nil {
		return totals, err
	}

	if common.RedisEnabled && common.RDB != nil {
		ttl := topUpDashboardHistoryCacheTTL
		if dayStart.Equal(nowDayStart) {
			ttl = topUpDashboardTodayCacheTTL
		}
		if err := common.RedisSet(cacheKey, encodeTopUpDashboardConsumeTotals(totals), ttl); err != nil {
			common.SysLog(fmt.Sprintf("failed to set topup dashboard consume cache %s: %s", cacheKey, err.Error()))
		}
	}
	return totals, nil
}

func getTopUpDashboardConsumeTotals() (topUpDashboardConsumeTotals, error) {
	var firstLog Log
	err := LOG_DB.Model(&Log{}).
		Select("created_at").
		Where("type = ?", LogTypeConsume).
		Order("created_at ASC").
		Limit(1).
		First(&firstLog).Error
	if err != nil {
		// GORM returns ErrRecordNotFound when there are no consume logs.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return topUpDashboardConsumeTotals{}, nil
		}
		return topUpDashboardConsumeTotals{}, err
	}

	nowDayStart := getDayStart(common.GetTimestamp())
	dayStart := getDayStart(firstLog.CreatedAt)
	var totals topUpDashboardConsumeTotals
	for !dayStart.After(nowDayStart) {
		dayTotals, err := getTopUpDashboardConsumeTotalsForDay(dayStart, nowDayStart)
		if err != nil {
			return totals, err
		}
		totals.Quota += dayTotals.Quota
		totals.Tokens += dayTotals.Tokens
		dayStart = dayStart.AddDate(0, 0, 1)
	}
	return totals, nil
}

func GetAdminTopUpDashboardStats(days int) (*TopUpDashboardStats, error) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	stats := &TopUpDashboardStats{
		Days:           days,
		DailyTrend:     make([]*TopUpDashboardTrendItem, 0, days),
		PaymentMethods: make([]*TopUpDashboardDistributionItem, 0),
		OrderStatuses:  make([]*TopUpDashboardDistributionItem, 0),
		RefundStatuses: make([]*TopUpDashboardDistributionItem, 0),
		ValuableUsers:  make([]*TopUpDashboardValuableUserItem, 0),
	}

	if err := DB.Model(&TopUp{}).Count(&stats.Overview.TotalOrders).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&TopUp{}).Where("status = ?", common.TopUpStatusSuccess).Count(&stats.Overview.SuccessOrders).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&TopUp{}).Where("status = ?", common.TopUpStatusPending).Count(&stats.Overview.PendingOrders).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&TopUp{}).Where("status = ?", common.TopUpStatusSuccess).Distinct("user_id").Count(&stats.Overview.PaidUserCount).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&User{}).Where("status = ? AND role = ?", common.UserStatusEnabled, common.RoleCommonUser).Count(&stats.Overview.TotalUserCount).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&TopUp{}).Select("COALESCE(SUM(money), 0)").Scan(&stats.Overview.TotalMoney).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&TopUp{}).Where("status = ?", common.TopUpStatusSuccess).Select("COALESCE(SUM(money), 0)").Scan(&stats.Overview.SuccessMoney).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&TopUpRefund{}).Where("status = ?", TopUpRefundStatusSuccess).Count(&stats.Overview.RefundCount).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&TopUpRefund{}).Where("status = ?", TopUpRefundStatusSuccess).Select("COALESCE(SUM(refund_amount), 0)").Scan(&stats.Overview.RefundedMoney).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&TopUpRefund{}).Where("status = ?", TopUpRefundStatusPending).Select("COALESCE(SUM(refund_amount), 0)").Scan(&stats.Overview.PendingRefundMoney).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&User{}).Select("COALESCE(SUM(quota), 0)").Scan(&stats.Overview.TotalUserQuota).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&User{}).Select("COALESCE(SUM(gift_quota), 0)").Scan(&stats.Overview.TotalUserGiftQuota).Error; err != nil {
		return nil, err
	}
	consumeTotals, err := getTopUpDashboardConsumeTotals()
	if err != nil {
		return nil, err
	}
	stats.Overview.TotalConsumedQuota = consumeTotals.Quota
	stats.Overview.TotalConsumedTokens = consumeTotals.Tokens
	stats.Overview.TotalMoney = decimalFromFloatMoney(stats.Overview.TotalMoney).InexactFloat64()
	stats.Overview.SuccessMoney = decimalFromFloatMoney(stats.Overview.SuccessMoney).InexactFloat64()
	stats.Overview.RefundedMoney = decimalFromFloatMoney(stats.Overview.RefundedMoney).InexactFloat64()
	stats.Overview.PendingRefundMoney = decimalFromFloatMoney(stats.Overview.PendingRefundMoney).InexactFloat64()
	stats.Overview.NetMoney = decimalFromFloatMoney(stats.Overview.SuccessMoney).
		Sub(decimalFromFloatMoney(stats.Overview.RefundedMoney)).
		Sub(decimalFromFloatMoney(stats.Overview.PendingRefundMoney)).
		InexactFloat64()

	cutoff := common.GetTimestamp() - int64(days-1)*24*60*60

	var recentTopUps []*TopUp
	if err := DB.Where("create_time >= ?", cutoff).Order("create_time asc").Find(&recentTopUps).Error; err != nil {
		return nil, err
	}

	var recentRefunds []*TopUpRefund
	if err := DB.Where("create_time >= ?", cutoff).Order("create_time asc").Find(&recentRefunds).Error; err != nil {
		return nil, err
	}

	trendMap := make(map[string]*TopUpDashboardTrendItem, days)
	now := time.Now().In(time.Local)
	for i := days - 1; i >= 0; i-- {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		item := &TopUpDashboardTrendItem{Date: day}
		trendMap[day] = item
		stats.DailyTrend = append(stats.DailyTrend, item)
	}

	paymentMethodMap := make(map[string]*TopUpDashboardDistributionItem)
	orderStatusMap := make(map[string]*TopUpDashboardDistributionItem)
	refundStatusMap := make(map[string]*TopUpDashboardDistributionItem)

	for _, topUp := range recentTopUps {
		if topUp == nil {
			continue
		}

		bucket := getDayBucket(topUp.CreateTime)
		if trend, ok := trendMap[bucket]; ok {
			trend.OrderCount++
			trend.TotalMoney = decimalFromFloatMoney(trend.TotalMoney).
				Add(decimalFromFloatMoney(topUp.Money)).
				InexactFloat64()
			if topUp.Status == common.TopUpStatusSuccess {
				trend.SuccessCount++
				trend.SuccessMoney = decimalFromFloatMoney(trend.SuccessMoney).
					Add(decimalFromFloatMoney(topUp.Money)).
					InexactFloat64()
			}
		}

		statusName := topUp.Status
		statusItem, ok := orderStatusMap[statusName]
		if !ok {
			statusItem = &TopUpDashboardDistributionItem{Name: statusName}
			orderStatusMap[statusName] = statusItem
		}
		statusItem.Count++
		statusItem.Amount = decimalFromFloatMoney(statusItem.Amount).
			Add(decimalFromFloatMoney(topUp.Money)).
			InexactFloat64()

		if topUp.Status == common.TopUpStatusSuccess {
			methodName := topUp.PaymentMethod
			methodItem, ok := paymentMethodMap[methodName]
			if !ok {
				methodItem = &TopUpDashboardDistributionItem{Name: methodName}
				paymentMethodMap[methodName] = methodItem
			}
			methodItem.Count++
			methodItem.Amount = decimalFromFloatMoney(methodItem.Amount).
				Add(decimalFromFloatMoney(topUp.Money)).
				InexactFloat64()
		}
	}

	for _, refund := range recentRefunds {
		if refund == nil {
			continue
		}

		statusItem, ok := refundStatusMap[refund.Status]
		if !ok {
			statusItem = &TopUpDashboardDistributionItem{Name: refund.Status}
			refundStatusMap[refund.Status] = statusItem
		}
		statusItem.Count++
		statusItem.Amount = decimalFromFloatMoney(statusItem.Amount).
			Add(decimalFromFloatMoney(refund.RefundAmount)).
			InexactFloat64()

		if refund.Status == TopUpRefundStatusSuccess {
			ts := refund.CompleteTime
			if ts <= 0 {
				ts = refund.CreateTime
			}
			bucket := getDayBucket(ts)
			if trend, ok := trendMap[bucket]; ok {
				trend.RefundCount++
				trend.RefundedMoney = decimalFromFloatMoney(trend.RefundedMoney).
					Add(decimalFromFloatMoney(refund.RefundAmount)).
					InexactFloat64()
			}
		}
	}

	for _, item := range paymentMethodMap {
		stats.PaymentMethods = append(stats.PaymentMethods, item)
	}
	for _, item := range orderStatusMap {
		stats.OrderStatuses = append(stats.OrderStatuses, item)
	}
	for _, item := range refundStatusMap {
		stats.RefundStatuses = append(stats.RefundStatuses, item)
	}

	valuableUsers, err := getTopUpDashboardValuableUsers(10)
	if err != nil {
		return nil, err
	}
	stats.ValuableUsers = valuableUsers

	return stats, nil
}

type topUpDashboardValuableUserRow struct {
	UserId        int     `gorm:"column:user_id"`
	TotalMoney    float64 `gorm:"column:total_money"`
	SuccessOrders int64   `gorm:"column:success_orders"`
	LastTopUpTime int64   `gorm:"column:last_topup_time"`
}

func getTopUpDashboardValuableUsers(limit int) ([]*TopUpDashboardValuableUserItem, error) {
	if limit <= 0 {
		limit = 10
	}

	var rows []*topUpDashboardValuableUserRow
	if err := DB.Model(&TopUp{}).
		Select("user_id, COALESCE(SUM(money), 0) AS total_money, COUNT(*) AS success_orders, COALESCE(MAX(complete_time), 0) AS last_topup_time").
		Where("status = ?", common.TopUpStatusSuccess).
		Group("user_id").
		Order("total_money DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return make([]*TopUpDashboardValuableUserItem, 0), nil
	}

	userIds := make([]int, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.UserId <= 0 {
			continue
		}
		userIds = append(userIds, row.UserId)
	}

	var users []*User
	if err := DB.Select("id", "username", "display_name").Where("id IN ?", userIds).Find(&users).Error; err != nil {
		return nil, err
	}
	userMap := make(map[int]*User, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		userMap[user.Id] = user
	}

	items := make([]*TopUpDashboardValuableUserItem, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.UserId <= 0 {
			continue
		}
		item := &TopUpDashboardValuableUserItem{
			UserId:        row.UserId,
			TotalMoney:    decimalFromFloatMoney(row.TotalMoney).InexactFloat64(),
			SuccessOrders: row.SuccessOrders,
			LastTopUpTime: row.LastTopUpTime,
		}
		if user, ok := userMap[row.UserId]; ok && user != nil {
			item.Username = user.Username
			item.DisplayName = user.DisplayName
		}
		items = append(items, item)
	}
	return items, nil
}
