package openai

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
)

func getResponsesStreamOpenAIError(streamResp dto.ResponsesStreamResponse) (*types.OpenAIError, int, bool) {
	oaiErr := streamResp.GetOpenAIError()
	if oaiErr == nil || (oaiErr.Type == "" && oaiErr.Message == "" && oaiErr.Code == nil) {
		return nil, 0, false
	}

	switch streamResp.Type {
	case "error", "response.error", "response.failed":
		return oaiErr, inferResponsesStreamErrorStatusCode(oaiErr), true
	}

	// Some upstreams return `data: {"error": ...}` without a stream event type.
	if streamResp.Type == "" && streamResp.Error != nil && streamResp.Response == nil {
		return oaiErr, inferResponsesStreamErrorStatusCode(oaiErr), true
	}

	return nil, 0, false
}

func inferResponsesStreamErrorStatusCode(oaiErr *types.OpenAIError) int {
	if oaiErr == nil {
		return http.StatusInternalServerError
	}

	switch code := oaiErr.Code.(type) {
	case float64:
		if code >= 100 && code <= 599 {
			return int(code)
		}
	case int:
		if code >= 100 && code <= 599 {
			return code
		}
	case string:
		if code == "429" {
			return http.StatusTooManyRequests
		}
	}

	errType := strings.ToLower(strings.TrimSpace(oaiErr.Type))
	errMsg := strings.ToLower(strings.TrimSpace(oaiErr.Message))

	if errType == "rate_limit_error" ||
		strings.Contains(errMsg, "concurrency limit exceeded") ||
		strings.Contains(errMsg, "retry later") ||
		strings.Contains(errMsg, "rate limit") {
		return http.StatusTooManyRequests
	}

	return http.StatusInternalServerError
}
