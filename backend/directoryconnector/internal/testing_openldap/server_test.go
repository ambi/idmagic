package testing_openldap

import (
	"crypto/x509"
	"errors"
	"testing"
)

func TestAdminConnectionRejectsAnotherCertificateAuthority(t *testing.T) {
	server := Start(t)
	otherCA, _, _ := issueCertificates(t)
	server.CA = otherCA
	conn, err := server.dialAdmin(t)
	if conn != nil {
		_ = conn.Close()
	}
	if _, ok := errors.AsType[x509.UnknownAuthorityError](err); !ok {
		t.Fatalf("別の CA による管理接続のエラー = %v、期待値は証明書の信頼エラー", err)
	}
}
