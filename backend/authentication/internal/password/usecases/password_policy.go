// Package usecases: Layer 3 - Application Logic (password policy).
//
// The policy's types, defaults, and evaluation live in password/domain. The
// names below are aliases so the use cases keep naming the same values; defining
// them twice would let the two copies drift apart.
package usecases

import (
	passworddomain "github.com/ambi/idmagic/backend/authentication/password/domain"
)

type PasswordPolicyViolation = passworddomain.PasswordPolicyViolation

const (
	ViolationTooShort = passworddomain.ViolationTooShort
	ViolationTooLong  = passworddomain.ViolationTooLong
	ViolationBreached = passworddomain.ViolationBreached
)

const (
	PasswordPolicyMinLength    = passworddomain.PasswordPolicyMinLength
	PasswordPolicyMaxLength    = passworddomain.PasswordPolicyMaxLength
	PasswordPolicyHistoryDepth = passworddomain.PasswordPolicyHistoryDepth
	PasswordPolicyMaxAgeDays   = passworddomain.PasswordPolicyMaxAgeDays
)

// PasswordPolicyBreachedCheckEnabled is the declared default for the breached
// password check. It is false, which selects NoopBreachedPasswordChecker; a
// deployment enables the check by swapping the adapter with
// BREACHED_PASSWORD_CHECKER=hibp. The choice of adapter is deployment-wide, not
// per tenant.
const PasswordPolicyBreachedCheckEnabled = false

type PasswordPolicyResult = passworddomain.PasswordPolicyResult

// PasswordPolicySnapshot holds the tenant-resolved thresholds an evaluation
// runs against.
type PasswordPolicySnapshot = passworddomain.PasswordPolicySnapshot

type PasswordPolicyError = passworddomain.PasswordPolicyError

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
