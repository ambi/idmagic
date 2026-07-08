package valkey

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ambi/idmagic/internal/shared/resilience"
	"github.com/ambi/idmagic/internal/shared/spec"
	"github.com/ambi/idmagic/internal/tenancy"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	goredis "github.com/redis/go-redis/v9"
)

// ValkeyConfig は Valkey 接続とレジリエンスの設定を集約する。
type ValkeyConfig struct {
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	QueryTimeout time.Duration // Valkey 操作時のコンテキストタイムアウト
}

func Open(ctx context.Context, rawURL string, cfg ValkeyConfig, cb *resilience.CircuitBreaker) (*goredis.Client, error) {
	if after, ok := strings.CutPrefix(rawURL, "valkey://"); ok {
		rawURL = "redis://" + after
	}
	options, err := goredis.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}

	options.DialTimeout = cfg.DialTimeout
	options.ReadTimeout = cfg.ReadTimeout
	options.WriteTimeout = cfg.WriteTimeout

	client := goredis.NewClient(options)

	// サーキットブレイカーとタイムアウトをフック
	if cb != nil {
		client.AddHook(&resilienceHook{
			cb:      cb,
			timeout: cfg.QueryTimeout,
		})
	}

	// Exponential Backoff を用いた初期接続（Ping）のリトライ
	err = resilience.RetryWithBackoff(ctx, func() error {
		pingCtx := ctx
		var cancel context.CancelFunc
		if cfg.DialTimeout > 0 {
			pingCtx, cancel = context.WithTimeout(ctx, cfg.DialTimeout)
			defer cancel()
		}
		return client.Ping(pingCtx).Err()
	})
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

type resilienceHook struct {
	cb      *resilience.CircuitBreaker
	timeout time.Duration
}

func (h *resilienceHook) DialHook(next goredis.DialHook) goredis.DialHook {
	return next
}

func (h *resilienceHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		qctx := ctx
		var cancel context.CancelFunc
		if h.timeout > 0 {
			qctx, cancel = context.WithTimeout(ctx, h.timeout)
			defer cancel()
		}

		if h.cb != nil {
			return h.cb.Execute(func() error { //nolint:contextcheck // CB state machine does not rely on request context
				return next(qctx, cmd)
			})
		}
		return next(qctx, cmd)
	}
}

func (h *resilienceHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []goredis.Cmder) error {
		qctx := ctx
		var cancel context.CancelFunc
		if h.timeout > 0 {
			qctx, cancel = context.WithTimeout(ctx, h.timeout)
			defer cancel()
		}

		if h.cb != nil {
			return h.cb.Execute(func() error { //nolint:contextcheck // CB state machine does not rely on request context
				return next(qctx, cmds)
			})
		}
		return next(qctx, cmds)
	}
}

func setJSON(ctx context.Context, client goredis.Cmdable, key string, value any, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return client.Set(ctx, key, payload, ttl).Err()
}

func getJSON(ctx context.Context, client goredis.Cmdable, key string, out any) error {
	payload, err := client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, out)
}

func ttlUntil(expiresAt time.Time) time.Duration {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return time.Millisecond
	}
	return ttl
}

func tenantKey(ctx context.Context, suffix string) string {
	return "tenant:" + tenancy.TenantID(ctx) + ":" + suffix
}

// WebAuthnSessionStore は WebAuthn ceremony の challenge を短命に保持する (wi-26 / ADR-087)。
// Take は GetDel で取得と同時に削除し、challenge の再利用 (replay) を防ぐ。
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

type AuthorizationRequestStore struct{ Client *goredis.Client }

func (s *AuthorizationRequestStore) Save(ctx context.Context, req *spec.AuthorizationRequest) error {
	req.TenantID = tenancy.TenantID(ctx)
	return setJSON(ctx, s.Client, tenantKey(ctx, "auth_request:"+req.ID), req, ttlUntil(req.ExpiresAt))
}

func (s *AuthorizationRequestStore) Find(ctx context.Context, id string) (*spec.AuthorizationRequest, error) {
	var req spec.AuthorizationRequest
	if err := getJSON(ctx, s.Client, tenantKey(ctx, "auth_request:"+id), &req); err != nil {
		return nil, err
	}
	if req.ID == "" {
		return nil, nil
	}
	return &req, nil
}

func (s *AuthorizationRequestStore) UpdateState(ctx context.Context, id string, state spec.AuthorizationCodeFlowState) error {
	return s.update(ctx, id, func(req *spec.AuthorizationRequest) error {
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
	acr string,
) error {
	return s.update(ctx, id, func(req *spec.AuthorizationRequest) error {
		req.UserID, req.AuthTime = &sub, &authTime
		req.AMR, req.ACR = amr, &acr
		return nil
	})
}

func (s *AuthorizationRequestStore) update(
	ctx context.Context,
	id string,
	change func(*spec.AuthorizationRequest) error,
) error {
	key := tenantKey(ctx, "auth_request:"+id)
	return s.Client.Watch(ctx, func(tx *goredis.Tx) error {
		var req spec.AuthorizationRequest
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

func (s *AuthorizationCodeStore) Save(ctx context.Context, rec *spec.AuthorizationCodeRecord) error {
	rec.TenantID = tenancy.TenantID(ctx)
	return setJSON(ctx, s.Client, tenantKey(ctx, "auth_code:"+rec.Code), rec, ttlUntil(rec.ExpiresAt))
}

func (s *AuthorizationCodeStore) Find(ctx context.Context, code string) (*spec.AuthorizationCodeRecord, error) {
	var rec spec.AuthorizationCodeRecord
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

func (s *AuthorizationCodeStore) Redeem(ctx context.Context, code string, now time.Time) (*spec.AuthorizationCodeRecord, error) {
	result, err := redeemCode.Run(ctx, s.Client, []string{tenantKey(ctx, "auth_code:"+code)}, now.UTC().Format(time.RFC3339Nano)).Text()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rec spec.AuthorizationCodeRecord
	if err := json.Unmarshal([]byte(result), &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *AuthorizationCodeStore) LinkFamily(ctx context.Context, code, familyID string) error {
	key := tenantKey(ctx, "auth_code:"+code)
	return s.Client.Watch(ctx, func(tx *goredis.Tx) error {
		var rec spec.AuthorizationCodeRecord
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

func (s *PARStore) Save(ctx context.Context, rec *spec.PARRecord) error {
	rec.TenantID = tenancy.TenantID(ctx)
	return setJSON(ctx, s.Client, tenantKey(ctx, "par:"+rec.RequestURI), rec, ttlUntil(rec.ExpiresAt))
}

func (s *PARStore) Find(ctx context.Context, uri string) (*spec.PARRecord, error) {
	var rec spec.PARRecord
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

func (s *PARStore) Consume(ctx context.Context, uri string) (*spec.PARRecord, error) {
	result, err := consumePAR.Run(ctx, s.Client, []string{tenantKey(ctx, "par:"+uri)}).Text()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rec spec.PARRecord
	if err := json.Unmarshal([]byte(result), &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

type DeviceCodeStore struct{ Client *goredis.Client }

func (s *DeviceCodeStore) Save(ctx context.Context, rec *spec.DeviceAuthorization) error {
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

func (s *DeviceCodeStore) FindByDeviceCodeHash(ctx context.Context, hash string) (*spec.DeviceAuthorization, error) {
	var rec spec.DeviceAuthorization
	if err := getJSON(ctx, s.Client, tenantKey(ctx, "device:"+hash), &rec); err != nil {
		return nil, err
	}
	if rec.DeviceCodeHash == "" {
		return nil, nil
	}
	return &rec, nil
}

func (s *DeviceCodeStore) FindByUserCode(ctx context.Context, code string) (*spec.DeviceAuthorization, error) {
	hash, err := s.Client.Get(ctx, tenantKey(ctx, "device:user_code:"+code)).Result()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.FindByDeviceCodeHash(ctx, hash)
}

func (s *DeviceCodeStore) Update(ctx context.Context, rec *spec.DeviceAuthorization) error {
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
		var rec spec.DeviceAuthorization
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

func (s *DeviceCodeStore) Exchange(ctx context.Context, hash string) (*spec.DeviceAuthorization, error) {
	result, err := exchangeDevice.Run(ctx, s.Client, []string{tenantKey(ctx, "device:"+hash)}).Text()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rec spec.DeviceAuthorization
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

type SessionStore struct{ Client *goredis.Client }

func (s *SessionStore) Save(ctx context.Context, session *spec.LoginSession) error {
	session.TenantID = tenancy.TenantID(ctx)
	return setJSON(ctx, s.Client, tenantKey(ctx, "session:"+session.ID), session, ttlUntil(session.ExpiresAt))
}

func (s *SessionStore) Find(ctx context.Context, id string) (*spec.LoginSession, error) {
	var session spec.LoginSession
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

func (s *SessionStore) ListBySub(ctx context.Context, sub string) ([]*spec.LoginSession, error) {
	pattern := tenantKey(ctx, "session:*")
	iter := s.Client.Scan(ctx, 0, pattern, 100).Iterator()
	out := []*spec.LoginSession{}
	for iter.Next(ctx) {
		var session spec.LoginSession
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
		var session spec.LoginSession
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
