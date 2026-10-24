package usecases

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/ports"
)

type fakeEnrollmentAPI struct {
	token string
	jwk   map[string]any
	err   error
}

func (a *fakeEnrollmentAPI) Enroll(_ context.Context, token string, jwk map[string]any) (ports.Enrollment, error) {
	a.token, a.jwk = token, jwk
	if a.err != nil {
		return ports.Enrollment{}, a.err
	}
	return ports.Enrollment{ConnectorID: "c1", KeyID: "k1", Audience: "https://idp.test/realms/acme/directory-connector/v1"}, nil
}

type memoryCredentials struct {
	key        *ecdsa.PrivateKey
	enrollment *ports.Enrollment
}

func (m *memoryCredentials) SaveCredentials(_ context.Context, key *ecdsa.PrivateKey, enrollment ports.Enrollment) error {
	m.key, m.enrollment = key, &enrollment
	return nil
}

//spec:covers REQ-DIRECTORYCONNECTOR-001: 新しい P-256 の鍵対の公開鍵だけを登録トークンと送り、成功したら秘密鍵と登録の応答を保存し、登録トークンを保存しない。
func TestEnrollSendsOnlyThePublicKeyAndSavesTheCredentials(t *testing.T) {
	api := &fakeEnrollmentAPI{}
	store := &memoryCredentials{}
	enrollment, err := Enroll(context.Background(), api, store, "idmagic_dce_token")
	if err != nil {
		t.Fatal(err)
	}
	if api.token != "idmagic_dce_token" || api.jwk["kty"] != "EC" || api.jwk["crv"] != "P-256" {
		t.Fatalf("sent token=%q jwk=%v", api.token, api.jwk)
	}
	if _, private := api.jwk["d"]; private {
		t.Fatalf("jwk carries the private key: %v", api.jwk)
	}
	if store.key == nil || *store.enrollment != enrollment {
		t.Fatalf("saved key=%v enrollment=%+v", store.key, store.enrollment)
	}
	x, _ := store.key.PublicKey.Bytes()
	if got := api.jwk["x"].(string); got == "" || len(x) != 65 {
		t.Fatalf("jwk x=%q", got)
	}
	saved, err := json.Marshal(store.enrollment)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(saved), "idmagic_dce_") {
		t.Fatalf("saved enrollment contains the token: %s", saved)
	}

	second := &fakeEnrollmentAPI{}
	if _, err := Enroll(context.Background(), second, store, "idmagic_dce_again"); err != nil {
		t.Fatal(err)
	}
	if second.jwk["x"] == api.jwk["x"] {
		t.Fatal("re-enrollment reused the previous key pair")
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-001: 登録が拒否されたら、保存済みの秘密鍵と登録の情報を変えない。
func TestEnrollKeepsTheStateWhenRefused(t *testing.T) {
	refused := &ports.APIError{Status: 401, Type: "invalid_token"}
	store := &memoryCredentials{}
	if _, err := Enroll(context.Background(), &fakeEnrollmentAPI{err: refused}, store, "idmagic_dce_expired"); !errors.Is(err, refused) {
		t.Fatalf("error = %v", err)
	}
	if store.key != nil || store.enrollment != nil {
		t.Fatal("refused enrollment changed the state")
	}
}
