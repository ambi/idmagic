// ConsoleSink は Console を oauth2/ports.EventSink interface に適合させるアダプタ。
package sinks_console

import (
	"context"

	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
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
