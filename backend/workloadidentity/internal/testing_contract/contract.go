// Package testing_contract defines the shared workload-identity persistence contract.
package testing_contract

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/ports"
)

type Fixture struct {
	TrustBundles ports.WorkloadTrustBundleRepository
	TenantA      string
	TenantB      string
	BundleID     string
	OtherID      string
	// SPIFFEIDs は spiffe の検査に使う、BundleID と OtherID 以外の三つの識別子。
	SPIFFEIDs [3]string
	Now       time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	bundle := &domain.WorkloadTrustBundle{
		ID: f.BundleID, TenantID: f.TenantA, Name: "Production", Profile: domain.WorkloadTrustBundleProfileOIDC, TrustDomain: "example.test",
		Issuer: "https://issuer.example.test", JWKS: map[string]any{"keys": []any{}},
		AcceptedAudiences: []string{"idmagic"}, MaxSubjectTokenTTLSeconds: 300,
		Status: domain.WorkloadTrustBundleStatusEnabled, CreatedAt: f.Now,
	}
	if err := f.TrustBundles.Save(ctx, bundle); err != nil {
		t.Fatalf("Save: %v", err)
	}
	found, err := f.TrustBundles.FindByID(ctx, f.TenantA, f.BundleID)
	if err != nil || found == nil || found.Issuer != bundle.Issuer || len(found.AcceptedAudiences) != 1 {
		t.Fatalf("FindByID = (%+v, %v)", found, err)
	}
	byIssuer, err := f.TrustBundles.FindByIssuer(ctx, f.TenantA, bundle.Issuer)
	if err != nil || byIssuer == nil || byIssuer.ID != f.BundleID {
		t.Fatalf("FindByIssuer = (%+v, %v)", byIssuer, err)
	}
	if leaked, err := f.TrustBundles.FindByID(ctx, f.TenantB, f.BundleID); err != nil || leaked != nil {
		t.Fatalf("other tenant FindByID = (%+v, %v)", leaked, err)
	}
	listed, err := f.TrustBundles.ListAll(ctx, f.TenantA)
	if err != nil || len(listed) != 1 || listed[0].ID != f.BundleID {
		t.Fatalf("ListAll = (%+v, %v)", listed, err)
	}
	duplicate := *bundle
	duplicate.ID = f.OtherID
	if err := f.TrustBundles.Save(ctx, &duplicate); err == nil {
		t.Fatal("duplicate issuer within one tenant was accepted")
	}
	otherTenant := *bundle
	otherTenant.ID, otherTenant.TenantID = f.OtherID, f.TenantB
	if err := f.TrustBundles.Save(ctx, &otherTenant); err != nil {
		t.Fatalf("same issuer in another tenant: %v", err)
	}
	// 取得結果の記録は既存の設定の一列だけを書き、テナントの境界を越えない。
	before, err := f.TrustBundles.FindByID(ctx, f.TenantA, f.BundleID)
	if err != nil || before == nil {
		t.Fatalf("before fetch = (%+v, %v)", before, err)
	}
	at := f.Now.Add(time.Minute)
	fetched := domain.JWKSSnapshot{Keys: []map[string]any{{"kty": "EC", "kid": "fetched", "crv": "P-256"}}, FetchedAt: at}
	if err := f.TrustBundles.RecordJWKSFetched(ctx, f.TenantB, f.BundleID, nil, fetched); err != nil {
		t.Fatalf("other tenant RecordJWKSFetched: %v", err)
	}
	unchanged, err := f.TrustBundles.FindByID(ctx, f.TenantA, f.BundleID)
	if err != nil || !reflect.DeepEqual(unchanged, before) {
		t.Fatalf("other tenant changed bundle = (%+v, %v), want %+v", unchanged, err, before)
	}
	if err := f.TrustBundles.RecordJWKSFetched(ctx, f.TenantA, f.BundleID, nil, fetched); err != nil {
		t.Fatalf("RecordJWKSFetched: %v", err)
	}
	after, err := f.TrustBundles.FindByID(ctx, f.TenantA, f.BundleID)
	if err != nil || after == nil || after.JWKSCachedAt == nil || !after.JWKSCachedAt.Equal(at) {
		t.Fatalf("after fetch = (%+v, %v), want fetch time %s", after, err, at)
	}
	before.JWKSCachedAt, before.JWKSCached = after.JWKSCachedAt, fetched.Keys
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("after fetch = (%+v, %v), want only the fetched keys and time changed: %+v", after, err, before)
	}
	runJWKSStoreConditions(t, f, after)
	if err := f.TrustBundles.Delete(ctx, f.TenantA, f.BundleID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := f.TrustBundles.RecordJWKSFetched(ctx, f.TenantA, f.BundleID, nil, fetched); err != nil {
		t.Fatalf("deleted RecordJWKSFetched: %v", err)
	}
	if deleted, err := f.TrustBundles.FindByID(ctx, f.TenantA, f.BundleID); err != nil || deleted != nil {
		t.Fatalf("FindByID after Delete = (%+v, %v)", deleted, err)
	}
	runSPIFFE(t, f)
}

// runJWKSStoreConditions は、最後に取得できた鍵の保存の条件を確かめる (REQ-WORKLOADIDENTITY-018)。
// stored は鍵を保存済みの、jwks_uri を持たない信頼設定である。
//   - 保存済みより古い取得は書かず、別の取得元からの取得も書かない。
//   - jwks_uri を変えない更新は、読んだときの値で保存した鍵を戻さない。
//   - jwks_uri を変える更新は、保存した鍵と時刻を消す。
func runJWKSStoreConditions(t *testing.T, f Fixture, stored *domain.WorkloadTrustBundle) {
	t.Helper()
	ctx := context.Background()
	reload := func() *domain.WorkloadTrustBundle {
		t.Helper()
		b, err := f.TrustBundles.FindByID(ctx, f.TenantA, f.BundleID)
		if err != nil || b == nil {
			t.Fatalf("FindByID = (%+v, %v)", b, err)
		}
		return b
	}
	staleRead := *stored
	stale := domain.JWKSSnapshot{Keys: []map[string]any{{"kid": "older"}}, FetchedAt: stored.JWKSCachedAt.Add(-time.Second)}
	if err := f.TrustBundles.RecordJWKSFetched(ctx, f.TenantA, f.BundleID, nil, stale); err != nil {
		t.Fatalf("older RecordJWKSFetched: %v", err)
	}
	otherSource := "https://other-source.example.test/jwks"
	foreign := domain.JWKSSnapshot{Keys: []map[string]any{{"kid": "foreign"}}, FetchedAt: stored.JWKSCachedAt.Add(time.Hour)}
	if err := f.TrustBundles.RecordJWKSFetched(ctx, f.TenantA, f.BundleID, &otherSource, foreign); err != nil {
		t.Fatalf("other source RecordJWKSFetched: %v", err)
	}
	if got := reload(); !reflect.DeepEqual(got, stored) {
		t.Fatalf("older or foreign fetch changed the stored keys = %+v, want %+v", got, stored)
	}

	newer := domain.JWKSSnapshot{Keys: []map[string]any{{"kid": "newer"}}, FetchedAt: stored.JWKSCachedAt.Add(time.Minute)}
	if err := f.TrustBundles.RecordJWKSFetched(ctx, f.TenantA, f.BundleID, nil, newer); err != nil {
		t.Fatalf("newer RecordJWKSFetched: %v", err)
	}
	staleRead.Name = "Renamed"
	if err := f.TrustBundles.Save(ctx, &staleRead); err != nil {
		t.Fatalf("Save without a source change: %v", err)
	}
	if got := reload(); got.Name != "Renamed" || got.JWKSCachedAt == nil || !got.JWKSCachedAt.Equal(newer.FetchedAt) || !reflect.DeepEqual(got.JWKSCached, newer.Keys) {
		t.Fatalf("Save without a source change = %+v, want the newer stored keys kept", got)
	}

	moved := *reload()
	moved.JWKSURI = &otherSource
	if err := f.TrustBundles.Save(ctx, &moved); err != nil {
		t.Fatalf("Save with a source change: %v", err)
	}
	if got := reload(); got.JWKSCachedAt != nil || got.JWKSCached != nil {
		t.Fatalf("Save with a source change = %+v, want the stored keys cleared", got)
	}
	if err := f.TrustBundles.RecordJWKSFetched(ctx, f.TenantA, f.BundleID, &otherSource, foreign); err != nil {
		t.Fatalf("RecordJWKSFetched from the new source: %v", err)
	}
	if got := reload(); got.JWKSCachedAt == nil || !got.JWKSCachedAt.Equal(foreign.FetchedAt) || !reflect.DeepEqual(got.JWKSCached, foreign.Keys) {
		t.Fatalf("fetch from the new source = %+v, want it stored", got)
	}
}

// runSPIFFE は spiffe の信頼設定の保存と探索を確かめる。発行者のない行を保存して読み戻せ、
// トラストドメインで探せ、同じテナントの spiffe の間だけトラストドメインが一意である。
func runSPIFFE(t *testing.T, f Fixture) {
	t.Helper()
	ctx := context.Background()
	oidc := &domain.WorkloadTrustBundle{
		ID: f.SPIFFEIDs[0], TenantID: f.TenantA, Name: "OIDC with the same label",
		Profile: domain.WorkloadTrustBundleProfileOIDC, TrustDomain: "spiffe.example.test",
		Issuer: "https://oidc.example.test", JWKS: map[string]any{"keys": []any{}},
		AcceptedAudiences: []string{"idmagic"}, MaxSubjectTokenTTLSeconds: 300,
		Status: domain.WorkloadTrustBundleStatusEnabled, CreatedAt: f.Now,
	}
	if err := f.TrustBundles.Save(ctx, oidc); err != nil {
		t.Fatalf("Save oidc: %v", err)
	}
	if found, err := f.TrustBundles.FindSPIFFEByTrustDomain(ctx, f.TenantA, "spiffe.example.test"); err != nil || found != nil {
		t.Fatalf("oidc bundle found by trust domain = (%+v, %v)", found, err)
	}
	spiffe := &domain.WorkloadTrustBundle{
		ID: f.SPIFFEIDs[1], TenantID: f.TenantA, Name: "SPIRE",
		Profile: domain.WorkloadTrustBundleProfileSPIFFE, TrustDomain: "spiffe.example.test",
		JWKS: map[string]any{"keys": []any{}}, AcceptedAudiences: []string{"idmagic"},
		MaxSubjectTokenTTLSeconds: 300, Status: domain.WorkloadTrustBundleStatusEnabled, CreatedAt: f.Now,
	}
	if err := f.TrustBundles.Save(ctx, spiffe); err != nil {
		t.Fatalf("Save spiffe without issuer: %v", err)
	}
	found, err := f.TrustBundles.FindSPIFFEByTrustDomain(ctx, f.TenantA, "spiffe.example.test")
	if err != nil || found == nil || found.ID != f.SPIFFEIDs[1] || found.Profile != domain.WorkloadTrustBundleProfileSPIFFE || found.Issuer != "" {
		t.Fatalf("FindSPIFFEByTrustDomain = (%+v, %v)", found, err)
	}
	if byEmptyIssuer, err := f.TrustBundles.FindByIssuer(ctx, f.TenantA, ""); err != nil || byEmptyIssuer != nil {
		t.Fatalf("FindByIssuer(\"\") = (%+v, %v)", byEmptyIssuer, err)
	}
	if leaked, err := f.TrustBundles.FindSPIFFEByTrustDomain(ctx, f.TenantB, "spiffe.example.test"); err != nil || leaked != nil {
		t.Fatalf("other tenant FindSPIFFEByTrustDomain = (%+v, %v)", leaked, err)
	}
	duplicate := *spiffe
	duplicate.ID, duplicate.Name = f.SPIFFEIDs[2], "SPIRE again"
	if err := f.TrustBundles.Save(ctx, &duplicate); err == nil {
		t.Fatal("duplicate spiffe trust domain within one tenant was accepted")
	}
	otherTenant := duplicate
	otherTenant.TenantID = f.TenantB
	if err := f.TrustBundles.Save(ctx, &otherTenant); err != nil {
		t.Fatalf("same spiffe trust domain in another tenant: %v", err)
	}
	// 発行者のない spiffe の行が二つあっても、空の発行者を衝突として数えない。
	second := *spiffe
	second.ID, second.Name, second.TrustDomain = f.SPIFFEIDs[2], "SPIRE B", "b.example.test"
	if err := f.TrustBundles.Delete(ctx, f.TenantB, f.SPIFFEIDs[2]); err != nil {
		t.Fatalf("Delete other tenant: %v", err)
	}
	if err := f.TrustBundles.Save(ctx, &second); err != nil {
		t.Fatalf("second spiffe bundle without issuer: %v", err)
	}
}
