// Package testing_openldap は、統合テストのために OpenLDAP のコンテナを起動し、エントリを書き換える。
// Docker が使えなければテストをスキップする。
package testing_openldap

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"
)

const (
	image = "osixia/openldap:1.5.0"
	// ServerName は証明書の名前である。コンテナのホスト名にする。
	ServerName    = "ldap.test"
	BaseDN        = "dc=example,dc=test"
	PeopleDN      = "ou=people," + BaseDN
	AdminDN       = "cn=admin," + BaseDN
	AdminPassword = "admin-secret"
)

type Server struct {
	// LDAPURL は StartTLS で使う ldap:// の URL、LDAPSURL は ldaps:// の URL である。
	LDAPURL  string
	LDAPSURL string
	// CA はサーバーの証明書を発行した CA の PEM である。
	CA []byte
	id string
}

// Start は OpenLDAP のコンテナを起動し、people の OU を作って返す。コンテナはテストの終わりに消す。
func Start(t *testing.T) *Server {
	t.Helper()
	if err := exec.CommandContext(t.Context(), "docker", "info").Run(); err != nil {
		t.Skipf("docker is not available, skipping the OpenLDAP integration test: %v", err)
	}
	// イメージに同梱の証明書は期限が切れているので、テストごとに CA とサーバーの証明書を作って入れる。
	ca, certificate, key := issueCertificates(t)
	id := strings.TrimSpace(docker(t, "create", "--hostname", ServerName,
		"-p", "127.0.0.1::389", "-p", "127.0.0.1::636",
		"-e", "LDAP_ORGANISATION=Example", "-e", "LDAP_DOMAIN=example.test",
		"-e", "LDAP_ADMIN_PASSWORD="+AdminPassword, "-e", "LDAP_TLS_VERIFY_CLIENT=never",
		"-e", "LDAP_TLS_CRT_FILENAME=server.crt", "-e", "LDAP_TLS_KEY_FILENAME=server.key", "-e", "LDAP_TLS_CA_CRT_FILENAME=ca.crt",
		image))
	t.Cleanup(func() { _ = exec.CommandContext(context.Background(), "docker", "rm", "-f", id).Run() })
	dir := t.TempDir()
	for name, content := range map[string][]byte{"ca.crt": ca, "server.crt": certificate, "server.key": key} {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
		docker(t, "cp", filepath.Join(dir, name), id+":/container/service/slapd/assets/certs/"+name)
	}
	docker(t, "start", id)
	s := &Server{
		LDAPURL:  "ldap://" + strings.TrimSpace(docker(t, "port", id, "389/tcp")),
		LDAPSURL: "ldaps://" + strings.TrimSpace(docker(t, "port", id, "636/tcp")),
		id:       id,
	}
	deadline := time.Now().Add(60 * time.Second)
	for {
		conn, err := s.dialAdmin()
		if err == nil {
			_ = conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("OpenLDAP did not become ready: %v\n%s", err, docker(t, "logs", id))
		}
		time.Sleep(500 * time.Millisecond)
	}
	s.CA = ca
	s.Do(t, func(conn *ldap.Conn) error {
		add := ldap.NewAddRequest(PeopleDN, nil)
		add.Attribute("objectClass", []string{"organizationalUnit"})
		add.Attribute("ou", []string{"people"})
		return conn.Add(add)
	})
	return s
}

// TLS は CA を信頼し、証明書の名前を検証する構成である。
func (s *Server) TLS(t *testing.T) *tls.Config {
	t.Helper()
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(s.CA) {
		t.Fatal("OpenLDAP CA is not PEM")
	}
	return &tls.Config{RootCAs: pool, ServerName: ServerName, MinVersion: tls.VersionTLS12}
}

// Do は管理者として bind した接続で change を行う。
func (s *Server) Do(t *testing.T, change func(*ldap.Conn) error) {
	t.Helper()
	conn, err := s.dialAdmin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if err := change(conn); err != nil {
		t.Fatal(err)
	}
}

// AddPerson は people の OU に inetOrgPerson を足す。
func (s *Server) AddPerson(t *testing.T, uid, cn, mail string) {
	t.Helper()
	s.Do(t, func(conn *ldap.Conn) error {
		add := ldap.NewAddRequest(fmt.Sprintf("uid=%s,%s", uid, PeopleDN), nil)
		add.Attribute("objectClass", []string{"inetOrgPerson"})
		add.Attribute("uid", []string{uid})
		add.Attribute("cn", []string{cn})
		add.Attribute("sn", []string{cn})
		add.Attribute("mail", []string{mail})
		return conn.Add(add)
	})
}

// EntryUUID は uid のエントリの entryUUID を読む。
func (s *Server) EntryUUID(t *testing.T, uid string) string {
	t.Helper()
	var value string
	s.Do(t, func(conn *ldap.Conn) error {
		result, err := conn.Search(ldap.NewSearchRequest(PeopleDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			fmt.Sprintf("(uid=%s)", ldap.EscapeFilter(uid)), []string{"entryUUID"}, nil))
		if err != nil {
			return err
		}
		if len(result.Entries) != 1 {
			return fmt.Errorf("uid %s: %d entries", uid, len(result.Entries))
		}
		value = result.Entries[0].GetAttributeValue("entryUUID")
		return nil
	})
	return value
}

func (s *Server) dialAdmin() (*ldap.Conn, error) {
	conn, err := ldap.DialURL(s.LDAPURL)
	if err != nil {
		return nil, err
	}
	if err := conn.Bind(AdminDN, AdminPassword); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

// issueCertificates は ServerName のサーバーの証明書と、それを発行した CA を PEM で返す。
func issueCertificates(t *testing.T) (ca, certificate, key []byte) {
	t.Helper()
	now := time.Now()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "OpenLDAP test CA"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: ServerName}, DNSNames: []string{ServerName},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}, caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(serverKey)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
}

func docker(t *testing.T, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var stdout, stderr bytes.Buffer
	command := exec.CommandContext(ctx, "docker", args...)
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("docker %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String()
}
