package controller

import (
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func TestNewChannelAutoTestConfigUsesOverrides(t *testing.T) {
	monitorSetting := operation_setting.GetMonitorSetting()
	originalMinutes := monitorSetting.AutoTestChannelMinutes
	originalDisableThreshold := common.ChannelDisableThreshold
	t.Cleanup(func() {
		monitorSetting.AutoTestChannelMinutes = originalMinutes
		common.ChannelDisableThreshold = originalDisableThreshold
	})

	monitorSetting.AutoTestChannelMinutes = 12
	common.ChannelDisableThreshold = 5

	channel := &model.Channel{
		Models: "gpt-4o,gpt-4.1-mini",
	}
	channel.SetSetting(dto.ChannelSettings{
		MonitorIntervalMinutes:  3,
		MonitorEnableThreshold:  1.2,
		MonitorDisableThreshold: 2.4,
		MonitorModels:           []string{" gpt-4o ", "", "gpt-4.1-mini", "gpt-4o"},
	})

	config := newChannelAutoTestConfig(channel)
	if config.Interval != 3*time.Minute {
		t.Fatalf("expected 3 minute interval, got %s", config.Interval)
	}
	if config.EnableThresholdMs != 1200 {
		t.Fatalf("expected enable threshold 1200ms, got %d", config.EnableThresholdMs)
	}
	if config.DisableThresholdMs != 2400 {
		t.Fatalf("expected disable threshold 2400ms, got %d", config.DisableThresholdMs)
	}
	if len(config.MonitorModels) != 2 {
		t.Fatalf("expected 2 normalized monitor models, got %v", config.MonitorModels)
	}
}

func TestChannelAutoTestResolveModelsAllUsesChannelModels(t *testing.T) {
	channel := &model.Channel{
		Models: "gpt-4o,gpt-4.1-mini,gpt-4o",
	}
	config := channelAutoTestConfig{
		MonitorModels: []string{channelMonitorAllModelsValue},
	}

	models := config.ResolveModels(channel)
	if len(models) != 2 {
		t.Fatalf("expected 2 channel models, got %v", models)
	}
	if models[0] != "gpt-4o" || models[1] != "gpt-4.1-mini" {
		t.Fatalf("unexpected resolved models: %v", models)
	}
}

func TestShouldRunChannelAutoTestByInterval(t *testing.T) {
	const channelID = 987654
	channelAutoTestLastRun.Delete(channelID)
	t.Cleanup(func() {
		channelAutoTestLastRun.Delete(channelID)
	})

	now := time.Now()
	if !shouldRunChannelAutoTest(channelID, 5*time.Minute, now) {
		t.Fatalf("expected first auto test to run")
	}

	markChannelAutoTestRun(channelID, now)
	if shouldRunChannelAutoTest(channelID, 5*time.Minute, now.Add(4*time.Minute)) {
		t.Fatalf("expected auto test to wait for interval")
	}
	if !shouldRunChannelAutoTest(channelID, 5*time.Minute, now.Add(5*time.Minute)) {
		t.Fatalf("expected auto test to run after interval")
	}
}

func TestValidateChannelRejectsInvalidMonitorThresholds(t *testing.T) {
	channel := &model.Channel{}
	channel.SetSetting(dto.ChannelSettings{
		MonitorEnableThreshold:  3,
		MonitorDisableThreshold: 2,
	})

	err := validateChannel(channel, false)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "启用延迟阈值不能大于禁用延迟阈值") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestHasChannelSpecificMonitorConfig(t *testing.T) {
	channel := &model.Channel{}
	if hasChannelSpecificMonitorConfig(channel) {
		t.Fatalf("expected empty channel monitor config to be false")
	}

	channel.SetSetting(dto.ChannelSettings{
		MonitorEnableThreshold: 1.5,
	})
	if !hasChannelSpecificMonitorConfig(channel) {
		t.Fatalf("expected channel monitor config override to be detected")
	}
}

func TestShouldRunAutomaticChannelTestForChannelIgnoresGlobalWhenChannelConfigured(t *testing.T) {
	monitorSetting := operation_setting.GetMonitorSetting()
	originalEnabled := monitorSetting.AutoTestChannelEnabled
	t.Cleanup(func() {
		monitorSetting.AutoTestChannelEnabled = originalEnabled
	})

	monitorSetting.AutoTestChannelEnabled = false

	channel := &model.Channel{}
	channel.SetSetting(dto.ChannelSettings{
		MonitorIntervalMinutes: 3,
	})
	if !shouldRunAutomaticChannelTestForChannel(channel) {
		t.Fatalf("expected channel-specific monitor config to enable auto test")
	}
}

func TestShouldAutomaticallyEnableChannelForChannelIgnoresGlobalWhenChannelConfigured(t *testing.T) {
	original := common.AutomaticEnableChannelEnabled
	t.Cleanup(func() {
		common.AutomaticEnableChannelEnabled = original
	})

	common.AutomaticEnableChannelEnabled = false
	channel := &model.Channel{}
	channel.SetSetting(dto.ChannelSettings{
		MonitorModels: []string{"gpt-4o"},
	})
	if !shouldAutomaticallyEnableChannelForChannel(channel) {
		t.Fatalf("expected channel-specific monitor config to enable auto enable")
	}
}

func TestShouldAutomaticallyDisableByThresholdForChannelIgnoresGlobalWhenChannelConfigured(t *testing.T) {
	original := common.AutomaticDisableChannelEnabled
	t.Cleanup(func() {
		common.AutomaticDisableChannelEnabled = original
	})

	common.AutomaticDisableChannelEnabled = false
	channel := &model.Channel{}
	channel.SetSetting(dto.ChannelSettings{
		MonitorDisableThreshold: 2.5,
	})
	if !shouldAutomaticallyDisableByThresholdForChannel(channel) {
		t.Fatalf("expected channel-specific monitor config to enable threshold auto disable")
	}
}
