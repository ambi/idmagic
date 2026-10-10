package db_memory

import (
	"context"
	"sync"
	"time"

	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
)

// =====================================================================
// WebAuthnSessionStore (Authentication) — wi-26 /
// WebAuthn ceremony の challenge を短命に保持する。Take は一度きりの消費。
// =====================================================================

type webAuthnSessionEntry struct {
	data      gowebauthn.SessionData
	expiresAt time.Time
}

type WebAuthnSessionStore struct {
	mu       sync.Mutex
	sessions map[string]webAuthnSessionEntry
}

func NewWebAuthnSessionStore() *WebAuthnSessionStore {
	return &WebAuthnSessionStore{sessions: map[string]webAuthnSessionEntry{}}
}

func (s *WebAuthnSessionStore) Save(
	ctx context.Context,
	key string,
	data gowebauthn.SessionData,
	expiresAt time.Time,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionKey(ctx, key)] = webAuthnSessionEntry{data: data, expiresAt: expiresAt}
	return nil
}

func (s *WebAuthnSessionStore) Take(
	ctx context.Context,
	key string,
) (*gowebauthn.SessionData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.sessions[sessionKey(ctx, key)]
	if !ok {
		return nil, nil
	}
	delete(s.sessions, sessionKey(ctx, key))
	if !time.Now().Before(entry.expiresAt) {
		return nil, nil
	}
	data := entry.data
	return &data, nil
}

func sessionKey(ctx context.Context, key string) string {
	return tenantports.TenantID(ctx) + "\x00" + key
}
