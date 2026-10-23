package keys_vault

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// KeySetState は Vault の公開部分と選択だけを保存する。秘密鍵は Vault に留まる。
type (
	KeySetState struct{ Records []KeyRecord }
	KeyRecord   struct {
		Key     signingdomain.SigningKey
		Version int
	}
)

// KeyStateStore は集合ごとに更新を直列化し、失敗した更新を残さない。
type KeyStateStore interface {
	Load(context.Context) (KeySetState, error)
	Update(context.Context, func(*KeySetState) error) error
	// Scopes は ctx のテナントと用途で状態を持つスコープを返す。
	Scopes(context.Context) ([]string, error)
	Healthy(context.Context) bool
}

type MemoryKeyStateStore struct {
	mu     sync.Mutex
	states map[string][]byte
}

func NewMemoryKeyStateStore() *MemoryKeyStateStore {
	return &MemoryKeyStateStore{states: map[string][]byte{}}
}

func stateID(ctx context.Context) string {
	return vaultSetID(tenantports.TenantID(ctx), signingports.KeyUsage(ctx), signingports.KeyScope(ctx))
}

// setScopes は、vaultSetID を鍵とする map から、ctx のテナントと用途に属するスコープを返す。
func setScopes[V any](ctx context.Context, sets map[string]V) []string {
	prefix := vaultSetID(tenantports.TenantID(ctx), signingports.KeyUsage(ctx), "")
	scopes := []string{}
	for id := range sets {
		if scopeID, ok := strings.CutPrefix(id, prefix); ok {
			scopes = append(scopes, scopeID)
		}
	}
	return scopes
}

func (m *MemoryKeyStateStore) Load(ctx context.Context) (KeySetState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var state KeySetState
	if payload := m.states[stateID(ctx)]; payload != nil {
		if err := json.Unmarshal(payload, &state); err != nil {
			return state, err
		}
	}
	return state, nil
}

func (m *MemoryKeyStateStore) Update(ctx context.Context, update func(*KeySetState) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var state KeySetState
	if payload := m.states[stateID(ctx)]; payload != nil {
		if err := json.Unmarshal(payload, &state); err != nil {
			return err
		}
	}
	if err := update(&state); err != nil {
		return err
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	m.states[stateID(ctx)] = payload
	return nil
}

func (m *MemoryKeyStateStore) Scopes(ctx context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return setScopes(ctx, m.states), nil
}

func (m *MemoryKeyStateStore) Healthy(context.Context) bool { return true }
