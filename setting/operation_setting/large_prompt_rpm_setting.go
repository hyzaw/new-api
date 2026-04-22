package operation_setting

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/config"
)

type LargePromptRPMRule struct {
	Group           string `json:"group"`
	ThresholdK      int    `json:"threshold_k"`
	TemporaryRPM    int    `json:"temporary_rpm"`
	DurationMinutes int    `json:"duration_minutes"`
}

type LargePromptRPMSetting struct {
	Rules []LargePromptRPMRule `json:"rules"`
}

type temporaryLargePromptRPMEntry struct {
	RPM       int
	ExpiresAt time.Time
}

var largePromptRPMSetting = LargePromptRPMSetting{
	Rules: []LargePromptRPMRule{},
}

var temporaryLargePromptRPMStore sync.Map

func init() {
	config.GlobalConfig.Register("large_prompt_rpm_setting", &largePromptRPMSetting)
}

func ParseLargePromptRPMRules(raw string) ([]LargePromptRPMRule, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return []LargePromptRPMRule{}, nil
	}

	var rules []LargePromptRPMRule
	if err := common.UnmarshalJsonStr(raw, &rules); err != nil {
		return nil, fmt.Errorf("大输入临时 RPM 规则格式不正确")
	}

	return normalizeLargePromptRPMRules(rules)
}

func UpdateLargePromptRPMRulesByJSONString(raw string) error {
	rules, err := ParseLargePromptRPMRules(raw)
	if err != nil {
		return err
	}
	largePromptRPMSetting.Rules = rules
	return nil
}

func GetLargePromptRPMRule(group string, promptTokens int) (LargePromptRPMRule, bool) {
	group = strings.TrimSpace(group)
	if group == "" || promptTokens <= 0 {
		return LargePromptRPMRule{}, false
	}

	for _, rule := range largePromptRPMSetting.Rules {
		if strings.TrimSpace(rule.Group) != group {
			continue
		}
		if promptTokens > rule.ThresholdK*1000 {
			return rule, true
		}
		return LargePromptRPMRule{}, false
	}

	return LargePromptRPMRule{}, false
}

func SetTemporaryLargePromptRPM(userID int, group string, rpm int, durationMinutes int) {
	group = strings.TrimSpace(group)
	if userID <= 0 || group == "" || rpm <= 0 {
		return
	}

	ttl := temporaryLargePromptRPMTTL(durationMinutes)
	if ttl <= 0 {
		return
	}

	key := largePromptRPMTempKey(userID, group)
	entry := temporaryLargePromptRPMEntry{
		RPM:       rpm,
		ExpiresAt: time.Now().Add(ttl),
	}
	temporaryLargePromptRPMStore.Store(key, entry)

	if common.RedisEnabled {
		_ = common.RedisSet(key, strconv.Itoa(rpm), ttl)
	}
}

func GetTemporaryLargePromptRPM(userID int, group string) (int, bool) {
	group = strings.TrimSpace(group)
	if userID <= 0 || group == "" {
		return 0, false
	}

	key := largePromptRPMTempKey(userID, group)
	if rpm, ok := getTemporaryLargePromptRPMFromMemory(key); ok {
		return rpm, true
	}

	if !common.RedisEnabled {
		return 0, false
	}

	raw, err := common.RedisGet(key)
	if err != nil {
		return 0, false
	}

	rpm, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || rpm <= 0 {
		return 0, false
	}

	ttl := getTemporaryLargePromptRPMRedisTTL(key)
	if ttl <= 0 {
		ttl = temporaryLargePromptRPMTTL(0)
	}
	temporaryLargePromptRPMStore.Store(key, temporaryLargePromptRPMEntry{
		RPM:       rpm,
		ExpiresAt: time.Now().Add(ttl),
	})
	return rpm, true
}

func ResetTemporaryLargePromptRPMStore() {
	temporaryLargePromptRPMStore = sync.Map{}
}

func getTemporaryLargePromptRPMFromMemory(key string) (int, bool) {
	value, ok := temporaryLargePromptRPMStore.Load(key)
	if !ok {
		return 0, false
	}

	entry, ok := value.(temporaryLargePromptRPMEntry)
	if !ok {
		temporaryLargePromptRPMStore.Delete(key)
		return 0, false
	}

	if time.Now().After(entry.ExpiresAt) {
		temporaryLargePromptRPMStore.Delete(key)
		return 0, false
	}

	if entry.RPM <= 0 {
		return 0, false
	}

	return entry.RPM, true
}

func largePromptRPMTempKey(userID int, group string) string {
	return fmt.Sprintf("rateLimit:tempRPM:%d:%s", userID, strings.TrimSpace(group))
}

func getTemporaryLargePromptRPMRedisTTL(key string) time.Duration {
	if !common.RedisEnabled || common.RDB == nil {
		return 0
	}

	ttl, err := common.RDB.TTL(context.Background(), key).Result()
	if err != nil || ttl <= 0 {
		return 0
	}
	return ttl
}

func temporaryLargePromptRPMTTL(durationMinutes int) time.Duration {
	if durationMinutes <= 0 {
		durationMinutes = setting.ModelRequestRateLimitDurationMinutes
	}
	if durationMinutes <= 0 {
		return 0
	}
	return time.Duration(durationMinutes) * time.Minute
}

func normalizeLargePromptRPMRules(rules []LargePromptRPMRule) ([]LargePromptRPMRule, error) {
	normalized := make([]LargePromptRPMRule, 0, len(rules))
	seenGroups := make(map[string]struct{}, len(rules))

	for _, rule := range rules {
		rule.Group = strings.TrimSpace(rule.Group)
		if rule.Group == "" {
			return nil, fmt.Errorf("大输入临时 RPM 规则中的分组名不能为空")
		}
		if rule.ThresholdK <= 0 {
			return nil, fmt.Errorf("分组 %s 的输入阈值必须大于 0", rule.Group)
		}
		if rule.TemporaryRPM <= 0 {
			return nil, fmt.Errorf("分组 %s 的临时 RPM 必须大于 0", rule.Group)
		}
		if rule.DurationMinutes < 0 {
			return nil, fmt.Errorf("分组 %s 的限制时长不能小于 0", rule.Group)
		}
		if _, exists := seenGroups[rule.Group]; exists {
			return nil, fmt.Errorf("分组 %s 的大输入临时 RPM 规则重复", rule.Group)
		}
		seenGroups[rule.Group] = struct{}{}
		normalized = append(normalized, rule)
	}

	return normalized, nil
}
