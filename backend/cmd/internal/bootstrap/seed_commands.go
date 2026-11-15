package bootstrap

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	"github.com/ambi/idmagic/backend/authentication"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	"github.com/ambi/idmagic/backend/seeding"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	wsfeddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
)

type seedCommands struct {
	deps       *Dependencies
	identities idmanagement.SeedCommands
	emit       func(spec.DomainEvent)
}

var _ seeding.ResourceCommands = seedCommands{}

func seedIdentityCommands(deps *Dependencies, emit func(spec.DomainEvent) error) idmanagement.SeedCommands {
	lifecycle := deps.UserLifecycleDeps()
	// 背景処理の context には ID だけが入るため、初期パスワードの検証時に設定を解決する。
	lifecycle.ValidateInitialPassword = func(ctx context.Context, password string) error {
		tenant, err := deps.Tenancy.TenantRepo.FindByID(ctx, tenantports.TenantID(ctx))
		if err != nil {
			return err
		}
		return authentication.ValidateInitialPassword(tenantports.WithTenant(ctx, tenant, "", ""), password)
	}
	return deps.IdManagement.SeedCommands(lifecycle, emit)
}

func (c seedCommands) CreateClient(ctx context.Context, client *oauthdomain.OAuth2Client) error {
	return c.deps.OAuth2.CreateSeedClient(ctx, client, c.deps.Tenancy.QuotaRepo, c.emit)
}

func (c seedCommands) CreateUser(ctx context.Context, user *userdomain.User, password string) error {
	return c.identities.CreateUser(ctx, user, password)
}

func (c seedCommands) EnsurePasswordHistory(ctx context.Context, userID, password string, now time.Time) error {
	return c.deps.Authentication.EnsureSeedPasswordHistory(ctx, c.deps.IdManagement.UserRepo, userID, password, now)
}

func (c seedCommands) CreateGroup(ctx context.Context, group *groupdomain.Group) error {
	return c.identities.CreateGroup(ctx, group)
}

func (c seedCommands) AddGroupMember(ctx context.Context, member *groupdomain.GroupMember) error {
	return c.identities.AddGroupMember(ctx, tenancydomain.DefaultTenantID, member.GroupID, member.UserID, member.CreatedAt)
}

func (c seedCommands) CreateTOTP(ctx context.Context, factor *totpdomain.MfaFactor) error {
	return c.deps.Authentication.CreateSeedTOTP(tenantports.WithTenantID(ctx, tenancydomain.DefaultTenantID), c.deps.IdManagement.UserRepo, factor, c.emit)
}

func (c seedCommands) CreateAuthorizationDetailType(ctx context.Context, detail *oauthdomain.AuthorizationDetailType) error {
	return c.deps.OAuth2.CreateSeedAuthorizationDetailType(ctx, detail)
}

func (c seedCommands) CreateRelyingParty(ctx context.Context, rp *wsfeddomain.WsFedRelyingParty) error {
	return c.deps.WsFederation.CreateSeedRelyingParty(ctx, rp)
}

func (c seedCommands) CreateServiceProvider(ctx context.Context, sp *samldomain.SamlServiceProvider) error {
	return c.deps.Saml.CreateSeedServiceProvider(ctx, sp)
}

func (c seedCommands) CreateApplication(ctx context.Context, app *appdomain.Application) error {
	return appusecases.CreateSeedApplication(ctx, appusecases.ApplicationDeps{
		Repo: c.deps.Application.Repo, QuotaRepo: c.deps.Tenancy.QuotaRepo, Emit: c.emit,
	}, app)
}

func (c seedCommands) AssignApplication(ctx context.Context, assignment *appdomain.ApplicationAssignment) error {
	_, err := appusecases.AssignApplication(tenantports.WithTenantID(ctx, assignment.TenantID), appusecases.AssignmentDeps{
		Repo: c.deps.Application.Repo, AssignmentRepo: c.deps.Application.AssignmentRepo,
		SubjectDirectory:     application.SubjectDirectory(c.deps.IdManagement.UserRepo, c.deps.IdManagement.GroupRepo),
		ProvisioningNotifier: c.deps.Application.ProvisioningNotifier, Emit: c.emit,
	}, appusecases.AssignApplicationInput{
		ActorUserID: "seed", ApplicationID: assignment.ApplicationID,
		SubjectType: assignment.SubjectType, SubjectID: assignment.SubjectID, Visibility: assignment.Visibility, Now: assignment.CreatedAt,
	})
	return err
}
