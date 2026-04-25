package controller

import (
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func GetAllRedemptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.GetAllRedemptions(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
	return
}

func SearchRedemptions(c *gin.Context) {
	keyword := c.Query("keyword")
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.SearchRedemptions(keyword, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetRedemption(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	redemption, err := model.GetRedemptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    redemption,
	})
	return
}

func AddRedemption(c *gin.Context) {
	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(redemption.Name) == 0 || utf8.RuneCountInString(redemption.Name) > 20 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionNameLength)
		return
	}
	redemption.NormalizeType()
	if redemption.Count <= 0 {
		if redemption.IsLottery() {
			redemption.Count = 1
		} else {
			common.ApiErrorI18n(c, i18n.MsgRedemptionCountPositive)
			return
		}
	}
	if !redemption.IsLottery() && redemption.Count > 100 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountMax)
		return
	}
	if redemption.IsLottery() {
		redemption.Count = 1
		redemption.Quota = 0
		if utf8.RuneCountInString(redemption.Key) > 64 {
			common.ApiError(c, model.ErrInvalidLotteryRedemptionQuota)
			return
		}
	}
	if err := model.ValidateLotteryRedemption(&redemption); err != nil {
		common.ApiError(c, err)
		return
	}
	if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}
	var keys []string
	for i := 0; i < redemption.Count; i++ {
		key := common.GetUUID()
		if redemption.IsLottery() && redemption.Key != "" {
			key = redemption.Key
		}
		cleanRedemption := model.Redemption{
			UserId:              c.GetInt("id"),
			Name:                redemption.Name,
			Key:                 key,
			CreatedTime:         common.GetTimestamp(),
			Quota:               redemption.Quota,
			GiftQuota:           redemption.GiftQuota,
			ExpiredTime:         redemption.ExpiredTime,
			Type:                redemption.Type,
			LotteryMode:         redemption.LotteryMode,
			LotteryQuotaMin:     redemption.LotteryQuotaMin,
			LotteryQuotaMax:     redemption.LotteryQuotaMax,
			LotteryQuotaChoices: redemption.LotteryQuotaChoices,
			MaxRedeemCount:      redemption.MaxRedeemCount,
		}
		err = cleanRedemption.Insert()
		if err != nil {
			common.SysError("failed to insert redemption: " + err.Error())
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": i18n.T(c, i18n.MsgRedemptionCreateFailed),
				"data":    keys,
			})
			return
		}
		keys = append(keys, key)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    keys,
	})
	return
}

func DeleteRedemption(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeleteRedemptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateRedemption(c *gin.Context) {
	statusOnly := c.Query("status_only")
	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanRedemption, err := model.GetRedemptionById(redemption.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if statusOnly == "" {
		if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
			return
		}
		// If you add more fields, please also update redemption.Update()
		cleanRedemption.Name = redemption.Name
		cleanRedemption.Quota = redemption.Quota
		cleanRedemption.GiftQuota = redemption.GiftQuota
		cleanRedemption.ExpiredTime = redemption.ExpiredTime
		cleanRedemption.Type = redemption.Type
		cleanRedemption.LotteryMode = redemption.LotteryMode
		cleanRedemption.LotteryQuotaMin = redemption.LotteryQuotaMin
		cleanRedemption.LotteryQuotaMax = redemption.LotteryQuotaMax
		cleanRedemption.LotteryQuotaChoices = redemption.LotteryQuotaChoices
		cleanRedemption.MaxRedeemCount = redemption.MaxRedeemCount
		if cleanRedemption.IsLottery() {
			cleanRedemption.Quota = 0
		}
		if err := model.ValidateLotteryRedemption(cleanRedemption); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if statusOnly != "" {
		cleanRedemption.Status = redemption.Status
	}
	err = cleanRedemption.Update()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanRedemption,
	})
	return
}

func DeleteInvalidRedemption(c *gin.Context) {
	rows, err := model.DeleteInvalidRedemptions()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    rows,
	})
	return
}

func validateExpiredTime(c *gin.Context, expired int64) (bool, string) {
	if expired != 0 && expired < common.GetTimestamp() {
		return false, i18n.T(c, i18n.MsgRedemptionExpireTimeInvalid)
	}
	return true, ""
}
