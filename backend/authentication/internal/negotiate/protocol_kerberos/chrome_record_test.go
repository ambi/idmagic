//go:build kerberosrecord

package protocol_kerberos_test

// mise run record-negotiate が使う記録の道具である。通常のテストには入らない。
//
// 隔離した Samba AD の KDC で利用者 alice のチケットを kinit で取り、手元の Chrome に Negotiate を許して
// sso.example.com の TLS の待ち受けを開かせ、Chrome が送った Authorization: Negotiate のトークンを記録する。
// 記録のときに IdMagic の Verifier でトークンを検証し、その結果も記録に残す。
// 資格情報は Samba AD の隔離した検証用の値だけで、記録には残さない。

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	negotiatedomain "github.com/ambi/idmagic/backend/authentication/internal/negotiate/domain"
	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/protocol_kerberos"
	"github.com/otuschhoff/gokrb5/v8/gssapi"
	"github.com/otuschhoff/gokrb5/v8/keytab"
	"github.com/otuschhoff/gokrb5/v8/service"
	"github.com/otuschhoff/gokrb5/v8/spnego"
	"github.com/otuschhoff/gokrb5/v8/types"
)

// ChromeNegotiateRecording は Chrome が送った SPNEGO の初期トークンと、その相手の証明書である。
type ChromeNegotiateRecording struct {
	RetrievedAt             string `json:"retrieved_at"`
	Client                  string `json:"client"`
	KDC                     string `json:"kdc"`
	Method                  string `json:"method"`
	CertificatePEM          string `json:"certificate_pem"`
	Token                   string `json:"token"`
	VerifiedAtRecording     bool   `json:"verified_at_recording"`
	VerificationAtRecording string `json:"verification_at_recording"`
	DiagnosisAtRecording    string `json:"diagnosis_at_recording"`
}

// diagnose は、記録の直後にトークンの構造と gokrb5 の受理の結果を読む。Verifier は受理の理由を捨てるためである。
func diagnose(t *testing.T, token []byte, keytabFile, certificateFile string) string {
	t.Helper()
	var parts []string
	var negotiation spnego.SPNEGOToken
	if err := negotiation.Unmarshal(token); err != nil {
		return "SPNEGO unmarshal: " + err.Error()
	}
	parts = append(parts, fmt.Sprintf("mechTypes=%v reqFlags=%x mechListMIC=%d byte(s)", negotiation.NegTokenInit.MechTypes, negotiation.NegTokenInit.ReqFlags.Bytes, len(negotiation.NegTokenInit.MechListMIC)))
	var mechanism spnego.KRB5Token
	if err := mechanism.Unmarshal(negotiation.NegTokenInit.MechTokenBytes); err != nil {
		return strings.Join(append(parts, "KRB5 unmarshal: "+err.Error()), "; ")
	}
	parts = append(parts, fmt.Sprintf("mechTokenOID=%v kvno=%d etype=%d apOptions=%x", mechanism.OID, mechanism.APReq.Ticket.EncPart.KVNO, mechanism.APReq.Ticket.EncPart.EType, mechanism.APReq.APOptions.Bytes))
	kt, err := keytab.Load(keytabFile)
	if err != nil {
		return strings.Join(append(parts, "keytab: "+err.Error()), "; ")
	}
	if ok, err := mechanism.APReq.Verify(kt, 5*time.Minute, types.HostAddress{}, nil); !ok || err != nil {
		return strings.Join(append(parts, fmt.Sprintf("AP_REQ verify: ok=%v err=%v", ok, err)), "; ")
	}
	checksum := mechanism.APReq.Authenticator.Cksum
	parts = append(parts, fmt.Sprintf("authenticator cksumtype=%#x cksum=%x", checksum.CksumType, checksum.Checksum))
	certificate, err := protocol_kerberos.LoadCertificate(certificateFile)
	if err != nil {
		return strings.Join(append(parts, "certificate: "+err.Error()), "; ")
	}
	bindings, err := gssapi.TLSServerEndPoint(tls.ConnectionState{PeerCertificates: []*x509.Certificate{certificate}})
	if err != nil {
		return strings.Join(append(parts, "bindings: "+err.Error()), "; ")
	}
	parts = append(parts, fmt.Sprintf("expected application data=%q", bindings.ApplicationData))
	for _, protection := range []service.ExtendedProtectionPolicy{service.ExtendedProtectionRequired, service.ExtendedProtectionDisabled} {
		acceptor := spnego.SPNEGOService(kt,
			service.ServicePrincipals("HTTP/sso.example.com@EXAMPLE.COM"), service.MaxClockSkew(5*time.Minute),
			service.ExtendedProtection(protection), service.ChannelBindings(&bindings),
		)
		var fresh spnego.SPNEGOToken
		_ = fresh.Unmarshal(token)
		ok, _, status := acceptor.AcceptSecContext(&fresh)
		parts = append(parts, fmt.Sprintf("acceptor(protection=%v): ok=%v code=%v message=%q", protection, ok, status.Code, status.Message))
	}
	return strings.Join(parts, "; ")
}

// apiCaches は klist -l が示す資格情報のキャッシュの名前を返す。principal が空でなければ、その主体のものに限る。
func apiCaches(t *testing.T, principal string) []string {
	t.Helper()
	output, _ := exec.Command("klist", "-l").CombinedOutput()
	var caches []string
	for line := range strings.SplitSeq(string(output), "\n") {
		fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "*"))
		if len(fields) < 2 || fields[0] == "Name" || !strings.Contains(fields[1], ":") {
			continue
		}
		if principal == "" || fields[0] == principal {
			caches = append(caches, fields[1])
		}
	}
	return caches
}

func TestRecordChromeNegotiate(t *testing.T) {
	kdc, keytabFile, chrome := os.Getenv("KERBEROS_KDC_ADDR"), os.Getenv("KERBEROS_KEYTAB_FILE"), os.Getenv("CHROME_BINARY")
	if kdc == "" || keytabFile == "" || chrome == "" {
		t.Fatal("KERBEROS_KDC_ADDR, KERBEROS_KEYTAB_FILE and CHROME_BINARY are set by mise run record-negotiate")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	work, err := os.MkdirTemp(filepath.Join(home, ".cache"), "idmagic-record-negotiate-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(work) })

	krb5 := filepath.Join(work, "krb5.conf")
	if err := os.WriteFile(krb5, []byte(`[libdefaults]
  default_realm = EXAMPLE.COM
  dns_lookup_kdc = false
  dns_lookup_realm = false
  udp_preference_limit = 1
[realms]
  EXAMPLE.COM = {
    kdc = tcp/`+kdc+`
  }
[domain_realm]
  sso.example.com = EXAMPLE.COM
`), 0o600); err != nil {
		t.Fatal(err)
	}
	passwordFile := filepath.Join(work, "alice.pw")
	if err := os.WriteFile(passwordFile, []byte("Test-Alice-Password-123!"), 0o600); err != nil {
		t.Fatal(err)
	}
	// macOS の GSS.framework は資格情報を GSSCred の API のキャッシュから探し、FILE のキャッシュを見ない。
	// API のキャッシュの名前は GSSCred が割り当てるので、kinit の後に klist -l から読む。
	// 既定のキャッシュを切り替えないよう、ほかのキャッシュがあれば記録しない。
	env := append(os.Environ(), "KRB5_CONFIG="+krb5)
	if caches := apiCaches(t, ""); len(caches) != 0 {
		t.Fatalf("refusing to record: Kerberos credential caches already exist and kinit would change the default: %v", caches)
	}
	kinit := exec.Command("kinit", "--password-file="+passwordFile, "alice@EXAMPLE.COM")
	kinit.Env = env
	if output, err := kinit.CombinedOutput(); err != nil {
		t.Fatalf("kinit: %v\n%s", err, output)
	}
	caches := apiCaches(t, "alice@EXAMPLE.COM")
	t.Cleanup(func() {
		for _, cache := range caches {
			if output, err := exec.Command("kdestroy", "-c", cache).CombinedOutput(); err != nil {
				t.Errorf("kdestroy -c %s failed; remove it by hand: %v\n%s", cache, err, output)
			}
		}
		if left := apiCaches(t, "alice@EXAMPLE.COM"); len(left) != 0 {
			t.Errorf("test Kerberos caches are left behind; remove them by hand: %v", left)
		}
	})
	if len(caches) != 1 {
		t.Fatalf("kinit made %d caches for alice, want 1: %v", len(caches), caches)
	}
	// GSS.framework が KRB5_CONFIG の KDC を使うとは限らないので、サービスチケットも先にキャッシュへ入れる。
	kgetcred := exec.Command("kgetcred", "--cache="+caches[0], "HTTP/sso.example.com@EXAMPLE.COM")
	kgetcred.Env = env
	if output, err := kgetcred.CombinedOutput(); err != nil {
		t.Fatalf("kgetcred: %v\n%s", err, output)
	}

	// sso.example.com の名前の TLS の証明書。Verifier はこの証明書から tls-server-end-point の束縛を作る。
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: "sso.example.com"}, DNSNames: []string{"sso.example.com"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	certificateFile := filepath.Join(work, "tls.pem")
	if err := os.WriteFile(certificateFile, certificatePEM, 0o600); err != nil {
		t.Fatal(err)
	}

	var (
		mu    sync.Mutex
		token string
	)
	got := make(chan struct{}, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Negotiate ")
		if !ok {
			w.Header().Set("WWW-Authenticate", "Negotiate")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		mu.Lock()
		if token == "" {
			token = value
			got <- struct{}{}
		}
		mu.Unlock()
		_, _ = w.Write([]byte("recorded"))
	}))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server.Listener = listener
	server.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}, MinVersion: tls.VersionTLS12}
	server.StartTLS()
	t.Cleanup(server.Close)

	// Chrome の名前の対応付け（--host-resolver-rules）は SPN を対応付けた先のアドレスで作るので、
	// CONNECT のプロキシで sso.example.com をこの待ち受けへ向け、Chrome には名前のまま要求させる。
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect || r.Host != "sso.example.com:443" {
			http.Error(w, "only CONNECT to sso.example.com:443", http.StatusForbidden)
			return
		}
		upstream, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			_ = upstream.Close()
			return
		}
		client, _, err := hijacker.Hijack()
		if err != nil {
			_ = upstream.Close()
			return
		}
		_, _ = client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		go func() { _, _ = io.Copy(upstream, client); _ = upstream.Close() }()
		go func() { _, _ = io.Copy(client, upstream); _ = client.Close() }()
	}))
	t.Cleanup(proxy.Close)

	version, err := exec.Command(chrome, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	browser := exec.CommandContext(ctx, chrome,
		"--headless=new", "--no-first-run", "--user-data-dir="+filepath.Join(work, "profile"),
		"--proxy-server="+proxy.URL,
		"--ignore-certificate-errors",
		"--auth-server-allowlist=sso.example.com",
		"--disable-auth-negotiate-cname-lookup",
		"--disable-features=NetworkServiceSandbox",
		"--log-net-log="+filepath.Join(home, ".cache", "idmagic-negotiate-netlog.json"), "--net-log-capture-mode=Everything",
		"--dump-dom", "https://sso.example.com/")
	browser.Env = env
	output, browserErr := browser.CombinedOutput()
	select {
	case <-got:
	case <-time.After(5 * time.Second):
		t.Fatalf("Chrome sent no Negotiate token (err=%v)\n%s", browserErr, output)
	}

	mu.Lock()
	recorded := token
	mu.Unlock()
	raw, err := base64.StdEncoding.DecodeString(recorded)
	if err != nil {
		t.Fatal(err)
	}
	config := negotiatedomain.Config{Realm: "EXAMPLE.COM", ServicePrincipal: "HTTP/sso.example.com", KeytabFile: keytabFile, CertificateFile: certificateFile}
	verification := "accepted"
	verified, verifyErr := (protocol_kerberos.Verifier{}).Verify(context.Background(), config, raw, "")
	if verifyErr != nil {
		verification = verifyErr.Error()
	} else {
		verification += " as " + verified.Principal.Name + "@" + verified.Principal.Realm
	}
	t.Logf("Chrome token verification at recording: %s", verification)
	diagnosis := diagnose(t, raw, keytabFile, certificateFile)
	t.Logf("diagnosis: %s", diagnosis)

	rec := ChromeNegotiateRecording{
		RetrievedAt: time.Now().Format(time.DateOnly),
		Client:      strings.TrimSpace(string(version)) + "（macOS の GSS.framework）",
		KDC:         "infra/test/kerberos の Samba AD（realm EXAMPLE.COM、SPN HTTP/sso.example.com）",
		Method: "mise run record-negotiate が隔離した Samba AD を起動し、macOS の kinit で alice@EXAMPLE.COM のチケットを FILE の資格情報のキャッシュへ取った。" +
			"手元の Chrome を同じ KRB5_CONFIG と KRB5CCNAME で、--auth-server-allowlist=sso.example.com と --disable-auth-negotiate-cname-lookup を付けて headless で起動し、" +
			"sso.example.com を 127.0.0.1 へ向け、この道具が作った自己署名の証明書の TLS の待ち受けを開かせた（--ignore-certificate-errors）。" +
			"401 と WWW-Authenticate: Negotiate を返し、Chrome が付け直した Authorization: Negotiate の値を記録した。certificate_pem はその待ち受けの証明書である。" +
			"記録のときに IdMagic の Verifier を Samba AD が書き出した keytab で動かし、結果を verification_at_recording に残した。" +
			"チケットと authenticator の時刻は記録のときの値であり、後日の再生では時刻に依らない形だけを確かめる。" +
			"Windows の SSPI と、Edge、Firefox が送るトークンの形は保証しない。",
		CertificatePEM: string(certificatePEM), Token: recorded,
		VerifiedAtRecording: verifyErr == nil, VerificationAtRecording: verification, DiagnosisAtRecording: diagnosis,
	}
	encoded, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("testdata/chrome-negotiate.json", append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
