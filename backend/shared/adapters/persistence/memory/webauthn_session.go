package memory

import (
	"context"
	"sync"
	"time"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
)

// =====================================================================
// WebAuthnSessionStore (Authentication) — wi-26 / ADR-087
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
	_ context.Context,
	key string,
	data gowebauthn.SessionData,
	expiresAt time.Time,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[key] = webAuthnSessionEntry{data: data, expiresAt: expiresAt}
	return nil
}

func (s *WebAuthnSessionStore) Take(
	_ context.Context,
	key string,
) (*gowebauthn.SessionData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.sessions[key]
	if !ok {
		return nil, nil
	}
	delete(s.sessions, key)
	if !time.Now().Before(entry.expiresAt) {
		return nil, nil
	}
	data := entry.data
	return &data, nil
}
