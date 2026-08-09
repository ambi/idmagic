package db_postgres_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/oauth2/client/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/client/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
)

func TestOAuth2ClientRepositoryListPage(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	repo := &db_postgres.OAuth2ClientRepository{Pool: db}
	ctx := context.Background()
	base := pgfixtures.TestClock()

	ids := make([]string, 5)
	for i := range ids {
		c := &domain.OAuth2Client{
			TenantID: tenant.ID, ClientID: pgfixtures.NewUUID(t), ClientType: spec.ClientConfidential,
			ClientSecretHash: new("secret-hash"), RedirectURIs: []string{"https://client.example/cb"},
			GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
			ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
			TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic, Scope: "openid offline_access",
			IDTokenSignedResponseAlg: signingdomain.SigAlgPS256, FapiProfile: domain.FapiNone,
			CreatedAt: base, UpdatedAt: base,
		}
		if err := repo.Save(ctx, c); err != nil {
			t.Fatalf("save client %d: %v", i, err)
		}
		ids[i] = c.ClientID
	}
	slices.SortFunc(ids, strings.Compare)

	first, err := repo.ListPage(ctx, tenant.ID, "", 2)
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(first) != 2 || first[0].ClientID != ids[0] || first[1].ClientID != ids[1] {
		t.Fatalf("unexpected first page: %+v", first)
	}

	last := first[len(first)-1]
	next, err := repo.ListPage(ctx, tenant.ID, last.ClientID, 2)
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(next) != 2 || next[0].ClientID != ids[2] || next[1].ClientID != ids[3] {
		t.Fatalf("unexpected continuation page: %+v", next)
	}

	all, err := repo.ListPage(ctx, tenant.ID, "", 100)
	if err != nil {
		t.Fatalf("list page all: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
}
