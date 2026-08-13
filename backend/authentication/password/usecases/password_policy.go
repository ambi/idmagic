// Package usecases: Layer 3 - Application Logic (password policy).
//
// The product defaults of the Authentication context's PasswordPolicy are
// declared here. The schema of the tenant overrides and the current values
// returned to the UI (PasswordPolicyDefaults) live in spec/contexts/tenancy;
// letting the two drift apart is a spec-implementation drift.
package usecases

import (
	"fmt"
	"strings"
	"unicode/utf8"

	z "github.com/Oudwins/zog"

	passworddomain "github.com/ambi/idmagic/backend/authentication/password/domain"
)

type PasswordPolicyViolation string

const (
	ViolationTooShort PasswordPolicyViolation = "too_short"
	ViolationTooLong  PasswordPolicyViolation = "too_long"
	ViolationBreached PasswordPolicyViolation = "breached"
)

const (
	PasswordPolicyMinLength    = 12
	PasswordPolicyMaxLength    = 128
	PasswordPolicyHistoryDepth = 5
	// PasswordPolicyMaxAgeDays is the declared default for expiry. 0 means "no
	// expiry", following NIST SP 800-63B-4's discouragement of forced rotation;
	// a tenant opts in through PasswordPolicyOverride.max_age_days
	// (REQ-AUTHENTICATION-024).
	PasswordPolicyMaxAgeDays = 0
)

// PasswordPolicyBreachedCheckEnabled is the declared default for the breached
// password check. It is false, which selects NoopBreachedPasswordChecker; a
// deployment enables the check by swapping the adapter with
// BREACHED_PASSWORD_CHECKER=hibp. The choice of adapter is deployment-wide, not
// per tenant.
const PasswordPolicyBreachedCheckEnabled = false

type PasswordPolicyResult struct {
	OK         bool
	Violations []PasswordPolicyViolation
}

// PasswordPolicySnapshot holds the tenant-resolved thresholds an evaluation
// runs against. The single definition lives in password/domain; this is an alias
// so use cases can name the same type. Defining it twice would mean repacking
// the resolved policy at every layer boundary, and every new field would add one
// more place a conversion can be forgotten.
type PasswordPolicySnapshot = passworddomain.PasswordPolicySnapshot

// DefaultPasswordPolicySnapshot returns the thresholds that apply with no override.
func DefaultPasswordPolicySnapshot() PasswordPolicySnapshot {
	return PasswordPolicySnapshot{
		MinLength:    PasswordPolicyMinLength,
		MaxLength:    PasswordPolicyMaxLength,
		HistoryDepth: PasswordPolicyHistoryDepth,
		MaxAgeDays:   PasswordPolicyMaxAgeDays,
	}
}

// resolveSnapshot prefers an already-resolved snapshot and still supports the
// older callers that pass HistoryDepth on its own. A fully zero snapshot falls
// back to the global defaults, and a positive legacyDepth overrides only
// HistoryDepth.
func resolveSnapshot(snap PasswordPolicySnapshot, legacyDepth int) PasswordPolicySnapshot {
	result := snap
	if result.MinLength == 0 {
		result.MinLength = PasswordPolicyMinLength
	}
	if result.MaxLength == 0 {
		result.MaxLength = PasswordPolicyMaxLength
	}
	if result.HistoryDepth == 0 {
		if legacyDepth > 0 {
			result.HistoryDepth = legacyDepth
		} else {
			result.HistoryDepth = PasswordPolicyHistoryDepth
		}
	}
	return result
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

func (e *PasswordPolicyError) Error() string {
	parts := make([]string, len(e.Violations))
	for i, v := range e.Violations {
		parts[i] = string(v)
	}
	return fmt.Sprintf("password policy violated: %s", strings.Join(parts, ", "))
}
