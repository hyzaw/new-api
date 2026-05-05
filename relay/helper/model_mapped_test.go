package helper

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

func TestModelMappedHelperResponsesCompactKeepsOriginModelForBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("model_mapping", `{"gpt-5.5":"gpt-5.5-2026-04-24"}`)

	request := &dto.OpenAIResponsesRequest{Model: "gpt-5.5"}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponsesCompact,
		OriginModelName: ratio_setting.WithCompactModelSuffix("gpt-5.5"),
		Request:         request,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.5",
		},
	}

	if err := ModelMappedHelper(ctx, info, request); err != nil {
		t.Fatalf("ModelMappedHelper returned error: %v", err)
	}

	if info.OriginModelName != ratio_setting.WithCompactModelSuffix("gpt-5.5") {
		t.Fatalf("origin model = %q, want %q", info.OriginModelName, ratio_setting.WithCompactModelSuffix("gpt-5.5"))
	}
	if info.UpstreamModelName != "gpt-5.5-2026-04-24" {
		t.Fatalf("upstream model = %q, want mapped model", info.UpstreamModelName)
	}
	if request.Model != "gpt-5.5-2026-04-24" {
		t.Fatalf("request model = %q, want mapped upstream model", request.Model)
	}
}

func TestModelMappedHelperResponsesCompactWithoutMappingKeepsCompactOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("model_mapping", "{}")

	request := &dto.OpenAIResponsesCompactionRequest{Model: "gpt-5.5"}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponsesCompact,
		OriginModelName: ratio_setting.WithCompactModelSuffix("gpt-5.5"),
		Request:         request,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.5",
		},
	}

	if err := ModelMappedHelper(ctx, info, request); err != nil {
		t.Fatalf("ModelMappedHelper returned error: %v", err)
	}

	if info.OriginModelName != ratio_setting.WithCompactModelSuffix("gpt-5.5") {
		t.Fatalf("origin model = %q, want %q", info.OriginModelName, ratio_setting.WithCompactModelSuffix("gpt-5.5"))
	}
	if info.UpstreamModelName != "gpt-5.5" {
		t.Fatalf("upstream model = %q, want original upstream model", info.UpstreamModelName)
	}
	if request.Model != "gpt-5.5" {
		t.Fatalf("request model = %q, want original upstream model", request.Model)
	}
}
