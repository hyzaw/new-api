package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompactModelFallbackWithMemoryCache(t *testing.T) {
	cleanupChannelAbilityTestData(t)

	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
		InitChannelCache()
	})

	channel := insertCompactFallbackTestChannel(t)
	InitChannelCache()

	compactModel := ratio_setting.WithCompactModelSuffix("gpt-5.5")

	selected, err := GetRandomSatisfiedChannel("codex_plus号池", compactModel, 0)
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, channel.Id, selected.Id)

	assert.True(t, HasEnabledChannelForGroupModel("codex_plus号池", compactModel))
	assert.True(t, IsChannelEnabledForGroupModel("codex_plus号池", compactModel, channel.Id))
}

func TestCompactModelFallbackWithoutMemoryCache(t *testing.T) {
	cleanupChannelAbilityTestData(t)

	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})

	initCol()
	channel := insertCompactFallbackTestChannel(t)
	compactModel := ratio_setting.WithCompactModelSuffix("gpt-5.5")

	selected, err := GetRandomSatisfiedChannel("codex_plus号池", compactModel, 0)
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, channel.Id, selected.Id)

	assert.True(t, HasEnabledChannelForGroupModel("codex_plus号池", compactModel))
	assert.True(t, IsChannelEnabledForGroupModel("codex_plus号池", compactModel, channel.Id))
}

func cleanupChannelAbilityTestData(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Exec("DELETE FROM abilities").Error)
	require.NoError(t, DB.Exec("DELETE FROM channels").Error)
}

func insertCompactFallbackTestChannel(t *testing.T) *Channel {
	t.Helper()

	weight := uint(100)
	priority := int64(10)
	channel := &Channel{
		Type:     1,
		Key:      "test-key",
		Status:   common.ChannelStatusEnabled,
		Name:     "compact-fallback-test",
		Models:   "gpt-5.5",
		Group:    "codex_plus号池",
		Weight:   &weight,
		Priority: &priority,
	}
	require.NoError(t, DB.Create(channel).Error)
	require.NoError(t, channel.UpdateAbilities(nil))
	return channel
}
