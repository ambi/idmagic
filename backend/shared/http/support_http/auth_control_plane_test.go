package support_http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

type controlPlaneAuthnResolver struct {
	ctx *authdomain.AuthenticationContext
}

func (r controlPlaneAuthnResolver) Resolve(context.Context, authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	return r.ctx, nil
}

func newControlPlaneTestContext() *echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "http://idp.test/api/admin/v1/whatever", http.NoBody)
	ctx := tenancy.WithTenant(req.Context(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: "default"}, "", "")
	return e.NewContext(req.WithContext(ctx), httptest.NewRecorder())
}

func groupRepoWithRole(t *testing.T, sub, tenantID, role string) groupports.GroupRepository {
	t.Helper()
	repo := groupmemory.NewGroupRepository()
	now := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	group := &groupdomain.Group{
		ID: "group-" + role, TenantID: tenantID, Name: "Role group", Roles: []string{role}, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Save(t.Context(), group); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddMember(t.Context(), &groupdomain.GroupMember{
		GroupID: group.ID, UserID: sub, Source: groupdomain.MembershipSourceManual, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	return repo
}

// REQ-SIGNINGKEYS-009、REQ-DATAKEYS-006、REQ-JOBS-012: 制御面主体の純粋な条件を表で固定する。
func TestIsControlPlaneActor(t *testing.T) {
	active := controlPlaneTestActor(tenancydomain.DefaultTenantID, "system_admin")
	disabled := controlPlaneTestActor(tenancydomain.DefaultTenantID, "system_admin")
	disabled.Lifecycle.Status = "disabled"
	tests := []struct {
		name            string
		actor           *userdomain.User
		requestTenantID string
		want            bool
	}{
		{name: "active system admin on the control plane", actor: active, requestTenantID: tenancydomain.DefaultTenantID, want: true},
		{name: "missing actor", actor: nil, requestTenantID: tenancydomain.DefaultTenantID},
		{name: "disabled actor", actor: disabled, requestTenantID: tenancydomain.DefaultTenantID},
		{name: "tenant administrator", actor: controlPlaneTestActor(tenancydomain.DefaultTenantID, "admin"), requestTenantID: tenancydomain.DefaultTenantID},
		{name: "system admin outside the control-plane tenant", actor: controlPlaneTestActor("acme", "system_admin"), requestTenantID: "acme"},
		{name: "tenant system admin through the control-plane route", actor: controlPlaneTestActor("acme", "system_admin"), requestTenantID: tenancydomain.DefaultTenantID},
		{name: "control-plane member through another tenant route", actor: active, requestTenantID: "acme"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsControlPlaneActor(tt.actor, tt.requestTenantID); got != tt.want {
				t.Fatalf("IsControlPlaneActor() = %t, want %t", got, tt.want)
			}
		})
	}
}

func controlPlaneTestActor(tenantID string, roles ...string) *userdomain.User {
	now := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	return &userdomain.User{
		ID: "operator", TenantID: tenantID, PreferredUsername: "operator", PasswordHash: "unused", Roles: roles,
		CreatedAt: now, UpdatedAt: now,
	}
}

// REQ-SIGNINGKEYS-009: 認証状態と Group 由来を含む有効ロールを共通境界で検証する。
func TestRequireControlPlaneUser(t *testing.T) {
	t.Run("requires completed authentication", func(t *testing.T) {
		users := usermemory.NewUserRepository()
		actor := controlPlaneTestActor(tenancydomain.DefaultTenantID, "system_admin")
		users.Seed(actor)
		a := &Authenticator{
			UserRepo: users,
			AuthnResolver: controlPlaneAuthnResolver{ctx: &authdomain.AuthenticationContext{
				UserID: actor.ID, AuthenticationPending: true,
			}},
		}
		if _, err := a.RequireControlPlaneUser(newControlPlaneTestContext()); !errors.Is(err, ErrAdminAuthenticationRequired) {
			t.Fatalf("err = %v, want ErrAdminAuthenticationRequired", err)
		}
	})

	t.Run("accepts a direct effective role", func(t *testing.T) {
		users := usermemory.NewUserRepository()
		actor := controlPlaneTestActor(tenancydomain.DefaultTenantID, "system_admin")
		users.Seed(actor)
		a := &Authenticator{
			UserRepo: users,
			AuthnResolver: controlPlaneAuthnResolver{ctx: &authdomain.AuthenticationContext{
				UserID: actor.ID,
			}},
		}
		got, err := a.RequireControlPlaneUser(newControlPlaneTestContext())
		if err != nil || got == nil || !slices.Contains(got.Roles, "system_admin") {
			t.Fatalf("actor = %+v, err = %v", got, err)
		}
	})

	t.Run("accepts a group-derived effective role", func(t *testing.T) {
		users := usermemory.NewUserRepository()
		actor := controlPlaneTestActor(tenancydomain.DefaultTenantID)
		users.Seed(actor)
		a := &Authenticator{
			UserRepo:  users,
			GroupRepo: groupRepoWithRole(t, actor.ID, actor.TenantID, "system_admin"),
			AuthnResolver: controlPlaneAuthnResolver{ctx: &authdomain.AuthenticationContext{
				UserID: actor.ID,
			}},
		}
		got, err := a.RequireControlPlaneUser(newControlPlaneTestContext())
		if err != nil || got == nil || !slices.Contains(got.Roles, "system_admin") {
			t.Fatalf("actor = %+v, err = %v", got, err)
		}
	})

	t.Run("rejects another role", func(t *testing.T) {
		users := usermemory.NewUserRepository()
		actor := controlPlaneTestActor(tenancydomain.DefaultTenantID, "admin")
		users.Seed(actor)
		a := &Authenticator{
			UserRepo: users,
			AuthnResolver: controlPlaneAuthnResolver{ctx: &authdomain.AuthenticationContext{
				UserID: actor.ID,
			}},
		}
		if _, err := a.RequireControlPlaneUser(newControlPlaneTestContext()); !errors.Is(err, ErrAdminAccessDenied) {
			t.Fatalf("err = %v, want ErrAdminAccessDenied", err)
		}
	})
}

// REQ-DATAKEYS-006: 要求先テナントの条件を所属先と独立に検証する。
func TestRequireControlPlaneUserRejectsRequestOutsideControlPlaneTenant(t *testing.T) {
	actor := controlPlaneTestActor(tenancydomain.DefaultTenantID, "system_admin")
	if IsControlPlaneActor(actor, "acme") {
		t.Fatal("制御面テナント外の要求を制御面主体として扱った")
	}
}
