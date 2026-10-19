package authentication_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/authentication"
)

func TestAccountIdentityRequiresAnAuthenticator(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("account identity accepted a missing authenticator")
		}
	}()
	authentication.AccountIdentity(nil)
}
