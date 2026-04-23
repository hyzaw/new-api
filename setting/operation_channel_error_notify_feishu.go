package setting

import "strings"

var (
	ChannelConsecutiveErrorFeishuEnabled   = false
	ChannelConsecutiveErrorFeishuThreshold = 3
	ChannelConsecutiveErrorFeishuAppID     = ""
	ChannelConsecutiveErrorFeishuAppSecret = ""
	ChannelConsecutiveErrorFeishuChatID    = ""
)

func IsChannelConsecutiveErrorFeishuReady() bool {
	return ChannelConsecutiveErrorFeishuEnabled &&
		ChannelConsecutiveErrorFeishuThreshold > 0 &&
		strings.TrimSpace(ChannelConsecutiveErrorFeishuAppID) != "" &&
		strings.TrimSpace(ChannelConsecutiveErrorFeishuAppSecret) != "" &&
		strings.TrimSpace(ChannelConsecutiveErrorFeishuChatID) != ""
}
