package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func TestValidateTopupAmount(t *testing.T) {
	originalOptions := append([]int(nil), operation_setting.GetPaymentSetting().AmountOptions...)
	originalDiscounts := make(map[int]float64, len(operation_setting.GetPaymentSetting().AmountDiscount))
	for amount, discount := range operation_setting.GetPaymentSetting().AmountDiscount {
		originalDiscounts[amount] = discount
	}
	t.Cleanup(func() {
		operation_setting.GetPaymentSetting().AmountOptions = originalOptions
		operation_setting.GetPaymentSetting().AmountDiscount = originalDiscounts
	})

	t.Run("keeps min topup validation when no discounts configured", func(t *testing.T) {
		operation_setting.GetPaymentSetting().AmountOptions = []int{20, 50, 100}
		operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{}

		require.Equal(t, "充值数量不能小于 10", validateTopupAmount(9, 10))
		require.Empty(t, validateTopupAmount(10, 10))
		require.Empty(t, validateTopupAmount(88, 10))
	})

	t.Run("keeps custom amounts enabled when no presets configured", func(t *testing.T) {
		operation_setting.GetPaymentSetting().AmountOptions = nil
		operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{}

		require.Equal(t, "充值数量不能小于 10", validateTopupAmount(9, 10))
		require.Empty(t, validateTopupAmount(10, 10))
		require.Empty(t, validateTopupAmount(88, 10))
	})

	t.Run("rejects stale discounted amount when preset list changed", func(t *testing.T) {
		operation_setting.GetPaymentSetting().AmountOptions = []int{20, 50, 100}
		operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{30: 0.8, 50: 0.9}

		require.Empty(t, validateTopupAmount(50, 10))
		require.Equal(t, "当前充值方案已变更，请刷新页面后重试", validateTopupAmount(30, 10))
	})
}

func TestGetTopupDiscountIgnoresStaleDiscounts(t *testing.T) {
	originalOptions := append([]int(nil), operation_setting.GetPaymentSetting().AmountOptions...)
	originalDiscounts := make(map[int]float64, len(operation_setting.GetPaymentSetting().AmountDiscount))
	for amount, discount := range operation_setting.GetPaymentSetting().AmountDiscount {
		originalDiscounts[amount] = discount
	}
	t.Cleanup(func() {
		operation_setting.GetPaymentSetting().AmountOptions = originalOptions
		operation_setting.GetPaymentSetting().AmountDiscount = originalDiscounts
	})

	operation_setting.GetPaymentSetting().AmountOptions = []int{20, 50, 100}
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{
		30: 0.8,
		50: 0.9,
	}

	require.Equal(t, 1.0, getTopupDiscount(30))
	require.Equal(t, 0.9, getTopupDiscount(50))
	require.Equal(t, 1.0, getTopupDiscount(88))
}
