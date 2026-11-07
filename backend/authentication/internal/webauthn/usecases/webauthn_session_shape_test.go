package usecases_test

// セレモニーのセッションは開始と完了の間に DB へ置かれ、開始と完了の間にデプロイをまたぐと、
// 旧いバージョンが書いた JSON を新しいバージョンが読む。ここではその JSON の形を固定する。

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	webauthnmemory "github.com/ambi/idmagic/backend/authentication/internal/webauthn/db_memory"
	"github.com/ambi/idmagic/backend/authentication/internal/webauthn/usecases"
	"github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

// recordingSessionStore は最後に保存された鍵とセッションを覚える。
type recordingSessionStore struct {
	*webauthnmemory.WebAuthnSessionStore
	key     string
	session ports.WebAuthnSession
}

func (s *recordingSessionStore) Save(
	ctx context.Context, key string, session ports.WebAuthnSession, expiresAt time.Time,
) error {
	s.key, s.session = key, session
	return s.WebAuthnSessionStore.Save(ctx, key, session, expiresAt)
}

func topLevelKeys(t *testing.T, payload []byte) []string {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatalf("stored session is not a JSON object: %v (%s)", err, payload)
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// 保存する JSON のキーは go-webauthn の SessionData の JSON タグである。キーが変わると、
// デプロイ前に開始したセレモニーの状態を新しいバージョンが読めず、そのセレモニーは失敗する。
func TestWebAuthnCeremonySessionsKeepTheStoredJSONShape(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	deps, _, _ := newWebAuthnDeps(t, testRP(t))
	store := &recordingSessionStore{WebAuthnSessionStore: webauthnmemory.NewWebAuthnSessionStore()}
	deps.SessionStore = store

	startRegistration(ctx, t, deps)
	registrationKeys := []string{"challenge", "credParams", "expires", "rpId", "userVerification", "user_id"}
	if got := topLevelKeys(t, store.session.Payload); !slices.Equal(got, registrationKeys) {
		t.Fatalf("registration session keys = %v, want %v\n%s", got, registrationKeys, store.session.Payload)
	}

	registerCredential(ctx, t, deps, time.Now())
	beginAssertion(ctx, t, deps)
	loginKeys := []string{"allowed_credentials", "challenge", "expires", "rpId", "userVerification", "user_id"}
	if got := topLevelKeys(t, store.session.Payload); !slices.Equal(got, loginKeys) {
		t.Fatalf("login session keys = %v, want %v\n%s", got, loginKeys, store.session.Payload)
	}
}

// 変更前のバージョンが書いた形の JSON を、そのまま登録の完了に使えることを確かめる。
// 値は変更前の保存処理 (go-webauthn の SessionData を json.Marshal したもの) の出力を写した。
func TestWebAuthnRegistrationFinishesFromASessionWrittenByAnEarlierVersion(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	deps, _, _ := newWebAuthnDeps(t, testRP(t))
	store := &recordingSessionStore{WebAuthnSessionStore: webauthnmemory.NewWebAuthnSessionStore()}
	deps.SessionStore = store

	challenge := startRegistration(ctx, t, deps)
	earlier := ports.WebAuthnSession{Payload: json.RawMessage(fmt.Sprintf(
		`{"challenge":%q,"rpId":"localhost","user_id":"dXNlci1hbGljZQ==","expires":"0001-01-01T00:00:00Z",`+
			`"userVerification":"preferred","credParams":[{"type":"public-key","alg":-7}]}`,
		challenge,
	))}
	if err := store.WebAuthnSessionStore.Save(ctx, store.key, earlier, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	body := newSoftwareAuthenticator(t).register(t, challenge, testRPID, testOrigin)
	if err := usecases.FinishWebAuthnRegistration(ctx, deps, "user-alice", body, nil, time.Now()); err != nil {
		t.Fatalf("finish from an earlier session: %v", err)
	}
	stored, err := deps.CredentialRepo.ListBySub(ctx, "user-alice")
	if err != nil || len(stored) != 1 {
		t.Fatalf("credentials = %d (%v), want 1", len(stored), err)
	}
}
