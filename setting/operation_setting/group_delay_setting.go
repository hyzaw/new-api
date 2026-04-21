package operation_setting

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type GroupDelayRule struct {
	Group      string `json:"group"`
	MinSeconds int    `json:"min_seconds"`
	MaxSeconds int    `json:"max_seconds"`
}

type GroupDelaySetting struct {
	Rules []GroupDelayRule `json:"rules"`
}

var groupDelaySetting = GroupDelaySetting{
	Rules: []GroupDelayRule{},
}

func init() {
	config.GlobalConfig.Register("group_delay_setting", &groupDelaySetting)
}

func GetGroupDelaySetting() *GroupDelaySetting {
	return &groupDelaySetting
}

func ParseGroupDelayRules(raw string) ([]GroupDelayRule, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return []GroupDelayRule{}, nil
	}

	var rules []GroupDelayRule
	if err := common.UnmarshalJsonStr(raw, &rules); err != nil {
		return nil, fmt.Errorf("分组延迟规则格式不正确")
	}

	return normalizeGroupDelayRules(rules)
}

func UpdateGroupDelayRulesByJSONString(raw string) error {
	rules, err := ParseGroupDelayRules(raw)
	if err != nil {
		return err
	}
	groupDelaySetting.Rules = rules
	return nil
}

func GetGroupDelayDuration(group string) time.Duration {
	group = strings.TrimSpace(group)
	if group == "" {
		return 0
	}

	for _, rule := range groupDelaySetting.Rules {
		if strings.TrimSpace(rule.Group) != group {
			continue
		}

		minSeconds := rule.MinSeconds
		maxSeconds := rule.MaxSeconds
		if maxSeconds < minSeconds {
			maxSeconds = minSeconds
		}
		if maxSeconds <= 0 {
			return 0
		}

		delaySeconds := minSeconds
		if maxSeconds > minSeconds {
			delaySeconds += rand.IntN(maxSeconds - minSeconds + 1)
		}
		if delaySeconds <= 0 {
			return 0
		}
		return time.Duration(delaySeconds) * time.Second
	}

	return 0
}

func normalizeGroupDelayRules(rules []GroupDelayRule) ([]GroupDelayRule, error) {
	normalized := make([]GroupDelayRule, 0, len(rules))
	seenGroups := make(map[string]struct{}, len(rules))

	for _, rule := range rules {
		rule.Group = strings.TrimSpace(rule.Group)
		if rule.Group == "" {
			return nil, fmt.Errorf("分组延迟规则中的分组名不能为空")
		}
		if rule.MinSeconds < 0 || rule.MaxSeconds < 0 {
			return nil, fmt.Errorf("分组 %s 的延迟秒数不能小于 0", rule.Group)
		}
		if rule.MaxSeconds < rule.MinSeconds {
			return nil, fmt.Errorf("分组 %s 的最大延迟不能小于最小延迟", rule.Group)
		}
		if _, exists := seenGroups[rule.Group]; exists {
			return nil, fmt.Errorf("分组 %s 的延迟规则重复", rule.Group)
		}
		seenGroups[rule.Group] = struct{}{}
		normalized = append(normalized, rule)
	}

	return normalized, nil
}
