package model

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func TestNormalizeClientIPCountryCode(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "upper alpha2", raw: "US", want: "US"},
		{name: "lower alpha2", raw: "cn", want: "CN"},
		{name: "trim spaces", raw: " jp ", want: "JP"},
		{name: "empty", raw: "", want: ""},
		{name: "too long", raw: "USA", want: ""},
		{name: "contains digit", raw: "C1", want: ""},
		{name: "contains symbol", raw: "U-", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeClientIPCountryCode(tt.raw); got != tt.want {
				t.Fatalf("normalizeClientIPCountryCode(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestRecordConsumeLogStoresUserAgent(t *testing.T) {
	truncateTables(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	req.Header.Set("User-Agent", "new-api-test-agent/1.0")
	ctx.Request = req
	ctx.Set("username", "tester")
	ctx.Set(common.RequestIdKey, "req-ua-test")

	RecordConsumeLog(ctx, 1, RecordConsumeLogParams{
		ChannelId:        1,
		ModelName:        "gpt-4o-mini",
		TokenName:        "token-a",
		Content:          "ok",
		TokenId:          1,
		Group:            "default",
		PromptTokens:     10,
		CompletionTokens: 5,
	})

	var log Log
	if err := LOG_DB.Last(&log).Error; err != nil {
		t.Fatalf("failed to load log: %v", err)
	}

	otherMap, err := common.StrToMap(log.Other)
	if err != nil {
		t.Fatalf("failed to parse log other: %v", err)
	}
	if got := otherMap["user_agent"]; got != "new-api-test-agent/1.0" {
		t.Fatalf("unexpected user_agent: got %v", got)
	}
}

func TestRecordConsumeLogStoresLogDetail(t *testing.T) {
	truncateTables(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("User-Agent", "new-api-test-agent/1.0")
	ctx.Request = req
	ctx.Set("username", "tester")
	ctx.Set(common.RequestIdKey, "req-body-test")

	storage, err := common.GetBodyStorage(ctx)
	if err != nil {
		t.Fatalf("failed to cache request body: %v", err)
	}
	ctx.Request.Body = io.NopCloser(storage)

	capture := common.NewResponseBodyCapture(ctx.Writer)
	common.SetResponseBodyCapture(ctx, capture)
	ctx.Writer = capture
	if _, err = ctx.Writer.Write([]byte(`{"id":"chatcmpl-test","choices":[]}`)); err != nil {
		t.Fatalf("failed to write response body: %v", err)
	}

	RecordConsumeLog(ctx, 1, RecordConsumeLogParams{
		ChannelId:        1,
		ModelName:        "gpt-4o-mini",
		TokenName:        "token-a",
		Content:          "ok",
		TokenId:          1,
		Group:            "default",
		PromptTokens:     10,
		CompletionTokens: 5,
	})

	var log Log
	if err := LOG_DB.Last(&log).Error; err != nil {
		t.Fatalf("failed to load log: %v", err)
	}

	otherMap, err := common.StrToMap(log.Other)
	if err != nil {
		t.Fatalf("failed to parse log other: %v", err)
	}
	if adminInfo, ok := otherMap["admin_info"].(map[string]interface{}); ok {
		if _, exists := adminInfo["request_body"]; exists {
			t.Fatalf("request_body should not be stored in logs.other: %v", adminInfo)
		}
		if _, exists := adminInfo["response_body"]; exists {
			t.Fatalf("response_body should not be stored in logs.other: %v", adminInfo)
		}
	}

	detail, err := GetLogDetail(log.Id)
	if err != nil {
		t.Fatalf("failed to load log detail: %v", err)
	}
	if detail.RequestBody != `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}` {
		t.Fatalf("unexpected request body: %v", detail.RequestBody)
	}
	if detail.ResponseBody != `{"id":"chatcmpl-test","choices":[]}` {
		t.Fatalf("unexpected response body: %v", detail.ResponseBody)
	}

	markLogsHasDetail([]*Log{&log})
	if !log.HasDetail {
		t.Fatalf("expected log has_detail to be true")
	}
	formatUserLogs([]*Log{&log}, 0)
	userOther, err := common.StrToMap(log.Other)
	if err != nil {
		t.Fatalf("failed to parse user log other: %v", err)
	}
	if _, exists := userOther["admin_info"]; exists {
		t.Fatalf("admin_info should be hidden from user logs: %v", userOther)
	}
}

func TestRecordErrorLogStoresLogDetail(t *testing.T) {
	truncateTables(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("User-Agent", "new-api-test-agent/1.0")
	ctx.Request = req
	ctx.Set("username", "tester")
	ctx.Set(common.RequestIdKey, "req-error-body-test")

	storage, err := common.GetBodyStorage(ctx)
	if err != nil {
		t.Fatalf("failed to cache request body: %v", err)
	}
	ctx.Request.Body = io.NopCloser(storage)

	capture := common.NewResponseBodyCapture(ctx.Writer)
	common.SetResponseBodyCapture(ctx, capture)
	ctx.Writer = capture
	if _, err = ctx.Writer.Write([]byte(`{"error":{"message":"boom"}}`)); err != nil {
		t.Fatalf("failed to write response body: %v", err)
	}

	RecordErrorLog(ctx, 1, 1, "gpt-4o-mini", "token-a", "upstream failed", 1, 0, false, "default", nil)

	var log Log
	if err := LOG_DB.Last(&log).Error; err != nil {
		t.Fatalf("failed to load log: %v", err)
	}

	detail, err := GetLogDetail(log.Id)
	if err != nil {
		t.Fatalf("failed to load log detail: %v", err)
	}
	if detail.RequestBody != `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}` {
		t.Fatalf("unexpected request body: %v", detail.RequestBody)
	}
	if detail.ResponseBody != `{"error":{"message":"boom"}}` {
		t.Fatalf("unexpected response body: %v", detail.ResponseBody)
	}
	if detail.RequestBodyEncoding != common.LogBodyEncodingText || detail.ResponseBodyEncoding != common.LogBodyEncodingText {
		t.Fatalf("unexpected encodings: request=%s response=%s", detail.RequestBodyEncoding, detail.ResponseBodyEncoding)
	}

	markLogsHasDetail([]*Log{&log})
	if !log.HasDetail {
		t.Fatalf("expected log has_detail to be true")
	}
}
