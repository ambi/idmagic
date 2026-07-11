package memory

import (
	"context"
	"sync"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedmem "github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
)

// RefreshTokenStore は OAuth2 token/grant の memory store を context 側から公開する。
// 実装本体は shared memory の同期プリミティブを再利用する。
type RefreshTokenStore = sharedmem.RefreshTokenStore

func NewRefreshTokenStore() *RefreshTokenStore { return sharedmem.NewRefreshTokenStore() }

// OAuth2ClientRepository (OAuth2)
type OAuth2ClientRepository struct {
	mu      sync.RWMutex
	clients map[string]*domain.OAuth2Client
}

func NewClientRepository() *OAuth2ClientRepository {
	return &OAuth2ClientRepository{clients: map[string]*domain.OAuth2Client{}}
}

func (r *OAuth2ClientRepository) Seed(c *domain.OAuth2Client) {
	_ = r.Save(context.Background(), c)
}

func (r *OAuth2ClientRepository) FindByID(_ context.Context, tenantID, clientID string) (*domain.OAuth2Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clients[sharedmem.TenantKey(tenantID, clientID)], nil
}

func (r *OAuth2ClientRepository) Save(_ context.Context, c *domain.OAuth2Client) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	sharedmem.DefaultTenant(&c.TenantID)
	r.clients[sharedmem.TenantKey(c.TenantID, c.ClientID)] = c
	return nil
}

func (r *OAuth2ClientRepository) Delete(_ context.Context, tenantID, clientID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, sharedmem.TenantKey(tenantID, clientID))
	return nil
}

func (r *OAuth2ClientRepository) FindAll(_ context.Context, tenantID string) ([]*domain.OAuth2Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.OAuth2Client, 0, len(r.clients))
	for _, c := range r.clients {
		if c.TenantID == tenantID {
			out = append(out, c)
		}
	}
	return out, nil
}
