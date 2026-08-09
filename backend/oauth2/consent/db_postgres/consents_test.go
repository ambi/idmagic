package db_postgres

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestConsentRepositoryListPageByTenant(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	repo := &ConsentRepository{Pool: db}
	ctx := context.Background()
	now := pgfixtures.TestClock()

	client := pgfixtures.SeedClient(t, db, tenant.ID)
	userIDs := make([]string, 0, 5)
	for range 5 {
		u := pgfixtures.SeedUser(t, db, tenant.ID)
		userIDs = append(userIDs, u.ID)
		if err := repo.Save(ctx, tenant.ID, &domain.Consent{
			UserID: u.ID, ClientID: client.ClientID, Scopes: []string{"read"},
			GrantedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		}); err != nil {
			t.Fatalf("save consent: %v", err)
		}
	}
	slices.SortFunc(userIDs, strings.Compare)

	first, err := repo.ListPage(ctx, tenant.ID, "", "", 2)
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(first) != 2 || first[0].UserID != userIDs[0] || first[1].UserID != userIDs[1] {
		t.Fatalf("unexpected first page: %+v", first)
	}

	last := first[len(first)-1]
	next, err := repo.ListPage(ctx, tenant.ID, last.UserID, last.ClientID, 2)
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(next) != 2 || next[0].UserID != userIDs[2] || next[1].UserID != userIDs[3] {
		t.Fatalf("unexpected continuation page: %+v", next)
	}

	all, err := repo.ListPage(ctx, tenant.ID, "", "", 100)
	if err != nil {
		t.Fatalf("list page all: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
}
