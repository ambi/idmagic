package bootstrap

import (
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// UserLifecycleCommands は、管理 API の外（LifecycleWorkflow と SCIM の取り込み）から User を止めるときに通す
// IdManagement の操作を組み立てる。依存は管理 API のハンドラーと同じ集合にして、どの経路で止めても
// 同じイベント、端末の失効、下流への通知、所有する Agent の無効化を伴わせる。actor は監査イベントに残る名前である。
func (d *Dependencies) UserLifecycleCommands(emit func(spec.DomainEvent), actor string) userusecases.UserLifecycleCommands {
	return userusecases.UserLifecycleCommands{
		Deps: userusecases.AdminUserDeps{
			UserRepo: d.IdManagement.UserRepo, GroupRepo: d.IdManagement.GroupRepo, AgentRepo: d.IdManagement.AgentRepo,
			AttrSchemaRepo:       d.Tenancy.AttrSchemaRepo,
			ProvisioningNotifier: d.IdManagement.ProvisioningNotifier,
			ConsentRepo:          d.OAuth2.ConsentRepo, RefreshStore: d.OAuth2.RefreshStore,
			DeviceCodeStore: d.OAuth2.DeviceCodeStore, ApprovalRequestStore: d.OAuth2.ApprovalRequestStore,
			SessionStore: d.Authentication.SessionStore, MfaFactorRepo: d.Authentication.MfaFactorRepo,
			TrustedDeviceRepo:      d.Authentication.TrustedDeviceRepo,
			WebAuthnCredentialRepo: d.Authentication.WebAuthnCredentialRepo, RecoveryCodeRepo: d.Authentication.RecoveryCodeRepo,
			PasswordHasher: d.Authentication.PasswordHasher, PasswordHistoryRepo: d.Authentication.PasswordHistoryRepo,
			QuotaRepo: d.Tenancy.QuotaRepo,
			Emit: func(event spec.DomainEvent) error {
				emit(event)
				return nil
			},
		},
		Actor: actor,
	}
}
