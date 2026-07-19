package postgres

import (
	"context"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmpg "github.com/ambi/idmagic/backend/idmanagement/adapters/persistence/postgres"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancypg "github.com/ambi/idmagic/backend/tenancy/adapters/persistence/postgres"
)

// testClock は決定的なタイムスタンプ生成に用いる基準時刻。
func testClock() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

// newUUID は UUID 列向けの一意な UUID を生成する。
func newUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("new uuid: %v", err)
	}
	return id
}

// seedTenant / seedUser は pgfixtures を使わず自前で用意する。pgfixtures は本パッケージ
// (oauthpg) を import しており、本パッケージの内部テストから pgfixtures を使うと
// postgres -> pgfixtures -> postgres の import cycle になるため (ADR-090, wi-172 と同じ制約)。

func seedTenant(t *testing.T, db sharedpg.DB) *tenancydomain.Tenant {
	t.Helper()
	now := testClock()
	tenant := &tenancydomain.Tenant{
		ID:          newUUID(t),
		Realm:       "tenant-" + newUUID(t)[:8],
		DisplayName: "Test Tenant",
		Status:      tenancydomain.TenantStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := (&tenancypg.TenantRepository{Pool: db}).Save(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenant
}

func seedUser(t *testing.T, db sharedpg.DB, tenantID string) *userdomain.User {
	t.Helper()
	now := testClock()
	user := &userdomain.User{
		ID:                newUUID(t),
		TenantID:          tenantID,
		PreferredUsername: "user-" + newUUID(t)[:8],
		PasswordHash:      "hash",
		Roles:             []string{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := (&idmpg.UserRepository{Pool: db}).Save(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

func seedClient(t *testing.T, db sharedpg.DB, tenantID string) *domain.OAuth2Client {
	t.Helper()
	now := testClock()
	client := &domain.OAuth2Client{
		TenantID:                 tenantID,
		ClientID:                 newUUID(t),
		ClientType:               spec.ClientConfidential,
		RedirectURIs:             []string{"https://client.example/cb"},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    "openid offline_access",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if err := (&OAuth2ClientRepository{Pool: db}).Save(context.Background(), client); err != nil {
		t.Fatalf("seed client: %v", err)
	}
	return client
}

func TestOAuth2ClientRepositoryRoundTrip(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	repo := &OAuth2ClientRepository{Pool: db}
	ctx := context.Background()

	client := seedClient(t, db, tenant.ID)

	got, err := repo.FindByID(ctx, tenant.ID, client.ClientID)
	if err != nil || got == nil {
		t.Fatalf("find: %v, %+v", err, got)
	}
	if got.Scope != "openid offline_access" || got.ClientType != spec.ClientConfidential {
		t.Fatalf("unexpected client: %+v", got)
	}
	if len(got.RedirectURIs) != 1 || got.RedirectURIs[0] != "https://client.example/cb" {
		t.Fatalf("redirect uris not round-tripped: %+v", got.RedirectURIs)
	}

	all, err := repo.FindAll(ctx, tenant.ID)
	if err != nil || len(all) != 1 {
		t.Fatalf("find all: %v len=%d", err, len(all))
	}

	if err := repo.Delete(ctx, tenant.ID, client.ClientID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err = repo.FindByID(ctx, tenant.ID, client.ClientID)
	if err != nil || got != nil {
		t.Fatalf("expected deleted, got %v %+v", err, got)
	}
}

func TestConsentRepositoryRoundTrip(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	client := seedClient(t, db, tenant.ID)
	repo := &ConsentRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	consent := &domain.Consent{
		UserID:   user.ID,
		ClientID: client.ClientID,
		Scopes:   []string{"openid", "profile"},
		// State は Find 時に実時刻 (time.Now) で導出されるため、有効期限は実時刻基準で未来に置く。
		GrantedAt: now,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := repo.Save(ctx, tenant.ID, consent); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.Find(ctx, tenant.ID, user.ID, client.ClientID)
	if err != nil || got == nil {
		t.Fatalf("find: %v %+v", err, got)
	}
	if got.State != domain.ConsentGranted || len(got.Scopes) != 2 {
		t.Fatalf("unexpected consent: %+v", got)
	}

	all, err := repo.FindAll(ctx, tenant.ID)
	if err != nil || len(all) != 1 {
		t.Fatalf("find all: %v len=%d", err, len(all))
	}

	if err := repo.Revoke(ctx, tenant.ID, user.ID, client.ClientID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	got, err = repo.Find(ctx, tenant.ID, user.ID, client.ClientID)
	if err != nil || got == nil || got.State != domain.ConsentRevoked {
		t.Fatalf("expected revoked: %v %+v", err, got)
	}

	if err := repo.DeleteAllForSub(ctx, user.ID); err != nil {
		t.Fatalf("delete all: %v", err)
	}
	got, err = repo.Find(ctx, tenant.ID, user.ID, client.ClientID)
	if err != nil || got != nil {
		t.Fatalf("expected gone: %v %+v", err, got)
	}
}

func TestAuthorizationDetailTypeRepositoryRoundTrip(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	repo := &AuthorizationDetailTypeRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	detailType := &domain.AuthorizationDetailType{
		TenantID:        tenant.ID,
		Type:            "payment_initiation",
		Description:     "Payment initiation details",
		DisplayTemplate: "Pay {{.amount}}",
		State:           domain.DetailTypeEnabled,
		Schema: domain.AuthorizationDetailsSchema{
			Rules: []domain.AuthorizationDetailFieldRule{
				{Name: "currency", Semantics: domain.DetailFieldEnum, Allowed: []string{"USD", "JPY"}},
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
	if got.DisplayTemplate != "Pay {{.amount}}" || got.State != domain.DetailTypeEnabled {
		t.Fatalf("unexpected detail type: %+v", got)
	}
	if len(got.Schema.Rules) != 1 || got.Schema.Rules[0].Semantics != domain.DetailFieldEnum {
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

func TestMcpResourceServerRepositoryRoundTrip(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	repo := &McpResourceServerRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	resourceServerID := newUUID(t)
	m := &domain.McpResourceServer{
		TenantID:         tenant.ID,
		ResourceServerID: resourceServerID,
		Resource:         "https://mcp.example.com/tools/github",
		Name:             "GitHub MCP Tools",
		Scopes:           []string{"mcp.read", "mcp.write"},
		State:            domain.McpResourceServerActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := repo.Save(ctx, m); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.FindByID(ctx, tenant.ID, resourceServerID)
	if err != nil || got == nil {
		t.Fatalf("find by id: %v %+v", err, got)
	}
	if got.Resource != m.Resource || got.State != domain.McpResourceServerActive {
		t.Fatalf("unexpected resource server: %+v", got)
	}
	if len(got.Scopes) != 2 {
		t.Fatalf("scopes not round-tripped: %+v", got.Scopes)
	}

	byResource, err := repo.FindByResource(ctx, tenant.ID, "https://mcp.example.com/tools/github")
	if err != nil || byResource == nil || byResource.ResourceServerID != resourceServerID {
		t.Fatalf("find by resource: %v %+v", err, byResource)
	}

	list, err := repo.ListByTenant(ctx, tenant.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list by tenant: %v len=%d", err, len(list))
	}

	if err := repo.Delete(ctx, tenant.ID, resourceServerID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err = repo.FindByID(ctx, tenant.ID, resourceServerID)
	if err != nil || got != nil {
		t.Fatalf("expected deleted: %v %+v", err, got)
	}
}

// ADR-055/wi-262: resource indicator (RFC 8707) の audience 束縛が Postgres 永続化を
// 跨いで保持されることを確認する。
func TestRefreshTokenStoreRoundTrip_PreservesResource(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	client := seedClient(t, db, tenant.ID)
	store := &RefreshTokenStore{Pool: db}
	ctx := context.Background()

	now := testClock()
	resource := "https://mcp.example.com/tools/github"
	gen, err := domain.GenerateInitialRefreshToken(client.ClientID, user.ID, []string{"openid", "offline_access"}, nil, nil, &resource, now)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	gen.Record.TenantID = tenant.ID
	if err := store.Save(ctx, gen.Record); err != nil {
		t.Fatalf("save: %v", err)
	}

	found, err := store.FindByHash(ctx, gen.Record.Hash)
	if err != nil || found == nil {
		t.Fatalf("find by hash: %v %+v", err, found)
	}
	if found.Resource == nil || *found.Resource != resource {
		t.Fatalf("expected resource to round-trip, got %v", found.Resource)
	}

	rotated, err := domain.RotateRefreshToken(found, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	rotated.Record.TenantID = tenant.ID
	if _, err := store.Rotate(ctx, found.ID, rotated.Record); err != nil {
		t.Fatalf("store rotate: %v", err)
	}

	afterRotation, err := store.FindByHash(ctx, rotated.Record.Hash)
	if err != nil || afterRotation == nil {
		t.Fatalf("find rotated by hash: %v %+v", err, afterRotation)
	}
	if afterRotation.Resource == nil || *afterRotation.Resource != resource {
		t.Fatalf("expected resource to survive rotation round-trip, got %v", afterRotation.Resource)
	}
}
