package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOaiResponsesStreamHandlerStopsAndReturnsImmediateTopLevelError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldStreamingTimeout
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(
			"event: \n" +
				"data: {\"error\":{\"type\":\"rate_limit_error\",\"message\":\"Concurrency limit exceeded for account, please retry later\"}}\n\n",
		)),
	}

	info := &relaycommon.RelayInfo{}
	usage, err := OaiResponsesStreamHandler(c, info, resp)
	require.Nil(t, usage)
	require.Error(t, err)
	require.Equal(t, http.StatusTooManyRequests, err.StatusCode)
	require.Contains(t, err.Error(), "Concurrency limit exceeded")
	require.Equal(t, "", w.Body.String())
}
