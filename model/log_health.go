package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
)

const (
	RequestStatusIntervalSeconds = int64(10 * 60)
	RequestStatusPointCount      = 144
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

type RequestStatusMonitor struct {
	GeneratedAt     int64                 `json:"generated_at"`
	WindowStart     int64                 `json:"window_start"`
	WindowEnd       int64                 `json:"window_end"`
	IntervalSeconds int64                 `json:"interval_seconds"`
	PointCount      int                   `json:"point_count"`
	Summary         RequestStatusSummary  `json:"summary"`
	Points          []*RequestStatusPoint `json:"points"`
}

type requestStatusLogRow struct {
	CreatedAt int64 `gorm:"column:created_at"`
	Type      int   `gorm:"column:type"`
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
	points := make([]*RequestStatusPoint, 0, pointCount)
	for i := 0; i < pointCount; i++ {
		start := windowStart + int64(i)*intervalSeconds
		points = append(points, &RequestStatusPoint{
			StartTime: start,
			EndTime:   start + intervalSeconds,
		})
	}

	var rows []*requestStatusLogRow
	if err := LOG_DB.Model(&Log{}).
		Select("created_at, type").
		Where("created_at >= ? AND created_at < ? AND type IN ?", windowStart, windowEnd, []int{LogTypeConsume, LogTypeError}).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	summary := RequestStatusSummary{}
	for _, row := range rows {
		if row == nil {
			continue
		}
		index := int((row.CreatedAt - windowStart) / intervalSeconds)
		if index < 0 || index >= len(points) {
			continue
		}

		point := points[index]
		switch row.Type {
		case LogTypeConsume:
			point.SuccessCount++
			summary.SuccessCount++
		case LogTypeError:
			point.ErrorCount++
			summary.ErrorCount++
		default:
			continue
		}
	}

	for _, point := range points {
		point.TotalCount = point.SuccessCount + point.ErrorCount
		if point.TotalCount > 0 {
			point.SuccessRate = float64(point.SuccessCount) * 100 / float64(point.TotalCount)
		}
		point.Status = classifyRequestStatus(point.SuccessRate, point.TotalCount)

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

	return &RequestStatusMonitor{
		GeneratedAt:     common.GetTimestamp(),
		WindowStart:     windowStart,
		WindowEnd:       windowEnd,
		IntervalSeconds: intervalSeconds,
		PointCount:      pointCount,
		Summary:         summary,
		Points:          points,
	}, nil
}
