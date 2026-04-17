package setting

var (
	TopupNotifyFeishuEnabled   = false
	TopupNotifyFeishuAppID     = ""
	TopupNotifyFeishuAppSecret = ""
	TopupNotifyFeishuChatID    = ""
)

func IsTopupNotifyFeishuReady() bool {
	return TopupNotifyFeishuEnabled &&
		TopupNotifyFeishuAppID != "" &&
		TopupNotifyFeishuAppSecret != "" &&
		TopupNotifyFeishuChatID != ""
}
