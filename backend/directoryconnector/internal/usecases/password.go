package usecases

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ports"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
)

type PasswordAPI interface {
	RegisterEncryption(context.Context, map[string]any) (string, error)
	PollPassword(context.Context) (*domain.PasswordChallenge, error)
	CompletePassword(context.Context, string, string) error
}
type PasswordDirectory interface {
	VerifyPassword(context.Context, domain.Scope, string, string) string
}

type PasswordReceiver struct {
	API         PasswordAPI
	Directory   PasswordDirectory
	Key         *ecdsa.PrivateKey
	ConnectorID string
	Scope       domain.Scope
	Now         func() time.Time
	Sleep       func(context.Context, time.Duration) error
}

func (r PasswordReceiver) Run(ctx context.Context) {
	point, err := r.Key.PublicKey.Bytes()
	if err != nil {
		return
	}
	jwk := map[string]any{"kty": "EC", "crv": "P-256", "x": base64.RawURLEncoding.EncodeToString(point[1:33]), "y": base64.RawURLEncoding.EncodeToString(point[33:65])}
	kid, err := tokens_jose.ValidateEncryptionJWK(jwk)
	if err != nil {
		return
	}
	for ctx.Err() == nil {
		got, err := r.API.RegisterEncryption(ctx, jwk)
		if err == nil && got == kid {
			break
		}
		if api, ok := errors.AsType[*ports.APIError](err); ok && (api.Status == 404 || api.Status == 405) {
			return
		}
		if r.Sleep(ctx, time.Second) != nil {
			return
		}
	}
	seen := map[string]time.Time{}
	for ctx.Err() == nil {
		challenge, err := r.API.PollPassword(ctx)
		if err != nil {
			if r.Sleep(ctx, time.Second) != nil {
				return
			}
			continue
		}
		if challenge == nil {
			continue
		}
		now := r.Now()
		for id, expires := range seen {
			if !now.Before(expires) {
				delete(seen, id)
			}
		}
		if _, duplicate := seen[challenge.ID]; duplicate {
			continue
		}
		plain, err := tokens_jose.DecryptCompact(r.Key, kid, challenge.EncryptedRequest)
		if err != nil {
			continue
		}
		var payload domain.PasswordPayload
		if json.Unmarshal(plain, &payload) != nil || !payload.Matches(*challenge, r.ConnectorID, kid, now) {
			continue
		}
		seen[challenge.ID] = challenge.ExpiresAt
		deadline, cancel := context.WithTimeout(ctx, min(5*time.Second, challenge.ExpiresAt.Sub(now)))
		result := r.Directory.VerifyPassword(deadline, r.Scope, payload.ExternalID, payload.Password)
		payload.Password = ""
		for deadline.Err() == nil {
			err := r.API.CompletePassword(deadline, challenge.ID, result)
			if err == nil {
				break
			}
			if api, ok := errors.AsType[*ports.APIError](err); ok && api.Status < 500 && api.Status != 408 && api.Status != 429 {
				break
			}
			if r.Sleep(deadline, 100*time.Millisecond) != nil {
				break
			}
		}
		cancel()
	}
}
