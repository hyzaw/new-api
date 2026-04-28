package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestForceGPTImageRelayConvertsChatRequest(t *testing.T) {
	n := 2
	req := &dto.GeneralOpenAIRequest{
		Model: "gpt-image-1",
		Messages: []dto.Message{
			{Role: "user", Content: "draw a neon city"},
		},
		N:              &n,
		Size:           "1024x1024",
		ResponseFormat: &dto.ResponseFormat{Type: "b64_json"},
		User:           []byte(`"user-1"`),
	}

	format, forcedReq, forced := forceGPTImageRelay(types.RelayFormatOpenAI, req)
	require.True(t, forced)
	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIImage), format)

	imageReq, ok := forcedReq.(*dto.ImageRequest)
	require.True(t, ok)
	require.Equal(t, "gpt-image-1", imageReq.Model)
	require.Equal(t, "draw a neon city", imageReq.Prompt)
	require.Equal(t, "1024x1024", imageReq.Size)
	require.Equal(t, "b64_json", imageReq.ResponseFormat)
	require.NotNil(t, imageReq.N)
	require.Equal(t, uint(2), *imageReq.N)
}

func TestForceGPTImageRelayLeavesTextModelsAlone(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{Model: "gpt-4o-mini"}

	format, forcedReq, forced := forceGPTImageRelay(types.RelayFormatOpenAI, req)
	require.False(t, forced)
	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAI), format)
	require.Same(t, req, forcedReq)
}

func TestImagePromptFromResponsesRequest(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{
		Model:        "gpt-image-1",
		Instructions: []byte(`"high detail"`),
		Input:        []byte(`"draw a red balloon"`),
		Prompt:       []byte(`"square composition"`),
		User:         []byte(`"user-1"`),
	}

	imageReq := imageRequestFromResponsesRequest(req)
	require.Equal(t, "gpt-image-1", imageReq.Model)
	require.Equal(t, "high detail\nsquare composition\ndraw a red balloon", imageReq.Prompt)
	require.JSONEq(t, `"user-1"`, string(imageReq.User))
}

func TestImagePromptFromOpenAIRequestIncludesPromptAndMessages(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Model:  "gpt-image-1",
		Prompt: "watercolor",
		Messages: []dto.Message{
			{
				Role: "user",
				Content: []any{
					map[string]any{"type": dto.ContentTypeText, "text": "a quiet library"},
				},
			},
		},
	}

	require.Equal(t, "watercolor\na quiet library", imagePromptFromOpenAIRequest(req))
}

func TestRawJSONStringIgnoresNonStringJSON(t *testing.T) {
	require.Equal(t, "", rawJSONString([]byte(`{"type":"json_object"}`)))
}
