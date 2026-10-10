package support_http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authhttpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	idmhttpdeps "github.com/ambi/idmagic/backend/idmanagement/deps_http"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

// newAdminTestContext builds a request whose tenant is already resolved (as
// tenant middleware would leave it) and whose X-Demo-Sub header authenticates
// as sub via authusecases.DemoHeaderResolver, so RequireAdmin and friends can
// be exercised without the full HTTP router.
func newAdminTestContext(sub string) *echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "http://idp.test/api/admin/v1/whatever", http.NoBody)
	if sub != "" {
		req.Header.Set("X-Demo-Sub", sub)
	}
	ctx := tenantports.WithTenant(req.Context(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm}, "", "")
	req = req.WithContext(ctx)
	return e.NewContext(req, httptest.NewRecorder())
}

func seedAdminUser(t *testing.T, users *usermemory.UserRepository, sub string, active bool, roles ...string) {
	t.Helper()
	now := time.Now().UTC()
	status := idmdomain.UserStatusActive
	if !active {
		status = idmdomain.UserStatusDisabled
	}
	user := &userdomain.User{
		ID: sub, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: sub, PasswordHash: "unused",
		Roles: roles, Lifecycle: userdomain.UserLifecycle{Status: status, StatusChangedAt: &now},
		CreatedAt: now, UpdatedAt: now,
	}
	users.Seed(user)
}

// groupRepoWithAdminGroup seeds a group granting "admin" and adds sub as a
// manual member, so EffectiveRoles/RequireAdmin can be exercised on a role
// that only exists via group membership.
func groupRepoWithAdminGroup(t *testing.T, sub string) groupports.GroupRepository {
	t.Helper()
	repo := groupmemory.NewGroupRepository()
	now := time.Now().UTC()
	group := &groupdomain.Group{
		ID: "group-admins", TenantID: tenancydomain.DefaultTenantID, Name: "Admins",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Save(context.Background(), group); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddMember(context.Background(), &groupdomain.GroupMember{
		GroupID: group.ID, UserID: sub, Source: groupdomain.MembershipSourceManual, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	return repo
}

type erroringGroupRepo struct{ groupports.GroupRepository }

func (erroringGroupRepo) ListGroupsByUser(context.Context, string, string) ([]*groupdomain.Group, error) {
	return nil, errors.New("boom")
}

func TestRequireAdmin(t *testing.T) {
	t.Run("requires an authenticated session", func(t *testing.T) {
		users := usermemory.NewUserRepository()
		a := adminAuthenticator(users, nil)
		c := newAdminTestContext("")
		if _, err := a.RequireAdmin(c); !errors.Is(err, support.ErrAdminAuthenticationRequired) {
			t.Fatalf("err=%v, want support.ErrAdminAuthenticationRequired", err)
		}
	})

	t.Run("rejects a non-admin user", func(t *testing.T) {
		users := usermemory.NewUserRepository()
		seedAdminUser(t, users, "alice", true)
		a := adminAuthenticator(users, nil)
		c := newAdminTestContext("alice")
		if _, err := a.RequireAdmin(c); !errors.Is(err, support.ErrAdminAccessDenied) {
			t.Fatalf("err=%v, want support.ErrAdminAccessDenied", err)
		}
	})

	t.Run("rejects a disabled admin", func(t *testing.T) {
		// ResolveAuthentication itself already treats an inactive user's session
		// as unauthenticated (defense-in-depth), so RequireAdmin's own inactive
		// check downstream never sees a disabled user — the caller observes
		// support.ErrAdminAuthenticationRequired, not support.ErrAdminAccessDenied.
		users := usermemory.NewUserRepository()
		seedAdminUser(t, users, "alice", false, "admin")
		a := adminAuthenticator(users, nil)
		c := newAdminTestContext("alice")
		if _, err := a.RequireAdmin(c); !errors.Is(err, support.ErrAdminAuthenticationRequired) {
			t.Fatalf("err=%v, want support.ErrAdminAuthenticationRequired", err)
		}
	})

	t.Run("accepts an active admin", func(t *testing.T) {
		users := usermemory.NewUserRepository()
		seedAdminUser(t, users, "alice", true, "admin")
		a := adminAuthenticator(users, nil)
		c := newAdminTestContext("alice")
		user, err := a.RequireAdmin(c)
		if err != nil || user == nil || user.ID != "alice" {
			t.Fatalf("user=%+v err=%v", user, err)
		}
	})

	t.Run("admin role granted only through group membership is honored", func(t *testing.T) {
		users := usermemory.NewUserRepository()
		seedAdminUser(t, users, "alice", true)
		groups := groupRepoWithAdminGroup(t, "alice")
		a := adminAuthenticator(users, groups)
		c := newAdminTestContext("alice")
		user, err := a.RequireAdmin(c)
		if err != nil || user == nil {
			t.Fatalf("user=%+v err=%v", user, err)
		}
	})
}

func TestWriteAdminAccessError(t *testing.T) {
	a := &support.Authenticator{}
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"authentication required", support.ErrAdminAuthenticationRequired, http.StatusUnauthorized},
		{"access denied", support.ErrAdminAccessDenied, http.StatusForbidden},
		{"invalid token", &support.InvalidTokenError{}, http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			// 応答を書いたうえで、呼び出し元が止まれる合図を返す。nil を返すと、
			// この関数を包む認可ヘルパーの拒否が呼び出し元へ伝わらない。
			if err := a.WriteAdminAccessError(c, tc.err); !errors.Is(err, support.ErrResponseWritten) {
				t.Fatalf("err=%v, want a refusal wrapping support.ErrResponseWritten", err)
			}
			if rec.Code != tc.wantStatus {
				t.Fatalf("status=%d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}

	t.Run("passes through an unmapped error", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
		c := e.NewContext(req, httptest.NewRecorder())
		other := errors.New("boom")
		if err := a.WriteAdminAccessError(c, other); !errors.Is(err, other) {
			t.Fatalf("err=%v, want passthrough of %v", err, other)
		}
	})
}

//spec:covers REQ-APPLICATION-004, EX-APPLICATION-004-03: admin API まで届いた別 realm 向けトークンの検証失敗を 401 invalid_token のまま公開する。
func TestWriteAdminAccessErrorPreservesInvalidTokenClassification(t *testing.T) {
	e := echo.New()
	request := httptest.NewRequest(http.MethodGet, "/api/admin/v1/applications", http.NoBody)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)

	err := (&support.Authenticator{}).WriteAdminAccessError(c, &support.InvalidTokenError{})
	if !errors.Is(err, support.ErrResponseWritten) {
		t.Fatalf("err=%v, want a refusal wrapping support.ErrResponseWritten", err)
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want %d; body=%s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
	var problem support.Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Type != "urn:idmagic:error:invalid_token" {
		t.Fatalf("problem type=%q, want invalid_token", problem.Type)
	}
	if challenge := recorder.Header().Get("WWW-Authenticate"); challenge != `Bearer error="invalid_token"` {
		t.Fatalf("WWW-Authenticate=%q, want invalid_token challenge", challenge)
	}
}

func TestResolveAdminActor(t *testing.T) {
	t.Run("requires authentication", func(t *testing.T) {
		users := usermemory.NewUserRepository()
		a := adminAuthenticator(users, nil)
		c := newAdminTestContext("")
		if _, err := a.ResolveAdminActor(c); !errors.Is(err, support.ErrAdminAuthenticationRequired) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("rejects a disabled user", func(t *testing.T) {
		// Same defense-in-depth path as RequireAdmin: ResolveAuthentication
		// already turns an inactive user's session into "unauthenticated".
		users := usermemory.NewUserRepository()
		seedAdminUser(t, users, "alice", false)
		a := adminAuthenticator(users, nil)
		c := newAdminTestContext("alice")
		if _, err := a.ResolveAdminActor(c); !errors.Is(err, support.ErrAdminAuthenticationRequired) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("returns the actor with group-derived roles composed in", func(t *testing.T) {
		users := usermemory.NewUserRepository()
		seedAdminUser(t, users, "alice", true, "editor")
		groups := groupRepoWithAdminGroup(t, "alice")
		a := adminAuthenticator(users, groups)
		c := newAdminTestContext("alice")
		actor, err := a.ResolveAdminActor(c)
		if err != nil {
			t.Fatal(err)
		}
		hasAdmin, hasEditor := false, false
		for _, r := range actor.Roles {
			hasAdmin = hasAdmin || r == "admin"
			hasEditor = hasEditor || r == "editor"
		}
		if !hasAdmin || !hasEditor {
			t.Fatalf("actor.Roles=%v, want both admin and editor", actor.Roles)
		}
	})
}

func TestRequireAdministrator(t *testing.T) {
	t.Run("rejects a user without either role", func(t *testing.T) {
		users := usermemory.NewUserRepository()
		seedAdminUser(t, users, "alice", true)
		a := adminAuthenticator(users, nil)
		c := newAdminTestContext("alice")
		if _, err := a.RequireAdministrator(c); !errors.Is(err, support.ErrAdminAccessDenied) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("accepts system_admin as well as admin", func(t *testing.T) {
		users := usermemory.NewUserRepository()
		seedAdminUser(t, users, "bob", true, "system_admin")
		a := adminAuthenticator(users, nil)
		c := newAdminTestContext("bob")
		actor, err := a.RequireAdministrator(c)
		if err != nil || actor == nil {
			t.Fatalf("actor=%+v err=%v", actor, err)
		}
	})
}

func TestEffectiveRolesWithoutGroupRepo(t *testing.T) {
	a := &support.Authenticator{}
	principal := support.Principal{Roles: []string{"editor"}}
	if got := a.EffectiveRoles(context.Background(), principal); len(got) != 1 || got[0] != "editor" {
		t.Fatalf("EffectiveRoles=%v", got)
	}
}

func TestEffectiveRolesPropagatesRepoErrorAsUserRoles(t *testing.T) {
	a := &support.Authenticator{Principals: idmhttpdeps.Principals{Groups: erroringGroupRepo{}}}
	principal := support.Principal{Roles: []string{"editor"}}
	if got := a.EffectiveRoles(context.Background(), principal); len(got) != 1 || got[0] != "editor" {
		t.Fatalf("EffectiveRoles=%v, want fallback to direct roles on repo error", got)
	}
}

func TestEffectiveRolesComposesGroupRolesIntoAFreshSlice(t *testing.T) {
	// With a GroupRepo wired, EffectiveRoles builds a new slice (groupdomain.EffectiveRoles),
	// so the composed roles no longer alias the principal's backing array.
	a := &support.Authenticator{Principals: idmhttpdeps.Principals{Groups: groupRepoWithAdminGroup(t, "alice")}}
	original := support.Principal{ID: "alice", TenantID: tenancydomain.DefaultTenantID, Roles: []string{"editor"}}
	composed := a.EffectiveRoles(context.Background(), original)
	composed[0] = "mutated"
	if original.Roles[0] != "editor" {
		t.Fatal("mutating the composed roles must not affect the principal")
	}
}

// adminAuthenticator は、利用者と Group の保存先、X-Demo-Sub ヘッダーのセッションで認証する
// Authenticator を、組み立て地点と同じアダプターで組み立てる。
func adminAuthenticator(users userports.UserRepository, groups groupports.GroupRepository) *support.Authenticator {
	return &support.Authenticator{
		Principals: idmhttpdeps.Principals{Users: users, Groups: groups},
		Sessions:   authhttpdeps.SessionAuthentications{Resolver: authusecases.DemoHeaderResolver{}},
	}
}
