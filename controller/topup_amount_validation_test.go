package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func TestValidateTopupAmount(t *testing.T) {
	originalOptions := append([]int(nil), operation_setting.GetPaymentSetting().AmountOptions...)
	t.Cleanup(func() {
		operation_setting.GetPaymentSetting().AmountOptions = originalOptions
	})

	t.Run("keeps min topup validation when no presets configured", func(t *testing.T) {
		operation_setting.GetPaymentSetting().AmountOptions = nil

		require.Equal(t, "充值数量不能小于 10", validateTopupAmount(9, 10))
		require.Empty(t, validateTopupAmount(10, 10))
		require.Empty(t, validateTopupAmount(88, 10))
	})

	t.Run("rejects stale preset amount when preset list changed", func(t *testing.T) {
		operation_setting.GetPaymentSetting().AmountOptions = []int{20, 50, 100}

		require.Empty(t, validateTopupAmount(50, 10))
		require.Equal(t, "当前充值方案已变更，请刷新页面后重试", validateTopupAmount(30, 10))
	})
}
