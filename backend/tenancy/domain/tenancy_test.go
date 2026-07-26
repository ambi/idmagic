package domain

import (
	"strings"
	"testing"
	"time"
)

// wi-129 の enum Valid / Validate カバレッジを wi-179 で shared/spec から移設。

func TestTenantStatusValid(t *testing.T) {
	cases := []struct {
		name string
		v    TenantStatus
		want bool
	}{
		{"tenant active", TenantStatusActive, true},
		{"tenant disabled", TenantStatusDisabled, true},
		{"tenant bad", TenantStatus("x"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.v.Valid(); got != c.want {
				t.Fatalf("%T(%v).Valid() = %v, want %v", c.v, c.v, got, c.want)
			}
		})
	}
}

func TestTenantValidateHappyAndFailure(t *testing.T) {
	now := time.Now().UTC()

	validTenant := Tenant{ID: "acme", Realm: "acme", DisplayName: "Acme", Status: TenantStatusActive, CreatedAt: now, UpdatedAt: now}
	badTenant := validTenant
	badTenant.Realm = "admin" // admin は予約語で realm として拒否される。

	cases := []struct {
		name    string
		v       interface{ Validate() error }
		wantErr bool
	}{
		{"tenant ok", validTenant, false},
		{"tenant bad", badTenant, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.v.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("%s: expected error, got nil", c.name)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("%s: expected valid, got %v", c.name, err)
			}
		})
	}
}

// wi-285: endpoint_style が Subdomain のとき realm はホスト名の最左ラベルになるため、
// 新規作成時の realm は単一 DNS ラベルとして妥当でなければならない (ADR-144)。
// 制約の強化は新規作成にのみ適用し、既存 realm は Tenant.Validate で再検証しない。
func TestValidateNewRealm(t *testing.T) {
	cases := []struct {
		name    string
		realm   string
		wantErr bool
	}{
		{"typical slug", "acme", false},
		{"internal hyphen", "acme-corp", false},
		{"digits", "t7f3k9m", false},
		{"empty", "", true},
		{"leading hyphen", "-acme", true},
		{"trailing hyphen is not a valid DNS label", "acme-", true},
		{"xn-- masquerades as an IDNA A-label", "xn--foo", true},
		{"uppercase", "Acme", true},
		{"dot is not allowed in a single label", "acme.corp", true},
		{"over 63 octets", strings.Repeat("a", 64), true},
		{"reserved admin", "admin", true},
		{"reserved www", "www", true},
		{"reserved api", "api", true},
		{"reserved login", "login", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateNewRealm(c.realm)
			if (err != nil) != c.wantErr {
				t.Fatalf("ValidateNewRealm(%q) error = %v, wantErr %v", c.realm, err, c.wantErr)
			}
		})
	}
}

func TestTenantEndpointStyleValid(t *testing.T) {
	cases := []struct {
		name string
		v    TenantEndpointStyle
		want bool
	}{
		{"path", TenantEndpointStylePath, true},
		{"subdomain", TenantEndpointStyleSubdomain, true},
		{"unknown", TenantEndpointStyle("custom_domain"), false},
		{"empty", TenantEndpointStyle(""), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.v.Valid(); got != c.want {
				t.Fatalf("%T(%v).Valid() = %v, want %v", c.v, c.v, got, c.want)
			}
		})
	}
}

// Subdomain は tenant base domain が設定された配備でのみ選択できる (ADR-144)。
// 未設定の配備で選ばせると、到達不能な正規ロケーションを持つテナントが生まれる。
func TestValidateEndpointStyleSelectable(t *testing.T) {
	cases := []struct {
		name       string
		style      TenantEndpointStyle
		baseDomain string
		wantErr    bool
	}{
		{"path is always selectable", TenantEndpointStylePath, "", false},
		{"path with base domain", TenantEndpointStylePath, "idmagic.example", false},
		{"subdomain needs a base domain", TenantEndpointStyleSubdomain, "", true},
		{"subdomain with base domain", TenantEndpointStyleSubdomain, "idmagic.example", false},
		{"unknown style", TenantEndpointStyle("x"), "idmagic.example", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateEndpointStyleSelectable(c.style, c.baseDomain)
			if (err != nil) != c.wantErr {
				t.Fatalf("ValidateEndpointStyleSelectable(%q, %q) error = %v, wantErr %v",
					c.style, c.baseDomain, err, c.wantErr)
			}
		})
	}
}

// 既存 realm は再検証しない。厳格化前に作られた realm を持つ Tenant が
// Validate で落ちると、起動や読み出しが壊れる (ADR-144 影響)。
func TestTenantValidateDoesNotReapplyNewRealmRules(t *testing.T) {
	now := time.Now().UTC()
	legacy := Tenant{
		ID: "11111111-1111-4111-8111-111111111111", Realm: "legacy-",
		DisplayName: "Legacy", Status: TenantStatusActive,
		EndpointStyle: TenantEndpointStylePath, CreatedAt: now, UpdatedAt: now,
	}
	if err := legacy.Validate(); err != nil {
		t.Fatalf("Tenant.Validate() on a pre-existing realm = %v, want nil", err)
	}
	if err := ValidateNewRealm(legacy.Realm); err == nil {
		t.Fatal("ValidateNewRealm() on the same realm = nil, want an error")
	}
}
