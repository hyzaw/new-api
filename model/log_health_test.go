package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRequestStatusMonitorSnapshot(t *testing.T) {
	truncateTables(t)

	windowEnd := int64(1_800)
	logs := []*Log{
		{CreatedAt: 10, Type: LogTypeConsume, Username: "u1", Group: "123team", ModelName: "gpt-4"},
		{CreatedAt: 100, Type: LogTypeError, Username: "u1", Group: "default", ModelName: "gpt-4"},
		{CreatedAt: 610, Type: LogTypeConsume, Username: "u1", Group: "vip", ModelName: "gpt-4"},
		{CreatedAt: 620, Type: LogTypeConsume, Username: "u1", Group: "default", ModelName: "claude-3"},
		{CreatedAt: 1_250, Type: LogTypeError, Username: "u1", Group: "vip", ModelName: "claude-3"},
		{CreatedAt: 1_260, Type: LogTypeConsume, Username: "u1", Group: "default", ModelName: ""},
	}
	for _, item := range logs {
		require.NoError(t, LOG_DB.Create(item).Error)
	}

	monitor, err := GetRequestStatusMonitorSnapshot(windowEnd, 3, 600)
	require.NoError(t, err)
	require.NotNil(t, monitor)
	require.Len(t, monitor.Points, 3)
	require.Len(t, monitor.Models, 5)

	assert.Equal(t, int64(0), monitor.WindowStart)
	assert.Equal(t, int64(1_800), monitor.WindowEnd)

	assert.Equal(t, int64(1), monitor.Points[0].SuccessCount)
	assert.Equal(t, int64(1), monitor.Points[0].ErrorCount)
	assert.Equal(t, "warning", monitor.Points[0].Status)

	assert.Equal(t, int64(2), monitor.Points[1].SuccessCount)
	assert.Equal(t, int64(0), monitor.Points[1].ErrorCount)
	assert.Equal(t, "healthy", monitor.Points[1].Status)

	assert.Equal(t, int64(1), monitor.Points[2].SuccessCount)
	assert.Equal(t, int64(1), monitor.Points[2].ErrorCount)
	assert.Equal(t, "warning", monitor.Points[2].Status)

	assert.Equal(t, int64(4), monitor.Summary.SuccessCount)
	assert.Equal(t, int64(2), monitor.Summary.ErrorCount)
	assert.Equal(t, int64(6), monitor.Summary.TotalCount)
	assert.Equal(t, 1, monitor.Summary.HealthyPoints)
	assert.Equal(t, 2, monitor.Summary.WarningPoints)
	assert.Equal(t, 0, monitor.Summary.ErrorPoints)

	modelsByDisplayName := make(map[string]*RequestStatusModelLine, len(monitor.Models))
	for _, item := range monitor.Models {
		modelsByDisplayName[item.DisplayName] = item
	}

	defaultGpt4 := modelsByDisplayName["default-gpt-4"]
	require.NotNil(t, defaultGpt4)
	assert.Equal(t, "default", defaultGpt4.GroupName)
	assert.Equal(t, "gpt-4", defaultGpt4.ModelName)
	assert.Equal(t, int64(2), defaultGpt4.Summary.TotalCount)
	assert.Equal(t, int64(1), defaultGpt4.Summary.SuccessCount)
	assert.Equal(t, int64(1), defaultGpt4.Summary.ErrorCount)

	vipGpt4 := modelsByDisplayName["vip-gpt-4"]
	require.NotNil(t, vipGpt4)
	assert.Equal(t, int64(1), vipGpt4.Summary.TotalCount)
	assert.Equal(t, int64(1), vipGpt4.Summary.SuccessCount)
	assert.Equal(t, int64(0), vipGpt4.Summary.ErrorCount)

	defaultClaude := modelsByDisplayName["default-claude-3"]
	require.NotNil(t, defaultClaude)
	assert.Equal(t, int64(1), defaultClaude.Summary.TotalCount)
	assert.Equal(t, int64(1), defaultClaude.Summary.SuccessCount)

	vipClaude := modelsByDisplayName["vip-claude-3"]
	require.NotNil(t, vipClaude)
	assert.Equal(t, int64(1), vipClaude.Summary.TotalCount)
	assert.Equal(t, int64(1), vipClaude.Summary.ErrorCount)

	defaultUnknown := modelsByDisplayName["default-"+unknownModelName]
	require.NotNil(t, defaultUnknown)
	assert.Equal(t, int64(1), defaultUnknown.Summary.TotalCount)
	assert.Equal(t, int64(1), defaultUnknown.Summary.SuccessCount)
}

func TestClassifyRequestStatus(t *testing.T) {
	assert.Equal(t, "no_data", classifyRequestStatus(0, 0))
	assert.Equal(t, "healthy", classifyRequestStatus(60, 10))
	assert.Equal(t, "warning", classifyRequestStatus(30, 10))
	assert.Equal(t, "error", classifyRequestStatus(29.9, 10))
}
