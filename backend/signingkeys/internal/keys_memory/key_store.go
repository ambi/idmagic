// Package crypto: 鍵ストアと JWT 署名 (PS256)。
//
// ローカル開発用 in-memory 鍵ストア。本番では KMS / HSM / Vault を使う想定。
// JWK サムプリント (RFC 7638) を kid として使用する。
package keys_memory

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	keysJOSE "github.com/ambi/idmagic/backend/signingkeys/internal/keys_jose"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"

	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// tenantKeys は 1 テナント分の署名鍵集合と active kid を保持する。
type tenantKeys struct {
	keys   []*signingdomain.SigningKey
	active string
}

// InMemoryKeyStore は dev/test 用の tenant-aware な in-memory 鍵ストア。
// tenant scope は ctx (tenantports.TenantID) から解決する。
type InMemoryKeyStore struct {
	mu       sync.RWMutex
	byTenant map[string]*tenantKeys
}

func NewInMemoryKeyStore() (*InMemoryKeyStore, error) {
	return &InMemoryKeyStore{byTenant: map[string]*tenantKeys{}}, nil
}

func (s *InMemoryKeyStore) GetActiveKey(ctx context.Context) (*signingdomain.SigningKey, error) {
	tenantID := tenantports.TenantID(ctx)
	usage := signingports.KeyUsage(ctx)
	scopeID := signingports.KeyScope(ctx)
	s.mu.RLock()
	if tk := s.byTenant[keySetID(tenantID, usage, scopeID)]; tk != nil {
		for _, k := range tk.keys {
			if k.Kid == tk.active {
				s.mu.RUnlock()
				return k, nil
			}
		}
	}
	s.mu.RUnlock()
	// まだ鍵の無いテナントは初回に遅延生成する。
	result, err := s.createKey(tenantID, usage, scopeID, time.Now().UTC(), 7*24*time.Hour, false, true)
	return result.Next, err
}

func (s *InMemoryKeyStore) GetAllKeys(ctx context.Context) ([]*signingdomain.SigningKey, error) {
	tenantID := tenantports.TenantID(ctx)
	usage := signingports.KeyUsage(ctx)
	scopeID := signingports.KeyScope(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	tk := s.byTenant[keySetID(tenantID, usage, scopeID)]
	if tk == nil {
		return []*signingdomain.SigningKey{}, nil
	}
	out := make([]*signingdomain.SigningKey, len(tk.keys))
	copy(out, tk.keys)
	return out, nil
}

func (s *InMemoryKeyStore) ListPublicKeys(ctx context.Context, now time.Time) ([]*signingdomain.SigningKey, error) {
	keys, err := s.GetAllKeys(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*signingdomain.SigningKey, 0, len(keys))
	for _, key := range keys {
		if key.ArchivedAt == nil && (key.ExpiresAt == nil || key.ExpiresAt.After(now)) {
			out = append(out, key)
		}
	}
	return out, nil
}

func (s *InMemoryKeyStore) FindByKID(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	tenantID := tenantports.TenantID(ctx)
	usage := signingports.KeyUsage(ctx)
	scopeID := signingports.KeyScope(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	tk := s.byTenant[keySetID(tenantID, usage, scopeID)]
	if tk == nil {
		return nil, nil //nolint:nilnil // repository contract: a missing key is not an adapter failure.
	}
	for _, k := range tk.keys {
		if k.Kid == kid {
			return k, nil
		}
	}
	return nil, nil //nolint:nilnil // repository contract: a missing key is not an adapter failure.
}

func (s *InMemoryKeyStore) KeyScopes(ctx context.Context) ([]string, error) {
	prefix := keySetID(tenantports.TenantID(ctx), signingports.KeyUsage(ctx), "")
	s.mu.RLock()
	defer s.mu.RUnlock()
	scopes := []string{}
	for id, tk := range s.byTenant {
		if scopeID, ok := strings.CutPrefix(id, prefix); ok && len(tk.keys) > 0 {
			scopes = append(scopes, scopeID)
		}
	}
	return scopes, nil
}

func (s *InMemoryKeyStore) Rotate(ctx context.Context, now time.Time, grace time.Duration) (*signingdomain.SigningKey, error) {
	return s.rotateInternal(tenantports.TenantID(ctx), signingports.KeyUsage(ctx), signingports.KeyScope(ctx), now, grace)
}

func (s *InMemoryKeyStore) Disable(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	tenantID := tenantports.TenantID(ctx)
	usage := signingports.KeyUsage(ctx)
	scopeID := signingports.KeyScope(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	tk := s.byTenant[keySetID(tenantID, usage, scopeID)]
	if tk == nil {
		return nil, nil //nolint:nilnil // repository contract: a missing key is not an adapter failure.
	}
	remaining := tk.keys[:0]
	var disabled *signingdomain.SigningKey
	for _, k := range tk.keys {
		if k.Kid == kid {
			if tk.active == kid {
				return nil, signingdomain.ErrActiveSigningKeyCannotBeDisabled
			}
			snapshot := *k
			snapshot.Prepared = false
			disabled = &snapshot
			continue
		}
		remaining = append(remaining, k)
	}
	tk.keys = remaining
	return disabled, nil
}

func (s *InMemoryKeyStore) ArchiveExpired(ctx context.Context, before time.Time) ([]*signingdomain.SigningKey, error) {
	tenantID := tenantports.TenantID(ctx)
	usage := signingports.KeyUsage(ctx)
	scopeID := signingports.KeyScope(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	tk := s.byTenant[keySetID(tenantID, usage, scopeID)]
	if tk == nil {
		return nil, nil
	}
	archived := []*signingdomain.SigningKey{}
	for i, key := range tk.keys {
		if key.ArchivedAt != nil || key.ExpiresAt == nil || key.ExpiresAt.After(before) {
			continue
		}
		at := before.UTC()
		snapshot := *key
		snapshot.ArchivedAt = &at
		tk.keys[i] = &snapshot
		archived = append(archived, &snapshot)
	}
	return archived, nil
}

func (s *InMemoryKeyStore) Provider() signingdomain.KeyProvider {
	return signingdomain.KeyProviderLocal
}

func (s *InMemoryKeyStore) Healthy(_ context.Context) bool { return true }

func keySetID(tenantID string, usage signingdomain.KeyUsage, scopeID string) string {
	return tenantID + "\x00" + string(usage) + "\x00" + scopeID
}

func (s *InMemoryKeyStore) rotateInternal(tenantID string, usage signingdomain.KeyUsage, scopeID string, now time.Time, grace time.Duration) (*signingdomain.SigningKey, error) {
	result, err := s.createKey(tenantID, usage, scopeID, now, grace, false, false)
	return result.Next, err
}

func (s *InMemoryKeyStore) createKey(tenantID string, usage signingdomain.KeyUsage, scopeID string, now time.Time, grace time.Duration, prepare, initialize bool) (signingdomain.RotationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tk := s.byTenant[keySetID(tenantID, usage, scopeID)]
	if tk == nil {
		tk = &tenantKeys{}
		s.byTenant[keySetID(tenantID, usage, scopeID)] = tk
	}
	switch {
	case prepare:
		if previous := signingdomain.PreparedKey(tk.keys); previous != nil {
			return signingdomain.RotationResult{Next: previous, Previous: signingdomain.ActiveKey(tk.keys)}, nil
		}
	case initialize && signingdomain.ActiveKey(tk.keys) != nil:
		return signingdomain.RotationResult{Next: signingdomain.ActiveKey(tk.keys)}, nil
	case signingdomain.ActiveKey(tk.keys) != nil && (signingdomain.RequiresPrepublication(usage, scopeID) || signingdomain.PreparedKey(tk.keys) != nil):
		return signingdomain.RotationResult{}, signingdomain.ErrPrepublicationRequired
	}
	priv, jwk, _, kid, err := keysJOSE.GenerateRSAJWKPair()
	if err != nil {
		return signingdomain.RotationResult{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if grace < 0 {
		return signingdomain.RotationResult{}, errors.New("signing key grace period must not be negative")
	}
	key := &signingdomain.SigningKey{
		TenantID:   tenantID,
		Kid:        kid,
		Alg:        signingdomain.SigAlgPS256,
		Provider:   signingdomain.KeyProviderLocal,
		Usage:      usage,
		ScopeID:    scopeID,
		PrivateKey: priv,
		PublicKey:  &priv.PublicKey,
		PublicJWK:  jwk,
		Active:     !prepare,
		Prepared:   prepare,
		CreatedAt:  now,
	}
	if usage == signingdomain.KeyUsageXMLFederationSigning {
		key.CertificateDER, err = keysJOSE.NewFederationCertificate(tenantID, kid, priv, now)
		if err != nil {
			return signingdomain.RotationResult{}, err
		}
	}
	tk.keys = append(tk.keys, key)
	if prepare {
		return signingdomain.RotationResult{Next: key, Previous: signingdomain.ActiveKey(tk.keys), Changed: true}, nil
	}
	// 有効化の共通核を通し、旧鍵の猶予をこの時刻から始める。
	key.Active, key.Prepared = false, true
	result, err := signingdomain.ActivatePrepared(tk.keys, kid, now, grace)
	if err == nil {
		tk.active = kid
	}
	return result, err
}

func (s *InMemoryKeyStore) Prepare(ctx context.Context, now time.Time) (signingdomain.RotationResult, error) {
	if signingports.KeyUsage(ctx) != signingdomain.KeyUsageXMLFederationSigning {
		return signingdomain.RotationResult{}, signingdomain.ErrKeyNotPrepared
	}
	if _, err := s.GetActiveKey(ctx); err != nil {
		return signingdomain.RotationResult{}, err
	}
	return s.createKey(tenantports.TenantID(ctx), signingports.KeyUsage(ctx), signingports.KeyScope(ctx), now, 0, true, false)
}

func (s *InMemoryKeyStore) Activate(ctx context.Context, kid string, now time.Time, grace time.Duration) (signingdomain.RotationResult, error) {
	if signingports.KeyUsage(ctx) != signingdomain.KeyUsageXMLFederationSigning {
		return signingdomain.RotationResult{}, signingdomain.ErrKeyNotPrepared
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tk := s.byTenant[keySetID(tenantports.TenantID(ctx), signingports.KeyUsage(ctx), signingports.KeyScope(ctx))]
	if tk == nil {
		return signingdomain.RotationResult{}, signingdomain.ErrKeyNotPrepared
	}
	result, err := signingdomain.ActivatePrepared(tk.keys, kid, now, grace)
	if err == nil {
		tk.active = result.Next.Kid
	}
	return result, err
}
