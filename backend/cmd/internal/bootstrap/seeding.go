package bootstrap

import (
	"context"
	"os"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/seeding"
	"github.com/ambi/idmagic/backend/seeding/domain"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
)

// Seed は環境のシークレットの提供元と各所有者の保存先を Seeding に結ぶ。
func Seed(ctx context.Context, deps *Dependencies, request domain.Request, secretRoot string) (domain.Plan, error) {
	materialized, err := seeding.Prepare(request, secretRoot, os.Getenv)
	if err != nil {
		return domain.Plan{}, err
	}
	contributor := seeding.NewContributor(seeding.ResourceDependencies{
		ClientRepo: deps.OAuth2.ClientRepo,
		UserRepo:   deps.IdManagement.UserRepo, GroupRepo: deps.IdManagement.GroupRepo,
		MfaFactorRepo: deps.Authentication.MfaFactorRepo, PasswordHistoryRepo: deps.Authentication.PasswordHistoryRepo,
		AuthzDetailTypeRepo: deps.OAuth2.AuthzDetailTypeRepo,
		RPRepo:              deps.WsFederation.RPRepo, SPRepo: deps.Saml.SPRepo,
		Applications: deps.Application.Repo, AssignmentRepo: deps.Application.AssignmentRepo,
		Hasher: passwords_argon2id.NewArgon2idPasswordHasher(),
	}, materialized)
	return seeding.Run(ctx, request, contributor)
}

// 既存のパッケージ内テストが使う名前を保ち、判断は Seeding に委ねる。
func firstPartyClients(seeds []domain.FirstPartyClientSeed, redirectURIs []string, now time.Time) []*oauthdomain.OAuth2Client {
	return domain.FirstPartyClients(seeds, redirectURIs, now)
}

func sameClient(actual, desired *oauthdomain.OAuth2Client) bool {
	return domain.SameFirstPartyClient(actual, desired)
}
