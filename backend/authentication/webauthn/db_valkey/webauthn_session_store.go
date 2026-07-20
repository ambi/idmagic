// Package valkey は webauthn feature 用の Valkey 永続化アダプタ。
package db_valkey

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	sharedvalkey "github.com/ambi/idmagic/backend/shared/storage/db_valkey"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	goredis "github.com/redis/go-redis/v9"
)

type WebAuthnSessionStore struct{ Client *goredis.Client }

func (s *WebAuthnSessionStore) Save(
	ctx context.Context,
	key string,
	data gowebauthn.SessionData,
	expiresAt time.Time,
) error {
	return sharedvalkey.SetJSON(ctx, s.Client, sharedvalkey.TenantKey(ctx, "webauthn_session:"+key), data, sharedvalkey.TTLUntil(expiresAt))
}

func (s *WebAuthnSessionStore) Take(
	ctx context.Context,
	key string,
) (*gowebauthn.SessionData, error) {
	payload, err := s.Client.GetDel(ctx, sharedvalkey.TenantKey(ctx, "webauthn_session:"+key)).Bytes()
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
