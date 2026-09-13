package usecases

import (
	"strings"
	"testing"
)

func TestValidatePasswordAcceptsMinimumLength(t *testing.T) {
	got := ValidatePassword(strings.Repeat("x", PasswordPolicyMinLength))
	if !got.OK {
		t.Fatalf("expected OK at min length, got %+v", got)
	}
}

//spec:covers REQ-AUTHENTICATION-010, EX-AUTHENTICATION-010-02: 12 文字未満の新しいパスワードが too_short として拒否されることを固定する。
func TestValidatePasswordRejectsTooShort(t *testing.T) {
	got := ValidatePassword(strings.Repeat("x", PasswordPolicyMinLength-1))
	if got.OK || len(got.Violations) != 1 || got.Violations[0] != ViolationTooShort {
		t.Fatalf("expected too_short, got %+v", got)
	}
}

func TestValidatePasswordAcceptsMaximumLength(t *testing.T) {
	got := ValidatePassword(strings.Repeat("x", PasswordPolicyMaxLength))
	if !got.OK {
		t.Fatalf("expected OK at max length, got %+v", got)
	}
}

func TestValidatePasswordRejectsTooLong(t *testing.T) {
	got := ValidatePassword(strings.Repeat("x", PasswordPolicyMaxLength+1))
	if got.OK || len(got.Violations) != 1 || got.Violations[0] != ViolationTooLong {
		t.Fatalf("expected too_long, got %+v", got)
	}
}

func TestValidatePasswordRejectsEmpty(t *testing.T) {
	got := ValidatePassword("")
	if got.OK || len(got.Violations) != 1 || got.Violations[0] != ViolationTooShort {
		t.Fatalf("expected too_short for empty, got %+v", got)
	}
}
