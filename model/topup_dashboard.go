package model

import (
	"time"

	"github.com/QuantumNous/new-api/common"
)

type TopUpDashboardOverview struct {
	TotalOrders        int64   `json:"total_orders"`
	SuccessOrders      int64   `json:"success_orders"`
	PendingOrders      int64   `json:"pending_orders"`
	TotalMoney         float64 `json:"total_money"`
	SuccessMoney       float64 `json:"success_money"`
	RefundedMoney      float64 `json:"refunded_money"`
	PendingRefundMoney float64 `json:"pending_refund_money"`
	RefundCount        int64   `json:"refund_count"`
	NetMoney           float64 `json:"net_money"`
}

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

type TopUpDashboardStats struct {
	Days           int                               `json:"days"`
	Overview       TopUpDashboardOverview            `json:"overview"`
	DailyTrend     []*TopUpDashboardTrendItem        `json:"daily_trend"`
	PaymentMethods []*TopUpDashboardDistributionItem `json:"payment_methods"`
	OrderStatuses  []*TopUpDashboardDistributionItem `json:"order_statuses"`
	RefundStatuses []*TopUpDashboardDistributionItem `json:"refund_statuses"`
}

func getDayBucket(ts int64) string {
	return time.Unix(ts, 0).In(time.Local).Format("2006-01-02")
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
	stats.Overview.TotalMoney = decimalFromFloatMoney(stats.Overview.TotalMoney).InexactFloat64()
	stats.Overview.SuccessMoney = decimalFromFloatMoney(stats.Overview.SuccessMoney).InexactFloat64()
	stats.Overview.RefundedMoney = decimalFromFloatMoney(stats.Overview.RefundedMoney).InexactFloat64()
	stats.Overview.PendingRefundMoney = decimalFromFloatMoney(stats.Overview.PendingRefundMoney).InexactFloat64()
	stats.Overview.NetMoney = decimalFromFloatMoney(stats.Overview.SuccessMoney).
		Sub(decimalFromFloatMoney(stats.Overview.RefundedMoney)).
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

	return stats, nil
}
