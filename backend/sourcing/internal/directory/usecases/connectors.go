// Package usecases は Directory Connector の作成、登録、無効化と、Connector の呼び出しの認証を編成する。
package usecases

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
)

// EnrollmentTokenPrefix は登録トークンを、ほかの秘密と見分けられるようにする接頭辞。
const EnrollmentTokenPrefix = "idmagic_dce_"

// AudiencePath は、テナントの発行者識別子に続けて Connector の JWT の aud にする経路。
const AudiencePath = "/directory-connector/v1"

// maxStaleRetries は、同時の更新で Revision が食い違ったときに管理操作を読み直す回数。
const maxStaleRetries = 3

var (
	ErrNotFound     = errors.New("directory connector: not found")
	ErrInvalidToken = errors.New("directory connector: invalid token")
	ErrInvalidKey   = errors.New("directory connector: invalid public key")
)

// Grant は、作成と再発行が返す Connector と、一度だけ返す登録トークン。
type Grant struct {
	Connector domain.Connector
	Token     string
}

type Service struct {
	repository ports.ConnectorRepository
	replay     tokens_jose.ReplayStore
	emit       func(spec.DomainEvent)
	now        func() time.Time
}

type Option func(*Service)

func WithClock(now func() time.Time) Option { return func(s *Service) { s.now = now } }

// New は Service を作る。replay は Connector の JWT の jti を一度だけ記録する。
func New(repository ports.ConnectorRepository, replay tokens_jose.ReplayStore, emit func(spec.DomainEvent), options ...Option) *Service {
	s := &Service{repository: repository, replay: replay, emit: emit, now: time.Now}
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *Service) Create(ctx context.Context, tenantID, displayName string) (Grant, error) {
	token, hash, err := newEnrollmentToken()
	if err != nil {
		return Grant{}, err
	}
	now := s.now().UTC()
	connector, err := domain.NewConnector(tenantID, uuid.NewString(), displayName, hash, now)
	if err != nil {
		return Grant{}, err
	}
	if err := s.repository.Create(ctx, connector); err != nil {
		return Grant{}, err
	}
	s.publish(&domain.DirectoryConnectorCreated{At: now, TenantID: tenantID, ConnectorID: connector.ID})
	return Grant{Connector: connector, Token: token}, nil
}

func (s *Service) List(ctx context.Context, tenantID string) ([]domain.Connector, error) {
	return s.repository.List(ctx, tenantID)
}

func (s *Service) Get(ctx context.Context, tenantID, id string) (domain.Connector, error) {
	connector, err := s.repository.Find(ctx, tenantID, id)
	if err != nil {
		return domain.Connector{}, err
	}
	if connector == nil {
		return domain.Connector{}, ErrNotFound
	}
	return *connector, nil
}

func (s *Service) ReissueEnrollment(ctx context.Context, tenantID, id string) (Grant, error) {
	token, hash, err := newEnrollmentToken()
	if err != nil {
		return Grant{}, err
	}
	var reissued domain.Connector
	var at time.Time
	err = s.update(ctx, tenantID, id, func(current domain.Connector, now time.Time) (domain.Connector, bool, error) {
		next, err := current.ReissueEnrollment(hash, now)
		reissued, at = next, now
		return next, true, err
	})
	if err != nil {
		return Grant{}, err
	}
	s.publish(&domain.DirectoryConnectorEnrollmentReissued{At: at, TenantID: tenantID, ConnectorID: id})
	return Grant{Connector: reissued, Token: token}, nil
}

func (s *Service) Disable(ctx context.Context, tenantID, id string) (domain.Connector, error) {
	var result domain.Connector
	var changed bool
	err := s.update(ctx, tenantID, id, func(current domain.Connector, now time.Time) (domain.Connector, bool, error) {
		result, changed = current.Disable(now)
		return result, changed, nil
	})
	if err != nil {
		return domain.Connector{}, err
	}
	if changed {
		s.publish(&domain.DirectoryConnectorDisabled{At: *result.DisabledAt, TenantID: tenantID, ConnectorID: id})
	}
	return result, nil
}

// Enroll は登録トークンで Connector を引き、公開鍵を登録する。トークンの不一致、期限切れ、
// 同時の登録に負けたことは、どれも ErrInvalidToken として区別せずに返す。
func (s *Service) Enroll(ctx context.Context, tenantID, token string, jwk map[string]any) (domain.Connector, error) {
	connector, err := s.repository.FindByEnrollmentTokenHash(ctx, tenantID, hashToken(token))
	if err != nil {
		return domain.Connector{}, err
	}
	now := s.now().UTC()
	if connector == nil || connector.Status != domain.StatusAwaitingEnrollment || now.After(connector.EnrollmentExpiresAt) {
		return domain.Connector{}, ErrInvalidToken
	}
	keyID, err := tokens_jose.ValidatePublicSigningJWK(jwk)
	if err != nil {
		return domain.Connector{}, ErrInvalidKey
	}
	stored := publicMembers(jwk)
	stored["kid"] = keyID
	enrolled, err := connector.Enroll(stored, keyID, now)
	if err != nil {
		return domain.Connector{}, ErrInvalidToken
	}
	if err := s.repository.Update(ctx, enrolled); err != nil {
		if errors.Is(err, ports.ErrStaleConnector) {
			return domain.Connector{}, ErrInvalidToken
		}
		return domain.Connector{}, err
	}
	s.publish(&domain.DirectoryConnectorEnrolled{At: now, TenantID: tenantID, ConnectorID: enrolled.ID, KeyID: keyID})
	return enrolled, nil
}

// Authenticate は Connector の JWT を検証し、要求したテナントの Active の Connector を返す。
// どの検査に失敗しても ErrInvalidToken を返す。
func (s *Service) Authenticate(ctx context.Context, tenantID, assertion, audience string) (domain.Connector, error) {
	issuer, ok := unverifiedIssuer(assertion)
	if !ok {
		return domain.Connector{}, ErrInvalidToken
	}
	var found domain.Connector
	_, err := tokens_jose.VerifyClientAssertion(ctx, assertion, issuer, []string{audience},
		func(ctx context.Context, id string) ([]map[string]any, error) {
			connector, err := s.repository.Find(ctx, tenantID, id)
			if err != nil {
				return nil, err
			}
			if connector == nil || connector.Status != domain.StatusActive || connector.PublicKey == nil {
				return nil, ErrInvalidToken
			}
			found = *connector
			return []map[string]any{connector.PublicKey}, nil
		},
		s.replay, s.now().UTC(), nil,
	)
	if err != nil {
		return domain.Connector{}, ErrInvalidToken
	}
	return found, nil
}

// Heartbeat は Connector を認証し、最終接続時刻を記録する。
func (s *Service) Heartbeat(ctx context.Context, tenantID, assertion, audience string) error {
	connector, err := s.Authenticate(ctx, tenantID, assertion, audience)
	if err != nil {
		return err
	}
	return s.repository.TouchLastSeen(ctx, tenantID, connector.ID, s.now().UTC())
}

// update は Connector を読んで change を適用し、同時の更新に負けたら読み直す。
// change の二つ目の値が false なら保存しない。
func (s *Service) update(ctx context.Context, tenantID, id string,
	change func(domain.Connector, time.Time) (domain.Connector, bool, error),
) error {
	for attempt := 0; ; attempt++ {
		current, err := s.Get(ctx, tenantID, id)
		if err != nil {
			return err
		}
		next, save, err := change(current, s.now().UTC())
		if err != nil || !save {
			return err
		}
		err = s.repository.Update(ctx, next)
		if !errors.Is(err, ports.ErrStaleConnector) || attempt+1 >= maxStaleRetries {
			return err
		}
	}
}

func (s *Service) publish(event spec.DomainEvent) {
	if s.emit != nil {
		s.emit(event)
	}
}

func newEnrollmentToken() (token, hash string, err error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", "", err
	}
	token = EnrollmentTokenPrefix + base64.RawURLEncoding.EncodeToString(random)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

// publicMembers は、検証済みの JWK から鍵の種類ごとの公開のメンバーだけを写す。
func publicMembers(jwk map[string]any) map[string]any {
	members := []string{"kty", "crv", "x", "y"}
	if jwk["kty"] == "RSA" {
		members = []string{"kty", "n", "e"}
	}
	copied := map[string]any{}
	for _, member := range members {
		if value, ok := jwk[member]; ok {
			copied[member] = value
		}
	}
	return copied
}

// unverifiedIssuer は、Connector を引く鍵として署名の検証前に iss を読む。
// 値は検証の成功まで信頼しない。
func unverifiedIssuer(assertion string) (string, bool) {
	parts := strings.Split(assertion, ".")
	if len(parts) != 3 {
		return "", false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	var claims struct {
		Issuer string `json:"iss"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Issuer == "" {
		return "", false
	}
	return claims.Issuer, true
}
