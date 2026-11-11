package domain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"strings"
	"testing"
	"time"
)

func testCertificate(t *testing.T, serial int64) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: "idmagic"},
		NotBefore: time.Unix(0, 0), NotAfter: time.Unix(0, 0).Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return certificate
}

func contosoExpected(signing, next *x509.Certificate) EntraExpectedSettings {
	return EntraExpectedSettings{
		Domain: "contoso.com", IssuerURI: "urn:contoso", PassiveSignInURI: "https://idp.example/wsfed",
		ActiveSignInURI: "https://idp.example/trust/usernamemixed", MetadataExchangeURI: "https://idp.example/trust/mex",
		SignOutURI: "https://idp.example/wsfed?wa=wsignout1.0", SigningCertificate: signing, NextSigningCertificate: next,
	}
}

func appliedObservation(signing string) EntraFederationObservation {
	return EntraFederationObservation{
		Format: EntraObservationFormat, EntraTenantID: "11111111-2222-3333-4444-555555555555", Domain: "contoso.com",
		AuthenticationType: "Federated", RetrievedAt: time.Unix(1000, 0).UTC(),
		FederationConfigurations: []EntraObservedFederationConfiguration{{
			IssuerURI: "urn:contoso", PassiveSignInURI: "https://idp.example/wsfed",
			ActiveSignInURI: "https://idp.example/trust/usernamemixed", MetadataExchangeURI: "https://idp.example/trust/mex",
			SignOutURI: "https://idp.example/wsfed?wa=wsignout1.0", SigningCertificate: signing,
			PreferredAuthenticationProtocol: "wsFed", FederatedIdpMfaBehavior: "acceptIfMfaDoneByFederatedIdp",
		}},
	}
}

func differenceFields(differences []EntraSettingDifference) string {
	fields := make([]string, 0, len(differences))
	for _, d := range differences {
		fields = append(fields, d.Field)
	}
	return strings.Join(fields, ",")
}

//spec:covers REQ-WSFEDERATION-001: 証明書を空白を除いた X.509 の DER で比べ、読めない値を不一致にし、URL を完全一致で、列挙値を大文字と小文字を区別せずに比べること。
func TestCompareEntraFederationReadsCertificatesAndCase(t *testing.T) {
	signing, other := testCertificate(t, 1), testCertificate(t, 2)
	encoded := base64.StdEncoding.EncodeToString(signing.Raw)
	cases := []struct {
		name   string
		edit   func(*EntraFederationObservation)
		next   *x509.Certificate
		fields string
	}{
		{name: "applied", edit: func(*EntraFederationObservation) {}},
		{name: "wrapped certificate", edit: func(o *EntraFederationObservation) {
			o.FederationConfigurations[0].SigningCertificate = encoded[:10] + "\n\t " + encoded[10:]
		}},
		{name: "enum case", edit: func(o *EntraFederationObservation) {
			o.AuthenticationType = "federated"
			o.FederationConfigurations[0].PreferredAuthenticationProtocol = "WSFED"
		}},
		{name: "url case", edit: func(o *EntraFederationObservation) {
			o.FederationConfigurations[0].PassiveSignInURI = "https://IDP.example/wsfed"
		}, fields: "passive_sign_in_uri"},
		{name: "other certificate", edit: func(o *EntraFederationObservation) {
			o.FederationConfigurations[0].SigningCertificate = base64.StdEncoding.EncodeToString(other.Raw)
		}, fields: "signing_certificate"},
		{name: "unreadable certificate", edit: func(o *EntraFederationObservation) {
			o.FederationConfigurations[0].SigningCertificate = "not-base64"
		}, fields: "signing_certificate"},
		{name: "next registered but not prepared", edit: func(o *EntraFederationObservation) {
			o.FederationConfigurations[0].NextSigningCertificate = base64.StdEncoding.EncodeToString(other.Raw)
		}, fields: "next_signing_certificate"},
		{name: "next prepared but not registered", edit: func(*EntraFederationObservation) {}, next: other, fields: "next_signing_certificate"},
		{name: "managed domain with two configurations", edit: func(o *EntraFederationObservation) {
			o.AuthenticationType = "Managed"
			o.FederationConfigurations = append(o.FederationConfigurations, o.FederationConfigurations[0])
		}, fields: "authentication_type,federation_configurations"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observation := appliedObservation(encoded)
			tc.edit(&observation)
			if got := differenceFields(CompareEntraFederation(contosoExpected(signing, tc.next), observation)); got != tc.fields {
				t.Fatalf("differences = %q, want %q", got, tc.fields)
			}
		})
	}
	unreadable := appliedObservation("not-base64")
	if d := CompareEntraFederation(contosoExpected(signing, nil), unreadable); len(d) != 1 || d[0].Observed != "unreadable" || d[0].Expected != CertificateThumbprint(signing) {
		t.Fatalf("unreadable difference = %+v", d)
	}
}

//spec:covers REQ-WSFEDERATION-001: 期待値の指紋がテナント、Graph の設定値、次の証明書のどれでも変わり、記録の状態が未確認、一致、再確認が必要に分かれること。
func TestEntraVerificationForFollowsTheFingerprint(t *testing.T) {
	signing, next := testCertificate(t, 1), testCertificate(t, 2)
	expected := contosoExpected(signing, nil)
	if got := EntraVerificationFor(nil, expected); got.Status != EntraVerificationUnverified || got.CheckedAt != nil {
		t.Fatalf("no record = %+v", got)
	}
	record := NewEntraFederationObservationRecord("tenant", expected, appliedObservation(base64.StdEncoding.EncodeToString(signing.Raw)), time.Unix(2000, 0))
	if got := EntraVerificationFor(&record, expected); got.Status != EntraVerificationMatched || got.Differences == nil || len(*got.Differences) != 0 {
		t.Fatalf("matched = %+v", got)
	}
	tenant := expected
	tenant.EntraTenantID = "11111111-2222-3333-4444-555555555555"
	issuer := expected
	issuer.IssuerURI = "urn:changed"
	for name, changed := range map[string]EntraExpectedSettings{
		"tenant": tenant, "issuer": issuer, "next certificate": contosoExpected(signing, next),
	} {
		if got := EntraVerificationFor(&record, changed); got.Status != EntraVerificationStale || got.Differences != nil {
			t.Fatalf("%s change = %+v, want stale without differences", name, got)
		}
	}
}

//spec:covers REQ-WSFEDERATION-001: 取得時刻が確認時刻より 5 分を超えて後の観測、4 件を超える構成、別のドメインを拒否し、境界の値を受けること。
func TestEntraObservationValidateBoundaries(t *testing.T) {
	checkedAt := time.Unix(10_000, 0).UTC()
	valid := appliedObservation("")
	valid.RetrievedAt = checkedAt.Add(5 * time.Minute)
	valid.FederationConfigurations = make([]EntraObservedFederationConfiguration, 4)
	if err := valid.Validate("contoso.com", checkedAt); err != nil {
		t.Fatalf("boundary observation: %v", err)
	}
	late := valid
	late.RetrievedAt = checkedAt.Add(5*time.Minute + time.Nanosecond)
	many := valid
	many.FederationConfigurations = make([]EntraObservedFederationConfiguration, 5)
	other := valid
	other.Domain = "fabrikam.com"
	for name, observation := range map[string]EntraFederationObservation{"late": late, "many": many, "other domain": other} {
		if err := observation.Validate("contoso.com", checkedAt); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
}

//spec:covers REQ-WSFEDERATION-001: GUID でない Entra のテナント ID を拒否し、大文字の GUID を小文字にすること。
func TestNormalizeEntraTenantID(t *testing.T) {
	if got, err := NormalizeEntraTenantID(" 0F1E2D3C-4B5A-6978-8796-A5B4C3D2E1F0 "); err != nil || got != "0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0" {
		t.Fatalf("NormalizeEntraTenantID = %q, %v", got, err)
	}
	for _, invalid := range []string{"contoso", "0f1e2d3c4b5a697887 96a5b4c3d2e1f0", "0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f"} {
		if _, err := NormalizeEntraTenantID(invalid); err == nil {
			t.Fatalf("%q was accepted", invalid)
		}
	}
}
