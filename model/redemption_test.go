package model

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func createRedemptionTestUser(t *testing.T, username string) User {
	t.Helper()
	user := User{
		Username:    username,
		Password:    "password",
		DisplayName: username,
		Status:      common.UserStatusEnabled,
		Role:        common.RoleCommonUser,
		AffCode:     username,
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func TestRedeemNormalCodeStillSingleUse(t *testing.T) {
	truncateTables(t)

	user1 := createRedemptionTestUser(t, "redeem-normal-1")
	user2 := createRedemptionTestUser(t, "redeem-normal-2")
	redemption := Redemption{
		UserId:      1,
		Key:         "normal-redemption-code-00000001",
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "normal",
		Quota:       100,
		CreatedTime: common.GetTimestamp(),
	}
	if err := redemption.Insert(); err != nil {
		t.Fatalf("failed to create redemption: %v", err)
	}

	result, err := Redeem(redemption.Key, user1.Id)
	if err != nil {
		t.Fatalf("first redeem failed: %v", err)
	}
	if result.Quota != 100 || result.TotalQuota != 100 {
		t.Fatalf("unexpected redeem result: %+v", result)
	}
	if _, err = Redeem(redemption.Key, user2.Id); !errors.Is(err, ErrRedeemFailed) {
		t.Fatalf("second normal redeem should fail with ErrRedeemFailed, got %v", err)
	}
}

func TestRedeemLotteryCodeAllowsDifferentUsersOnce(t *testing.T) {
	truncateTables(t)

	user1 := createRedemptionTestUser(t, "redeem-lottery-1")
	user2 := createRedemptionTestUser(t, "redeem-lottery-2")
	redemption := Redemption{
		UserId:              1,
		Key:                 "lottery-redemption-code-000001",
		Status:              common.RedemptionCodeStatusEnabled,
		Name:                "lottery",
		Type:                RedemptionTypeLottery,
		LotteryMode:         RedemptionLotteryModeChoices,
		LotteryQuotaChoices: "200:80,500:20",
		CreatedTime:         common.GetTimestamp(),
	}
	if err := redemption.Insert(); err != nil {
		t.Fatalf("failed to create lottery redemption: %v", err)
	}

	result1, err := Redeem(redemption.Key, user1.Id)
	if err != nil {
		t.Fatalf("first user lottery redeem failed: %v", err)
	}
	if result1.Quota != 200 && result1.Quota != 500 {
		t.Fatalf("first user got quota outside choices: %+v", result1)
	}
	result2, err := Redeem(redemption.Key, user2.Id)
	if err != nil {
		t.Fatalf("second user lottery redeem failed: %v", err)
	}
	if result2.Quota != 200 && result2.Quota != 500 {
		t.Fatalf("second user got quota outside choices: %+v", result2)
	}
	if _, err = Redeem(redemption.Key, user1.Id); !errors.Is(err, ErrLotteryRedemptionAlreadyRedeemed) {
		t.Fatalf("same user should not redeem lottery twice, got %v", err)
	}

	var refreshed Redemption
	if err = DB.First(&refreshed, redemption.Id).Error; err != nil {
		t.Fatalf("failed to reload redemption: %v", err)
	}
	if refreshed.Status != common.RedemptionCodeStatusEnabled {
		t.Fatalf("lottery redemption should remain enabled, got status %d", refreshed.Status)
	}
	if refreshed.RedeemedCount != 2 {
		t.Fatalf("unexpected redeemed count: %d", refreshed.RedeemedCount)
	}
}

func TestRedeemLotteryRange(t *testing.T) {
	truncateTables(t)

	user := createRedemptionTestUser(t, "redeem-lottery-range")
	redemption := Redemption{
		UserId:          1,
		Key:             "lottery-redemption-code-000002",
		Status:          common.RedemptionCodeStatusEnabled,
		Name:            "lottery-range",
		Type:            RedemptionTypeLottery,
		LotteryMode:     RedemptionLotteryModeRange,
		LotteryQuotaMin: 10,
		LotteryQuotaMax: 20,
		CreatedTime:     common.GetTimestamp(),
	}
	if err := redemption.Insert(); err != nil {
		t.Fatalf("failed to create lottery redemption: %v", err)
	}

	result, err := Redeem(redemption.Key, user.Id)
	if err != nil {
		t.Fatalf("lottery range redeem failed: %v", err)
	}
	if result.Quota < 10 || result.Quota > 20 {
		t.Fatalf("range lottery quota outside expected bounds: %+v", result)
	}
}

func TestRedeemLotteryGiftBalance(t *testing.T) {
	truncateTables(t)

	user := createRedemptionTestUser(t, "redeem-lottery-gift")
	redemption := Redemption{
		UserId:              1,
		Key:                 "lottery-redemption-code-gift",
		Status:              common.RedemptionCodeStatusEnabled,
		Name:                "lottery-gift",
		Type:                RedemptionTypeLottery,
		LotteryMode:         RedemptionLotteryModeChoices,
		LotteryQuotaChoices: "300:1",
		LotteryBalanceType:  RedemptionLotteryBalanceGift,
		CreatedTime:         common.GetTimestamp(),
	}
	if err := redemption.Insert(); err != nil {
		t.Fatalf("failed to create lottery redemption: %v", err)
	}

	result, err := Redeem(redemption.Key, user.Id)
	if err != nil {
		t.Fatalf("lottery gift redeem failed: %v", err)
	}
	if result.Quota != 0 || result.GiftQuota != 300 || result.TotalQuota != 300 {
		t.Fatalf("unexpected lottery gift result: %+v", result)
	}

	var refreshed User
	if err = DB.First(&refreshed, user.Id).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if refreshed.Quota != 0 || refreshed.GiftQuota != 300 {
		t.Fatalf("unexpected user balances: quota=%d gift_quota=%d", refreshed.Quota, refreshed.GiftQuota)
	}
}

func TestRedeemLotteryMaxRedeemCount(t *testing.T) {
	truncateTables(t)

	user1 := createRedemptionTestUser(t, "redeem-lottery-max-1")
	user2 := createRedemptionTestUser(t, "redeem-lottery-max-2")
	redemption := Redemption{
		UserId:              1,
		Key:                 "custom-lottery-key",
		Status:              common.RedemptionCodeStatusEnabled,
		Name:                "lottery-max",
		Type:                RedemptionTypeLottery,
		LotteryMode:         RedemptionLotteryModeChoices,
		LotteryQuotaChoices: "100:1",
		MaxRedeemCount:      1,
		CreatedTime:         common.GetTimestamp(),
	}
	if err := redemption.Insert(); err != nil {
		t.Fatalf("failed to create lottery redemption: %v", err)
	}

	if _, err := Redeem(redemption.Key, user1.Id); err != nil {
		t.Fatalf("first lottery redeem failed: %v", err)
	}
	if _, err := Redeem(redemption.Key, user2.Id); !errors.Is(err, ErrLotteryRedemptionExhausted) {
		t.Fatalf("lottery should be exhausted, got %v", err)
	}
}
