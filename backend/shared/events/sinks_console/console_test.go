package sinks_console

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

type consoleTestEvent struct {
	At time.Time
}

func (consoleTestEvent) EventType() string       { return "ConsoleTestEvent" }
func (e consoleTestEvent) OccurredAt() time.Time { return e.At }

type unmarshalableConsoleEvent struct{ Fn func() }

func (unmarshalableConsoleEvent) EventType() string     { return "Unmarshalable" }
func (unmarshalableConsoleEvent) OccurredAt() time.Time { return time.Now() }

func TestConsoleEmitWritesOneJSONLine(t *testing.T) {
	var buf bytes.Buffer
	c := &Console{out: &buf}
	c.Emit(context.Background(), consoleTestEvent{At: time.Now()})
	out := buf.String()
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("expected trailing newline, got %q", out)
	}
	if !strings.Contains(out, `"type":"ConsoleTestEvent"`) {
		t.Fatalf("expected type field, got %q", out)
	}
}

func TestConsoleEmitSkipsWriteOnMarshalError(t *testing.T) {
	var buf bytes.Buffer
	c := &Console{out: &buf}
	c.Emit(context.Background(), unmarshalableConsoleEvent{Fn: func() {}})
	if buf.Len() != 0 {
		t.Fatalf("expected no output on marshal error, got %q", buf.String())
	}
}

func TestNewConsoleWritesToStdout(t *testing.T) {
	c := NewConsole()
	if c.out == nil {
		t.Fatal("expected a non-nil writer")
	}
}

func TestConsoleSinkEmitDelegatesAndReturnsNil(t *testing.T) {
	var buf bytes.Buffer
	sink := &ConsoleSink{console: &Console{out: &buf}}
	if err := sink.Emit(context.Background(), consoleTestEvent{At: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected the underlying console to receive the event")
	}
}

func TestNewConsoleSinkReturnsAnEventSink(t *testing.T) {
	sink := NewConsoleSink()
	if sink == nil {
		t.Fatal("expected a non-nil EventSink")
	}
}
