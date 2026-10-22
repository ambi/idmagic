package domain

import (
	"errors"
	"testing"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

func TestNormalizeImmutableID_GUIDUsesMicrosoftByteOrder(t *testing.T) {
	got, err := NormalizeImmutableID("6f9619ff-8b86-d011-b42d-00c04fc964ff")
	if err != nil {
		t.Fatalf("NormalizeImmutableID: %v", err)
	}
	if got != "/xmWb4aLEdC0LQDAT8lk/w==" {
		t.Fatalf("ImmutableID = %q", got)
	}
}

func TestNormalizeImmutableID_AcceptsExistingBase64(t *testing.T) {
	got, err := NormalizeImmutableID("/xmWb4aLEdC0LQDAT8lk/w==")
	if err != nil {
		t.Fatalf("NormalizeImmutableID: %v", err)
	}
	if got != "/xmWb4aLEdC0LQDAT8lk/w==" {
		t.Fatalf("ImmutableID = %q", got)
	}
}

func TestApplyEntraProfileAddsSyntheticImmutableID(t *testing.T) {
	attrs, err := ApplyEntraProfile(claimdomain.Attributes{
		"object_guid": {"6f9619ff-8b86-d011-b42d-00c04fc964ff"},
	}, &EntraFederationProfile{SourceAnchorAttribute: "object_guid"})
	if err != nil {
		t.Fatalf("ApplyEntraProfile: %v", err)
	}
	if got := attrs[EntraImmutableIDAttribute]; len(got) != 1 || got[0] != "/xmWb4aLEdC0LQDAT8lk/w==" {
		t.Fatalf("entra_immutable_id = %#v", got)
	}
}

func TestApplyEntraProfileRejectsMissingSourceAnchor(t *testing.T) {
	if _, err := ApplyEntraProfile(claimdomain.Attributes{}, &EntraFederationProfile{SourceAnchorAttribute: "object_guid"}); err == nil {
		t.Fatal("expected missing sourceAnchor error")
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
