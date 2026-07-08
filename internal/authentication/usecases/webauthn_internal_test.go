package usecases

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/ambi/idmagic/internal/shared/spec"
)

func TestWebAuthnCredentialConversionRoundTrip(t *testing.T) {
	aaguid := base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3, 4})
	original := &spec.WebAuthnCredential{
		CredentialID:   base64.RawURLEncoding.EncodeToString([]byte("credential-id-bytes")),
		UserID:         "user-alice",
		PublicKey:      base64.RawURLEncoding.EncodeToString([]byte("cose-public-key")),
		SignCount:      42,
		Transports:     []string{"internal", "hybrid"},
		AAGUID:         &aaguid,
		BackupEligible: true,
		BackupState:    true,
	}

	converted, err := toWebAuthnCredential(original)
	if err != nil {
		t.Fatal(err)
	}
	label := "My Passkey"
	back := fromWebAuthnCredential("user-alice", &converted, &label, time.Now())

	if back.CredentialID != original.CredentialID {
		t.Fatalf("credential_id=%q, want %q", back.CredentialID, original.CredentialID)
	}
	if back.PublicKey != original.PublicKey {
		t.Fatalf("public_key mismatch")
	}
	if back.SignCount != original.SignCount {
		t.Fatalf("sign_count=%d, want %d", back.SignCount, original.SignCount)
	}
	if len(back.Transports) != 2 || back.Transports[0] != "internal" || back.Transports[1] != "hybrid" {
		t.Fatalf("transports=%v", back.Transports)
	}
	if back.AAGUID == nil || *back.AAGUID != aaguid {
		t.Fatalf("aaguid=%v, want %q", back.AAGUID, aaguid)
	}
	if !back.BackupEligible || !back.BackupState {
		t.Fatalf("backup flags lost: BE=%v BS=%v", back.BackupEligible, back.BackupState)
	}
	if back.Label == nil || *back.Label != label {
		t.Fatalf("label=%v, want %q", back.Label, label)
	}
}

func TestSanitizeWebAuthnLabel(t *testing.T) {
	if got := sanitizeWebAuthnLabel(nil); got != nil {
		t.Fatalf("nil label -> %v", got)
	}
	blank := "   "
	if got := sanitizeWebAuthnLabel(&blank); got != nil {
		t.Fatalf("blank label -> %v", got)
	}
	value := "  Touch ID  "
	got := sanitizeWebAuthnLabel(&value)
	if got == nil || *got != "Touch ID" {
		t.Fatalf("trimmed label=%v", got)
	}
}
