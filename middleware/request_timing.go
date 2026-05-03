package middleware

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

// RequestTiming stamps the earliest request arrival time for admin-only diagnostics.
func RequestTiming() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.GetContextKeyTime(c, constant.ContextKeyRequestEntryTime).IsZero() {
			common.SetContextKey(c, constant.ContextKeyRequestEntryTime, time.Now())
		}
		c.Next()
	}
}
