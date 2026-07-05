package postgres

import (
	"context"
	"testing"
	"time"

	authnports "github.com/ambi/idmagic/internal/authentication/ports"
	oauthports "github.com/ambi/idmagic/internal/oauth2/ports"
	"github.com/ambi/idmagic/internal/shared/spec"
	"github.com/ambi/idmagic/internal/tenancy"
)

func TestAuditEventRepositoryAppendAndList(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	repo := &AuditEventRepository{Pool: db}
	ctx := context.Background()

	base := testClock()
	first := &oauthports.AuditEventRecord{
		ID:         newUUID(t),
		TenantID:   tenant.ID,
		Type:       "UserAuthenticated",
		OccurredAt: base,
		Payload:    map[string]any{"userId": user.ID},
	}
	second := &oauthports.AuditEventRecord{
		ID:         newUUID(t),
		TenantID:   tenant.ID,
		Type:       "AuthenticationFailed",
		OccurredAt: base.Add(time.Minute),
		Payload:    map[string]any{"userId": user.ID},
	}
	if err := repo.Append(ctx, first); err != nil {
		t.Fatalf("append first: %v", err)
	}
	if err := repo.Append(ctx, second); err != nil {
		t.Fatalf("append second: %v", err)
	}
	// idempotent: 同一 ID の再 Append は ON CONFLICT DO NOTHING。
	if err := repo.Append(ctx, first); err != nil {
		t.Fatalf("re-append: %v", err)
	}

	all, err := repo.List(ctx, oauthports.AuditEventQuery{TenantID: tenant.ID})
	if err != nil || len(all) != 2 {
		t.Fatalf("list all: %v len=%d", err, len(all))
	}
	// occurred_at DESC 順のため、新しい second が先頭。
	if all[0].ID != second.ID {
		t.Fatalf("expected DESC ordering, got %+v", all[0])
	}

	byType, err := repo.List(ctx, oauthports.AuditEventQuery{TenantID: tenant.ID, Type: "UserAuthenticated"})
	if err != nil || len(byType) != 1 || byType[0].ID != first.ID {
		t.Fatalf("list by type: %v %+v", err, byType)
	}

	byUser, err := repo.List(ctx, oauthports.AuditEventQuery{TenantID: tenant.ID, UserID: user.ID})
	if err != nil || len(byUser) != 2 {
		t.Fatalf("list by user: %v len=%d", err, len(byUser))
	}

	byWindow, err := repo.List(ctx, oauthports.AuditEventQuery{
		TenantID: tenant.ID, After: base.Add(30 * time.Second),
	})
	if err != nil || len(byWindow) != 1 || byWindow[0].ID != second.ID {
		t.Fatalf("list by window: %v %+v", err, byWindow)
	}

	found, err := repo.FindByID(ctx, first.ID)
	if err != nil || found == nil || found.Type != "UserAuthenticated" {
		t.Fatalf("find by id: %v %+v", err, found)
	}
}

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

func TestAuthorizationDetailTypeRepositoryRoundTrip(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	repo := &AuthorizationDetailTypeRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	detailType := &spec.AuthorizationDetailType{
		TenantID:        tenant.ID,
		Type:            "payment_initiation",
		Description:     "Payment initiation details",
		DisplayTemplate: "Pay {{.amount}}",
		State:           spec.DetailTypeEnabled,
		Schema: spec.AuthorizationDetailsSchema{
			Rules: []spec.AuthorizationDetailFieldRule{
				{Name: "currency", Semantics: spec.DetailFieldEnum, Allowed: []string{"USD", "JPY"}},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Save(ctx, detailType); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.FindByType(ctx, tenant.ID, "payment_initiation")
	if err != nil || got == nil {
		t.Fatalf("find by type: %v %+v", err, got)
	}
	if got.DisplayTemplate != "Pay {{.amount}}" || got.State != spec.DetailTypeEnabled {
		t.Fatalf("unexpected detail type: %+v", got)
	}
	if len(got.Schema.Rules) != 1 || got.Schema.Rules[0].Semantics != spec.DetailFieldEnum {
		t.Fatalf("schema not round-tripped: %+v", got.Schema)
	}

	list, err := repo.ListByTenant(ctx, tenant.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list by tenant: %v len=%d", err, len(list))
	}

	if err := repo.Delete(ctx, tenant.ID, "payment_initiation"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err = repo.FindByType(ctx, tenant.ID, "payment_initiation")
	if err != nil || got != nil {
		t.Fatalf("expected deleted: %v %+v", err, got)
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

func TestOutboxEventSinkEmit(t *testing.T) {
	db := requireDB(t)
	sink := &OutboxEventSink{Pool: db}
	ctx := context.Background()

	clientID := newUUID(t)
	event := &spec.ClientRegistered{
		At:         testClock(),
		TenantID:   spec.DefaultTenantID,
		ClientID:   clientID,
		ClientType: spec.ClientConfidential,
	}
	if err := sink.Emit(ctx, event); err != nil {
		t.Fatalf("emit: %v", err)
	}

	var (
		topic     string
		eventType string
	)
	err := db.QueryRow(ctx,
		"SELECT topic,event_type FROM outbox WHERE payload->>'clientId'=$1", clientID).
		Scan(&topic, &eventType)
	if err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	if topic != "oauth2.client.lifecycle.v1" || eventType != "ClientRegistered" {
		t.Fatalf("unexpected outbox row: topic=%s type=%s", topic, eventType)
	}

	// トピック未定義のイベントは拒否される。
	if err := sink.Emit(ctx, &spec.AppAccessDeniedByPolicy{At: testClock()}); err == nil {
		t.Fatal("expected error for event without topic mapping")
	}
}

func TestKeyStoreRotateAndLookup(t *testing.T) {
	db := requireDB(t)
	// signing_keys.tenant_id は tenants(id) を参照する。NewKeyStore は default テナントの
	// active 鍵を bootstrap するため、default テナント行を用意しておく。
	now := testClock()
	defaultTenant := &spec.Tenant{
		ID:          spec.DefaultTenantID,
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
