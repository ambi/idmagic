package bootstrap

import (
	"context"
	"errors"
	"os"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/seeding"
	"github.com/ambi/idmagic/backend/seeding/domain"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// Seed は環境のシークレットの提供元と各所有者の保存先を Seeding に結ぶ。
func Seed(ctx context.Context, deps *Dependencies, request domain.Request, secretRoot string) (domain.Plan, error) {
	materialized, err := seeding.Prepare(request, secretRoot, os.Getenv)
	if err != nil {
		return domain.Plan{}, err
	}
	if deps.Tenancy.QuotaRepo == nil || deps.OAuth2.EventSink == nil || deps.Audit.AuditEventRepo == nil || deps.SeedRequestLock == nil {
		return domain.Plan{}, errors.New("seed quota, event, audit and lock dependencies are required")
	}
	seedDeps := *deps
	if seedDeps.Authentication.PasswordHasher == nil {
		seedDeps.Authentication.PasswordHasher = passwords_argon2id.NewArgon2idPasswordHasher()
	}
	emit := seedDeps.NewEmitFunc(logging.Default()) //nolint:contextcheck // 起動時の配線。監査と通知は操作の終了後にも記録する。
	// 初期データは User と Group を作るだけで、止めるイベントを発行しないので、失効の反応器を通さない。
	audit := func(event spec.DomainEvent) error { emit(event); return nil }
	commands := seedCommands{deps: &seedDeps, identities: seedIdentityCommands(&seedDeps, audit), emit: emit}
	contributor, err := seeding.NewContributor(seeding.ResourceDependencies{
		ClientRepo: deps.OAuth2.ClientRepo,
		UserRepo:   deps.IdManagement.UserRepo, GroupRepo: deps.IdManagement.GroupRepo,
		MfaFactorRepo: deps.Authentication.MfaFactorRepo, PasswordHistoryRepo: deps.Authentication.PasswordHistoryRepo,
		AuthzDetailTypeRepo: deps.OAuth2.AuthzDetailTypeRepo,
		RPRepo:              deps.WsFederation.RPRepo, SPRepo: deps.Saml.SPRepo,
		Applications: deps.Application.Repo, AssignmentRepo: deps.Application.AssignmentRepo,
		Hasher: seedDeps.Authentication.PasswordHasher,
	}, commands, materialized)
	if err != nil {
		return domain.Plan{}, err
	}
	if request.Mode == domain.ModeApply {
		var plan domain.Plan
		err := deps.SeedRequestLock(ctx, request, func(ctx context.Context) error {
			var runErr error
			plan, runErr = seeding.Run(ctx, request, contributor)
			return runErr
		})
		return plan, err
	}
	return seeding.Run(ctx, request, contributor)
}

// 既存のパッケージ内テストが使う名前を保ち、判断は Seeding に委ねる。
func firstPartyClients(seeds []domain.FirstPartyClientSeed, redirectURIs []string, now time.Time) []*oauthdomain.OAuth2Client {
	return domain.FirstPartyClients(seeds, redirectURIs, now)
}

func sameClient(actual, desired *oauthdomain.OAuth2Client) bool {
	return domain.SameFirstPartyClient(actual, desired)
}
