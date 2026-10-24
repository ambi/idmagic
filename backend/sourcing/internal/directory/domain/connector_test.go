package domain

import (
	"errors"
	"testing"
	"time"
)

var created = time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)

func awaiting(t *testing.T) Connector {
	t.Helper()
	c, err := NewConnector("tenant", "connector", "  東京の AD  ", "hash-1", created)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

var key = map[string]any{"kty": "EC", "crv": "P-256", "x": "x", "y": "y"}

//spec:covers REQ-SOURCING-008
func TestNewConnectorAwaitsEnrollmentForAnHour(t *testing.T) {
	c := awaiting(t)
	if c.Status != StatusAwaitingEnrollment || c.DisplayName != "東京の AD" || c.EnrollmentTokenHash != "hash-1" {
		t.Fatalf("connector = %+v", c)
	}
	if want := created.Add(time.Hour); !c.EnrollmentExpiresAt.Equal(want) {
		t.Fatalf("expires = %v, want %v", c.EnrollmentExpiresAt, want)
	}
	if c.Revision != 1 {
		t.Fatalf("revision = %d", c.Revision)
	}
}

//spec:covers REQ-SOURCING-008
func TestNewConnectorRejectsAnUnusableDisplayName(t *testing.T) {
	if _, err := NewConnector("tenant", "connector", " \t ", "hash", created); !errors.Is(err, ErrDisplayNameRequired) {
		t.Fatalf("blank name error = %v", err)
	}
	long := make([]rune, 201)
	for i := range long {
		long[i] = 'あ'
	}
	if _, err := NewConnector("tenant", "connector", string(long), "hash", created); err == nil || errors.Is(err, ErrDisplayNameRequired) {
		t.Fatalf("long name error = %v", err)
	}
	if _, err := NewConnector("tenant", "connector", string(long[:200]), "hash", created); err != nil {
		t.Fatalf("200 characters error = %v", err)
	}
}

//spec:covers REQ-SOURCING-009
func TestEnrollActivatesWithTheKeyWithinTheTokenLifetime(t *testing.T) {
	c := awaiting(t)
	at := created.Add(time.Hour)
	enrolled, err := c.Enroll(key, "kid", at)
	if err != nil {
		t.Fatal(err)
	}
	if enrolled.Status != StatusActive || enrolled.KeyID != "kid" || enrolled.PublicKey["kty"] != "EC" ||
		enrolled.EnrollmentTokenHash != "" || enrolled.EnrolledAt == nil || !enrolled.EnrolledAt.Equal(at) ||
		enrolled.Revision != c.Revision+1 {
		t.Fatalf("enrolled = %+v", enrolled)
	}
	if _, err := c.Enroll(key, "kid", at.Add(time.Nanosecond)); !errors.Is(err, ErrEnrollmentUnavailable) {
		t.Fatalf("expired token error = %v", err)
	}
	if _, err := enrolled.Enroll(key, "kid", at); !errors.Is(err, ErrEnrollmentUnavailable) {
		t.Fatalf("active connector error = %v", err)
	}
	disabled, _ := c.Disable(at)
	if _, err := disabled.Enroll(key, "kid", at); !errors.Is(err, ErrEnrollmentUnavailable) {
		t.Fatalf("disabled connector error = %v", err)
	}
}

//spec:covers REQ-SOURCING-011
func TestReissueDiscardsTheKeyAndAwaitsEnrollment(t *testing.T) {
	enrolled, err := awaiting(t).Enroll(key, "kid", created)
	if err != nil {
		t.Fatal(err)
	}
	at := created.Add(2 * time.Hour)
	reissued, err := enrolled.ReissueEnrollment("hash-2", at)
	if err != nil {
		t.Fatal(err)
	}
	if reissued.Status != StatusAwaitingEnrollment || reissued.PublicKey != nil || reissued.KeyID != "" ||
		reissued.EnrolledAt != nil || reissued.EnrollmentTokenHash != "hash-2" ||
		!reissued.EnrollmentExpiresAt.Equal(at.Add(time.Hour)) || reissued.Revision != enrolled.Revision+1 {
		t.Fatalf("reissued = %+v", reissued)
	}
	again, err := reissued.ReissueEnrollment("hash-3", at)
	if err != nil || again.EnrollmentTokenHash != "hash-3" {
		t.Fatalf("reissue while awaiting = %+v, %v", again, err)
	}
	disabled, _ := reissued.Disable(at)
	if _, err := disabled.ReissueEnrollment("hash-4", at); !errors.Is(err, ErrConnectorDisabled) {
		t.Fatalf("disabled reissue error = %v", err)
	}
}

//spec:covers REQ-SOURCING-012
func TestDisableIsTerminalAndIdempotent(t *testing.T) {
	for _, c := range []Connector{awaiting(t), mustEnroll(t, awaiting(t))} {
		at := created.Add(time.Minute)
		disabled, changed := c.Disable(at)
		if !changed || disabled.Status != StatusDisabled || disabled.EnrollmentTokenHash != "" ||
			disabled.DisabledAt == nil || !disabled.DisabledAt.Equal(at) || disabled.Revision != c.Revision+1 {
			t.Fatalf("disabled = %+v changed=%v", disabled, changed)
		}
		again, changed := disabled.Disable(at.Add(time.Hour))
		if changed || again.Revision != disabled.Revision || !again.DisabledAt.Equal(at) {
			t.Fatalf("disable again = %+v changed=%v", again, changed)
		}
	}
}

func mustEnroll(t *testing.T, c Connector) Connector {
	t.Helper()
	enrolled, err := c.Enroll(key, "kid", created)
	if err != nil {
		t.Fatal(err)
	}
	return enrolled
}
