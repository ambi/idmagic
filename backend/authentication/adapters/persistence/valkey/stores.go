// Package valkey は Authentication 用の Valkey 永続化アダプタ。
package valkey

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	sharedvalkey "github.com/ambi/idmagic/backend/shared/adapters/persistence/valkey"
	"github.com/ambi/idmagic/backend/tenancy"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	goredis "github.com/redis/go-redis/v9"
)

func setJSON(ctx context.Context, client goredis.Cmdable, key string, value any, ttl time.Duration) error {
	return sharedvalkey.SetJSON(ctx, client, key, value, ttl)
}

func getJSON(ctx context.Context, client goredis.Cmdable, key string, out any) error {
	return sharedvalkey.GetJSON(ctx, client, key, out)
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

type SessionStore struct{ Client *goredis.Client }

func (s *SessionStore) Save(ctx context.Context, session *authdomain.LoginSession) error {
	session.TenantID = tenancy.TenantID(ctx)
	return setJSON(ctx, s.Client, tenantKey(ctx, "session:"+session.ID), session, ttlUntil(session.ExpiresAt))
}

func (s *SessionStore) Find(ctx context.Context, id string) (*authdomain.LoginSession, error) {
	var session authdomain.LoginSession
	if err := getJSON(ctx, s.Client, tenantKey(ctx, "session:"+id), &session); err != nil {
		return nil, err
	}
	if session.ID == "" {
		return nil, nil
	}
	return &session, nil
}

func (s *SessionStore) Delete(ctx context.Context, id string) error {
	return s.Client.Del(ctx, tenantKey(ctx, "session:"+id)).Err()
}

func (s *SessionStore) ListBySub(ctx context.Context, sub string) ([]*authdomain.LoginSession, error) {
	pattern := tenantKey(ctx, "session:*")
	iter := s.Client.Scan(ctx, 0, pattern, 100).Iterator()
	out := []*authdomain.LoginSession{}
	for iter.Next(ctx) {
		var session authdomain.LoginSession
		if err := getJSON(ctx, s.Client, iter.Val(), &session); err != nil {
			return nil, err
		}
		if session.ID == "" {
			continue
		}
		if session.UserID == sub && !session.AuthenticationPending {
			copied := session
			out = append(out, &copied)
		}
	}
	return out, iter.Err()
}

func (s *SessionStore) DeleteAllForSub(ctx context.Context, sub string) error {
	pattern := tenantKey(ctx, "session:*")
	iter := s.Client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		var session authdomain.LoginSession
		if err := getJSON(ctx, s.Client, key, &session); err != nil {
			return err
		}
		if session.UserID == sub {
			if err := s.Client.Del(ctx, key).Err(); err != nil {
				return err
			}
		}
	}
	return iter.Err()
}
