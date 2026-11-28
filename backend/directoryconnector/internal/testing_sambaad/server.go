// Package testing_sambaad は、統合テストのために隔離した Samba AD の DC を起動する。
// Active Directory の読み取り方（objectGUID、userAccountControl、uSNChanged、rootDSE）を実物の DC で試すためにある。
// イメージは Kerberos の試験と同じ infra/test/kerberos を使い、LDAPS には SAN を持つテストの証明書を渡す。
package testing_sambaad

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
	"math/big"
	"net"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"
)

const (
	image = "idmagic-kerberos-test"
	// ServerName はテストの証明書の名前である。Samba の DC のホスト名（dc.example.com）に合わせる。
	ServerName = "dc.example.com"
	BaseDN     = "DC=example,DC=com"
	UsersDN    = "CN=Users," + BaseDN
	// AdminDN と AdminPassword は infra/test/kerberos/start.sh の隔離した検証用の値である。
	AdminDN       = "CN=Administrator," + UsersDN
	AdminPassword = "Test-Admin-Password-123!"
	// Alice は start.sh が作る利用者である。
	Alice         = "alice"
	AlicePassword = "Test-Alice-Password-123!"
)

type Server struct {
	// LDAPSURL は ldaps:// の URL である。
	LDAPSURL string
	// CA はサーバーの証明書を発行した CA の PEM である。
	CA []byte
	id string
}

// Start は Samba AD の DC を起動し、LDAPS で管理者として bind できるまで待つ。コンテナはテストの終わりに消す。
func Start(t *testing.T) *Server {
	t.Helper()
	if output, err := exec.CommandContext(t.Context(), "docker", "info").CombinedOutput(); err != nil {
		t.Fatalf("Samba AD 試験には稼働中の Docker が必要: %v\n%s", err, output)
	}
	_, here, _, _ := runtime.Caller(0)
	docker(t, 10*time.Minute, "build", "--quiet", "--tag", image, filepath.Join(filepath.Dir(here), "..", "..", "..", "..", "infra", "test", "kerberos"))
	ca, certificate, key := issueCertificates(t)
	id := strings.TrimSpace(docker(t, time.Minute, "run", "--detach", "--hostname", "dc",
		"-p", "127.0.0.1::636",
		"-e", "TLS_CERT_PEM="+string(certificate), "-e", "TLS_KEY_PEM="+string(key), "-e", "TLS_CA_PEM="+string(ca),
		image))
	t.Cleanup(func() { _ = exec.CommandContext(context.Background(), "docker", "rm", "-f", id).Run() })
	s := &Server{CA: ca, id: id}
	// 起動の直後は provision の途中で、636 の待受けがまだない。docker port は待受けの前でも返るので、bind で確かめる。
	deadline := time.Now().Add(3 * time.Minute)
	for {
		if s.LDAPSURL == "" {
			if port, err := exec.CommandContext(t.Context(), "docker", "port", id, "636/tcp").Output(); err == nil {
				s.LDAPSURL = "ldaps://" + strings.TrimSpace(strings.SplitN(string(port), "\n", 2)[0])
			}
		}
		if s.LDAPSURL != "" {
			conn, err := s.dial(t, AdminDN, AdminPassword)
			if err == nil {
				_ = conn.Close()
				return s
			}
			if time.Now().After(deadline) {
				t.Fatalf("Samba AD did not become ready: %v\n%s", err, docker(t, time.Minute, "logs", "--tail", "50", id))
			}
		}
		time.Sleep(time.Second)
	}
}

// TLS は CA を信頼し、証明書の名前を検証する構成である。
func (s *Server) TLS(t *testing.T) *tls.Config {
	t.Helper()
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(s.CA) {
		t.Fatal("Samba AD CA is not PEM")
	}
	return &tls.Config{RootCAs: pool, ServerName: ServerName, MinVersion: tls.VersionTLS12}
}

// Do は管理者として bind した接続で change を行う。
func (s *Server) Do(t *testing.T, change func(*ldap.Conn) error) {
	t.Helper()
	conn, err := s.dial(t, AdminDN, AdminPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if err := change(conn); err != nil {
		t.Fatal(err)
	}
}

// SambaTool は DC の中で samba-tool を実行する。利用者の無効化など、AD の手順そのものを使う変更に使う。
func (s *Server) SambaTool(t *testing.T, args ...string) string {
	t.Helper()
	return docker(t, time.Minute, append([]string{"exec", s.id, "samba-tool"}, args...)...)
}

func (s *Server) dial(t *testing.T, bindDN, password string) (*ldap.Conn, error) {
	t.Helper()
	conn, err := ldap.DialURL(s.LDAPSURL, ldap.DialWithTLSConfig(s.TLS(t)), ldap.DialWithDialer(&net.Dialer{Timeout: 2 * time.Second}))
	if err != nil {
		return nil, err
	}
	if err := conn.Bind(bindDN, password); err != nil {
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
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Samba AD test CA"},
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

func docker(t *testing.T, timeout time.Duration, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var stdout, stderr bytes.Buffer
	command := exec.CommandContext(ctx, "docker", args...)
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("docker %s: %v\n%s", strings.Join(args[:1], " "), err, stderr.String())
	}
	return stdout.String()
}
