// Package valkey は OAuth2 token/grant 用の Valkey 永続化アダプタ。
package db_valkey

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedvalkey "github.com/ambi/idmagic/backend/shared/storage/db_valkey"
	"github.com/ambi/idmagic/backend/tenancy"

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

type AuthorizationRequestStore struct{ Client *goredis.Client }

func (s *AuthorizationRequestStore) Save(ctx context.Context, req *domain.AuthorizationRequest) error {
	req.TenantID = tenancy.TenantID(ctx)
	return setJSON(ctx, s.Client, tenantKey(ctx, "auth_request:"+req.ID), req, ttlUntil(req.ExpiresAt))
}

func (s *AuthorizationRequestStore) Find(ctx context.Context, id string) (*domain.AuthorizationRequest, error) {
	var req domain.AuthorizationRequest
	if err := getJSON(ctx, s.Client, tenantKey(ctx, "auth_request:"+id), &req); err != nil {
		return nil, err
	}
	if req.ID == "" {
		return nil, nil
	}
	return &req, nil
}

func (s *AuthorizationRequestStore) UpdateState(ctx context.Context, id string, state spec.AuthorizationCodeFlowState) error {
	return s.update(ctx, id, func(req *domain.AuthorizationRequest) error {
		next, err := spec.TransitionAuthorizationCodeFlow(req.State, eventForTargetState(state))
		if err != nil {
			return err
		}
		req.State = next
		return nil
	})
}

func (s *AuthorizationRequestStore) AttachAuthentication(
	ctx context.Context,
	id, sub string,
	authTime int64,
	amr []string,
	acr, sid string,
) error {
	return s.update(ctx, id, func(req *domain.AuthorizationRequest) error {
		req.UserID, req.AuthTime = &sub, &authTime
		req.AMR, req.ACR = amr, &acr
		if sid != "" {
			req.Sid = &sid
		}
		return nil
	})
}

func (s *AuthorizationRequestStore) update(
	ctx context.Context,
	id string,
	change func(*domain.AuthorizationRequest) error,
) error {
	key := tenantKey(ctx, "auth_request:"+id)
	return s.Client.Watch(ctx, func(tx *goredis.Tx) error {
		var req domain.AuthorizationRequest
		if err := getJSON(ctx, tx, key, &req); err != nil {
			return err
		}
		if req.ID == "" {
			return fmt.Errorf("authorization request %q not found", id)
		}
		if err := change(&req); err != nil {
			return err
		}
		payload, err := json.Marshal(&req)
		if err != nil {
			return err
		}
		ttl, err := tx.TTL(ctx, key).Result()
		if err != nil {
			return err
		}
		_, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
			pipe.Set(ctx, key, payload, ttl)
			return nil
		})
		return err
	}, key)
}

func eventForTargetState(to spec.AuthorizationCodeFlowState) spec.AuthorizationCodeFlowEvent {
	switch to {
	case spec.AuthFlowAuthenticationPending:
		return spec.EventStartAuthentication
	case spec.AuthFlowAuthenticated:
		return spec.EventAuthenticateUser
	case spec.AuthFlowConsentPending:
		return spec.EventRequestConsent
	case spec.AuthFlowConsented:
		return spec.EventGrantConsent
	case spec.AuthFlowCodeIssued:
		return spec.EventIssueCode
	case spec.AuthFlowExchanged:
		return spec.EventRedeemCode
	case spec.AuthFlowRejected:
		return spec.EventRejectAuthorization
	case spec.AuthFlowExpired:
		return spec.EventExpireRequest
	default:
		return "unknown"
	}
}

type AuthorizationCodeStore struct{ Client *goredis.Client }

func (s *AuthorizationCodeStore) Save(ctx context.Context, rec *domain.AuthorizationCodeRecord) error {
	rec.TenantID = tenancy.TenantID(ctx)
	return setJSON(ctx, s.Client, tenantKey(ctx, "auth_code:"+rec.Code), rec, ttlUntil(rec.ExpiresAt))
}

func (s *AuthorizationCodeStore) Find(ctx context.Context, code string) (*domain.AuthorizationCodeRecord, error) {
	var rec domain.AuthorizationCodeRecord
	if err := getJSON(ctx, s.Client, tenantKey(ctx, "auth_code:"+code), &rec); err != nil {
		return nil, err
	}
	if rec.Code == "" {
		return nil, nil
	}
	return &rec, nil
}

var redeemCode = goredis.NewScript(`
local payload = redis.call('GET', KEYS[1])
if not payload then return false end
local rec = cjson.decode(payload)
if rec.state ~= 'issued' then return false end
rec.state = 'redeemed'
rec.redeemed_at = ARGV[1]
redis.call('SET', KEYS[1], cjson.encode(rec), 'KEEPTTL')
return cjson.encode(rec)
`)

func (s *AuthorizationCodeStore) Redeem(ctx context.Context, code string, now time.Time) (*domain.AuthorizationCodeRecord, error) {
	result, err := redeemCode.Run(ctx, s.Client, []string{tenantKey(ctx, "auth_code:"+code)}, now.UTC().Format(time.RFC3339Nano)).Text()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rec domain.AuthorizationCodeRecord
	if err := json.Unmarshal([]byte(result), &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *AuthorizationCodeStore) LinkFamily(ctx context.Context, code, familyID string) error {
	key := tenantKey(ctx, "auth_code:"+code)
	return s.Client.Watch(ctx, func(tx *goredis.Tx) error {
		var rec domain.AuthorizationCodeRecord
		if err := getJSON(ctx, tx, key, &rec); err != nil {
			return err
		}
		if rec.Code == "" {
			return errors.New("authorization code not found")
		}
		rec.IssuedFamilyID = &familyID
		payload, _ := json.Marshal(&rec)
		ttl, err := tx.TTL(ctx, key).Result()
		if err != nil {
			return err
		}
		_, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
			pipe.Set(ctx, key, payload, ttl)
			return nil
		})
		return err
	}, key)
}

type PARStore struct{ Client *goredis.Client }

func (s *PARStore) Save(ctx context.Context, rec *domain.PARRecord) error {
	rec.TenantID = tenancy.TenantID(ctx)
	return setJSON(ctx, s.Client, tenantKey(ctx, "par:"+rec.RequestURI), rec, ttlUntil(rec.ExpiresAt))
}

func (s *PARStore) Find(ctx context.Context, uri string) (*domain.PARRecord, error) {
	var rec domain.PARRecord
	if err := getJSON(ctx, s.Client, tenantKey(ctx, "par:"+uri), &rec); err != nil {
		return nil, err
	}
	if rec.RequestURI == "" {
		return nil, nil
	}
	return &rec, nil
}

var consumePAR = goredis.NewScript(`
local payload = redis.call('GET', KEYS[1])
if not payload then return false end
local rec = cjson.decode(payload)
if rec.used then return false end
rec.used = true
redis.call('SET', KEYS[1], cjson.encode(rec), 'KEEPTTL')
return cjson.encode(rec)
`)

func (s *PARStore) Consume(ctx context.Context, uri string) (*domain.PARRecord, error) {
	result, err := consumePAR.Run(ctx, s.Client, []string{tenantKey(ctx, "par:"+uri)}).Text()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rec domain.PARRecord
	if err := json.Unmarshal([]byte(result), &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

type DeviceCodeStore struct{ Client *goredis.Client }

func (s *DeviceCodeStore) Save(ctx context.Context, rec *domain.DeviceAuthorization) error {
	rec.TenantID = tenancy.TenantID(ctx)
	ttl := ttlUntil(rec.ExpiresAt)
	pipe := s.Client.TxPipeline()
	payload, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	pipe.Set(ctx, tenantKey(ctx, "device:"+rec.DeviceCodeHash), payload, ttl)
	pipe.Set(ctx, tenantKey(ctx, "device:user_code:"+rec.UserCode), rec.DeviceCodeHash, ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *DeviceCodeStore) FindByDeviceCodeHash(ctx context.Context, hash string) (*domain.DeviceAuthorization, error) {
	var rec domain.DeviceAuthorization
	if err := getJSON(ctx, s.Client, tenantKey(ctx, "device:"+hash), &rec); err != nil {
		return nil, err
	}
	if rec.DeviceCodeHash == "" {
		return nil, nil
	}
	return &rec, nil
}

func (s *DeviceCodeStore) FindByUserCode(ctx context.Context, code string) (*domain.DeviceAuthorization, error) {
	hash, err := s.Client.Get(ctx, tenantKey(ctx, "device:user_code:"+code)).Result()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.FindByDeviceCodeHash(ctx, hash)
}

func (s *DeviceCodeStore) Update(ctx context.Context, rec *domain.DeviceAuthorization) error {
	key := tenantKey(ctx, "device:"+rec.DeviceCodeHash)
	ttl, err := s.Client.TTL(ctx, key).Result()
	if err != nil {
		return err
	}
	return setJSON(ctx, s.Client, key, rec, ttl)
}

var exchangeDevice = goredis.NewScript(`
local payload = redis.call('GET', KEYS[1])
if not payload then return false end
local rec = cjson.decode(payload)
if rec.state ~= 'approved' then return false end
rec.state = 'exchanged'
redis.call('SET', KEYS[1], cjson.encode(rec), 'KEEPTTL')
return cjson.encode(rec)
`)

func (s *DeviceCodeStore) DeleteAllForSub(ctx context.Context, sub string) error {
	pattern := tenantKey(ctx, "device:*")
	userCodePrefix := tenantKey(ctx, "device:user_code:")
	iter := s.Client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		if strings.HasPrefix(key, userCodePrefix) {
			continue
		}
		var rec domain.DeviceAuthorization
		if err := getJSON(ctx, s.Client, key, &rec); err != nil {
			return err
		}
		if rec.UserID == nil || *rec.UserID != sub {
			continue
		}
		pipe := s.Client.TxPipeline()
		pipe.Del(ctx, key)
		pipe.Del(ctx, tenantKey(ctx, "device:user_code:"+rec.UserCode))
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
	}
	return iter.Err()
}

func (s *DeviceCodeStore) Exchange(ctx context.Context, hash string) (*domain.DeviceAuthorization, error) {
	result, err := exchangeDevice.Run(ctx, s.Client, []string{tenantKey(ctx, "device:"+hash)}).Text()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rec domain.DeviceAuthorization
	if err := json.Unmarshal([]byte(result), &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

type ReplayStore struct {
	Client *goredis.Client
	Prefix string
}

func (s *ReplayStore) RecordIfNew(ctx context.Context, jti string, seconds int, _ time.Time) (bool, error) {
	return s.Client.SetNX(ctx, tenantKey(ctx, s.Prefix+jti), "1", time.Duration(seconds)*time.Second).Result()
}

type AccessTokenDenylist struct{ Client *goredis.Client }

func (d *AccessTokenDenylist) Add(ctx context.Context, jti string, expiresAt time.Time) error {
	return d.Client.Set(ctx, tenantKey(ctx, "token_denylist:"+jti), "1", ttlUntil(expiresAt)).Err()
}

func (d *AccessTokenDenylist) IsRevoked(ctx context.Context, jti string) (bool, error) {
	count, err := d.Client.Exists(ctx, tenantKey(ctx, "token_denylist:"+jti)).Result()
	return count > 0, err
}
