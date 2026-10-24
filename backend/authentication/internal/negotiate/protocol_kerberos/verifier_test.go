package protocol_kerberos_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/protocol_kerberos"
	fixture "github.com/ambi/idmagic/backend/authentication/testing_kerberos"
	"github.com/otuschhoff/gokrb5/v8/keytab"
)

//spec:covers REQ-AUTHENTICATION-042: 起動前の診断は keytab の realm と SPN、鍵と証明書の存在を検査する。
func TestTrustFileDiagnosticsRejectMismatches(t *testing.T) {
	f := fixture.New(t)
	if err := protocol_kerberos.CheckFiles(f.Config); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"realm", "spn", "keytab", "certificate"} {
		t.Run(name, func(t *testing.T) {
			config := f.Config
			switch name {
			case "realm":
				config.Realm = "OTHER.COM"
			case "spn":
				config.ServicePrincipal = "HTTP/other.example.com"
			case "keytab":
				config.KeytabFile += ".missing"
			case "certificate":
				config.CertificateFile += ".missing"
			}
			if err := protocol_kerberos.CheckFiles(config); err == nil {
				t.Fatal("invalid trust accepted")
			}
		})
	}
}

//spec:covers REQ-AUTHENTICATION-042: 実物の暗号化チケットと CBT を検証し、期限、realm、再送、改竄と CBT の欠落を拒否する。
func TestVerifierAcceptsBoundTicketAndRejectsInvalidCredentials(t *testing.T) {
	f := fixture.New(t)
	verifier := protocol_kerberos.Verifier{}
	token := f.Token(t, "alice", f.Config.Realm, time.Now().Add(time.Hour), f.Certificate)
	verified, err := verifier.Verify(context.Background(), f.Config, token, "203.0.113.10")
	if err != nil || verified.Principal.Name != "alice" || verified.ReplayID == "" || len(verified.Reply) == 0 {
		t.Fatalf("verified=%+v err=%v", verified, err)
	}
	if _, err := verifier.Verify(context.Background(), f.Config, token, "203.0.113.10"); err == nil {
		t.Fatal("replay accepted")
	}
	for _, tc := range []struct {
		name, realm string
		expiry      time.Time
		bound       bool
	}{
		{"expired", f.Config.Realm, time.Now().Add(-time.Second), true},
		{"wrong-realm", "OTHER.COM", time.Now().Add(time.Hour), true},
		{"missing-binding", f.Config.Realm, time.Now().Add(time.Hour), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			certificate := f.Certificate
			if !tc.bound {
				certificate = nil
			}
			bad := f.Token(t, "alice", tc.realm, tc.expiry, certificate)
			if _, err := verifier.Verify(context.Background(), f.Config, bad, "203.0.113.10"); err == nil {
				t.Fatal("invalid ticket accepted")
			}
		})
	}
	token[len(token)-1] ^= 1
	if _, err := verifier.Verify(context.Background(), f.Config, token, "203.0.113.10"); err == nil {
		t.Fatal("tampering accepted")
	}
}

// THREAT-083: 異なるサービス名、証明書と時刻のチケットを、暗号化された実物の入力で拒否する。
//
//spec:covers REQ-AUTHENTICATION-042: SPN、TLS の終端と 5 分の許容時刻差を独立に検証する。
func TestVerifierRejectsWrongSPNBindingAndClock(t *testing.T) {
	for _, name := range []string{"spn", "binding", "past-clock", "future-clock", "ntlm"} {
		t.Run(name, func(t *testing.T) {
			f := fixture.New(t)
			config := f.Config
			cert, authTime := f.Certificate, time.Now().UTC()
			if name == "spn" {
				config.ServicePrincipal = "HTTP/other.example.com"
			}
			if name == "binding" {
				cert = fixture.New(t).Certificate
			}
			if name == "past-clock" {
				authTime = authTime.Add(-6 * time.Minute)
			}
			if name == "future-clock" {
				authTime = authTime.Add(6 * time.Minute)
			}
			token := f.TokenAt(t, "alice", config.Realm, time.Now().Add(time.Hour), cert, authTime)
			if name == "ntlm" {
				token = []byte("NTLMSSP\x00")
			}
			if _, err := (protocol_kerberos.Verifier{}).Verify(context.Background(), config, token, ""); err == nil {
				t.Fatal("invalid credential accepted")
			}
		})
	}
}

//spec:covers REQ-AUTHENTICATION-042: 同じパスへ新旧 KVNO を併記した keytab を原子的に配備すると、再起動なしで両世代を検証する。
func TestVerifierReloadsOverlappingKeyVersions(t *testing.T) {
	old, next := fixture.New(t), fixture.New(t)
	next.KVNO = 2
	next.Keytab.Entries[0].KVNO = 2
	combined := keytab.New()
	combined.Entries = append(combined.Entries, old.Keytab.Entries...)
	combined.Entries = append(combined.Entries, next.Keytab.Entries...)
	data, err := combined.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	verifier := protocol_kerberos.Verifier{}
	if _, err := verifier.Verify(context.Background(), old.Config, old.Token(t, "alice", old.Config.Realm, time.Now().Add(time.Hour), old.Certificate), ""); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(old.Config.KeytabFile+".next", data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(old.Config.KeytabFile+".next", old.Config.KeytabFile); err != nil {
		t.Fatal(err)
	}
	for _, generation := range []*fixture.Fixture{old, next} {
		token := generation.Token(t, "alice", old.Config.Realm, time.Now().Add(time.Hour), old.Certificate)
		if _, err := verifier.Verify(context.Background(), old.Config, token, ""); err != nil {
			t.Fatal(err)
		}
	}
}
