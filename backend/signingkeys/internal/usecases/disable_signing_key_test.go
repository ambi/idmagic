package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	"github.com/ambi/idmagic/backend/signingkeys/internal/usecases"
	"github.com/ambi/idmagic/backend/signingkeys/ports"
)

type disableStore struct {
	ports.KeyStore
	key     *signingdomain.SigningKey
	changed bool
	err     error
}

func (s disableStore) Disable(context.Context, string) (*signingdomain.SigningKey, bool, error) {
	return s.key, s.changed, s.err
}

//spec:covers REQ-SIGNINGKEYS-010: 保存失敗、拒否、存在しない鍵と変更のない無効化ではイベントを発行しない。
func TestDisableSigningKeyEmitsOnlyCommittedChanges(t *testing.T) {
	at := time.Now().UTC()
	key := &signingdomain.SigningKey{TenantID: "tenant", Kid: "old", ArchivedAt: &at}
	failed := errors.New("persistence failed")
	for _, tc := range []struct {
		name       string
		store      disableStore
		wantEvents int
	}{
		{"changed", disableStore{key: key, changed: true}, 1},
		{"unchanged", disableStore{key: key}, 0},
		{"missing", disableStore{}, 0},
		{"active", disableStore{err: signingdomain.ErrActiveSigningKeyCannotBeDisabled}, 0},
		{"failed commit", disableStore{key: key, changed: true, err: failed}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var events []spec.DomainEvent
			got, err := usecases.DisableSigningKey(context.Background(), usecases.DisableSigningKeyDeps{KeyStore: tc.store, Emit: func(event spec.DomainEvent) { events = append(events, event) }}, "old")
			if !errors.Is(err, tc.store.err) {
				t.Fatalf("error = %v, want %v", err, tc.store.err)
			}
			if err == nil && got != tc.store.key {
				t.Fatalf("key = %+v, want %+v", got, tc.store.key)
			}
			if len(events) != tc.wantEvents {
				t.Fatalf("events = %v, want %d", events, tc.wantEvents)
			}
		})
	}
}
