package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestConfigLoaderAggregatesEveryError(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{
		"AN_INT":      "not-an-int",
		"A_DURATION":  "not-a-duration",
		"A_URL":       "not a url",
		"AN_ENUM":     "bogus",
		"UNSET_FIELD": "",
	}))
	l.Int("AN_INT", 0)
	l.Duration("A_DURATION", time.Second)
	l.URL("A_URL", "")
	l.Enum("AN_ENUM", "default", "a", "b")
	l.RequiredString("UNSET_FIELD")

	err := l.Err()
	if err == nil {
		t.Fatal("expected an aggregated error")
	}
	for _, key := range []string{"AN_INT", "A_DURATION", "A_URL", "AN_ENUM", "UNSET_FIELD"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("aggregated error %q does not mention %s", err.Error(), key)
		}
	}
	var errs ConfigErrors
	if !errors.As(err, &errs) || len(errs) != 5 {
		t.Fatalf("Err() = %#v, want 5 ConfigErrors", err)
	}
}

func TestConfigLoaderStopsFallingBackSilentlyOnParseFailure(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"TRUSTED_FORWARDED_HOPS": "one"}))
	got := l.NonNegativeInt("TRUSTED_FORWARDED_HOPS", 0)
	if got != 0 {
		t.Errorf("got %d, want the fallback value returned alongside a recorded error", got)
	}
	if err := l.Err(); err == nil || !strings.Contains(err.Error(), "TRUSTED_FORWARDED_HOPS") {
		t.Fatalf("err=%v, want a TRUSTED_FORWARDED_HOPS parse error instead of a silent fallback", err)
	}
}

func TestConfigLoaderRejectsNegativeWhereNonNegativeRequired(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"TRUSTED_FORWARDED_HOPS": "-1"}))
	l.NonNegativeInt("TRUSTED_FORWARDED_HOPS", 0)
	if err := l.Err(); err == nil {
		t.Fatal("expected an error for a negative TRUSTED_FORWARDED_HOPS")
	}
}

func TestConfigLoaderNoErrorsWhenEverythingValid(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"OK": "5"}))
	if got := l.PositiveInt("OK", 1); got != 5 {
		t.Fatalf("got %d, want 5", got)
	}
	if err := l.Err(); err != nil {
		t.Fatalf("Err() = %v, want nil", err)
	}
}

func TestSecretNeverLeaksThroughFormattingOrJSON(t *testing.T) {
	t.Parallel()
	s := NewSecret("super-sensitive-value")

	if got := s.String(); strings.Contains(got, "super-sensitive-value") {
		t.Fatalf("String() leaked the secret: %q", got)
	}
	if got := fmt.Sprintf("%+v", struct{ S Secret }{s}); strings.Contains(got, "super-sensitive-value") {
		t.Fatalf("%%+v on an enclosing struct leaked the secret: %q", got)
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(b), "super-sensitive-value") {
		t.Fatalf("json.Marshal leaked the secret: %q", b)
	}
	if s.Value() != "super-sensitive-value" {
		t.Fatal("Value() must still return the underlying secret for the one caller that needs it")
	}
}

func TestSecretEmpty(t *testing.T) {
	t.Parallel()
	if !NewSecret("").Empty() {
		t.Fatal("empty secret must report Empty() == true")
	}
	if NewSecret("x").Empty() {
		t.Fatal("non-empty secret must report Empty() == false")
	}
}
