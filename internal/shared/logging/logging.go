// Package logging is the application-log port and its slog-backed adapter
// (ADR-018). Application logs are structured JSON Lines on stdout with level,
// service, and — when an OpenTelemetry span is active — trace_id / span_id for
// correlation with traces.
//
// This is the mutable, short-retention half of ADR-018. The immutable audit log
// (DomainEvent) travels a separate path via the EventSink port and must not be
// written through this package. x-pii fields (email etc.) must never reach an
// application log in plaintext; use MaskEmail and friends before logging.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// Logger is the application-log port. Methods mirror slog's key/value variadic
// convention: Info(ctx, "msg", "key", value, ...). The context carries the
// active span so trace_id / span_id are attached automatically.
type Logger interface {
	Debug(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
	Warn(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)
	// With returns a Logger that includes the given attributes on every record.
	With(args ...any) Logger
}

// slogLogger adapts *slog.Logger to the Logger port.
type slogLogger struct{ l *slog.Logger }

func (s slogLogger) Debug(ctx context.Context, msg string, args ...any) {
	s.l.DebugContext(ctx, msg, args...)
}

func (s slogLogger) Info(ctx context.Context, msg string, args ...any) {
	s.l.InfoContext(ctx, msg, args...)
}

func (s slogLogger) Warn(ctx context.Context, msg string, args ...any) {
	s.l.WarnContext(ctx, msg, args...)
}

func (s slogLogger) Error(ctx context.Context, msg string, args ...any) {
	s.l.ErrorContext(ctx, msg, args...)
}

func (s slogLogger) With(args ...any) Logger { return slogLogger{l: s.l.With(args...)} }

// New builds a JSON Lines Logger writing to w. service and version are attached
// to every record; the field convention follows ADR-018 §3
// (timestamp / level / service / message plus trace_id / span_id).
func New(w io.Writer, level slog.Level, service, version string) Logger {
	return slogLogger{l: NewSlog(w, level, service, version)}
}

// NewSlog builds the underlying *slog.Logger used by New, with the ADR-018
// field convention and trace correlation. It is exposed so framework loggers
// that require a *slog.Logger (e.g. Echo's e.Logger) can share the same format.
func NewSlog(w io.Writer, level slog.Level, service, version string) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: replaceAttr,
	})
	return slog.New(&traceHandler{Handler: handler}).With(
		slog.String("service", service),
		slog.String("version", version),
	)
}

// replaceAttr renames slog's default keys to the ADR-018 field names.
func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.TimeKey:
		a.Key = "timestamp"
	case slog.MessageKey:
		a.Key = "message"
	}
	return a
}

// requestIDKey is the context key under which the per-request correlation id is
// stored so it can be attached to every application-log record for the request.
type requestIDKey struct{}

// ContextWithRequestID returns a context carrying the per-request correlation
// id. Application logs emitted with this context gain a request_id field, which
// lets multiple lines and client reports for one request be correlated even
// when no OpenTelemetry span is active (RequestFaultIsolation objective).
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFromContext returns the per-request correlation id, or "" if none is
// present.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// traceHandler injects trace_id / span_id from the active OpenTelemetry span and
// request_id from the request context, so application logs correlate with traces
// and with a single request (ADR-017 / ADR-018).
type traceHandler struct{ slog.Handler }

func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	if id := RequestIDFromContext(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithGroup(name)}
}

// ParseLevel maps a LOG_LEVEL string to a slog.Level. Unknown values fall back
// to info.
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// MaskEmail redacts the local part of an email address so it can appear in an
// application log without leaking PII (ADR-018 §4). The domain is retained for
// operational debugging: "alice@example.com" -> "***@example.com". Values that
// are not addresses are fully masked.
func MaskEmail(addr string) string {
	at := strings.LastIndex(addr, "@")
	if at <= 0 || at+1 >= len(addr) {
		return "***"
	}
	return "***@" + addr[at+1:]
}

// defaultLogger is the process-wide Logger. It starts at info level on stdout so
// that code constructed before bootstrap configuration still produces
// structured output; bootstrap replaces it via SetDefault once env is loaded.
var defaultLogger = New(os.Stdout, slog.LevelInfo, "idmagic", "")

// SetDefault installs the process-wide Logger. A nil logger is ignored.
func SetDefault(l Logger) {
	if l != nil {
		defaultLogger = l
	}
}

// Default returns the process-wide Logger.
func Default() Logger { return defaultLogger }

// Package-level convenience wrappers over the default Logger, for adapters that
// are not (yet) constructed with an injected Logger.
func Debug(ctx context.Context, msg string, args ...any) { defaultLogger.Debug(ctx, msg, args...) }
func Info(ctx context.Context, msg string, args ...any)  { defaultLogger.Info(ctx, msg, args...) }
func Warn(ctx context.Context, msg string, args ...any)  { defaultLogger.Warn(ctx, msg, args...) }
func Error(ctx context.Context, msg string, args ...any) { defaultLogger.Error(ctx, msg, args...) }
