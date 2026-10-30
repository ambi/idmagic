package db_postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/saml/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSamlIdentityProviderProfileRepository(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	ctx := context.Background()
	profiles := &SamlIdentityProviderProfileRepository{Pool: db}
	serviceProviders := &SamlServiceProviderRepository{Pool: db}

	defaultProfile, err := profiles.EnsureDefaultIDPProfile(ctx, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !defaultProfile.IsDefault || defaultProfile.ProfileID != domain.DefaultIDPProfileID {
		t.Fatalf("default profile = %+v", defaultProfile)
	}

	dedicated := &domain.SamlIdentityProviderProfile{
		TenantID: tenant.ID, ProfileID: "dedicated-a", Name: "Dedicated A", Mode: domain.IDPProfileModeDedicated,
	}
	if err := profiles.SaveIDPProfile(ctx, dedicated); err != nil {
		t.Fatal(err)
	}
	if err := serviceProviders.Save(ctx, &domain.SamlServiceProvider{
		TenantID: tenant.ID, EntityID: "urn:sp:a", IDPProfileID: dedicated.ProfileID,
	}); err != nil {
		t.Fatal(err)
	}
	if err := serviceProviders.Save(ctx, &domain.SamlServiceProvider{
		TenantID: tenant.ID, EntityID: "urn:sp:b", IDPProfileID: dedicated.ProfileID,
	}); !errors.Is(err, domain.ErrDedicatedIDPProfileCardinality) {
		t.Fatalf("second dedicated binding error = %v", err)
	}
	if err := profiles.DeleteIDPProfile(ctx, tenant.ID, dedicated.ProfileID); !errors.Is(err, domain.ErrIDPProfileInUse) {
		t.Fatalf("delete in-use profile error = %v", err)
	}
	if err := profiles.DeleteIDPProfile(ctx, tenant.ID, domain.DefaultIDPProfileID); !errors.Is(err, domain.ErrDefaultIDPProfile) {
		t.Fatalf("delete default profile error = %v", err)
	}

	list, err := profiles.ListIDPProfilesByTenant(ctx, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || !list[0].IsDefault {
		t.Fatalf("profiles = %+v", list)
	}
}

// 別のトランザクションが作成の途中で保存を重ねる。ロック待ちを観測してから確定するので、
// タイミングに依らず同じ順序になる。
//
//spec:covers REQ-SAML-004: 同時に作成された shared のプロファイルに SP が二つ関連付いていれば、dedicated への保存は確定後の件数で拒否され、プロファイルは shared のまま残る。
func TestSaveIDPProfileSeesConcurrentCreationBeforeValidating(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	ctx := context.Background()
	profiles := &SamlIdentityProviderProfileRepository{Pool: db}
	now := pgtest.Now()

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	if err := q.UpsertSamlIDPProfile(ctx, UpsertSamlIDPProfileParams{
		TenantID: tenant.ID, ProfileID: "concurrent", Name: "Concurrent", Mode: string(domain.IDPProfileModeShared), CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	for _, entityID := range []string{"urn:sp:a", "urn:sp:b"} {
		if err := q.UpsertSamlServiceProvider(ctx, UpsertSamlServiceProviderParams{
			TenantID: tenant.ID, EntityID: entityID, IdpProfileID: "concurrent", AcsUrls: []byte("[]"), ClaimPolicy: []byte("{}"), CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	saved := make(chan error, 1)
	go func() {
		saved <- profiles.SaveIDPProfile(ctx, &domain.SamlIdentityProviderProfile{
			TenantID: tenant.ID, ProfileID: "concurrent", Name: "Dedicated", Mode: domain.IDPProfileModeDedicated,
		})
	}()
	waitForBlockedInsert(t, db)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	if err := <-saved; !errors.Is(err, domain.ErrDedicatedIDPProfileCardinality) {
		t.Fatalf("save error = %v, want ErrDedicatedIDPProfileCardinality", err)
	}
	stored, err := profiles.FindIDPProfileByID(ctx, tenant.ID, "concurrent")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Mode != domain.IDPProfileModeShared {
		t.Fatalf("mode = %q, want shared", stored.Mode)
	}
}

func waitForBlockedInsert(t *testing.T, db *pgxpool.Pool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var blocked int
		if err := db.QueryRow(context.Background(), `SELECT count(*) FROM pg_stat_activity
			WHERE wait_event_type = 'Lock' AND query LIKE '%saml_identity_provider_profiles%'`).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the save never blocked on the uncommitted creation")
}
