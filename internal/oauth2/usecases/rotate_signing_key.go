// /rotate (内部運用 — JWKS 鍵回転)
package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/internal/oauth2/ports"
	"github.com/ambi/idmagic/internal/shared/spec"
)

type RotateSigningKeyDeps struct {
	KeyStore ports.KeyStore
	Emit     func(spec.DomainEvent)
}

func RotateSigningKey(ctx context.Context, deps RotateSigningKeyDeps, now time.Time) (*ports.SigningKey, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	prev, _ := deps.KeyStore.GetActiveKey(ctx)
	next, err := deps.KeyStore.Rotate(ctx)
	if err != nil {
		return nil, err
	}
	prevKID := ""
	if prev != nil {
		prevKID = prev.Kid
	}
	// 回転した鍵の帰属テナントを載せ、テナント所属 admin の監査ビューに出す。
	emit(deps.Emit, &spec.SigningKeyRotated{
		At: now, TenantID: next.TenantID, NewKID: next.Kid, PreviousKID: prevKID,
	})
	return next, nil
}
