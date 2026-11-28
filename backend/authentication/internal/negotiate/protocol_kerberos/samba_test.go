package protocol_kerberos_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	negotiateports "github.com/ambi/idmagic/backend/authentication/internal/negotiate/ports"
	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/protocol_kerberos"
	fixture "github.com/ambi/idmagic/backend/authentication/testing_kerberos"
	"github.com/otuschhoff/gokrb5/v8/client"
	"github.com/otuschhoff/gokrb5/v8/config"
	"github.com/otuschhoff/gokrb5/v8/gssapi"
	"github.com/otuschhoff/gokrb5/v8/spnego"
)

//spec:covers REQ-AUTHENTICATION-042: Samba AD の KDC が実際に発行したチケットと PAC を、サービス keytab と CBT で検証し、RC4 のアカウントのチケットを暗号方式の拒否として区別する。
func TestVerifierAgainstSambaAD(t *testing.T) {
	address := os.Getenv("KERBEROS_KDC_ADDR")
	if address == "" {
		t.Skip("run mise run test-kerberos-ad to start the isolated domain controller")
	}
	// KDC の状態は Go の結果キャッシュの鍵に入らないため、実行ごとの値で再利用を防ぐ。
	_ = os.Getenv("VERIFICATION_RUN_ID")
	deadline := time.Now().Add(20 * time.Second)
	for {
		connection, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			_ = connection.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("KDC unavailable: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	configuration, err := config.NewFromString(fmt.Sprintf(`[libdefaults]
 default_realm = EXAMPLE.COM
 dns_lookup_kdc = false
 dns_lookup_realm = false
 udp_preference_limit = 1
[realms]
 EXAMPLE.COM = {
  kdc = %s
 }
`, address))
	if err != nil {
		t.Fatal(err)
	}
	cl := client.NewWithPassword("alice", "EXAMPLE.COM", "Test-Alice-Password-123!", configuration, client.DisablePAReqEncPARep(true))
	defer cl.Destroy()
	if err := cl.Login(); err != nil {
		t.Fatalf("AD login failed: %v", err)
	}
	f := fixture.New(t)
	verify := func(t *testing.T, spn, keytabFile string) (negotiateports.Verified, error) {
		t.Helper()
		ticket, sessionKey, err := cl.GetServiceTicket(spn)
		if err != nil {
			t.Fatalf("AD TGS failed: %v", err)
		}
		bindings, err := gssapi.TLSServerEndPoint(tls.ConnectionState{PeerCertificates: []*x509.Certificate{f.Certificate}})
		if err != nil {
			t.Fatal(err)
		}
		initial, err := spnego.NewNegTokenInitKRB5WithOptions(cl, ticket, sessionKey, spnego.KRB5TokenAPREQOptions{
			GSSAPIFlags: []int{gssapi.ContextFlagMutual, gssapi.ContextFlagInteg, gssapi.ContextFlagConf}, ChannelBindings: &bindings,
		})
		if err != nil {
			t.Fatal(err)
		}
		token, err := (&spnego.SPNEGOToken{Init: true, NegTokenInit: initial}).Marshal()
		if err != nil {
			t.Fatal(err)
		}
		config := f.Config
		config.ServicePrincipal, config.KeytabFile = spn, keytabFile
		return (protocol_kerberos.Verifier{}).Verify(context.Background(), config, token, "")
	}
	// start.sh は svc-sso の msDS-SupportedEncryptionTypes を AES に限る。
	t.Run("aes-service-account", func(t *testing.T) {
		verified, err := verify(t, "HTTP/sso.example.com", os.Getenv("KERBEROS_KEYTAB_FILE"))
		if err != nil || verified.Principal.Name != "alice" || verified.Principal.Realm != "EXAMPLE.COM" || len(verified.Reply) == 0 {
			t.Fatalf("AD verification=%+v err=%v", verified, err)
		}
	})
	// start.sh は svc-legacy の msDS-SupportedEncryptionTypes を設定しない。keytab は AES の鍵も持つが、KDC は RC4 でチケットを出す。
	t.Run("rc4-service-account", func(t *testing.T) {
		_, err := verify(t, "HTTP/legacy.example.com", os.Getenv("KERBEROS_LEGACY_KEYTAB_FILE"))
		if !errors.Is(err, negotiateports.ErrUnsupportedEncryption) {
			t.Fatalf("RC4 ticket: err=%v, want ErrUnsupportedEncryption", err)
		}
	})
}
