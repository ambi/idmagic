package ldap_goldap_test

import (
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/ldap_goldap"
)

//spec:covers REQ-DIRECTORYCONNECTOR-002: ldap:// の URL で StartTLS を指定しない構成を、接続する前に拒否する。
func TestDirectoryRefusesPlaintextLDAP(t *testing.T) {
	if _, err := ldap_goldap.New(ldap_goldap.Config{URL: "ldap://127.0.0.1:1"}); !errors.Is(err, ldap_goldap.ErrPlaintextLDAP) {
		t.Fatalf("error = %v, want ErrPlaintextLDAP", err)
	}
}
