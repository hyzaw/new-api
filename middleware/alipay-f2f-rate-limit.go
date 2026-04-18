package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const (
	alipayF2FCreateOrderRateLimitMark     = "AF2F"
	alipayF2FCreateOrderRateLimitNum      = 3
	alipayF2FCreateOrderRateLimitDuration = 5 * 60
)

func abortAlipayF2FRateLimit(c *gin.Context) {
	c.JSON(http.StatusTooManyRequests, gin.H{
		"success": false,
		"message": "请求过于频繁，请5分钟后再试",
		"data":    "5分钟内最多允许创建3次支付宝当面付订单",
	})
	c.Abort()
}

func alipayF2FCreateOrderRedisRateLimiter(c *gin.Context) {
	ctx := context.Background()
	rdb := common.RDB
	key := fmt.Sprintf("rateLimit:%s:%s", alipayF2FCreateOrderRateLimitMark, c.ClientIP())

	listLength, err := rdb.LLen(ctx, key).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "限流检查失败",
		})
		c.Abort()
		return
	}

	if listLength < int64(alipayF2FCreateOrderRateLimitNum) {
		rdb.LPush(ctx, key, time.Now().Format(timeFormat))
		rdb.Expire(ctx, key, common.RateLimitKeyExpirationDuration)
		c.Next()
		return
	}

	oldTimeStr, _ := rdb.LIndex(ctx, key, -1).Result()
	oldTime, err := time.Parse(timeFormat, oldTimeStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "限流检查失败",
		})
		c.Abort()
		return
	}

	nowTimeStr := time.Now().Format(timeFormat)
	nowTime, err := time.Parse(timeFormat, nowTimeStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "限流检查失败",
		})
		c.Abort()
		return
	}

	if int64(nowTime.Sub(oldTime).Seconds()) < alipayF2FCreateOrderRateLimitDuration {
		rdb.Expire(ctx, key, common.RateLimitKeyExpirationDuration)
		abortAlipayF2FRateLimit(c)
		return
	}

	rdb.LPush(ctx, key, time.Now().Format(timeFormat))
	rdb.LTrim(ctx, key, 0, int64(alipayF2FCreateOrderRateLimitNum-1))
	rdb.Expire(ctx, key, common.RateLimitKeyExpirationDuration)
	c.Next()
}

func alipayF2FCreateOrderMemoryRateLimiter(c *gin.Context) {
	key := fmt.Sprintf("%s:%s", alipayF2FCreateOrderRateLimitMark, c.ClientIP())
	if !inMemoryRateLimiter.Request(
		key,
		alipayF2FCreateOrderRateLimitNum,
		alipayF2FCreateOrderRateLimitDuration,
	) {
		abortAlipayF2FRateLimit(c)
		return
	}
	c.Next()
}

func AlipayF2FCreateOrderRateLimit() gin.HandlerFunc {
	if common.RedisEnabled {
		return alipayF2FCreateOrderRedisRateLimiter
	}

	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	return alipayF2FCreateOrderMemoryRateLimiter
}
