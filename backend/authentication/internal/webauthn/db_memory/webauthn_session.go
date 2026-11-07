package db_memory

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// =====================================================================
// WebAuthnSessionStore (Authentication) — wi-26 /
// WebAuthn ceremony の challenge を短命に保持する。Take は一度きりの消費。
// =====================================================================

type webAuthnSessionEntry struct {
	payload   []byte
	expiresAt time.Time
}

type WebAuthnSessionStore struct {
	mu       sync.Mutex
	sessions map[string]webAuthnSessionEntry
}

func NewWebAuthnSessionStore() *WebAuthnSessionStore {
	return &WebAuthnSessionStore{sessions: map[string]webAuthnSessionEntry{}}
}

// Save は PostgreSQL の実装と同じく内容を写して持ち、呼び出し元が後で書き換えても保存した値は変わらない。
func (s *WebAuthnSessionStore) Save(
	ctx context.Context,
	key string,
	session ports.WebAuthnSession,
	expiresAt time.Time,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionKey(ctx, key)] = webAuthnSessionEntry{payload: bytes.Clone(session.Payload), expiresAt: expiresAt}
	return nil
}

func (s *WebAuthnSessionStore) Take(
	ctx context.Context,
	key string,
) (*ports.WebAuthnSession, error) {
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
	return &ports.WebAuthnSession{Payload: entry.payload}, nil
}

func sessionKey(ctx context.Context, key string) string {
	return tenantports.TenantID(ctx) + "\x00" + key
}
