// Package valkey は Authentication 用の Valkey 永続化アダプタ。
package valkey

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	sharedvalkey "github.com/ambi/idmagic/backend/shared/adapters/persistence/valkey"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	goredis "github.com/redis/go-redis/v9"
)

func setJSON(ctx context.Context, client goredis.Cmdable, key string, value any, ttl time.Duration) error {
	return sharedvalkey.SetJSON(ctx, client, key, value, ttl)
}

func ttlUntil(expiresAt time.Time) time.Duration { return sharedvalkey.TTLUntil(expiresAt) }

func tenantKey(ctx context.Context, suffix string) string { return sharedvalkey.TenantKey(ctx, suffix) }

type WebAuthnSessionStore struct{ Client *goredis.Client }

func (s *WebAuthnSessionStore) Save(
	ctx context.Context,
	key string,
	data gowebauthn.SessionData,
	expiresAt time.Time,
) error {
	return setJSON(ctx, s.Client, tenantKey(ctx, "webauthn_session:"+key), data, ttlUntil(expiresAt))
}

func (s *WebAuthnSessionStore) Take(
	ctx context.Context,
	key string,
) (*gowebauthn.SessionData, error) {
	payload, err := s.Client.GetDel(ctx, tenantKey(ctx, "webauthn_session:"+key)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var data gowebauthn.SessionData
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// LoginSession は PostgreSQL を単一正本として永続化する (wi-253 / ADR-126)。Valkey は
// login 完了後の active session を保存しない。SessionStore はここでは提供しない。
