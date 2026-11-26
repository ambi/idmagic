// Package testing_oauthtwin は、上流の OAuth 2.0 のトークンエンドポイントの代わりにテストが使う相手を提供する。
//
// 相手は二つある。RFC 6749 と RFC 6750 の例を転記した記録をそのまま返す再生と、
// 認可コード、更新トークン、クライアント資格情報を扱う状態を持つ模倣である。
// 記録は模倣の忠実さの基準でもあり、模倣が記録と同じ応答を返すことをこのパッケージのテストが確かめる。
// 記録の形式、再生、照合は testing_httprecord の部品を使う。
// どちらもテストの中だけで使い、製品の経路から import しない。
package testing_oauthtwin

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

//go:embed testdata/rfc6749.json
var rfc6749Recording []byte

// 記録の要求が使う RFC 6749 の例の値である。再生の検査は、クライアントにこの値を渡して要求を記録と比べる。
const (
	// RecordedClientID と RecordedClientSecret は §4.1.3 などの例の Basic 認証が符号化する資格情報である。
	RecordedClientID     = "s6BhdRkqt3"
	RecordedClientSecret = "gX1fBat3bV"
	// RecordedCode と RecordedRedirectURI は §4.1.3 の認可コードとリダイレクト URI である。
	RecordedCode        = "SplxlOBeZQQYbYS6WxSbIA"
	RecordedRedirectURI = "https://client.example.com/cb"
	// RecordedRefreshToken は §6 の更新トークンである。
	RecordedRefreshToken = "tGzv3JOkF0XG5Qx2TlKWIA"
)

// Recording は一つの相手との往復の列と、その出どころである。
type Recording = testing_httprecord.Recording

// Exchange は一つの要求と、相手が返した応答である。
type Exchange = testing_httprecord.Exchange

// RFC6749Recording は RFC 6749 と RFC 6750 の例を転記した記録を返す。
//
// RFC 6749 の応答の例は `token_type` がどれも `example` である。Bearer の応答は RFC 6750 §4 の例を使い、
// 往復の名前の末尾で `-example-type` と `-bearer` を区別する。
func RFC6749Recording(tb testing.TB) Recording {
	tb.Helper()
	rec, err := testing_httprecord.Decode(rfc6749Recording)
	if err != nil {
		tb.Fatalf("decode RFC 6749 recording: %v", err)
	}
	return rec
}

// setup は、記録の前から相手が持っていた認可コードと更新トークンである。
type setup struct {
	AuthorizationCodes []struct {
		Code        string `json:"code"`
		RedirectURI string `json:"redirect_uri"`
	} `json:"authorization_codes"`
	RefreshTokens []string `json:"refresh_tokens"`
}

// NewReplayServer は、記録した往復を順に一度ずつ返すサーバーを起動する。
//
// 受け取った要求は、方式とパス、記録した要求が持つ Content-Type と Authorization、
// フォームの値で記録と照合し、食い違えばテストを失敗させる。フォームは記録が持つ欄の値がすべて一致することを求め、
// 記録にない欄（PKCE の code_verifier など）は許す。Authorization まで比べるので、
// クライアントには RecordedClientID と RecordedClientSecret を渡す。
func NewReplayServer(tb testing.TB, exchanges ...Exchange) *httptest.Server {
	tb.Helper()
	return testing_httprecord.Replay{
		Headers:   []string{"Accept", "Content-Type", "Authorization"},
		CheckBody: testing_httprecord.CompareForm,
	}.Serve(tb, exchanges...)
}

// Diverge は記録の往復を順に模倣へ送り、模倣の応答が記録と食い違う点を返す。
//
// 往復の setup が挙げる認可コードと更新トークンは、送る前に模倣へ置く。
// アクセストークンと更新トークンの値は模倣が割り当てるので型だけを比べ、それ以外の本文の値、
// 状態コード、Content-Type のメディア型、Cache-Control と Pragma の値を比べる。
func Diverge(tw *Twin, rec Recording) []string {
	var setupErrs []string
	diffs := testing_httprecord.Fidelity{
		VolatileKeys: map[string]bool{"access_token": true, "refresh_token": true},
		Headers:      []string{"Content-Type"},
		ExactHeaders: []string{"Cache-Control", "Pragma"},
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
				tw.AddAuthorizationCode(code.Code, code.RedirectURI)
			}
			for _, token := range s.RefreshTokens {
				tw.AddRefreshToken(token)
			}
		},
	}.Diverge(tw, rec)
	return append(setupErrs, diffs...)
}
