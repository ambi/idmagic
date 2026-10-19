package idmanagement_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/idmanagement"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

func TestIdentityRoutesRejectMissingCredentialOperations(t *testing.T) {
	for _, missing := range []string{"purge", "password", "identity"} {
		t.Run(missing, func(t *testing.T) {
			deps := idmanagement.RouteDeps{
				UserCredentials:         authentication.NewMemoryModule(nil).UserCredentials(),
				ValidateInitialPassword: authentication.ValidateInitialPassword,
				AccountIdentity:         authentication.AccountIdentity(&support.Authenticator{}),
			}
			switch missing {
			case "purge":
				deps.UserCredentials = nil
			case "password":
				deps.ValidateInitialPassword = nil
			case "identity":
				deps.AccountIdentity = nil
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
