package domain

import (
	"fmt"
	"strings"
	"unicode/utf8"

	z "github.com/Oudwins/zog"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type PasswordPolicyViolation string

const (
	ViolationTooShort PasswordPolicyViolation = "too_short"
	ViolationTooLong  PasswordPolicyViolation = "too_long"
	ViolationBreached PasswordPolicyViolation = "breached"
)

// 設定 API と同じ製品共通の既定値を参照する。パスワードの検証と期限の判定は
// Authentication が所有し、テナントの上書きと既定値の定義は Tenancy が持つ。
const (
	PasswordPolicyMinLength    = tenancydomain.PasswordPolicyMinLength
	PasswordPolicyMaxLength    = tenancydomain.PasswordPolicyMaxLength
	PasswordPolicyHistoryDepth = tenancydomain.PasswordPolicyHistoryDepth
	PasswordPolicyMaxAgeDays   = tenancydomain.PasswordPolicyMaxAgeDays
)

type PasswordPolicyResult struct {
	OK         bool
	Violations []PasswordPolicyViolation
}

// DefaultPasswordPolicySnapshot returns the thresholds that apply with no override.
func DefaultPasswordPolicySnapshot() PasswordPolicySnapshot {
	return PasswordPolicySnapshot{
		MinLength:    PasswordPolicyMinLength,
		MaxLength:    PasswordPolicyMaxLength,
		HistoryDepth: PasswordPolicyHistoryDepth,
		MaxAgeDays:   PasswordPolicyMaxAgeDays,
	}
}

// PolicyForTenant applies the tenant's override to the global defaults. A nil
// tenant gets the global defaults: failing a login or a password change over a
// policy lookup would be out of proportion, and the defaults are never weaker
// than an override.
func PolicyForTenant(tenant *tenancydomain.Tenant) PasswordPolicySnapshot {
	return ResolvePasswordPolicy(tenant, DefaultPasswordPolicySnapshot())
}

func passwordSchemaFor(snap PasswordPolicySnapshot) *z.StringSchema[string] {
	return z.String().
		Required(z.Message(string(ViolationTooShort))).
		TestFunc(
			func(value *string, _ z.Ctx) bool {
				return utf8.RuneCountInString(*value) >= snap.MinLength
			},
			z.Message(string(ViolationTooShort)),
		).
		TestFunc(
			func(value *string, _ z.Ctx) bool {
				return utf8.RuneCountInString(*value) <= snap.MaxLength
			},
			z.Message(string(ViolationTooLong)),
		)
}

// ValidatePassword evaluates against the global defaults, for paths that do not
// need the tenant policy (such as the weak re-check on the login path).
//
// Length counts UTF-8 code points (runes). The TypeScript side counts UTF-16
// code units, so the two differ on surrogates; they agree for the ASCII demo
// passwords.
func ValidatePassword(plain string) PasswordPolicyResult {
	return ValidatePasswordWith(plain, DefaultPasswordPolicySnapshot())
}

// ValidatePasswordWith evaluates against tenant-resolved thresholds, for the
// paths that apply the policy in full such as change-password and
// reset-password.
func ValidatePasswordWith(plain string, snap PasswordPolicySnapshot) PasswordPolicyResult {
	var violations []PasswordPolicyViolation
	for _, issue := range passwordSchemaFor(snap).Validate(&plain) {
		switch PasswordPolicyViolation(issue.Message) {
		case ViolationTooShort:
			violations = append(violations, ViolationTooShort)
		case ViolationTooLong:
			violations = append(violations, ViolationTooLong)
		}
	}
	return PasswordPolicyResult{OK: len(violations) == 0, Violations: violations}
}

type PasswordPolicyError struct {
	Violations []PasswordPolicyViolation
}

// PasswordPolicyViolations は、HTTP の表示に必要なコードだけを所有者の順序で返す。
func (e *PasswordPolicyError) PasswordPolicyViolations() []string {
	codes := make([]string, len(e.Violations))
	for i, violation := range e.Violations {
		codes[i] = string(violation)
	}
	return codes
}

func (e *PasswordPolicyError) Error() string {
	parts := make([]string, len(e.Violations))
	for i, v := range e.Violations {
		parts[i] = string(v)
	}
	return fmt.Sprintf("password policy violated: %s", strings.Join(parts, ", "))
}
