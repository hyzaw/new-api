package operation_setting

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type CacheHitMissMaskRule struct {
	Group      string `json:"group"`
	MinPercent int    `json:"min_percent"`
	MaxPercent int    `json:"max_percent"`
}

type CacheHitMissMaskSetting struct {
	Rules []CacheHitMissMaskRule `json:"rules"`
}

var cacheHitMissMaskSetting = CacheHitMissMaskSetting{
	Rules: []CacheHitMissMaskRule{},
}

func init() {
	config.GlobalConfig.Register("cache_hit_miss_mask_setting", &cacheHitMissMaskSetting)
}

func ParseCacheHitMissMaskRules(raw string) ([]CacheHitMissMaskRule, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return []CacheHitMissMaskRule{}, nil
	}

	var rules []CacheHitMissMaskRule
	if err := common.UnmarshalJsonStr(raw, &rules); err != nil {
		return nil, fmt.Errorf("缓存命中转未命中规则格式不正确")
	}

	return normalizeCacheHitMissMaskRules(rules)
}

func UpdateCacheHitMissMaskRulesByJSONString(raw string) error {
	rules, err := ParseCacheHitMissMaskRules(raw)
	if err != nil {
		return err
	}
	cacheHitMissMaskSetting.Rules = rules
	return nil
}

func GetCacheHitMissMaskRule(group string) (CacheHitMissMaskRule, bool) {
	group = strings.TrimSpace(group)
	if group == "" {
		return CacheHitMissMaskRule{}, false
	}

	for _, rule := range cacheHitMissMaskSetting.Rules {
		if strings.TrimSpace(rule.Group) == group {
			return rule, true
		}
	}

	return CacheHitMissMaskRule{}, false
}

func RandomCacheHitMissMaskPercent(rule CacheHitMissMaskRule) int {
	minPercent := rule.MinPercent
	maxPercent := rule.MaxPercent
	if maxPercent < minPercent {
		maxPercent = minPercent
	}
	if minPercent <= 0 || maxPercent <= 0 {
		return 0
	}
	if minPercent == maxPercent {
		return minPercent
	}
	return minPercent + rand.IntN(maxPercent-minPercent+1)
}

func normalizeCacheHitMissMaskRules(rules []CacheHitMissMaskRule) ([]CacheHitMissMaskRule, error) {
	normalized := make([]CacheHitMissMaskRule, 0, len(rules))
	seenGroups := make(map[string]struct{}, len(rules))

	for _, rule := range rules {
		rule.Group = strings.TrimSpace(rule.Group)
		if rule.Group == "" {
			return nil, fmt.Errorf("缓存命中转未命中规则中的分组名不能为空")
		}
		if rule.MinPercent <= 0 || rule.MaxPercent <= 0 {
			return nil, fmt.Errorf("分组 %s 的比例必须大于 0", rule.Group)
		}
		if rule.MinPercent > 100 || rule.MaxPercent > 100 {
			return nil, fmt.Errorf("分组 %s 的比例不能大于 100", rule.Group)
		}
		if rule.MaxPercent < rule.MinPercent {
			return nil, fmt.Errorf("分组 %s 的最大比例不能小于最小比例", rule.Group)
		}
		if _, exists := seenGroups[rule.Group]; exists {
			return nil, fmt.Errorf("分组 %s 的缓存命中转未命中规则重复", rule.Group)
		}
		seenGroups[rule.Group] = struct{}{}
		normalized = append(normalized, rule)
	}

	return normalized, nil
}
