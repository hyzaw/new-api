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

func TestRecordConsumeLogStoresAdminOnlyBodies(t *testing.T) {
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
	adminInfo, ok := otherMap["admin_info"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing admin_info: %v", otherMap)
	}
	reqBody, ok := adminInfo["request_body"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing request_body: %v", adminInfo)
	}
	if got := reqBody["body"]; got != `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}` {
		t.Fatalf("unexpected request body: %v", got)
	}
	respBody, ok := adminInfo["response_body"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing response_body: %v", adminInfo)
	}
	if got := respBody["body"]; got != `{"id":"chatcmpl-test","choices":[]}` {
		t.Fatalf("unexpected response body: %v", got)
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
