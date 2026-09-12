package tokens_jose

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	logoutports "github.com/ambi/idmagic/backend/oauth2/logout/ports"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
)

// back-channel logout イベントを運び、nonce を運ばないことを固定する。
//
//spec:covers OIDC-BACKCHANNEL-LOGOUT-TOKEN: logout token が iss、sub、aud、iat、jti、sid と
func TestSignLogoutToken_REQ_OAUTH2_025(t *testing.T) {
	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	token, err := NewJWTSigner("https://unused.example", keyStore).SignLogoutToken(context.Background(), logoutports.LogoutTokenInput{Issuer: "https://idp.example/realms/default", Subject: "alice", Audience: "client-1", Sid: "session-1", JTI: "logout-1", IssuedAt: time.Unix(1_700_000_000, 0).UTC()})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("parts=%d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatal(err)
	}
	if claims["iss"] != "https://idp.example/realms/default" || claims["sub"] != "alice" || claims["aud"] != "client-1" || claims["sid"] != "session-1" || claims["jti"] != "logout-1" || claims["iat"] != float64(1_700_000_000) {
		t.Fatalf("claims=%+v", claims)
	}
	if _, exists := claims["nonce"]; exists {
		t.Fatal("logout token contains nonce")
	}
	events, ok := claims["events"].(map[string]any)
	if !ok || events["http://schemas.openid.net/event/backchannel-logout"] == nil {
		t.Fatalf("events=%+v", claims["events"])
	}
}
