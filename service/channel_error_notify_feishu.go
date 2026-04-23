package service

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/google/uuid"
)

type channelConsecutiveErrorState struct {
	mu       sync.Mutex
	count    int
	notified bool
}

type channelConsecutiveErrorNotifyPayload struct {
	ChannelID   int
	ChannelName string
	ChannelType int
	UsingKey    string
	RequestPath string
	Count       int
	StatusCode  int
	ErrorCode   string
	ErrorMsg    string
	OccurredAt  time.Time
}

var (
	channelConsecutiveErrorStates  sync.Map
	sendChannelErrorFeishuCardFunc = sendChannelConsecutiveErrorFeishuCard
)

func RecordChannelConsecutiveError(channelError types.ChannelError, err *types.NewAPIError, requestPath string) {
	if err == nil {
		return
	}

	stateKey := buildChannelConsecutiveErrorStateKey(channelError.ChannelId, channelError.UsingKey)
	if !setting.ChannelConsecutiveErrorFeishuEnabled || setting.ChannelConsecutiveErrorFeishuThreshold <= 0 {
		channelConsecutiveErrorStates.Delete(stateKey)
		return
	}
	stateAny, _ := channelConsecutiveErrorStates.LoadOrStore(stateKey, &channelConsecutiveErrorState{})
	state := stateAny.(*channelConsecutiveErrorState)

	threshold := setting.ChannelConsecutiveErrorFeishuThreshold

	var payload *channelConsecutiveErrorNotifyPayload
	state.mu.Lock()
	state.count++
	count := state.count
	if setting.IsChannelConsecutiveErrorFeishuReady() && !state.notified && count >= threshold {
		state.notified = true
		payload = &channelConsecutiveErrorNotifyPayload{
			ChannelID:   channelError.ChannelId,
			ChannelName: channelError.ChannelName,
			ChannelType: channelError.ChannelType,
			UsingKey:    channelError.UsingKey,
			RequestPath: requestPath,
			Count:       count,
			StatusCode:  err.StatusCode,
			ErrorCode:   string(err.GetErrorCode()),
			ErrorMsg:    err.MaskSensitiveErrorWithStatusCode(),
			OccurredAt:  time.Now(),
		}
	}
	state.mu.Unlock()

	if payload == nil {
		return
	}

	gopool.Go(func() {
		if notifyErr := sendChannelErrorFeishuCardFunc(*payload); notifyErr != nil {
			common.SysLog(fmt.Sprintf("failed to send feishu channel consecutive error notification for channel #%d: %s", payload.ChannelID, notifyErr.Error()))
			state.mu.Lock()
			state.notified = false
			state.mu.Unlock()
		}
	})
}

func ResetChannelConsecutiveError(channelID int, usingKey string) {
	if channelID <= 0 {
		return
	}
	channelConsecutiveErrorStates.Delete(buildChannelConsecutiveErrorStateKey(channelID, usingKey))
}

func buildChannelConsecutiveErrorStateKey(channelID int, usingKey string) string {
	return fmt.Sprintf("%d:%s", channelID, usingKey)
}

func buildChannelConsecutiveErrorFeishuMessageUUID(channelID int, usingKey string, count int, occurredAt time.Time) string {
	raw := fmt.Sprintf("new-api/channel-error/%d/%s/%d/%d", channelID, usingKey, count, occurredAt.UnixNano())
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(raw)).String()
}

func sendChannelConsecutiveErrorFeishuCard(payload channelConsecutiveErrorNotifyPayload) error {
	token, err := getFeishuTenantAccessToken(setting.ChannelConsecutiveErrorFeishuAppID, setting.ChannelConsecutiveErrorFeishuAppSecret)
	if err != nil {
		return err
	}

	cardContent, err := buildChannelConsecutiveErrorFeishuCardContent(payload)
	if err != nil {
		return err
	}

	messageReq := feishuMessageRequest{
		ReceiveID: setting.ChannelConsecutiveErrorFeishuChatID,
		MsgType:   "interactive",
		Content:   cardContent,
		UUID:      buildChannelConsecutiveErrorFeishuMessageUUID(payload.ChannelID, payload.UsingKey, payload.Count, payload.OccurredAt),
	}
	payloadBytes, err := common.Marshal(messageReq)
	if err != nil {
		return err
	}

	respBody, err := doFeishuRequest(http.MethodPost, feishuMessageCreateURL, payloadBytes, token)
	if err != nil {
		return err
	}

	var messageResp feishuMessageResponse
	if err = common.Unmarshal(respBody, &messageResp); err != nil {
		return err
	}
	if messageResp.Code != 0 {
		return fmt.Errorf("feishu send message failed: %s", messageResp.Msg)
	}
	return nil
}

func buildChannelConsecutiveErrorFeishuCardContent(payload channelConsecutiveErrorNotifyPayload) (string, error) {
	card := feishuCard{
		Config: feishuCardConfig{
			WideScreenMode: true,
		},
		Header: feishuCardHeader{
			Template: "red",
			Title: feishuCardTextBlock{
				Tag:     "plain_text",
				Content: "渠道连续错误告警",
			},
		},
		Elements: []feishuCardElement{
			{
				Tag: "div",
				Fields: []feishuCardField{
					newFeishuField("渠道", fmt.Sprintf("%s (#%d)", common.GetStringIfEmpty(payload.ChannelName, "-"), payload.ChannelID), true),
					newFeishuField("渠道类型", constant.GetChannelTypeName(payload.ChannelType), true),
					newFeishuField("连续错误次数", fmt.Sprintf("%d 次", payload.Count), true),
					newFeishuField("HTTP 状态码", fmt.Sprintf("%d", payload.StatusCode), true),
					newFeishuField("错误码", common.GetStringIfEmpty(payload.ErrorCode, "-"), true),
					newFeishuField("请求路径", common.GetStringIfEmpty(payload.RequestPath, "-"), true),
					newFeishuField("使用 Key", formatChannelErrorNotifyUsingKey(payload.UsingKey), false),
					newFeishuField("发生时间", payload.OccurredAt.Format("2006-01-02 15:04:05"), false),
				},
			},
			{
				Tag: "div",
				Text: &feishuCardTextBlock{
					Tag:     "lark_md",
					Content: fmt.Sprintf("**错误信息**\n%s", common.GetStringIfEmpty(payload.ErrorMsg, "-")),
				},
			},
		},
	}

	cardBytes, err := common.Marshal(card)
	if err != nil {
		return "", err
	}
	return string(cardBytes), nil
}

func formatChannelErrorNotifyUsingKey(usingKey string) string {
	usingKey = strings.TrimSpace(usingKey)
	if usingKey == "" {
		return "-"
	}
	return common.MaskSensitiveInfo(usingKey)
}
