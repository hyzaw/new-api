package middleware

import (
	"crypto/hmac"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func PublicRequestSignatureRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		timestampStr := c.GetHeader(common.PublicRequestTimestampHeader)
		signature := c.GetHeader(common.PublicRequestSignatureHeader)
		if timestampStr == "" || signature == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "请求缺少签名或时间戳",
			})
			c.Abort()
			return
		}

		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "请求时间戳无效",
			})
			c.Abort()
			return
		}

		now := time.Now().Unix()
		delta := now - timestamp
		if delta < 0 {
			delta = -delta
		}
		if delta > common.PublicRequestSigningWindowSeconds {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "请求已过期，请刷新后重试",
			})
			c.Abort()
			return
		}

		body := ""
		if c.Request != nil && c.Request.Body != nil && c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			storage, err := common.GetBodyStorage(c)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "读取请求体失败",
				})
				c.Abort()
				return
			}
			bodyBytes, err := storage.Bytes()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "读取请求体失败",
				})
				c.Abort()
				return
			}
			body = string(bodyBytes)
			if _, err = storage.Seek(0, io.SeekStart); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "重置请求体失败",
				})
				c.Abort()
				return
			}
			c.Request.Body = io.NopCloser(storage)
		}

		target := ""
		if c.Request != nil && c.Request.URL != nil {
			target = c.Request.URL.RequestURI()
		}
		if target == "" && c.Request != nil {
			target = c.Request.RequestURI
		}

		expected := common.SignPublicRequest(c.Request.Method, target, timestamp, body)
		if !hmac.Equal([]byte(signature), []byte(expected)) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "请求验签失败",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
