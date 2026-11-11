package testing_contract

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/wsfederation/domain"
	"github.com/ambi/idmagic/backend/wsfederation/ports"
)

// ObservationFixture は、観測記録の保存先の共有契約に渡す保存先と二つのテナントである。
type ObservationFixture struct {
	Repository ports.EntraFederationObservationRepository
	TenantA    string
	TenantB    string
	Now        time.Time
}

// RunEntraObservations は、観測記録がテナントとドメインごとに一件で、古い取得時刻で上書きされず、
// 別テナントから読めず、削除で残らないことを確かめる。
func RunEntraObservations(t *testing.T, newFixture func(t *testing.T) ObservationFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	record := func(tenantID, domainName string, retrievedAt time.Time, matched bool) domain.EntraFederationObservationRecord {
		return domain.EntraFederationObservationRecord{
			TenantID: tenantID, Domain: domainName,
			Observation: domain.EntraFederationObservation{
				Format: domain.EntraObservationFormat, EntraTenantID: "11111111-2222-3333-4444-555555555555",
				Domain: domainName, AuthenticationType: "Federated", RetrievedAt: retrievedAt,
				FederationConfigurations: []domain.EntraObservedFederationConfiguration{{IssuerURI: "urn:example:" + domainName}},
			},
			ExpectedFingerprint: "fingerprint", SigningCertificateThumbprint: "current", NextSigningCertificateThumbprint: "next",
			Matched: matched, Differences: []domain.EntraSettingDifference{{Field: "issuer_uri", Expected: "a", Observed: "b"}},
			CheckedAt: f.Now,
		}
	}
	if got, err := f.Repository.Find(ctx, f.TenantA, "contoso.com"); err != nil || got != nil {
		t.Fatalf("missing Find = (%+v, %v)", got, err)
	}
	newer := record(f.TenantA, "contoso.com", f.Now.Add(-time.Minute), false)
	if err := f.Repository.Save(ctx, newer); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := f.Repository.Find(ctx, f.TenantA, "contoso.com")
	if err != nil || got == nil || !got.Observation.RetrievedAt.Equal(newer.Observation.RetrievedAt) || got.Matched ||
		len(got.Differences) != 1 || got.Differences[0] != newer.Differences[0] ||
		len(got.Observation.FederationConfigurations) != 1 || got.Observation.FederationConfigurations[0].IssuerURI != "urn:example:contoso.com" ||
		got.ExpectedFingerprint != "fingerprint" || got.NextSigningCertificateThumbprint != "next" {
		t.Fatalf("Find = (%+v, %v), want round trip", got, err)
	}
	older := record(f.TenantA, "contoso.com", f.Now.Add(-time.Hour), true)
	if err := f.Repository.Save(ctx, older); !errors.Is(err, ports.ErrEntraObservationOutdated) {
		t.Fatalf("Save older = %v, want ErrEntraObservationOutdated", err)
	}
	if got, err := f.Repository.Find(ctx, f.TenantA, "contoso.com"); err != nil || got == nil || got.Matched {
		t.Fatalf("older observation replaced the record: (%+v, %v)", got, err)
	}
	same := record(f.TenantA, "contoso.com", f.Now.Add(-time.Minute), true)
	if err := f.Repository.Save(ctx, same); err != nil {
		t.Fatalf("Save same retrieval: %v", err)
	}
	if got, err := f.Repository.Find(ctx, f.TenantA, "contoso.com"); err != nil || got == nil || !got.Matched {
		t.Fatalf("re-import of the same retrieval was not stored: (%+v, %v)", got, err)
	}
	if leaked, err := f.Repository.Find(ctx, f.TenantB, "contoso.com"); err != nil || leaked != nil {
		t.Fatalf("other tenant Find = (%+v, %v)", leaked, err)
	}
	if err := f.Repository.Save(ctx, record(f.TenantA, "adatum.com", f.Now, true)); err != nil {
		t.Fatalf("Save second domain: %v", err)
	}
	if err := f.Repository.Save(ctx, record(f.TenantB, "contoso.com", f.Now, true)); err != nil {
		t.Fatalf("Save other tenant: %v", err)
	}
	list, err := f.Repository.ListAll(ctx, f.TenantA)
	if err != nil || len(list) != 2 || list[0].Domain != "adatum.com" || list[1].Domain != "contoso.com" {
		t.Fatalf("ListAll = (%+v, %v), want the tenant's records in domain order", list, err)
	}
	if err := f.Repository.Delete(ctx, f.TenantA, "contoso.com"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, err := f.Repository.Find(ctx, f.TenantA, "contoso.com"); err != nil || got != nil {
		t.Fatalf("Find after Delete = (%+v, %v)", got, err)
	}
	if kept, err := f.Repository.Find(ctx, f.TenantB, "contoso.com"); err != nil || kept == nil {
		t.Fatalf("Delete removed another tenant's record: (%+v, %v)", kept, err)
	}
}
