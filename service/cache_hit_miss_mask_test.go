package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func TestApplyCacheHitMissMaskByUsageReducesCachedTokens(t *testing.T) {
	t.Cleanup(func() {
		_ = operation_setting.UpdateCacheHitMissMaskRulesByJSONString("[]")
	})

	require.NoError(t, operation_setting.UpdateCacheHitMissMaskRulesByJSONString(
		`[{"group":"default","min_percent":10,"max_percent":10}]`,
	))

	usage := &dto.Usage{
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 100,
		},
		PromptCacheHitTokens: 100,
		InputTokensDetails: &dto.InputTokenDetails{
			CachedTokens: 100,
		},
	}
	info := &relaycommon.RelayInfo{
		UserGroup: "default",
	}

	modified := ApplyCacheHitMissMaskByUsage(info, usage)
	require.True(t, modified)
	require.Equal(t, 90, usage.PromptTokensDetails.CachedTokens)
	require.Equal(t, 90, usage.PromptCacheHitTokens)
	require.NotNil(t, usage.InputTokensDetails)
	require.Equal(t, 90, usage.InputTokensDetails.CachedTokens)
}
