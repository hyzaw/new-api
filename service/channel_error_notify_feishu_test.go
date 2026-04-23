package service

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/types"
)

func TestRecordChannelConsecutiveErrorThresholdNotifyOnce(t *testing.T) {
	channelConsecutiveErrorStates = sync.Map{}
	originEnabled := setting.ChannelConsecutiveErrorFeishuEnabled
	originThreshold := setting.ChannelConsecutiveErrorFeishuThreshold
	originAppID := setting.ChannelConsecutiveErrorFeishuAppID
	originAppSecret := setting.ChannelConsecutiveErrorFeishuAppSecret
	originChatID := setting.ChannelConsecutiveErrorFeishuChatID
	originSendFunc := sendChannelErrorFeishuCardFunc
	defer func() {
		channelConsecutiveErrorStates = sync.Map{}
		setting.ChannelConsecutiveErrorFeishuEnabled = originEnabled
		setting.ChannelConsecutiveErrorFeishuThreshold = originThreshold
		setting.ChannelConsecutiveErrorFeishuAppID = originAppID
		setting.ChannelConsecutiveErrorFeishuAppSecret = originAppSecret
		setting.ChannelConsecutiveErrorFeishuChatID = originChatID
		sendChannelErrorFeishuCardFunc = originSendFunc
	}()

	setting.ChannelConsecutiveErrorFeishuEnabled = true
	setting.ChannelConsecutiveErrorFeishuThreshold = 2
	setting.ChannelConsecutiveErrorFeishuAppID = "app"
	setting.ChannelConsecutiveErrorFeishuAppSecret = "secret"
	setting.ChannelConsecutiveErrorFeishuChatID = "chat"

	var calls atomic.Int32
	done := make(chan struct{}, 4)
	sendChannelErrorFeishuCardFunc = func(payload channelConsecutiveErrorNotifyPayload) error {
		calls.Add(1)
		done <- struct{}{}
		return nil
	}

	channelError := types.ChannelError{ChannelId: 1, UsingKey: "k1"}
	apiErr := types.NewErrorWithStatusCode(errors.New("boom"), types.ErrorCodeBadResponseStatusCode, 500)

	RecordChannelConsecutiveError(channelError, apiErr, "/v1/chat/completions")
	RecordChannelConsecutiveError(channelError, apiErr, "/v1/chat/completions")
	<-done
	RecordChannelConsecutiveError(channelError, apiErr, "/v1/chat/completions")

	if calls.Load() != 1 {
		t.Fatalf("expected 1 notify call, got %d", calls.Load())
	}
}

func TestResetChannelConsecutiveErrorAllowsRenotify(t *testing.T) {
	channelConsecutiveErrorStates = sync.Map{}
	originEnabled := setting.ChannelConsecutiveErrorFeishuEnabled
	originThreshold := setting.ChannelConsecutiveErrorFeishuThreshold
	originAppID := setting.ChannelConsecutiveErrorFeishuAppID
	originAppSecret := setting.ChannelConsecutiveErrorFeishuAppSecret
	originChatID := setting.ChannelConsecutiveErrorFeishuChatID
	originSendFunc := sendChannelErrorFeishuCardFunc
	defer func() {
		channelConsecutiveErrorStates = sync.Map{}
		setting.ChannelConsecutiveErrorFeishuEnabled = originEnabled
		setting.ChannelConsecutiveErrorFeishuThreshold = originThreshold
		setting.ChannelConsecutiveErrorFeishuAppID = originAppID
		setting.ChannelConsecutiveErrorFeishuAppSecret = originAppSecret
		setting.ChannelConsecutiveErrorFeishuChatID = originChatID
		sendChannelErrorFeishuCardFunc = originSendFunc
	}()

	setting.ChannelConsecutiveErrorFeishuEnabled = true
	setting.ChannelConsecutiveErrorFeishuThreshold = 1
	setting.ChannelConsecutiveErrorFeishuAppID = "app"
	setting.ChannelConsecutiveErrorFeishuAppSecret = "secret"
	setting.ChannelConsecutiveErrorFeishuChatID = "chat"

	var calls atomic.Int32
	done := make(chan struct{}, 4)
	sendChannelErrorFeishuCardFunc = func(payload channelConsecutiveErrorNotifyPayload) error {
		calls.Add(1)
		done <- struct{}{}
		return nil
	}

	channelError := types.ChannelError{ChannelId: 2, UsingKey: "k2"}
	apiErr := types.NewErrorWithStatusCode(errors.New("boom"), types.ErrorCodeBadResponseStatusCode, 500)

	RecordChannelConsecutiveError(channelError, apiErr, "/v1/chat/completions")
	<-done
	ResetChannelConsecutiveError(2, "k2")
	RecordChannelConsecutiveError(channelError, apiErr, "/v1/chat/completions")
	<-done

	if calls.Load() != 2 {
		t.Fatalf("expected 2 notify calls after reset, got %d", calls.Load())
	}
}
