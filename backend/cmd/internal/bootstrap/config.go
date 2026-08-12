package bootstrap

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// ConfigError is one field-level startup configuration failure: a missing
// required key, a value that fails to parse as its declared type, or a
// value outside its declared range. Key names the environment variable so
// an operator can fix it without reading source.
type ConfigError struct {
	Key     string
	Problem string
}

func (e ConfigError) Error() string { return fmt.Sprintf("%s: %s", e.Key, e.Problem) }

// ConfigErrors aggregates every ConfigError found while loading and
// validating startup configuration (REQ-SYSTEM-016): every problem from one
// startup attempt is reported together, so an operator does not fix one
// typo only to hit the next one on the following restart.
type ConfigErrors []ConfigError

func (e ConfigErrors) Error() string {
	lines := make([]string, len(e))
	for i, err := range e {
		lines[i] = "- " + err.Error()
	}
	return fmt.Sprintf("invalid startup configuration (%d):\n%s", len(e), strings.Join(lines, "\n"))
}

// Secret wraps a configuration value that must never reach an error
// message, log line, or generated reference (DSNs, SMTP credentials, API
// keys, tokens). Only Value() exposes the underlying string; every
// formatting path redacts it, including fmt's %v/%s/%+v (via String) and
// JSON encoding (via MarshalJSON), so a Secret embedded in a larger struct
// stays redacted even when that struct is logged or dumped wholesale.
type Secret struct{ value string }

// NewSecret wraps value as a Secret.
func NewSecret(value string) Secret { return Secret{value: value} }

// Value returns the underlying secret value. Callers must not log, wrap in
// an error, or otherwise let the result reach output visible outside the
// process boundary that consumes it.
func (s Secret) Value() string { return s.value }

// Empty reports whether the secret is unset.
func (s Secret) Empty() bool { return s.value == "" }

func (s Secret) String() string               { return "[REDACTED]" }
func (s Secret) GoString() string             { return "[REDACTED]" }
func (s Secret) MarshalJSON() ([]byte, error) { return []byte(`"[REDACTED]"`), nil }
func (s Secret) MarshalText() ([]byte, error) { return []byte("[REDACTED]"), nil }

// ConfigLoader reads typed startup configuration from an env-like source and
// aggregates every parse or required-value failure instead of silently
// falling back to a default. A single ConfigLoader is meant to be shared
// across every Load*Config call for one process startup, so Err() reflects
// every field read during that startup attempt in one place (REQ-SYSTEM-016).
type ConfigLoader struct {
	getenv func(string) string
	errs   ConfigErrors
}

// NewConfigLoader creates a ConfigLoader reading from getenv (os.Getenv in
// production, a stub map in tests).
func NewConfigLoader(getenv func(string) string) *ConfigLoader {
	return &ConfigLoader{getenv: getenv}
}

func (l *ConfigLoader) fail(key, problem string) {
	l.errs = append(l.errs, ConfigError{Key: key, Problem: problem})
}

// Err returns the aggregated validation error accumulated so far, or nil if
// every field loaded (and every Require call) has succeeded.
func (l *ConfigLoader) Err() error {
	if len(l.errs) == 0 {
		return nil
	}
	return l.errs
}

func (l *ConfigLoader) raw(key string) (string, bool) {
	v := strings.TrimSpace(l.getenv(key))
	return v, v != ""
}

// String returns the trimmed value of key, or fallback when unset.
func (l *ConfigLoader) String(key, fallback string) string {
	if v, ok := l.raw(key); ok {
		return v
	}
	return fallback
}

// RequiredString returns the trimmed value of key, recording a ConfigError
// when it is unset.
func (l *ConfigLoader) RequiredString(key string) string {
	v, ok := l.raw(key)
	if !ok {
		l.fail(key, "is required")
	}
	return v
}

// Secret returns key wrapped as a Secret, redacted-by-default.
func (l *ConfigLoader) Secret(key string) Secret {
	v, _ := l.raw(key)
	return NewSecret(v)
}

// RequiredSecret returns key wrapped as a Secret, recording a ConfigError
// when it is unset. The error message never includes the value.
func (l *ConfigLoader) RequiredSecret(key string) Secret {
	v, ok := l.raw(key)
	if !ok {
		l.fail(key, "is required")
	}
	return NewSecret(v)
}

// Bool parses key as "true"/"false" (case-insensitive), recording a
// ConfigError for any other non-empty value instead of treating it as false.
func (l *ConfigLoader) Bool(key string, fallback bool) bool {
	v, ok := l.raw(key)
	if !ok {
		return fallback
	}
	switch strings.ToLower(v) {
	case "true":
		return true
	case "false":
		return false
	default:
		l.fail(key, fmt.Sprintf("must be true or false, got %q", v))
		return fallback
	}
}

// Int parses key as a base-10 integer, recording a ConfigError on a parse
// failure instead of silently keeping fallback.
func (l *ConfigLoader) Int(key string, fallback int) int {
	v, ok := l.raw(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		l.fail(key, fmt.Sprintf("must be an integer, got %q", v))
		return fallback
	}
	return parsed
}

// PositiveInt is Int plus a > 0 range check.
func (l *ConfigLoader) PositiveInt(key string, fallback int) int {
	parsed := l.Int(key, fallback)
	if parsed <= 0 {
		l.fail(key, fmt.Sprintf("must be positive, got %d", parsed))
		return fallback
	}
	return parsed
}

// NonNegativeInt is Int plus a >= 0 range check.
func (l *ConfigLoader) NonNegativeInt(key string, fallback int) int {
	parsed := l.Int(key, fallback)
	if parsed < 0 {
		l.fail(key, fmt.Sprintf("must not be negative, got %d", parsed))
		return fallback
	}
	return parsed
}

// Int32 parses key as a base-10 32-bit integer.
func (l *ConfigLoader) Int32(key string, fallback int32) int32 {
	v, ok := l.raw(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		l.fail(key, fmt.Sprintf("must be a 32-bit integer, got %q", v))
		return fallback
	}
	return int32(parsed)
}

// NonNegativeInt32 is Int32 plus a >= 0 range check.
func (l *ConfigLoader) NonNegativeInt32(key string, fallback int32) int32 {
	parsed := l.Int32(key, fallback)
	if parsed < 0 {
		l.fail(key, fmt.Sprintf("must not be negative, got %d", parsed))
		return fallback
	}
	return parsed
}

// NonNegativeUint32 parses key as a base-10 unsigned 32-bit integer.
func (l *ConfigLoader) NonNegativeUint32(key string, fallback uint32) uint32 {
	v, ok := l.raw(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		l.fail(key, fmt.Sprintf("must be a non-negative 32-bit integer, got %q", v))
		return fallback
	}
	return uint32(parsed)
}

// Float parses key as a base-10 float.
func (l *ConfigLoader) Float(key string, fallback float64) float64 {
	v, ok := l.raw(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseFloat(v, 64)
	if err != nil {
		l.fail(key, fmt.Sprintf("must be a number, got %q", v))
		return fallback
	}
	return parsed
}

// FloatRange is Float plus a [minVal, maxVal] inclusive range check.
func (l *ConfigLoader) FloatRange(key string, fallback, minVal, maxVal float64) float64 {
	parsed := l.Float(key, fallback)
	if parsed < minVal || parsed > maxVal {
		l.fail(key, fmt.Sprintf("must be between %v and %v, got %v", minVal, maxVal, parsed))
		return fallback
	}
	return parsed
}

// Duration parses key with time.ParseDuration.
func (l *ConfigLoader) Duration(key string, fallback time.Duration) time.Duration {
	v, ok := l.raw(key)
	if !ok {
		return fallback
	}
	parsed, err := time.ParseDuration(v)
	if err != nil {
		l.fail(key, fmt.Sprintf("must be a duration (e.g. \"30s\"), got %q: %v", v, err))
		return fallback
	}
	return parsed
}

// PositiveDuration is Duration plus a > 0 range check.
func (l *ConfigLoader) PositiveDuration(key string, fallback time.Duration) time.Duration {
	parsed := l.Duration(key, fallback)
	if parsed <= 0 {
		l.fail(key, fmt.Sprintf("must be positive, got %s", parsed))
		return fallback
	}
	return parsed
}

// URL returns key (or fallback when unset) and records a ConfigError when
// the non-empty value does not parse as an absolute URL.
func (l *ConfigLoader) URL(key, fallback string) string {
	v := l.String(key, fallback)
	if v == "" {
		return v
	}
	parsed, err := url.Parse(v)
	if err != nil || !parsed.IsAbs() {
		l.fail(key, fmt.Sprintf("must be an absolute URL, got %q", v))
	}
	return v
}

// RequiredURL is URL plus a required-value check.
func (l *ConfigLoader) RequiredURL(key string) string {
	v := l.RequiredString(key)
	if v == "" {
		return v
	}
	parsed, err := url.Parse(v)
	if err != nil || !parsed.IsAbs() {
		l.fail(key, fmt.Sprintf("must be an absolute URL, got %q", v))
	}
	return v
}

// Enum returns key (or fallback when unset) and records a ConfigError when
// the non-empty value is not one of allowed. Comparison is exact on the
// trimmed value; callers normalize case before calling Enum when the field
// is case-insensitive.
func (l *ConfigLoader) Enum(key, fallback string, allowed ...string) string {
	v := l.String(key, fallback)
	if slices.Contains(allowed, v) {
		return v
	}
	l.fail(key, fmt.Sprintf("must be one of %s, got %q", strings.Join(allowed, ", "), v))
	return fallback
}

// Require records a ConfigError attributed to key when ok is false, for
// cross-field checks such as "persistence=postgres requires DATABASE_URL"
// that cannot be expressed as a single field's type or range.
func (l *ConfigLoader) Require(key string, ok bool, problem string) {
	if !ok {
		l.fail(key, problem)
	}
}
