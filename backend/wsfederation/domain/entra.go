package domain

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// EntraFederationProfile は、テナントに一つの Entra の RP の定型設定である。Entra はどのドメインの
// 要求でも同じ realm を送るので、ドメインごとの IssuerUri をこの中に持つ。
type EntraFederationProfile struct {
	Domains               []EntraDomainFederation `json:"domains"`
	SourceAnchorAttribute string                  `json:"source_anchor_attribute"`
	ImmutableIDAttribute  string                  `json:"immutable_id_attribute"`
	PassiveLogOnURI       string                  `json:"passive_logon_uri,omitempty"`
	ActiveLogOnURI        string                  `json:"active_logon_uri,omitempty"`
	MetadataExchangeURI   string                  `json:"metadata_exchange_uri,omitempty"`
}

// EntraDomainFederation は、Entra でフェデレーションした一つのドメインと、その IssuerUri である。
type EntraDomainFederation struct {
	Domain    string `json:"domain"`
	IssuerURI string `json:"issuer_uri"`
}

const (
	// EntraRealm は、Entra が wtrealm、AppliesTo、audience に使う自身の識別子である。
	EntraRealm                   = "urn:federation:MicrosoftOnline"
	EntraUPNClaim                = "http://schemas.xmlsoap.org/claims/UPN"
	EntraNameIdentifierClaim     = "http://schemas.xmlsoap.org/claims/nameidentifier"
	EntraImmutableIDClaim        = "http://schemas.microsoft.com/LiveID/Federation/2008/05/ImmutableID"
	EntraPersistentNameIDFormat  = "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
	EntraImmutableIDAttribute    = "entra_immutable_id"
	DefaultEntraSourceAnchorAttr = "object_guid"
	DefaultEntraUPNAttribute     = "preferred_username"
)

var (
	// ErrEntraDomainNotFederated は、UPN のドメインがどの構成済みのドメインにも属さないことを表す。
	ErrEntraDomainNotFederated = errors.New("entra federation: the UPN domain is not federated")
	// ErrEntraIssuerInUse は、IssuerUri がほかのドメインで使われていることを表す。Entra は同じ
	// IssuerUri を二つのドメインに設定できない。
	ErrEntraIssuerInUse = errors.New("entra federation: the IssuerUri is used by another domain")
	// ErrEntraSourceAnchorChanged は、ほかのドメインを構成している間に sourceAnchor の属性を変える
	// 要求を表す。変えると、そのドメインの利用者の ImmutableID が変わる。
	ErrEntraSourceAnchorChanged = errors.New("entra federation: the sourceAnchor attribute differs from the configured domains")
)

// IssuerFor は、UPN のドメインに最も長く一致する構成済みのドメインの IssuerUri を返す。サブドメインの
// 利用者は、Entra が親のドメインの設定を継ぐので、親のドメインの IssuerUri で発行する。
func (p EntraFederationProfile) IssuerFor(upn string) (string, error) {
	_, upnDomain, ok := strings.CutLast(upn, "@")
	if !ok {
		return "", ErrEntraDomainNotFederated
	}
	upnDomain = strings.ToLower(upnDomain)
	issuer, matched := "", ""
	for _, d := range p.Domains {
		if (upnDomain == d.Domain || strings.HasSuffix(upnDomain, "."+d.Domain)) && len(d.Domain) > len(matched) {
			issuer, matched = d.IssuerURI, d.Domain
		}
	}
	if matched == "" {
		return "", ErrEntraDomainNotFederated
	}
	return issuer, nil
}

// WithDomain は、ドメインの組を追加または置き換え、sourceAnchor の属性を設定した定型設定を返す。
// 受け取った定型設定は変えない。added は、そのドメインを新たに加えたかを表す。
func (p EntraFederationProfile) WithDomain(domain EntraDomainFederation, sourceAnchorAttribute string) (EntraFederationProfile, bool, error) {
	added := true
	domains := make([]EntraDomainFederation, 0, len(p.Domains)+1)
	for _, d := range p.Domains {
		if d.Domain == domain.Domain {
			added = false
			continue
		}
		if d.IssuerURI == domain.IssuerURI {
			return EntraFederationProfile{}, false, ErrEntraIssuerInUse
		}
		if sourceAnchorAttribute != p.SourceAnchorAttribute {
			return EntraFederationProfile{}, false, ErrEntraSourceAnchorChanged
		}
		domains = append(domains, d)
	}
	domains = append(domains, domain)
	out := p
	out.Domains = domains
	out.SourceAnchorAttribute = sourceAnchorAttribute
	return out, added, nil
}

// EntraAttributeDefs は、sourceAnchor の属性を公開できるときだけ、合成の ImmutableID の属性を公開
// できる定義に加える。ImmutableID は sourceAnchor の値そのものを符号化したものだからである。
func EntraAttributeDefs(defs []userdomain.UserAttributeDef, profile *EntraFederationProfile) []userdomain.UserAttributeDef {
	if profile == nil || !claimdomain.IsAttributeReleasable(profile.sourceAnchorAttribute(), defs) {
		return defs
	}
	return append(slices.Clone(defs), userdomain.UserAttributeDef{
		Key: profile.immutableIDAttribute(), Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityAdminReadable,
	})
}

func (p EntraFederationProfile) sourceAnchorAttribute() string {
	if attr := strings.TrimSpace(p.SourceAnchorAttribute); attr != "" {
		return attr
	}
	return DefaultEntraSourceAnchorAttr
}

func (p EntraFederationProfile) immutableIDAttribute() string {
	if attr := strings.TrimSpace(p.ImmutableIDAttribute); attr != "" {
		return attr
	}
	return EntraImmutableIDAttribute
}

// BuildEntraClaimPolicy は Microsoft Entra domain federation 用 claim preset を返す。
func BuildEntraClaimPolicy() claimdomain.ClaimMappingPolicy {
	return claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{
			Format:          EntraPersistentNameIDFormat,
			SourceAttribute: EntraImmutableIDAttribute,
		},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: EntraUPNClaim, Source: claimdomain.ClaimSourceUserAttribute, SourceKey: DefaultEntraUPNAttribute, Required: true},
			{ClaimType: EntraNameIdentifierClaim, Source: claimdomain.ClaimSourceNameID, Required: true},
			{ClaimType: EntraImmutableIDClaim, Source: claimdomain.ClaimSourceUserAttribute, SourceKey: EntraImmutableIDAttribute, Required: true},
		},
	}
}

// ApplyEntraProfile derives the ImmutableID synthetic attribute required by Entra.
func ApplyEntraProfile(attrs claimdomain.Attributes, profile *EntraFederationProfile) (claimdomain.Attributes, error) {
	if profile == nil {
		return attrs, nil
	}
	sourceAttr := profile.sourceAnchorAttribute()
	raw, ok := firstNonEmptyAttribute(attrs[sourceAttr])
	if !ok {
		return nil, fmt.Errorf("entra federation: sourceAnchor attribute %q has no value", sourceAttr)
	}
	immutableID, err := NormalizeImmutableID(raw)
	if err != nil {
		return nil, fmt.Errorf("entra federation: sourceAnchor attribute %q: %w", sourceAttr, err)
	}
	targetAttr := profile.immutableIDAttribute()
	out := make(claimdomain.Attributes, len(attrs)+1)
	for key, values := range attrs {
		out[key] = append([]string(nil), values...)
	}
	out[targetAttr] = []string{immutableID}
	return out, nil
}

func firstNonEmptyAttribute(values []string) (string, bool) {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value, true
		}
	}
	return "", false
}

// NormalizeImmutableID returns the Azure AD ImmutableID base64 value for a sourceAnchor.
func NormalizeImmutableID(value string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", fmt.Errorf("empty sourceAnchor")
	}
	if guidBytes, ok := parseGUIDToMicrosoftBytes(v); ok {
		return base64.StdEncoding.EncodeToString(guidBytes), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(v)
	if err == nil && len(decoded) > 0 {
		return v, nil
	}
	return "", fmt.Errorf("must be a GUID or base64 ImmutableID")
}

func parseGUIDToMicrosoftBytes(value string) ([]byte, bool) {
	compact := strings.ReplaceAll(strings.TrimSpace(value), "-", "")
	if len(compact) != 32 {
		return nil, false
	}
	raw, err := hex.DecodeString(compact)
	if err != nil || len(raw) != 16 {
		return nil, false
	}
	return []byte{
		raw[3], raw[2], raw[1], raw[0],
		raw[5], raw[4],
		raw[7], raw[6],
		raw[8], raw[9],
		raw[10], raw[11], raw[12], raw[13], raw[14], raw[15],
	}, true
}
