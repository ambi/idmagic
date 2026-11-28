package protocol_kerberos_test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/otuschhoff/gokrb5/v8/crypto"
	"github.com/otuschhoff/gokrb5/v8/gssapi"
	"github.com/otuschhoff/gokrb5/v8/iana/nametype"
	"github.com/otuschhoff/gokrb5/v8/keytab"
	"github.com/otuschhoff/gokrb5/v8/spnego"
	"github.com/otuschhoff/gokrb5/v8/types"
)

// chromeNegotiate は frontend の外の記録の道具（chrome_record_test.go、mise run record-negotiate）が
// 保存した、手元の Chrome が送った SPNEGO の初期トークンである。
type chromeNegotiate struct {
	RetrievedAt             string `json:"retrieved_at"`
	Client                  string `json:"client"`
	Method                  string `json:"method"`
	CertificatePEM          string `json:"certificate_pem"`
	Token                   string `json:"token"`
	VerifiedAtRecording     bool   `json:"verified_at_recording"`
	VerificationAtRecording string `json:"verification_at_recording"`
}

// recordedServiceKeytab は、記録の Samba AD のサービスのアカウント svc-sso の鍵を、そのパスワードから作り直す。
// パスワードは infra/test/kerberos/start.sh の隔離した検証用の値である。
// RC4-HMAC の鍵は salt を使わないので、パスワードだけから決まる。
func recordedServiceKeytab(t *testing.T, etypeID int32, kvno uint32) *keytab.Keytab {
	t.Helper()
	etype, err := crypto.GetEtype(etypeID)
	if err != nil {
		t.Fatal(err)
	}
	// AD のユーザーのアカウントの AES の salt は realm と sAMAccountName である。RC4 では使われない。
	key, err := etype.StringToKey("Test-Service-Password-123!", "EXAMPLE.COMsvc-sso", etype.GetDefaultStringToKeyParams())
	if err != nil {
		t.Fatal(err)
	}
	kt := keytab.New()
	if err := kt.AddKey(keytab.Principal{Realm: "EXAMPLE.COM", Components: []string{"HTTP", "sso.example.com"}, NameType: nametype.KRB_NT_SRV_INST}, kvno,
		types.EncryptionKey{KeyType: etypeID, KeyValue: key}, time.Now()); err != nil {
		t.Fatal(err)
	}
	return kt
}

// 手元の Chrome が送った Negotiate のトークンを、時刻に依らない判断だけで読む。
// チケットと authenticator の時刻は記録のときの値なので、gokrb5 の受理（時刻の検査を含む）は後日には通らない。
// 受理の結果は記録のときに Verifier で観測し、verification_at_recording に残した。
//
// RFC 4178 の NegTokenInit が Kerberos の AP-REQ を運び、RFC 4121 §4.1.1 の authenticator のチェックサム（0x8003）の
// Bnd が RFC 5929 の tls-server-end-point の束縛の MD5 を持つかを確かめる。IdMagic は束縛を必須とする（negotiate.md）。
// macOS の Chrome 158 の記録では Bnd がすべて 0、つまり束縛を送っておらず、記録のときも束縛の不一致で拒否された。
// 束縛のない Chrome を受け付けるかは wi-55734 が決めるまで、ここでは記録の事実を固定する。
//
//spec:covers REQ-AUTHENTICATION-042: 手元の Chrome が送った SPNEGO の初期トークンの AP-REQ を、Samba AD のサービスの鍵で復号し、SPN と realm と、束縛を運ぶチェックサムの形を読むこと
func TestChromeNegotiate_TokenCarriesNoChannelBindings(t *testing.T) {
	data, err := os.ReadFile("testdata/chrome-negotiate.json")
	if err != nil {
		t.Fatal(err)
	}
	var rec chromeNegotiate
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatal(err)
	}
	if rec.RetrievedAt == "" || rec.Client == "" || rec.Method == "" {
		t.Fatal("the recording must state its retrieval date, client and method")
	}
	token, err := base64.StdEncoding.DecodeString(rec.Token)
	if err != nil {
		t.Fatal(err)
	}

	var negotiation spnego.SPNEGOToken
	if err := negotiation.Unmarshal(token); err != nil || !negotiation.Init {
		t.Fatalf("Chrome's token is not an SPNEGO NegTokenInit: %v", err)
	}
	if len(negotiation.NegTokenInit.MechTypes) == 0 || !negotiation.NegTokenInit.MechTypes[0].Equal(gssapi.OIDKRB5.OID()) {
		t.Fatalf("mechTypes = %v, want Kerberos V5 first", negotiation.NegTokenInit.MechTypes)
	}
	var mechanism spnego.KRB5Token
	if err := mechanism.Unmarshal(negotiation.NegTokenInit.MechTokenBytes); err != nil || !mechanism.IsAPReq() {
		t.Fatalf("the mechanism token is not a Kerberos AP-REQ: %v", err)
	}
	request := &mechanism.APReq
	if request.Ticket.Realm != "EXAMPLE.COM" || request.Ticket.SName.PrincipalNameString() != "HTTP/sso.example.com" {
		t.Fatalf("ticket for %s@%s, want HTTP/sso.example.com@EXAMPLE.COM", request.Ticket.SName.PrincipalNameString(), request.Ticket.Realm)
	}

	kt := recordedServiceKeytab(t, request.Ticket.EncPart.EType, uint32(request.Ticket.EncPart.KVNO))
	if err := request.Ticket.DecryptEncPart(kt, nil); err != nil {
		t.Fatalf("decrypt the ticket with the service key: %v", err)
	}
	if err := request.DecryptAuthenticator(request.Ticket.DecryptedEncPart.Key); err != nil {
		t.Fatalf("decrypt the authenticator: %v", err)
	}
	if request.Authenticator.CRealm != "EXAMPLE.COM" || len(request.Authenticator.CName.NameString) != 1 || request.Authenticator.CName.NameString[0] != "alice" {
		t.Fatalf("client = %v@%s, want alice@EXAMPLE.COM", request.Authenticator.CName.NameString, request.Authenticator.CRealm)
	}

	checksum := request.Authenticator.Cksum
	if checksum.CksumType != 0x8003 || len(checksum.Checksum) < 24 || binary.LittleEndian.Uint32(checksum.Checksum[0:4]) != 16 {
		t.Fatalf("authenticator checksum type %#x of %d byte(s), want the RFC 4121 GSS checksum", checksum.CksumType, len(checksum.Checksum))
	}
	bnd := checksum.Checksum[4:20]

	block, _ := pem.Decode([]byte(rec.CertificatePEM))
	if block == nil {
		t.Fatal("the recording has no server certificate")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := gssapi.TLSServerEndPoint(tls.ConnectionState{PeerCertificates: []*x509.Certificate{certificate}})
	if err != nil {
		t.Fatal(err)
	}
	expected := bindings.MD5Hash()
	if bytes.Equal(bnd, expected[:]) {
		t.Fatalf("Chrome now sends the tls-server-end-point binding; record again and accept it (wi-55734)")
	}
	if !bytes.Equal(bnd, make([]byte, 16)) {
		t.Fatalf("Bnd = %x, want either the expected binding %x or all zero (no binding)", bnd, expected)
	}
	if rec.VerifiedAtRecording {
		t.Fatalf("a token without channel bindings was accepted at recording: %s", rec.VerificationAtRecording)
	}
}
