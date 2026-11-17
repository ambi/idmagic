// Package httpdeps holds the IdManagement HTTP layer's Deps type. It is a
// leaf package (no dependency on the feature handlers_http packages) so that
// user/group/agent handlers_http can depend on it without an import cycle
// back to the context-root handlers_http package that wires routes (
// Phase 2).
package deps_http

import (
	"context"

	"github.com/labstack/echo/v5"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	idmports "github.com/ambi/idmagic/backend/idmanagement/internal/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// AccountIdentity は、認証の所有者が提供する本人の再認証と、認証エラーの表示である。
type AccountIdentity interface {
	RequireStepUpSub(c *echo.Context) (string, error)
	WriteAuthenticationError(c *echo.Context, err error) (bool, error)
}

// Deps は identity management HTTP ハンドラが必要とする依存。
type Deps struct {
	TenantRepo tenantports.TenantRepository
	support.Deps
	*support.Authenticator

	UserRepo              userports.UserRepository
	GroupRepo             groupports.GroupRepository
	AgentRepo             agentports.AgentRepository
	UserMutationCommitter userports.UserMutationCommitter
	ProvisioningNotifier  userports.ProvisioningNotifier
	// GroupProvisioningNotifier reports committed Group mutations to outbound
	// Provisioning. Separate from ProvisioningNotifier because the two carry
	// different IdManagement-owned trigger vocabularies.
	GroupProvisioningNotifier groupports.ProvisioningNotifier
	// Reactor は User と Agent を止めたイベントでアクセスを失効させる（SharedSignals の失効エポック、wi-58）。
	// 止める操作は監査の Emit と別にこれを呼び、未配線なら失敗する。
	Reactor              userports.AccessRevocationReactor
	JobRepo              jobsports.JobRepository
	JobEnqueuer          jobsports.Enqueuer
	JobCanceler          jobsports.Canceler
	ClientRepo           agentports.ClientLookup
	AgentActivity        agentports.AgentActivityReader
	AgentObservationRepo agentports.AgentObservationRepository
	WorkloadSubjects     agentports.WorkloadSubjectMatcher
	SCIMUsers            userports.SCIMUserLookup
	SCIMGroups           groupports.SCIMGroupLookup
	AttrSchemaRepo       userports.TenantUserAttributeSchemaRepository
	GroupAttrSchemaRepo  groupports.TenantGroupAttributeSchemaRepository
	ConsentRepo          userports.ConsentPurger
	// Consents はデータエクスポートが本人の Granted の同意を読む先である。
	Consents             func(context.Context, string) ([]userports.AccountConsent, error)
	RefreshStore         userports.UserRecordsPurger
	DeviceCodeStore      userports.UserRecordsPurger
	ApprovalRequestStore userports.UserRecordsPurger
	UserCredentials      userports.UserCredentialPurger
	// TrustedDevices は無効化と削除の予約で信頼済みデバイスをすべて失効させる。
	TrustedDevices          userports.UserDeviceRevoker
	ValidateInitialPassword func(context.Context, string) error
	AccountIdentity         AccountIdentity
	PasswordHasher          userports.PasswordHasher
	PasswordHistoryRepo     userports.PasswordHistoryWriter
	// SessionStore は匿名化 cascade がログインセッションを消す先である。nil なら消さない。
	SessionStore          userports.SessionPurger
	EmailChangeTokenStore userports.EmailChangeTokenStore
	CSVArtifacts          idmports.CSVArtifactStore
	EmailSender           sharednotification.EmailSender
	Notifier              sharednotification.Notifier
	// QuotaRepo enforces the tenant's Hard Quota on users, groups, and agents
	// (wi-160). nil skips enforcement.
	QuotaRepo tenantports.QuotaRepository
}

// LegacyEmit adapts the fire-and-forget support.Deps.Emit to the
// error-returning signature usecases in this context require (wi-184 T003).
// It is the default for handlers not yet migrated to the transaction
// runner; migrated handlers (admin_user_handler.go Create/Update/
// SetDisabled) override deps.Emit with a transaction-bound one instead.
// Exported (unlike its wi-184 origin) so the user/group/agent feature
// packages can call it across the Phase 2 package boundary.
func (d Deps) LegacyEmit() func(spec.DomainEvent) error {
	return func(event spec.DomainEvent) error {
		if d.Emit != nil {
			d.Emit(event)
		}
		return nil
	}
}
