package ports

import "context"

type EmailMessage struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

type EmailSender interface {
	SendEmail(ctx context.Context, message EmailMessage) bool
}

// Module bundles the shared notification capability for composition roots.
type Module struct {
	EmailSender EmailSender
}
