package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildFeishuMessageUUIDDeterministic(t *testing.T) {
	tradeNo := "ALIPAYF2F_1_1776460482441323523"

	u1 := buildFeishuMessageUUID(tradeNo)
	u2 := buildFeishuMessageUUID(tradeNo)

	require.Equal(t, u1, u2)
	require.Len(t, u1, 36)
}

func TestTruncateFeishuErrorBody(t *testing.T) {
	msg := truncateFeishuErrorBody([]byte(`{"code":230099,"msg":"Bad Request","chat_id":"oc_xxx"}`))
	require.Contains(t, msg, "Bad Request")
	require.NotEmpty(t, msg)
}
