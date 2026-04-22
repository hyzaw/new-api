package operation_setting

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/require"
)

func TestParseLargePromptRPMRules(t *testing.T) {
	rules, err := ParseLargePromptRPMRules(`[{"group":"default","threshold_k":32,"temporary_rpm":5}]`)
	require.NoError(t, err)
	require.Equal(t, []LargePromptRPMRule{
		{
			Group:        "default",
			ThresholdK:   32,
			TemporaryRPM: 5,
		},
	}, rules)
}

func TestParseLargePromptRPMRulesInvalid(t *testing.T) {
	_, err := ParseLargePromptRPMRules(`[{"group":"","threshold_k":32,"temporary_rpm":5}]`)
	require.Error(t, err)

	_, err = ParseLargePromptRPMRules(`[{"group":"default","threshold_k":0,"temporary_rpm":5}]`)
	require.Error(t, err)

	_, err = ParseLargePromptRPMRules(`[{"group":"default","threshold_k":32,"temporary_rpm":0}]`)
	require.Error(t, err)

	_, err = ParseLargePromptRPMRules(`[{"group":"default","threshold_k":32,"temporary_rpm":5},{"group":"default","threshold_k":64,"temporary_rpm":3}]`)
	require.Error(t, err)
}

func TestGetLargePromptRPMRule(t *testing.T) {
	original := largePromptRPMSetting
	t.Cleanup(func() {
		largePromptRPMSetting = original
	})

	largePromptRPMSetting.Rules = []LargePromptRPMRule{
		{
			Group:        "default",
			ThresholdK:   32,
			TemporaryRPM: 5,
		},
	}

	rule, ok := GetLargePromptRPMRule("default", 32001)
	require.True(t, ok)
	require.Equal(t, 5, rule.TemporaryRPM)

	_, ok = GetLargePromptRPMRule("default", 32000)
	require.False(t, ok)
}

func TestTemporaryLargePromptRPMMemoryStore(t *testing.T) {
	originalDuration := setting.ModelRequestRateLimitDurationMinutes
	originalRedisEnabled := common.RedisEnabled
	t.Cleanup(func() {
		setting.ModelRequestRateLimitDurationMinutes = originalDuration
		common.RedisEnabled = originalRedisEnabled
		ResetTemporaryLargePromptRPMStore()
	})

	setting.ModelRequestRateLimitDurationMinutes = 1
	common.RedisEnabled = false
	ResetTemporaryLargePromptRPMStore()

	SetTemporaryLargePromptRPM(1, "default", 7)
	rpm, ok := GetTemporaryLargePromptRPM(1, "default")
	require.True(t, ok)
	require.Equal(t, 7, rpm)
}

func TestTemporaryLargePromptRPMExpires(t *testing.T) {
	originalDuration := setting.ModelRequestRateLimitDurationMinutes
	originalRedisEnabled := common.RedisEnabled
	t.Cleanup(func() {
		setting.ModelRequestRateLimitDurationMinutes = originalDuration
		common.RedisEnabled = originalRedisEnabled
		ResetTemporaryLargePromptRPMStore()
	})

	setting.ModelRequestRateLimitDurationMinutes = 1
	common.RedisEnabled = false
	ResetTemporaryLargePromptRPMStore()

	key := largePromptRPMTempKey(1, "default")
	temporaryLargePromptRPMStore.Store(key, temporaryLargePromptRPMEntry{
		RPM:       5,
		ExpiresAt: time.Now().Add(-time.Second),
	})

	_, ok := GetTemporaryLargePromptRPM(1, "default")
	require.False(t, ok)
}
