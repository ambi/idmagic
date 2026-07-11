package domain_test

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func mustUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("NewUUIDv4: %v", err)
	}
	return id
}

func TestAuthenticationValidateHappyAndFailure(t *testing.T) {
	now := time.Now().UTC()

	validMfa := domain.MfaFactor{UserID: "user_1", Type: spec.MfaFactorWebAuthn, CreatedAt: now}
	// TOTP は secret 必須なので secret 無しは失敗する。
	badMfa := domain.MfaFactor{UserID: "user_1", Type: spec.MfaFactorTOTP, CreatedAt: now}

	validSession := domain.LoginSession{ID: mustUUID(t), UserID: "user_1", AMR: []string{"pwd"}, ACR: "1", ExpiresAt: now}
	badSession := validSession
	badSession.AMR = nil

	validLoginReq := domain.LoginRequest{RequestID: mustUUID(t), Username: "alice", Password: "pw"}
	badLoginReq := domain.LoginRequest{RequestID: "not-a-uuid", Username: "alice", Password: "pw"}

	cases := []struct {
		name    string
		v       interface{ Validate() error }
		wantErr bool
	}{
		{"mfa ok", validMfa, false},
		{"mfa bad", badMfa, true},
		{"session ok", validSession, false},
		{"session bad", badSession, true},
		{"login req ok", validLoginReq, false},
		{"login req bad", badLoginReq, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.v.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("%s: expected error, got nil", c.name)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("%s: expected valid, got %v", c.name, err)
			}
		})
	}
}
