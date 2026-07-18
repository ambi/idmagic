// /rotate (内部運用 — JWKS 鍵回転)
package usecases

import (
	"context"
	"errors"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys/ports"
)

type RotateSigningKeyDeps struct {
	KeyStore ports.KeyStore
	Emit     func(spec.DomainEvent)
	Grace    time.Duration
}

func RotateSigningKey(ctx context.Context, deps RotateSigningKeyDeps, now time.Time) (*signingdomain.SigningKey, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	prev, _ := deps.KeyStore.GetActiveKey(ctx)
	if deps.Grace == 0 {
		deps.Grace = 7 * 24 * time.Hour
	}
	next, err := deps.KeyStore.Rotate(ctx, now, deps.Grace)
	if err != nil {
		return nil, err
	}
	prevKID := ""
	if prev != nil {
		prevKID = prev.Kid
	}
	// 回転した鍵の帰属テナントを載せ、テナント所属 admin の監査ビューに出す。
	if deps.Emit != nil {
		deps.Emit(&signingdomain.SigningKeyRotated{
			At: now, TenantID: next.TenantID, NewKID: next.Kid, PreviousKID: prevKID,
		})
	}
	return next, nil
}

// RotateSigningKeyIfDue is the one-shot batch entry: it avoids rotation until
// the active key reaches cadence. KeyStore.Rotate serializes the state change
// per tenant; a concurrent newer rotation is observed again by the adapter.
func RotateSigningKeyIfDue(
	ctx context.Context,
	deps RotateSigningKeyDeps,
	now time.Time,
	cadence time.Duration,
) (*signingdomain.SigningKey, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if cadence <= 0 {
		return nil, errors.New("signing key rotation cadence must be positive")
	}
	if deps.Grace == 0 {
		deps.Grace = 7 * 24 * time.Hour
	}
	if rotator, ok := deps.KeyStore.(ports.DueKeyRotator); ok {
		next, err := rotator.RotateIfDue(ctx, now, cadence, deps.Grace)
		if err != nil || next == nil {
			return next, err
		}
		if deps.Emit != nil {
			deps.Emit(&signingdomain.SigningKeyRotated{
				At: now, TenantID: next.TenantID, NewKID: next.Kid,
			})
		}
		return next, nil
	}
	active, err := deps.KeyStore.GetActiveKey(ctx)
	if err != nil {
		return nil, err
	}
	if active != nil && now.Sub(active.CreatedAt) < cadence {
		return nil, nil //nolint:nilnil // nil key explicitly means cadence is not due.
	}
	return RotateSigningKey(ctx, deps, now)
}
