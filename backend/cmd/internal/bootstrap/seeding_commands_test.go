package bootstrap

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/seeding"
	"github.com/ambi/idmagic/backend/seeding/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-SEEDING-006: 必須の上限、監査、イベントまたは排他の配線が欠ければ作成前に拒否する。
func TestSeedRejectsMissingRequiredDependencies(t *testing.T) {
	for name, remove := range map[string]func(*Dependencies){
		"quota":  func(d *Dependencies) { d.Tenancy.QuotaRepo = nil },
		"audit":  func(d *Dependencies) { d.Audit.AuditEventRepo = nil },
		"events": func(d *Dependencies) { d.OAuth2.EventSink = nil },
		"lock":   func(d *Dependencies) { d.SeedRequestLock = nil },
	} {
		t.Run(name, func(t *testing.T) {
			deps := newSeedRefusalDeps(t)
			remove(deps)
			_, err := Seed(t.Context(), deps, domain.Request{Environment: domain.EnvironmentTest, Profile: domain.ProfileBootstrap, Mode: domain.ModeApply}, "")
			if err == nil {
				t.Fatal("seed accepted a missing required dependency")
			}
			if after := readSeedStoreState(t, deps); after != (seedStoreState{}) {
				t.Fatalf("seed wrote before rejecting dependencies: %+v", after)
			}
		})
	}
}

//spec:covers REQ-SEEDING-006: 正式な入口の適用が、書き込みの前に構成済みの排他を取得する。
func TestSeedRunsWritesInsideConfiguredLock(t *testing.T) {
	deps := newSeedRefusalDeps(t)
	called := false
	deps.SeedRequestLock = func(ctx context.Context, request domain.Request, apply func(context.Context) error) error {
		called = true
		before, err := deps.OAuth2.ClientRepo.FindAll(ctx, tenancydomain.DefaultTenantID)
		if err != nil {
			t.Fatal(err)
		}
		if len(before) != 0 {
			t.Fatal("seed wrote before acquiring the lock")
		}
		if err := apply(ctx); err != nil {
			return err
		}
		after, err := deps.OAuth2.ClientRepo.FindAll(ctx, tenancydomain.DefaultTenantID)
		if err != nil {
			t.Fatal(err)
		}
		if len(after) != 2 {
			t.Fatalf("lock did not enclose seed writes: %d clients", len(after))
		}
		return nil
	}
	if _, err := Seed(t.Context(), deps, domain.Request{Environment: domain.EnvironmentTest, Profile: domain.ProfileBootstrap, Mode: domain.ModeApply}, ""); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("configured seed lock was bypassed")
	}
}

//spec:covers REQ-SEEDING-006: seed のパスワードも対象 Tenant の初期パスワード方針で拒否し、User とその使用量を残さない。
func TestSeedUsesTenantInitialPasswordPolicy(t *testing.T) {
	t.Setenv("DEMO_CLIENT_SECRET", "demo-client-secret")
	t.Setenv("DEMO_USER_PASSWORD", "demo-password-1234")
	deps := newSeedRefusalDeps(t)
	if err := deps.Tenancy.EnsureDefault(t.Context(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	tenant, err := deps.Tenancy.TenantRepo.FindByID(t.Context(), tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	minimum := 30
	tenant.PasswordPolicyOverride = &tenancydomain.PasswordPolicyOverride{MinLength: &minimum}
	if err := deps.Tenancy.TenantRepo.Save(t.Context(), tenant); err != nil {
		t.Fatal(err)
	}
	request := domain.Request{Environment: domain.EnvironmentDevelopment, Profile: domain.ProfileDevelopment, Mode: domain.ModeApply}
	if _, err := Seed(t.Context(), deps, request, ""); err == nil {
		t.Fatal("seed bypassed tenant password policy")
	}
	usage, err := deps.Tenancy.QuotaRepo.GetUsage(t.Context(), tenant.ID)
	if err != nil || usage.Users != 0 {
		t.Fatalf("rejected user usage = %+v, %v", usage, err)
	}
	if state := readSeedStoreState(t, deps); state.users != 0 {
		t.Fatalf("rejected password created a user: %+v", state)
	}
}

//spec:covers REQ-SEEDING-009: デモの User、Group、Application、割り当て、認可詳細型と連携先の変更を計画時に検出して保存値を残す。
func TestSeedPlansChangedDemoRecords(t *testing.T) {
	t.Setenv("DEMO_CLIENT_SECRET", "demo-client-secret")
	t.Setenv("DEMO_USER_PASSWORD", "demo-password-1234")
	const appID = "00000000-0000-4000-8000-000000000101"
	cases := map[string]func(*testing.T, *Dependencies){
		"user": func(t *testing.T, d *Dependencies) {
			t.Helper()
			u, err := d.IdManagement.UserRepo.FindBySub(t.Context(), seedUserAliceID)
			if err != nil {
				t.Fatal(err)
			}
			u.Roles = []string{"catalog:read"}
			if err := d.IdManagement.UserRepo.Save(t.Context(), u); err != nil {
				t.Fatal(err)
			}
		},
		"group": func(t *testing.T, d *Dependencies) {
			t.Helper()
			groups, err := d.IdManagement.GroupRepo.ListAll(t.Context(), tenancydomain.DefaultTenantID)
			if err != nil {
				t.Fatal(err)
			}
			groups[0].Name = "manual group"
			if err := d.IdManagement.GroupRepo.Save(t.Context(), groups[0]); err != nil {
				t.Fatal(err)
			}
		},
		"application": func(t *testing.T, d *Dependencies) {
			t.Helper()
			app, err := d.Application.Repo.FindByID(t.Context(), tenancydomain.DefaultTenantID, appID)
			if err != nil {
				t.Fatal(err)
			}
			app.Name = "manual application"
			if err := d.Application.Repo.Save(t.Context(), app); err != nil {
				t.Fatal(err)
			}
		},
		"assignment": func(t *testing.T, d *Dependencies) {
			t.Helper()
			assignments, err := d.Application.AssignmentRepo.ListByApplication(t.Context(), tenancydomain.DefaultTenantID, appID)
			if err != nil {
				t.Fatal(err)
			}
			assignments[0].Visibility = appdomain.AssignmentHidden
			if err := d.Application.AssignmentRepo.Save(t.Context(), assignments[0]); err != nil {
				t.Fatal(err)
			}
		},
		"authorization detail type": func(t *testing.T, d *Dependencies) {
			t.Helper()
			detail, err := d.OAuth2.AuthzDetailTypeRepo.FindByType(t.Context(), tenancydomain.DefaultTenantID, "payment_initiation")
			if err != nil {
				t.Fatal(err)
			}
			detail.State = oauthdomain.DetailTypeDisabled
			if err := d.OAuth2.AuthzDetailTypeRepo.Save(t.Context(), detail); err != nil {
				t.Fatal(err)
			}
		},
		"relying party": func(t *testing.T, d *Dependencies) {
			t.Helper()
			rp, err := d.WsFederation.RPRepo.FindByWtrealm(t.Context(), tenancydomain.DefaultTenantID, "urn:idmagic:demo-rp")
			if err != nil {
				t.Fatal(err)
			}
			rp.DisplayName = "manual RP"
			if err := d.WsFederation.RPRepo.Save(t.Context(), rp); err != nil {
				t.Fatal(err)
			}
		},
		"service provider": func(t *testing.T, d *Dependencies) {
			t.Helper()
			sp, err := d.Saml.SPRepo.FindByEntityID(t.Context(), tenancydomain.DefaultTenantID, "urn:idmagic:demo-sp")
			if err != nil {
				t.Fatal(err)
			}
			sp.DisplayName = "manual SP"
			if err := d.Saml.SPRepo.Save(t.Context(), sp); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			deps := newSeedRefusalDeps(t)
			request := domain.Request{Environment: domain.EnvironmentDevelopment, Profile: domain.ProfileDevelopment, Mode: domain.ModeApply}
			if _, err := Seed(t.Context(), deps, request, ""); err != nil {
				t.Fatal(err)
			}
			edit(t, deps)
			before := seedDemoSnapshot(t, deps)
			request.Mode = domain.ModeDryRun
			plan, _ := Seed(t.Context(), deps, request, "")
			if plan.Count(domain.OperationConflict) != 1 {
				t.Fatalf("manual change plan = %+v", plan)
			}
			request.Mode = domain.ModeApply
			if _, err := Seed(t.Context(), deps, request, ""); err == nil {
				t.Fatal("apply accepted a manual change")
			}
			if !reflect.DeepEqual(before, seedDemoSnapshot(t, deps)) {
				t.Fatal("planning or rejected apply changed demo records")
			}
		})
	}
}

func seedDemoSnapshot(t *testing.T, d *Dependencies) []any {
	t.Helper()
	ctx := t.Context()
	user, err := d.IdManagement.UserRepo.FindBySub(ctx, seedUserAliceID)
	if err != nil {
		t.Fatal(err)
	}
	groups, err := d.IdManagement.GroupRepo.ListAll(ctx, tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	const appID = "00000000-0000-4000-8000-000000000101"
	app, err := d.Application.Repo.FindByID(ctx, tenancydomain.DefaultTenantID, appID)
	if err != nil {
		t.Fatal(err)
	}
	assignments, err := d.Application.AssignmentRepo.ListByApplication(ctx, tenancydomain.DefaultTenantID, appID)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := d.OAuth2.AuthzDetailTypeRepo.FindByType(ctx, tenancydomain.DefaultTenantID, "payment_initiation")
	if err != nil {
		t.Fatal(err)
	}
	rp, err := d.WsFederation.RPRepo.FindByWtrealm(ctx, tenancydomain.DefaultTenantID, "urn:idmagic:demo-rp")
	if err != nil {
		t.Fatal(err)
	}
	sp, err := d.Saml.SPRepo.FindByEntityID(ctx, tenancydomain.DefaultTenantID, "urn:idmagic:demo-sp")
	if err != nil {
		t.Fatal(err)
	}
	events, err := d.Audit.AuditEventRepo.List(ctx, auditports.AuditEventQuery{TenantID: tenancydomain.DefaultTenantID, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	return []any{user, groups, app, assignments, detail, rp, sp, events}
}

//spec:covers REQ-SEEDING-006, REQ-SEEDING-010: 合成 User も上限で拒否され、上限の引き上げ後に未完了分だけ作成する。
func TestPerformanceSeedRespectsQuotaAndResumesWithoutDuplicateEvents(t *testing.T) {
	deps, err := assembleMemory(SharedConfig{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	limit := 2
	if err := deps.Tenancy.QuotaRepo.SetQuota(ctx, tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{Users: &limit}); err != nil {
		t.Fatal(err)
	}
	request := domain.Request{Environment: domain.EnvironmentTest, Profile: domain.ProfilePerformance, Mode: domain.ModeApply, Count: 3, GeneratorSeed: "quota-test"}
	for range 2 {
		_, err := Seed(ctx, deps, request, "")
		if _, ok := errors.AsType[*tenancydomain.QuotaExceededError](err); !ok {
			t.Fatalf("quota rejection = %v", err)
		}
	}
	usage, err := deps.Tenancy.QuotaRepo.GetUsage(ctx, tenancydomain.DefaultTenantID)
	if err != nil || usage.Users != 2 {
		t.Fatalf("usage = %+v, %v", usage, err)
	}
	users, err := deps.IdManagement.UserRepo.Count(ctx, tenancydomain.DefaultTenantID)
	if err != nil || users != 2 {
		t.Fatalf("users = %d, %v", users, err)
	}
	limit = 3
	if err := deps.Tenancy.QuotaRepo.SetQuota(ctx, tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{Users: &limit}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := Seed(ctx, deps, request, ""); err != nil {
			t.Fatal(err)
		}
	}
	usage, err = deps.Tenancy.QuotaRepo.GetUsage(ctx, tenancydomain.DefaultTenantID)
	if err != nil || usage.Users != 3 {
		t.Fatalf("resumed usage = %+v, %v", usage, err)
	}
	events, err := deps.Audit.AuditEventRepo.Count(ctx, auditports.AuditEventQuery{TenantID: tenancydomain.DefaultTenantID, Type: "UserCreated"})
	if err != nil || events != 3 {
		t.Fatalf("UserCreated events = %d, %v", events, err)
	}
}

//spec:covers REQ-SEEDING-006, REQ-SEEDING-009: TOTP も登録イベントを伴い、再適用では変更せず、シークレットの変更を計画時に拒否する。
func TestSeedTOTPUsesEnrollmentEffectsAndDetectsDrift(t *testing.T) {
	t.Setenv("DEMO_CLIENT_SECRET", "demo-client-secret")
	t.Setenv("DEMO_USER_PASSWORD", "demo-password-1234")
	t.Setenv("SEED_TOTP_SECRET", "JBSWY3DPEHPK3PXP")
	path, err := seeding.LocateDefaultPath(domain.ProfileDevelopment)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	manifest := writeSeedManifest(t, t.TempDir(), "totp.yaml", string(body)+`      totp_secret:
        provider: env
        locator: SEED_TOTP_SECRET
        version: v1
`)
	deps, err := assembleMemory(SharedConfig{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	request := domain.Request{Environment: domain.EnvironmentDevelopment, Profile: domain.ProfileDevelopment, Mode: domain.ModeApply, ManifestPath: manifest}
	for range 2 {
		if _, err := Seed(ctx, deps, request, ""); err != nil {
			t.Fatal(err)
		}
	}
	events, err := deps.Audit.AuditEventRepo.Count(ctx, auditports.AuditEventQuery{TenantID: tenancydomain.DefaultTenantID, Type: "MfaFactorEnrolled"})
	if err != nil || events != 1 {
		t.Fatalf("enrollment events = %d, %v", events, err)
	}
	factor, err := deps.Authentication.MfaFactorRepo.Find(ctx, seedUserAliceID, spec.MfaFactorTOTP)
	if err != nil || factor == nil {
		t.Fatalf("factor = %+v, %v", factor, err)
	}
	manual := "KRUGS4ZANFZSAYJA"
	factor.Secret = &manual
	if err := deps.Authentication.MfaFactorRepo.Save(ctx, factor); err != nil {
		t.Fatal(err)
	}
	request.Mode = domain.ModeDryRun
	plan, _ := Seed(ctx, deps, request, "")
	if plan.Count(domain.OperationConflict) != 1 {
		t.Fatalf("TOTP drift plan = %+v", plan)
	}
}

//spec:covers REQ-SEEDING-006: 本番の PostgreSQL 構成でも所有者の保存、使用量、監査と再適用が収束する。
func TestSeedPostgresOwnerCommandsConverge(t *testing.T) {
	db := pgtest.Require(t)
	t.Setenv("DEMO_CLIENT_SECRET", "demo-client-secret")
	t.Setenv("DEMO_USER_PASSWORD", "demo-password-1234")
	settings := map[string]string{"PERSISTENCE": "postgres", "DATABASE_URL": db.Config().ConnString(), "KEY_PROVIDER": "local", "ISSUER": "http://idp.test"}
	loader := NewConfigLoader(func(key string) string { return settings[key] })
	config := LoadSharedConfig(loader)
	if err := loader.Err(); err != nil {
		t.Fatal(err)
	}
	deps, err := Assemble(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(deps.Close)
	if deps.SeedRequestLock == nil {
		t.Fatal("PostgreSQL seed lock is not wired")
	}
	request := domain.Request{Environment: domain.EnvironmentDevelopment, Profile: domain.ProfileDevelopment, Mode: domain.ModeApply}
	for range 2 {
		plan, err := Seed(t.Context(), deps, request, "")
		if err != nil {
			t.Fatalf("seed plan = %+v: %v", plan, err)
		}
		if plan.Count(domain.OperationNoop) != len(plan.Operations) {
			t.Fatalf("not converged: %+v", plan)
		}
	}
	usage, err := deps.Tenancy.QuotaRepo.GetUsage(t.Context(), tenancydomain.DefaultTenantID)
	if err != nil || usage.Users != 2 || usage.Applications != 4 {
		t.Fatalf("usage = %+v, %v", usage, err)
	}
	events, err := deps.Audit.AuditEventRepo.Count(t.Context(), auditports.AuditEventQuery{TenantID: tenancydomain.DefaultTenantID, Type: "UserCreated"})
	if err != nil || events != 2 {
		t.Fatalf("UserCreated events = %d, %v", events, err)
	}
}
