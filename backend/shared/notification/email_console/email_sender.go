package email_console

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/logging"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
)

// ConsoleEmailSender は dev / demo 用の配送アダプタ。本番では EMAIL_SENDER=smtp を使う。
// 宛先アドレスは PII なのでマスクする (ADR-018 §4)。本文は dev でリセットリンク等を
// 確認するために出力する (本アダプタは本番では動かない)。
type ConsoleEmailSender struct{}

func (ConsoleEmailSender) SendEmail(ctx context.Context, message sharednotification.EmailMessage) bool {
	logging.Info(ctx, "email delivered (console sender)",
		"to", logging.MaskEmail(message.To),
		"subject", message.Subject,
		"body", message.Text)
	return true
}
