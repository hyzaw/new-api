package controller

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/require"
)

func TestBuildAlipaySignContentForRequest(t *testing.T) {
	params := map[string]string{
		"app_id":      "2021001",
		"method":      "alipay.trade.precreate",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   "2026-04-18 12:00:00",
		"version":     "1.0",
		"biz_content": `{"out_trade_no":"A1"}`,
		"sign":        "should_be_ignored",
		"empty":       "",
	}

	content := buildAlipaySignContent(params, false)
	require.Equal(t,
		`app_id=2021001&biz_content={"out_trade_no":"A1"}&charset=utf-8&method=alipay.trade.precreate&sign_type=RSA2&timestamp=2026-04-18 12:00:00&version=1.0`,
		content,
	)
}

func TestBuildAlipaySignContentForNotifyVerify(t *testing.T) {
	params := map[string]string{
		"notify_time":  "2026-04-18 12:00:00",
		"out_trade_no": "A1",
		"trade_status": "TRADE_SUCCESS",
		"sign_type":    "RSA2",
		"sign":         "should_be_ignored",
	}

	content := buildAlipaySignContent(params, true)
	require.Equal(t,
		`notify_time=2026-04-18 12:00:00&out_trade_no=A1&trade_status=TRADE_SUCCESS`,
		content,
	)
}

func TestGetAlipayVerifyHash(t *testing.T) {
	hashType, err := getAlipayVerifyHash("RSA2")
	require.NoError(t, err)
	require.Equal(t, "SHA-256", hashType.String())

	hashType, err = getAlipayVerifyHash("RSA")
	require.NoError(t, err)
	require.Equal(t, "SHA-1", hashType.String())
}

func TestCollectAlipayNotifyParamsUsesFirstValue(t *testing.T) {
	values := url.Values{
		"trade_status": []string{"TRADE_SUCCESS", "TRADE_FINISHED"},
		"out_trade_no": []string{"A1"},
	}

	params := collectAlipayNotifyParams(values)
	require.Equal(t, "TRADE_SUCCESS", params["trade_status"])
	require.Equal(t, "A1", params["out_trade_no"])
}

func TestParseAlipayNotifyPayloadPreservesRawValues(t *testing.T) {
	payload, err := parseAlipayNotifyPayload([]byte("notify_time=2026-04-18+12%3A00%3A00&body=%7B%22title%22%3A%22A%2BB+C%22%7D&sign=abc%2B123%3D"))
	require.NoError(t, err)

	require.Equal(t, "2026-04-18 12:00:00", payload.DecodedParams["notify_time"])
	require.Equal(t, "2026-04-18+12%3A00%3A00", payload.RawParams["notify_time"])
	require.Equal(t, `{"title":"A+B C"}`, payload.DecodedParams["body"])
	require.Equal(t, `%7B%22title%22%3A%22A%2BB+C%22%7D`, payload.RawParams["body"])
	require.Equal(t, "abc+123=", payload.DecodedParams["sign"])
	require.Equal(t, "abc%2B123%3D", payload.RawParams["sign"])
}

func TestVerifyAlipayNotifySignatureUsesRawEncodedValues(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pubASN1, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	oldPublicKey := setting.AlipayPublicKey
	setting.AlipayPublicKey = string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	}))
	defer func() {
		setting.AlipayPublicKey = oldPublicKey
	}()

	rawParams := map[string]string{
		"body":         `%7B%22title%22%3A%22A%2BB+C%22%7D`,
		"notify_time":  `2026-04-18+12%3A00%3A00`,
		"out_trade_no": `A1`,
		"trade_status": `TRADE_SUCCESS`,
		"sign_type":    `RSA2`,
	}
	signContent := buildAlipaySignContent(rawParams, true)
	hashed := sha256.Sum256([]byte(signContent))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	require.NoError(t, err)

	decodedParams := map[string]string{
		"body":         `{"title":"A+B C"}`,
		"notify_time":  `2026-04-18 12:00:00`,
		"out_trade_no": `A1`,
		"trade_status": `TRADE_SUCCESS`,
		"sign_type":    `RSA2`,
		"sign":         base64.StdEncoding.EncodeToString(signature),
	}

	require.NoError(t, verifyAlipaySignatureForContent(decodedParams, rawParams))
	require.Error(t, verifyAlipaySignature(decodedParams))
}
