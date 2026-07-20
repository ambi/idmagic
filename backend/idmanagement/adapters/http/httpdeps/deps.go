// Package httpdeps holds the IdManagement HTTP layer's Deps type. It is a
// leaf package (no dependency on the feature adapters/http packages) so that
// user/group/agent adapters/http can depend on it without an import cycle
// back to the context-root adapters/http package that wires routes (ADR-130
// Phase 2).
package httpdeps

import (
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	mfaports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	oauthusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	scimports "github.com/ambi/idmagic/backend/scim/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// Deps は identity management HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator

	UserRepo              userports.UserRepository
	GroupRepo             groupports.GroupRepository
	AgentRepo             agentports.AgentRepository
	UserMutationCommitter userports.UserMutationCommitter
	ProvisioningNotifier  userports.ProvisioningNotifier
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
	EmailSender           sharednotification.EmailSender
}

func (d Deps) ConsentDeps() oauthusecases.ConsentDeps {
	return oauthusecases.ConsentDeps{ConsentRepo: d.ConsentRepo, Emit: d.Emit}
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
