package ldap_goldap_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ldap_goldap"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/testing_openldap"
)

func config(url string, startTLS bool, tlsConfig *tls.Config) ldap_goldap.Config {
	return ldap_goldap.Config{
		URL: url, StartTLS: startTLS, TLS: tlsConfig, Dialect: domain.OpenLDAP,
		BindDN: testing_openldap.AdminDN, BindPassword: testing_openldap.AdminPassword, PageSize: 2, Timeout: 10 * time.Second,
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-002: ldap:// の URL で StartTLS を指定しない構成を、接続する前に拒否する。
func TestDirectoryRefusesPlaintextLDAP(t *testing.T) {
	if _, err := ldap_goldap.New(config("ldap://127.0.0.1:1", false, nil)); !errors.Is(err, ldap_goldap.ErrPlaintextLDAP) {
		t.Fatalf("error = %v, want ErrPlaintextLDAP", err)
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-002: 構成した CA で検証できない証明書の LDAP サーバーには、LDAPS でも StartTLS でも bind しない。
func TestDirectoryRejectsAnUntrustedServerCertificate(t *testing.T) {
	server := testing_openldap.Start(t)
	untrusted := &tls.Config{RootCAs: x509.NewCertPool(), ServerName: testing_openldap.ServerName, MinVersion: tls.VersionTLS12}
	wrongName := server.TLS(t)
	wrongName.ServerName = "other.test"
	for name, c := range map[string]ldap_goldap.Config{
		"ldaps with an unknown ca":    config(server.LDAPSURL, false, untrusted),
		"starttls with an unknown ca": config(server.LDAPURL, true, untrusted),
		"ldaps with another name":     config(server.LDAPSURL, false, wrongName),
	} {
		directory, err := ldap_goldap.New(c)
		if err != nil {
			t.Fatal(err)
		}
		session, err := directory.Connect(context.Background())
		if err == nil {
			_ = session.Close()
			t.Fatalf("%s: connected", name)
		}
		if !strings.Contains(err.Error(), "certificate") {
			t.Fatalf("%s: error = %v, want a certificate failure", name, err)
		}
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-002, REQ-DIRECTORYCONNECTOR-004: 検証した TLS で bind し、要求した属性をページごとに返し、誤ったパスワードの bind を失敗させる。
func TestDirectoryBindsOverTLSAndPagesTheSearch(t *testing.T) {
	server := testing_openldap.Start(t)
	for i := range 5 {
		server.AddPerson(t, fmt.Sprintf("u%d", i), fmt.Sprintf("User %d", i), fmt.Sprintf("u%d@example.test", i))
	}
	for name, c := range map[string]ldap_goldap.Config{
		"ldaps":    config(server.LDAPSURL, false, server.TLS(t)),
		"starttls": config(server.LDAPURL, true, server.TLS(t)),
	} {
		directory, err := ldap_goldap.New(c)
		if err != nil {
			t.Fatal(err)
		}
		session, err := directory.Connect(context.Background())
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if root, err := session.RootInfo(context.Background()); err != nil || root != (domain.RootInfo{}) {
			t.Fatalf("%s: root=%+v err=%v, want empty for openldap", name, root, err)
		}
		var pages []int
		var uids []string
		err = session.Search(context.Background(), testing_openldap.PeopleDN, "(objectClass=inetOrgPerson)", []string{"entryUUID", "uid"},
			func(entries []domain.Entry) error {
				pages = append(pages, len(entries))
				for _, entry := range entries {
					if len(entry.Attributes) != 2 {
						t.Errorf("%s: %s attributes = %+v, want only the requested", name, entry.DN, entry.Attributes)
					}
					for _, attribute := range entry.Attributes {
						if attribute.Name == "uid" {
							uids = append(uids, string(attribute.Values[0]))
						}
					}
				}
				return nil
			})
		_ = session.Close()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		slices.Sort(uids)
		if !slices.Equal(pages, []int{2, 2, 1}) || !slices.Equal(uids, []string{"u0", "u1", "u2", "u3", "u4"}) {
			t.Fatalf("%s: pages=%v uids=%v", name, pages, uids)
		}
	}

	wrong := config(server.LDAPSURL, false, server.TLS(t))
	wrong.BindPassword = "wrong"
	directory, err := ldap_goldap.New(wrong)
	if err != nil {
		t.Fatal(err)
	}
	if session, err := directory.Connect(context.Background()); err == nil {
		_ = session.Close()
		t.Fatal("bind with a wrong password succeeded")
	} else if strings.Contains(err.Error(), "wrong") {
		t.Fatalf("error leaks the password: %v", err)
	}
}
