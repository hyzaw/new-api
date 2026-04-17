package controller

import (
	"testing"

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
