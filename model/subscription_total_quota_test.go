package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestCalculateSubscriptionTotalQuota_DailySevenDays(t *testing.T) {
	plan := &SubscriptionPlan{
		TotalAmount:      1200,
		DurationUnit:     SubscriptionDurationDay,
		DurationValue:    7,
		QuotaResetPeriod: SubscriptionResetDaily,
	}
	start := time.Date(2026, 4, 27, 6, 30, 0, 0, time.Local)

	require.Equal(t, int64(8400), CalculateSubscriptionTotalQuota(plan, start))
}

func TestCalculateSubscriptionTotalQuota_MonthlyOneMonth(t *testing.T) {
	plan := &SubscriptionPlan{
		TotalAmount:      5000,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		QuotaResetPeriod: SubscriptionResetMonthly,
	}
	start := time.Date(2026, 4, 27, 6, 30, 0, 0, time.Local)

	require.Equal(t, int64(5000), CalculateSubscriptionTotalQuota(plan, start))
}

func TestCompleteSubscriptionOrder_CreatesIndependentSubscriptionsForSamePlan(t *testing.T) {
	truncateTables(t)
	insertUserForPaymentGuardTest(t, 901, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 902)
	plan.TotalAmount = 1000
	require.NoError(t, DB.Save(plan).Error)
	insertSubscriptionOrderForPaymentGuardTest(t, "sub-independent-1", 901, plan.Id, PaymentProviderEpay)
	insertSubscriptionOrderForPaymentGuardTest(t, "sub-independent-2", 901, plan.Id, PaymentProviderEpay)

	require.NoError(t, CompleteSubscriptionOrder("sub-independent-1", "", PaymentProviderEpay, "alipay"))
	require.NoError(t, CompleteSubscriptionOrder("sub-independent-2", "", PaymentProviderEpay, "alipay"))

	var subs []UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 901, plan.Id).
		Order("id asc").
		Find(&subs).Error)
	require.Len(t, subs, 2)
	require.NotEqual(t, subs[0].Id, subs[1].Id)
	require.Equal(t, int64(1000), subs[0].AmountTotal)
	require.Equal(t, int64(1000), subs[1].AmountTotal)

	var paidOrderCount int64
	require.NoError(t, DB.Model(&SubscriptionOrder{}).
		Where("user_id = ? AND plan_id = ? AND status = ?", 901, plan.Id, common.TopUpStatusSuccess).
		Count(&paidOrderCount).Error)
	require.Equal(t, int64(2), paidOrderCount)
}
