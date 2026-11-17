package idmanagement_test

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/idmanagement"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

func TestIdentityRoutesRejectMissingCredentialOperations(t *testing.T) {
	for _, missing := range []string{"purge", "password", "identity", "revocation"} {
		t.Run(missing, func(t *testing.T) {
			deps := idmanagement.RouteDeps{
				UserCredentials:         authentication.NewMemoryModule(nil).UserCredentials(),
				ValidateInitialPassword: authentication.ValidateInitialPassword,
				AccountIdentity:         authentication.AccountIdentity(&support.Authenticator{}),
				Reactor:                 ignoreRevocation{},
			}
			switch missing {
			case "purge":
				deps.UserCredentials = nil
			case "password":
				deps.ValidateInitialPassword = nil
			case "identity":
				deps.AccountIdentity = nil
			case "revocation":
				deps.Reactor = nil
			}
			defer func() {
				if recover() == nil {
					t.Fatal("identity routes accepted a missing credential operation")
				}
			}()
			idmanagement.NewMemoryModule().RegisterRoutes(echo.New().Group(""), deps)
		})
	}
}

// 管理 API の外から User を止める操作は、失効の反応器なしでは組み立てられない。
func TestStoppingCommandsRejectAMissingRevocationReactor(t *testing.T) {
	module := idmanagement.NewMemoryModule()
	emit := func(spec.DomainEvent) error { return nil }
	for name, build := range map[string]func(){
		"UserLifecycleCommands": func() { module.UserLifecycleCommands(idmanagement.LifecycleDeps{}, emit, nil, "scim") },
		"AdminUsers":            func() { module.AdminUsers(idmanagement.LifecycleDeps{}, emit, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("%s accepted a missing revocation reactor", name)
				}
			}()
			build()
		})
	}
}

// ignoreRevocation は、失効エポックを確かめないテストが、止める操作へ明示して渡す何もしない反応器である。
type ignoreRevocation struct{}

func (ignoreRevocation) React(context.Context, spec.DomainEvent) error { return nil }
