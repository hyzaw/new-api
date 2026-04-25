package controller

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func mustMarshalProbeInput(t *testing.T, text string) []byte {
	t.Helper()
	data, err := common.Marshal([]map[string]any{
		{
			"role":    "user",
			"content": text,
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal probe input: %v", err)
	}
	return data
}

func TestDetectLocalResponsesArithmeticProbe(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{
		Model: "gpt-5.3-codex",
		Input: mustMarshalProbeInput(t, "Calculate and respond with ONLY the number, nothing else.\n\nQ: 3 + 5 = ?\nA: 8\n\nQ: 12 - 7 = ?\nA: 5\n\nQ: 18 - 16 = ?\nA:"),
	}

	answer, ok := detectLocalResponsesArithmeticProbe(req)
	if !ok {
		t.Fatalf("expected local probe to be detected")
	}
	if answer != "2" {
		t.Fatalf("expected answer 2, got %q", answer)
	}
}

func TestDetectLocalResponsesArithmeticProbeRejectsGenericMathPrompt(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{
		Model: "gpt-5.3-codex",
		Input: mustMarshalProbeInput(t, "What is 18 - 16? Reply with only the number."),
	}

	if _, ok := detectLocalResponsesArithmeticProbe(req); ok {
		t.Fatalf("expected generic math prompt not to be treated as local probe")
	}
}

func TestTryHandleLocalRelayProbeStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/v1/responses", nil)

	stream := true
	req := &dto.OpenAIResponsesRequest{
		Model:  "gpt-5.3-codex",
		Input:  mustMarshalProbeInput(t, "Calculate and respond with ONLY the number, nothing else.\n\nQ: 3 + 5 = ?\nA: 8\n\nQ: 12 - 7 = ?\nA: 5\n\nQ: 18 - 16 = ?\nA:"),
		Stream: &stream,
	}

	handled, err := tryHandleLocalRelayProbe(ctx, types.RelayFormatOpenAIResponses, req)
	if !handled {
		t.Fatalf("expected probe to be handled locally")
	}
	if err != nil {
		t.Fatalf("unexpected local probe error: %v", err)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "event: response.output_text.delta") {
		t.Fatalf("expected SSE delta event, got %s", body)
	}
	if !strings.Contains(body, "\"delta\":\"2\"") {
		t.Fatalf("expected SSE response to contain answer 2, got %s", body)
	}
}
