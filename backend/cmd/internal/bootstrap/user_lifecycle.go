package bootstrap

import (
	"github.com/ambi/idmagic/backend/authentication"

	"github.com/ambi/idmagic/backend/idmanagement"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// UserLifecycleDeps は、管理 API の外（LifecycleWorkflow、SCIM の取り込み、保持期間の掃除）から User を止めるときに
// IdManagement の操作へ渡す依存である。管理 API のハンドラーと同じ集合にして、どの経路で止めても
// 同じイベント、端末の失効、下流への通知を伴わせる。
func (d *Dependencies) UserLifecycleDeps() idmanagement.LifecycleDeps {
	return idmanagement.LifecycleDeps{
		AttrSchemaRepo: d.IdManagement.AttrSchemaRepo, QuotaRepo: d.Tenancy.QuotaRepo,
		ConsentRepo: d.OAuth2.ConsentRepo, RefreshStore: d.OAuth2.RefreshStore,
		DeviceCodeStore: d.OAuth2.DeviceCodeStore, ApprovalRequestStore: d.OAuth2.ApprovalRequestStore,
		SessionStore: d.Authentication.SessionStore, UserCredentials: d.Authentication.UserCredentials(),
		TrustedDevices:          d.Authentication.UserDeviceRevoker(),
		ValidateInitialPassword: authentication.ValidateInitialPassword,
		PasswordHasher:          d.Authentication.PasswordHasher, PasswordHistoryRepo: d.Authentication.PasswordHistoryRepo,
	}
}

// UserLifecycleCommands は、管理 API の外（LifecycleWorkflow、SCIM と Directory の取り込み、保持期間の掃除）から
// User を止める操作を組み立てる。イベントを emit へ記録し、止めたイベントでは失効エポックを進める。
// 前進の失敗は誤りとして呼び出し元へ返り、ワークフローの手順や取り込みの要求が失敗する。
func (d *Dependencies) UserLifecycleCommands(emit func(spec.DomainEvent), actor string) idmanagement.UserLifecycleCommands {
	audit := func(event spec.DomainEvent) error {
		emit(event)
		return nil
	}
	return d.IdManagement.UserLifecycleCommands(d.UserLifecycleDeps(), audit, d.AgentRevocationReactor(emit), actor)
}

// AgentRevocationReactor は、Agent と所有者のイベントで失効エポックを進め、外部の受信側へ SET を投影する反応器を
// 組み立てる。派生イベントは emit へ記録する。
func (d *Dependencies) AgentRevocationReactor(emit func(spec.DomainEvent)) userports.AccessRevocationReactor {
	return d.SharedSignals.AgentRevocationReactor(d.IdManagement.AgentRepo, d.SigningKeys.KeyStore, d.Issuer, emit)
}
