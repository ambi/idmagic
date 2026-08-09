// Package httpdeps holds the IdManagement HTTP layer's Deps type. It is a
// leaf package (no dependency on the feature handlers_http packages) so that
// user/group/agent handlers_http can depend on it without an import cycle
// back to the context-root handlers_http package that wires routes (ADR-130
// Phase 2).
package deps_http

import (
	"context"
	"time"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	mfaports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	consentusecases "github.com/ambi/idmagic/backend/oauth2/consent/usecases"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	scimports "github.com/ambi/idmagic/backend/sourcing/scim/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// EventReactor receives every DomainEvent this context's Emit already emits
// and fail-closed reacts to the ones it cares about. Unlike Emit (fire-and-
// forget, ADR-184/wi-190's reverted transactional-event-log decided against
// making the general audit path error-propagating), a reactor's error
// propagates back through ReactiveEmit. SharedSignals implements this
// shape for Agent revocation epoch enforcement (ADR-057, wi-58) without
// idmanagement importing sharedsignals — only context.Context and
// spec.DomainEvent are named in the signature.
type EventReactor interface {
	React(ctx context.Context, event spec.DomainEvent) error
}

// Deps は identity management HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator

	UserRepo              userports.UserRepository
	GroupRepo             groupports.GroupRepository
	AgentRepo             agentports.AgentRepository
	UserMutationCommitter userports.UserMutationCommitter
	ProvisioningNotifier  userports.ProvisioningNotifier
	// Reactor fail-closed reacts to emitted DomainEvents (currently
	// SharedSignals' Agent revocation epoch enforcement; ADR-057, wi-58).
	// nil skips reaction. See ReactiveEmit.
	Reactor               EventReactor
	JobRepo               jobsports.JobRepository
	ClientRepo            oauthports.OAuth2ClientRepository
	ScimRepo              scimports.ScimRepository
	AttrSchemaRepo        tenantports.TenantUserAttributeSchemaRepository
	ConsentRepo           oauthports.ConsentRepository
	RefreshStore          oauthports.RefreshTokenStore
	DeviceCodeStore       oauthports.DeviceCodeStore
	MfaFactorRepo         mfaports.MfaFactorRepository
	PasswordHasher        passwordports.PasswordHasher
	PasswordHistoryRepo   passwordports.PasswordHistoryRepository
	EmailChangeTokenStore userports.EmailChangeTokenStore
	UserCSVArtifacts      userports.UserCSVArtifactStore
	EmailSender           sharednotification.EmailSender
	Notifier              sharednotification.Notifier
	// QuotaRepo enforces the tenant's Hard Quota on users, groups, and agents
	// (wi-160, ADR-134). nil skips enforcement.
	QuotaRepo tenantports.QuotaRepository
}

func (d Deps) ConsentDeps() consentusecases.ConsentDeps {
	return consentusecases.ConsentDeps{ConsentRepo: d.ConsentRepo, Emit: d.Emit}
}

// LegacyEmit adapts the fire-and-forget support.Deps.Emit to the
// error-returning signature usecases in this context require (wi-184 T003).
// It is the default for handlers not yet migrated to the transaction
// runner; migrated handlers (admin_user_handler.go Create/Update/
// SetDisabled) override deps.Emit with a transaction-bound one instead.
// Exported (unlike its wi-184 origin) so the user/group/agent feature
// packages can call it across the ADR-130 Phase 2 package boundary.
func (d Deps) LegacyEmit() func(spec.DomainEvent) error {
	return func(event spec.DomainEvent) error {
		if d.Emit != nil {
			d.Emit(event)
		}
		return nil
	}
}

// ReactiveEmit composes LegacyEmit's best-effort audit trail with d.Reactor's
// fail-closed reaction: the mutation that emits (KillAgent, SetUserDisabled,
// ...) gets Reactor's error back, so a failed Agent revocation epoch advance
// fails the admin operation instead of being silently swallowed like the
// audit write is (ADR-057, wi-58). d.Reactor uses its own background
// context with a timeout, not the caller's request context, matching
// NewEmitFunc's existing convention of not tying event recording to
// request cancellation.
func (d Deps) ReactiveEmit() func(spec.DomainEvent) error {
	legacy := d.LegacyEmit()
	return func(event spec.DomainEvent) error {
		legacy(event) //nolint:errcheck // LegacyEmit always returns nil (best-effort by design)
		if d.Reactor == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return d.Reactor.React(ctx, event)
	}
}
