package middleware

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func ResponseBodyCapture() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get(common.KeyResponseBodyCapture); exists {
			c.Next()
			return
		}
		capture := common.NewResponseBodyCapture(c.Writer)
		common.SetResponseBodyCapture(c, capture)
		c.Writer = capture
		c.Next()
	}
}
