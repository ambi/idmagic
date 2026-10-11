package sinks_console

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// ConsoleSink は共通のドメインイベントを JSON へ出力する。
type ConsoleSink struct {
	console *Console
}

func NewConsoleSink() *ConsoleSink {
	return &ConsoleSink{console: NewConsole()}
}

func (s *ConsoleSink) Emit(ctx context.Context, event spec.DomainEvent) error {
	s.console.Emit(ctx, event)
	return nil
}
