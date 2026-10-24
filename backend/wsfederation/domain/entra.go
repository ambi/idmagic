package domain

import (
	"errors"
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
	DefaultEntraSourceAnchorAttr = "on_premises_immutable_id"
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
// できる定義に加える。ImmutableID は sourceAnchor の値そのものだからである。
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
	immutableID, err := ImmutableIDOf(attrs, profile)
	if err != nil {
		return nil, err
	}
	targetAttr := profile.immutableIDAttribute()
	out := make(claimdomain.Attributes, len(attrs)+1)
	for key, values := range attrs {
		out[key] = append([]string(nil), values...)
	}
	out[targetAttr] = []string{immutableID}
	return out, nil
}

var (
	// ErrSourceAnchorMissing は、利用者が sourceAnchor の値を持たないことを表す。
	ErrSourceAnchorMissing = errors.New("entra federation: the user has no sourceAnchor value")
	// ErrSourceAnchorInvalid は、sourceAnchor の値が ImmutableID として使えないことを表す。
	ErrSourceAnchorInvalid = errors.New("entra federation: the sourceAnchor value cannot be an ImmutableID")
	// ErrSourceAnchorNotUnique は、sourceAnchor の属性が属性スキーマで unique でないことを表す。
	ErrSourceAnchorNotUnique = errors.New("entra federation: the sourceAnchor attribute is not unique")
	// ErrSourceAnchorEditable は、sourceAnchor の属性を利用者が編集できることを表す。
	ErrSourceAnchorEditable = errors.New("entra federation: the sourceAnchor attribute is editable by the user")
)

// ImmutableIDOf は、sourceAnchor の属性の値を変えずに ImmutableID として返す。Entra Connect は
// sourceAnchor を immutableId と同じものとして扱い、文字列でない属性は同期の側で base64 にする。
// 値を変換しないので、IdManagement が守る値の一意性が、そのまま ImmutableID の一意性になる。
func ImmutableIDOf(attrs claimdomain.Attributes, profile *EntraFederationProfile) (string, error) {
	values := attrs[profile.sourceAnchorAttribute()]
	if len(values) == 0 || values[0] == "" {
		return "", ErrSourceAnchorMissing
	}
	value := values[0]
	// Entra は `@` を含む属性を sourceAnchor に選ばせない。空白だけの値も利用者を識別しない。
	if strings.TrimSpace(value) == "" || strings.Contains(value, "@") {
		return "", ErrSourceAnchorInvalid
	}
	return value, nil
}

// CheckEntraSourceAnchorAttribute は、sourceAnchor の属性が、ほかの利用者と重ならず、利用者が
// 書き換えられない値かを、テナントの実効の属性の定義から確かめる。定義のない属性は重なりを
// 守られていないものとして扱う。
func CheckEntraSourceAnchorAttribute(defs []userdomain.UserAttributeDef, profile *EntraFederationProfile) error {
	attribute := profile.sourceAnchorAttribute()
	for _, def := range defs {
		if def.Key != attribute {
			continue
		}
		if !def.Unique {
			return ErrSourceAnchorNotUnique
		}
		if def.EditableByUser {
			return ErrSourceAnchorEditable
		}
		return nil
	}
	return ErrSourceAnchorNotUnique
}

// SourceAnchorValueProblem は、構成の時点に sourceAnchor の値を使えない利用者の件数と、ユーザー名の
// 順で先頭の利用者である。
type SourceAnchorValueProblem struct {
	Err             error // ErrSourceAnchorMissing または ErrSourceAnchorInvalid
	UserCount       int
	SampleUsernames []string
}

// sourceAnchorSampleSize は、構成の拒否に載せる利用者のユーザー名の数である。
const sourceAnchorSampleSize = 5

// UnfederatedUsers は、users のうち、profile で UPN がどの構成済みのドメインにも属さない削除されて
// いない利用者の数と、ユーザー名の順で先頭の利用者のユーザー名を返す。
func UnfederatedUsers(profile EntraFederationProfile, users []*userdomain.User) (int, []string) {
	var names []string
	for _, user := range users {
		if user.IsDeleted() {
			continue
		}
		if _, err := profile.IssuerFor(user.PreferredUsername); err != nil {
			names = append(names, user.PreferredUsername)
		}
	}
	slices.Sort(names)
	return len(names), names[:min(len(names), sourceAnchorSampleSize)]
}

// CheckEntraSourceAnchorValues は、profile で UPN がフェデレーションするドメインに属する利用者の
// sourceAnchor の値を確かめる。値のない利用者がいればそれを、いなければ使えない値の利用者を返し、
// 問題がなければ nil を返す。フェデレーションしないドメインの利用者は確かめない。
func CheckEntraSourceAnchorValues(profile EntraFederationProfile, users []*userdomain.User) *SourceAnchorValueProblem {
	affected := map[error][]string{}
	for _, user := range users {
		if user.IsDeleted() {
			continue
		}
		if _, err := profile.IssuerFor(user.PreferredUsername); err != nil {
			continue
		}
		if _, err := ImmutableIDOf(claimdomain.ResolveUserAttributes(*user), &profile); err != nil {
			affected[err] = append(affected[err], user.PreferredUsername)
		}
	}
	for _, kind := range []error{ErrSourceAnchorMissing, ErrSourceAnchorInvalid} {
		names := affected[kind]
		if len(names) == 0 {
			continue
		}
		slices.Sort(names)
		return &SourceAnchorValueProblem{Err: kind, UserCount: len(names), SampleUsernames: names[:min(len(names), sourceAnchorSampleSize)]}
	}
	return nil
}
