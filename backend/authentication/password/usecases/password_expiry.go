// Package usecases: Layer 3 - Application Logic (password expiry).
package usecases

import (
	"context"
	"slices"
	"time"

	passworddomain "github.com/ambi/idmagic/backend/authentication/password/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// ResolveTenantPolicy returns the global defaults with the request tenant's
// override applied. Every path that sets or validates a password starts here.
//
// The tenant middleware already resolved is used when the context carries it,
// and the repository is only consulted otherwise. A tenant that cannot be
// resolved falls back to the global defaults: failing a login or a password
// change over a policy lookup would be out of proportion, and the defaults are
// never weaker than an override.
func ResolveTenantPolicy(ctx context.Context, repo tenantports.TenantRepository) PasswordPolicySnapshot {
	defaults := DefaultPasswordPolicySnapshot()
	tenant := tenancy.Tenant(ctx)
	if tenant == nil && repo != nil {
		if id := tenancy.TenantID(ctx); id != "" {
			if found, err := repo.FindByID(ctx, id); err == nil {
				tenant = found
			}
		}
	}
	return passworddomain.ResolvePasswordPolicy(tenant, defaults)
}

// ResolvePolicyForTenant builds the policy from an already-resolved tenant, for
// paths that carry no tenant in their context (batch jobs and the like).
func ResolvePolicyForTenant(tenant *tenancydomain.Tenant) PasswordPolicySnapshot {
	return passworddomain.ResolvePasswordPolicy(tenant, DefaultPasswordPolicySnapshot())
}

// EnforcePasswordExpiry gives a user whose password has expired the
// update_password required action, and reports whether it changed the user
// (REQ-AUTHENTICATION-024).
//
// It never fails authentication. Callers invoke it once the login has succeeded
// and, on true, persist the user and emit the event. Users without a password
// credential (federated/passwordless) are excluded, and a user who already has
// the required action is left alone.
func EnforcePasswordExpiry(user *userdomain.User, snap PasswordPolicySnapshot, now time.Time) bool {
	if user == nil || user.PasswordHash == "" {
		return false
	}
	if slices.Contains(user.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword) {
		return false
	}
	if !passworddomain.PasswordExpired(snap, user.Lifecycle.PasswordChangedAt, now) {
		return false
	}
	// Copy into a fresh backing array so the caller's original slice is not shared.
	actions := make([]idmdomain.RequiredAction, 0, len(user.Lifecycle.RequiredActions)+1)
	actions = append(actions, user.Lifecycle.RequiredActions...)
	actions = append(actions, idmdomain.RequiredActionUpdatePassword)
	user.Lifecycle.RequiredActions = actions
	return true
}
