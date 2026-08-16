package db_postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func serviceProvider(tenantID, entityID string) *samldomain.SamlServiceProvider {
	now := time.Now().UTC()
	return &samldomain.SamlServiceProvider{
		TenantID:     tenantID,
		EntityID:     entityID,
		IDPProfileID: samldomain.DefaultIDPProfileID,
		ACSURLs:      []string{"https://sp.example.com/acs"},
		ClaimPolicy: claimdomain.ClaimMappingPolicy{
			NameID: claimdomain.NameIdConfiguration{
				Format:          samldomain.SamlNameIDFormatPersistent,
				SourceAttribute: "user_id",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// domain が受ける値は DB も受ける。上限が 2 つの境界で食い違うと、Go を通った
// 書き込みが CHECK で落ちる区間ができる。
func TestSamlServiceProviderAcceptsEntityIDAtTheCeiling(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	repo := &SamlServiceProviderRepository{Pool: db}

	const prefix = "https://sp.example.com/"
	sp := serviceProvider(tenant.ID, prefix+strings.Repeat("a", spec.LengthSamlEntityID-len(prefix)))
	if err := sp.Validate(); err != nil {
		t.Fatalf("domain rejected a value at the ceiling: %v", err)
	}
	if err := repo.Save(context.Background(), sp); err != nil {
		t.Fatalf("database rejected a value the domain accepted: %v", err)
	}
}

// domain を迂回した超過は CHECK が止める。btree の索引行上限まで到達させない。
func TestSamlServiceProviderCheckStopsOversizedEntityID(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	repo := &SamlServiceProviderRepository{Pool: db}

	sp := serviceProvider(tenant.ID, strings.Repeat("a", spec.BytesSamlEntityID+1))
	err := repo.Save(context.Background(), sp)
	if err == nil {
		t.Fatal("oversized entity id was stored")
	}
	if !strings.Contains(err.Error(), "saml_service_providers_entity_id_length") {
		t.Fatalf("expected the length CHECK to reject the write, got: %v", err)
	}
}

// 多バイトの値は、契約の上限 (コードポイント) の内側でもバイトの上限で止まる。
// 止めなければ btree が SQLSTATE 54000 を返す。
func TestSamlServiceProviderCheckStopsMultibyteEntityIDOverTheByteCeiling(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	repo := &SamlServiceProviderRepository{Pool: db}

	sp := serviceProvider(tenant.ID, strings.Repeat("あ", 1024)) // 1024 code points / 3072 bytes
	err := repo.Save(context.Background(), sp)
	if err == nil {
		t.Fatal("entity id of 3072 bytes was stored")
	}
	if !strings.Contains(err.Error(), "saml_service_providers_entity_id_length") {
		t.Fatalf("expected the length CHECK, not a btree index row error, got: %v", err)
	}
}
