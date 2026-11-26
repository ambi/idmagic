package protocol_oidc

import (
	"context"
	"strings"
	"testing"
	"time"

	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	"github.com/ambi/idmagic/backend/authentication/internal/federation/testing_oidctwin"
)

// recordedConnection は Core と Discovery の例の OP へ向けた接続である。
func recordedConnection(issuer string) federationdomain.IdentityProviderConnection {
	connection := testConnection()
	connection.Issuer = issuer
	connection.ClientID = testing_oidctwin.RecordedClientID
	connection.SecretReference = testing_oidctwin.RecordedClientSecret
	connection.AuthorizationEndpoint = testing_oidctwin.RecordedIssuer + "/connect/authorize"
	connection.TokenEndpoint = testing_oidctwin.RecordedIssuer + "/token"
	connection.JWKSURI = testing_oidctwin.RecordedIssuer + "/jwks.json"
	return connection
}

var recordedAttempt = federationdomain.FederatedLoginAttempt{
	State: "af0ifjsldkj", Nonce: testing_oidctwin.RecordedNonce, PKCEVerifier: strings.Repeat("v", 43),
}

// 上流の OP の実際の応答の形に対して、Discovery と認可コードの交換が取り出す結果を確かめる。
//
// 応答は OpenID Connect Core 1.0、Discovery 1.0、RFC 7517 の例を転記した記録から返す。
// 期待値は製品の定数からではなく、記録が示す値と仕様の文面から書く。
// ID トークンの例は Core 付録 A.7 の鍵で署名を検証できるので（Core §3.1.3.3）、検証の時刻を例の iat と exp の間に置いて
// 実際に署名を検証する。要求の方式、パス、Content-Type、Authorization、フォームの値が記録と食い違うと、
// 再生がテストを失敗させる。
//
//spec:covers OIDC-CORE-CODE-FLOW, OIDC-CORE-CLIENT-SECRET-BASIC, OIDC-DISCOVERY-ISSUER, REQ-AUTHENTICATION-001, REQ-AUTHENTICATION-041: Core §3.1.3.1 の Basic 認証と本文の交換の要求、§3.1.3.3 と付録 A.2 の ID トークンの署名、issuer、audience、時刻、nonce の検証、Discovery §4.2 の文書と JWK Set の受け入れ
func TestUpstreamReplay_OIDCRecordings(t *testing.T) {
	rec := testing_oidctwin.OIDCRecording(t)
	ctx := context.Background()
	during := testing_oidctwin.RecordedIssuedAt.Add(time.Minute)

	t.Run("Discovery §4.2 の文書と付録 A.7 の JWK Set", func(t *testing.T) {
		client := Client{HTTPClient: testing_oidctwin.NewReplayClient(t, rec.Exchange(t, "discovery"), rec.Exchange(t, "jwks"))}
		preview, err := client.Discover(ctx, testing_oidctwin.RecordedIssuer)
		if err != nil {
			t.Fatalf("Discover() error = %v", err)
		}
		want := federationdomain.MetadataPreview{
			Issuer:                "https://server.example.com",
			AuthorizationEndpoint: "https://server.example.com/connect/authorize",
			TokenEndpoint:         "https://server.example.com/connect/token",
			JWKSURI:               "https://server.example.com/jwks.json",
		}
		if *preview != want {
			t.Errorf("Discover() = %+v, want %+v", *preview, want)
		}
	})
	t.Run("RFC 7517 付録 A.1 の暗号化用の EC 鍵を読み飛ばす", func(t *testing.T) {
		client := Client{HTTPClient: testing_oidctwin.NewReplayClient(t, rec.Exchange(t, "discovery"), rec.Exchange(t, "jwks-encryption-and-signing-keys"))}
		if _, err := client.Discover(ctx, testing_oidctwin.RecordedIssuer); err != nil {
			t.Errorf("Discover() error = %v, want the RS256 key to be accepted beside the EC encryption key", err)
		}
	})
	t.Run("Core §3.1.3.3 の応答の ID トークンを付録 A.7 の鍵で検証する", func(t *testing.T) {
		client := Client{HTTPClient: testing_oidctwin.NewReplayClient(t, rec.Exchange(t, "token-authorization-code"), rec.Exchange(t, "jwks"))}
		connection := recordedConnection(testing_oidctwin.RecordedHTTPIssuer)
		connection.ClaimMapping = federationdomain.ClaimMapping{Subject: "sub", Username: "sub"}
		claims, err := client.ExchangeAndValidate(ctx, connection, recordedAttempt, testing_oidctwin.RecordedCode, testing_oidctwin.RecordedRedirectURI, during)
		if err != nil {
			t.Fatalf("ExchangeAndValidate() error = %v", err)
		}
		if claims.Subject != "248289761001" {
			t.Errorf("Subject = %q, want the recorded sub 248289761001", claims.Subject)
		}
	})
	t.Run("付録 A.2 の ID トークンのプロファイルのクレームを対応付ける", func(t *testing.T) {
		client := Client{HTTPClient: testing_oidctwin.NewReplayClient(t, rec.Exchange(t, "token-authorization-code-profile-claims"), rec.Exchange(t, "jwks"))}
		claims, err := client.ExchangeAndValidate(ctx, recordedConnection(testing_oidctwin.RecordedIssuer), recordedAttempt, testing_oidctwin.RecordedCode, testing_oidctwin.RecordedRedirectURI, during)
		if err != nil {
			t.Fatalf("ExchangeAndValidate() error = %v", err)
		}
		want := federationdomain.NormalizedClaims{
			Subject: "248289761001", Username: "janedoe@example.com", Email: "janedoe@example.com", Name: "Jane Doe",
		}
		// A.2 は email_verified を持たないので、検証済みとは扱わない。
		if claims != want {
			t.Errorf("claims = %+v, want %+v", claims, want)
		}
	})
	t.Run("例の exp を過ぎた ID トークンを拒否する", func(t *testing.T) {
		client := Client{HTTPClient: testing_oidctwin.NewReplayClient(t, rec.Exchange(t, "token-authorization-code-profile-claims"), rec.Exchange(t, "jwks"))}
		if _, err := client.ExchangeAndValidate(ctx, recordedConnection(testing_oidctwin.RecordedIssuer), recordedAttempt, testing_oidctwin.RecordedCode, testing_oidctwin.RecordedRedirectURI, testing_oidctwin.RecordedExpiry); err == nil {
			t.Error("ExchangeAndValidate() accepted an ID token at its exp")
		}
	})
	t.Run("Core §3.1.3.4 のエラー応答で交換を失敗にする", func(t *testing.T) {
		client := Client{HTTPClient: testing_oidctwin.NewReplayClient(t, rec.Exchange(t, "token-invalid-request"))}
		if _, err := client.ExchangeAndValidate(ctx, recordedConnection(testing_oidctwin.RecordedIssuer), recordedAttempt, "unknown", testing_oidctwin.RecordedRedirectURI, during); err == nil {
			t.Error("ExchangeAndValidate() succeeded on a 400 invalid_request")
		}
	})
}
