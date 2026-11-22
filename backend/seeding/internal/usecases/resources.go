package usecases

import (
	"context"
	"crypto/sha256"
	"fmt"
	"reflect"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	seedingdomain "github.com/ambi/idmagic/backend/seeding/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func NewContributor(deps ResourceDependencies, commands ResourceCommands, manifest MaterializedManifest) (Contributor, error) {
	if commands == nil {
		return nil, fmt.Errorf("seed resource commands are required")
	}
	if deps.ClientRepo == nil || deps.UserRepo == nil || deps.GroupRepo == nil || deps.MfaFactorRepo == nil || deps.PasswordHistoryRepo == nil || deps.AuthzDetailTypeRepo == nil || deps.RPRepo == nil || deps.SPRepo == nil || deps.Applications == nil || deps.AssignmentRepo == nil || deps.Hasher == nil {
		return nil, fmt.Errorf("seed resource readers and password verifier are required")
	}
	return &seedContributor{deps: deps, commands: commands, manifest: manifest}, nil
}

type seedContributor struct {
	deps     ResourceDependencies
	commands ResourceCommands
	manifest MaterializedManifest
}

func ensureClient(ctx context.Context, repo interface {
	FindByID(context.Context, string, string) (*oauthdomain.OAuth2Client, error)
}, commands ResourceCommands, desired *oauthdomain.OAuth2Client, apply bool,
) (seedingdomain.Operation, error) {
	existing, err := repo.FindByID(ctx, desired.TenantID, desired.ClientID)
	if err != nil {
		return seedingdomain.Operation{}, err
	}
	op := seedingdomain.Operation{LogicalKey: "oauth2-client:" + desired.ClientID, Summary: "first-party client"}
	if existing == nil {
		op.Kind = seedingdomain.OperationCreate
		if apply {
			err = commands.CreateClient(ctx, desired)
		}
		return op, err
	}
	if seedingdomain.SameFirstPartyClient(existing, desired) {
		op.Kind = seedingdomain.OperationNoop
		return op, nil
	}
	op.Kind = seedingdomain.OperationConflict
	return op, nil
}

func (c *seedContributor) Plan(ctx context.Context, request seedingdomain.Request) (seedingdomain.Plan, error) {
	operations, err := c.operations(ctx, request, false)
	return seedingdomain.Plan{Operations: operations}, err
}

func (c *seedContributor) Apply(ctx context.Context, request seedingdomain.Request) error {
	_, err := c.operations(ctx, request, true)
	return err
}

func (c *seedContributor) operations(ctx context.Context, request seedingdomain.Request, apply bool) ([]seedingdomain.Operation, error) {
	now := time.Now().UTC()
	redirectURIs := request.FirstPartyRedirectURIs
	if len(redirectURIs) == 0 {
		redirectURIs = []string{
			"http://localhost:3000/callback",
			"http://localhost:5173/realms/default/callback",
			"http://localhost:8080/realms/default/callback",
		}
	}
	operations := make([]seedingdomain.Operation, 0, 16)
	appendOperation := func(operation seedingdomain.Operation, err error) error {
		if err != nil {
			return err
		}
		operations = append(operations, operation)
		if operation.Kind == seedingdomain.OperationConflict {
			return fmt.Errorf("seed drift at %s", operation.LogicalKey)
		}
		return nil
	}
	for _, resource := range c.manifest.Manifest.Resources {
		if resource.Kind == seedingdomain.ResourceKindFirstPartyClients {
			for _, client := range seedingdomain.FirstPartyClients(resource.Clients, redirectURIs, now) {
				if err := appendOperation(ensureClient(ctx, c.deps.ClientRepo, c.commands, client, apply)); err != nil {
					return operations, err
				}
			}
		}
	}
	if request.Profile == seedingdomain.ProfileBootstrap {
		return operations, nil
	}
	if request.Profile == seedingdomain.ProfilePerformance {
		if len(c.manifest.Manifest.Generators) != 1 || c.manifest.Manifest.Generators[0].Kind != "performance_users" {
			return operations, fmt.Errorf("performance manifest requires one performance_users generator")
		}
		return c.performanceOperations(ctx, request, apply, operations, now)
	}
	var demo *seedingdomain.Resource
	for index := range c.manifest.Manifest.Resources {
		if c.manifest.Manifest.Resources[index].Kind == seedingdomain.ResourceKindDevelopmentDemo {
			demo = &c.manifest.Manifest.Resources[index]
			break
		}
	}
	if demo == nil {
		return operations, nil
	}
	demoOperations := &resourceOperations{apply: apply, items: operations}
	if err := SeedDemoData(ctx, c.deps.ClientRepo, c.deps.UserRepo, c.deps.MfaFactorRepo, c.deps.PasswordHistoryRepo, c.deps.GroupRepo, c.deps.AuthzDetailTypeRepo, c.deps.Hasher,
		*demo.Demo, c.manifest.Secret(demo.LogicalKey, "client_secret"), c.manifest.Secret(demo.LogicalKey, "user_password"), c.manifest.Secret(demo.LogicalKey, "totp_secret"), demoOperations, c.commands); err != nil {
		return demoOperations.items, err
	}
	if err := SeedWsFedRelyingParty(ctx, c.deps.RPRepo, *demo.Demo, demoOperations, c.commands); err != nil {
		return demoOperations.items, err
	}
	if err := SeedSamlServiceProvider(ctx, c.deps.SPRepo, *demo.Demo, demoOperations, c.commands); err != nil {
		return demoOperations.items, err
	}
	err := SeedDemoApplications(ctx, c.deps.Applications, c.deps.AssignmentRepo, *demo.Demo, now, demoOperations, c.commands)
	return demoOperations.items, err
}

func (c *seedContributor) performanceOperations(ctx context.Context, request seedingdomain.Request, apply bool, operations []seedingdomain.Operation, now time.Time) ([]seedingdomain.Operation, error) {
	creates := 0
	batchSize := request.EffectiveBatchSize()
	for start := 0; start < request.Count; start += batchSize {
		end := min(start+batchSize, request.Count)
		for index := start; index < end; index++ {
			user := performanceUser(index, request.GeneratorSeed, now)
			existing, err := c.deps.UserRepo.FindBySub(ctx, user.ID)
			if err != nil {
				return operations, err
			}
			if existing == nil {
				creates++
				if apply {
					if err := c.commands.CreateUser(ctx, user, ""); err != nil {
						return operations, err
					}
				}
				continue
			}
			if !samePerformanceUser(existing, user) {
				return append(operations, seedingdomain.Operation{LogicalKey: fmt.Sprintf("performance-user:%d", index), Kind: seedingdomain.OperationConflict, Summary: "synthetic user drift"}), fmt.Errorf("seed drift at performance-user:%d", index)
			}
		}
	}
	kind := seedingdomain.OperationNoop
	if creates > 0 {
		kind = seedingdomain.OperationCreate
	}
	return append(operations, seedingdomain.Operation{LogicalKey: "performance-users", Kind: kind, Summary: fmt.Sprintf("%d synthetic users", request.Count)}), nil
}

func performanceUser(index int, generatorSeed string, now time.Time) *userdomain.User {
	digest := sha256.Sum256([]byte(generatorSeed))
	prefix := fmt.Sprintf("%x", digest[:3])
	return &userdomain.User{
		ID:                fmt.Sprintf("00000000-0000-4000-9000-%06s%06d", prefix, index+1),
		TenantID:          tenancydomain.DefaultTenantID,
		PreferredUsername: fmt.Sprintf("perf-%s-%06d", prefix, index+1),
		Roles:             []string{},
		Lifecycle:         userdomain.UserLifecycle{Status: idmdomain.UserStatusDisabled},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func samePerformanceUser(actual, desired *userdomain.User) bool {
	left, right := *actual, *desired
	left.Roles, right.Roles = normalizedSeedRoles(left.Roles), normalizedSeedRoles(right.Roles)
	left.CreatedAt, left.UpdatedAt = time.Time{}, time.Time{}
	right.CreatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}
