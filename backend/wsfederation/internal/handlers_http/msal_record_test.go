//go:build msalrecord

package handlers_http_test

// mise run record-msal が使う記録の道具である。通常のテストには入らない。
//
// WS-Trust の能動的な STS を idp.example の名前の証明書で待ち受け、docker の .NET SDK で本物の MSAL.NET を動かし、
// IdMagic が受け取った MEX の取得と RST を testdata/msal-wstrust.json に記録する。
// 記録に残る資格情報は、受け入れテストのスタックの利用者 alice とそのパスワード（correct-password）だけである。

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"maps"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// idpExampleCertificate は idp.example の名前を持つ自己署名の証明書である。コンテナはこれを信頼の根に足す。
func idpExampleCertificate(t *testing.T) (tls.Certificate, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: "idp.example"},
		DNSNames: []string{"idp.example"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true, IsCA: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// capture は IdMagic が受け取った要求と、IdMagic の応答を順に覚える。
type capture struct {
	mu        sync.Mutex
	exchanges []testing_httprecord.Exchange
}

func (c *capture) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		recorder := httptest.NewRecorder()
		next.ServeHTTP(recorder, r)
		maps.Copy(w.Header(), recorder.Header())
		w.WriteHeader(recorder.Code)
		_, _ = w.Write(recorder.Body.Bytes())

		headers := map[string]string{}
		for _, name := range []string{"Content-Type", "SOAPAction", "Accept", "User-Agent", "client-request-id", "x-client-SKU", "x-client-Ver", "x-client-OS"} {
			if value := r.Header.Get(name); value != "" {
				headers[name] = value
			}
		}
		var requestBody json.RawMessage
		if len(body) > 0 {
			requestBody, _ = json.Marshal(string(body))
		}
		responseBody, _ := json.Marshal(recorder.Body.String())
		name := "mex"
		if r.Method == http.MethodPost {
			name = "rst"
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		c.exchanges = append(c.exchanges, testing_httprecord.Exchange{
			Name:     name,
			Source:   "MSAL.NET が " + r.Method + " " + r.URL.Path + " へ送った要求（" + strconv.Itoa(len(c.exchanges)+1) + " 番目）と IdMagic の応答",
			Request:  testing_httprecord.RecordedRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Headers: headers, Body: requestBody},
			Response: testing_httprecord.RecordedResponse{Status: recorder.Code, Headers: map[string]string{"Content-Type": recorder.Header().Get("Content-Type")}, Body: responseBody},
		})
	})
}

func TestRecordMSALWsTrust(t *testing.T) {
	image := os.Getenv("MSAL_RECORD_IMAGE")
	if image == "" {
		t.Fatal("MSAL_RECORD_IMAGE is set by mise run record-msal")
	}
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	issuer := "https://idp.example:" + port
	// MEX の各エンドポイントの住所は発行者から作るので、MSAL が同じポートへ RST を送るよう発行者にポートを含める。
	e, _, _ := newServerWithSigner(t, nil, func(d *httpadapter.Deps) { d.Issuer = issuer })

	certificate, certificatePEM := idpExampleCertificate(t)
	recorded := &capture{}
	server := httptest.NewUnstartedServer(recorded.wrap(e))
	server.Listener = listener
	server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12}
	server.StartTLS()
	t.Cleanup(server.Close)

	// colima は既定でホームディレクトリだけをコンテナへ見せるので、t.TempDir（/var/folders）ではなくホームの下に置く。
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	work, err := os.MkdirTemp(filepath.Join(home, ".cache"), "idmagic-record-msal-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(work) })
	source, out := filepath.Join(work, "src"), filepath.Join(work, "out")
	for _, dir := range []string{source, out} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"Program.cs", "msal.csproj"} {
		data, err := os.ReadFile(filepath.Join("testdata", "record", "msal", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(source, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(work, "idp-example.crt"), certificatePEM, 0o644); err != nil {
		t.Fatal(err)
	}

	// MSAL の UserRealmDiscoveryResponse の形で、連携した WS-Trust のアカウントを答える。
	// cloud_audience_urn は MSAL が RST の AppliesTo に置く値なので、受け入れテストのスタックの RP の realm にする。
	userRealm, err := json.Marshal(map[string]string{
		"ver": "1.0", "account_type": "Federated", "domain_name": "idp.example",
		"federation_protocol": "WSTrust", "cloud_instance_name": "microsoftonline.com", "cloud_audience_urn": passiveRealm,
		"federation_metadata_url":    issuer + "/realms/default/trust/mex",
		"federation_active_auth_url": issuer + "/realms/default/trust/usernamemixed",
	})
	if err != nil {
		t.Fatal(err)
	}

	run := exec.Command("docker", "run", "--rm",
		"--add-host", "idp.example:host-gateway",
		"--volume", source+":/src", "--volume", out+":/out",
		"--volume", filepath.Join(work, "idp-example.crt")+":/usr/local/share/ca-certificates/idp-example.crt:ro",
		"--workdir", "/src",
		"--env", "USERREALM_JSON="+string(userRealm),
		"--env", "CLIENT_ID=00000000-0000-4000-8000-0000000000aa",
		"--env", "USERNAME=alice", "--env", "PASSWORD=correct-password",
		"--env", "DOTNET_CLI_TELEMETRY_OPTOUT=1", "--env", "DOTNET_NOLOGO=1",
		image, "sh", "-c", "update-ca-certificates >/dev/null && dotnet run --configuration Release")
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("dotnet run: %v\n%s", err, output)
	}
	t.Logf("%s", output)
	outcome, err := os.ReadFile(filepath.Join(out, "outcome.txt"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitN(strings.TrimSpace(string(outcome)), "\n", 3)
	if len(lines) != 3 {
		t.Fatalf("outcome.txt = %q", outcome)
	}
	msalVersion, runtime, result := lines[0], lines[1], lines[2]

	recorded.mu.Lock()
	exchanges := recorded.exchanges
	recorded.mu.Unlock()
	if len(exchanges) == 0 {
		t.Fatalf("MSAL sent no request to IdMagic: %s", result)
	}
	rec := testing_httprecord.Recording{
		Counterpart:        "MSAL.NET の AcquireTokenByUsernamePassword（連携した WS-Trust のアカウント）",
		CounterpartVersion: "Microsoft.Identity.Client " + msalVersion + "（.NET " + runtime + "）",
		RetrievedAt:        time.Now().Format(time.DateOnly),
		Method: "mise run record-msal が、wsfederation の受け入れテストのスタック（発行者 " + issuer + "、利用者 alice、RP " + passiveRealm +
			"）を idp.example の名前のテストの証明書で待ち受け、" + image + " の中で testdata/record/msal の MSAL.NET を dotnet run で実行した。" +
			"login.microsoftonline.com の userrealm だけを WithHttpClientFactory の手元の応答で連携の WS-Trust のアカウントとして答えた。" +
			"MSAL は MEX の取得と RST を WithHttpClientFactory を通さずに送るので、コンテナの idp.example を IdMagic のホストへ向けた。" +
			"IdMagic が受け取った要求と応答をサーバーの側でそのまま記録した。ヘッダーは結果を左右しうるものだけを残した。" +
			"RSTR の後に MSAL が Entra ID へ送る要求は記録の対象ではないので、誤りを返した（MSAL の結果: " + result + "）。" +
			"RST の時刻、MessageID、ポートは記録のときの値のままである。" +
			"Windows の統合認証（windowstransport）の経路と、実際の ADFS の MEX での選択は保証しない。",
		Exchanges: exchanges,
	}
	encoded, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("testdata/msal-wstrust.json", append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
