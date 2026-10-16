// Package testing_contract defines the shared WebAuthn session-store contract.
package testing_contract

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
)

type Fixture struct {
	Store   ports.WebAuthnSessionStore
	Context func(context.Context) context.Context
	Other   func(context.Context) context.Context
	Now     time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx, other := f.Context(context.Background()), f.Other(context.Background())
	data := gowebauthn.SessionData{Challenge: "challenge", UserID: []byte("user-1")}
	if err := f.Store.Save(ctx, "key-1", data, f.Now.Add(time.Minute)); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got, err := f.Store.Take(ctx, "key-1"); err != nil || got == nil || got.Challenge != data.Challenge || !bytes.Equal(got.UserID, data.UserID) {
		t.Fatalf("first Take = (%+v, %v)", got, err)
	}
	if got, err := f.Store.Take(ctx, "key-1"); err != nil || got != nil {
		t.Fatalf("second Take = (%+v, %v), want once-only", got, err)
	}
	if err := f.Store.Save(ctx, "key-expired", data, f.Now.Add(-time.Minute)); err != nil {
		t.Fatalf("Save expired: %v", err)
	}
	if got, err := f.Store.Take(ctx, "key-expired"); err != nil || got != nil {
		t.Fatalf("expired Take = (%+v, %v)", got, err)
	}
	if err := f.Store.Save(ctx, "key-isolated", data, f.Now.Add(time.Minute)); err != nil {
		t.Fatalf("Save isolated: %v", err)
	}
	if got, err := f.Store.Take(other, "key-isolated"); err != nil || got != nil {
		t.Fatalf("cross-tenant Take = (%+v, %v)", got, err)
	}
}
