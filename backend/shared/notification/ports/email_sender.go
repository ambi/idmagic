package ports

import "context"

type EmailMessage struct {
	To      string
	Subject string
	Text    string
	HTML    string
	// FromDisplayName overrides only the display part of the From mailbox. The
	// address itself stays server configuration, so a tenant can
	// never point the envelope sender somewhere else.
	FromDisplayName string
}

type EmailSender interface {
	SendEmail(ctx context.Context, message EmailMessage) bool
}

// Module bundles the shared notification capability for composition roots.
type Module struct {
	EmailSender EmailSender
	// Notifier resolves the localized template catalog before delegating to
	// EmailSender. Use cases depend on this rather than EmailSender so no call
	// site composes message bodies of its own.
	Notifier Notifier
}
