package memory

import (
	"context"
	"sync"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
)

// =====================================================================
// OAuth2ClientRepository (OAuth2)
// =====================================================================

type OAuth2ClientRepository struct {
	mu      sync.RWMutex
	clients map[string]*oauthdomain.OAuth2Client
}

func NewClientRepository() *OAuth2ClientRepository {
	return &OAuth2ClientRepository{clients: map[string]*oauthdomain.OAuth2Client{}}
}

func (r *OAuth2ClientRepository) Seed(c *oauthdomain.OAuth2Client) {
	_ = r.Save(context.Background(), c)
}

func (r *OAuth2ClientRepository) FindByID(_ context.Context, tenantID, clientID string) (*oauthdomain.OAuth2Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clients[TenantKey(tenantID, clientID)], nil
}

func (r *OAuth2ClientRepository) Save(_ context.Context, c *oauthdomain.OAuth2Client) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	defaultTenant(&c.TenantID)
	r.clients[TenantKey(c.TenantID, c.ClientID)] = c
	return nil
}

func (r *OAuth2ClientRepository) Delete(_ context.Context, tenantID, clientID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, TenantKey(tenantID, clientID))
	return nil
}

func (r *OAuth2ClientRepository) FindAll(_ context.Context, tenantID string) ([]*oauthdomain.OAuth2Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*oauthdomain.OAuth2Client, 0, len(r.clients))
	for _, c := range r.clients {
		if c.TenantID == tenantID {
			out = append(out, c)
		}
	}
	return out, nil
}
