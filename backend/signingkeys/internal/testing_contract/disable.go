package testing_contract

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// RunDisable は保存先を越えて無効化後の素材と公開状態を検査する。
func RunDisable(t *testing.T, newFixture func(*testing.T) Fixture) {
	t.Helper()
	for _, usage := range []signingdomain.KeyUsage{signingdomain.KeyUsageSigning, signingdomain.KeyUsageXMLFederationSigning} {
		t.Run(string(usage), func(t *testing.T) {
			f := newFixture(t)
			ctx := signingports.WithKeyScope(signingports.WithKeyUsage(f.Context, usage), "disable-contract")
			now := time.Now().UTC().Truncate(time.Microsecond)
			old, err := f.Store.GetActiveKey(ctx)
			if err != nil {
				t.Fatal(err)
			}
			current, err := f.Store.Rotate(ctx, now, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			before, err := f.Store.FindByKID(ctx, old.Kid)
			if err != nil || before == nil {
				t.Fatalf("before disable: %v %+v", err, before)
			}
			foreignTenant := tenantports.WithTenantID(ctx, "00000000-0000-4000-8000-000000000099")
			if f.OtherTenantID != "" {
				foreignTenant = tenantports.WithTenantID(ctx, f.OtherTenantID)
			}
			if key, changed, err := f.Store.Disable(foreignTenant, old.Kid); err != nil || key != nil || changed {
				t.Fatalf("foreign tenant disable: %v %+v", err, key)
			}
			if key, changed, err := f.Store.Disable(signingports.WithKeyScope(ctx, "other"), old.Kid); err != nil || key != nil || changed {
				t.Fatalf("foreign scope disable: %v %+v", err, key)
			}
			if _, changed, err := f.Store.Disable(ctx, current.Kid); !errors.Is(err, signingdomain.ErrActiveSigningKeyCannotBeDisabled) || changed {
				t.Fatalf("active disable: %v", err)
			}
			disabled, changed, err := f.Store.Disable(ctx, old.Kid)
			if err != nil || !changed || disabled == nil || disabled.ArchivedAt == nil {
				t.Fatalf("disable must return archived key: %v %+v", err, disabled)
			}
			stored, err := f.Store.FindByKID(ctx, old.Kid)
			if err != nil || stored == nil || stored.ArchivedAt == nil {
				t.Fatalf("archived key must remain stored: %v %+v", err, stored)
			}
			if !reflect.DeepEqual(stored.PrivateKey, before.PrivateKey) || !reflect.DeepEqual(stored.PublicJWK, before.PublicJWK) || !reflect.DeepEqual(stored.CertificateDER, before.CertificateDER) {
				t.Fatal("disable changed key material")
			}
			if repeated, changed, err := f.Store.Disable(ctx, old.Kid); err != nil || changed || repeated == nil || !repeated.ArchivedAt.Equal(*stored.ArchivedAt) {
				t.Fatalf("repeated disable changed archive: %v %+v", err, repeated)
			}
			keys, err := f.Store.ListPublicKeys(ctx, now)
			if err != nil || len(keys) != 1 || keys[0].Kid != current.Kid {
				t.Fatalf("public keys after disable: %v %+v", err, keys)
			}
			active, err := f.Store.GetActiveKey(ctx)
			if err != nil || active.Kid != current.Kid {
				t.Fatalf("active key changed: %v %+v", err, active)
			}
		})
	}
	t.Run("prepared", func(t *testing.T) {
		f := newFixture(t)
		ctx := signingports.WithKeyUsage(f.Context, signingdomain.KeyUsageXMLFederationSigning)
		prepared, err := f.Store.Prepare(ctx, time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		key, changed, err := f.Store.Disable(ctx, prepared.Next.Kid)
		if err != nil || !changed || key == nil || key.Prepared || key.ArchivedAt == nil {
			t.Fatalf("disable prepared: %v %+v", err, key)
		}
		stored, err := f.Store.FindByKID(ctx, key.Kid)
		if err != nil || stored == nil || stored.Prepared || stored.ArchivedAt == nil || !reflect.DeepEqual(stored.CertificateDER, prepared.Next.CertificateDER) {
			t.Fatal("prepared material was not archived")
		}
	})
	t.Run("retired", func(t *testing.T) {
		f := newFixture(t)
		old, err := f.Store.GetActiveKey(f.Context)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Store.Rotate(f.Context, time.Now().UTC().Add(-2*time.Hour), time.Hour); err != nil {
			t.Fatal(err)
		}
		key, changed, err := f.Store.Disable(f.Context, old.Kid)
		if err != nil || !changed || key == nil || key.ArchivedAt == nil {
			t.Fatalf("disable retired: %v %+v", err, key)
		}
	})
	t.Run("concurrent", func(t *testing.T) {
		f := newFixture(t)
		old, err := f.Store.GetActiveKey(f.Context)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Store.Rotate(f.Context, time.Now().UTC(), time.Hour); err != nil {
			t.Fatal(err)
		}
		type result struct {
			changed bool
			err     error
		}
		results := make(chan result, 2)
		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() { _, changed, err := f.Store.Disable(f.Context, old.Kid); results <- result{changed, err} })
		}
		wg.Wait()
		close(results)
		changes := 0
		for result := range results {
			if result.err != nil {
				t.Fatal(result.err)
			}
			if result.changed {
				changes++
			}
		}
		if changes != 1 {
			t.Fatalf("concurrent disable changes = %d, want 1", changes)
		}
	})
}
