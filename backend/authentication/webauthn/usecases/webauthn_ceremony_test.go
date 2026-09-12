package usecases_test

// WebAuthn の登録と認証を、実際に鍵を持つソフトウェア認証器で通しで踏む。既存のテストは
// RP 未設定と壊れた body しか踏んでおらず、challenge / RP ID / origin の検証も、COSE 公開鍵と
// sign count の保存も観測していなかった。攻撃者が差し替えられるのは authenticator の応答
// なので、その応答を組み立てられるところまで下りないと、検証の有無を区別できない。

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	"github.com/ambi/idmagic/backend/authentication/webauthn/usecases"
)

const (
	testRPID   = "localhost"
	testOrigin = "http://localhost"

	// authenticator data の flags。UP は user present、UV は user verified、
	// AT は attested credential data が続くこと。
	flagUserPresent  = 0x01
	flagUserVerified = 0x04
	flagAttestedData = 0x40
)

// softwareAuthenticator は 1 つの ES256 クレデンシャルを持つ認証器である。RP ID と origin を
// 引数に取るのは、テストがそれを詐称した応答を作れるようにするためである。
type softwareAuthenticator struct {
	key          *ecdsa.PrivateKey
	credentialID []byte
	signCount    uint32
}

func newSoftwareAuthenticator(t *testing.T) *softwareAuthenticator {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	credentialID := make([]byte, 32)
	if _, err := rand.Read(credentialID); err != nil {
		t.Fatal(err)
	}
	return &softwareAuthenticator{key: key, credentialID: credentialID}
}

// coseKey は ES256 の公開鍵を COSE_Key として符号化する。保存された公開鍵がこれと
// 一致することが、「COSE の公開鍵を保存する」の観測になる。
func (a *softwareAuthenticator) coseKey(t *testing.T) []byte {
	t.Helper()
	// 非圧縮点は 0x04 || X(32) || Y(32)。
	point, err := a.key.PublicKey.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(point) != 65 || point[0] != 4 {
		t.Fatalf("unexpected uncompressed point of %d byte(s)", len(point))
	}
	x, y := point[1:33], point[33:65]
	// CTAP2 の正準符号化。実機の authenticator が出すのと同じ並びになり、保存された
	// バイト列とそのまま比較できる。
	encoder, encErr := cbor.CTAP2EncOptions().EncMode()
	if encErr != nil {
		t.Fatal(encErr)
	}
	encoded, err := encoder.Marshal(map[int]any{
		1:  2,  // kty: EC2
		3:  -7, // alg: ES256
		-1: 1,  // crv: P-256
		-2: x,
		-3: y,
	})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func clientData(t *testing.T, ceremony, challenge, origin string) []byte {
	t.Helper()
	encoded, err := json.Marshal(map[string]any{
		"type": ceremony, "challenge": challenge, "origin": origin, "crossOrigin": false,
	})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func authenticatorData(rpID string, flags byte, signCount uint32, attested []byte) []byte {
	rpIDHash := sha256.Sum256([]byte(rpID))
	data := make([]byte, 0, 37+len(attested))
	data = append(data, rpIDHash[:]...)
	data = append(data, flags)
	data = binary.BigEndian.AppendUint32(data, signCount)
	return append(data, attested...)
}

// attestedCredentialData は aaguid(16) + credential id 長(2) + credential id + COSE 公開鍵。
func (a *softwareAuthenticator) attestedCredentialData(t *testing.T) []byte {
	t.Helper()
	coseKey := a.coseKey(t)
	data := make([]byte, 0, 16+2+len(a.credentialID)+len(coseKey))
	data = append(data, make([]byte, 16)...) // aaguid はゼロ (self attestation を持たない authenticator)
	data = binary.BigEndian.AppendUint16(data, uint16(len(a.credentialID)))
	data = append(data, a.credentialID...)
	return append(data, coseKey...)
}

// register は attestation 応答の JSON を組み立てる。fmt は "none" で、prod が
// PreferNoAttestation を要求しているのと一致する。
func (a *softwareAuthenticator) register(t *testing.T, challenge, rpID, origin string) []byte {
	t.Helper()
	collected := clientData(t, "webauthn.create", challenge, origin)
	authData := authenticatorData(
		rpID, flagUserPresent|flagUserVerified|flagAttestedData, a.signCount, a.attestedCredentialData(t),
	)
	attestation, err := cbor.Marshal(map[string]any{
		"fmt": "none", "attStmt": map[string]any{}, "authData": authData,
	})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"id":    base64.RawURLEncoding.EncodeToString(a.credentialID),
		"rawId": base64.RawURLEncoding.EncodeToString(a.credentialID),
		"type":  "public-key",
		"response": map[string]any{
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString(collected),
			"attestationObject": base64.RawURLEncoding.EncodeToString(attestation),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// assert は assertion 応答の JSON を組み立てる。signer が nil でなければその鍵で署名する
// ので、登録済みでない鍵の署名を作れる。
func (a *softwareAuthenticator) assert(
	t *testing.T, challenge, rpID, origin, userHandle string, signer *ecdsa.PrivateKey,
) []byte {
	t.Helper()
	if signer == nil {
		signer = a.key
	}
	a.signCount++
	collected := clientData(t, "webauthn.get", challenge, origin)
	authData := authenticatorData(rpID, flagUserPresent|flagUserVerified, a.signCount, nil)
	digest := sha256.Sum256(append(append([]byte{}, authData...), hashOf(collected)...))
	signature, err := ecdsa.SignASN1(rand.Reader, signer, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"id":    base64.RawURLEncoding.EncodeToString(a.credentialID),
		"rawId": base64.RawURLEncoding.EncodeToString(a.credentialID),
		"type":  "public-key",
		"response": map[string]any{
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString(collected),
			"authenticatorData": base64.RawURLEncoding.EncodeToString(authData),
			"signature":         base64.RawURLEncoding.EncodeToString(signature),
			"userHandle":        base64.RawURLEncoding.EncodeToString([]byte(userHandle)),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func hashOf(value []byte) []byte {
	sum := sha256.Sum256(value)
	return sum[:]
}

func testRP(t *testing.T) *gowebauthn.WebAuthn {
	t.Helper()
	rp, err := usecases.NewWebAuthn(usecases.WebAuthnConfig{
		RPID: testRPID, RPDisplayName: "idmagic", RPOrigins: []string{testOrigin},
	})
	if err != nil {
		t.Fatal(err)
	}
	return rp
}

// startRegistration は challenge を発行し、その base64url 値を返す。
func startRegistration(ctx context.Context, t *testing.T, deps usecases.WebAuthnDeps) string {
	t.Helper()
	creation, err := usecases.StartWebAuthnRegistration(ctx, deps, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	return creation.Response.Challenge.String()
}

// registerCredential は渡された deps へ正当な登録を 1 件済ませ、その鍵を持つ認証器を返す。
func registerCredential(
	ctx context.Context, t *testing.T, deps usecases.WebAuthnDeps, now time.Time,
) *softwareAuthenticator {
	t.Helper()
	authenticator := newSoftwareAuthenticator(t)
	body := authenticator.register(t, startRegistration(ctx, t, deps), testRPID, testOrigin)
	if err := usecases.FinishWebAuthnRegistration(ctx, deps, "user-alice", body, nil, now); err != nil {
		t.Fatal(err)
	}
	return authenticator
}

// beginAssertion は assertion challenge を発行し、その base64url 値を返す。
func beginAssertion(ctx context.Context, t *testing.T, deps usecases.WebAuthnDeps) string {
	t.Helper()
	assertion, err := usecases.BeginWebAuthnAssertion(ctx, deps, "login-1", "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	return assertion.Response.Challenge.String()
}

// COSE の公開鍵と sign count を保存することを固定する。4 つを別々の入力で観測する:
// 詐称する要素をひとつだけ差し替えた 3 通りの応答が拒否され、そのとき credential が
// 1 つも保存されていないこと (拒否が防いだ効果) と、正当な応答では authenticator が
// 出した COSE 公開鍵そのものと sign count が保存先に残っていることである。
// 保存の観測を「保存された」だけにすると、公開鍵を捨てて credential id だけを持つ実装を
// 区別できない。
//
//spec:covers WEBAUTHN3-REGISTRATION: 登録が attestation の challenge / RP ID / origin を検証し、
func TestWebAuthnRegistrationVerifiesTheCeremonyAndStoresTheCOSEKey(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	for name, forge := range map[string]struct {
		challenge, rpID, origin string
	}{
		"another challenge": {challenge: base64.RawURLEncoding.EncodeToString([]byte("not-the-issued-challenge")), rpID: testRPID, origin: testOrigin},
		"another RP ID":     {rpID: "evil.example", origin: testOrigin},
		"another origin":    {rpID: testRPID, origin: "http://evil.example"},
	} {
		deps, _, _ := newWebAuthnDeps(t, testRP(t))
		authenticator := newSoftwareAuthenticator(t)
		challenge := startRegistration(ctx, t, deps)
		if forge.challenge != "" {
			challenge = forge.challenge
		}
		body := authenticator.register(t, challenge, forge.rpID, forge.origin)
		if err := usecases.FinishWebAuthnRegistration(ctx, deps, "user-alice", body, nil, now); !errors.Is(err, usecases.ErrWebAuthnVerification) {
			t.Fatalf("%s: err=%v, want ErrWebAuthnVerification", name, err)
		}
		stored, err := deps.CredentialRepo.ListBySub(ctx, "user-alice")
		if err != nil {
			t.Fatal(err)
		}
		if len(stored) != 0 {
			t.Fatalf("%s: %d credential(s) registered despite the refusal", name, len(stored))
		}
	}

	deps, _, _ := newWebAuthnDeps(t, testRP(t))
	authenticator := newSoftwareAuthenticator(t)
	authenticator.signCount = 7
	challenge := startRegistration(ctx, t, deps)
	body := authenticator.register(t, challenge, testRPID, testOrigin)
	if err := usecases.FinishWebAuthnRegistration(ctx, deps, "user-alice", body, nil, now); err != nil {
		t.Fatalf("legitimate registration rejected: %v", err)
	}
	stored, err := deps.CredentialRepo.ListBySub(ctx, "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 {
		t.Fatalf("credentials=%d, want 1", len(stored))
	}
	want := base64.RawURLEncoding.EncodeToString(authenticator.coseKey(t))
	if stored[0].PublicKey != want {
		t.Fatalf("stored public key=%q, want the COSE key the authenticator produced %q", stored[0].PublicKey, want)
	}
	if stored[0].CredentialID != base64.RawURLEncoding.EncodeToString(authenticator.credentialID) {
		t.Fatalf("stored credential id=%q", stored[0].CredentialID)
	}
	if stored[0].SignCount != 7 {
		t.Fatalf("stored sign_count=%d, want the 7 the authenticator reported", stored[0].SignCount)
	}
}

// クレデンシャルを検証することを固定する。登録した鍵の署名だけが通ること、別の鍵で
// 署名した assertion、別の origin の assertion、別の RP ID の assertion がいずれも拒否
// されること、そして拒否のとき sign count が進んでいないこと (拒否が防いだ効果) を
// 観測する。戻り値の error だけを見ると、検証に失敗しても sign count を書き換える実装を
// 区別できない。登録側と別のテストにしてあるのは、片方の検証だけを持つ実装が 1 つの
// テストでは区別できないからである。
//
//spec:covers WEBAUTHN3-AUTHENTICATION: 認証が、オリジンと Relying Party の範囲に限定された公開鍵
func TestWebAuthnAuthenticationVerifiesTheOriginAndRelyingPartyScopedCredential(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	otherKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for name, forge := range map[string]struct {
		rpID, origin string
		signer       *ecdsa.PrivateKey
	}{
		"a key that was never registered": {rpID: testRPID, origin: testOrigin, signer: otherKey},
		"another origin":                  {rpID: testRPID, origin: "http://evil.example"},
		"another RP ID":                   {rpID: "evil.example", origin: testOrigin},
	} {
		deps, _, _ := newWebAuthnDeps(t, testRP(t))
		authenticator := registerCredential(ctx, t, deps, now)
		challenge := beginAssertion(ctx, t, deps)
		body := authenticator.assert(t, challenge, forge.rpID, forge.origin, "user-alice", forge.signer)
		if _, err := usecases.FinishWebAuthnAssertion(ctx, deps, "login-1", "user-alice", body, now); !errors.Is(err, usecases.ErrWebAuthnVerification) {
			t.Fatalf("%s: err=%v, want ErrWebAuthnVerification", name, err)
		}
		credential, err := deps.CredentialRepo.FindByCredentialID(
			ctx, base64.RawURLEncoding.EncodeToString(authenticator.credentialID),
		)
		if err != nil {
			t.Fatal(err)
		}
		if credential.SignCount != 0 {
			t.Fatalf("%s: sign_count advanced to %d despite the refusal", name, credential.SignCount)
		}
		if credential.LastUsedAt != nil {
			t.Fatalf("%s: last_used_at recorded despite the refusal", name)
		}
	}

	deps, _, _ := newWebAuthnDeps(t, testRP(t))
	authenticator := registerCredential(ctx, t, deps, now)
	challenge := beginAssertion(ctx, t, deps)
	body := authenticator.assert(t, challenge, testRPID, testOrigin, "user-alice", nil)
	verified, err := usecases.FinishWebAuthnAssertion(ctx, deps, "login-1", "user-alice", body, now)
	if err != nil {
		t.Fatalf("legitimate assertion rejected: %v", err)
	}
	if verified.CredentialID != base64.RawURLEncoding.EncodeToString(authenticator.credentialID) {
		t.Fatalf("verified credential=%q", verified.CredentialID)
	}
	if verified.SignCount != authenticator.signCount {
		t.Fatalf("sign_count=%d, want the %d the authenticator reported", verified.SignCount, authenticator.signCount)
	}
}
