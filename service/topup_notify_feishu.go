package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	feishuTenantAccessTokenURL = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
	feishuMessageCreateURL     = "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id"
)

type feishuTenantAccessTokenRequest struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

type feishuTenantAccessTokenResponse struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
}

type feishuMessageRequest struct {
	ReceiveID string `json:"receive_id"`
	MsgType   string `json:"msg_type"`
	Content   string `json:"content"`
	UUID      string `json:"uuid"`
}

type feishuMessageResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func buildFeishuMessageUUID(tradeNo string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("new-api/topup/"+tradeNo)).String()
}

type feishuCard struct {
	Config   feishuCardConfig    `json:"config,omitempty"`
	Header   feishuCardHeader    `json:"header"`
	Elements []feishuCardElement `json:"elements"`
}

type feishuCardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
}

type feishuCardHeader struct {
	Template string              `json:"template,omitempty"`
	Title    feishuCardTextBlock `json:"title"`
}

type feishuCardElement struct {
	Tag      string                `json:"tag"`
	Text     *feishuCardTextBlock  `json:"text,omitempty"`
	Fields   []feishuCardField     `json:"fields,omitempty"`
	Elements []feishuCardTextBlock `json:"elements,omitempty"`
}

type feishuCardField struct {
	IsShort bool                `json:"is_short"`
	Text    feishuCardTextBlock `json:"text"`
}

type feishuCardTextBlock struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

func NotifyTopupSuccessAsync(tradeNo string, callerIP string, callbackSource string) {
	if !setting.IsTopupNotifyFeishuReady() || tradeNo == "" {
		return
	}
	gopool.Go(func() {
		if err := notifyTopupSuccessToFeishu(tradeNo, callerIP, callbackSource); err != nil {
			common.SysLog(fmt.Sprintf("failed to send feishu topup notification for %s: %s", tradeNo, err.Error()))
		}
	})
}

func notifyTopupSuccessToFeishu(tradeNo string, callerIP string, callbackSource string) error {
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		return fmt.Errorf("topup order not found")
	}
	if topUp.Status != common.TopUpStatusSuccess {
		return fmt.Errorf("topup order status is %s", topUp.Status)
	}

	user, err := model.GetUserById(topUp.UserId, false)
	if err != nil {
		return err
	}

	token, err := getFeishuTenantAccessToken()
	if err != nil {
		return err
	}

	cardContent, err := buildTopupFeishuCardContent(topUp, user, callerIP, callbackSource)
	if err != nil {
		return err
	}

	messageReq := feishuMessageRequest{
		ReceiveID: setting.TopupNotifyFeishuChatID,
		MsgType:   "interactive",
		Content:   cardContent,
		UUID:      buildFeishuMessageUUID(tradeNo),
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
	if err := common.Unmarshal(respBody, &messageResp); err != nil {
		return err
	}
	if messageResp.Code != 0 {
		return fmt.Errorf("feishu send message failed: %s", messageResp.Msg)
	}
	return nil
}

func getFeishuTenantAccessToken() (string, error) {
	reqBody := feishuTenantAccessTokenRequest{
		AppID:     setting.TopupNotifyFeishuAppID,
		AppSecret: setting.TopupNotifyFeishuAppSecret,
	}
	payloadBytes, err := common.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	respBody, err := doFeishuRequest(http.MethodPost, feishuTenantAccessTokenURL, payloadBytes, "")
	if err != nil {
		return "", err
	}

	var tokenResp feishuTenantAccessTokenResponse
	if err := common.Unmarshal(respBody, &tokenResp); err != nil {
		return "", err
	}
	if tokenResp.Code != 0 {
		return "", fmt.Errorf("feishu tenant_access_token failed: %s", tokenResp.Msg)
	}
	if tokenResp.TenantAccessToken == "" {
		return "", fmt.Errorf("feishu tenant_access_token missing")
	}
	return tokenResp.TenantAccessToken, nil
}

func doFeishuRequest(method string, requestURL string, payload []byte, bearerToken string) ([]byte, error) {
	if systemWorkerEnabledForHTTPS() {
		headers := map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		}
		if bearerToken != "" {
			headers["Authorization"] = "Bearer " + bearerToken
		}
		resp, err := DoWorkerRequest(&WorkerRequest{
			URL:     requestURL,
			Key:     system_setting.WorkerValidKey,
			Method:  method,
			Headers: headers,
			Body:    json.RawMessage(payload),
		})
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("feishu request failed with status %d: %s", resp.StatusCode, truncateFeishuErrorBody(body))
		}
		return body, nil
	}

	req, err := http.NewRequest(method, requestURL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("feishu request failed with status %d: %s", resp.StatusCode, truncateFeishuErrorBody(body))
	}
	return body, nil
}

func systemWorkerEnabledForHTTPS() bool {
	return system_setting.EnableWorker()
}

func truncateFeishuErrorBody(body []byte) string {
	msg := strings.TrimSpace(string(body))
	msg = common.MaskSensitiveInfo(msg)
	if msg == "" {
		return "<empty>"
	}
	if len(msg) > 512 {
		return msg[:512] + "..."
	}
	return msg
}

func buildTopupFeishuCardContent(topUp *model.TopUp, user *model.User, callerIP string, callbackSource string) (string, error) {
	completedAt := time.Now()
	if topUp.CompleteTime > 0 {
		completedAt = time.Unix(topUp.CompleteTime, 0)
	}

	card := feishuCard{
		Config: feishuCardConfig{
			WideScreenMode: true,
		},
		Header: feishuCardHeader{
			Template: "green",
			Title: feishuCardTextBlock{
				Tag:     "plain_text",
				Content: "用户充值成功通知",
			},
		},
		Elements: []feishuCardElement{
			{
				Tag: "div",
				Fields: []feishuCardField{
					newFeishuField("用户", fmt.Sprintf("%s (#%d)", user.Username, user.Id), true),
					newFeishuField("到账额度", logger.FormatQuota(calculateTopupQuota(topUp)), true),
					newFeishuField("支付方式", formatTopupMethod(topUp.PaymentMethod), true),
					newFeishuField("支付金额", fmt.Sprintf("%.2f", topUp.Money), true),
					newFeishuField("订单号", topUp.TradeNo, false),
					newFeishuField("完成时间", completedAt.Format("2006-01-02 15:04:05"), false),
				},
			},
			{
				Tag: "note",
				Elements: []feishuCardTextBlock{
					{
						Tag:     "plain_text",
						Content: fmt.Sprintf("回调来源: %s", formatCallbackSource(callbackSource)),
					},
					{
						Tag:     "plain_text",
						Content: fmt.Sprintf("回调IP: %s", common.GetStringIfEmpty(callerIP, "-")),
					},
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

func newFeishuField(title string, value string, isShort bool) feishuCardField {
	return feishuCardField{
		IsShort: isShort,
		Text: feishuCardTextBlock{
			Tag:     "lark_md",
			Content: fmt.Sprintf("**%s**\n%s", title, value),
		},
	}
}

func calculateTopupQuota(topUp *model.TopUp) int {
	if topUp == nil {
		return 0
	}
	switch topUp.PaymentMethod {
	case "stripe":
		return int(decimal.NewFromFloat(topUp.Money).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	case "creem":
		return int(topUp.Amount)
	default:
		return int(decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	}
}

func formatTopupMethod(method string) string {
	switch method {
	case "stripe":
		return "Stripe"
	case "creem":
		return "Creem"
	case "waffo":
		return "Waffo"
	case "alipay_f2f":
		return "支付宝当面付"
	case "alipay":
		return "支付宝"
	case "wxpay":
		return "微信支付"
	default:
		if method == "" {
			return "-"
		}
		return method
	}
}

func formatCallbackSource(source string) string {
	switch source {
	case "stripe":
		return "Stripe Webhook"
	case "creem":
		return "Creem Webhook"
	case "waffo":
		return "Waffo Webhook"
	case "alipay_f2f":
		return "支付宝回调"
	case "epay":
		return "易支付回调"
	case "admin_manual":
		return "管理员补单"
	default:
		if source == "" {
			return "-"
		}
		return source
	}
}
