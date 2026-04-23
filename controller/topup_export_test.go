package controller

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestBuildTopUpOrdersXLSX(t *testing.T) {
	content, err := buildTopUpOrdersXLSX([]*model.AdminTopUpItem{
		{
			Id:                     1,
			UserId:                 2,
			Username:               "alice",
			TradeNo:                "ORD-1<&>",
			PaymentMethod:          model.PaymentMethodStripe,
			Amount:                 10,
			GrantedQuota:           500000,
			Money:                  10,
			Status:                 common.TopUpStatusSuccess,
			RefundStatus:           "none",
			RefundableAmount:       10,
			SuccessfulRefundAmount: 0,
			CreateTime:             1700000000,
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, content)

	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)

	var worksheet string
	for _, file := range reader.File {
		if file.Name != "xl/worksheets/sheet1.xml" {
			continue
		}
		rc, err := file.Open()
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		worksheet = string(data)
		break
	}

	require.Contains(t, worksheet, "订单ID")
	require.Contains(t, worksheet, "ORD-1&lt;&amp;&gt;")
	require.Contains(t, worksheet, "Stripe")
}
