package support_http

import (
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// 発行者は discovery が広告する URL の出所である。subdomain の発行者がデプロイの scheme と
// port を落とすと、ローカル開発のように port を持つ配備では広告した URL に到達できない。
//
//spec:covers EX-TENANCY-024-02: port を持つ発行者のデプロイでは、subdomain テナントの発行者も同じ scheme と port を持つ。
func TestCanonicalLocationKeepsTheDeploymentSchemeAndPort(t *testing.T) {
	deps := Deps{Issuer: "http://localhost:5173/", TenantBaseDomain: "idp.test"}
	for _, tc := range []struct {
		style      tenancydomain.TenantEndpointStyle
		wantIssuer string
		wantPrefix string
	}{
		{tenancydomain.TenantEndpointStyleSubdomain, "http://acme.idp.test:5173", ""},
		{tenancydomain.TenantEndpointStylePath, "http://localhost:5173/realms/acme", "/realms/acme"},
	} {
		issuer, prefix := deps.CanonicalLocation(&tenancydomain.Tenant{Realm: "acme", EndpointStyle: tc.style})
		if issuer != tc.wantIssuer || prefix != tc.wantPrefix {
			t.Fatalf("%s: issuer = %q, prefix = %q, want %q, %q", tc.style, issuer, prefix, tc.wantIssuer, tc.wantPrefix)
		}
	}

	secure := Deps{Issuer: "https://login.example.com", TenantBaseDomain: "idp.test."}
	if issuer, _ := secure.CanonicalLocation(&tenancydomain.Tenant{
		Realm: "acme", EndpointStyle: tenancydomain.TenantEndpointStyleSubdomain,
	}); issuer != "https://acme.idp.test" {
		t.Fatalf("issuer = %q, want https without a port", issuer)
	}
}
