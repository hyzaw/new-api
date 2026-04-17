package setting

const (
	AlipayGatewayProduction  = "https://openapi.alipay.com/gateway.do"
	AlipayGatewaySandbox     = "https://openapi.alipaydev.com/gateway.do"
	AlipayDefaultProductCode = "FACE_TO_FACE_PAYMENT"
)

var (
	AlipayF2FEnabled   = false
	AlipaySandbox      = false
	AlipayAppID        = ""
	AlipayPrivateKey   = ""
	AlipayPublicKey    = ""
	AlipayAppAuthToken = ""
	AlipaySellerID     = ""
	AlipayNotifyURL    = ""
	AlipayProductCode  = AlipayDefaultProductCode
)

func GetAlipayGateway() string {
	if AlipaySandbox {
		return AlipayGatewaySandbox
	}
	return AlipayGatewayProduction
}
