package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestBuildFeishuMessageUUIDDeterministic(t *testing.T) {
	tradeNo := "ALIPAYF2F_1_1776460482441323523"

	u1 := buildFeishuMessageUUID(tradeNo)
	u2 := buildFeishuMessageUUID(tradeNo)

	require.Equal(t, u1, u2)
	require.Len(t, u1, 36)
}

func TestBuildSubscriptionFeishuMessageUUIDDeterministic(t *testing.T) {
	tradeNo := "SUBALIPAYF2F_1_1776460482441323523"

	u1 := buildSubscriptionFeishuMessageUUID(tradeNo)
	u2 := buildSubscriptionFeishuMessageUUID(tradeNo)

	require.Equal(t, u1, u2)
	require.Len(t, u1, 36)
	require.NotEqual(t, buildFeishuMessageUUID(tradeNo), u1)
}

func TestTruncateFeishuErrorBody(t *testing.T) {
	msg := truncateFeishuErrorBody([]byte(`{"code":230099,"msg":"Bad Request","chat_id":"oc_xxx"}`))
	require.Contains(t, msg, "Bad Request")
	require.NotEmpty(t, msg)
}

func TestBuildSubscriptionFeishuCardContent(t *testing.T) {
	order := &model.SubscriptionOrder{
		UserId:        1,
		PlanId:        2,
		Money:         9.99,
		TradeNo:       "SUB_TEST_1",
		PaymentMethod: "alipay_f2f",
		CreateTime:    time.Date(2026, 4, 1, 10, 0, 0, 0, time.Local).Unix(),
		CompleteTime:  time.Date(2026, 4, 1, 10, 1, 0, 0, time.Local).Unix(),
	}
	plan := &model.SubscriptionPlan{
		Id:               2,
		Title:            "Pro",
		TotalAmount:      1000,
		DurationUnit:     model.SubscriptionDurationMonth,
		DurationValue:    1,
		QuotaResetPeriod: model.SubscriptionResetDaily,
	}
	user := &model.User{Id: 1, Username: "alice"}

	content, err := buildSubscriptionFeishuCardContent(order, plan, user, "127.0.0.1", "subscription_alipay_f2f")
	require.NoError(t, err)
	require.Contains(t, content, "订阅购买成功通知")
	require.Contains(t, content, "alice (#1)")
	require.Contains(t, content, "Pro")
	require.Contains(t, content, "周期额度")
	require.Contains(t, content, "计算总额度")
	require.Contains(t, content, "订阅支付宝回调")
}
