package controller

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const paymentMethodAlipayF2F = "alipay_f2f"

type AlipayF2FPayRequest struct {
	Amount int64 `json:"amount"`
}

type alipayTradePrecreateRequest struct {
	OutTradeNo     string            `json:"out_trade_no"`
	TotalAmount    string            `json:"total_amount"`
	Subject        string            `json:"subject"`
	ProductCode    string            `json:"product_code,omitempty"`
	SellerID       string            `json:"seller_id,omitempty"`
	Body           string            `json:"body,omitempty"`
	BusinessParams map[string]string `json:"business_params,omitempty"`
	StoreID        string            `json:"store_id,omitempty"`
	OperatorID     string            `json:"operator_id,omitempty"`
	TerminalID     string            `json:"terminal_id,omitempty"`
}

type alipayTradeQueryRequest struct {
	OutTradeNo string `json:"out_trade_no"`
}

type alipayTradePrecreateResponse struct {
	Code       string `json:"code"`
	Msg        string `json:"msg"`
	SubCode    string `json:"sub_code"`
	SubMsg     string `json:"sub_msg"`
	OutTradeNo string `json:"out_trade_no"`
	QRCode     string `json:"qr_code"`
}

type alipayTradeQueryResponse struct {
	Code        string `json:"code"`
	Msg         string `json:"msg"`
	SubCode     string `json:"sub_code"`
	SubMsg      string `json:"sub_msg"`
	OutTradeNo  string `json:"out_trade_no"`
	TradeNo     string `json:"trade_no"`
	TradeStatus string `json:"trade_status"`
	TotalAmount string `json:"total_amount"`
}

func isAlipayF2FEnabled() bool {
	return setting.AlipayF2FEnabled &&
		setting.AlipayAppID != "" &&
		setting.AlipayPrivateKey != "" &&
		setting.AlipayPublicKey != ""
}

func normalizeAlipayKey(raw string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(raw, "\r", ""), "\n", ""))
}

func parseAlipayPrivateKey(raw string) (*rsa.PrivateKey, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("支付宝私钥未配置")
	}

	if block, _ := pem.Decode([]byte(trimmed)); block != nil {
		if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
			rsaKey, ok := key.(*rsa.PrivateKey)
			if !ok {
				return nil, errors.New("支付宝私钥不是 RSA 私钥")
			}
			return rsaKey, nil
		}
		if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
			return key, nil
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(normalizeAlipayKey(trimmed))
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKCS8PrivateKey(decoded); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("支付宝私钥不是 RSA 私钥")
		}
		return rsaKey, nil
	}
	return x509.ParsePKCS1PrivateKey(decoded)
}

func parseAlipayPublicKey(raw string) (*rsa.PublicKey, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("支付宝公钥未配置")
	}

	if block, _ := pem.Decode([]byte(trimmed)); block != nil {
		if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
			rsaPub, ok := pub.(*rsa.PublicKey)
			if !ok {
				return nil, errors.New("支付宝公钥不是 RSA 公钥")
			}
			return rsaPub, nil
		}
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			rsaPub, ok := cert.PublicKey.(*rsa.PublicKey)
			if !ok {
				return nil, errors.New("支付宝证书不是 RSA 公钥")
			}
			return rsaPub, nil
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(normalizeAlipayKey(trimmed))
	if err != nil {
		return nil, err
	}
	if pub, err := x509.ParsePKIXPublicKey(decoded); err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("支付宝公钥不是 RSA 公钥")
		}
		return rsaPub, nil
	}
	if cert, err := x509.ParseCertificate(decoded); err == nil {
		rsaPub, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("支付宝证书不是 RSA 公钥")
		}
		return rsaPub, nil
	}
	return nil, errors.New("无法解析支付宝公钥")
}

func buildAlipaySignContent(params map[string]string, excludeSignType bool) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if value == "" || key == "sign" || (excludeSignType && key == "sign_type") {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	return strings.Join(parts, "&")
}

func alipaySign(signContent string) (string, error) {
	privateKey, err := parseAlipayPrivateKey(setting.AlipayPrivateKey)
	if err != nil {
		return "", err
	}
	hashed := sha256.Sum256([]byte(signContent))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func getAlipayVerifyHash(signType string) (crypto.Hash, error) {
	switch strings.ToUpper(strings.TrimSpace(signType)) {
	case "", "RSA2":
		return crypto.SHA256, nil
	case "RSA":
		return crypto.SHA1, nil
	default:
		return 0, fmt.Errorf("unsupported alipay sign_type: %s", signType)
	}
}

func verifyAlipaySignatureWithContent(publicKey *rsa.PublicKey, signature string, signType string, signContent string) error {
	signBytes, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(signature, " ", "+"))
	if err != nil {
		return err
	}

	hashType, err := getAlipayVerifyHash(signType)
	if err != nil {
		return err
	}

	var digest []byte
	switch hashType {
	case crypto.SHA256:
		hashed := sha256.Sum256([]byte(signContent))
		digest = hashed[:]
	case crypto.SHA1:
		hashed := sha1.Sum([]byte(signContent))
		digest = hashed[:]
	default:
		return fmt.Errorf("unsupported alipay verify hash: %v", hashType)
	}

	return rsa.VerifyPKCS1v15(publicKey, hashType, digest, signBytes)
}

func verifyAlipaySignature(params map[string]string) error {
	signature := params["sign"]
	if signature == "" {
		return errors.New("支付宝通知缺少签名")
	}
	publicKey, err := parseAlipayPublicKey(setting.AlipayPublicKey)
	if err != nil {
		return err
	}

	signType := params["sign_type"]
	signContents := []struct {
		name    string
		content string
	}{
		{name: "exclude_sign_type", content: buildAlipaySignContent(params, true)},
	}
	if signType != "" {
		signContents = append(signContents, struct {
			name    string
			content string
		}{
			name:    "include_sign_type",
			content: buildAlipaySignContent(params, false),
		})
	}

	errs := make([]string, 0, len(signContents))
	for _, candidate := range signContents {
		if err := verifyAlipaySignatureWithContent(publicKey, signature, signType, candidate.content); err == nil {
			return nil
		} else {
			errs = append(errs, fmt.Sprintf("%s:%v", candidate.name, err))
		}
	}

	return fmt.Errorf("alipay signature verification failed (sign_type=%s): %s", common.GetStringIfEmpty(signType, "RSA2"), strings.Join(errs, "; "))
}

func doAlipayGatewayRequest(method string, bizContent any, notifyURL string) ([]byte, error) {
	bizBytes, err := common.Marshal(bizContent)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"app_id":      setting.AlipayAppID,
		"method":      method,
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": string(bizBytes),
	}
	if notifyURL != "" {
		params["notify_url"] = notifyURL
	}
	if setting.AlipayAppAuthToken != "" {
		params["app_auth_token"] = setting.AlipayAppAuthToken
	}

	params["sign"], err = alipaySign(buildAlipaySignContent(params, false))
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	for key, value := range params {
		if value == "" {
			continue
		}
		form.Set(key, value)
	}

	req, err := http.NewRequest(http.MethodPost, setting.GetAlipayGateway(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("支付宝网关返回 HTTP %d", resp.StatusCode)
	}
	return body, nil
}

func parseAlipayMethodResponse(body []byte, responseKey string, out any) error {
	var envelope map[string]any
	if err := common.Unmarshal(body, &envelope); err != nil {
		return err
	}
	rawResponse, ok := envelope[responseKey]
	if !ok {
		return errors.New("支付宝响应缺少业务字段")
	}
	responseBytes, err := common.Marshal(rawResponse)
	if err != nil {
		return err
	}
	return common.Unmarshal(responseBytes, out)
}

func getAlipayNotifyURL() string {
	if setting.AlipayNotifyURL != "" {
		return setting.AlipayNotifyURL
	}
	return service.GetCallbackAddress() + "/api/alipay/f2f/notify"
}

func createAlipayF2FOrder(id int, amount int64, payMoney float64) (*model.TopUp, error) {
	tradeNo := fmt.Sprintf("ALIPAYF2F_%d_%d", id, time.Now().UnixNano())
	orderAmount := amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		orderAmount = dAmount.Div(dQuotaPerUnit).IntPart()
	}

	topUp := &model.TopUp{
		UserId:        id,
		Amount:        orderAmount,
		Money:         payMoney,
		TradeNo:       tradeNo,
		PaymentMethod: paymentMethodAlipayF2F,
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	return topUp, topUp.Insert()
}

func RequestAlipayF2FPay(c *gin.Context) {
	if !isAlipayF2FEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付宝当面付未启用"})
		return
	}

	var req AlipayF2FPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if errMsg := validateTopupAmount(req.Amount, getMinTopup()); errMsg != "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": errMsg})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	topUp, err := createAlipayF2FOrder(id, req.Amount, payMoney)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	subject := fmt.Sprintf("账户充值 %d", req.Amount)
	precreateReq := alipayTradePrecreateRequest{
		OutTradeNo:  topUp.TradeNo,
		TotalAmount: strconv.FormatFloat(payMoney, 'f', 2, 64),
		Subject:     subject,
		ProductCode: common.GetStringIfEmpty(setting.AlipayProductCode, setting.AlipayDefaultProductCode),
		SellerID:    setting.AlipaySellerID,
		Body:        subject,
		BusinessParams: map[string]string{
			"mc_create_trade_ip": c.ClientIP(),
		},
	}

	body, err := doAlipayGatewayRequest("alipay.trade.precreate", precreateReq, getAlipayNotifyURL())
	if err != nil {
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		log.Printf("支付宝预下单失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付宝支付失败"})
		return
	}

	var precreateResp alipayTradePrecreateResponse
	if err := parseAlipayMethodResponse(body, "alipay_trade_precreate_response", &precreateResp); err != nil {
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		log.Printf("解析支付宝预下单响应失败: %v, body=%s", err, common.MaskSensitiveInfo(string(body)))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付宝响应解析失败"})
		return
	}
	if precreateResp.Code != "10000" || precreateResp.QRCode == "" {
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		errMsg := precreateResp.SubMsg
		if errMsg == "" {
			errMsg = precreateResp.Msg
		}
		if errMsg == "" {
			errMsg = "支付宝预下单失败"
		}
		log.Printf("支付宝预下单业务失败: tradeNo=%s code=%s subCode=%s msg=%s subMsg=%s",
			topUp.TradeNo, precreateResp.Code, precreateResp.SubCode, precreateResp.Msg, precreateResp.SubMsg)
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"trade_no":   topUp.TradeNo,
			"qr_code":    precreateResp.QRCode,
			"expires_in": 7200,
			"amount":     strconv.FormatFloat(topUp.Money, 'f', 2, 64),
		},
	})
}

func queryAlipayTrade(outTradeNo string) (*alipayTradeQueryResponse, error) {
	queryReq := alipayTradeQueryRequest{OutTradeNo: outTradeNo}
	body, err := doAlipayGatewayRequest("alipay.trade.query", queryReq, "")
	if err != nil {
		return nil, err
	}
	var queryResp alipayTradeQueryResponse
	if err := parseAlipayMethodResponse(body, "alipay_trade_query_response", &queryResp); err != nil {
		return nil, err
	}
	return &queryResp, nil
}

func markAlipayTopUpFailedIfPending(tradeNo string) {
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil || topUp.PaymentMethod != paymentMethodAlipayF2F || topUp.Status != common.TopUpStatusPending {
		return
	}
	topUp.Status = common.TopUpStatusFailed
	topUp.CompleteTime = common.GetTimestamp()
	_ = topUp.Update()
}

func syncAlipayTradeStatus(tradeNo string, callerIP string) (string, bool, error) {
	queryResp, err := queryAlipayTrade(tradeNo)
	if err != nil {
		return "", false, err
	}
	if queryResp.Code != "10000" {
		errMsg := queryResp.SubMsg
		if errMsg == "" {
			errMsg = queryResp.Msg
		}
		if errMsg == "" {
			errMsg = "支付宝查询失败"
		}
		return queryResp.TradeStatus, false, errors.New(errMsg)
	}

	completed := false
	switch queryResp.TradeStatus {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		completed, err = model.RechargeAlipayF2F(tradeNo, callerIP)
		if err != nil {
			return queryResp.TradeStatus, false, err
		}
		if completed {
			service.NotifyTopupSuccessAsync(tradeNo, callerIP, "alipay_f2f")
		}
	case "TRADE_CLOSED":
		markAlipayTopUpFailedIfPending(tradeNo)
	}
	return queryResp.TradeStatus, completed, nil
}

func AlipayF2FStatus(c *gin.Context) {
	tradeNo := c.Query("trade_no")
	if tradeNo == "" {
		common.ApiErrorMsg(c, "订单号不能为空")
		return
	}

	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil || topUp.UserId != c.GetInt("id") || topUp.PaymentMethod != paymentMethodAlipayF2F {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}

	tradeStatus := ""
	if topUp.Status == common.TopUpStatusPending {
		LockOrder(tradeNo)
		tradeStatus, _, _ = syncAlipayTradeStatus(tradeNo, c.ClientIP())
		UnlockOrder(tradeNo)
		topUp = model.GetTopUpByTradeNo(tradeNo)
	}

	common.ApiSuccess(c, gin.H{
		"trade_no":     tradeNo,
		"status":       topUp.Status,
		"trade_status": tradeStatus,
		"paid":         topUp.Status == common.TopUpStatusSuccess,
	})
}

func collectAlipayNotifyParams(values url.Values) map[string]string {
	params := make(map[string]string, len(values))
	for key, items := range values {
		if len(items) == 0 {
			params[key] = ""
			continue
		}
		params[key] = items[0]
	}
	return params
}

func AlipayF2FNotify(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	params := collectAlipayNotifyParams(c.Request.PostForm)
	if len(params) == 0 {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	if err := verifyAlipaySignature(params); err != nil {
		log.Printf("支付宝回调验签失败: %v", err)
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	outTradeNo := params["out_trade_no"]
	if outTradeNo == "" {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	LockOrder(outTradeNo)
	defer UnlockOrder(outTradeNo)

	tradeStatus, _, err := syncAlipayTradeStatus(outTradeNo, c.ClientIP())
	if err != nil {
		log.Printf("支付宝回调同步订单状态失败: tradeNo=%s status=%s err=%v", outTradeNo, tradeStatus, err)
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	_, _ = c.Writer.Write([]byte("success"))
}
