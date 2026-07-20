package tokens_jose

import (
	"context"
	"strings"
	"testing"

	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

// ADR-127 決定4: id_token_hint は本 OP が署名した ID Token のみを受理し、署名・iss を
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
