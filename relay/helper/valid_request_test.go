package helper

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetAndValidateRequestForcesGPTImagePromptToImageRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := []byte(`{"model":"gpt-image-1","prompt":"draw a city","size":"1024x1024","response_format":"b64_json"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	request, err := GetAndValidateRequest(c, types.RelayFormatOpenAI)
	require.NoError(t, err)
	require.True(t, c.GetBool("force_image_relay"))

	imageReq, ok := request.(*dto.ImageRequest)
	require.True(t, ok)
	require.Equal(t, "gpt-image-1", imageReq.Model)
	require.Equal(t, "draw a city", imageReq.Prompt)
	require.Equal(t, "1024x1024", imageReq.Size)
	require.Equal(t, "b64_json", imageReq.ResponseFormat)
}

func TestGetAndValidateResponsesRequestForcesGPTImagePromptToImageRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := []byte(`{"model":"gpt-image-1","prompt":"draw a city"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	request, err := GetAndValidateRequest(c, types.RelayFormatOpenAIResponses)
	require.NoError(t, err)
	require.True(t, c.GetBool("force_image_relay"))

	imageReq, ok := request.(*dto.ImageRequest)
	require.True(t, ok)
	require.Equal(t, "gpt-image-1", imageReq.Model)
	require.Equal(t, "draw a city", imageReq.Prompt)
}
