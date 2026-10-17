package db_postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/saml"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfeddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	"github.com/jackc/pgx/v5"
)

func testProtocolLinkers() ProtocolLinkers {
	return ProtocolLinkers{OIDC: oauth2.LinkApplicationInTx, SAML: saml.LinkApplicationInTx, WsFed: wsfederation.LinkApplicationInTx}
}

func seedProtocolSetting(t *testing.T, db sharedpg.DB, tenantID string, kind domain.ApplicationProtocolType) (*domain.ApplicationProtocol, func() string) {
	t.Helper()
	ctx := context.Background()
	now := pgfixtures.TestClock()
	key := pgfixtures.UniqueID("protocol")
	switch kind {
	case domain.ApplicationProtocolOIDC:
		repo := oauth2.NewPostgresModule(db).ClientRepo
		client := pgfixtures.SeedClient(t, repo, tenantID)
		return &domain.ApplicationProtocol{Type: kind, ClientID: client.ClientID}, func() string {
			stored, err := repo.FindByID(ctx, tenantID, client.ClientID)
			if err != nil || stored == nil {
				t.Fatalf("client after linking: %+v, %v", stored, err)
			}
			return stored.ApplicationID
		}
	case domain.ApplicationProtocolSAML:
		repo := saml.NewPostgresModule(db).SPRepo
		if err := repo.Save(ctx, &samldomain.SamlServiceProvider{
			TenantID: tenantID, EntityID: key, ACSURLs: []string{"https://sp.example/acs"}, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
		return &domain.ApplicationProtocol{Type: kind, EntityID: key}, func() string {
			stored, err := repo.FindByEntityID(ctx, tenantID, key)
			if err != nil || stored == nil {
				t.Fatalf("SP after linking: %+v, %v", stored, err)
			}
			return stored.ApplicationID
		}
	case domain.ApplicationProtocolWsFed:
		repo := wsfederation.NewPostgresModule(db).RPRepo
		if err := repo.Save(ctx, &wsfeddomain.WsFedRelyingParty{
			TenantID: tenantID, Wtrealm: key, ReplyURLs: []string{"https://rp.example/reply"}, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
		return &domain.ApplicationProtocol{Type: kind, Wtrealm: key}, func() string {
			stored, err := repo.FindByWtrealm(ctx, tenantID, key)
			if err != nil || stored == nil {
				t.Fatalf("RP after linking: %+v, %v", stored, err)
			}
			return stored.ApplicationID
		}
	default:
		t.Fatalf("unknown protocol %q", kind)
		return nil, nil
	}
}

//spec:covers REQ-APPLICATION-007: 三つのプロトコルの所有者が同一 Tx で結び付け、更新後の失敗では両側を巻き戻す。
func TestProtocolOwnersJoinTheApplicationTransaction(t *testing.T) {
	for _, kind := range []domain.ApplicationProtocolType{
		domain.ApplicationProtocolOIDC, domain.ApplicationProtocolSAML, domain.ApplicationProtocolWsFed,
	} {
		for _, failAfterWrite := range []bool{false, true} {
			name := string(kind) + "/commit"
			if failAfterWrite {
				name = string(kind) + "/rollback"
			}
			t.Run(name, func(t *testing.T) {
				db := pgtest.Require(t)
				tenant := pgfixtures.SeedTenant(t, db)
				protocol, linkedApplication := seedProtocolSetting(t, db, tenant.ID, kind)
				now := pgfixtures.TestClock()
				app := &domain.Application{
					ID: pgfixtures.NewUUID(t), TenantID: tenant.ID, Name: pgfixtures.UniqueID("linked"),
					Kind: domain.ApplicationFederated, Status: domain.ApplicationActive, Protocol: protocol,
					CategoryIDs: []string{}, CreatedAt: now, UpdatedAt: now,
				}
				linkers := testProtocolLinkers()
				injected := errors.New("failure after the owner updated the protocol")
				if failAfterWrite {
					wrap := func(link func(context.Context, pgx.Tx, string, string, string) error) func(context.Context, pgx.Tx, string, string, string) error {
						return func(ctx context.Context, tx pgx.Tx, tenantID, applicationID, key string) error {
							if err := link(ctx, tx, tenantID, applicationID, key); err != nil {
								return err
							}
							return injected
						}
					}
					linkers = ProtocolLinkers{OIDC: wrap(linkers.OIDC), SAML: wrap(linkers.SAML), WsFed: wrap(linkers.WsFed)}
				}
				repo := NewApplicationRepository(db, linkers)
				err := repo.Create(context.Background(), app)
				stored, readErr := repo.FindByID(context.Background(), tenant.ID, app.ID)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if failAfterWrite {
					if !errors.Is(err, injected) || stored != nil || linkedApplication() != "" {
						t.Fatalf("rollback: error=%v application=%+v protocol application=%q", err, stored, linkedApplication())
					}
				} else if err != nil || stored == nil || linkedApplication() != app.ID {
					t.Fatalf("commit: error=%v application=%+v protocol application=%q, want %s", err, stored, linkedApplication(), app.ID)
				}
			})
		}
	}
}

//spec:covers REQ-APPLICATION-007: プロトコル設定を結び付けられない作成は Application を残さない。
func TestApplicationCreateRollsBackWhenTheProtocolCannotBeLinked(t *testing.T) {
	for _, kind := range []domain.ApplicationProtocolType{
		domain.ApplicationProtocolOIDC, domain.ApplicationProtocolSAML, domain.ApplicationProtocolWsFed,
	} {
		t.Run(string(kind), func(t *testing.T) {
			db := pgtest.Require(t)
			tenant := pgfixtures.SeedTenant(t, db)
			now := pgfixtures.TestClock()
			app := &domain.Application{
				ID: pgfixtures.NewUUID(t), TenantID: tenant.ID, Name: pgfixtures.UniqueID("missing-protocol"),
				Kind: domain.ApplicationFederated, Status: domain.ApplicationActive,
				CategoryIDs: []string{}, CreatedAt: now, UpdatedAt: now,
				Protocol: &domain.ApplicationProtocol{Type: kind, ClientID: "missing", EntityID: "missing", Wtrealm: "missing"},
			}
			repo := NewApplicationRepository(db, testProtocolLinkers())
			if err := repo.Create(context.Background(), app); err == nil {
				t.Fatal("create without a protocol setting succeeded")
			}
			stored, err := repo.FindByID(context.Background(), tenant.ID, app.ID)
			if err != nil || stored != nil {
				t.Fatalf("application after failed create = (%+v, %v), want no application", stored, err)
			}
		})
	}
}
