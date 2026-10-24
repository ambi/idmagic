// Package testing_kerberos は、暗号化した実物のサービスチケットを作るテスト専用のフィクスチャである。
// KDC との相互運用の証拠とは分け、検証器の暗号処理と HTTP 配線を決定的に検査する。
package testing_kerberos

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/domain"
	"github.com/otuschhoff/gokrb5/v8/client"
	"github.com/otuschhoff/gokrb5/v8/config"
	"github.com/otuschhoff/gokrb5/v8/gssapi"
	"github.com/otuschhoff/gokrb5/v8/iana/etypeID"
	"github.com/otuschhoff/gokrb5/v8/iana/nametype"
	"github.com/otuschhoff/gokrb5/v8/keytab"
	"github.com/otuschhoff/gokrb5/v8/messages"
	"github.com/otuschhoff/gokrb5/v8/spnego"
	"github.com/otuschhoff/gokrb5/v8/types"
)

type Fixture struct {
	Config      domain.Config
	Certificate *x509.Certificate
	Keytab      *keytab.Keytab
	Directory   string
	KVNO        int
	GSSAPIFlags []int
}

func New(tb testing.TB) *Fixture {
	tb.Helper()
	directory := tb.TempDir()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	now := time.Now().UTC()
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), DNSNames: []string{"sso.example.com"}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		tb.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		tb.Fatal(err)
	}
	certificateFile := filepath.Join(directory, "tls.pem")
	if err := os.WriteFile(certificateFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		tb.Fatal(err)
	}
	kt := keytab.New()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		tb.Fatal(err)
	}
	if err := kt.AddKey(keytab.Principal{Realm: "EXAMPLE.COM", Components: []string{"HTTP", "sso.example.com"}, NameType: nametype.KRB_NT_SRV_INST}, 1,
		types.EncryptionKey{KeyType: etypeID.AES256_CTS_HMAC_SHA1_96, KeyValue: key}, now); err != nil {
		tb.Fatal(err)
	}
	encoded, err := kt.Marshal()
	if err != nil {
		tb.Fatal(err)
	}
	keytabFile := filepath.Join(directory, "http.keytab")
	if err := os.WriteFile(keytabFile, encoded, 0o600); err != nil {
		tb.Fatal(err)
	}
	return &Fixture{Certificate: cert, Keytab: kt, Directory: directory, KVNO: 1, GSSAPIFlags: []int{gssapi.ContextFlagMutual, gssapi.ContextFlagInteg, gssapi.ContextFlagConf}, Config: domain.Config{
		Realm: "EXAMPLE.COM", ServicePrincipal: "HTTP/sso.example.com", KeytabFile: keytabFile, CertificateFile: certificateFile,
		IntranetCIDRs: []string{"203.0.113.0/24"}, Bindings: []domain.Binding{{Principal: "alice", UserID: "alice-id"}},
	}}
}

func (f *Fixture) WriteConfiguration(tb testing.TB, tenantID string, automatic bool) string {
	tb.Helper()
	f.Config.TenantID = tenantID
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		tb.Fatal(err)
	}
	//nolint:gosec // テスト用の一時ファイルへ、生成した試行 Cookie の鍵を明示的に書き込む。
	encoded, err := json.Marshal(struct {
		Secret    string          `json:"attempt_secret"`
		Automatic bool            `json:"automatic"`
		Tenants   []domain.Config `json:"tenants"`
	}{base64.StdEncoding.EncodeToString(secret), automatic, []domain.Config{f.Config}})
	if err != nil {
		tb.Fatal(err)
	}
	path := filepath.Join(f.Directory, "negotiate.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		tb.Fatal(err)
	}
	return path
}

// Token は、指定した principal の暗号化した AP_REQ と CBT を持つ SPNEGO を返す。
func (f *Fixture) Token(tb testing.TB, principal, realm string, endTime time.Time, bindCertificate *x509.Certificate) []byte {
	tb.Helper()
	return f.TokenAt(tb, principal, realm, endTime, bindCertificate, time.Now().UTC())
}

// TokenAt は authenticator の時刻差も指定し、KDC の鍵で暗号化した検査用チケットを返す。
func (f *Fixture) TokenAt(tb testing.TB, principal, realm string, endTime time.Time, bindCertificate *x509.Certificate, authenticatorTime time.Time) []byte {
	tb.Helper()
	now := time.Now().UTC()
	ticket, sessionKey, err := messages.NewTicket(types.NewPrincipalName(nametype.KRB_NT_PRINCIPAL, principal), realm,
		types.NewPrincipalName(nametype.KRB_NT_SRV_INST, f.Config.ServicePrincipal), f.Config.Realm, types.NewKrbFlags(), f.Keytab,
		etypeID.AES256_CTS_HMAC_SHA1_96, f.KVNO, now.Add(-time.Minute), now.Add(-time.Minute), endTime, endTime)
	if err != nil {
		tb.Fatal(err)
	}
	var bindings *gssapi.ChannelBindings
	if bindCertificate != nil {
		value, err := gssapi.TLSServerEndPoint(tls.ConnectionState{PeerCertificates: []*x509.Certificate{bindCertificate}})
		if err != nil {
			tb.Fatal(err)
		}
		bindings = &value
	}
	cl := client.NewWithPassword(principal, realm, "unused", config.New())
	negotiation, err := spnego.NewNegTokenInitKRB5WithOptions(cl, ticket, sessionKey, spnego.KRB5TokenAPREQOptions{
		GSSAPIFlags: f.GSSAPIFlags, ChannelBindings: bindings,
	})
	if err != nil {
		tb.Fatal(err)
	}
	var mechanism spnego.KRB5Token
	if err := mechanism.Unmarshal(negotiation.MechTokenBytes); err != nil {
		tb.Fatal(err)
	}
	if err := mechanism.APReq.DecryptAuthenticator(sessionKey); err != nil {
		tb.Fatal(err)
	}
	authenticator := mechanism.APReq.Authenticator
	authenticator.CTime = authenticatorTime
	authenticator.Cusec = authenticatorTime.Nanosecond() / 1000
	mechanism.APReq, err = messages.NewAPReq(ticket, sessionKey, authenticator)
	if err != nil {
		tb.Fatal(err)
	}
	negotiation.MechTokenBytes, err = mechanism.Marshal()
	if err != nil {
		tb.Fatal(err)
	}
	token, err := (&spnego.SPNEGOToken{Init: true, NegTokenInit: negotiation}).Marshal()
	if err != nil {
		tb.Fatal(err)
	}
	return token
}
