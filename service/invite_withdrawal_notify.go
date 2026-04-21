package service

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/bytedance/gopkg/util/gopool"
)

func NotifyInviteWithdrawalReviewedAsync(withdrawal *model.InviteWithdrawal) {
	if withdrawal == nil || withdrawal.Id == 0 {
		return
	}
	withdrawalCopy := *withdrawal
	gopool.Go(func() {
		if err := notifyInviteWithdrawalReviewed(&withdrawalCopy); err != nil {
			common.SysLog(fmt.Sprintf("failed to send invite withdrawal review email for %d: %s", withdrawalCopy.Id, err.Error()))
		}
	})
}

func notifyInviteWithdrawalReviewed(withdrawal *model.InviteWithdrawal) error {
	user, err := model.GetUserById(withdrawal.UserId, false)
	if err != nil {
		return err
	}
	userSetting := user.GetSetting()
	receiver := strings.TrimSpace(userSetting.NotificationEmail)
	if receiver == "" {
		receiver = strings.TrimSpace(user.Email)
	}
	if receiver == "" {
		common.SysLog(fmt.Sprintf("user %d has no email, skip invite withdrawal review notification", user.Id))
		return nil
	}

	subject, content := buildInviteWithdrawalReviewEmail(withdrawal, user.Username)
	return common.SendEmail(subject, receiver, content)
}

func buildInviteWithdrawalReviewEmail(withdrawal *model.InviteWithdrawal, username string) (string, string) {
	statusLabel := "已处理"
	statusDesc := "管理员已经处理了你的邀请提现申请。"
	resultColor := "#16a34a"
	extraDesc := ""

	switch withdrawal.Status {
	case model.InviteWithdrawalStatusPaid:
		statusLabel = "已打款"
		statusDesc = "你的邀请提现申请已审核通过，管理员将按收款码信息线下打款。"
	case model.InviteWithdrawalStatusRejected:
		statusLabel = "已驳回"
		statusDesc = "你的邀请提现申请未通过审核，预扣的邀请余额已经退回到账户。"
		resultColor = "#dc2626"
		extraDesc = fmt.Sprintf("<p style='margin:8px 0 0;color:#475569;'>已退回邀请余额：<strong>%s</strong></p>", html.EscapeString(logger.LogQuota(withdrawal.Quota)))
	}

	processedAt := "-"
	if withdrawal.ProcessedAt > 0 {
		processedAt = time.Unix(withdrawal.ProcessedAt, 0).Format("2006-01-02 15:04:05")
	}

	adminRemark := strings.TrimSpace(withdrawal.AdminRemark)
	adminRemarkHTML := "<span style='color:#94a3b8;'>无</span>"
	if adminRemark != "" {
		adminRemarkHTML = html.EscapeString(adminRemark)
	}

	subject := fmt.Sprintf("%s邀请提现申请%s通知", common.SystemName, statusLabel)
	content := fmt.Sprintf(`
<div style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;line-height:1.7;color:#0f172a;">
  <h2 style="margin:0 0 16px;">邀请提现申请%s</h2>
  <p style="margin:0 0 16px;">%s，您好。</p>
  <div style="border:1px solid #e2e8f0;border-radius:12px;padding:16px 18px;background:#f8fafc;">
    <p style="margin:0 0 8px;">处理结果：<strong style="color:%s;">%s</strong></p>
    <p style="margin:8px 0 0;">提现金额：<strong>%.2f</strong></p>
    <p style="margin:8px 0 0;">占用邀请余额：<strong>%s</strong></p>
    <p style="margin:8px 0 0;">处理时间：<strong>%s</strong></p>
    <p style="margin:8px 0 0;">管理员备注：%s</p>
    %s
  </div>
  <p style="margin:16px 0 0;color:#334155;">%s</p>
</div>`,
		statusLabel,
		html.EscapeString(common.GetStringIfEmpty(username, withdrawal.Username)),
		resultColor,
		statusLabel,
		withdrawal.Amount,
		html.EscapeString(logger.LogQuota(withdrawal.Quota)),
		html.EscapeString(processedAt),
		adminRemarkHTML,
		extraDesc,
		statusDesc,
	)
	return subject, content
}
