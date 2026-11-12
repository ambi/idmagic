package ldap_goldap

import (
	"testing"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
)

//spec:covers REQ-DIRECTORYCONNECTOR-008: AD 固有の拒否を診断用に分類し、未知の原文を外へ返さない。
func TestADPasswordDiagnosticClassification(t *testing.T) {
	for code, want := range map[string]string{"775": "account_locked", "533": "account_disabled", "701": "account_expired", "532": "password_expired", "773": "password_change_required", "52e": "invalid_credentials", "bad": "invalid_credentials"} {
		if got := adPasswordRejection("80090308: LdapErr: DSID-0C09030B, comment: AcceptSecurityContext error, data " + code + ", v893"); got != want {
			t.Errorf("%s=%s want %s", code, got, want)
		}
	}
	if got := adPasswordRejection("unknown directory error with private data"); got != "invalid_credentials" {
		t.Fatal(got)
	}
	if _, ok := identityFilter(domain.ActiveDirectory, "not-a-guid"); ok {
		t.Fatal("invalid AD id accepted")
	}
	if _, ok := identityFilter(domain.OpenLDAP, "*)(uid=*)"); ok {
		t.Fatal("LDAP filter injection accepted")
	}
}
