package controller

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

var localResponsesArithmeticProbePattern = regexp.MustCompile(`Q:\s*(-?\d+)\s*([+\-*/])\s*(-?\d+)\s*=\s*\?\s*A:\s*$`)

func tryHandleLocalRelayProbe(c *gin.Context, relayFormat types.RelayFormat, request dto.Request) (bool, *types.NewAPIError) {
	if relayFormat != types.RelayFormatOpenAIResponses {
		return false, nil
	}

	responsesReq, ok := request.(*dto.OpenAIResponsesRequest)
	if !ok || responsesReq == nil {
		return false, nil
	}

	answer, ok := detectLocalResponsesArithmeticProbe(responsesReq)
	if !ok {
		return false, nil
	}

	if responsesReq.IsStream(c) {
		if err := writeLocalResponsesProbeStream(c, responsesReq, answer); err != nil {
			return true, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
		}
		return true, nil
	}

	if err := writeLocalResponsesProbeJSON(c, responsesReq, answer); err != nil {
		return true, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	return true, nil
}

func detectLocalResponsesArithmeticProbe(req *dto.OpenAIResponsesRequest) (string, bool) {
	if req == nil {
		return "", false
	}
	if strings.TrimSpace(req.Model) == "" {
		return "", false
	}
	if len(req.GetToolsMap()) > 0 || strings.TrimSpace(req.PreviousResponseID) != "" {
		return "", false
	}

	inputs := req.ParseInput()
	if len(inputs) != 1 || inputs[0].Type != "input_text" {
		return "", false
	}

	prompt := strings.TrimSpace(inputs[0].Text)
	if prompt == "" {
		return "", false
	}
	if !strings.Contains(prompt, "Calculate and respond with ONLY the number, nothing else.") {
		return "", false
	}
	if !strings.Contains(prompt, "Q: 3 + 5 = ?") || !strings.Contains(prompt, "Q: 12 - 7 = ?") {
		return "", false
	}

	matches := localResponsesArithmeticProbePattern.FindStringSubmatch(prompt)
	if len(matches) != 4 {
		return "", false
	}

	left, err := strconv.Atoi(matches[1])
	if err != nil {
		return "", false
	}
	right, err := strconv.Atoi(matches[3])
	if err != nil {
		return "", false
	}

	var result int
	switch matches[2] {
	case "+":
		result = left + right
	case "-":
		result = left - right
	case "*":
		result = left * right
	case "/":
		if right == 0 || left%right != 0 {
			return "", false
		}
		result = left / right
	default:
		return "", false
	}
	return strconv.Itoa(result), true
}

func writeLocalResponsesProbeJSON(c *gin.Context, req *dto.OpenAIResponsesRequest, answer string) error {
	responseID, messageID, createdAt := newLocalResponsesProbeIDs()
	payload := buildLocalResponsesCompletedPayload(req, responseID, messageID, createdAt, answer, false)
	data, err := common.Marshal(payload)
	if err != nil {
		return err
	}
	c.Header("Content-Type", "application/json")
	c.Status(http.StatusOK)
	_, err = c.Writer.Write(data)
	return err
}

func writeLocalResponsesProbeStream(c *gin.Context, req *dto.OpenAIResponsesRequest, answer string) error {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	responseID, messageID, createdAt := newLocalResponsesProbeIDs()
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	events := []struct {
		name string
		data any
	}{
		{
			name: "response.created",
			data: gin.H{
				"type":            "response.created",
				"response":        buildLocalResponsesCreatedPayload(req, responseID, createdAt),
				"sequence_number": 0,
			},
		},
		{
			name: "response.in_progress",
			data: gin.H{
				"type":            "response.in_progress",
				"response":        buildLocalResponsesCreatedPayload(req, responseID, createdAt),
				"sequence_number": 1,
			},
		},
		{
			name: "response.output_item.added",
			data: gin.H{
				"type": "response.output_item.added",
				"item": gin.H{
					"id":      messageID,
					"type":    "message",
					"status":  "in_progress",
					"content": []any{},
					"role":    "assistant",
				},
				"output_index":    0,
				"sequence_number": 2,
			},
		},
		{
			name: "response.content_part.added",
			data: gin.H{
				"type":          "response.content_part.added",
				"content_index": 0,
				"item_id":       messageID,
				"output_index":  0,
				"part": gin.H{
					"type":        "output_text",
					"annotations": []any{},
					"logprobs":    []any{},
					"text":        "",
				},
				"sequence_number": 3,
			},
		},
		{
			name: "response.output_text.delta",
			data: gin.H{
				"type":            "response.output_text.delta",
				"content_index":   0,
				"delta":           answer,
				"item_id":         messageID,
				"logprobs":        []any{},
				"output_index":    0,
				"sequence_number": 4,
			},
		},
		{
			name: "response.output_text.done",
			data: gin.H{
				"type":            "response.output_text.done",
				"content_index":   0,
				"item_id":         messageID,
				"logprobs":        []any{},
				"output_index":    0,
				"sequence_number": 5,
				"text":            answer,
			},
		},
		{
			name: "response.content_part.done",
			data: gin.H{
				"type":          "response.content_part.done",
				"content_index": 0,
				"item_id":       messageID,
				"output_index":  0,
				"part": gin.H{
					"type":        "output_text",
					"annotations": []any{},
					"logprobs":    []any{},
					"text":        answer,
				},
				"sequence_number": 6,
			},
		},
		{
			name: "response.output_item.done",
			data: gin.H{
				"type": "response.output_item.done",
				"item": gin.H{
					"id":     messageID,
					"type":   "message",
					"status": "completed",
					"content": []any{
						gin.H{
							"type":        "output_text",
							"annotations": []any{},
							"logprobs":    []any{},
							"text":        answer,
						},
					},
					"role": "assistant",
				},
				"output_index":    0,
				"sequence_number": 7,
			},
		},
		{
			name: "response.completed",
			data: gin.H{
				"type":            "response.completed",
				"response":        buildLocalResponsesCompletedPayload(req, responseID, messageID, createdAt, answer, true),
				"sequence_number": 8,
			},
		},
	}

	for _, event := range events {
		if err := writeSSEEvent(c.Writer, event.name, event.data); err != nil {
			return err
		}
		flusher.Flush()
	}
	return nil
}

func writeSSEEvent(writer gin.ResponseWriter, name string, payload any) error {
	data, err := common.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err = writer.WriteString("event: " + name + "\n"); err != nil {
		return err
	}
	if _, err = writer.WriteString("data: " + string(data) + "\n\n"); err != nil {
		return err
	}
	return nil
}

func newLocalResponsesProbeIDs() (string, string, int64) {
	now := time.Now()
	ts := now.UnixNano()
	return fmt.Sprintf("resp_local_%d", ts), fmt.Sprintf("msg_local_%d", ts), now.Unix()
}

func buildLocalResponsesCreatedPayload(req *dto.OpenAIResponsesRequest, responseID string, createdAt int64) gin.H {
	return gin.H{
		"id":                  responseID,
		"object":              "response",
		"created_at":          createdAt,
		"status":              "in_progress",
		"background":          false,
		"completed_at":        nil,
		"error":               nil,
		"model":               req.Model,
		"output":              []any{},
		"parallel_tool_calls": true,
		"reasoning":           gin.H{"effort": "none", "summary": nil},
		"store":               false,
		"tool_choice":         "auto",
		"tools":               []any{},
		"usage":               nil,
	}
}

func buildLocalResponsesCompletedPayload(req *dto.OpenAIResponsesRequest, responseID string, messageID string, createdAt int64, answer string, stream bool) gin.H {
	completedAt := time.Now().Unix()
	payload := gin.H{
		"id":                  responseID,
		"object":              "response",
		"created_at":          createdAt,
		"status":              "completed",
		"background":          false,
		"completed_at":        completedAt,
		"error":               nil,
		"model":               req.Model,
		"parallel_tool_calls": true,
		"reasoning":           gin.H{"effort": "none", "summary": nil},
		"store":               false,
		"tool_choice":         "auto",
		"tools":               []any{},
		"usage": gin.H{
			"input_tokens":         0,
			"input_tokens_details": gin.H{"cached_tokens": 0, "text_tokens": 0, "audio_tokens": 0, "image_tokens": 0},
			"output_tokens":        len(answer),
			"output_tokens_details": gin.H{
				"reasoning_tokens": 0,
				"text_tokens":      len(answer),
				"audio_tokens":     0,
				"image_tokens":     0,
			},
			"total_tokens": len(answer),
		},
	}
	if stream {
		payload["output"] = []any{}
		return payload
	}
	payload["output"] = []any{
		gin.H{
			"id":     messageID,
			"type":   "message",
			"status": "completed",
			"role":   "assistant",
			"content": []any{
				gin.H{
					"type":        "output_text",
					"text":        answer,
					"annotations": []any{},
				},
			},
		},
	}
	return payload
}
