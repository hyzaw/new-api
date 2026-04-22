package service

import (
	"strings"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func ApplyCacheHitMissMaskByUsage(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) bool {
	if relayInfo == nil || usage == nil {
		return false
	}

	group := ResolveRelayScopeGroup(relayInfo)
	if group == "" {
		return false
	}

	rule, ok := operation_setting.GetCacheHitMissMaskRule(group)
	if !ok {
		return false
	}

	cachedTokens := usage.PromptTokensDetails.CachedTokens
	if cachedTokens <= 0 && usage.InputTokensDetails != nil {
		cachedTokens = usage.InputTokensDetails.CachedTokens
	}
	if cachedTokens <= 0 && usage.PromptCacheHitTokens > 0 {
		cachedTokens = usage.PromptCacheHitTokens
	}
	if cachedTokens <= 0 {
		return false
	}

	maskPercent := operation_setting.RandomCacheHitMissMaskPercent(rule)
	if maskPercent <= 0 {
		return false
	}

	maskTokens := cachedTokens * maskPercent / 100
	if maskTokens <= 0 {
		return false
	}

	usage.PromptTokensDetails.CachedTokens = decreaseButKeepNonNegative(
		usage.PromptTokensDetails.CachedTokens,
		maskTokens,
	)
	if usage.InputTokensDetails != nil {
		usage.InputTokensDetails.CachedTokens = decreaseButKeepNonNegative(
			usage.InputTokensDetails.CachedTokens,
			maskTokens,
		)
	}
	if usage.PromptCacheHitTokens > 0 {
		usage.PromptCacheHitTokens = decreaseButKeepNonNegative(
			usage.PromptCacheHitTokens,
			maskTokens,
		)
	}
	return true
}

func ResolveRelayScopeGroup(relayInfo *relaycommon.RelayInfo) string {
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

func decreaseButKeepNonNegative(current int, delta int) int {
	if current <= 0 || delta <= 0 {
		return current
	}
	if delta >= current {
		return 0
	}
	return current - delta
}
