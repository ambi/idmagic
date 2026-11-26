package protocol_oidc

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	"github.com/ambi/idmagic/backend/authentication/internal/federation/testing_oidctwin"
)

const twinRedirectURI = "https://broker.example/callback"

// twinConnection は模倣の OP へ向けた有効な接続である。
func twinConnection(clientID, secret string) federationdomain.IdentityProviderConnection {
	connection := testConnection()
	connection.Issuer = testing_oidctwin.RecordedIssuer
	connection.ClientID = clientID
	connection.SecretReference = secret
	connection.AuthorizationEndpoint = testing_oidctwin.RecordedIssuer + "/authorize"
	connection.TokenEndpoint = testing_oidctwin.RecordedIssuer + "/token"
	connection.JWKSURI = testing_oidctwin.RecordedIssuer + "/jwks.json"
	return connection
}

// signIn は AuthorizationURL が組み立てた要求を模倣の認可エンドポイントへ送り、戻された認可コードを返す。
func signIn(t *testing.T, client *http.Client, connection federationdomain.IdentityProviderConnection, attempt federationdomain.FederatedLoginAttempt) string {
	t.Helper()
	target, err := AuthorizationURL(connection, attempt, twinRedirectURI)
	if err != nil {
		t.Fatalf("AuthorizationURL() error = %v", err)
	}
	resp, err := client.Get(target)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	location, err := url.Parse(resp.Header.Get("Location"))
	if resp.StatusCode != http.StatusFound || err != nil || location.Query().Get("state") != attempt.State {
		t.Fatalf("authorize = %d %q, want 302 with the attempt's state", resp.StatusCode, resp.Header.Get("Location"))
	}
	return location.Query().Get("code")
}

// testNow は、模倣が現在時刻で署名した ID トークンを検証する時刻である。零値なら製品のクライアントが現在時刻を使う。
func testNow() time.Time { return time.Time{} }

func twinAttempt() federationdomain.FederatedLoginAttempt {
	return federationdomain.FederatedLoginAttempt{State: "state-1", Nonce: "nonce-1", PKCEVerifier: strings.Repeat("p", 43)}
}

// 状態を持つ OP を相手に、認可の要求から ID トークンの検証までを通す。
// 模倣は PKCE の S256、一度だけの認可コード、client_secret_basic を確かめ、自分の鍵で署名する。
//
//spec:covers OIDC-CORE-CODE-FLOW, OIDC-CORE-CLIENT-SECRET-BASIC, REQ-AUTHENTICATION-001: 認可コードと PKCE の交換、Basic だけを受け付ける OP への認証、交換し直した使用済みのコードの拒否
func TestUpstreamTwin_CodeFlow(t *testing.T) {
	ctx := context.Background()
	tw := testing_oidctwin.NewTwin(t, testing_oidctwin.RecordedIssuer, "client", "secret")
	tw.AuthMethods = []string{testing_oidctwin.ClientSecretBasic}
	tw.SignedInSubject = "twin-user"
	tw.SignedInClaims = map[string]any{"email": "user@example.com", "email_verified": true, "name": "User"}
	client := Client{HTTPClient: tw.Client(t)}
	connection := twinConnection("client", "secret")
	attempt := twinAttempt()

	code := signIn(t, client.HTTPClient, connection, attempt)
	claims, err := client.ExchangeAndValidate(ctx, connection, attempt, code, twinRedirectURI, testNow())
	if err != nil {
		t.Fatalf("ExchangeAndValidate() error = %v", err)
	}
	want := federationdomain.NormalizedClaims{Subject: "twin-user", Username: "user@example.com", Email: "user@example.com", EmailVerified: true, Name: "User"}
	if claims != want {
		t.Errorf("claims = %+v, want %+v", claims, want)
	}
	if _, err := client.ExchangeAndValidate(ctx, connection, attempt, code, twinRedirectURI, testNow()); err == nil {
		t.Error("ExchangeAndValidate() accepted a code that was already exchanged")
	}
}

// Basic の資格情報は、client_id とシークレットを form-urlencoded で符号化してから作る（RFC 6749 §2.3.1、付録 B）。
// シークレットのない接続は、Basic を使わず client_id を本文に入れる公開クライアントとして交換する。
//
//spec:covers OIDC-CORE-CLIENT-SECRET-BASIC: 予約文字と非 ASCII を含む資格情報の符号化と、シークレットのない接続の交換
func TestUpstreamTwin_ClientAuthentication(t *testing.T) {
	ctx := context.Background()
	t.Run("予約文字と非 ASCII を含む資格情報", func(t *testing.T) {
		const clientID, secret = "client:1 é", "s3:cr t+%&é"
		tw := testing_oidctwin.NewTwin(t, testing_oidctwin.RecordedIssuer, clientID, secret)
		tw.AuthMethods = []string{testing_oidctwin.ClientSecretBasic}
		client := Client{HTTPClient: tw.Client(t)}
		connection := twinConnection(clientID, secret)
		connection.ClaimMapping = federationdomain.ClaimMapping{Subject: "sub", Username: "sub"}
		code := signIn(t, client.HTTPClient, connection, twinAttempt())
		if _, err := client.ExchangeAndValidate(ctx, connection, twinAttempt(), code, twinRedirectURI, testNow()); err != nil {
			t.Errorf("ExchangeAndValidate() error = %v", err)
		}
	})
	t.Run("シークレットのない接続", func(t *testing.T) {
		var sawAuthorization, sawClientID bool
		tw := testing_oidctwin.NewTwin(t, testing_oidctwin.RecordedIssuer, "public", "")
		twinClient := tw.Client(t)
		base := twinClient.Transport
		twinClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/token" {
				sawAuthorization = r.Header.Get("Authorization") != ""
				_ = r.ParseForm()
				sawClientID = r.PostForm.Get("client_id") == "public" && !r.PostForm.Has("client_secret")
			}
			return base.RoundTrip(r)
		})
		client := Client{HTTPClient: twinClient}
		connection := twinConnection("public", "")
		code := signIn(t, client.HTTPClient, connection, twinAttempt())
		// 模倣は認証のない要求を invalid_client で拒否するので、交換の成否ではなく送った要求を観測する。
		_, _ = client.ExchangeAndValidate(ctx, connection, twinAttempt(), code, twinRedirectURI, testNow())
		if sawAuthorization || !sawClientID {
			t.Errorf("token request Authorization present = %v, client_id only in body = %v; want no Basic and client_id in the body", sawAuthorization, sawClientID)
		}
	})
}

// 鍵の回転、kid の欠落、回転に遅れた JWK Set に対する ID トークンの鍵の選び方を確かめる。
//
//spec:covers OIDC-CORE-ID-TOKEN-KID-OPTIONAL, REQ-AUTHENTICATION-001: 回転後の新しい kid の受け入れ、kid のない鍵が一つなら受け入れ二つなら拒否、JWK Set にない kid の拒否
func TestUpstreamTwin_SigningKeys(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		setup  func(t *testing.T, tw *testing_oidctwin.Twin)
		accept bool
	}{
		{"回転した鍵と旧い鍵を公開する", func(t *testing.T, tw *testing_oidctwin.Twin) {
			t.Helper()
			tw.RotateSigningKey(t)
		}, true},
		{"kid のない鍵が一つ", func(_ *testing.T, tw *testing_oidctwin.Twin) { tw.OmitKid = true }, true},
		{"kid のない鍵が二つ", func(t *testing.T, tw *testing_oidctwin.Twin) {
			t.Helper()
			tw.OmitKid = true
			tw.RotateSigningKey(t)
		}, false},
		{"JWK Set の配信が回転に遅れる", func(t *testing.T, tw *testing_oidctwin.Twin) {
			t.Helper()
			tw.RotateSigningKey(t)
			tw.StaleJWKS = true
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tw := testing_oidctwin.NewTwin(t, testing_oidctwin.RecordedIssuer, "client", "secret")
			tc.setup(t, tw)
			client := Client{HTTPClient: tw.Client(t)}
			connection := twinConnection("client", "secret")
			connection.ClaimMapping = federationdomain.ClaimMapping{Subject: "sub", Username: "sub"}
			code := signIn(t, client.HTTPClient, connection, twinAttempt())
			_, err := client.ExchangeAndValidate(ctx, connection, twinAttempt(), code, twinRedirectURI, testNow())
			if (err == nil) != tc.accept {
				t.Errorf("ExchangeAndValidate() error = %v, want accepted = %v", err, tc.accept)
			}
		})
	}
}

// 経路ごとの 5xx と不正な本文で、Discovery と交換がどちらも失敗を返し、部分的な結果を返さない。
//
//spec:covers REQ-AUTHENTICATION-001, REQ-AUTHENTICATION-041: Discovery、JWK Set、トークンの取得の失敗を候補の取得とコールバックの失敗にする
func TestUpstreamTwin_Faults(t *testing.T) {
	ctx := context.Background()
	unavailable := testing_oidctwin.Fault{Status: http.StatusServiceUnavailable, ContentType: "text/plain", Body: "down"}
	for _, tc := range []struct {
		path  string
		fault testing_oidctwin.Fault
	}{
		{"/.well-known/openid-configuration", unavailable},
		{"/.well-known/openid-configuration", testing_oidctwin.MalformedJSON},
		{"/jwks.json", unavailable},
		{"/jwks.json", testing_oidctwin.MalformedJSON},
	} {
		t.Run("Discover "+tc.path+" "+http.StatusText(tc.fault.Status), func(t *testing.T) {
			tw := testing_oidctwin.NewTwin(t, testing_oidctwin.RecordedIssuer, "client", "secret")
			tw.InjectFaults(tc.path, tc.fault)
			client := Client{HTTPClient: tw.Client(t)}
			if preview, err := client.Discover(ctx, testing_oidctwin.RecordedIssuer); err == nil {
				t.Errorf("Discover() = %+v, want an error", preview)
			}
		})
	}
	for _, tc := range []struct {
		path  string
		fault testing_oidctwin.Fault
	}{
		{"/token", unavailable},
		{"/token", testing_oidctwin.MalformedJSON},
		{"/jwks.json", unavailable},
		{"/jwks.json", testing_oidctwin.MalformedJSON},
	} {
		t.Run("ExchangeAndValidate "+tc.path+" "+http.StatusText(tc.fault.Status), func(t *testing.T) {
			tw := testing_oidctwin.NewTwin(t, testing_oidctwin.RecordedIssuer, "client", "secret")
			client := Client{HTTPClient: tw.Client(t)}
			connection := twinConnection("client", "secret")
			code := signIn(t, client.HTTPClient, connection, twinAttempt())
			tw.InjectFaults(tc.path, tc.fault)
			if claims, err := client.ExchangeAndValidate(ctx, connection, twinAttempt(), code, twinRedirectURI, testNow()); err == nil {
				t.Errorf("ExchangeAndValidate() = %+v, want an error", claims)
			}
		})
	}
}
