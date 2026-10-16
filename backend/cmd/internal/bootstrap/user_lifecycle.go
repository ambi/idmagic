package bootstrap

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// UserLifecycleDeps は、管理 API の外（LifecycleWorkflow、SCIM の取り込み、保持期間の掃除）から User を止めるときに
// IdManagement の操作へ渡す依存である。管理 API のハンドラーと同じ集合にして、どの経路で止めても
// 同じイベント、端末の失効、下流への通知を伴わせる。
func (d *Dependencies) UserLifecycleDeps() idmanagement.LifecycleDeps {
	return idmanagement.LifecycleDeps{
		AttrSchemaRepo: d.Tenancy.AttrSchemaRepo, QuotaRepo: d.Tenancy.QuotaRepo,
		ConsentRepo: d.OAuth2.ConsentRepo, RefreshStore: d.OAuth2.RefreshStore,
		DeviceCodeStore: d.OAuth2.DeviceCodeStore, ApprovalRequestStore: d.OAuth2.ApprovalRequestStore,
		SessionStore: d.Authentication.SessionStore, MfaFactorRepo: d.Authentication.MfaFactorRepo,
		TrustedDeviceRepo:      d.Authentication.TrustedDeviceRepo,
		TrustedDevices:         d.Authentication.UserDeviceRevoker(),
		WebAuthnCredentialRepo: d.Authentication.WebAuthnCredentialRepo, RecoveryCodeRepo: d.Authentication.RecoveryCodeRepo,
		PasswordHasher: d.Authentication.PasswordHasher, PasswordHistoryRepo: d.Authentication.PasswordHistoryRepo,
	}
}

// ReactiveEmit は、イベントを emit へ記録した後に、所有する Agent の無効化と失効エポックの前進を反応させる。
// 失効エポックの反応は管理 API の ReactiveEmit と同じく fail-closed にする。前進に失敗した無効化は
// 誤りとして呼び出し元へ返り、ワークフローの手順や SCIM の要求が失敗する。
func (d *Dependencies) ReactiveEmit(emit func(spec.DomainEvent)) func(spec.DomainEvent) error {
	reactor := d.AgentRevocationReactor(emit)
	return func(event spec.DomainEvent) error {
		emit(event)
		// 管理 API と同じく、要求の context ではなく自前の期限で反応させる。
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return reactor.React(ctx, event)
	}
}

// AgentRevocationReactor は、Agent と所有者のイベントで失効エポックを進め、外部の受信側へ SET を投影する反応器を
// 組み立てる。派生イベントは emit へ記録する。
func (d *Dependencies) AgentRevocationReactor(emit func(spec.DomainEvent)) interface {
	React(context.Context, spec.DomainEvent) error
} {
	return d.SharedSignals.AgentRevocationReactor(d.IdManagement.AgentRepo, d.SigningKeys.KeyStore, d.Issuer, emit)
}
