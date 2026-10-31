// Package usecases は TokenVault の操作を編成する。
package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

var (
	ErrConnectorNotFound       = errors.New("connector not found")
	ErrConnectionNotFound      = errors.New("connection not found")
	ErrReauthorizationRequired = errors.New("connection requires reauthorization")
	ErrAccessDenied            = errors.New("connector access denied")
	ErrInvalidScope            = errors.New("scope is not allowed by the connector")
	ErrInvalidToken            = errors.New("invalid access token")
	ErrTemporarilyUnavailable  = errors.New("temporarily unavailable")
	ErrRequestTooLarge         = errors.New("request body too large")
)

// Deps は TokenVault のユースケースの依存である。
type Deps struct {
	Connectors  ports.ConnectorRepository
	Connections ports.ConnectionRepository
	Attempts    ports.AttemptStore
	Upstream    ports.UpstreamClient
	Verifier    ports.TokenVerifier
	Emit        func(spec.DomainEvent)
	Now         func() time.Time
	// Sleep は更新の待機に使う。テストが時間を進められるよう差し替える。
	Sleep func(ctx context.Context, d time.Duration) error
}

func (d Deps) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now().UTC()
}

func (d Deps) emit(event spec.DomainEvent) {
	if d.Emit != nil {
		d.Emit(event)
	}
}

func (d Deps) sleep(ctx context.Context, wait time.Duration) error {
	if d.Sleep != nil {
		return d.Sleep(ctx, wait)
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
