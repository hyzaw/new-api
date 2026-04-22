package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

func ApplyLargePromptRPMByUsage(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage) {
	inputTokens, ok := actualInputTokensFromUsage(ctx, usage)
	if !ok {
		return
	}
	applyLargePromptRPM(relayInfo, inputTokens)
}

func ApplyLargePromptRPMByRealtimeUsage(relayInfo *relaycommon.RelayInfo, usage *dto.RealtimeUsage) {
	if usage == nil || usage.InputTokens <= 0 {
		return
	}
	applyLargePromptRPM(relayInfo, usage.InputTokens)
}

func applyLargePromptRPM(relayInfo *relaycommon.RelayInfo, inputTokens int) {
	if relayInfo == nil || inputTokens <= 0 {
		return
	}
	if !setting.ModelRequestRateLimitEnabled || setting.ModelRequestRateLimitDurationMinutes <= 0 {
		return
	}

	group := resolveLargePromptRPMScopeGroup(relayInfo)
	if group == "" {
		return
	}

	rule, matched := operation_setting.GetLargePromptRPMRule(group, inputTokens)
	if !matched {
		return
	}

	operation_setting.SetTemporaryLargePromptRPM(relayInfo.UserId, group, rule.TemporaryRPM)
}

func actualInputTokensFromUsage(ctx *gin.Context, usage *dto.Usage) (int, bool) {
	if usage == nil {
		return 0, false
	}
	if ctx != nil && common.GetContextKeyBool(ctx, constant.ContextKeyLocalCountTokens) {
		return 0, false
	}
	if usage.InputTokens > 0 {
		return usage.InputTokens, true
	}
	if usage.PromptTokens > 0 {
		return usage.PromptTokens, true
	}
	return 0, false
}

func resolveLargePromptRPMScopeGroup(relayInfo *relaycommon.RelayInfo) string {
	if relayInfo == nil {
		return ""
	}

	group := strings.TrimSpace(relayInfo.TokenGroup)
	if group != "" {
		return group
	}

	group = strings.TrimSpace(relayInfo.UserGroup)
	if group != "" {
		return group
	}

	return strings.TrimSpace(relayInfo.UsingGroup)
}
