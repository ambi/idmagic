package bootstrap

import (
	"maps"
	"strings"
	"testing"

	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
)

// assembleKeyStore は Run() と同じ順序を辿る。起動時設定を読み、拒否が 1 つも
// 無いときにだけ永続層の KeyStore を組み立てて KeyProvider の選択へ渡す。
// PERSISTENCE=postgres での fallback は signingkeys/db_postgres の KeyStore で
// あり (postgres.go)、秘密 JWK を signing_keys.private_jwk に平文で持つ。ここ
// では同じ位置に代役を置き、返った KeyStore がその代役と同一かどうかで、平文の
// 鍵素材を持つ経路が選ばれたかを判定する。
func assembleKeyStore(t *testing.T, env map[string]string) (selected, persisted signingports.KeyStore, err error) {
	t.Helper()
	l := NewConfigLoader(stubEnv(env))
	cfg := LoadSharedConfig(l)
	if err := l.Err(); err != nil {
		return nil, nil, err
	}
	persisted, storeErr := signingmemory.NewInMemoryKeyStore()
	if storeErr != nil {
		t.Fatalf("NewInMemoryKeyStore: %v", storeErr)
	}
	return selectKeyStore(cfg, persisted), persisted, nil
}

// persistentEnv は鍵素材が永続化される最小の環境に choice を重ねる。
func persistentEnv(choice map[string]string) map[string]string {
	env := map[string]string{
		"PERSISTENCE":  "postgres",
		"DATABASE_URL": "postgres://idmagic@db.internal:5432/idmagic",
	}
	maps.Copy(env, choice)
	return env
}

// TestPlaintextKeyCustodyIsNeverImplicit covers REQ-SIGNINGKEYS-012: 鍵素材が
// 永続化される配備で KEY_PROVIDER を書かなければ起動を拒否し、秘密鍵を平文で
// 保存する KeyStore はそもそも組み立てられない。明示して選んだときにだけその
// 経路が開く。
func TestPlaintextKeyCustodyIsNeverImplicit(t *testing.T) {
	t.Parallel()

	selected, _, err := assembleKeyStore(t, persistentEnv(nil))
	if err == nil {
		t.Fatal("PERSISTENCE=postgres without KEY_PROVIDER must be refused before any key store is assembled")
	}
	if !strings.Contains(err.Error(), "KEY_PROVIDER") {
		t.Errorf("err=%v, want the refusal to name KEY_PROVIDER", err)
	}
	if selected != nil {
		t.Errorf("a key store was assembled despite the refusal: %T", selected)
	}

	selected, persisted, err := assembleKeyStore(t, persistentEnv(map[string]string{"KEY_PROVIDER": "local"}))
	if err != nil {
		t.Fatalf("an explicit KEY_PROVIDER=local must load: %v", err)
	}
	if selected != persisted {
		t.Errorf("selected = %T, want the persistence layer's key store once the operator chose it", selected)
	}

	selected, persisted, err = assembleKeyStore(t, persistentEnv(map[string]string{
		"KEY_PROVIDER": "vault",
		"VAULT_ADDR":   "https://vault.example.com",
		"VAULT_TOKEN":  "t",
	}))
	if err != nil {
		t.Fatalf("an explicit KEY_PROVIDER=vault must load: %v", err)
	}
	if selected == persisted {
		t.Error("KEY_PROVIDER=vault must not fall back to the key store that persists cleartext key material")
	}
}

// TestMemoryPersistenceNeedsNoKeyCustodyChoice guards the other half of
// REQ-SIGNINGKEYS-012's precondition: 鍵素材がプロセスの外へ出ない配備まで
// KEY_PROVIDER の明示を求めると、要求が「平文で永続化される構成を選んだこと」
// を表さなくなる。
func TestMemoryPersistenceNeedsNoKeyCustodyChoice(t *testing.T) {
	t.Parallel()
	if _, _, err := assembleKeyStore(t, map[string]string{}); err != nil {
		t.Fatalf("PERSISTENCE=memory must load without KEY_PROVIDER: %v", err)
	}
}
