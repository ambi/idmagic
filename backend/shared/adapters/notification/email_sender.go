package notification

import (
	"context"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
)

// ConsoleEmailSender は dev / demo 用の配送アダプタ。本番では EMAIL_SENDER=smtp を使う。
// 宛先アドレスは PII なのでマスクする (ADR-018 §4)。本文は dev でリセットリンク等を
// 確認するために出力する (本アダプタは本番では動かない)。
type ConsoleEmailSender struct{}

func (ConsoleEmailSender) SendEmail(ctx context.Context, message authnports.EmailMessage) bool {
	logging.Info(ctx, "email delivered (console sender)",
		"to", logging.MaskEmail(message.To),
		"subject", message.Subject,
		"body", message.Text)
	return true
}

type NoopEmailSender struct {
	Sent []authnports.EmailMessage
}

func (s *NoopEmailSender) SendEmail(_ context.Context, message authnports.EmailMessage) bool {
	s.Sent = append(s.Sent, message)
	return true
}
