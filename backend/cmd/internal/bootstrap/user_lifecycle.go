package bootstrap

import (
	"context"
	"time"

	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sharedsignals/sign_jose"
	sharedsignalsusecases "github.com/ambi/idmagic/backend/sharedsignals/usecases"
)

// UserLifecycleCommands は、管理 API の外（LifecycleWorkflow と SCIM の取り込み）から User を止めるときに通す
// IdManagement の操作を組み立てる。依存は管理 API のハンドラーと同じ集合にして、どの経路で止めても
// 同じイベント、端末の失効、下流への通知、所有する Agent の無効化と失効エポックの前進を伴わせる。
// actor は監査イベントに残る名前である。
//
// 失効エポックの反応は管理 API の ReactiveEmit と同じく fail-closed にする。前進に失敗した無効化は
// 誤りとして呼び出し元へ返り、ワークフローの手順や SCIM の要求が失敗する。
func (d *Dependencies) UserLifecycleCommands(emit func(spec.DomainEvent), actor string) userusecases.UserLifecycleCommands {
	reactor := d.AgentRevocationReactor(emit)
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
				// 管理 API と同じく、要求の context ではなく自前の期限で反応させる。
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				return reactor.React(ctx, event)
			},
		},
		Actor: actor,
	}
}

// AgentRevocationReactor は、Agent と所有者のイベントで失効エポックを進め、外部の受信側へ SET を投影する反応器を
// 組み立てる。派生イベントは emit へ記録する。
func (d *Dependencies) AgentRevocationReactor(emit func(spec.DomainEvent)) *sharedsignalsusecases.AgentRevocationReactor {
	return sharedsignalsusecases.NewAgentRevocationReactor(sharedsignalsusecases.RevocationReactorDeps{
		EpochRepo: d.SharedSignals.RevocationEpochRepo,
		AgentRepo: d.IdManagement.AgentRepo,
		Projector: sharedsignalsusecases.ProjectorDeps{
			StreamRepo: d.SharedSignals.StreamRepo, TransmitterConfigRepo: d.SharedSignals.TransmitterConfigRepo,
			DeliveryRepo: d.SharedSignals.DeliveryRepo, Signer: &sign_jose.Signer{KeyStore: d.SigningKeys.KeyStore}, Issuer: d.Issuer,
		},
		Emit: emit,
	})
}
