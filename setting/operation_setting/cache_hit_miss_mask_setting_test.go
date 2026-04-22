package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCacheHitMissMaskRules(t *testing.T) {
	rules, err := ParseCacheHitMissMaskRules(`[{"group":"default","min_percent":10,"max_percent":20}]`)
	require.NoError(t, err)
	require.Equal(t, []CacheHitMissMaskRule{
		{
			Group:      "default",
			MinPercent: 10,
			MaxPercent: 20,
		},
	}, rules)
}

func TestParseCacheHitMissMaskRulesInvalid(t *testing.T) {
	_, err := ParseCacheHitMissMaskRules(`[{"group":"","min_percent":10,"max_percent":20}]`)
	require.Error(t, err)

	_, err = ParseCacheHitMissMaskRules(`[{"group":"default","min_percent":0,"max_percent":20}]`)
	require.Error(t, err)

	_, err = ParseCacheHitMissMaskRules(`[{"group":"default","min_percent":30,"max_percent":20}]`)
	require.Error(t, err)

	_, err = ParseCacheHitMissMaskRules(`[{"group":"default","min_percent":10,"max_percent":120}]`)
	require.Error(t, err)
}

func TestRandomCacheHitMissMaskPercent(t *testing.T) {
	rule := CacheHitMissMaskRule{
		Group:      "default",
		MinPercent: 10,
		MaxPercent: 20,
	}

	for i := 0; i < 20; i++ {
		percent := RandomCacheHitMissMaskPercent(rule)
		require.GreaterOrEqual(t, percent, 10)
		require.LessOrEqual(t, percent, 20)
	}
}
