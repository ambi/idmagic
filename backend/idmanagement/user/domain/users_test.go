package domain_test

import (
	"testing"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

func validUser() userdomain.User {
	now := time.Now().UTC()
	return userdomain.User{
		ID:                "user_alice",
		PreferredUsername: "alice",
		PasswordHash:      "$argon2id$v=19$m=19456,t=2,p=1$...",
		EmailVerified:     true,
		MfaEnrolled:       false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func TestUserValidateAcceptsMinimumValidShape(t *testing.T) {
	if err := validUser().Validate(); err != nil {
		t.Fatalf("expected valid user, got %v", err)
	}
}

func TestUserValidateRejectsEmptySub(t *testing.T) {
	u := validUser()
	u.ID = ""
	if err := u.Validate(); err == nil {
		t.Fatal("expected error for empty sub")
	}
}

func TestUserValidateRejectsOversizedUsername(t *testing.T) {
	u := validUser()
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'x'
	}
	u.PreferredUsername = string(long)
	if err := u.Validate(); err == nil {
		t.Fatal("expected error for >100-char preferred_username")
	}
}

func TestUserValidateRejectsMalformedEmail(t *testing.T) {
	u := validUser()
	bad := "not-an-email"
	u.Email = &bad
	if err := u.Validate(); err == nil {
		t.Fatal("expected error for malformed email")
	}
}
