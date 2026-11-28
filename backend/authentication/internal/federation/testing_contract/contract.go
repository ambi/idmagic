// Package testing_contract defines the shared federation persistence contract.
package testing_contract

import (
	"context"
	"errors"
	"testing"
	"time"

	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	federationports "github.com/ambi/idmagic/backend/authentication/internal/federation/ports"
)

type Fixture struct {
	Attempts   federationports.AttemptStore
	Replay     federationports.ReplayStore
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
		attempt := &federationdomain.FederatedLoginAttempt{
			State: f.State, TenantID: f.TenantA, ProviderID: f.ProviderID, Protocol: federationdomain.ProtocolOIDC,
			Nonce: "nonce", PKCEVerifier: "verifier", ReturnTo: "/account",
			CreatedAt: f.Now, ExpiresAt: f.Now.Add(time.Minute),
		}
		if err := f.Attempts.Save(ctx, attempt); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if got, err := f.Attempts.Consume(ctx, f.TenantB, f.State, f.Now); got != nil || !errors.Is(err, federationports.ErrAttemptNotFound) {
			t.Fatalf("other tenant Consume = (%+v, %v)", got, err)
		}
		consumed, err := f.Attempts.Consume(ctx, f.TenantA, f.State, f.Now.Add(time.Second))
		if err != nil || consumed == nil || consumed.ConsumedAt == nil || consumed.Nonce != attempt.Nonce {
			t.Fatalf("first Consume = (%+v, %v)", consumed, err)
		}
		if got, err := f.Attempts.Consume(ctx, f.TenantA, f.State, f.Now.Add(2*time.Second)); got != nil || !errors.Is(err, federationports.ErrAttemptConsumed) {
			t.Fatalf("second Consume = (%+v, %v)", got, err)
		}
	})

	t.Run("expired attempt is not consumable", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		attempt := &federationdomain.FederatedLoginAttempt{
			State: f.State, TenantID: f.TenantA, ProviderID: f.ProviderID, Protocol: federationdomain.ProtocolOIDC,
			CreatedAt: f.Now, ExpiresAt: f.Now.Add(time.Minute),
		}
		if err := f.Attempts.Save(ctx, attempt); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if got, err := f.Attempts.Consume(ctx, f.TenantA, f.State, f.Now.Add(2*time.Minute)); got != nil || !errors.Is(err, federationports.ErrAttemptConsumed) {
			t.Fatalf("expired Consume = (%+v, %v)", got, err)
		}
	})

	// 予約の期限は呼び出し側が渡した検証の時刻で判定する。f.Now は実時計より過去なので、
	// 実装が実時計と比べると、期限内の再予約まで受け入れてしまう。
	t.Run("replay reservation expires on the caller's clock", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		const ttl = 10 * time.Minute
		steps := []struct {
			name   string
			tenant string
			at     time.Time
			want   bool
		}{
			{"first reservation", f.TenantA, f.Now, true},
			{"same tenant just before expiry", f.TenantA, f.Now.Add(ttl - time.Microsecond), false},
			{"other tenant within the window", f.TenantB, f.Now.Add(time.Minute), true},
			{"same tenant at expiry", f.TenantA, f.Now.Add(ttl), true},
			{"same tenant within the renewed window", f.TenantA, f.Now.Add(ttl + time.Minute), false},
		}
		for _, step := range steps {
			if ok, err := f.Replay.Reserve(ctx, step.tenant, "response", step.at, ttl); err != nil || ok != step.want {
				t.Fatalf("%s: Reserve = (%v, %v), want (%v, nil)", step.name, ok, err, step.want)
			}
		}
	})

	t.Run("replay reservation requires a clock and a positive ttl", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		if ok, err := f.Replay.Reserve(ctx, f.TenantA, "zero-now", time.Time{}, time.Minute); err == nil || ok {
			t.Fatalf("zero now: Reserve = (%v, %v), want (false, error)", ok, err)
		}
		if ok, err := f.Replay.Reserve(ctx, f.TenantA, "zero-ttl", f.Now, 0); err == nil || ok {
			t.Fatalf("zero ttl: Reserve = (%v, %v), want (false, error)", ok, err)
		}
		// 拒否した入力は予約を残さない。
		if ok, err := f.Replay.Reserve(ctx, f.TenantA, "zero-ttl", f.Now, time.Minute); err != nil || !ok {
			t.Fatalf("after rejected input: Reserve = (%v, %v), want (true, nil)", ok, err)
		}
	})
}
