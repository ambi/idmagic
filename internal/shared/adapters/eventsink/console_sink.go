// ConsoleSink は Console を oauth2/ports.EventSink interface に適合させるアダプタ。
package eventsink

import (
	"context"

	oauthports "idmagic/internal/oauth2/ports"
	"idmagic/internal/shared/spec"
)

type ConsoleSink struct {
	console *Console
}

func NewConsoleSink() oauthports.EventSink {
	return &ConsoleSink{console: NewConsole()}
}

func (s *ConsoleSink) Emit(ctx context.Context, event spec.DomainEvent) error {
	s.console.Emit(ctx, event)
	return nil
}
