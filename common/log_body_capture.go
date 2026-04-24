package common

import (
	"bytes"
	"encoding/base64"
	"sync"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

const (
	KeyResponseBodyCapture = "key_response_body_capture"

	LogBodyEncodingText   = "text"
	LogBodyEncodingBase64 = "base64"
)

type CapturedLogBody struct {
	Encoding string `json:"encoding,omitempty"`
	Body     string `json:"body"`
}

type ResponseBodyCapture struct {
	gin.ResponseWriter
	mu  sync.Mutex
	buf bytes.Buffer
}

func NewResponseBodyCapture(writer gin.ResponseWriter) *ResponseBodyCapture {
	return &ResponseBodyCapture{ResponseWriter: writer}
}

func (w *ResponseBodyCapture) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	if n > 0 {
		w.mu.Lock()
		_, _ = w.buf.Write(data[:n])
		w.mu.Unlock()
	}
	return n, err
}

func (w *ResponseBodyCapture) WriteString(s string) (int, error) {
	n, err := w.ResponseWriter.WriteString(s)
	if n > 0 {
		w.mu.Lock()
		_, _ = w.buf.WriteString(s[:n])
		w.mu.Unlock()
	}
	return n, err
}

func (w *ResponseBodyCapture) BodyBytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	body := w.buf.Bytes()
	cloned := make([]byte, len(body))
	copy(cloned, body)
	return cloned
}

func SetResponseBodyCapture(c *gin.Context, capture *ResponseBodyCapture) {
	if c == nil || capture == nil {
		return
	}
	if existing, ok := c.Get(KeyResponseBodyCapture); ok && existing != nil {
		return
	}
	c.Set(KeyResponseBodyCapture, capture)
}

func GetCapturedResponseBody(c *gin.Context) []byte {
	if c == nil {
		return nil
	}
	value, ok := c.Get(KeyResponseBodyCapture)
	if !ok || value == nil {
		return nil
	}
	capture, ok := value.(*ResponseBodyCapture)
	if !ok {
		return nil
	}
	return capture.BodyBytes()
}

func BuildCapturedLogBody(data []byte) *CapturedLogBody {
	if len(data) == 0 {
		return nil
	}
	if utf8.Valid(data) {
		return &CapturedLogBody{
			Encoding: LogBodyEncodingText,
			Body:     string(data),
		}
	}
	return &CapturedLogBody{
		Encoding: LogBodyEncodingBase64,
		Body:     base64.StdEncoding.EncodeToString(data),
	}
}
