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
	firstResponseTime := upstreamResponseAt.Add(250 * time.Millisecond)

	common.SetContextKey(ctx, constant.ContextKeyRequestEntryTime, entryTime)
	ctx.Set("use_channel", []string{"3", "6"})

	info := &relaycommon.RelayInfo{
		StartTime:          startTime,
		FirstResponseTime:  firstResponseTime,
		UpstreamRequestAt:  upstreamRequestAt,
		UpstreamResponseAt: upstreamResponseAt,
		AppliedGroupDelay:  2 * time.Second,
		ChannelMeta:        &relaycommon.ChannelMeta{},
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
	assertTimingValue(t, timing, "headers_to_first_token_ms", int64(250))
	assertTimingValue(t, timing, "relay_to_first_token_ms", int64(1230))
	assertTimingValue(t, timing, "entry_to_first_token_ms", int64(1350))
	assertTimingValue(t, timing, "retry_count", int64(1))
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
