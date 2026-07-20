// Package crypto: 鍵ストアと JWT 署名 (PS256)。
//
// ローカル開発用 in-memory 鍵ストア。本番では KMS / HSM / Vault を使う想定。
// JWK サムプリント (RFC 7638) を kid として使用する。
package keys_memory

import (
	"context"
	"errors"
	"sync"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	keysJOSE "github.com/ambi/idmagic/backend/signingkeys/keys_jose"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/tenancy"
)

// tenantKeys は 1 テナント分の署名鍵集合と active kid を保持する。
type tenantKeys struct {
	keys   []*signingdomain.SigningKey
	active string
}

// InMemoryKeyStore は dev/test 用の tenant-aware な in-memory 鍵ストア。
// tenant scope は ctx (tenancy.TenantID) から解決する。
type InMemoryKeyStore struct {
	mu       sync.RWMutex
	byTenant map[string]*tenantKeys
}

func NewInMemoryKeyStore() (*InMemoryKeyStore, error) {
	ks := &InMemoryKeyStore{byTenant: map[string]*tenantKeys{}}
	// default テナントの鍵を先に作る (後方互換)。
	if _, err := ks.rotateInternal(tenancydomain.DefaultTenantID, time.Now().UTC(), 7*24*time.Hour); err != nil {
		return nil, err
	}
	return ks, nil
}

func (s *InMemoryKeyStore) GetActiveKey(ctx context.Context) (*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	s.mu.RLock()
	if tk := s.byTenant[tenantID]; tk != nil {
		for _, k := range tk.keys {
			if k.Kid == tk.active {
				s.mu.RUnlock()
				return k, nil
			}
		}
	}
	s.mu.RUnlock()
	// まだ鍵の無いテナントは初回に遅延生成する。
	return s.rotateInternal(tenantID, time.Now().UTC(), 7*24*time.Hour)
}

func (s *InMemoryKeyStore) GetAllKeys(ctx context.Context) ([]*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	tk := s.byTenant[tenantID]
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
	tenantID := tenancy.TenantID(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	tk := s.byTenant[tenantID]
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

func (s *InMemoryKeyStore) Rotate(ctx context.Context, now time.Time, grace time.Duration) (*signingdomain.SigningKey, error) {
	return s.rotateInternal(tenancy.TenantID(ctx), now, grace)
}

func (s *InMemoryKeyStore) Disable(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	tk := s.byTenant[tenantID]
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
			disabled = k
			continue
		}
		remaining = append(remaining, k)
	}
	tk.keys = remaining
	return disabled, nil
}

func (s *InMemoryKeyStore) ArchiveExpired(ctx context.Context, before time.Time) ([]*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	tk := s.byTenant[tenantID]
	if tk == nil {
		return nil, nil
	}
	archived := []*signingdomain.SigningKey{}
	for _, key := range tk.keys {
		if key.ArchivedAt == nil && key.ExpiresAt != nil && !key.ExpiresAt.After(before) {
			at := before.UTC()
			key.ArchivedAt = &at
			archived = append(archived, key)
		}
	}
	return archived, nil
}

func (s *InMemoryKeyStore) Provider() signingdomain.KeyProvider {
	return signingdomain.KeyProviderLocal
}

func (s *InMemoryKeyStore) Healthy(_ context.Context) bool { return true }

func (s *InMemoryKeyStore) rotateInternal(tenantID string, now time.Time, grace time.Duration) (*signingdomain.SigningKey, error) {
	priv, jwk, _, kid, err := keysJOSE.GenerateRSAJWKPair()
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	tk := s.byTenant[tenantID]
	if tk == nil {
		tk = &tenantKeys{}
		s.byTenant[tenantID] = tk
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if grace < 0 {
		return nil, errors.New("signing key grace period must not be negative")
	}
	for _, k := range tk.keys {
		k.Active = false
		k.RetiredAt = &now
		expires := now.Add(grace)
		k.ExpiresAt = &expires
	}
	key := &signingdomain.SigningKey{
		TenantID:   tenantID,
		Kid:        kid,
		Alg:        signingdomain.SigAlgPS256,
		Provider:   signingdomain.KeyProviderLocal,
		Usage:      signingdomain.KeyUsageSigning,
		PrivateKey: priv,
		PublicKey:  &priv.PublicKey,
		PublicJWK:  jwk,
		Active:     true,
		CreatedAt:  now,
	}
	tk.keys = append(tk.keys, key)
	tk.active = kid
	return key, nil
}
