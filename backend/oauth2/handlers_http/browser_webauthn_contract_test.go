package handlers_http_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	webauthnmemory "github.com/ambi/idmagic/backend/authentication/webauthn/db_memory"
	webauthndomain "github.com/ambi/idmagic/backend/authentication/webauthn/domain"
	webauthnusecases "github.com/ambi/idmagic/backend/authentication/webauthn/usecases"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
	"github.com/fxamacker/cbor/v2"
)

//verify:operation StartBrowserWebAuthn: challenge と署名済み assertion の 200 JSON 応答、session の MFA と sign count、および CSRF の拒否
//verify:operation SubmitBrowserWebAuthn: challenge と署名済み assertion の 200 JSON 応答、session の MFA と sign count、および CSRF の拒否
func TestBrowserWebAuthnHTTPContractsVerifyAssertionAndCompletePendingSession(t *testing.T) {
	ctx := testing_tenant.Default(t.Context())
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	point, err := key.PublicKey.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: point[1:33], -3: point[33:65]})
	if err != nil {
		t.Fatal(err)
	}
	id := base64.RawURLEncoding.EncodeToString([]byte("browser-contract-credential"))
	credentials := webauthnmemory.NewWebAuthnCredentialRepository()
	if err := credentials.Save(ctx, &webauthndomain.WebAuthnCredential{
		CredentialID: id, UserID: "user_alice", PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
		SignCount: 7, CreatedAt: time.Now().UTC(), Transports: []string{"internal"},
	}); err != nil {
		t.Fatal(err)
	}
	rp, err := webauthnusecases.NewWebAuthn(webauthnusecases.WebAuthnConfig{RPID: "localhost", RPDisplayName: "Contract", RPOrigins: []string{"http://localhost"}})
	if err != nil {
		t.Fatal(err)
	}
	sessions := sessionmemory.NewSessionStore()
	manager := sessionusecases.NewSessionManager(sessions)
	e, _ := newAuthorizeTestServer(t, &authdomain.AuthenticationContext{UserID: "user_alice"}, nil, func(d *httpadapter.Deps) {
		d.AuthnResolver = nil
		d.SessionManager = manager
		d.Authentication.WebAuthnRP = rp
		d.Authentication.WebAuthnCredentialRepo = credentials
		d.Authentication.WebAuthnSessionStore = webauthnmemory.NewWebAuthnSessionStore()
	})
	sessionID, cookie := pendingSession(t, manager)
	send := func(path string, body any, csrf bool) *httptest.ResponseRecorder {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "/realms/default/api/auth/webauthn"+path, strings.NewReader(string(raw)))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "http://test")
		request.Header.Set("Cookie", cookie+"; idmagic_csrf=csrf-val")
		if csrf {
			request.Header.Set("X-Csrf-Token", "csrf-val")
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, request)
		return rec
	}
	for _, path := range []string{"/challenge", ""} {
		oauthContractProblem(t, send(path, map[string]any{}, false), http.StatusForbidden, "csrf_failed")
	}
	challengeResponse := oauthContractJSON(t, send("/challenge", map[string]any{}, true), http.StatusOK)
	options := challengeResponse["publicKey"].(map[string]any)
	challenge := options["challenge"].(string)
	decoded, err := base64.RawURLEncoding.DecodeString(challenge)
	if err != nil || len(decoded) != 32 || options["rpId"] != "localhost" || options["userVerification"] != "preferred" {
		t.Fatalf("assertion options=%v err=%v", options, err)
	}
	allowed := options["allowCredentials"].([]any)
	if len(allowed) != 1 || allowed[0].(map[string]any)["id"] != id || allowed[0].(map[string]any)["type"] != "public-key" {
		t.Fatalf("allowCredentials=%v", allowed)
	}
	clientData, err := json.Marshal(map[string]any{"type": "webauthn.get", "challenge": challenge, "origin": "http://localhost", "crossOrigin": false})
	if err != nil {
		t.Fatal(err)
	}
	rpHash := sha256.Sum256([]byte("localhost"))
	authData := append([]byte{}, rpHash[:]...)
	authData = append(authData, 0x05) // UP と UV。
	authData = binary.BigEndian.AppendUint32(authData, 8)
	clientHash := sha256.Sum256(clientData)
	digest := sha256.Sum256(append(append([]byte{}, authData...), clientHash[:]...))
	signature, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	assertion := map[string]any{
		"id": id, "rawId": id, "type": "public-key",
		"response": map[string]string{
			"clientDataJSON": base64.RawURLEncoding.EncodeToString(clientData), "authenticatorData": base64.RawURLEncoding.EncodeToString(authData),
			"signature": base64.RawURLEncoding.EncodeToString(signature), "userHandle": base64.RawURLEncoding.EncodeToString([]byte("user_alice")),
		},
	}
	completed := oauthContractJSON(t, send("", map[string]any{"assertion": assertion, "return_to": recoveryReturnTo}, true), http.StatusOK)
	if completed["redirect_to"] != recoveryReturnTo {
		t.Fatalf("completed=%v", completed)
	}
	stored, err := sessions.Find(ctx, sessionID)
	if err != nil || stored == nil || stored.AuthenticationPending || stored.ACR != "urn:idmagic:acr:mfa" || !slices.Contains(stored.AMR, "webauthn") {
		t.Fatalf("completed session=%+v err=%v", stored, err)
	}
	saved, err := credentials.ListBySub(ctx, "user_alice")
	if err != nil || len(saved) != 1 || saved[0].SignCount != 8 || saved[0].LastUsedAt == nil {
		t.Fatalf("saved credentials=%v err=%v", saved, err)
	}
}
