package db_postgres_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/oauth2"
	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	clientusecases "github.com/ambi/idmagic/backend/oauth2/internal/client/usecases"
	tokenpg "github.com/ambi/idmagic/backend/oauth2/internal/token/db_postgres"
	tokendomain "github.com/ambi/idmagic/backend/oauth2/internal/token/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ambi/idmagic/backend/oauth2/internal/client/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/internal/client/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
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

//spec:covers REQ-OAUTH2-080: 実 DB に参照トークンがあっても、組み込みクライアントの削除は SQL の実行前に業務上の拒否となる。
func TestFirstPartyDeletionWithRefreshToken(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	repo := &db_postgres.OAuth2ClientRepository{Pool: db}
	client := pgfixtures.SeedClient(t, repo, tenant.ID)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
	store := &tokenpg.RefreshTokenStore{Pool: db}
	now := pgfixtures.TestClock()
	token := &tokendomain.RefreshTokenRecord{
		ID: pgfixtures.NewUUID(t), FamilyID: pgfixtures.NewUUID(t), Hash: pgfixtures.UniqueID("refresh"),
		TenantID: tenant.ID, ClientID: client.ClientID, UserID: user.ID,
		Scopes: []string{"openid"}, IssuedAt: now, ExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour),
	}
	if err := store.Save(t.Context(), token); err != nil {
		t.Fatal(err)
	}
	// 元の制約違反を再現し、トークン参照による拒否と組み込み保護の拒否を区別する。
	err := repo.Delete(t.Context(), tenant.ID, client.ClientID)
	pgerr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok || pgerr.Code != "23001" || pgerr.ConstraintName != "refresh_tokens_client_fkey" {
		t.Fatalf("delete error=%v", err)
	}
	client.FirstParty = true
	if err := repo.Save(t.Context(), client); err != nil {
		t.Fatal(err)
	}
	before, err := repo.FindByID(t.Context(), tenant.ID, client.ClientID)
	if err != nil {
		t.Fatal(err)
	}
	deletion := oauth2.NewPostgresModule(db).WithPostgresClientDeletion(db, oauth2.PostgresClientDeletionParticipants{
		DeleteBindingsInTx: idmanagement.DeleteClientBindingsInTx, QuotaInTx: tenancy.QuotaRepositoryInTx,
	}).StandaloneClientDeletion()
	ctx := tenantports.WithTenantID(t.Context(), tenant.ID)
	if err := clientusecases.DeleteAdminOAuth2Client(ctx, clientusecases.AdminOAuth2ClientDeps{ClientRepo: repo, ClientDeletion: deletion}, "admin", client.ClientID, now); !errors.Is(err, clientports.ErrFirstPartyClientProtected) {
		t.Fatalf("delete first-party client: %v", err)
	}
	stored, err := repo.FindByID(ctx, tenant.ID, client.ClientID)
	if err != nil || !reflect.DeepEqual(stored, before) {
		t.Fatalf("client=%+v err=%v", stored, err)
	}
	if storedToken, err := store.FindByHash(ctx, token.Hash); err != nil || storedToken == nil || storedToken.Revoked {
		t.Fatalf("token=%+v err=%v", storedToken, err)
	}
}
