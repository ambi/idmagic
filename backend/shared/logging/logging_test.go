package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func decodeLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("no log line written")
	}
	if strings.Contains(line, "\n") {
		t.Fatalf("expected a single JSON line, got:\n%s", line)
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("log line is not valid JSON: %v\n%s", err, line)
	}
	return rec
}

func TestNew_StructuredRequiredFields(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, slog.LevelInfo, "idmagic", "1.2.3")
	log.Info(context.Background(), "listening", "addr", ":8080")

	rec := decodeLine(t, &buf)
	for _, field := range []string{"timestamp", "level", "service", "message"} {
		if _, ok := rec[field]; !ok {
			t.Errorf("required field %q missing: %v", field, rec)
		}
	}
	if rec["service"] != "idmagic" {
		t.Errorf("service = %v, want idmagic", rec["service"])
	}
	if rec["message"] != "listening" {
		t.Errorf("message = %v, want listening", rec["message"])
	}
	if rec["addr"] != ":8080" {
		t.Errorf("addr = %v, want :8080", rec["addr"])
	}
}

func TestNew_LevelFiltersBelowThreshold(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, slog.LevelWarn, "idmagic", "")
	log.Info(context.Background(), "should be filtered")
	if buf.Len() != 0 {
		t.Fatalf("info line emitted at warn level: %s", buf.String())
	}
	log.Warn(context.Background(), "kept")
	if rec := decodeLine(t, &buf); rec["level"] != "WARN" {
		t.Errorf("level = %v, want WARN", rec["level"])
	}
}

func TestNew_TraceCorrelation(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, slog.LevelInfo, "idmagic", "")

	traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := trace.SpanIDFromHex("0102030405060708")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	log.Info(ctx, "with span")
	rec := decodeLine(t, &buf)
	if rec["trace_id"] != traceID.String() {
		t.Errorf("trace_id = %v, want %s", rec["trace_id"], traceID)
	}
	if rec["span_id"] != spanID.String() {
		t.Errorf("span_id = %v, want %s", rec["span_id"], spanID)
	}
}

func TestNew_NoTraceFieldsWithoutSpan(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, slog.LevelInfo, "idmagic", "")
	log.Info(context.Background(), "no span")
	rec := decodeLine(t, &buf)
	if _, ok := rec["trace_id"]; ok {
		t.Error("trace_id present without an active span")
	}
}

func TestNew_RequestIDCorrelation(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, slog.LevelInfo, "idmagic", "")

	ctx := ContextWithRequestID(context.Background(), "req-abc-123")
	log.Info(ctx, "with request id")
	if rec := decodeLine(t, &buf); rec["request_id"] != "req-abc-123" {
		t.Errorf("request_id = %v, want req-abc-123", rec["request_id"])
	}

	buf.Reset()
	log.Info(context.Background(), "no request id")
	if _, ok := decodeLine(t, &buf)["request_id"]; ok {
		t.Error("request_id present without a request context")
	}
}

func TestMaskEmail(t *testing.T) {
	cases := map[string]string{
		"alice@example.com":    "***@example.com",
		"a@b.co":               "***@b.co",
		"bob.smith@corp.co.jp": "***@corp.co.jp",
		"not-an-email":         "***",
		"":                     "***",
		"@example.com":         "***",
		"trailing@":            "***",
	}
	for in, want := range cases {
		if got := MaskEmail(in); got != want {
			t.Errorf("MaskEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"info":    slog.LevelInfo,
		"warn":    slog.LevelWarn,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
		"":        slog.LevelInfo,
		"bogus":   slog.LevelInfo,
	}
	for in, want := range cases {
		if got := ParseLevel(in); got != want {
			t.Errorf("ParseLevel(%q) = %v, want %v", in, got, want)
		}
	}
	// 前後の空白と大文字は正規化される。
	if got := ParseLevel("  WARN "); got != slog.LevelWarn {
		t.Errorf("ParseLevel(%q) = %v, want %v", "  WARN ", got, slog.LevelWarn)
	}
}
