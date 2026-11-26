// Package testing_oidctwin は、上流の OpenID Connect の OP の代わりに Authentication の連携のテストが使う相手を提供する。
//
// 相手は二つある。OpenID Connect Core 1.0、Discovery 1.0、RFC 7517 の例を転記した記録をそのまま返す再生と、
// Discovery、認可コードの発行と交換、ID トークンの署名、JWK Set の配信を扱う状態を持つ模倣である。
// 記録は模倣の忠実さの基準でもあり、模倣が記録と同じ応答を返すことをこのパッケージのテストが確かめる。
// 記録の形式、再生、照合は testing_httprecord の部品を使う。
// どちらもテストの中だけで使い、製品の経路から import しない。
package testing_oidctwin

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

//go:embed testdata/oidc.json
var oidcRecording []byte

// 記録が使う OpenID Connect Core 1.0 の例の値である。再生の検査は、クライアントにこの値を渡して要求を記録と比べる。
const (
	// RecordedIssuer は Discovery §4.2 と Core 付録 A.2 の issuer である。
	RecordedIssuer = "https://server.example.com"
	// RecordedHTTPIssuer は Core §3.1.3.3 の ID トークンの iss である。§3.1.3.3 の例だけが http を使う。
	RecordedHTTPIssuer = "http://server.example.com"
	// RecordedClientID と RecordedClientSecret は Core §3.1.3.1 の Basic 認証が符号化する資格情報であり、
	// ID トークンの例の aud でもある。
	RecordedClientID     = "s6BhdRkqt3"
	RecordedClientSecret = "gX1fBat3bV"
	// RecordedCode と RecordedRedirectURI は Core §3.1.3.1 の認可コードとリダイレクト URI である。
	RecordedCode        = "SplxlOBeZQQYbYS6WxSbIA"
	RecordedRedirectURI = "https://client.example.org/cb"
	// RecordedSubject と RecordedNonce は Core の例の ID トークンの sub と nonce である。
	RecordedSubject = "248289761001"
	RecordedNonce   = "n-0S6_WzA2Mj"
)

// RecordedIssuedAt と RecordedExpiry は Core の例の ID トークンの iat と exp である。
// 例の署名を検証する再生では、検証の時刻をこの間に置く。
var (
	RecordedIssuedAt = time.Unix(1311280970, 0).UTC()
	RecordedExpiry   = time.Unix(1311281970, 0).UTC()
)

// Recording は一つの相手との往復の列と、その出どころである。
type Recording = testing_httprecord.Recording

// Exchange は一つの要求と、相手が返した応答である。
type Exchange = testing_httprecord.Exchange

// OIDCRecording は OpenID Connect Core 1.0、Discovery 1.0、RFC 7517 の例を転記した記録を返す。
func OIDCRecording(tb testing.TB) Recording {
	tb.Helper()
	rec, err := testing_httprecord.Decode(oidcRecording)
	if err != nil {
		tb.Fatalf("decode OIDC recording: %v", err)
	}
	return rec
}

// NewReplayClient は、記録した往復を順に一度ずつ返す再生サーバーを起動し、そこへ送る HTTP クライアントを返す。
//
// クライアントは要求の URL の方式とホストを再生サーバーへ置き換え、パスとクエリはそのまま送る。
// そのため、製品のクライアントへは記録の issuer とエンドポイント（https://server.example.com）をそのまま渡せる。
// 受け取った要求は、方式とパス、記録した要求が持つ Content-Type と Authorization、フォームの値で記録と照合し、
// 食い違えばテストを失敗させる。フォームは記録にない欄（PKCE の code_verifier など）を許す。
func NewReplayClient(tb testing.TB, exchanges ...Exchange) *http.Client {
	tb.Helper()
	server := testing_httprecord.Replay{
		Headers:   []string{"Accept", "Content-Type", "Authorization"},
		CheckBody: testing_httprecord.CompareForm,
	}.Serve(tb, exchanges...)
	return routedClient(server)
}

// routedClient は、どのホストへの要求も server へ送るクライアントを返す。リダイレクトは追わない。
func routedClient(server *httptest.Server) *http.Client {
	target, _ := url.Parse(server.URL)
	base := server.Client().Transport
	return &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			routed := r.Clone(r.Context())
			routed.URL.Scheme = target.Scheme
			routed.URL.Host = target.Host
			routed.Host = r.URL.Host
			return base.RoundTrip(routed)
		}),
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// setup は、記録の前から相手が持っていた認可コードである。
type setup struct {
	AuthorizationCodes []struct {
		Code        string `json:"code"`
		RedirectURI string `json:"redirect_uri"`
		Subject     string `json:"subject"`
		Nonce       string `json:"nonce"`
	} `json:"authorization_codes"`
}

// modeledMetadata は、模倣が Discovery の文書で返すメタデータである。
// 記録のほかのメタデータ（userinfo_endpoint、acr_values_supported など）は模倣が扱わない機能なので照合から除く。
var modeledMetadata = map[string]bool{
	"issuer": true, "authorization_endpoint": true, "token_endpoint": true, "jwks_uri": true,
	"token_endpoint_auth_methods_supported": true, "response_types_supported": true,
	"subject_types_supported": true, "id_token_signing_alg_values_supported": true,
}

// Diverge は記録の往復を順に模倣へ送り、模倣の応答が記録と食い違う点を返す。
//
// 往復の setup が挙げる認可コードは、送る前に模倣へ置く。
// 模倣が割り当てる値（トークン、自分の鍵で署名した ID トークン、鍵の kid と n）と、相手ごとに違ってよい値
// （Discovery のエンドポイントと機能の一覧）は型だけを比べる。issuer、jwks_uri、鍵の kty と e、
// token_type、expires_in、エラーの error、状態コード、Content-Type のメディア型、Cache-Control は値まで比べる。
// Discovery の文書は modeledMetadata の属性だけを比べる。
func Diverge(tw *Twin, rec Recording) []string {
	var setupErrs []string
	trimmed := rec
	trimmed.Exchanges = make([]Exchange, 0, len(rec.Exchanges))
	for _, ex := range rec.Exchanges {
		if ex.Request.Path == "/.well-known/openid-configuration" {
			ex.Response.Body = modeledDiscovery(ex.Response.Body)
		}
		trimmed.Exchanges = append(trimmed.Exchanges, ex)
	}
	diffs := testing_httprecord.Fidelity{
		VolatileKeys: map[string]bool{
			"access_token": true, "refresh_token": true, "id_token": true, "kid": true, "n": true,
			"authorization_endpoint": true, "token_endpoint": true,
			"token_endpoint_auth_methods_supported": true, "response_types_supported": true,
			"subject_types_supported": true, "id_token_signing_alg_values_supported": true,
		},
		Headers:      []string{"Content-Type"},
		ExactHeaders: []string{"Cache-Control"},
		Setup: func(ex Exchange) {
			if len(ex.Setup) == 0 {
				return
			}
			var s setup
			if err := json.Unmarshal(ex.Setup, &s); err != nil {
				setupErrs = append(setupErrs, fmt.Sprintf("%s: setup: %v", ex.Name, err))
				return
			}
			for _, code := range s.AuthorizationCodes {
				tw.AddAuthorizationCode(AuthorizationCode{
					Code: code.Code, RedirectURI: code.RedirectURI, Subject: code.Subject, Nonce: code.Nonce,
				})
			}
		},
	}.Diverge(tw, trimmed)
	return append(setupErrs, diffs...)
}

func modeledDiscovery(body json.RawMessage) json.RawMessage {
	var document map[string]any
	if err := json.Unmarshal(body, &document); err != nil {
		return body
	}
	for key := range document {
		if !modeledMetadata[key] {
			delete(document, key)
		}
	}
	trimmed, err := json.Marshal(document)
	if err != nil {
		return body
	}
	return trimmed
}
