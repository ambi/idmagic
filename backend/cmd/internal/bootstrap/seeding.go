package bootstrap

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"reflect"
	"time"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/seeding/domain"
	manifestadapter "github.com/ambi/idmagic/backend/seeding/manifests_yaml"
	seedusecases "github.com/ambi/idmagic/backend/seeding/usecases"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// Seed は runtime composition から Seeding usecase へ既存 record context の port を接続する。
func Seed(ctx context.Context, deps *Dependencies, request domain.Request) (domain.Plan, error) {
	path := request.ManifestPath
	if path == "" {
		var err error
		path, err = manifestadapter.LocateDefaultPath(request.Profile)
		if err != nil {
			return domain.Plan{}, err
		}
	}
	seedManifest, err := manifestadapter.Load(path)
	if err != nil {
		return domain.Plan{}, err
	}
	materialized, err := seedusecases.MaterializeManifest(request, seedManifest, manifestadapter.SecretResolver{
		SecretRoot: os.Getenv("SEED_SECRET_ROOT"),
		Getenv:     os.Getenv,
	})
	if err != nil {
		return domain.Plan{}, err
	}
	return seedusecases.Run(ctx, request, &seedContributor{deps: deps, hasher: passwords_argon2id.NewArgon2idPasswordHasher(), manifest: materialized})
}

type seedContributor struct {
	deps     *Dependencies
	hasher   passwordports.PasswordHasher
	manifest seedusecases.MaterializedManifest
}

func firstPartyClients(seeds []domain.FirstPartyClientSeed, redirectURIs []string, now time.Time) []*oauthdomain.OAuth2Client {
	result := make([]*oauthdomain.OAuth2Client, 0, len(seeds))
	for _, client := range seeds {
		name := client.Name
		result = append(result, &oauthdomain.OAuth2Client{
			TenantID: tenancydomain.DefaultTenantID, ClientID: client.ID, ClientName: &name,
			ClientType: spec.ClientPublic, RedirectURIs: redirectURIs,
			GrantTypes:    []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
			ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode}, TokenEndpointAuthMethod: oauthdomain.AuthMethodNone,
			Scope: client.Scope, IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
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
	for _, resource := range c.manifest.Manifest.Resources {
		if resource.Kind == domain.ResourceKindFirstPartyClients {
			for _, client := range firstPartyClients(resource.Clients, redirectURIs, now) {
				if err := appendOperation(ensureClient(ctx, c.deps.OAuth2.ClientRepo, client, apply)); err != nil {
					return operations, err
				}
			}
		}
	}
	if request.Profile == domain.ProfileBootstrap {
		return operations, nil
	}
	if request.Profile == domain.ProfilePerformance {
		if len(c.manifest.Manifest.Generators) != 1 || c.manifest.Manifest.Generators[0].Kind != "performance_users" {
			return operations, fmt.Errorf("performance manifest requires one performance_users generator")
		}
		return c.performanceOperations(ctx, request, apply, operations, now)
	}
	var demo *domain.Resource
	for index := range c.manifest.Manifest.Resources {
		if c.manifest.Manifest.Resources[index].Kind == domain.ResourceKindDevelopmentDemo {
			demo = &c.manifest.Manifest.Resources[index]
			break
		}
	}
	if demo == nil {
		return operations, nil
	}
	// 既存 demo manifest の移設はこの contributor が唯一の呼び出し元となる。詳細な
	// semantic diff は次の contributor 分割で resource ごとに導入する。
	complete, err := c.demoManifestComplete(ctx, *demo.Demo)
	if err != nil {
		return operations, err
	}
	operation := domain.Operation{LogicalKey: demo.LogicalKey, Kind: domain.OperationNoop, Summary: "demo manifest"}
	if !complete {
		operation.Kind = domain.OperationCreate
	}
	operations = append(operations, operation)
	if !apply {
		return operations, nil
	}
	if err := SeedDemoData(ctx, c.deps.OAuth2.ClientRepo, c.deps.IdManagement.UserRepo, c.deps.Authentication.MfaFactorRepo, c.deps.Authentication.PasswordHistoryRepo, c.deps.IdManagement.GroupRepo, c.deps.OAuth2.AuthzDetailTypeRepo, c.hasher,
		*demo.Demo, c.manifest.Secret(demo.LogicalKey, "client_secret"), c.manifest.Secret(demo.LogicalKey, "user_password"), c.manifest.Secret(demo.LogicalKey, "totp_secret")); err != nil {
		return operations, err
	}
	if err := SeedWsFedRelyingParty(ctx, c.deps.WsFederation.RPRepo, *demo.Demo); err != nil {
		return operations, err
	}
	if err := SeedSamlServiceProvider(ctx, c.deps.Saml.SPRepo, *demo.Demo); err != nil {
		return operations, err
	}
	return operations, SeedDemoApplications(ctx, c.deps.Application.Repo, c.deps.Application.AssignmentRepo, *demo.Demo, now)
}

// demoManifestComplete は apply 前後で同じ最小完了条件を判定する。各 resource の
// 詳細な semantic drift は apply 側の ensure が検出し、既存値を上書きしない。
func (c *seedContributor) demoManifestComplete(ctx context.Context, seed domain.DevelopmentDemoSeed) (bool, error) {
	demo, err := c.deps.OAuth2.ClientRepo.FindByID(ctx, tenancydomain.DefaultTenantID, seed.ClientID)
	if err != nil || demo == nil {
		return false, err
	}
	for _, desired := range seed.Users {
		user, findErr := c.deps.IdManagement.UserRepo.FindBySub(ctx, desired.ID)
		if findErr != nil || user == nil {
			return false, findErr
		}
	}
	for _, desired := range seed.Groups {
		group, findErr := c.deps.IdManagement.GroupRepo.FindByID(ctx, tenancydomain.DefaultTenantID, desired.ID)
		if findErr != nil || group == nil {
			return false, findErr
		}
	}
	for _, desired := range seed.Applications {
		app, findErr := c.deps.Application.Repo.FindByID(ctx, tenancydomain.DefaultTenantID, desired.ID)
		if findErr != nil || app == nil {
			return false, findErr
		}
	}
	return true, nil
}

func (c *seedContributor) performanceOperations(ctx context.Context, request domain.Request, apply bool, operations []domain.Operation, now time.Time) ([]domain.Operation, error) {
	creates := 0
	batchSize := request.EffectiveBatchSize()
	for start := 0; start < request.Count; start += batchSize {
		end := min(start+batchSize, request.Count)
		for index := start; index < end; index++ {
			user := performanceUser(index, request.GeneratorSeed, now)
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
			if !samePerformanceUser(existing, user) {
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

func performanceUser(index int, generatorSeed string, now time.Time) *userdomain.User {
	digest := sha256.Sum256([]byte(generatorSeed))
	prefix := fmt.Sprintf("%x", digest[:3])
	return &userdomain.User{
		ID:                fmt.Sprintf("00000000-0000-4000-9000-%06s%06d", prefix, index+1),
		TenantID:          tenancydomain.DefaultTenantID,
		PreferredUsername: fmt.Sprintf("perf-%s-%06d", prefix, index+1),
		Lifecycle:         userdomain.UserLifecycle{Status: idmdomain.UserStatusDisabled},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func samePerformanceUser(actual, desired *userdomain.User) bool {
	left, right := *actual, *desired
	left.CreatedAt, left.UpdatedAt = time.Time{}, time.Time{}
	right.CreatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}
