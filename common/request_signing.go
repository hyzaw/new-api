package common

import (
	"strconv"
	"strings"
)

const (
	PublicRequestTimestampHeader      = "X-NewAPI-Timestamp"
	PublicRequestSignatureHeader      = "X-NewAPI-Signature"
	PublicRequestSigningWindowSeconds = int64(300)
)

func GetPublicRequestSigningKey() string {
	return GenerateHMAC("public-request-signing-key-v1")
}

func BuildPublicRequestSignaturePayload(method string, target string, timestamp int64, body string) string {
	return strings.ToUpper(method) + "\n" + target + "\n" + strconv.FormatInt(timestamp, 10) + "\n" + body
}

func SignPublicRequest(method string, target string, timestamp int64, body string) string {
	payload := BuildPublicRequestSignaturePayload(method, target, timestamp, body)
	return GenerateHMACWithKey([]byte(GetPublicRequestSigningKey()), payload)
}
