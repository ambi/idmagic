// Package testing_contract defines the shared WebAuthn session-store contract.
package testing_contract

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/webauthn/ports"
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
	session := ports.WebAuthnSession{Payload: json.RawMessage(`{"challenge":"challenge","user_id":"dXNlci0x"}`)}
	if err := f.Store.Save(ctx, "key-1", session, f.Now.Add(time.Minute)); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got, err := f.Store.Take(ctx, "key-1"); err != nil || got == nil || !sameJSON(t, got.Payload, session.Payload) {
		t.Fatalf("first Take = (%+v, %v)", got, err)
	}
	if got, err := f.Store.Take(ctx, "key-1"); err != nil || got != nil {
		t.Fatalf("second Take = (%+v, %v), want once-only", got, err)
	}
	if err := f.Store.Save(ctx, "key-expired", session, f.Now.Add(-time.Minute)); err != nil {
		t.Fatalf("Save expired: %v", err)
	}
	if got, err := f.Store.Take(ctx, "key-expired"); err != nil || got != nil {
		t.Fatalf("expired Take = (%+v, %v)", got, err)
	}
	if err := f.Store.Save(ctx, "key-isolated", session, f.Now.Add(time.Minute)); err != nil {
		t.Fatalf("Save isolated: %v", err)
	}
	if got, err := f.Store.Take(other, "key-isolated"); err != nil || got != nil {
		t.Fatalf("cross-tenant Take = (%+v, %v)", got, err)
	}
}

// sameJSON は JSON の値として等しいかを比べる。PostgreSQL の JSONB はキーの順と空白を
// 保たないので、バイト列では比べない。
func sameJSON(t *testing.T, got, want []byte) bool {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("stored payload is not JSON: %v (%s)", err, got)
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(gotValue, wantValue)
}
