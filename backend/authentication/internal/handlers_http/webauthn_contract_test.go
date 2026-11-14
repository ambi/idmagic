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
	"testing"

	"github.com/fxamacker/cbor/v2"

	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

//verify:operation StartWebAuthnRegistration: 登録 challenge、署名検証した登録、解除の HTTP 応答と credential の保存結果
//verify:operation FinishWebAuthnRegistration: 登録 challenge、署名検証した登録、解除の HTTP 応答と credential の保存結果
//verify:operation RemoveWebAuthnCredential: 登録 challenge、署名検証した登録、解除の HTTP 応答と credential の保存結果
func TestWebAuthnRegistrationHTTPContracts(t *testing.T) {
	f := newAuthRefusalServer(t)
	session := f.seedSession(t, "webauthn-contract", tenancydomain.DefaultTenantID, authRefusalAlice, withFreshStepUp)
	start := authContractJSON(t, f.send(t, authRefusalRequest{method: http.MethodPost, path: "/api/account/v1/mfa/webauthn/register/start", sessionID: session, csrf: authRefusalCSRF}), http.StatusOK)
	options := start["publicKey"].(map[string]any)
	challenge := options["challenge"].(string)
	if decoded, err := base64.RawURLEncoding.DecodeString(challenge); err != nil || len(decoded) != 32 || options["attestation"] != "none" || options["rp"].(map[string]any)["id"] != "idp.test" || options["user"].(map[string]any)["name"] != authRefusalAlice {
		t.Fatalf("options=%v err=%v", options, err)
	}
	selection := options["authenticatorSelection"].(map[string]any)
	if selection["userVerification"] != "preferred" || selection["residentKey"] != "discouraged" {
		t.Fatalf("authenticator selection=%v", selection)
	}
	attestation, credentialID, publicKey := authContractAttestation(t, challenge)
	authContractEmpty(t, f.send(t, authRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/mfa/webauthn/register/finish", sessionID: session, csrf: authRefusalCSRF,
		body: map[string]any{"attestation": attestation, "label": "Contract key"},
	}))
	ctx := testing_tenant.Default(t.Context())
	stored, err := f.credentials.FindByCredentialID(ctx, credentialID)
	if err != nil || stored == nil || stored.UserID != authRefusalAlice || stored.PublicKey != publicKey || stored.Label == nil || *stored.Label != "Contract key" || stored.SignCount != 7 {
		t.Fatalf("credential=%v err=%v", stored, err)
	}
	authContractEmpty(t, f.send(t, authRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/mfa/webauthn/remove", sessionID: session, csrf: authRefusalCSRF,
		body: map[string]string{"credential_id": credentialID},
	}))
	remaining, err := f.credentials.ListBySub(ctx, authRefusalAlice)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("remaining credentials=%v err=%v", remaining, err)
	}
}

// 他人の credential ID を名乗る登録を拒否することを固定する。応答だけでなく、持ち主の
// 記録が上書きされていないこと（user_id、公開鍵、sign_count、label）、登録者に記録が
// 増えていないこと、登録のイベントが出ていないことを観測する。sign_count を大きくした
// 上書きは、持ち主の次のアサーションを複製の疑いで拒否させるため、表明に含める。
//
//spec:covers REQ-AUTHENTICATION-038: 既に登録された credential ID の確定を 400 と invalid_webauthn で拒否し、既存の資格情報を変えず、イベントを発行しないこと
func TestWebAuthnRegistrationRefusesAnAlreadyRegisteredCredentialID(t *testing.T) {
	f := newAuthRefusalServer(t)
	ctx := testing_tenant.Default(t.Context())
	alice := f.seedSession(t, "webauthn-owner", tenancydomain.DefaultTenantID, authRefusalAlice)
	bob := f.seedSession(t, "webauthn-claimant", tenancydomain.DefaultTenantID, authRefusalBob)

	register := func(session string, id []byte, signCount uint32, label string) (*httptest.ResponseRecorder, string, string) {
		start := authContractJSON(t, f.send(t, authRefusalRequest{method: http.MethodPost, path: "/api/account/v1/mfa/webauthn/register/start", sessionID: session, csrf: authRefusalCSRF}), http.StatusOK)
		challenge := start["publicKey"].(map[string]any)["challenge"].(string)
		attestation, credentialID, publicKey := authContractAttestationWithID(t, challenge, id, signCount)
		return f.send(t, authRefusalRequest{
			method: http.MethodPost, path: "/api/account/v1/mfa/webauthn/register/finish", sessionID: session, csrf: authRefusalCSRF,
			body: map[string]any{"attestation": attestation, "label": label},
		}), credentialID, publicKey
	}

	id := make([]byte, 32)
	if _, err := rand.Read(id); err != nil {
		t.Fatal(err)
	}
	rec, credentialID, ownerKey := register(alice, id, 7, "Alice key")
	authContractEmpty(t, rec)
	registered := countEvents(*f.events, "WebAuthnCredentialRegistered")

	rec, _, _ = register(bob, id, 1000, "Bob key")
	authContractProblem(t, rec, http.StatusBadRequest, "invalid_webauthn")

	stored, err := f.credentials.FindByCredentialID(ctx, credentialID)
	if err != nil || stored == nil {
		t.Fatalf("owner credential=%v err=%v", stored, err)
	}
	if stored.UserID != authRefusalAlice || stored.PublicKey != ownerKey || stored.SignCount != 7 || stored.Label == nil || *stored.Label != "Alice key" {
		t.Fatalf("owner credential was overwritten: user=%s sign_count=%d label=%v public_key_unchanged=%t", stored.UserID, stored.SignCount, stored.Label, stored.PublicKey == ownerKey)
	}
	if claimed, err := f.credentials.ListBySub(ctx, authRefusalBob); err != nil || len(claimed) != 0 {
		t.Fatalf("claimant credentials=%v err=%v, want none", claimed, err)
	}
	if got := countEvents(*f.events, "WebAuthnCredentialRegistered"); got != registered {
		t.Fatalf("WebAuthnCredentialRegistered count=%d after the refusal, want %d: %v", got, registered, eventTypes(*f.events))
	}
}

func countEvents(events []spec.DomainEvent, eventType string) int {
	count := 0
	for _, event := range events {
		if event.EventType() == eventType {
			count++
		}
	}
	return count
}

// authContractAttestation は既存のセレモニーテストと同じ ES256 のソフトウェア認証器で、
// 実際の HTTP で受け取った challenge に対する none attestation を作る。
func authContractAttestation(t *testing.T, challenge string) (json.RawMessage, string, string) {
	t.Helper()
	id := make([]byte, 32)
	if _, err := rand.Read(id); err != nil {
		t.Fatal(err)
	}
	return authContractAttestationWithID(t, challenge, id, 7)
}

// authContractAttestationWithID は credential ID と sign count を呼び出し側が決める。
// none attestation は秘密鍵の所持を証明しないので、他人の credential ID を名乗る応答を
// 正当な形のまま作れる。
func authContractAttestationWithID(t *testing.T, challenge string, id []byte, signCount uint32) (json.RawMessage, string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	point, err := key.PublicKey.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	encoder, err := cbor.CTAP2EncOptions().EncMode()
	if err != nil {
		t.Fatal(err)
	}
	cose, err := encoder.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: point[1:33], -3: point[33:65]})
	if err != nil {
		t.Fatal(err)
	}
	attested := binary.BigEndian.AppendUint16(make([]byte, 16), uint16(len(id)))
	attested = append(attested, id...)
	attested = append(attested, cose...)
	rpHash := sha256.Sum256([]byte("idp.test"))
	authData := append(rpHash[:], byte(0x45)) // UP、UV、AT。
	authData = binary.BigEndian.AppendUint32(authData, signCount)
	authData = append(authData, attested...)
	attestation, err := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": authData})
	if err != nil {
		t.Fatal(err)
	}
	clientData, err := json.Marshal(map[string]any{"type": "webauthn.create", "challenge": challenge, "origin": authRefusalIssuer, "crossOrigin": false})
	if err != nil {
		t.Fatal(err)
	}
	encode := base64.RawURLEncoding.EncodeToString
	body, err := json.Marshal(map[string]any{
		"id": encode(id), "rawId": encode(id), "type": "public-key",
		"response": map[string]any{"clientDataJSON": encode(clientData), "attestationObject": encode(attestation)},
	})
	if err != nil {
		t.Fatal(err)
	}
	return body, encode(id), encode(cose)
}

//verify:operation StartWebAuthnRegistration: CSRF 拒否の 403 Problem Details
//verify:operation FinishWebAuthnRegistration: CSRF 拒否の 403 Problem Details
//verify:operation RemoveWebAuthnCredential: CSRF 拒否の 403 Problem Details
func TestWebAuthnRegistrationHTTPContractsRefuseCSRF(t *testing.T) {
	f := newAuthRefusalServer(t)
	session := f.seedSession(t, "webauthn-refusal", tenancydomain.DefaultTenantID, authRefusalAlice, withFreshStepUp)
	for _, path := range []string{"/api/account/v1/mfa/webauthn/register/start", "/api/account/v1/mfa/webauthn/register/finish", "/api/account/v1/mfa/webauthn/remove"} {
		authContractProblem(t, f.send(t, authRefusalRequest{method: http.MethodPost, path: path, sessionID: session}), http.StatusForbidden, "csrf_failed")
	}
	stored, err := f.credentials.ListBySub(testing_tenant.Default(t.Context()), authRefusalAlice)
	if err != nil || len(stored) != 0 || len(*f.events) != 0 || len(f.challenges.keys) != 0 {
		t.Fatalf("refusal changed state: credentials=%v err=%v", stored, err)
	}
}
