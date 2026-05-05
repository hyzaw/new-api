package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRelayPanicRecoverDoesNotAppendSecondJSONAfterWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(recorder)
	engine.Use(RelayPanicRecover())
	engine.GET("/panic-after-write", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "first error",
				"type":    "new_api_error",
			},
		})
		panic("secondary panic after write")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic-after-write", nil)
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	body := recorder.Body.String()
	want := `{"error":{"message":"first error","type":"new_api_error"}}`
	if body != want {
		t.Fatalf("response body = %q, want %q", body, want)
	}
}
