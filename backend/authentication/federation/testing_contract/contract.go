// Package testing_contract defines the shared federation persistence contract.
package testing_contract

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/federation/domain"
	"github.com/ambi/idmagic/backend/authentication/federation/ports"
)

type Fixture struct {
	Attempts   ports.AttemptStore
	TenantA    string
	TenantB    string
	ProviderID string
	State      string
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	t.Run("consume once and isolate tenants", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		attempt := &domain.FederatedLoginAttempt{
			State: f.State, TenantID: f.TenantA, ProviderID: f.ProviderID, Protocol: domain.ProtocolOIDC,
			Nonce: "nonce", PKCEVerifier: "verifier", ReturnTo: "/account",
			CreatedAt: f.Now, ExpiresAt: f.Now.Add(time.Minute),
		}
		if err := f.Attempts.Save(ctx, attempt); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if got, err := f.Attempts.Consume(ctx, f.TenantB, f.State, f.Now); got != nil || !errors.Is(err, ports.ErrAttemptNotFound) {
			t.Fatalf("other tenant Consume = (%+v, %v)", got, err)
		}
		consumed, err := f.Attempts.Consume(ctx, f.TenantA, f.State, f.Now.Add(time.Second))
		if err != nil || consumed == nil || consumed.ConsumedAt == nil || consumed.Nonce != attempt.Nonce {
			t.Fatalf("first Consume = (%+v, %v)", consumed, err)
		}
		if got, err := f.Attempts.Consume(ctx, f.TenantA, f.State, f.Now.Add(2*time.Second)); got != nil || !errors.Is(err, ports.ErrAttemptConsumed) {
			t.Fatalf("second Consume = (%+v, %v)", got, err)
		}
	})

	t.Run("expired attempt is not consumable", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		attempt := &domain.FederatedLoginAttempt{
			State: f.State, TenantID: f.TenantA, ProviderID: f.ProviderID, Protocol: domain.ProtocolOIDC,
			CreatedAt: f.Now, ExpiresAt: f.Now.Add(time.Minute),
		}
		if err := f.Attempts.Save(ctx, attempt); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if got, err := f.Attempts.Consume(ctx, f.TenantA, f.State, f.Now.Add(2*time.Minute)); got != nil || !errors.Is(err, ports.ErrAttemptConsumed) {
			t.Fatalf("expired Consume = (%+v, %v)", got, err)
		}
	})
}
