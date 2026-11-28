package usecases_test

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	webauthnusecases "github.com/ambi/idmagic/backend/authentication/internal/webauthn/usecases"
	webauthnports "github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

// chromeRecording は frontend/tests/record/webauthn.ts が記録した、画面がサーバーへ送った本文である。
type chromeRecording struct {
	testing_httprecord.Recording
	Setup struct {
		RPID     string `json:"rp_id"`
		RPOrigin string `json:"rp_origin"`
	} `json:"setup"`
}

// creationOptions は記録の登録の開始の応答の publicKey である。
type creationOptions struct {
	Challenge string `json:"challenge"`
	RP        struct {
		ID string `json:"id"`
	} `json:"rp"`
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	PubKeyCredParams       []json.RawMessage `json:"pubKeyCredParams"`
	AuthenticatorSelection struct {
		UserVerification string `json:"userVerification"`
	} `json:"authenticatorSelection"`
}

type requestOptions struct {
	Challenge        string `json:"challenge"`
	RPID             string `json:"rpId"`
	UserVerification string `json:"userVerification"`
	AllowCredentials []struct {
		ID string `json:"id"`
	} `json:"allowCredentials"`
}

func loadChromeRecording(t *testing.T) chromeRecording {
	t.Helper()
	data, err := os.ReadFile("testdata/chrome-virtual-authenticator.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testing_httprecord.Decode(data); err != nil {
		t.Fatal(err)
	}
	var rec chromeRecording
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatal(err)
	}
	return rec
}

// field は記録の本文から名前の欄を取り出す。
func field(t *testing.T, body json.RawMessage, name string) json.RawMessage {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatal(err)
	}
	value, ok := object[name]
	if !ok {
		t.Fatalf("recorded body has no %q: %s", name, body)
	}
	return value
}

func decodeInto(t *testing.T, raw json.RawMessage, into any) {
	t.Helper()
	if err := json.Unmarshal(raw, into); err != nil {
		t.Fatal(err)
	}
}

// seedSession は、サーバーが開始の応答とともに保存したはずのセッションを、記録の開始の応答から置く。
// 形は go-webauthn の SessionData の JSON であり、TestWebAuthnCeremonySessionsKeepTheStoredJSONShape が固定している。
func seedSession(ctx context.Context, t *testing.T, store webauthnports.WebAuthnSessionStore, key string, session map[string]any) {
	t.Helper()
	payload, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, key, webauthnports.WebAuthnSession{Payload: payload}, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
}

// Chrome の仮想認証器が画面を通して送った登録とサインインの二段目の本文を再生する。
// 期待値は WebAuthn Level 3 §7.1 と §7.2 から書く。登録は記録の credential ID と transports を保存して
// 利用者の MFA を有効にし、認証は同じ credential として検証を通って authenticatorData の signCount を保存する。
// 記録の値は challenge、origin、RP ID と認証器の鍵に縛られるので、サーバーの状態を記録に合わせる。
//
//spec:covers WEBAUTHN3-REGISTRATION: Chrome が画面を通して送った attestation を、記録の challenge、RP ID、origin で検証し、credential ID と transports を保存すること,
//spec:covers WEBAUTHN3-AUTHENTICATION, REQ-AUTHENTICATION-040: 同じ credential の assertion をサインインの二段目で受け付けて signCount を保存し、署名を変えた assertion を拒否すること
func TestWebAuthnCeremonies_AcceptTheBodiesChromeSentThroughTheUI(t *testing.T) {
	rec := loadChromeRecording(t)
	ctx := testing_tenant.Default(context.Background())

	var creation creationOptions
	decodeInto(t, field(t, rec.Exchange(t, "01-register-start").Response.Body, "publicKey"), &creation)
	userID, err := base64.RawURLEncoding.DecodeString(creation.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	sub := string(userID)

	rp, err := webauthnusecases.NewRelyingParty(webauthnusecases.WebAuthnConfig{
		RPID: rec.Setup.RPID, RPDisplayName: "IdMagic E2E", RPOrigins: []string{rec.Setup.RPOrigin},
	})
	if err != nil {
		t.Fatal(err)
	}
	deps, userRepo, _ := newWebAuthnDeps(t, rp)
	now := time.Now().UTC()
	if err := userRepo.Save(ctx, &userdomain.User{ID: sub, PreferredUsername: "root", PasswordHash: "unused", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}

	params := make([]any, len(creation.PubKeyCredParams))
	for i, param := range creation.PubKeyCredParams {
		params[i] = param
	}
	seedSession(ctx, t, deps.SessionStore, "reg:"+sub, map[string]any{
		"challenge": creation.Challenge, "rpId": creation.RP.ID, "user_id": userID,
		"expires": time.Time{}, "userVerification": creation.AuthenticatorSelection.UserVerification, "credParams": params,
	})
	attestation := field(t, rec.Exchange(t, "02-register-finish").Request.Body, "attestation")
	if err := webauthnusecases.FinishWebAuthnRegistration(ctx, deps, sub, attestation, nil, now); err != nil {
		t.Fatalf("FinishWebAuthnRegistration rejected the attestation Chrome sent: %v", err)
	}
	var sent struct {
		ID       string `json:"id"`
		Response struct {
			Transports []string `json:"transports"`
		} `json:"response"`
	}
	decodeInto(t, attestation, &sent)
	stored, err := deps.CredentialRepo.ListBySub(ctx, sub)
	if err != nil || len(stored) != 1 || stored[0].CredentialID != sent.ID {
		t.Fatalf("stored credentials = %+v (%v), want the one credential %s", stored, err, sent.ID)
	}
	if fmt.Sprint(stored[0].Transports) != fmt.Sprint(sent.Response.Transports) {
		t.Errorf("transports = %v, want the sent %v", stored[0].Transports, sent.Response.Transports)
	}
	if user, _ := userRepo.FindBySub(ctx, sub); user == nil || !user.MfaEnrolled {
		t.Errorf("the first credential must enrol MFA: %+v", user)
	}

	var request requestOptions
	decodeInto(t, field(t, rec.Exchange(t, "03-webauthn-challenge").Response.Body, "publicKey"), &request)
	allowed := make([][]byte, 0, len(request.AllowCredentials))
	for _, credential := range request.AllowCredentials {
		id, err := base64.RawURLEncoding.DecodeString(credential.ID)
		if err != nil {
			t.Fatal(err)
		}
		allowed = append(allowed, id)
	}
	loginSession := map[string]any{
		"challenge": request.Challenge, "rpId": request.RPID, "user_id": userID, "allowed_credentials": allowed,
		"expires": time.Time{}, "userVerification": request.UserVerification,
	}
	assertion := field(t, rec.Exchange(t, "04-auth-webauthn").Request.Body, "assertion")

	t.Run("a signature changed after signing is rejected", func(t *testing.T) {
		seedSession(ctx, t, deps.SessionStore, "login:replay", loginSession)
		var forged map[string]any
		decodeInto(t, assertion, &forged)
		response := forged["response"].(map[string]any)
		signature, err := base64.RawURLEncoding.DecodeString(response["signature"].(string))
		if err != nil {
			t.Fatal(err)
		}
		signature[len(signature)-1] ^= 0x01
		response["signature"] = base64.RawURLEncoding.EncodeToString(signature)
		body, err := json.Marshal(forged)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := webauthnusecases.FinishWebAuthnAssertion(ctx, deps, "replay", sub, body, now); !errors.Is(err, webauthnusecases.ErrWebAuthnVerification) {
			t.Fatalf("err = %v, want ErrWebAuthnVerification", err)
		}
	})

	seedSession(ctx, t, deps.SessionStore, "login:replay", loginSession)
	credential, err := webauthnusecases.FinishWebAuthnAssertion(ctx, deps, "replay", sub, assertion, now)
	if err != nil {
		t.Fatalf("FinishWebAuthnAssertion rejected the assertion Chrome sent: %v", err)
	}
	var signed struct {
		Response struct {
			AuthenticatorData string `json:"authenticatorData"`
		} `json:"response"`
	}
	decodeInto(t, assertion, &signed)
	authData, err := base64.RawURLEncoding.DecodeString(signed.Response.AuthenticatorData)
	if err != nil || len(authData) < 37 {
		t.Fatalf("authenticatorData = %d byte(s) (%v)", len(authData), err)
	}
	if credential.CredentialID != sent.ID || credential.SignCount != binary.BigEndian.Uint32(authData[33:37]) {
		t.Fatalf("credential = %s with sign count %d, want %s with %d", credential.CredentialID, credential.SignCount, sent.ID, binary.BigEndian.Uint32(authData[33:37]))
	}
}
