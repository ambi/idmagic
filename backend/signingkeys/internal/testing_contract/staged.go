package testing_contract

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type (
	StagedStore interface {
		signingports.KeyStore
		signingports.FederationKeyStore
	}
	Fixture struct {
		Store   StagedStore
		Context context.Context
	}
)

//spec:covers REQ-SIGNINGKEYS-006: 準備と切り替え、拒否後の状態、重複、スコープと猶予の境界を同じ契約で検査すること。
func Run(t *testing.T, newFixture func(*testing.T) Fixture) {
	t.Helper()
	f := newFixture(t)
	ctx := signingports.WithKeyUsage(f.Context, signingdomain.KeyUsageXMLFederationSigning)
	now := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	old, err := f.Store.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	jwt, err := f.Store.GetActiveKey(f.Context)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Store.Rotate(ctx, now, time.Hour); !errors.Is(err, signingdomain.ErrPrepublicationRequired) {
		t.Fatalf("immediate shared rotation: %v", err)
	}
	var wg sync.WaitGroup
	results := make(chan signingdomain.RotationResult, 2)
	errorsOut := make(chan error, 2)
	for range 2 {
		wg.Go(func() { result, err := f.Store.Prepare(ctx, now); results <- result; errorsOut <- err })
	}
	wg.Wait()
	close(results)
	close(errorsOut)
	for err := range errorsOut {
		if err != nil {
			t.Fatal(err)
		}
	}
	var prepared *signingdomain.SigningKey
	changes := 0
	for result := range results {
		if result.Changed {
			changes++
		}
		if prepared != nil && prepared.Kid != result.Next.Kid {
			t.Fatal("concurrent prepare produced different keys")
		}
		prepared = result.Next
	}
	if changes != 1 || prepared == nil || !prepared.Prepared || prepared.Active {
		t.Fatalf("prepare changes=%d key=%+v", changes, prepared)
	}
	if found, err := f.Store.FindByKID(ctx, prepared.Kid); err != nil || found == nil || !found.Prepared {
		t.Fatalf("prepared key lookup: %v %+v", err, found)
	}
	if found, err := f.Store.FindByKID(ctx, "unknown"); err != nil || found != nil {
		t.Fatalf("unknown key lookup: %v %+v", err, found)
	}
	public, err := f.Store.ListPublicKeys(ctx, now)
	if err != nil || len(public) != 2 {
		t.Fatalf("active and prepared must both be published: %v %+v", err, public)
	}
	active, err := f.Store.GetActiveKey(ctx)
	if err != nil || active.Kid != old.Kid || active.ExpiresAt != nil {
		t.Fatalf("prepare changed active: %v %+v", err, active)
	}
	otherScope := signingports.WithKeyScope(ctx, "another-profile")
	if _, err := f.Store.Activate(otherScope, prepared.Kid, now, time.Hour); !errors.Is(err, signingdomain.ErrKeyNotPrepared) {
		t.Fatalf("foreign scope activation: %v", err)
	}
	otherTenant := tenantports.WithTenant(ctx, &tenancydomain.Tenant{ID: "8d01be1a-b6d8-43f0-8bcf-13a5f888d011", Realm: "other"}, "", "")
	if _, err := f.Store.Activate(otherTenant, prepared.Kid, now, time.Hour); !errors.Is(err, signingdomain.ErrKeyNotPrepared) {
		t.Fatalf("foreign tenant activation: %v", err)
	}
	if _, err := f.Store.Activate(ctx, "unknown", now, time.Hour); !errors.Is(err, signingdomain.ErrKeyNotPrepared) {
		t.Fatalf("unknown activation: %v", err)
	}
	before := now.Add(24 * time.Hour)
	activated, err := f.Store.Activate(ctx, prepared.Kid, before, 7*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !activated.Changed || !activated.Next.Active || activated.Next.Prepared || activated.Previous == nil || activated.Previous.Active || activated.Previous.ExpiresAt == nil || !activated.Previous.ExpiresAt.Equal(before.Add(7*24*time.Hour)) {
		t.Fatalf("activate result: %+v", activated)
	}
	duplicate, err := f.Store.Activate(ctx, prepared.Kid, before.Add(time.Hour), 7*24*time.Hour)
	if err != nil || duplicate.Changed {
		t.Fatalf("duplicate activation: %v %+v", err, duplicate)
	}
	retired, err := f.Store.FindByKID(ctx, old.Kid)
	if err != nil || retired == nil || retired.ExpiresAt == nil || !retired.ExpiresAt.Equal(before.Add(7*24*time.Hour)) {
		t.Fatalf("duplicate activation changed the grace period: %v %+v", err, retired)
	}
	public, err = f.Store.ListPublicKeys(ctx, before.Add(7*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(public) != 1 || public[0].Kid != prepared.Kid {
		t.Fatalf("current key missing at old expiry: %+v", public)
	}
	for _, key := range public {
		if key.Kid == old.Kid {
			t.Fatal("old certificate remains at expiry")
		}
	}
	if _, err := f.Store.Activate(ctx, old.Kid, before, time.Hour); !errors.Is(err, signingdomain.ErrKeyNotPrepared) {
		t.Fatal("retired key can be reactivated")
	}
	archived, err := f.Store.ArchiveExpired(ctx, before.Add(7*24*time.Hour))
	if err != nil || len(archived) != 1 || archived[0].Kid != old.Kid {
		t.Fatalf("archive expired key: %v %+v", err, archived)
	}
	if archived, err := f.Store.ArchiveExpired(ctx, before.Add(8*24*time.Hour)); err != nil || len(archived) != 0 {
		t.Fatalf("duplicate archive: %v %+v", err, archived)
	}
	second, err := f.Store.Prepare(ctx, before)
	if err != nil {
		t.Fatal(err)
	}
	if disabled, err := f.Store.Disable(ctx, second.Next.Kid); err != nil || disabled == nil || disabled.Prepared {
		t.Fatalf("disabled candidate is still prepared: %v %+v", err, disabled)
	}
	if _, err := f.Store.Activate(ctx, second.Next.Kid, before, time.Hour); !errors.Is(err, signingdomain.ErrKeyNotPrepared) {
		t.Fatal("disabled candidate can be activated")
	}
	if key, err := f.Store.GetActiveKey(f.Context); err != nil || key.Kid != jwt.Kid {
		t.Fatal("XML rotation changed JWT key")
	}
	if _, err := f.Store.Rotate(otherScope, before, time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Store.Rotate(otherScope, before, time.Hour); err != nil {
		t.Fatalf("dedicated SAML key cannot rotate immediately: %v", err)
	}
}
