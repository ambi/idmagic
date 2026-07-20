package bootstrap

import (
	"context"
	"crypto/sha256"
	"fmt"
	"reflect"
	"time"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/seeding/domain"
	seedusecases "github.com/ambi/idmagic/backend/seeding/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// Seed は runtime composition から Seeding usecase へ既存 record context の port を接続する。
func Seed(ctx context.Context, deps *Dependencies, request domain.Request) (domain.Plan, error) {
	return seedusecases.Run(ctx, request, &seedContributor{deps: deps, hasher: crypto.NewArgon2idPasswordHasher()})
}

type seedContributor struct {
	deps   *Dependencies
	hasher passwordports.PasswordHasher
}

func firstPartyClients(redirectURIs []string, now time.Time) []*oauthdomain.OAuth2Client {
	clients := []struct{ id, name, scope string }{
		{seedAdminConsoleClientID, "IdMagic Admin Console", "openid profile idmagic.admin offline_access"},
		{seedAccountPortalClientID, "IdMagic Account Portal", "openid profile idmagic.account offline_access"},
	}
	result := make([]*oauthdomain.OAuth2Client, 0, len(clients))
	for _, client := range clients {
		name := client.name
		result = append(result, &oauthdomain.OAuth2Client{
			TenantID: tenancydomain.DefaultTenantID, ClientID: client.id, ClientName: &name,
			ClientType: spec.ClientPublic, RedirectURIs: redirectURIs,
			GrantTypes:    []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
			ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode}, TokenEndpointAuthMethod: oauthdomain.AuthMethodNone,
			Scope: client.scope, IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
			FapiProfile: oauthdomain.FapiNone, FirstParty: true, CreatedAt: now, UpdatedAt: now,
		})
	}
	return result
}

func ensureClient(ctx context.Context, repo interface {
	FindByID(context.Context, string, string) (*oauthdomain.OAuth2Client, error)
	Save(context.Context, *oauthdomain.OAuth2Client) error
}, desired *oauthdomain.OAuth2Client, apply bool,
) (domain.Operation, error) {
	existing, err := repo.FindByID(ctx, desired.TenantID, desired.ClientID)
	if err != nil {
		return domain.Operation{}, err
	}
	op := domain.Operation{LogicalKey: "oauth2-client:" + desired.ClientID, Summary: "first-party client"}
	if existing == nil {
		op.Kind = domain.OperationCreate
		if apply {
			err = repo.Save(ctx, desired)
		}
		return op, err
	}
	if sameClient(existing, desired) {
		op.Kind = domain.OperationNoop
		return op, nil
	}
	op.Kind = domain.OperationConflict
	return op, nil
}

func sameClient(actual, desired *oauthdomain.OAuth2Client) bool {
	left, right := *actual, *desired
	left.CreatedAt, left.UpdatedAt = time.Time{}, time.Time{}
	right.CreatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}

func (c *seedContributor) Plan(ctx context.Context, request domain.Request) (domain.Plan, error) {
	operations, err := c.operations(ctx, request, false)
	return domain.Plan{Operations: operations}, err
}

func (c *seedContributor) Apply(ctx context.Context, request domain.Request) error {
	_, err := c.operations(ctx, request, true)
	return err
}

func (c *seedContributor) operations(ctx context.Context, request domain.Request, apply bool) ([]domain.Operation, error) {
	now := time.Now().UTC()
	redirectURIs := request.FirstPartyRedirectURIs
	if len(redirectURIs) == 0 {
		redirectURIs = []string{"http://localhost:3000/callback", "http://localhost:5173/callback", "http://localhost:8080/callback"}
	}
	operations := make([]domain.Operation, 0, 16)
	appendOperation := func(operation domain.Operation, err error) error {
		if err != nil {
			return err
		}
		operations = append(operations, operation)
		if operation.Kind == domain.OperationConflict {
			return fmt.Errorf("seed drift at %s", operation.LogicalKey)
		}
		return nil
	}
	for _, client := range firstPartyClients(redirectURIs, now) {
		if err := appendOperation(ensureClient(ctx, c.deps.OAuth2.ClientRepo, client, apply)); err != nil {
			return operations, err
		}
	}
	if request.Profile == domain.ProfileBootstrap {
		return operations, nil
	}
	if request.Profile == domain.ProfilePerformance {
		return c.performanceOperations(ctx, request, apply, operations, now)
	}
	// 既存 demo manifest の移設はこの contributor が唯一の呼び出し元となる。詳細な
	// semantic diff は次の contributor 分割で resource ごとに導入する。
	complete, err := c.demoManifestComplete(ctx)
	if err != nil {
		return operations, err
	}
	operation := domain.Operation{LogicalKey: "development-demo", Kind: domain.OperationNoop, Summary: "development demo manifest"}
	if !complete {
		operation.Kind = domain.OperationCreate
	}
	operations = append(operations, operation)
	if !apply {
		return operations, nil
	}
	if err := SeedDemoData(ctx, c.deps.OAuth2.ClientRepo, c.deps.IdManagement.UserRepo, c.deps.Authentication.MfaFactorRepo, c.deps.Authentication.PasswordHistoryRepo, c.deps.IdManagement.GroupRepo, c.deps.OAuth2.AuthzDetailTypeRepo, c.hasher); err != nil {
		return operations, err
	}
	if err := SeedWsFedRelyingParty(ctx, c.deps.WsFederation.RPRepo); err != nil {
		return operations, err
	}
	if err := SeedSamlServiceProvider(ctx, c.deps.Saml.SPRepo); err != nil {
		return operations, err
	}
	return operations, SeedDemoApplications(ctx, c.deps.Application.Repo, c.deps.Application.AssignmentRepo, now)
}

// demoManifestComplete は apply 前後で同じ最小完了条件を判定する。各 resource の
// 詳細な semantic drift は apply 側の ensure が検出し、既存値を上書きしない。
func (c *seedContributor) demoManifestComplete(ctx context.Context) (bool, error) {
	demo, err := c.deps.OAuth2.ClientRepo.FindByID(ctx, tenancydomain.DefaultTenantID, seedDemoClientID)
	if err != nil || demo == nil {
		return false, err
	}
	for _, id := range []string{seedUserAliceID, seedUserRootID} {
		user, findErr := c.deps.IdManagement.UserRepo.FindBySub(ctx, id)
		if findErr != nil || user == nil {
			return false, findErr
		}
	}
	for _, id := range []string{seedGroupEngineeringID, seedGroupSupportID} {
		group, findErr := c.deps.IdManagement.GroupRepo.FindByID(ctx, tenancydomain.DefaultTenantID, id)
		if findErr != nil || group == nil {
			return false, findErr
		}
	}
	for _, id := range []string{"00000000-0000-4000-8000-000000000101", "00000000-0000-4000-8000-000000000102", "00000000-0000-4000-8000-000000000103", "00000000-0000-4000-8000-000000000104"} {
		app, findErr := c.deps.Application.Repo.FindByID(ctx, tenancydomain.DefaultTenantID, id)
		if findErr != nil || app == nil {
			return false, findErr
		}
	}
	return true, nil
}

func (c *seedContributor) performanceOperations(ctx context.Context, request domain.Request, apply bool, operations []domain.Operation, now time.Time) ([]domain.Operation, error) {
	const password = "performance-seed-password-not-for-login"
	hash, err := c.hasher.Hash(password)
	if err != nil {
		return operations, err
	}
	creates := 0
	batchSize := request.EffectiveBatchSize()
	for start := 0; start < request.Count; start += batchSize {
		end := min(start+batchSize, request.Count)
		for index := start; index < end; index++ {
			user := performanceUser(index, request.GeneratorSeed, hash, now)
			existing, err := c.deps.IdManagement.UserRepo.FindBySub(ctx, user.ID)
			if err != nil {
				return operations, err
			}
			if existing == nil {
				creates++
				if apply {
					if err := c.deps.IdManagement.UserRepo.Save(ctx, user); err != nil {
						return operations, err
					}
				}
				continue
			}
			if !samePerformanceUser(existing, user, password, c.hasher) {
				return append(operations, domain.Operation{LogicalKey: fmt.Sprintf("performance-user:%d", index), Kind: domain.OperationConflict, Summary: "synthetic user drift"}), fmt.Errorf("seed drift at performance-user:%d", index)
			}
		}
	}
	kind := domain.OperationNoop
	if creates > 0 {
		kind = domain.OperationCreate
	}
	return append(operations, domain.Operation{LogicalKey: "performance-users", Kind: kind, Summary: fmt.Sprintf("%d synthetic users", request.Count)}), nil
}

func performanceUser(index int, generatorSeed, passwordHash string, now time.Time) *userdomain.User {
	digest := sha256.Sum256([]byte(generatorSeed))
	prefix := fmt.Sprintf("%x", digest[:3])
	return &userdomain.User{
		ID:                fmt.Sprintf("00000000-0000-4000-9000-%06s%06d", prefix, index+1),
		TenantID:          tenancydomain.DefaultTenantID,
		PreferredUsername: fmt.Sprintf("perf-%s-%06d", prefix, index+1),
		PasswordHash:      passwordHash,
		Lifecycle:         userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func samePerformanceUser(actual, desired *userdomain.User, password string, hasher passwordports.PasswordHasher) bool {
	matches, err := hasher.Verify(password, actual.PasswordHash)
	if err != nil || !matches {
		return false
	}
	left, right := *actual, *desired
	left.PasswordHash, right.PasswordHash = "", ""
	left.CreatedAt, left.UpdatedAt = time.Time{}, time.Time{}
	right.CreatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}
