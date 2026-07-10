package crypto

import (
	"context"
	"crypto/rand"
	"sync"

	"github.com/ambi/idmagic/backend/tenancy"
)

// TenantSaltBytes は相関 salt のバイト長 (256 bit)。
const TenantSaltBytes = 32

// InMemoryTenantSaltStore は dev/test 用の tenant-aware な相関 salt ストア (wi-145 / ADR-046)。
// tenant scope は ctx (tenancy.TenantID) から解決し、初回取得時に 32 byte を生成する。
// tenancy.TenantID は未設定 ctx で DefaultTenantID を返すため、空テナントでも安全に動く。
type InMemoryTenantSaltStore struct {
	mu       sync.Mutex
	byTenant map[string][]byte
}

// NewInMemoryTenantSaltStore は空のストアを返す。salt は GetSalt の初回呼び出しで遅延生成する。
func NewInMemoryTenantSaltStore() *InMemoryTenantSaltStore {
	return &InMemoryTenantSaltStore{byTenant: map[string][]byte{}}
}

// GetSalt は ctx のテナントの salt を返す。未生成なら生成して保持する。
func (s *InMemoryTenantSaltStore) GetSalt(ctx context.Context) ([]byte, error) {
	tenantID := tenancy.TenantID(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	if salt, ok := s.byTenant[tenantID]; ok {
		return salt, nil
	}
	salt := make([]byte, TenantSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	s.byTenant[tenantID] = salt
	return salt, nil
}
