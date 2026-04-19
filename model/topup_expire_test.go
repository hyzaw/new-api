package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpirePendingTopUps(t *testing.T) {
	truncateTables(t)

	now := common.GetTimestamp()
	expiredCandidate := &TopUp{
		UserId:        1,
		Amount:        10,
		Money:         10,
		TradeNo:       "topup_expire_candidate",
		PaymentMethod: "alipay_f2f",
		CreateTime:    now - 3*60*60,
		Status:        common.TopUpStatusPending,
	}
	freshPending := &TopUp{
		UserId:        1,
		Amount:        10,
		Money:         10,
		TradeNo:       "topup_fresh_pending",
		PaymentMethod: "alipay_f2f",
		CreateTime:    now - 30*60,
		Status:        common.TopUpStatusPending,
	}
	successOrder := &TopUp{
		UserId:        1,
		Amount:        10,
		Money:         10,
		TradeNo:       "topup_success_order",
		PaymentMethod: "alipay_f2f",
		CreateTime:    now - 4*60*60,
		CompleteTime:  now - 4*60*60 + 60,
		Status:        common.TopUpStatusSuccess,
	}

	require.NoError(t, DB.Create(expiredCandidate).Error)
	require.NoError(t, DB.Create(freshPending).Error)
	require.NoError(t, DB.Create(successOrder).Error)

	count, err := ExpirePendingTopUps(10, 2*60*60)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	reloadedExpired := GetTopUpByTradeNo(expiredCandidate.TradeNo)
	require.NotNil(t, reloadedExpired)
	assert.Equal(t, common.TopUpStatusExpired, reloadedExpired.Status)
	assert.NotZero(t, reloadedExpired.CompleteTime)

	reloadedFresh := GetTopUpByTradeNo(freshPending.TradeNo)
	require.NotNil(t, reloadedFresh)
	assert.Equal(t, common.TopUpStatusPending, reloadedFresh.Status)
	assert.Zero(t, reloadedFresh.CompleteTime)

	reloadedSuccess := GetTopUpByTradeNo(successOrder.TradeNo)
	require.NotNil(t, reloadedSuccess)
	assert.Equal(t, common.TopUpStatusSuccess, reloadedSuccess.Status)
}

func TestExpireTopUpOrder(t *testing.T) {
	truncateTables(t)

	topUp := &TopUp{
		UserId:        1,
		Amount:        10,
		Money:         10,
		TradeNo:       "topup_expire_by_trade_no",
		PaymentMethod: "stripe",
		CreateTime:    common.GetTimestamp() - 4*60*60,
		Status:        common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(topUp).Error)

	require.NoError(t, ExpireTopUpOrder(topUp.TradeNo))

	reloaded := GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, reloaded)
	assert.Equal(t, common.TopUpStatusExpired, reloaded.Status)
	assert.NotZero(t, reloaded.CompleteTime)
}
