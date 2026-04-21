package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	RequestStatusIntervalSeconds = int64(10 * 60)
	RequestStatusPointCount      = 144
	defaultGroupName             = "default"
	redirectGroupName123Team     = "123team"
	unknownModelName             = "(unknown)"
)

type RequestStatusPoint struct {
	StartTime    int64   `json:"start_time"`
	EndTime      int64   `json:"end_time"`
	SuccessCount int64   `json:"success_count"`
	ErrorCount   int64   `json:"error_count"`
	TotalCount   int64   `json:"total_count"`
	SuccessRate  float64 `json:"success_rate"`
	Status       string  `json:"status"`
}

type RequestStatusSummary struct {
	SuccessCount  int64   `json:"success_count"`
	ErrorCount    int64   `json:"error_count"`
	TotalCount    int64   `json:"total_count"`
	SuccessRate   float64 `json:"success_rate"`
	HealthyPoints int     `json:"healthy_points"`
	WarningPoints int     `json:"warning_points"`
	ErrorPoints   int     `json:"error_points"`
	NoDataPoints  int     `json:"no_data_points"`
}

type RequestStatusModelLine struct {
	GroupName   string                `json:"group_name"`
	ModelName   string                `json:"model_name"`
	DisplayName string                `json:"display_name"`
	Summary     RequestStatusSummary  `json:"summary"`
	Points      []*RequestStatusPoint `json:"points"`
}

type RequestStatusMonitor struct {
	GeneratedAt     int64                     `json:"generated_at"`
	WindowStart     int64                     `json:"window_start"`
	WindowEnd       int64                     `json:"window_end"`
	IntervalSeconds int64                     `json:"interval_seconds"`
	PointCount      int                       `json:"point_count"`
	Summary         RequestStatusSummary      `json:"summary"`
	Points          []*RequestStatusPoint     `json:"points"`
	Models          []*RequestStatusModelLine `json:"models"`
}

type requestStatusLogRow struct {
	CreatedAt int64  `gorm:"column:created_at"`
	Type      int    `gorm:"column:type"`
	GroupName string `gorm:"column:group_name"`
	ModelName string `gorm:"column:model_name"`
}

func classifyRequestStatus(successRate float64, totalCount int64) string {
	if totalCount == 0 {
		return "no_data"
	}
	if successRate >= 60 {
		return "healthy"
	}
	if successRate >= 30 {
		return "warning"
	}
	return "error"
}

func cloneRequestStatusPoints(windowStart int64, pointCount int, intervalSeconds int64) []*RequestStatusPoint {
	points := make([]*RequestStatusPoint, 0, pointCount)
	for i := 0; i < pointCount; i++ {
		start := windowStart + int64(i)*intervalSeconds
		points = append(points, &RequestStatusPoint{
			StartTime: start,
			EndTime:   start + intervalSeconds,
		})
	}
	return points
}

func normalizeRequestStatusModelName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return unknownModelName
	}
	return name
}

func normalizeRequestStatusGroupName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return defaultGroupName
	}
	if strings.EqualFold(name, redirectGroupName123Team) {
		return defaultGroupName
	}
	return name
}

func buildRequestStatusDisplayName(groupName string, modelName string) string {
	return fmt.Sprintf("%s-%s", groupName, modelName)
}

func buildRequestStatusModelKey(groupName string, modelName string) string {
	return groupName + "\x00" + modelName
}

func getRequestStatusLogGroupColumn() string {
	if logGroupCol != "" {
		return logGroupCol
	}
	if common.LogSqlType == common.DatabaseTypePostgreSQL || (common.LogSqlType == "" && common.UsingPostgreSQL) {
		return `"group"`
	}
	return "`group`"
}

func finalizeRequestStatusSummary(points []*RequestStatusPoint) RequestStatusSummary {
	summary := RequestStatusSummary{}
	for _, point := range points {
		if point == nil {
			continue
		}
		point.TotalCount = point.SuccessCount + point.ErrorCount
		if point.TotalCount > 0 {
			point.SuccessRate = float64(point.SuccessCount) * 100 / float64(point.TotalCount)
		}
		point.Status = classifyRequestStatus(point.SuccessRate, point.TotalCount)

		summary.SuccessCount += point.SuccessCount
		summary.ErrorCount += point.ErrorCount
		switch point.Status {
		case "healthy":
			summary.HealthyPoints++
		case "warning":
			summary.WarningPoints++
		case "error":
			summary.ErrorPoints++
		default:
			summary.NoDataPoints++
		}
	}
	summary.TotalCount = summary.SuccessCount + summary.ErrorCount
	if summary.TotalCount > 0 {
		summary.SuccessRate = float64(summary.SuccessCount) * 100 / float64(summary.TotalCount)
	}
	return summary
}

func GetRequestStatusMonitorSnapshot(windowEnd int64, pointCount int, intervalSeconds int64) (*RequestStatusMonitor, error) {
	if pointCount <= 0 {
		pointCount = RequestStatusPointCount
	}
	if intervalSeconds <= 0 {
		intervalSeconds = RequestStatusIntervalSeconds
	}
	if windowEnd <= 0 {
		return nil, errors.New("无效的状态监控时间窗口")
	}

	windowStart := windowEnd - int64(pointCount)*intervalSeconds
	points := cloneRequestStatusPoints(windowStart, pointCount, intervalSeconds)
	modelPointMap := make(map[string][]*RequestStatusPoint)

	var rows []*requestStatusLogRow
	if err := LOG_DB.Model(&Log{}).
		Select(fmt.Sprintf("created_at, type, model_name, %s AS group_name", getRequestStatusLogGroupColumn())).
		Where("created_at >= ? AND created_at < ? AND type IN ?", windowStart, windowEnd, []int{LogTypeConsume, LogTypeError}).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		if row == nil {
			continue
		}
		index := int((row.CreatedAt - windowStart) / intervalSeconds)
		if index < 0 || index >= len(points) {
			continue
		}

		groupName := normalizeRequestStatusGroupName(row.GroupName)
		modelName := normalizeRequestStatusModelName(row.ModelName)
		modelKey := buildRequestStatusModelKey(groupName, modelName)
		modelPoints, ok := modelPointMap[modelKey]
		if !ok {
			modelPoints = cloneRequestStatusPoints(windowStart, pointCount, intervalSeconds)
			modelPointMap[modelKey] = modelPoints
		}

		var targetPoints [][]*RequestStatusPoint
		targetPoints = append(targetPoints, points, modelPoints)
		for _, target := range targetPoints {
			switch row.Type {
			case LogTypeConsume:
				target[index].SuccessCount++
			case LogTypeError:
				target[index].ErrorCount++
			}
		}
	}

	models := make([]*RequestStatusModelLine, 0, len(modelPointMap))
	for modelKey, modelPoints := range modelPointMap {
		parts := strings.SplitN(modelKey, "\x00", 2)
		groupName := defaultGroupName
		modelName := unknownModelName
		if len(parts) > 0 && parts[0] != "" {
			groupName = parts[0]
		}
		if len(parts) > 1 && parts[1] != "" {
			modelName = parts[1]
		}
		line := &RequestStatusModelLine{
			GroupName:   groupName,
			ModelName:   modelName,
			DisplayName: buildRequestStatusDisplayName(groupName, modelName),
			Points:      modelPoints,
		}
		line.Summary = finalizeRequestStatusSummary(line.Points)
		models = append(models, line)
	}

	sort.Slice(models, func(i, j int) bool {
		if models[i].Summary.TotalCount == models[j].Summary.TotalCount {
			return models[i].DisplayName < models[j].DisplayName
		}
		return models[i].Summary.TotalCount > models[j].Summary.TotalCount
	})

	return &RequestStatusMonitor{
		GeneratedAt:     common.GetTimestamp(),
		WindowStart:     windowStart,
		WindowEnd:       windowEnd,
		IntervalSeconds: intervalSeconds,
		PointCount:      pointCount,
		Summary:         finalizeRequestStatusSummary(points),
		Points:          points,
		Models:          models,
	}, nil
}
