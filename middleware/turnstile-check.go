package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type turnstileCheckResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

func mapTurnstileErrorMessage(errorCodes []string) string {
	if len(errorCodes) == 0 {
		return "Turnstile 校验失败，请刷新重试！"
	}

	for _, code := range errorCodes {
		switch code {
		case "timeout-or-duplicate":
			return "Turnstile 已过期或已被使用，请重新完成校验后再试！"
		case "missing-input-response":
			return "Turnstile 响应缺失，请重新完成校验后再试！"
		case "invalid-input-response":
			return "Turnstile 响应无效，请重新完成校验后再试！"
		case "missing-input-secret":
			return "Turnstile 服务端密钥缺失，请检查系统配置！"
		case "invalid-input-secret":
			return "Turnstile 服务端密钥无效，请检查系统配置！"
		case "bad-request":
			return "Turnstile 请求格式错误，请稍后重试！"
		case "internal-error":
			return "Turnstile 服务暂时异常，请稍后重试！"
		}
	}

	return "Turnstile 校验失败（" + strings.Join(errorCodes, ", ") + "）"
}

func TurnstileCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.TurnstileCheckEnabled {
			session := sessions.Default(c)
			turnstileChecked := session.Get("turnstile")
			if turnstileChecked != nil {
				c.Next()
				return
			}
			response := c.Query("turnstile")
			if response == "" {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "Turnstile token 为空",
				})
				c.Abort()
				return
			}
			rawRes, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", url.Values{
				"secret":   {common.TurnstileSecretKey},
				"response": {response},
				"remoteip": {c.ClientIP()},
			})
			if err != nil {
				common.SysLog(err.Error())
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": err.Error(),
				})
				c.Abort()
				return
			}
			defer rawRes.Body.Close()
			var res turnstileCheckResponse
			err = common.DecodeJson(rawRes.Body, &res)
			if err != nil {
				common.SysLog(err.Error())
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": err.Error(),
				})
				c.Abort()
				return
			}
			if !res.Success {
				if len(res.ErrorCodes) > 0 {
					common.SysLog("turnstile verify failed: " + strings.Join(res.ErrorCodes, ","))
				}
				errorMessage := mapTurnstileErrorMessage(res.ErrorCodes)
				c.JSON(http.StatusOK, gin.H{
					"success":     false,
					"message":     errorMessage,
					"error_codes": res.ErrorCodes,
				})
				c.Abort()
				return
			}
			session.Set("turnstile", true)
			err = session.Save()
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"message": "无法保存会话信息，请重试",
					"success": false,
				})
				return
			}
		}
		c.Next()
	}
}
