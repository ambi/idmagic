package idmanagement

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	auditports "github.com/ambi/idmagic/backend/audit/ports"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/internal/agent/db_memory"
	agentpostgres "github.com/ambi/idmagic/backend/idmanagement/internal/agent/db_postgres"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/internal/db_memory"
	idmpostgres "github.com/ambi/idmagic/backend/idmanagement/internal/db_postgres"
	idmhttpdeps "github.com/ambi/idmagic/backend/idmanagement/internal/deps_http"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/internal/group/db_memory"
	grouppostgres "github.com/ambi/idmagic/backend/idmanagement/internal/group/db_postgres"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/internal/group/usecases"
	idmhttp "github.com/ambi/idmagic/backend/idmanagement/internal/handlers_http"
	idmports "github.com/ambi/idmagic/backend/idmanagement/internal/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/internal/user/db_memory"
	userpostgres "github.com/ambi/idmagic/backend/idmanagement/internal/user/db_postgres"
	userhttp "github.com/ambi/idmagic/backend/idmanagement/internal/user/handlers_http"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/internal/user/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// NewMemoryModule は IdManagement が所有する保存先をメモリで作る。User の CSV の取り込みの確定は
// ほかのモジュールの保存先も使うので、WithMemoryUserImport で設定する。
func NewMemoryModule() Module {
	users := usermemory.NewUserRepository()
	groups := groupmemory.NewGroupRepository()
	return Module{
		UserRepo:              users,
		GroupRepo:             groups,
		AgentRepo:             agentmemory.NewAgentRepository(),
		EmailChangeTokenStore: usermemory.NewEmailChangeTokenStore(users),
		CSVArtifacts:          idmmemory.NewCSVArtifactStore(),
		newGroupImportCommitter: func(_ func(pgx.Tx, tenantports.QuotaRepository, func(spec.DomainEvent)) jobsports.Enqueuer, _ func(spec.DomainEvent)) groupports.GroupImportRowCommitter {
			return groupmemory.NewGroupImportRowCommitter(groups)
		},
		GroupMembershipImportCommitter: groupmemory.NewGroupMembershipImportRowCommitter(groups),
		AttrSchemaRepo:                 usermemory.NewTenantUserAttributeSchemaRepository(),
		GroupAttrSchemaRepo:            groupmemory.NewTenantGroupAttributeSchemaRepository(),
	}
}

// WithMemoryUserImport は、User の CSV の取り込みの 1 行を、パスワードの履歴、使用量、監査の記録と
// まとめて確定する committer を設定した Module を返す。
func (m Module) WithMemoryUserImport(
	passwordHistory userports.PasswordHistoryWriter, quota tenantports.QuotaRepository, audit auditports.AuditEventRepository,
) Module {
	m.UserImportCommitter = usermemory.UserImportRowCommitter{
		Users: m.UserRepo, PasswordHistory: passwordHistory, Quota: quota, Audit: audit,
	}
	return m
}

// NewPostgresModule は IdManagement が所有する保存先を PostgreSQL で作る。
// CSV の行の確定は、他モジュールの操作とともに WithPostgresImports で組み立てる。
func NewPostgresModule(pool postgres.DB) Module {
	return Module{
		UserRepo:              &userpostgres.UserRepository{Pool: pool},
		GroupRepo:             &grouppostgres.GroupRepository{Pool: pool},
		AgentRepo:             &agentpostgres.AgentRepository{Pool: pool},
		EmailChangeTokenStore: &userpostgres.EmailChangeTokenStore{Pool: pool},
		CSVArtifacts:          &idmpostgres.CSVArtifactStore{Pool: pool},
		AttrSchemaRepo:        &userpostgres.TenantUserAttributeSchemaRepository{Pool: pool},
		GroupAttrSchemaRepo:   &grouppostgres.TenantGroupAttributeSchemaRepository{Pool: pool},
	}
}

// PostgresImportDeps は、CSV の行と同じ Tx に参加する所有者の操作である。
type PostgresImportDeps struct {
	QuotasInTx  func(pgx.Tx) tenantports.QuotaRepository
	AuditInTx   func(pgx.Tx) auditports.AuditEventWriter
	HistoryInTx func(pgx.Tx) userports.PasswordHistoryWriter
}

// WithPostgresImports は行、使用量、監査と履歴を同じ Tx で確定する CSV の操作を組み立てる。
func (m Module) WithPostgresImports(pool postgres.DB, d PostgresImportDeps) Module {
	m.UserImportCommitter = userpostgres.NewUserImportCommitter(pool, d.QuotasInTx, d.AuditInTx, d.HistoryInTx)
	m.newGroupImportCommitter = func(enqueuersInTx func(pgx.Tx, tenantports.QuotaRepository, func(spec.DomainEvent)) jobsports.Enqueuer, emit func(spec.DomainEvent)) groupports.GroupImportRowCommitter {
		return grouppostgres.NewGroupImportCommitter(pool, d.QuotasInTx, enqueuersInTx, emit, d.AuditInTx)
	}
	m.GroupMembershipImportCommitter = grouppostgres.NewGroupMembershipImportRowCommitter(pool, d.AuditInTx)
	return m
}

// UserWriterInTx は、呼び出し側のトランザクションで User を保存する UserWriter を返す。
// ほかのモジュールの書き込みと User の変更を同じトランザクションで確定させるため、組み立て地点が渡す。
func UserWriterInTx(tx pgx.Tx) userports.UserWriter { return userpostgres.UserWriterInTx(tx) }

// JobHandlerDeps は、CSV の取り込みと書き出し、動的 Group の再計算のジョブが、ほかのモジュールと
// 共通基盤から受け取る依存である。
type JobHandlerDeps struct {
	EnqueuerInTx        func(pgx.Tx, tenantports.QuotaRepository, func(spec.DomainEvent)) jobsports.Enqueuer
	Jobs                jobsports.JobRepository
	AttrSchemaRepo      userports.TenantUserAttributeSchemaRepository
	GroupAttrSchemaRepo groupports.TenantGroupAttributeSchemaRepository
	UserOwnershipGuard  userports.UserSourceOwnershipGuard
	GroupOwnershipGuard groupports.GroupSourceOwnershipGuard
	PasswordHasher      userports.PasswordHasher
	Emit                func(spec.DomainEvent) error
}

// RegisterJobHandlers は、User、Group、所属の CSV の取り込み、データの書き出し、動的 Group の再計算の
// ジョブのハンドラーを登録する。
func (m Module) RegisterJobHandlers(registry jobsports.HandlerRegistrar, d JobHandlerDeps) {
	if d.Emit == nil {
		panic("idmanagement: job handlers require an event sink")
	}
	groupCommitter := m.newGroupImportCommitter(d.EnqueuerInTx, func(event spec.DomainEvent) { _ = d.Emit(event) })
	policy := idmdomain.DefaultCSVTransferPolicy()
	userSchemaReader := userusecases.TenantUserCSVSchemaReader{Repository: d.AttrSchemaRepo}
	dynamicGroups := groupusecases.DynamicGroupDeps{
		GroupRepo: m.GroupRepo, UserRepo: m.UserRepo, SchemaRepo: d.AttrSchemaRepo,
	}

	userPlan := userusecases.UserImportPlanDeps{
		UserRepo: m.UserRepo, SchemaReader: userSchemaReader, OwnershipGuard: d.UserOwnershipGuard,
	}
	userImport := userusecases.UserImportJobDeps{
		Artifacts: m.CSVArtifacts, Jobs: d.Jobs, Plan: userPlan,
		Apply: userusecases.UserImportApplyDeps{
			Plan: userPlan, Committer: m.UserImportCommitter, PasswordHasher: d.PasswordHasher, DynamicGroups: dynamicGroups,
		},
		Policy: policy,
	}
	registry.Register(jobsdomain.KindUserImportPreview, userusecases.UserImportJobHandler(userImport, userusecases.UserImportModePreview))
	registry.Register(jobsdomain.KindUserImportApply, userusecases.UserImportJobHandler(userImport, userusecases.UserImportModeApply))

	groupSchemaReader := groupusecases.TenantGroupCSVSchemaReader{Repository: d.GroupAttrSchemaRepo}
	groupPlan := groupusecases.GroupImportPlanDeps{
		GroupRepo: m.GroupRepo, SchemaRepo: d.AttrSchemaRepo, GroupSchemaReader: groupSchemaReader,
		OwnershipGuard: d.GroupOwnershipGuard,
	}
	groupImport := groupusecases.GroupImportJobDeps{
		Artifacts: m.CSVArtifacts, Jobs: d.Jobs, Plan: groupPlan,
		Apply:  groupusecases.GroupImportApplyDeps{Plan: groupPlan, Committer: groupCommitter},
		Policy: policy,
	}
	registry.Register(jobsdomain.KindGroupImportPreview, groupusecases.GroupImportJobHandler(groupImport, groupusecases.GroupImportModePreview))
	registry.Register(jobsdomain.KindGroupImportApply, groupusecases.GroupImportJobHandler(groupImport, groupusecases.GroupImportModeApply))

	membershipPlan := groupusecases.GroupMembershipImportPlanDeps{
		GroupRepo: m.GroupRepo, UserRepo: m.UserRepo,
		GroupOwnershipGuard: d.GroupOwnershipGuard, UserOwnershipGuard: d.UserOwnershipGuard,
	}
	membershipImport := groupusecases.GroupMembershipImportJobDeps{
		Artifacts: m.CSVArtifacts, Jobs: d.Jobs, Plan: membershipPlan,
		Apply: groupusecases.GroupMembershipImportApplyDeps{
			Plan: membershipPlan, Committer: m.GroupMembershipImportCommitter,
		},
		Policy: policy,
	}
	registry.Register(jobsdomain.KindGroupMembershipImportPreview,
		groupusecases.GroupMembershipImportJobHandler(membershipImport, groupusecases.GroupMembershipImportModePreview))
	registry.Register(jobsdomain.KindGroupMembershipImportApply,
		groupusecases.GroupMembershipImportJobHandler(membershipImport, groupusecases.GroupMembershipImportModeApply))

	reconcile := dynamicGroups
	reconcile.Emit = d.Emit
	registry.Register(jobsdomain.KindDynamicGroupReconcile, groupusecases.DynamicGroupReconcileHandler(reconcile))

	registry.Register(idmusecases.KindDataExport, idmusecases.DataExportHandler(idmusecases.DataExportDeps{
		UserRepo: m.UserRepo, GroupRepo: m.GroupRepo, JobRepo: d.Jobs, CSVArtifacts: m.CSVArtifacts,
		UserCSVExporter: userusecases.UserCSVExporter{
			Deps:   userusecases.UserCSVExportDeps{UserRepo: m.UserRepo, SchemaReader: userSchemaReader, Artifacts: m.CSVArtifacts},
			Policy: policy,
		},
		GroupCSVExporter: groupusecases.GroupCSVExporter{
			Deps:   groupusecases.GroupCSVExportDeps{GroupRepo: m.GroupRepo, SchemaReader: groupSchemaReader, Artifacts: m.CSVArtifacts},
			Policy: policy,
		},
		GroupMembershipCSVExporter: groupusecases.GroupMembershipCSVExporter{
			Deps:   groupusecases.GroupMembershipCSVExportDeps{GroupRepo: m.GroupRepo, UserRepo: m.UserRepo, Artifacts: m.CSVArtifacts},
			Policy: policy,
		},
		Emit: d.Emit,
	}))
}

// LifecycleDeps は、User を止める、削除する操作が、ほかのモジュールから受け取る依存である。
// 管理 API と、管理 API の外（LifecycleWorkflow、SCIM の取り込み、保持期間の掃除）の経路が同じ集合を使う。
type LifecycleDeps struct {
	AttrSchemaRepo          userports.TenantUserAttributeSchemaRepository
	QuotaRepo               tenantports.QuotaRepository
	ConsentRepo             userports.UserRecordsPurger
	RefreshStore            userports.UserRecordsPurger
	DeviceCodeStore         userports.UserRecordsPurger
	ApprovalRequestStore    userports.UserRecordsPurger
	SessionStore            userports.SessionPurger
	UserCredentials         userports.UserCredentialPurger
	TrustedDevices          userports.UserDeviceRevoker
	ValidateInitialPassword func(context.Context, string) error
	PasswordHasher          userports.PasswordHasher
	PasswordHistoryRepo     userports.PasswordHistoryWriter
}

// UserLifecycleCommands は、管理 API の外から User を止める、必須操作を設定する、削除を予約する操作を返す。
// emit は発行したイベントの後に必要な反応まで済ませ、反応の失敗を誤りとして返す。actor は監査イベントに残る名前である。
func (m Module) UserLifecycleCommands(d LifecycleDeps, emit func(spec.DomainEvent) error, actor string) userusecases.UserLifecycleCommands {
	return userusecases.UserLifecycleCommands{Deps: m.adminUserDeps(d, emit), Actor: actor}
}

func (m Module) adminUserDeps(d LifecycleDeps, emit func(spec.DomainEvent) error) userusecases.AdminUserDeps {
	return userusecases.AdminUserDeps{
		UserRepo: m.UserRepo, GroupRepo: m.GroupRepo, AgentRepo: m.AgentRepo,
		AttrSchemaRepo:       d.AttrSchemaRepo,
		ProvisioningNotifier: m.ProvisioningNotifier,
		ConsentRepo:          d.ConsentRepo, RefreshStore: d.RefreshStore,
		DeviceCodeStore: d.DeviceCodeStore, ApprovalRequestStore: d.ApprovalRequestStore,
		SessionStore: d.SessionStore, UserCredentials: d.UserCredentials,
		TrustedDevices:          d.TrustedDevices,
		ValidateInitialPassword: d.ValidateInitialPassword,
		PasswordHasher:          d.PasswordHasher, PasswordHistoryRepo: d.PasswordHistoryRepo,
		QuotaRepo: d.QuotaRepo,
		Emit:      emit,
	}
}

// GroupMembershipCommands は、管理 API と同じ所属のイベントとプロビジョニングの通知を伴って、Group の所属を
// 変える操作を返す。actor は監査イベントに残る名前である。
func (m Module) GroupMembershipCommands(emit func(spec.DomainEvent) error, actor string) groupusecases.GroupMembershipCommands {
	return groupusecases.GroupMembershipCommands{Actor: actor, Deps: groupusecases.AdminGroupDeps{
		GroupRepo: m.GroupRepo, UserRepo: m.UserRepo,
		ProvisioningNotifier: m.GroupProvisioningNotifier,
		Emit:                 emit,
	}}
}

// PurgeExpiredCSVArtifacts は保持期間を過ぎた CSV の成果物を削除し、削除した件数を返す。
// 保存先が期限切れの削除を持たなければ purged を false にする。
func (m Module) PurgeExpiredCSVArtifacts(ctx context.Context, now time.Time) (deleted int64, purged bool, err error) {
	artifacts, ok := m.CSVArtifacts.(idmports.CSVArtifactPurger)
	if !ok {
		return 0, false, nil
	}
	deleted, err = idmusecases.PurgeExpiredCSVArtifacts(ctx, artifacts, now)
	return deleted, true, err
}

// Principals は、管理 API と account API の認証が主体の User と所属 Group を引く先を返す。
func (m Module) Principals() support.PrincipalDirectory {
	return idmhttpdeps.Principals{Users: m.UserRepo, Groups: m.GroupRepo}
}

// RouteDeps は IdManagement の経路が共通基盤とほかのモジュールから受け取る依存である。
type RouteDeps struct {
	TenantRepo      tenantports.TenantRepository
	AccountIdentity idmhttpdeps.AccountIdentity
	LifecycleDeps
	support.Deps
	*support.Authenticator
	// Reactor は管理 API の User と Agent の操作が発行したイベントに反応する。
	Reactor             idmhttpdeps.EventReactor
	ClientRepo          agentports.ClientLookup
	SCIMUsers           userports.SCIMUserLookup
	SCIMGroups          groupports.SCIMGroupLookup
	GroupAttrSchemaRepo groupports.TenantGroupAttributeSchemaRepository
	Consents            func(context.Context, string) ([]userports.AccountConsent, error)
	EmailSender         sharednotification.EmailSender
	Notifier            sharednotification.Notifier
	JobRepo             jobsports.JobRepository
	JobEnqueuer         jobsports.Enqueuer
}

// RegisterRoutes は User、Group、Agent の管理 API と、本人のプロフィールとデータの経路を登録する。
func (m Module) RegisterRoutes(g *echo.Group, d RouteDeps) {
	idmhttp.RegisterRoutes(g, m.httpDeps(d))
}

// FederatedUserInput は、外部の IdP で認証した利用者から作る User の属性である。
type FederatedUserInput = userusecases.ProvisionFederatedUserInput

// FederatedUserProvisioner は、外部の IdP で認証した利用者を、管理 API と同じ規則で User として作る操作を返す。
func (m Module) FederatedUserProvisioner(d RouteDeps) func(context.Context, FederatedUserInput) (*userdomain.User, error) {
	return userhttp.FederatedUserProvisioner(m.httpDeps(d))
}

func (m Module) httpDeps(d RouteDeps) idmhttp.Deps {
	if d.UserCredentials == nil || d.ValidateInitialPassword == nil || d.AccountIdentity == nil {
		panic("identity routes require credential deletion, password validation, and account identity")
	}
	return idmhttp.Deps{
		TenantRepo:                d.TenantRepo,
		Deps:                      d.Deps,
		Authenticator:             d.Authenticator,
		UserRepo:                  m.UserRepo,
		GroupRepo:                 m.GroupRepo,
		AgentRepo:                 m.AgentRepo,
		UserMutationCommitter:     m.UserMutationCommitter,
		ProvisioningNotifier:      m.ProvisioningNotifier,
		GroupProvisioningNotifier: m.GroupProvisioningNotifier,
		Reactor:                   d.Reactor,
		ClientRepo:                d.ClientRepo,
		SCIMUsers:                 d.SCIMUsers,
		SCIMGroups:                d.SCIMGroups,
		AttrSchemaRepo:            d.AttrSchemaRepo,
		GroupAttrSchemaRepo:       d.GroupAttrSchemaRepo,
		ConsentRepo:               d.ConsentRepo,
		Consents:                  d.Consents,
		RefreshStore:              d.RefreshStore,
		DeviceCodeStore:           d.DeviceCodeStore,
		ApprovalRequestStore:      d.ApprovalRequestStore,
		UserCredentials:           d.UserCredentials,
		TrustedDevices:            d.TrustedDevices,
		ValidateInitialPassword:   d.ValidateInitialPassword,
		AccountIdentity:           d.AccountIdentity,
		PasswordHasher:            d.PasswordHasher,
		PasswordHistoryRepo:       d.PasswordHistoryRepo,
		SessionStore:              d.SessionStore,
		EmailChangeTokenStore:     m.EmailChangeTokenStore,
		CSVArtifacts:              m.CSVArtifacts,
		EmailSender:               d.EmailSender,
		Notifier:                  d.Notifier,
		JobRepo:                   d.JobRepo,
		JobEnqueuer:               d.JobEnqueuer,
		QuotaRepo:                 d.QuotaRepo,
	}
}
