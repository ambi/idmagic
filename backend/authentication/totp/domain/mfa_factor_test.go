package domain_test

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/totp/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestMfaFactorValidateHappyAndFailure(t *testing.T) {
	now := time.Now().UTC()

	validMfa := domain.MfaFactor{UserID: "user_1", Type: spec.MfaFactorWebAuthn, CreatedAt: now}
	// TOTP は secret 必須なので secret 無しは失敗する。
	badMfa := domain.MfaFactor{UserID: "user_1", Type: spec.MfaFactorTOTP, CreatedAt: now}

	cases := []struct {
		name    string
		v       interface{ Validate() error }
		wantErr bool
	}{
		{"mfa ok", validMfa, false},
		{"mfa bad", badMfa, true},
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
