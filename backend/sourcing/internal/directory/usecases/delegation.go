package usecases

import (
	"context"
	"encoding/json"
	"time"
	"uuid"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type Delegation struct {
	Service      *Service
	Correlations ports.CorrelationRepository
	Challenges   ports.PasswordChallengeStore
	Locks        ports.IngestionRunRepository
	Users        userports.UserRepository
	Now          func() time.Time
}

type PasswordVerification struct {
	Delegated bool
	Result    domain.PasswordResult
}

// WithPasswordConfigurationLock は短い設定変更と資格情報の保存だけを直列化する。
// 配送の待機で使うとテナント全体のパスワード操作を停止させるため、Verify では保持しない。
func (d *Delegation) WithPasswordConfigurationLock(ctx context.Context, apply func() error) error {
	tenantID := tenantports.TenantID(ctx)
	return d.Locks.WithConnectorLock(ctx, tenantID, "password-configuration", apply)
}

func (d *Delegation) RegisterEncryption(ctx context.Context, connector domain.Connector, jwk map[string]any) (string, error) {
	kid, err := tokens_jose.ValidateEncryptionJWK(jwk)
	if err != nil || kid == connector.KeyID {
		return "", ErrInvalidKey
	}
	err = d.WithPasswordConfigurationLock(ctx, func() error {
		return d.Service.update(ctx, connector.TenantID, connector.ID, func(c domain.Connector, _ time.Time) (domain.Connector, bool, error) {
			if c.Status != domain.StatusActive || c.KeyID != connector.KeyID {
				return c, false, ports.ErrDelegationUnavailable
			}
			if c.EncryptionKeyID == kid {
				return c, false, nil
			}
			c.EncryptionPublicKey = publicMembers(jwk)
			c.EncryptionKeyID = kid
			c.Revision++
			return c, true, nil
		})
	})
	return kid, err
}

func (d *Delegation) Configure(ctx context.Context, id string, enabled bool) (domain.Connector, error) {
	tenantID := tenantports.TenantID(ctx)
	err := d.WithPasswordConfigurationLock(ctx, func() error {
		return d.Service.update(ctx, tenantID, id, func(c domain.Connector, _ time.Time) (domain.Connector, bool, error) {
			if enabled {
				if c.Status != domain.StatusActive || c.EncryptionKeyID == "" || c.EncryptionPublicKey == nil {
					return c, false, ports.ErrDelegationUnavailable
				}
				users, err := d.Correlations.List(ctx, tenantID, id)
				if err != nil {
					return c, false, err
				}
				for _, correlation := range users {
					u, err := d.Users.FindBySub(ctx, correlation.UserID)
					if err != nil {
						return c, false, err
					}
					if u != nil && u.PasswordHash != "" {
						return c, false, ports.ErrDelegationUnavailable
					}
				}
			}
			c.DelegatedAuthentication = enabled
			c.Revision++
			return c, true, nil
		})
	})
	if err != nil {
		return domain.Connector{}, err
	}
	return d.Service.Get(ctx, tenantID, id)
}

func (d *Delegation) Verify(ctx context.Context, userID, password string) (PasswordVerification, error) {
	tenantID := tenantports.TenantID(ctx)
	corr, err := d.Correlations.FindByUser(ctx, tenantID, userID)
	if err != nil {
		return PasswordVerification{Delegated: true, Result: domain.PasswordConnectorUnavailable}, err
	}
	if corr == nil {
		return PasswordVerification{}, nil
	}
	c, err := d.Service.Get(ctx, tenantID, corr.ConnectorID)
	if err != nil {
		return PasswordVerification{Delegated: true, Result: domain.PasswordConnectorUnavailable}, err
	}
	if !c.DelegatedAuthentication {
		return PasswordVerification{}, nil
	}
	result, err := d.verify(ctx, c, corr.ExternalID, password)
	return PasswordVerification{Delegated: true, Result: result}, err
}

func (d *Delegation) AllowsLocalPassword(ctx context.Context, userID string) (bool, error) {
	corr, err := d.Correlations.FindByUser(ctx, tenantports.TenantID(ctx), userID)
	if err != nil {
		return false, err
	}
	if corr == nil {
		return true, nil
	}
	c, err := d.Service.Get(ctx, corr.TenantID, corr.ConnectorID)
	return err == nil && !c.DelegatedAuthentication, err
}

func (d *Delegation) Test(ctx context.Context, connectorID, userID, password string) (domain.PasswordResult, error) {
	corr, err := d.Correlations.FindByUser(ctx, tenantports.TenantID(ctx), userID)
	if err != nil {
		return "", err
	}
	if corr == nil || corr.ConnectorID != connectorID {
		return "", ErrNotFound
	}
	c, err := d.Service.Get(ctx, corr.TenantID, connectorID)
	if err != nil {
		return "", err
	}
	return d.verify(ctx, c, corr.ExternalID, password)
}

func (d *Delegation) verify(ctx context.Context, c domain.Connector, externalID, password string) (domain.PasswordResult, error) {
	if password == "" || len(password) > 16384 {
		return domain.PasswordInvalid, nil
	}
	if c.Status != domain.StatusActive || c.EncryptionPublicKey == nil || c.EncryptionKeyID == "" {
		return domain.PasswordConnectorUnavailable, nil
	}
	now := d.Now().UTC().Truncate(time.Microsecond)
	challenge := domain.PasswordChallenge{ID: uuid.NewV4().String(), TenantID: c.TenantID, ConnectorID: c.ID, KeyID: c.EncryptionKeyID, Revision: c.Revision, ExpiresAt: now.Add(domain.PasswordChallengeLifetime)}
	payload, err := json.Marshal(domain.PasswordPayload{ID: challenge.ID, TenantID: c.TenantID, ConnectorID: c.ID, KeyID: c.EncryptionKeyID, ExpiresAt: challenge.ExpiresAt, Operation: "verify_password", ExternalID: externalID, Password: password}) //nolint:gosec // 直後に公開鍵で暗号化し、平文はログと保存先へ出さない。
	if err != nil {
		return "", err
	}
	challenge.EncryptedRequest, err = tokens_jose.EncryptCompact(c.EncryptionPublicKey, payload)
	if err != nil {
		return "", err
	}
	if err := d.Challenges.Sweep(ctx, now); err != nil {
		return domain.PasswordConnectorUnavailable, err
	}
	if err := d.Challenges.Enqueue(ctx, challenge, now); err != nil {
		return domain.PasswordConnectorUnavailable, err
	}
	completed := false
	defer func() {
		if completed {
			return
		}
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		_ = d.Challenges.Delete(cleanup, c.TenantID, challenge.ID)
	}()
	wait, cancel := context.WithTimeout(ctx, domain.PasswordChallengeLifetime)
	defer cancel()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-wait.Done():
			return domain.PasswordConnectorUnavailable, nil
		case <-tick.C:
			item, err := d.Challenges.Find(wait, c.TenantID, challenge.ID)
			if err != nil {
				return domain.PasswordConnectorUnavailable, err
			}
			if item == nil || !d.Now().Before(challenge.ExpiresAt) {
				return domain.PasswordConnectorUnavailable, nil
			}
			if item.Result != "" {
				latest, err := d.Service.Get(wait, c.TenantID, c.ID)
				if err != nil {
					return domain.PasswordConnectorUnavailable, err
				}
				if latest.Status != domain.StatusActive || latest.Revision != c.Revision || latest.EncryptionKeyID != c.EncryptionKeyID {
					return domain.PasswordConnectorUnavailable, nil
				}
				completed = true
				return item.Result, nil
			}
		}
	}
}

func (d *Delegation) Poll(ctx context.Context, c domain.Connector) (*domain.PasswordChallenge, error) {
	if c.Status != domain.StatusActive || c.EncryptionKeyID == "" {
		return nil, ports.ErrDelegationUnavailable
	}
	if err := d.Challenges.Sweep(ctx, d.Now()); err != nil {
		return nil, err
	}
	wait, cancel := context.WithTimeout(ctx, domain.PasswordPollLifetime)
	defer cancel()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		latest, err := d.Service.Get(wait, c.TenantID, c.ID)
		if err != nil {
			return nil, err
		}
		if latest.Status != domain.StatusActive || latest.Revision != c.Revision {
			return nil, ports.ErrDelegationUnavailable
		}
		item, err := d.Challenges.Claim(wait, c.TenantID, c.ID, c.Revision, d.Now())
		if err != nil || item != nil {
			return item, err
		}
		select {
		case <-wait.Done():
			return nil, nil //nolint:nilnil // 待機の期限切れは空の 204 で再接続させる。
		case <-tick.C:
		}
	}
}

func (d *Delegation) Complete(ctx context.Context, c domain.Connector, id string, result domain.PasswordResult) error {
	latest, err := d.Service.Get(ctx, c.TenantID, c.ID)
	if err != nil {
		return err
	}
	if latest.Status != domain.StatusActive || latest.Revision != c.Revision {
		return ports.ErrDelegationUnavailable
	}
	return d.Challenges.Complete(ctx, c.TenantID, c.ID, id, c.Revision, result, d.Now())
}
