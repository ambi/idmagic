package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// rollbackTenantID は、contoso.com を持つ Entra のテナントである。
const rollbackTenantID = "abcdef01-2222-3333-4444-555555555555"

// rolledBack は、contoso.com を切り戻した後に読み戻した観測である。
func rolledBack(authenticationType string, configs ...EntraObservedFederationConfiguration) EntraFederationObservation {
	return EntraFederationObservation{
		Format: EntraObservationFormat, EntraTenantID: rollbackTenantID, Domain: "contoso.com",
		AuthenticationType: authenticationType, RetrievedAt: time.Unix(1000, 0).UTC(), FederationConfigurations: configs,
	}
}

func previousADFS() EntraObservedFederationConfiguration {
	return EntraObservedFederationConfiguration{
		IssuerURI: "http://contoso.com/adfs/services/trust", PassiveSignInURI: "https://adfs.contoso.com/adfs/ls/",
		ActiveSignInURI:     "https://adfs.contoso.com/adfs/services/trust/2005/usernamemixed",
		MetadataExchangeURI: "https://adfs.contoso.com/adfs/services/trust/mex", PreferredAuthenticationProtocol: "wsFed",
	}
}

// federatedTo は、Entra のサインインの画面が WS-Fed の受動のエンドポイントへ送るときの探索の答えである。
// AuthURL は、登録した PassiveSignInUri に Entra が問い合わせ文字列を付けた形になる。
func federatedTo(passive string) *EntraDomainRealm {
	return &EntraDomainRealm{
		NamespaceType: "Federated",
		AuthURL:       passive + "?username=realm-check%40contoso.com&wa=wsignin1.0&wtrealm=urn%3afederation%3aMicrosoftOnline&wctx=",
	}
}

//spec:covers REQ-WSFEDERATION-001: レルム探索が IdMagic の受動のエンドポイント（ホストの大文字と小文字、問い合わせ文字列を問わない）を答えれば観測記録を問わずに拒否し、Managed か IdMagic を指さない Federated なら探索を根拠に解除し、探索の失敗、Unknown、ほかの値では切り戻しを示す観測記録があるときだけ解除すること。
func TestDecideEntraRelease(t *testing.T) {
	expected := contosoExpected(testCertificate(t, 1), nil)
	expected.EntraTenantID = rollbackTenantID
	upperHost := strings.Replace(expected.PassiveSignInURI, "idp.example", "IDP.Example", 1)
	released := rolledBack("Managed")
	for _, tc := range []struct {
		name        string
		realm       *EntraDomainRealm
		observation *EntraFederationObservation
		want        EntraReleaseEvidence
		wantErr     error
	}{
		{name: "managed", realm: &EntraDomainRealm{NamespaceType: "Managed"}, want: EntraReleasedByRealmDiscovery},
		{name: "managed in another case", realm: &EntraDomainRealm{NamespaceType: "managed"}, want: EntraReleasedByRealmDiscovery},
		{name: "federated to the previous IdP", realm: federatedTo("https://adfs.contoso.com/adfs/ls/"), want: EntraReleasedByRealmDiscovery},
		{name: "federated to another path of the IdMagic host", realm: federatedTo("https://idp.example/other"), want: EntraReleasedByRealmDiscovery},
		{name: "still federated to IdMagic", realm: federatedTo(expected.PassiveSignInURI), wantErr: ErrEntraFederationStillActive},
		{name: "still federated to IdMagic in another host case", realm: federatedTo(upperHost), wantErr: ErrEntraFederationStillActive},
		{name: "still federated to IdMagic despite an observation", realm: federatedTo(expected.PassiveSignInURI), observation: &released, wantErr: ErrEntraFederationStillActive},
		{name: "federated without AuthURL", realm: &EntraDomainRealm{NamespaceType: "Federated"}, wantErr: ErrEntraRollbackUnconfirmed},
		{name: "federated with a malformed AuthURL", realm: federatedTo("https://%zz"), wantErr: ErrEntraRollbackUnconfirmed},
		{name: "federated with a relative AuthURL", realm: federatedTo("/wsfed"), wantErr: ErrEntraRollbackUnconfirmed},
		{name: "federated without a host", realm: federatedTo("https:///wsfed"), wantErr: ErrEntraRollbackUnconfirmed},
		{name: "federated with another scheme", realm: federatedTo("javascript:signIn()"), wantErr: ErrEntraRollbackUnconfirmed},
		{name: "malformed AuthURL with an observation", realm: federatedTo("https://%zz"), observation: &released, want: EntraReleasedByObservation},
		{name: "unknown without an observation", realm: &EntraDomainRealm{NamespaceType: "Unknown"}, wantErr: ErrEntraRollbackUnconfirmed},
		{name: "unknown with an observation", realm: &EntraDomainRealm{NamespaceType: "Unknown"}, observation: &released, want: EntraReleasedByObservation},
		{name: "lookup failed without an observation", wantErr: ErrEntraRollbackUnconfirmed},
		{name: "lookup failed with an observation", observation: &released, want: EntraReleasedByObservation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var record *EntraFederationObservationRecord
			if tc.observation != nil {
				r := NewEntraFederationObservationRecord("tenant", expected, *tc.observation, time.Unix(1000, 0))
				record = &r
			}
			got, err := DecideEntraRelease(tc.realm, record, expected)
			if !errors.Is(err, tc.wantErr) || (tc.wantErr == nil && err != nil) || got != tc.want {
				t.Fatalf("DecideEntraRelease = (%q, %v), want (%q, %v)", got, err, tc.want, tc.wantErr)
			}
		})
	}
}

//spec:covers REQ-WSFEDERATION-001: 探索で確かめられないとき、IdMagic を指す構成、別のテナント、Managed で構成が残る、Federated で構成が一つでない、再確認が必要な観測記録を切り戻しの根拠にしないこと。
func TestDecideEntraReleaseFallsBackOnlyToAReleasingObservation(t *testing.T) {
	expected := contosoExpected(testCertificate(t, 1), nil)
	expected.EntraTenantID = rollbackTenantID
	pointing := func(edit func(*EntraObservedFederationConfiguration)) EntraFederationObservation {
		config := previousADFS()
		edit(&config)
		return rolledBack("Federated", config)
	}
	otherTenant := rolledBack("Managed")
	otherTenant.EntraTenantID = "99999999-9999-9999-9999-999999999999"
	upperTenant := rolledBack("Managed")
	upperTenant.EntraTenantID = strings.ToUpper(rollbackTenantID)
	for _, tc := range []struct {
		name        string
		observation EntraFederationObservation
		stale       bool
		releases    bool
	}{
		{name: "managed", observation: rolledBack("Managed"), releases: true},
		{name: "previous IdP", observation: rolledBack("Federated", previousADFS()), releases: true},
		{name: "tenant in upper case", observation: upperTenant, releases: true},
		{name: "managed with a configuration left", observation: rolledBack("Managed", previousADFS())},
		{name: "federated with two configurations", observation: rolledBack("Federated", previousADFS(), previousADFS())},
		{name: "federated without a configuration", observation: rolledBack("Federated")},
		{name: "IdMagic issuer", observation: pointing(func(c *EntraObservedFederationConfiguration) { c.IssuerURI = expected.IssuerURI })},
		{name: "IdMagic passive endpoint", observation: pointing(func(c *EntraObservedFederationConfiguration) { c.PassiveSignInURI = expected.PassiveSignInURI })},
		{name: "IdMagic active endpoint", observation: pointing(func(c *EntraObservedFederationConfiguration) { c.ActiveSignInURI = expected.ActiveSignInURI })},
		{name: "IdMagic MEX endpoint", observation: pointing(func(c *EntraObservedFederationConfiguration) { c.MetadataExchangeURI = expected.MetadataExchangeURI })},
		{name: "another tenant", observation: otherTenant},
		{name: "stale", observation: rolledBack("Managed"), stale: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := NewEntraFederationObservationRecord("tenant", expected, tc.observation, time.Unix(1000, 0))
			if tc.stale {
				record.ExpectedFingerprint = "changed"
			}
			got, err := DecideEntraRelease(nil, &record, expected)
			if tc.releases != (err == nil) || (tc.releases && got != EntraReleasedByObservation) ||
				(!tc.releases && !errors.Is(err, ErrEntraRollbackUnconfirmed)) {
				t.Fatalf("DecideEntraRelease = (%q, %v), releases want %v", got, err, tc.releases)
			}
		})
	}
	// テナントを持たない組は、観測したテナントを比べない。
	withoutTenant := expected
	withoutTenant.EntraTenantID = ""
	record := NewEntraFederationObservationRecord("tenant", withoutTenant, otherTenant, time.Unix(1000, 0))
	if _, err := DecideEntraRelease(nil, &record, withoutTenant); err != nil {
		t.Fatalf("pair without a tenant: %v", err)
	}
}

//spec:covers REQ-WSFEDERATION-001: ドメインの解除が、そのドメインの組だけを外し、ほかの組と sourceAnchor とエンドポイントを保ち、最後の組を外したことを示すこと。
func TestEntraWithoutDomainRemovesOnlyThatPair(t *testing.T) {
	profile := contosoAndFabrikam()
	profile.PassiveLogOnURI = "https://idp.example/wsfed"
	// 親のドメインを外しても、サブドメインの組は別の組として残る。
	out, last, ok := profile.WithoutDomain("contoso.com")
	if !ok || last || len(out.Domains) != 2 || out.Domains[0] != profile.Domains[1] || out.Domains[1] != profile.Domains[2] ||
		out.SourceAnchorAttribute != profile.SourceAnchorAttribute || out.PassiveLogOnURI != profile.PassiveLogOnURI {
		t.Fatalf("WithoutDomain(contoso.com) = %+v, last=%v ok=%v", out, last, ok)
	}
	if len(profile.Domains) != 3 {
		t.Fatalf("WithoutDomain changed the receiver: %+v", profile.Domains)
	}
	if _, _, ok := out.WithoutDomain("contoso.com"); ok {
		t.Fatal("WithoutDomain of a removed domain reported ok")
	}
	one, last, ok := out.WithoutDomain("corp.contoso.com")
	if !ok || last || len(one.Domains) != 1 {
		t.Fatalf("WithoutDomain(corp.contoso.com) = %+v, last=%v ok=%v", one, last, ok)
	}
	empty, last, ok := one.WithoutDomain("fabrikam.com")
	if !ok || !last || len(empty.Domains) != 0 {
		t.Fatalf("WithoutDomain of the last domain = %+v, last=%v ok=%v", empty, last, ok)
	}
}
