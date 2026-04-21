package operation_setting

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseGroupDelayRules(t *testing.T) {
	rules, err := ParseGroupDelayRules(`[{"group":"default","min_seconds":1,"max_seconds":3}]`)
	require.NoError(t, err)
	require.Equal(t, []GroupDelayRule{
		{
			Group:      "default",
			MinSeconds: 1,
			MaxSeconds: 3,
		},
	}, rules)
}

func TestParseGroupDelayRules_Invalid(t *testing.T) {
	_, err := ParseGroupDelayRules(`[{"group":"","min_seconds":1,"max_seconds":3}]`)
	require.Error(t, err)

	_, err = ParseGroupDelayRules(`[{"group":"default","min_seconds":5,"max_seconds":3}]`)
	require.Error(t, err)

	_, err = ParseGroupDelayRules(`[{"group":"default","min_seconds":1,"max_seconds":3},{"group":"default","min_seconds":2,"max_seconds":4}]`)
	require.Error(t, err)
}

func TestGetGroupDelayDuration(t *testing.T) {
	original := groupDelaySetting
	t.Cleanup(func() {
		groupDelaySetting = original
	})

	groupDelaySetting.Rules = []GroupDelayRule{
		{
			Group:      "default",
			MinSeconds: 2,
			MaxSeconds: 2,
		},
		{
			Group:      "vip",
			MinSeconds: 1,
			MaxSeconds: 3,
		},
	}

	require.Equal(t, 2*time.Second, GetGroupDelayDuration("default"))

	for i := 0; i < 20; i++ {
		delay := GetGroupDelayDuration("vip")
		require.GreaterOrEqual(t, delay, 1*time.Second)
		require.LessOrEqual(t, delay, 3*time.Second)
	}

	require.Zero(t, GetGroupDelayDuration("missing"))
}
