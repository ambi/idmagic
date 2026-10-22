package domain

import (
	"errors"
	"slices"
	"strings"
	"testing"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

//spec:covers REQ-WSFEDERATION-002: sourceAnchor の値を変換せずにそのまま ImmutableID にし、値がないか空白だけか `@` を含めば ImmutableID にしないこと。
func TestImmutableIDOfUsesTheSourceAnchorVerbatim(t *testing.T) {
	profile := &EntraFederationProfile{SourceAnchorAttribute: "source_anchor"}
	for _, value := range []string{"/xmWb4aLEdC0LQDAT8lk/w==", "E-00042", "6f9619ff-8b86-d011-b42d-00c04fc964ff", "MixedCase"} {
		got, err := ImmutableIDOf(claimdomain.Attributes{"source_anchor": {value}}, profile)
		if err != nil || got != value {
			t.Errorf("ImmutableIDOf(%q) = (%q, %v), want the value unchanged", value, got, err)
		}
	}
	for name, tc := range map[string]struct {
		attrs claimdomain.Attributes
		want  error
	}{
		"absent":          {claimdomain.Attributes{}, ErrSourceAnchorMissing},
		"empty":           {claimdomain.Attributes{"source_anchor": {""}}, ErrSourceAnchorMissing},
		"whitespace only": {claimdomain.Attributes{"source_anchor": {"  "}}, ErrSourceAnchorInvalid},
		"contains @":      {claimdomain.Attributes{"source_anchor": {"bob@corp"}}, ErrSourceAnchorInvalid},
		"other attribute": {claimdomain.Attributes{"object_guid": {"E-1"}}, ErrSourceAnchorMissing},
	} {
		if _, err := ImmutableIDOf(tc.attrs, profile); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.want)
		}
	}
}

//spec:covers REQ-WSFEDERATION-002: 文字列の sourceAnchor は大文字と小文字や前後の空白を含めて保存値のまま発行し、@ を含む値は拒否すること。
func FuzzImmutableIDPreservesTheStoredString(f *testing.F) {
	for _, seed := range []string{"E-00042", " MixedCase ", "/xmWb4aLEdC0LQDAT8lk/w==", "", "@"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		profile := &EntraFederationProfile{SourceAnchorAttribute: "source_anchor"}
		value := "ID-" + strings.ReplaceAll(input, "@", "-")
		got, err := ImmutableIDOf(claimdomain.Attributes{"source_anchor": {value}}, profile)
		if err != nil || got != value {
			t.Fatalf("stored value %q became (%q, %v)", value, got, err)
		}
		if _, err := ImmutableIDOf(claimdomain.Attributes{"source_anchor": {value + "@"}}, profile); !errors.Is(err, ErrSourceAnchorInvalid) {
			t.Fatalf("sourceAnchor containing @ was not refused: %v", err)
		}
	})
}

func TestApplyEntraProfileAddsTheSourceAnchorAsTheImmutableID(t *testing.T) {
	attrs, err := ApplyEntraProfile(claimdomain.Attributes{"object_guid": {"E-00042"}}, &EntraFederationProfile{SourceAnchorAttribute: "object_guid"})
	if err != nil {
		t.Fatalf("ApplyEntraProfile: %v", err)
	}
	if got := attrs[EntraImmutableIDAttribute]; len(got) != 1 || got[0] != "E-00042" {
		t.Fatalf("entra_immutable_id = %#v", got)
	}
	if _, err := ApplyEntraProfile(claimdomain.Attributes{}, &EntraFederationProfile{SourceAnchorAttribute: "object_guid"}); !errors.Is(err, ErrSourceAnchorMissing) {
		t.Fatalf("missing sourceAnchor err = %v", err)
	}
}

//spec:covers REQ-WSFEDERATION-001: sourceAnchor の属性は unique で editable_by_user でない属性に限り、定義のない属性も拒否すること。
func TestCheckEntraSourceAnchorAttributeRequiresAUniqueAdminManagedAttribute(t *testing.T) {
	profile := &EntraFederationProfile{SourceAnchorAttribute: "source_anchor"}
	def := func(unique, editable bool) []userdomain.UserAttributeDef {
		return []userdomain.UserAttributeDef{
			{Key: "other", Type: idmdomain.AttributeTypeString, Unique: true},
			{Key: "source_anchor", Type: idmdomain.AttributeTypeString, Unique: unique, EditableByUser: editable},
		}
	}
	for name, tc := range map[string]struct {
		defs []userdomain.UserAttributeDef
		want error
	}{
		"protected":           {def(true, false), nil},
		"not unique":          {def(false, false), ErrSourceAnchorNotUnique},
		"editable":            {def(true, true), ErrSourceAnchorEditable},
		"not unique editable": {def(false, true), ErrSourceAnchorNotUnique},
		"undefined":           {def(true, false)[:1], ErrSourceAnchorNotUnique},
	} {
		if err := CheckEntraSourceAnchorAttribute(tc.defs, profile); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.want)
		}
	}
}

//spec:covers REQ-WSFEDERATION-001: 構成の時点の値の検査が、フェデレーションするドメインの削除されていない利用者だけを対象にし、問題のある値の件数とユーザー名の順で先頭の 5 人を返すこと。
func TestCheckEntraSourceAnchorValuesChecksOnlyFederatedUsers(t *testing.T) {
	profile := EntraFederationProfile{SourceAnchorAttribute: "source_anchor", Domains: []EntraDomainFederation{{Domain: "contoso.com", IssuerURI: "urn:c"}}}
	person := func(upn, anchor string) *userdomain.User {
		u := &userdomain.User{ID: upn, PreferredUsername: upn}
		if anchor != "" {
			v := anchor
			u.Attributes = map[string]userdomain.AttributeValue{"source_anchor": {Type: idmdomain.AttributeTypeString, String: &v}}
		}
		return u
	}
	deleted := person("gone@contoso.com", "")
	deleted.Lifecycle.Status = idmdomain.UserStatusDeleted
	ok := []*userdomain.User{person("alice@contoso.com", "A-1"), person("admin@local.example", ""), deleted}
	if got := CheckEntraSourceAnchorValues(profile, ok); got != nil {
		t.Fatalf("federated users with values: %+v, want nil", got)
	}

	invalid := append(slices.Clone(ok), person("zed@contoso.com", "z@corp"))
	if got := CheckEntraSourceAnchorValues(profile, invalid); got == nil || !errors.Is(got.Err, ErrSourceAnchorInvalid) || got.UserCount != 1 {
		t.Fatalf("invalid value: %+v", got)
	}

	missing := slices.Clone(ok)
	for _, upn := range []string{"g@contoso.com", "b@corp.contoso.com", "f@contoso.com", "a@contoso.com", "e@contoso.com", "c@contoso.com"} {
		missing = append(missing, person(upn, ""))
	}
	got := CheckEntraSourceAnchorValues(profile, missing)
	if got == nil || !errors.Is(got.Err, ErrSourceAnchorMissing) || got.UserCount != 6 ||
		!slices.Equal(got.SampleUsernames, []string{"a@contoso.com", "b@corp.contoso.com", "c@contoso.com", "e@contoso.com", "f@contoso.com"}) {
		t.Fatalf("missing values: %+v", got)
	}
}

func TestBuildEntraClaimPolicy(t *testing.T) {
	policy := BuildEntraClaimPolicy()
	if policy.NameID.Format != EntraPersistentNameIDFormat {
		t.Fatalf("NameID format = %q", policy.NameID.Format)
	}
	if policy.NameID.SourceAttribute != EntraImmutableIDAttribute {
		t.Fatalf("NameID source = %q", policy.NameID.SourceAttribute)
	}
	if len(policy.Rules) != 3 {
		t.Fatalf("rules len = %d", len(policy.Rules))
	}
}

func contosoAndFabrikam() EntraFederationProfile {
	return EntraFederationProfile{
		SourceAnchorAttribute: "object_guid",
		Domains: []EntraDomainFederation{
			{Domain: "contoso.com", IssuerURI: "urn:idmagic:entra:contoso.com"},
			{Domain: "corp.contoso.com", IssuerURI: "urn:idmagic:entra:corp.contoso.com"},
			{Domain: "fabrikam.com", IssuerURI: "http://fabrikam.com/adfs/services/trust/"},
		},
	}
}

func TestEntraIssuerForChoosesTheLongestFederatedParentOfTheUPNDomain(t *testing.T) {
	profile := contosoAndFabrikam()
	for upn, want := range map[string]string{
		"alice@contoso.com":         "urn:idmagic:entra:contoso.com",
		"alice@CONTOSO.com":         "urn:idmagic:entra:contoso.com",
		"bob@corp.contoso.com":      "urn:idmagic:entra:corp.contoso.com",
		"carol@eu.corp.contoso.com": "urn:idmagic:entra:corp.contoso.com",
		"dave@sales.contoso.com":    "urn:idmagic:entra:contoso.com",
		"erin@fabrikam.com":         "http://fabrikam.com/adfs/services/trust/",
		"first@last@fabrikam.com":   "http://fabrikam.com/adfs/services/trust/",
	} {
		got, err := profile.IssuerFor(upn)
		if err != nil || got != want {
			t.Errorf("IssuerFor(%q) = (%q, %v), want %q", upn, got, err, want)
		}
	}
}

func TestEntraIssuerForRefusesAUPNOutsideTheFederatedDomains(t *testing.T) {
	profile := contosoAndFabrikam()
	for _, upn := range []string{"eve@example.org", "eve@notcontoso.com", "eve@contoso.com.example", "contoso.com", "eve@", ""} {
		if got, err := profile.IssuerFor(upn); !errors.Is(err, ErrEntraDomainNotFederated) {
			t.Errorf("IssuerFor(%q) = (%q, %v), want ErrEntraDomainNotFederated", upn, got, err)
		}
	}
}

func TestEntraWithDomainAddsReplacesAndRefusesASharedIssuer(t *testing.T) {
	profile := contosoAndFabrikam()

	added, isNew, err := profile.WithDomain(EntraDomainFederation{Domain: "northwind.com", IssuerURI: "urn:northwind"}, "object_guid")
	if err != nil || !isNew || len(added.Domains) != 4 {
		t.Fatalf("add = (%+v, %v, %v), want a fourth domain", added.Domains, isNew, err)
	}
	if len(profile.Domains) != 3 {
		t.Fatalf("WithDomain changed the receiver: %+v", profile.Domains)
	}

	replaced, isNew, err := profile.WithDomain(EntraDomainFederation{Domain: "fabrikam.com", IssuerURI: "urn:fabrikam:v2"}, "object_guid")
	if err != nil || isNew || len(replaced.Domains) != 3 {
		t.Fatalf("replace = (%+v, %v, %v), want the same three domains", replaced.Domains, isNew, err)
	}
	if got, _ := replaced.IssuerFor("erin@fabrikam.com"); got != "urn:fabrikam:v2" {
		t.Fatalf("replaced issuer = %q", got)
	}

	if _, _, err := profile.WithDomain(EntraDomainFederation{Domain: "northwind.com", IssuerURI: "urn:idmagic:entra:contoso.com"}, "object_guid"); !errors.Is(err, ErrEntraIssuerInUse) {
		t.Fatalf("shared issuer err = %v, want ErrEntraIssuerInUse", err)
	}
	if _, _, err := profile.WithDomain(EntraDomainFederation{Domain: "northwind.com", IssuerURI: "urn:northwind"}, "employee_id"); !errors.Is(err, ErrEntraSourceAnchorChanged) {
		t.Fatalf("changed sourceAnchor err = %v, want ErrEntraSourceAnchorChanged", err)
	}
	// 構成済みのドメインがそのドメインだけなら、置き換えで sourceAnchor の属性を変えられる。
	only := EntraFederationProfile{SourceAnchorAttribute: "object_guid", Domains: []EntraDomainFederation{{Domain: "contoso.com", IssuerURI: "urn:c"}}}
	if changed, _, err := only.WithDomain(EntraDomainFederation{Domain: "contoso.com", IssuerURI: "urn:c"}, "employee_id"); err != nil || changed.SourceAnchorAttribute != "employee_id" {
		t.Fatalf("sole domain sourceAnchor change = (%+v, %v)", changed, err)
	}
}

func TestEntraAttributeDefsReleaseTheImmutableIDOnlyWhenTheSourceAnchorIsReleasable(t *testing.T) {
	profile := &EntraFederationProfile{SourceAnchorAttribute: "object_guid"}
	for visibility, want := range map[idmdomain.AttrVisibility]bool{
		idmdomain.AttrVisibilityAdminReadable: true,
		idmdomain.AttrVisibilityClaimExposed:  true,
		idmdomain.AttrVisibilityPrivate:       false,
	} {
		defs := []userdomain.UserAttributeDef{{Key: "object_guid", Type: idmdomain.AttributeTypeString, Visibility: visibility}}
		if got := claimdomain.IsAttributeReleasable(EntraImmutableIDAttribute, EntraAttributeDefs(defs, profile)); got != want {
			t.Errorf("sourceAnchor %s: ImmutableID releasable = %v, want %v", visibility, got, want)
		}
	}
	// 判定に使うのは、既定の object_guid ではなく、定型設定が指定した sourceAnchor の属性である。
	mixed := []userdomain.UserAttributeDef{
		{Key: "object_guid", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityAdminReadable},
		{Key: "employee_id", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityPrivate},
	}
	if claimdomain.IsAttributeReleasable(EntraImmutableIDAttribute, EntraAttributeDefs(mixed, &EntraFederationProfile{SourceAnchorAttribute: "employee_id"})) {
		t.Error("a private sourceAnchor released the ImmutableID because the default attribute is releasable")
	}
	if claimdomain.IsAttributeReleasable(EntraImmutableIDAttribute, EntraAttributeDefs(nil, profile)) {
		t.Error("an undefined sourceAnchor attribute released the ImmutableID")
	}
	if got := EntraAttributeDefs(nil, nil); len(got) != 0 {
		t.Errorf("no profile added definitions: %+v", got)
	}
}

func TestBuildEntraClaimPolicyIssuesTheImmutableIDClaim(t *testing.T) {
	for _, rule := range BuildEntraClaimPolicy().Rules {
		if rule.ClaimType == EntraImmutableIDClaim && rule.SourceKey == EntraImmutableIDAttribute && rule.Required {
			return
		}
	}
	t.Fatalf("policy lacks a required %s claim from %s", EntraImmutableIDClaim, EntraImmutableIDAttribute)
}
