package tokens_jose

import (
	"context"
	"strings"
	"testing"

	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

// 決定4: id_token_hint は本 OP が署名した ID Token のみを受理し、署名・iss を
// fail-closed で検証する。exp は検証しない。

func TestVerifyIDTokenHintReturnsClaimsForValidToken(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	token, err := signer.SignIDToken(context.Background(), ports.IDTokenInput{
		Client: &domain.OAuth2Client{ClientID: "web-app"}, User: idTokenTestUser(),
		Scopes: []string{"openid"}, Sid: "session-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := signer.VerifyIDTokenHint(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Sid != "session-1" {
		t.Fatalf("sid=%q, want session-1", claims.Sid)
	}
	if claims.Audience != "web-app" {
		t.Fatalf("audience=%q, want web-app", claims.Audience)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("subject=%q, want user-1", claims.Subject)
	}
}

func TestVerifyIDTokenHintRejectsTamperedSignature(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	token, err := signer.SignIDToken(context.Background(), ports.IDTokenInput{
		Client: &domain.OAuth2Client{ClientID: "web-app"}, User: idTokenTestUser(),
		Scopes: []string{"openid"}, Sid: "session-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("malformed jwt: %q", token)
	}
	tampered := parts[0] + "." + parts[1] + "X." + parts[2]
	if _, err := signer.VerifyIDTokenHint(context.Background(), tampered); err == nil {
		t.Fatal("expected signature verification failure")
	}
}

func TestVerifyIDTokenHintRejectsOtherIssuer(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	otherKS, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	otherIssuerSigner := NewJWTSigner("https://other-idp.test", otherKS)
	token, err := otherIssuerSigner.SignIDToken(context.Background(), ports.IDTokenInput{
		Client: &domain.OAuth2Client{ClientID: "web-app"}, User: idTokenTestUser(),
		Scopes: []string{"openid"}, Sid: "session-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	if _, err := signer.VerifyIDTokenHint(context.Background(), token); err == nil {
		t.Fatal("expected issuer mismatch to be rejected")
	}
}

// signHintClaims は署名と iss だけが正しい ID Token を作る。claim の欠落を検査に
// かけるため、SignIDToken を経由せず claim 集合を直接指定する。
func signHintClaims(t *testing.T, signer *JWTSigner, ks *signingcrypto.InMemoryKeyStore, claims map[string]any) string {
	t.Helper()
	key, err := ks.GetActiveKey(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	claims["iss"] = signer.Issuer
	token, err := SignPS256(key, map[string]string{"typ": "JWT"}, claims)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// 署名も iss も正しいが sub または aud を欠く token は、ログアウトと CIBA の
// どちらにとっても主体を決められない。claim を運ぶ前に fail-closed で拒否する。
func TestVerifyIDTokenHintRejectsMissingSubjectOrAudience(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)

	for name, claims := range map[string]map[string]any{
		"sub 無し":   {"aud": "web-app", "sid": "session-1"},
		"aud 無し":   {"sub": "user-1", "sid": "session-1"},
		"sub が空文字": {"sub": "", "aud": "web-app", "sid": "session-1"},
		"aud が空文字": {"sub": "user-1", "aud": "", "sid": "session-1"},
		"aud が空配列": {"sub": "user-1", "aud": []any{}, "sid": "session-1"},
	} {
		token := signHintClaims(t, signer, ks, claims)
		if _, err := signer.VerifyIDTokenHint(context.Background(), token); err == nil {
			t.Fatalf("%s: 必須 claim を欠く id_token_hint が受理された", name)
		}
	}
}
