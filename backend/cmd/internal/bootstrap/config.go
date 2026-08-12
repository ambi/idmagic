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

// Field type names used by ConfigField.Type and rendered verbatim into the
// generated ConfigurationReference.
const (
	fieldTypeString   = "string"
	fieldTypeSecret   = "secret"
	fieldTypeBool     = "boolean"
	fieldTypeInt      = "integer"
	fieldTypeFloat    = "number"
	fieldTypeDuration = "duration"
	fieldTypeURL      = "url"
	fieldTypeEnum     = "enum"
	fieldTypeEnumList = "enum list"
	fieldTypeList     = "list"
)

// ConfigField is the metadata ConfigLoader records for every key it reads.
// It carries exactly what an operator needs to set the key — type, default,
// whether it is required, its allowed values or range — so the
// ConfigurationReference is generated from the same calls that parse the
// value and cannot drift from what a process actually reads
// (REQ-SYSTEM-017). Secret fields record no Default: their value never
// leaves the process.
type ConfigField struct {
	Key        string
	Type       string
	Default    string
	Constraint string
	Allowed    []string
	Required   bool
	// RequiredWhen names the condition that makes an otherwise optional
	// field mandatory, for example PERSISTENCE=postgres. It is metadata for
	// the generated reference; runtime enforcement remains in Required* or
	// Require calls beside the Config definition.
	RequiredWhen string
	Secret       bool
}

// ConfigLoader reads typed startup configuration from an env-like source and
// aggregates every parse or required-value failure instead of silently
// falling back to a default. A single ConfigLoader is meant to be shared
// across every Load*Config call for one process startup, so Err() reflects
// every field read during that startup attempt in one place (REQ-SYSTEM-016).
type ConfigLoader struct {
	getenv func(string) string
	errs   ConfigErrors
	fields []ConfigField
	index  map[string]int
}

// NewConfigLoader creates a ConfigLoader reading from getenv (os.Getenv in
// production, a stub map in tests).
func NewConfigLoader(getenv func(string) string) *ConfigLoader {
	return &ConfigLoader{getenv: getenv, index: map[string]int{}}
}

func (l *ConfigLoader) fail(key, problem string) {
	l.errs = append(l.errs, ConfigError{Key: key, Problem: problem})
}

// record registers key's metadata, merging it with an earlier read of the
// same key: a field read twice (once unconditionally, once as required
// under a cross-field condition) is one row in the reference, and any read
// that required it makes it required.
func (l *ConfigLoader) record(f ConfigField) {
	if i, ok := l.index[f.Key]; ok {
		existing := &l.fields[i]
		existing.Required = existing.Required || f.Required
		if existing.RequiredWhen == "" {
			existing.RequiredWhen = f.RequiredWhen
		}
		existing.Secret = existing.Secret || f.Secret
		if existing.Default == "" {
			existing.Default = f.Default
		}
		if existing.Constraint == "" {
			existing.Constraint = f.Constraint
		}
		if len(existing.Allowed) == 0 {
			existing.Allowed = f.Allowed
		}
		return
	}
	l.index[f.Key] = len(l.fields)
	l.fields = append(l.fields, f)
}

// constrain attaches a range constraint to an already recorded key.
func (l *ConfigLoader) constrain(key, constraint string) {
	if i, ok := l.index[key]; ok {
		l.fields[i].Constraint = constraint
	}
}

// retype narrows an already recorded key's type, for accessors layered on a
// more general one (URL and Enum both read a String first).
func (l *ConfigLoader) retype(key, fieldType string) {
	if i, ok := l.index[key]; ok {
		l.fields[i].Type = fieldType
	}
}

// allow attaches the accepted values to an already recorded key.
func (l *ConfigLoader) allow(key string, allowed []string) {
	if i, ok := l.index[key]; ok && len(allowed) > 0 {
		l.fields[i].Allowed = allowed
	}
}

// RequiredWhen records the condition under which key is required. Call it
// beside the corresponding cross-field validation so the generated
// ConfigurationReference reports conditional requirements without needing
// to execute every possible configuration combination (REQ-SYSTEM-017).
func (l *ConfigLoader) RequiredWhen(key, condition string) {
	if i, ok := l.index[key]; ok {
		l.fields[i].RequiredWhen = condition
	}
}

// Fields returns every key read through this loader, in first-read order.
// The ConfigurationReference generator loads each process's config into its
// own loader and renders these rows (REQ-SYSTEM-017).
func (l *ConfigLoader) Fields() []ConfigField { return slices.Clone(l.fields) }

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
	l.record(ConfigField{Key: key, Type: fieldTypeString, Default: fallback})
	if v, ok := l.raw(key); ok {
		return v
	}
	return fallback
}

// RequiredString returns the trimmed value of key, recording a ConfigError
// when it is unset.
func (l *ConfigLoader) RequiredString(key string) string {
	l.record(ConfigField{Key: key, Type: fieldTypeString, Required: true})
	v, ok := l.raw(key)
	if !ok {
		l.fail(key, "is required")
	}
	return v
}

// StringList returns key split on commas with each element trimmed, or
// fallback when unset.
func (l *ConfigLoader) StringList(key string, fallback []string) []string {
	l.record(ConfigField{Key: key, Type: fieldTypeList, Default: strings.Join(fallback, ",")})
	v, ok := l.raw(key)
	if !ok {
		return fallback
	}
	return splitAndTrim(v)
}

// Secret returns key wrapped as a Secret, redacted-by-default.
func (l *ConfigLoader) Secret(key string) Secret {
	l.record(ConfigField{Key: key, Type: fieldTypeSecret, Secret: true})
	v, _ := l.raw(key)
	return NewSecret(v)
}

// RequiredSecret returns key wrapped as a Secret, recording a ConfigError
// when it is unset. The error message never includes the value.
func (l *ConfigLoader) RequiredSecret(key string) Secret {
	l.record(ConfigField{Key: key, Type: fieldTypeSecret, Secret: true, Required: true})
	v, ok := l.raw(key)
	if !ok {
		l.fail(key, "is required")
	}
	return NewSecret(v)
}

// Bool parses key as "true"/"false" (case-insensitive), recording a
// ConfigError for any other non-empty value instead of treating it as false.
func (l *ConfigLoader) Bool(key string, fallback bool) bool {
	l.record(ConfigField{
		Key: key, Type: fieldTypeBool,
		Default: strconv.FormatBool(fallback), Allowed: []string{"true", "false"},
	})
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
	l.record(ConfigField{Key: key, Type: fieldTypeInt, Default: strconv.Itoa(fallback)})
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
	l.constrain(key, "> 0")
	if parsed <= 0 {
		l.fail(key, fmt.Sprintf("must be positive, got %d", parsed))
		return fallback
	}
	return parsed
}

// NonNegativeInt is Int plus a >= 0 range check.
func (l *ConfigLoader) NonNegativeInt(key string, fallback int) int {
	parsed := l.Int(key, fallback)
	l.constrain(key, ">= 0")
	if parsed < 0 {
		l.fail(key, fmt.Sprintf("must not be negative, got %d", parsed))
		return fallback
	}
	return parsed
}

// Int32 parses key as a base-10 32-bit integer.
func (l *ConfigLoader) Int32(key string, fallback int32) int32 {
	l.record(ConfigField{Key: key, Type: fieldTypeInt, Default: strconv.FormatInt(int64(fallback), 10)})
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
	l.constrain(key, ">= 0")
	if parsed < 0 {
		l.fail(key, fmt.Sprintf("must not be negative, got %d", parsed))
		return fallback
	}
	return parsed
}

// NonNegativeUint32 parses key as a base-10 unsigned 32-bit integer.
func (l *ConfigLoader) NonNegativeUint32(key string, fallback uint32) uint32 {
	l.record(ConfigField{
		Key: key, Type: fieldTypeInt,
		Default: strconv.FormatUint(uint64(fallback), 10), Constraint: ">= 0",
	})
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
	l.record(ConfigField{
		Key: key, Type: fieldTypeFloat,
		Default: strconv.FormatFloat(fallback, 'g', -1, 64),
	})
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
	l.constrain(key, fmt.Sprintf("%v..%v", minVal, maxVal))
	if parsed < minVal || parsed > maxVal {
		l.fail(key, fmt.Sprintf("must be between %v and %v, got %v", minVal, maxVal, parsed))
		return fallback
	}
	return parsed
}

// Duration parses key with time.ParseDuration.
func (l *ConfigLoader) Duration(key string, fallback time.Duration) time.Duration {
	l.record(ConfigField{Key: key, Type: fieldTypeDuration, Default: fallback.String()})
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
	l.constrain(key, "> 0")
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
	l.retype(key, fieldTypeURL)
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
	l.retype(key, fieldTypeURL)
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
	l.retype(key, fieldTypeEnum)
	l.allow(key, allowed)
	if slices.Contains(allowed, v) {
		return v
	}
	l.fail(key, fmt.Sprintf("must be one of %s, got %q", strings.Join(allowed, ", "), v))
	return fallback
}

// EnumFold is Enum with case-insensitive input, returning the normalized
// lower-case value. It preserves existing configuration compatibility while
// still rejecting unknown selectors instead of silently choosing an adapter.
func (l *ConfigLoader) EnumFold(key, fallback string, allowed ...string) string {
	raw := l.String(key, fallback)
	l.retype(key, fieldTypeEnum)
	l.allow(key, allowed)
	v := strings.ToLower(raw)
	if slices.Contains(allowed, v) {
		return v
	}
	l.fail(key, fmt.Sprintf("must be one of %s, got %q", strings.Join(allowed, ", "), raw))
	return fallback
}

// OptionalEnum is Enum for a setting whose empty value means "disabled".
// Empty is not rendered as an allowed value, while any non-empty unknown
// value is retained in the returned Config only so related validation can
// still aggregate; callers must not use the Config when Err() is non-nil.
func (l *ConfigLoader) OptionalEnum(key string, allowed ...string) string {
	v := l.String(key, "")
	l.retype(key, fieldTypeEnum)
	l.allow(key, allowed)
	if v == "" || slices.Contains(allowed, v) {
		return v
	}
	l.fail(key, fmt.Sprintf("must be one of %s, got %q", strings.Join(allowed, ", "), v))
	return v
}

// OptionalEnumFold is OptionalEnum with case-insensitive input.
func (l *ConfigLoader) OptionalEnumFold(key string, allowed ...string) string {
	raw := l.String(key, "")
	l.retype(key, fieldTypeEnum)
	l.allow(key, allowed)
	v := strings.ToLower(raw)
	if v == "" || slices.Contains(allowed, v) {
		return v
	}
	l.fail(key, fmt.Sprintf("must be one of %s, got %q", strings.Join(allowed, ", "), raw))
	return v
}

// EnumList returns key split on commas with each element trimmed (or
// fallback when unset), recording a ConfigError for any element that is not
// one of allowed.
func (l *ConfigLoader) EnumList(key string, fallback []string, allowed ...string) []string {
	v := l.StringList(key, fallback)
	l.retype(key, fieldTypeEnumList)
	l.allow(key, allowed)
	for _, element := range v {
		if !slices.Contains(allowed, element) {
			l.fail(key, fmt.Sprintf("must contain only %s, got %q", strings.Join(allowed, ", "), element))
		}
	}
	return v
}

// Require records a ConfigError attributed to key when ok is false, for
// cross-field checks such as "persistence=postgres requires DATABASE_URL"
// that cannot be expressed as a single field's type or range.
func (l *ConfigLoader) Require(key string, ok bool, problem string) {
	if !ok {
		l.fail(key, problem)
	}
}
