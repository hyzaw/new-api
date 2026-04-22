package operation_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type GiftQuotaRule struct {
	Group string `json:"group"`
	Model string `json:"model"`
}

type GiftQuotaSetting struct {
	Rules []GiftQuotaRule `json:"rules"`
}

var giftQuotaSetting = GiftQuotaSetting{
	Rules: []GiftQuotaRule{},
}

func init() {
	config.GlobalConfig.Register("gift_quota_setting", &giftQuotaSetting)
}

func normalizeGiftQuotaRule(rule GiftQuotaRule) GiftQuotaRule {
	group := strings.TrimSpace(rule.Group)
	model := strings.TrimSpace(rule.Model)
	if group == "" {
		group = "*"
	}
	if model == "" {
		model = "*"
	}
	return GiftQuotaRule{
		Group: group,
		Model: model,
	}
}

func NormalizeGiftQuotaRules(rules []GiftQuotaRule) []GiftQuotaRule {
	if len(rules) == 0 {
		return []GiftQuotaRule{}
	}
	normalized := make([]GiftQuotaRule, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		current := normalizeGiftQuotaRule(rule)
		key := strings.ToLower(current.Group) + "\n" + strings.ToLower(current.Model)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, current)
	}
	return normalized
}

func ParseGiftQuotaRules(raw string) ([]GiftQuotaRule, error) {
	if strings.TrimSpace(raw) == "" {
		return []GiftQuotaRule{}, nil
	}
	var rules []GiftQuotaRule
	if err := common.UnmarshalJsonStr(raw, &rules); err != nil {
		return nil, err
	}
	return NormalizeGiftQuotaRules(rules), nil
}

func GetGiftQuotaSetting() *GiftQuotaSetting {
	return &giftQuotaSetting
}

func IsGiftQuotaAllowed(group string, model string) bool {
	group = strings.TrimSpace(group)
	model = strings.TrimSpace(model)
	if group == "" || model == "" {
		return false
	}
	for _, rawRule := range giftQuotaSetting.Rules {
		rule := normalizeGiftQuotaRule(rawRule)
		groupMatch := rule.Group == "*" || strings.EqualFold(rule.Group, group)
		modelMatch := rule.Model == "*" || strings.EqualFold(rule.Model, model)
		if groupMatch && modelMatch {
			return true
		}
	}
	return false
}
