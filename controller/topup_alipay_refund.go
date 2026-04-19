package controller

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/model"

	"github.com/shopspring/decimal"
)

type alipayTradeRefundRequest struct {
	OutTradeNo   string   `json:"out_trade_no,omitempty"`
	TradeNo      string   `json:"trade_no,omitempty"`
	RefundAmount string   `json:"refund_amount"`
	RefundReason string   `json:"refund_reason,omitempty"`
	OutRequestNo string   `json:"out_request_no,omitempty"`
	QueryOptions []string `json:"query_options,omitempty"`
}

type alipayTradeRefundResponse struct {
	Code         string `json:"code"`
	Msg          string `json:"msg"`
	SubCode      string `json:"sub_code"`
	SubMsg       string `json:"sub_msg"`
	TradeNo      string `json:"trade_no"`
	OutTradeNo   string `json:"out_trade_no"`
	BuyerLogonID string `json:"buyer_logon_id"`
	RefundFee    string `json:"refund_fee"`
	SendBackFee  string `json:"send_back_fee"`
	FundChange   string `json:"fund_change"`
}

func refundAlipayTrade(refund *model.TopUpRefund) (*alipayTradeRefundResponse, error) {
	if refund == nil {
		return nil, errors.New("退款记录不存在")
	}

	amount := decimal.NewFromFloat(refund.RefundAmount).Round(2)
	if !amount.IsPositive() {
		return nil, errors.New("退款金额无效")
	}

	refundReq := alipayTradeRefundRequest{
		OutTradeNo:   refund.TradeNo,
		RefundAmount: amount.StringFixed(2),
		RefundReason: strings.TrimSpace(refund.RefundReason),
		OutRequestNo: refund.RefundNo,
		QueryOptions: []string{"refund_detail_item_list"},
	}

	body, err := doAlipayGatewayRequest("alipay.trade.refund", refundReq, "")
	if err != nil {
		return nil, err
	}

	var refundResp alipayTradeRefundResponse
	if err := parseAlipayMethodResponse(body, "alipay_trade_refund_response", &refundResp); err != nil {
		return nil, err
	}
	return &refundResp, nil
}
