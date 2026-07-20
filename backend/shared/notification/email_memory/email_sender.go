package email_memory

import (
	"context"

	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
)

// NoopEmailSender captures messages in memory for tests and local composition.
type NoopEmailSender struct {
	Sent []notificationports.EmailMessage
}

func (s *NoopEmailSender) SendEmail(_ context.Context, message notificationports.EmailMessage) bool {
	s.Sent = append(s.Sent, message)
	return true
}
