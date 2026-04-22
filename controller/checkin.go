package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	checkinStatusScope  = "gift_calendar_v2"
	checkinClaimAction  = "daily_gift_claim_v2"
	checkinChallengeTTL = 120
)

type checkinClaimRequest struct {
	ActionCode  string `json:"action_code"`
	VerifyToken string `json:"verify_token,omitempty"`
	Seed        string `json:"seed"`
	Stamp       int64  `json:"stamp"`
	Proof       string `json:"proof"`
}

type checkinClaimMeta struct {
	Seed      string `json:"seed"`
	Stamp     int64  `json:"stamp"`
	Proof     string `json:"proof"`
	ExpiresIn int64  `json:"expires_in"`
}

func getCheckinChallengeSignature(userId int, seed string, stamp int64, userAgent string) string {
	payload := fmt.Sprintf("checkin|%d|%s|%d|%s", userId, seed, stamp, strings.TrimSpace(userAgent))
	return common.GenerateHMAC(payload)
}

func issueCheckinClaimMeta(c *gin.Context, userId int) (*checkinClaimMeta, error) {
	session := sessions.Default(c)
	stamp := common.GetTimestamp()
	seed := common.GetRandomString(24)
	userAgent := ""
	if c != nil && c.Request != nil {
		userAgent = c.Request.UserAgent()
	}
	proof := getCheckinChallengeSignature(userId, seed, stamp, userAgent)
	session.Set("checkin_seed", seed)
	session.Set("checkin_stamp", stamp)
	session.Set("checkin_proof", proof)
	session.Set("checkin_deadline", stamp+checkinChallengeTTL)
	if err := session.Save(); err != nil {
		return nil, err
	}
	return &checkinClaimMeta{
		Seed:      seed,
		Stamp:     stamp,
		Proof:     proof,
		ExpiresIn: checkinChallengeTTL,
	}, nil
}

func clearCheckinClaimMeta(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("checkin_seed")
	session.Delete("checkin_stamp")
	session.Delete("checkin_proof")
	session.Delete("checkin_deadline")
	_ = session.Save()
}

func getSessionString(session sessions.Session, key string) string {
	raw := session.Get(key)
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", raw))
}

func validateCheckinClaimMeta(c *gin.Context, userId int, req *checkinClaimRequest) error {
	if req == nil {
		return fmt.Errorf("invalid request")
	}
	session := sessions.Default(c)
	rawSeed := getSessionString(session, "checkin_seed")
	rawProof := getSessionString(session, "checkin_proof")
	if rawSeed == "" || rawProof == "" {
		return fmt.Errorf("签到校验已失效，请刷新页面后重试")
	}
	rawStamp, ok := session.Get("checkin_stamp").(int64)
	if !ok {
		switch v := session.Get("checkin_stamp").(type) {
		case int:
			rawStamp = int64(v)
		case int32:
			rawStamp = int64(v)
		case int64:
			rawStamp = v
		case float64:
			rawStamp = int64(v)
		default:
			clearCheckinClaimMeta(c)
			return fmt.Errorf("签到校验已失效，请刷新页面后重试")
		}
	}
	rawDeadline, ok := session.Get("checkin_deadline").(int64)
	if !ok {
		switch v := session.Get("checkin_deadline").(type) {
		case int:
			rawDeadline = int64(v)
		case int32:
			rawDeadline = int64(v)
		case int64:
			rawDeadline = v
		case float64:
			rawDeadline = int64(v)
		default:
			clearCheckinClaimMeta(c)
			return fmt.Errorf("签到校验已失效，请刷新页面后重试")
		}
	}

	now := common.GetTimestamp()
	if req.Stamp <= 0 || now-req.Stamp > checkinChallengeTTL || req.Stamp-now > 15 {
		clearCheckinClaimMeta(c)
		return fmt.Errorf("签到请求时间戳无效，请刷新页面后重试")
	}
	if now > rawDeadline || req.Stamp != rawStamp {
		clearCheckinClaimMeta(c)
		return fmt.Errorf("签到校验已过期，请刷新页面后重试")
	}
	if req.Seed != rawSeed {
		clearCheckinClaimMeta(c)
		return fmt.Errorf("签到校验失败，请刷新页面后重试")
	}
	userAgent := ""
	if c != nil && c.Request != nil {
		userAgent = c.Request.UserAgent()
	}
	expectedProof := getCheckinChallengeSignature(userId, req.Seed, req.Stamp, userAgent)
	if req.Proof != rawProof || req.Proof != expectedProof {
		clearCheckinClaimMeta(c)
		return fmt.Errorf("签到动态签名校验失败，请刷新页面后重试")
	}
	return nil
}

// GetCheckinStatus 获取用户签到状态和历史记录
func GetCheckinStatus(c *gin.Context) {
	setting := operation_setting.GetCheckinSetting()
	if !setting.Enabled {
		common.ApiErrorMsg(c, "签到功能未启用")
		return
	}
	userId := c.GetInt("id")
	if strings.TrimSpace(c.DefaultQuery("scope", "")) != checkinStatusScope {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	// 获取月份参数，默认为当前月份
	month := c.DefaultQuery("period", time.Now().Format("2006-01"))

	stats, err := model.GetUserCheckinStats(userId, month)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	eligibility, err := model.GetUserCheckinEligibility(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	claimMeta, err := issueCheckinClaimMeta(c, userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":              setting.Enabled,
			"min_quota":            setting.MinQuota,
			"max_quota":            setting.MaxQuota,
			"min_topup_amount":     setting.MinTopUpAmount,
			"current_topup_amount": eligibility.CurrentTopUpAmount,
			"can_checkin":          eligibility.Eligible,
			"stats":                stats,
			"claim_meta":           claimMeta,
		},
	})
}

// DoCheckin 执行用户签到
func DoCheckin(c *gin.Context) {
	setting := operation_setting.GetCheckinSetting()
	if !setting.Enabled {
		common.ApiErrorMsg(c, "签到功能未启用")
		return
	}
	var req checkinClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if strings.TrimSpace(req.ActionCode) != checkinClaimAction {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	userId := c.GetInt("id")
	if err := validateCheckinClaimMeta(c, userId, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	checkin, err := model.UserCheckin(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	clearCheckinClaimMeta(c)
	model.RecordLog(userId, model.LogTypeSystem, fmt.Sprintf("用户签到，获得赠送余额 %s", logger.LogQuota(checkin.QuotaAwarded)))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "签到成功",
		"data": gin.H{
			"quota_awarded": checkin.QuotaAwarded,
			"reward_type":   "gift_quota",
			"checkin_date":  checkin.CheckinDate},
	})
}
