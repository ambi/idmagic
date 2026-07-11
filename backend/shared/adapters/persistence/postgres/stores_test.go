package postgres

import (
	"context"
	"testing"
	"time"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

func TestAuthEventBucketStoreRecordListAndSweep(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	store := &AuthEventBucketStore{Pool: db}
	ctx := context.Background()

	now := testClock()
	keyHash := uniqueID("keyhash")
	first, err := store.Record(ctx, authnports.AuthEventBucketFailedLogin, tenant.ID, keyHash, now)
	if err != nil {
		t.Fatalf("record first: %v", err)
	}
	if !first.FirstInWindow || first.Bucket.Count != 1 {
		t.Fatalf("unexpected first record: %+v", first)
	}

	// now (03:04:05) と同じ 5 分窓 (03:00:00〜) に収まる時刻で 2 回目を記録する。
	second, err := store.Record(ctx, authnports.AuthEventBucketFailedLogin, tenant.ID, keyHash, now.Add(10*time.Second))
	if err != nil {
		t.Fatalf("record second: %v", err)
	}
	// 同一 5 分窓なので同じ bucket に畳み込まれ、最初の記録ではない。
	if second.FirstInWindow || second.Bucket.Count != 2 {
		t.Fatalf("unexpected second record: %+v", second)
	}

	list, err := store.List(ctx, tenant.ID, 10)
	if err != nil || len(list) != 1 || list[0].Count != 2 {
		t.Fatalf("list: %v %+v", err, list)
	}

	deleted, err := store.DeleteOlderThan(ctx, now.Add(time.Hour))
	if err != nil || deleted != 1 {
		t.Fatalf("delete older than: %v deleted=%d", err, deleted)
	}
	list, err = store.List(ctx, tenant.ID, 10)
	if err != nil || len(list) != 0 {
		t.Fatalf("expected empty after sweep: %v %+v", err, list)
	}
}

func TestEmailChangeTokenStoreSaveAndConsume(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	store := &EmailChangeTokenStore{Pool: db}
	ctx := context.Background()

	now := testClock()
	record := authnports.EmailChangeTokenRecord{
		Sub:       user.ID,
		TokenHash: uniqueID("token"),
		NewEmail:  "new@example.com",
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}
	if err := store.Save(ctx, record); err != nil {
		t.Fatalf("save: %v", err)
	}

	// 期限切れ扱いの時刻では消費できない (nil, nil)。
	expired, err := store.Consume(ctx, record.TokenHash, now.Add(2*time.Hour))
	if err != nil || expired != nil {
		t.Fatalf("expired consume should be nil: %v %+v", err, expired)
	}

	// Save は user 単位で最新 1 本のみを残すため、消費対象を再度保存する。
	if err := store.Save(ctx, record); err != nil {
		t.Fatalf("resave: %v", err)
	}
	got, err := store.Consume(ctx, record.TokenHash, now.Add(time.Minute))
	if err != nil || got == nil || got.NewEmail != "new@example.com" {
		t.Fatalf("consume: %v %+v", err, got)
	}

	// 消費済みトークンは二度と使えない。
	again, err := store.Consume(ctx, record.TokenHash, now.Add(time.Minute))
	if err != nil || again != nil {
		t.Fatalf("second consume should be nil: %v %+v", err, again)
	}
}

func TestPasswordResetTokenStoreSaveAndConsume(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	store := &PasswordResetTokenStore{Pool: db}
	ctx := context.Background()

	now := testClock()
	record := authnports.PasswordResetTokenRecord{
		Sub:       user.ID,
		TokenHash: uniqueID("reset"),
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}
	if err := store.Save(ctx, record); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := store.Consume(ctx, record.TokenHash, now.Add(time.Minute))
	if err != nil || got == nil || got.Sub != user.ID {
		t.Fatalf("consume: %v %+v", err, got)
	}

	again, err := store.Consume(ctx, record.TokenHash, now.Add(time.Minute))
	if err != nil || again != nil {
		t.Fatalf("second consume should be nil: %v %+v", err, again)
	}
}

func TestTenantUserAttributeSchemaRepositoryRoundTrip(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	repo := &TenantUserAttributeSchemaRepository{Pool: db}
	ctx := context.Background()

	if got, err := repo.FindByTenant(ctx, tenant.ID); err != nil || got != nil {
		t.Fatalf("expected no schema initially: %v %+v", err, got)
	}

	now := testClock()
	schema := &spec.TenantUserAttributeSchema{
		TenantID: tenant.ID,
		Attributes: []spec.UserAttributeDef{
			{Key: "department", Label: "Department", Type: spec.AttributeTypeString},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Save(ctx, schema); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.FindByTenant(ctx, tenant.ID)
	if err != nil || got == nil || len(got.Attributes) != 1 {
		t.Fatalf("find by tenant: %v %+v", err, got)
	}
	if got.Attributes[0].Key != "department" {
		t.Fatalf("attributes not round-tripped: %+v", got.Attributes)
	}

	if err := repo.Delete(ctx, tenant.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err = repo.FindByTenant(ctx, tenant.ID)
	if err != nil || got != nil {
		t.Fatalf("expected deleted: %v %+v", err, got)
	}
}

func TestKeyStoreRotateAndLookup(t *testing.T) {
	db := requireDB(t)
	// signing_keys.tenant_id は tenants(id) を参照する。NewKeyStore は default テナントの
	// active 鍵を bootstrap するため、default テナント行を用意しておく。
	now := testClock()
	defaultTenant := &spec.Tenant{
		ID:          spec.DefaultTenantID,
		Realm:       spec.DefaultRealm,
		DisplayName: "Default",
		Status:      spec.TenantStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := (&TenantRepository{Pool: db}).Save(context.Background(), defaultTenant); err != nil {
		t.Fatalf("seed default tenant: %v", err)
	}
	ctx := tenancy.WithTenant(context.Background(), defaultTenant, "", "")

	store, err := NewKeyStore(ctx, db)
	if err != nil {
		t.Fatalf("new key store: %v", err)
	}

	active, err := store.GetActiveKey(ctx)
	if err != nil || active == nil || !active.Active {
		t.Fatalf("get active key: %v %+v", err, active)
	}

	found, err := store.FindByKID(ctx, active.Kid)
	if err != nil || found == nil || found.Kid != active.Kid {
		t.Fatalf("find by kid: %v %+v", err, found)
	}

	rotated, err := store.Rotate(ctx)
	if err != nil || rotated == nil || rotated.Kid == active.Kid {
		t.Fatalf("rotate: %v %+v (prev %s)", err, rotated, active.Kid)
	}

	newActive, err := store.GetActiveKey(ctx)
	if err != nil || newActive == nil || newActive.Kid != rotated.Kid {
		t.Fatalf("active after rotate: %v %+v", err, newActive)
	}

	all, err := store.GetAllKeys(ctx)
	if err != nil || len(all) < 2 {
		t.Fatalf("get all keys: %v len=%d", err, len(all))
	}
}
