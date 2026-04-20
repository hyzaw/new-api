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
		{CreatedAt: 10, Type: LogTypeConsume, Username: "u1", ModelName: "gpt-4"},
		{CreatedAt: 100, Type: LogTypeError, Username: "u1", ModelName: "gpt-4"},
		{CreatedAt: 610, Type: LogTypeConsume, Username: "u1", ModelName: "gpt-4"},
		{CreatedAt: 620, Type: LogTypeConsume, Username: "u1", ModelName: "claude-3"},
		{CreatedAt: 1_250, Type: LogTypeError, Username: "u1", ModelName: "claude-3"},
		{CreatedAt: 1_260, Type: LogTypeConsume, Username: "u1", ModelName: ""},
	}
	for _, item := range logs {
		require.NoError(t, LOG_DB.Create(item).Error)
	}

	monitor, err := GetRequestStatusMonitorSnapshot(windowEnd, 3, 600)
	require.NoError(t, err)
	require.NotNil(t, monitor)
	require.Len(t, monitor.Points, 3)
	require.Len(t, monitor.Models, 3)

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

	firstModel := monitor.Models[0]
	assert.Equal(t, "gpt-4", firstModel.ModelName)
	assert.Equal(t, int64(3), firstModel.Summary.TotalCount)
	assert.Equal(t, int64(2), firstModel.Summary.SuccessCount)
	assert.Equal(t, int64(1), firstModel.Summary.ErrorCount)

	secondModel := monitor.Models[1]
	assert.Equal(t, "claude-3", secondModel.ModelName)
	assert.Equal(t, int64(2), secondModel.Summary.TotalCount)

	thirdModel := monitor.Models[2]
	assert.Equal(t, unknownModelName, thirdModel.ModelName)
	assert.Equal(t, int64(1), thirdModel.Summary.TotalCount)
	assert.Equal(t, int64(1), thirdModel.Summary.SuccessCount)
}

func TestClassifyRequestStatus(t *testing.T) {
	assert.Equal(t, "no_data", classifyRequestStatus(0, 0))
	assert.Equal(t, "healthy", classifyRequestStatus(60, 10))
	assert.Equal(t, "warning", classifyRequestStatus(30, 10))
	assert.Equal(t, "error", classifyRequestStatus(29.9, 10))
}
