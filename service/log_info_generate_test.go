package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestGenerateTextOtherInfoIncludesAdminTiming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	entryTime := time.Unix(1000, 0)
	startTime := entryTime.Add(120 * time.Millisecond)
	upstreamRequestAt := startTime.Add(80 * time.Millisecond)
	upstreamResponseAt := upstreamRequestAt.Add(900 * time.Millisecond)
	upstreamFirstByteAt := upstreamResponseAt.Add(30 * time.Millisecond)
	upstreamFirstLineAt := upstreamFirstByteAt.Add(20 * time.Millisecond)
	firstResponseTime := upstreamFirstLineAt.Add(200 * time.Millisecond)

	common.SetContextKey(ctx, constant.ContextKeyRequestEntryTime, entryTime)
	ctx.Set("use_channel", []string{"3", "6"})

	info := &relaycommon.RelayInfo{
		StartTime:                    startTime,
		FirstResponseTime:            firstResponseTime,
		UpstreamRequestAt:            upstreamRequestAt,
		UpstreamResponseAt:           upstreamResponseAt,
		UpstreamFirstByteAt:          upstreamFirstByteAt,
		UpstreamFirstLineAt:          upstreamFirstLineAt,
		AppliedGroupDelay:            2 * time.Second,
		PreFirstDataLineCount:        2,
		PreFirstDataEmptyLineCount:   1,
		PreFirstDataNonDataLineCount: 2,
		PreFirstDataPreview:          []string{": ping", "event: metadata"},
		ChannelMeta:                  &relaycommon.ChannelMeta{},
		IsStream:                     true,
	}

	other := GenerateTextOtherInfo(ctx, info, 1, 1, 1, 0, 0, 0, 1)
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected admin_info map, got %T", other["admin_info"])
	}
	timing, ok := adminInfo["timing"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected timing map, got %T", adminInfo["timing"])
	}

	assertTimingValue(t, timing, "entry_to_relay_ms", int64(120))
	assertTimingValue(t, timing, "group_delay_ms", int64(2000))
	assertTimingValue(t, timing, "relay_to_upstream_ms", int64(80))
	assertTimingValue(t, timing, "upstream_headers_ms", int64(900))
	assertTimingValue(t, timing, "headers_to_first_byte_ms", int64(30))
	assertTimingValue(t, timing, "headers_to_first_line_ms", int64(50))
	assertTimingValue(t, timing, "first_byte_to_first_line_ms", int64(20))
	assertTimingValue(t, timing, "headers_to_first_token_ms", int64(250))
	assertTimingValue(t, timing, "first_line_to_first_token_ms", int64(200))
	assertTimingValue(t, timing, "relay_to_first_token_ms", int64(1230))
	assertTimingValue(t, timing, "entry_to_first_token_ms", int64(1350))
	assertTimingValue(t, timing, "retry_count", int64(1))

	probe, ok := adminInfo["stream_probe"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected stream_probe map, got %T", adminInfo["stream_probe"])
	}
	assertTimingValue(t, probe, "lines_before_first_data", int64(2))
	assertTimingValue(t, probe, "empty_lines_before_first_data", int64(1))
	assertTimingValue(t, probe, "non_data_lines_before_first_data", int64(2))
	lines, ok := probe["preview_lines_before_first_data"].([]string)
	if !ok {
		t.Fatalf("expected preview lines slice, got %T", probe["preview_lines_before_first_data"])
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 preview lines, got %d", len(lines))
	}
}

func assertTimingValue(t *testing.T, timing map[string]interface{}, key string, want int64) {
	t.Helper()
	got, ok := timing[key]
	if !ok {
		t.Fatalf("expected timing[%q] to exist", key)
	}

	switch v := got.(type) {
	case int64:
		if v != want {
			t.Fatalf("timing[%q] = %d, want %d", key, v, want)
		}
	case int:
		if int64(v) != want {
			t.Fatalf("timing[%q] = %d, want %d", key, v, want)
		}
	case float64:
		if int64(v) != want {
			t.Fatalf("timing[%q] = %v, want %d", key, v, want)
		}
	default:
		t.Fatalf("timing[%q] has unexpected type %T", key, got)
	}
}
