package handlers_http_test

// docs/standards.md の GDPR-ERASURE のうち、Purge が Authentication の資格情報まで届いて
// いることを、管理 API の入口から観測する。
//
// use case を直接呼ぶテストでは、この観測は書けない。cascade の到達先は
// `backend/shared/http/server_http/routes.go` が `idmhttp.Deps` を組み、
// `adminUserDeps` がそれを `AdminUserDeps` へ写して初めて届く。どちらの層も use case を
// 直接呼ぶ経路には現れないので、写し漏れがあっても use case 側のテストは緑のままになる。
// したがってここでは httpadapter.Register が組んだ本番と同じ経路を通す。

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	recoverymemory "github.com/ambi/idmagic/backend/authentication/recovery/db_memory"
	recoverydomain "github.com/ambi/idmagic/backend/authentication/recovery/domain"
	webauthnmemory "github.com/ambi/idmagic/backend/authentication/webauthn/db_memory"
	webauthndomain "github.com/ambi/idmagic/backend/authentication/webauthn/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
)

// credentialCascadeFixture は消去する利用者と、消去しない対照の利用者に、同じ形の資格情報を
// 置く。対照が無いと、sub で絞らず全件消す実装も同じように緑になる。
type credentialCascadeFixture struct {
	credentials *webauthnmemory.WebAuthnCredentialRepository
	recovery    *recoverymemory.RecoveryCodeRepository
}

func (f *credentialCascadeFixture) seed(t *testing.T, sub string) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	label := "YubiKey " + sub
	if err := f.credentials.Save(ctx, &webauthndomain.WebAuthnCredential{
		CredentialID: "credential-" + sub, UserID: sub,
		PublicKey: "cG9zdC1xdWFudHVtLXB1YmxpYy1rZXk", Label: &label, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.recovery.ReplaceAll(ctx, sub, []*recoverydomain.RecoveryCode{
		{UserID: sub, CodeHash: "hash-a-" + sub, GeneratedAt: now},
		{UserID: sub, CodeHash: "hash-b-" + sub, GeneratedAt: now},
	}); err != nil {
		t.Fatal(err)
	}
}

// counts は、その利用者にまだ引ける資格情報の数を返す。WebAuthn の側は
// BeginWebAuthnAssertion / FinishWebAuthnAssertion が ErrWebAuthnNoCredential を返すかどうかを
// 決める述語そのものである。
func (f *credentialCascadeFixture) counts(t *testing.T, sub string) (int, int) {
	t.Helper()
	ctx := context.Background()
	credentials, err := f.credentials.ListBySub(ctx, sub)
	if err != nil {
		t.Fatal(err)
	}
	codes, err := f.recovery.ListBySub(ctx, sub)
	if err != nil {
		t.Fatal(err)
	}
	return len(credentials), len(codes)
}

// リカバリコードまで届く。届かせているのは本番の配線なので、入口も本番と同じにする。
//
//spec:covers GDPR-ERASURE: 管理 API の消去要求が、Authentication が持つ WebAuthn 資格情報と
func TestAdminUserAPIPurgeDestroysWebAuthnCredentialsAndRecoveryCodes(t *testing.T) {
	fixture := &credentialCascadeFixture{
		credentials: webauthnmemory.NewWebAuthnCredentialRepository(),
		recovery:    recoverymemory.NewRecoveryCodeRepository(),
	}
	e, _ := newAdminUserHandler(t, func(deps *httpadapter.Deps) {
		deps.WebAuthnCredentialRepo = fixture.credentials
		deps.RecoveryCodeRepo = fixture.recovery
	})
	csrf, cookie := adminCSRF(t, e)

	create := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/users", csrf, cookie, map[string]any{
		"preferred_username": "erasure-subject",
		"password":           "initial-password-9182",
		"email":              "erasure.subject@example.com",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	fixture.seed(t, created.ID)
	// 対照。消去要求の対象ではないので、資格情報は 1 つも減ってはならない。
	fixture.seed(t, "regular")

	if credentials, codes := fixture.counts(t, created.ID); credentials != 1 || codes != 2 {
		t.Fatalf("消去前の資格情報 webauthn=%d recovery=%d, want 1 と 2", credentials, codes)
	}

	del := adminJSONRequest(t, e, http.MethodDelete,
		"/api/admin/v1/users/"+created.ID+"?purge=true", csrf, cookie,
		map[string]any{"reason": "erasure request"})
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", del.Code, del.Body.String())
	}

	if credentials, codes := fixture.counts(t, created.ID); credentials != 0 || codes != 0 {
		t.Fatalf("消去後に残った資格情報 webauthn=%d recovery=%d, want どちらも 0", credentials, codes)
	}
	// credential id を知っていても引けない。
	if found, err := fixture.credentials.FindByCredentialID(
		context.Background(), "credential-"+created.ID,
	); err != nil || found != nil {
		t.Fatalf("消去後も credential id から引ける: %+v err=%v", found, err)
	}

	if credentials, codes := fixture.counts(t, "regular"); credentials != 1 || codes != 2 {
		t.Fatalf("対照の利用者の資格情報が消えた webauthn=%d recovery=%d, want 1 と 2", credentials, codes)
	}
}
