package domain

// PasswordPolicyResolver applies a tenant's overrides on top of the global
// defaults. Only the non-nil fields of Tenant.PasswordPolicyOverride override the
// baseline; the rest keep the product-wide value.

import (
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type PasswordPolicySnapshot struct {
	MinLength    int
	MaxLength    int
	HistoryDepth int
	// MaxAgeDays is 0 when expiry is disabled (the default); a positive value is
	// the tenant's opt-in.
	MaxAgeDays int
	// PolicyUpdatedAt is when the tenant last changed its override. It is the
	// lower bound of the expiry reference time (the grace period in
	// REQ-AUTHENTICATION-024).
	PolicyUpdatedAt *time.Time
}

// ResolvePasswordPolicy merges the global defaults with the tenant override.
// A nil tenant, or a tenant with no override, keeps the defaults as they are.
func ResolvePasswordPolicy(tenant *tenancydomain.Tenant, defaults PasswordPolicySnapshot) PasswordPolicySnapshot {
	snapshot := defaults
	if tenant == nil {
		return snapshot
	}
	snapshot.PolicyUpdatedAt = tenant.PasswordPolicyUpdatedAt
	if tenant.PasswordPolicyOverride == nil {
		return snapshot
	}
	o := tenant.PasswordPolicyOverride
	if o.MinLength != nil && *o.MinLength > 0 {
		snapshot.MinLength = *o.MinLength
	}
	if o.MaxLength != nil && *o.MaxLength > 0 {
		snapshot.MaxLength = *o.MaxLength
	}
	if o.HistoryDepth != nil && *o.HistoryDepth > 0 {
		snapshot.HistoryDepth = *o.HistoryDepth
	}
	if o.MaxAgeDays != nil && *o.MaxAgeDays > 0 {
		snapshot.MaxAgeDays = *o.MaxAgeDays
	}
	return snapshot
}

// PasswordExpired reports whether the password has expired under the
// tenant-resolved policy (REQ-AUTHENTICATION-024).
//
// The reference time is the later of password_changed_at and the time the tenant
// last changed its policy. Measuring from password_changed_at alone would expire
// every long-lived password the moment an administrator enables expiry; taking
// the policy update time as a lower bound guarantees every user a full
// max_age_days window after any policy change.
//
// A user with neither reference time (no override and no recorded change) never
// expires: forcing a change there would rest on a missing record rather than on
// elapsed time.
func PasswordExpired(snap PasswordPolicySnapshot, passwordChangedAt *time.Time, now time.Time) bool {
	if snap.MaxAgeDays <= 0 {
		return false
	}
	since, ok := expiryReference(snap.PolicyUpdatedAt, passwordChangedAt)
	if !ok {
		return false
	}
	return now.After(since.AddDate(0, 0, snap.MaxAgeDays))
}

// expiryReference returns the later of the two candidate reference times, or
// ok=false when there is neither.
func expiryReference(policyUpdatedAt, passwordChangedAt *time.Time) (time.Time, bool) {
	switch {
	case policyUpdatedAt == nil && passwordChangedAt == nil:
		return time.Time{}, false
	case policyUpdatedAt == nil:
		return *passwordChangedAt, true
	case passwordChangedAt == nil:
		return *policyUpdatedAt, true
	case passwordChangedAt.After(*policyUpdatedAt):
		return *passwordChangedAt, true
	default:
		return *policyUpdatedAt, true
	}
}
