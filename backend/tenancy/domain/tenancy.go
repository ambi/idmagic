// Package domain は Tenancy bounded context の業務ドメイン型を所有する
// (ADR-089, wi-179)。
package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	z "github.com/Oudwins/zog"

	"github.com/ambi/idmagic/backend/shared/kernel"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// Tenancy bounded context の双子定義 (ADR-032 / ADR-034)。

// DefaultTenantID は既定テナントの不変 UUID 代理キー (ADR-085)。tenant_id FK・
// 内部のテナント参照はこの値を用いる。DefaultRealm は URL `/realms/{realm}/` 等の
// 公開語彙に現れる既定 realm slug。真の値は shared/kernel が持つ (wi-179, ADR-089):
// shared/spec の AuthZEN policy 述語からも参照され、tenancy/domain は import cycle に
// なるため re-export する。
const (
	DefaultTenantID = kernel.DefaultTenantID
	DefaultRealm    = kernel.DefaultRealm
)

type TenantStatus string

const (
	TenantStatusActive   TenantStatus = "active"
	TenantStatusDisabled TenantStatus = "disabled"
)

func (s TenantStatus) Valid() bool {
	return s == TenantStatusActive || s == TenantStatusDisabled
}

// TenantEndpointStyle はテナントの正規ロケーションの形 (ADR-144)。SCL
// `TenantEndpointStyle` の双子定義。1 テナントは 1 つの正規ロケーションしか持たず、
// issuer / cookie scope / WebAuthn RP ID はすべてそこから導出される。
type TenantEndpointStyle string

const (
	// TenantEndpointStylePath は共有ホスト上の `{base}/realms/{realm}`。ワイルドカード
	// DNS も証明書も要らないため既定であり、tenant base domain を持たない配備では
	// 唯一の選択肢になる。
	TenantEndpointStylePath TenantEndpointStyle = "path"
	// TenantEndpointStyleSubdomain はテナント固有ホスト `{realm}.{base}`。テナント固有
	// origin を必要とする機能 (__Host- cookie、テナント別 RP ID) が成立する。
	TenantEndpointStyleSubdomain TenantEndpointStyle = "subdomain"
)

func (s TenantEndpointStyle) Valid() bool {
	return s == TenantEndpointStylePath || s == TenantEndpointStyleSubdomain
}

type Tenant struct {
	ID                     string                  `json:"id"`
	Realm                  string                  `json:"realm"`
	DisplayName            string                  `json:"display_name"`
	Status                 TenantStatus            `json:"status"`
	PasswordPolicyOverride *PasswordPolicyOverride `json:"password_policy_override,omitempty"`
	// DefaultLocale は通知の locale 解決の第 2 段 (ADR-142 決定 7)。nil / 空文字列は
	// 「システム既定 locale を使う」を意味する。値の妥当性 (同梱翻訳を持つ locale か)
	// は shared/notification のカタログが正本なので、ここでは形だけを検証する。
	DefaultLocale *string `json:"default_locale,omitempty"`
	// EndpointStyle はこのテナントの正規ロケーションの形 (ADR-144)。ゼロ値は
	// TenantEndpointStylePath として扱う: 既存行と、この列を知らない呼び出し元が
	// 作る Tenant が、暗黙に到達不能な subdomain 側へ倒れないようにするため。
	EndpointStyle TenantEndpointStyle `json:"endpoint_style,omitempty"`
	Quota         *TenantQuota        `json:"quota,omitempty"`
	Usage         *TenantUsage        `json:"usage,omitempty"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	DisabledAt    *time.Time          `json:"disabled_at,omitempty"`
}

func (t Tenant) Validate() error {
	return spec.Validate(tenantSchema, &t)
}

// PasswordPolicyOverride はテナント固有の objectives.PasswordPolicy 上書き値。
// SCL `PasswordPolicyOverride` の双子定義。省略フィールドは global default を継承する。
type PasswordPolicyOverride struct {
	MinLength    *int `json:"min_length,omitempty"`
	MaxLength    *int `json:"max_length,omitempty"`
	HistoryDepth *int `json:"history_depth,omitempty"`
}

type TenantQuota struct {
	Users                *int `json:"users,omitempty"`
	Groups               *int `json:"groups,omitempty"`
	Agents               *int `json:"agents,omitempty"`
	Applications         *int `json:"applications,omitempty"`
	OAuth2Clients        *int `json:"oauth2_clients,omitempty"`
	ActiveSessions       *int `json:"active_sessions,omitempty"`
	Consents             *int `json:"consents,omitempty"`
	ActiveJobs           *int `json:"active_jobs,omitempty"`
	AuditEventsRetained  *int `json:"audit_events_retained,omitempty"`
	ExportArtifactsBytes *int `json:"export_artifacts_bytes,omitempty"`
}

type TenantUsage struct {
	Users                int `json:"users"`
	Groups               int `json:"groups"`
	Agents               int `json:"agents"`
	Applications         int `json:"applications"`
	OAuth2Clients        int `json:"oauth2_clients"`
	ActiveSessions       int `json:"active_sessions"`
	Consents             int `json:"consents"`
	ActiveJobs           int `json:"active_jobs"`
	AuditEventsRetained  int `json:"audit_events_retained"`
	ExportArtifactsBytes int `json:"export_artifacts_bytes"`
}

type QuotaExceededError struct {
	TenantID string
	Resource string
}

func (e *QuotaExceededError) Error() string {
	return "quota exceeded for resource " + e.Resource + " in tenant " + e.TenantID
}

// IsQuotaExceeded / GetResource / GetTenantID satisfy the quotaExceeded
// interface support_http.ErrorHandler matches via errors.As, so quota
// rejections get a dedicated 422 response and metrics.RecordQuotaExceeded
// instead of falling through as an unhandled 500 (wi-160 T004).
func (e *QuotaExceededError) IsQuotaExceeded() bool { return true }
func (e *QuotaExceededError) GetResource() string   { return e.Resource }
func (e *QuotaExceededError) GetTenantID() string   { return e.TenantID }

// Hard Quota resource identifiers (ADR-134). These are the exact strings
// QuotaRepository implementations switch on and TenantQuota/TenantUsage JSON
// tags use; defining them here lets call sites avoid retyping raw strings
// across the ~8 bounded contexts that enforce quota at creation time.
const (
	ResourceUsers          = "users"
	ResourceGroups         = "groups"
	ResourceAgents         = "agents"
	ResourceApplications   = "applications"
	ResourceOAuth2Clients  = "oauth2_clients"
	ResourceActiveSessions = "active_sessions"
	ResourceConsents       = "consents"
	ResourceActiveJobs     = "active_jobs"
)

// DefaultTenantQuota is the ADR-134 baseline Hard Quota applied when a tenant
// has no per-resource override. Unlike TenantQuota's fields, a system default
// is never "unset", so plain ints are the right shape here (not *int) — it is
// the single source of truth for these numbers: both the memory and postgres
// QuotaRepository implementations resolve limits through
// TenantQuota.EffectiveLimit instead of duplicating the values themselves, so
// the two backends cannot silently drift apart (wi-160 scope: "memory /
// postgres の両方で同じ quota enforcement を満たす"). An unrecognized resource
// resolves to the zero value (0), which EffectiveLimit's caller treats as
// fail-closed rather than unlimited.
var DefaultTenantQuota = map[string]int{
	ResourceUsers:          10000,
	ResourceGroups:         1000,
	ResourceAgents:         100,
	ResourceApplications:   50,
	ResourceOAuth2Clients:  100,
	ResourceActiveSessions: 50000,
	ResourceConsents:       10000,
	ResourceActiveJobs:     10,
}

// resourceOverride returns q's explicit limit for resource, or nil when q has
// not customized it (or q is nil). Unknown resources also return nil. This is
// the one place *int is warranted: "not customized" is a meaningful third
// state distinct from any concrete limit.
func (q *TenantQuota) resourceOverride(resource string) *int {
	if q == nil {
		return nil
	}
	switch resource {
	case ResourceUsers:
		return q.Users
	case ResourceGroups:
		return q.Groups
	case ResourceAgents:
		return q.Agents
	case ResourceApplications:
		return q.Applications
	case ResourceOAuth2Clients:
		return q.OAuth2Clients
	case ResourceActiveSessions:
		return q.ActiveSessions
	case ResourceConsents:
		return q.Consents
	case ResourceActiveJobs:
		return q.ActiveJobs
	default:
		return nil
	}
}

// EffectiveLimit returns the Hard Quota limit for resource: q's override when
// set, otherwise the ADR-134 system default (DefaultTenantQuota). Unknown
// resources return 0 (fail-closed: an unrecognized resource is never treated
// as unlimited).
func (q *TenantQuota) EffectiveLimit(resource string) int {
	if override := q.resourceOverride(resource); override != nil {
		return *override
	}
	return DefaultTenantQuota[resource]
}

var tenantIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// newRealmPattern は realm を単一 DNS ラベルとして受け入れる形 (ADR-144)。
// tenantIDPattern との差は hyphen 終端を許さない点で、endpoint_style が Subdomain の
// とき realm はホスト名の最左ラベルになるため、`acme-` のような値を作らせない。
var newRealmPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// reservedRealms は realm として採らせないラベル。realm はサブドメインになりうるので、
// base domain 上に first-party ホストとして置きうる名前をテナントに奪われないようにする。
var reservedRealms = map[string]struct{}{
	"admin": {}, "www": {}, "api": {}, "login": {}, "id": {}, "sso": {},
	"mail": {}, "smtp": {}, "status": {}, "docs": {}, "app": {}, "static": {},
	"cdn": {}, "auth": {}, "account": {},
}

var (
	ErrRealmNotDNSLabel     = errors.New("tenant realm must be a single DNS label: lowercase alphanumerics and inner hyphens only")
	ErrRealmLooksLikeALabel = errors.New("tenant realm must not start with xn-- (reserved for IDNA A-labels)")
	ErrRealmReserved        = errors.New("tenant realm is reserved")
	ErrEndpointStyleUnknown = errors.New("tenant endpoint style is not in enum")
	ErrSubdomainStyleNoBase = errors.New("subdomain endpoint style requires a tenant base domain to be configured")
)

// ValidateNewRealm は**新規に採番する** realm を検証する (ADR-144)。Tenant.Validate とは
// 別に置くのは、厳格化した規則を既存 realm に遡って適用しないため: 遡及すると厳格化前に
// 作られたテナントの読み出しや起動が壊れる。
func ValidateNewRealm(realm string) error {
	if !newRealmPattern.MatchString(realm) || len(realm) > 63 {
		return ErrRealmNotDNSLabel
	}
	// xn-- は IDNA A-label の予約 prefix。punycode でない値がホスト名の最左ラベルに
	// 入ると、解決側とブラウザ側で別の名前として扱われうる。
	if strings.HasPrefix(realm, "xn--") {
		return ErrRealmLooksLikeALabel
	}
	if _, reserved := reservedRealms[realm]; reserved {
		return ErrRealmReserved
	}
	return nil
}

// ValidateEndpointStyleSelectable は配備構成のもとで style を選べるかを判定する。
// baseDomain は設定済みの tenant base domain (未設定なら空文字列)。
func ValidateEndpointStyleSelectable(style TenantEndpointStyle, baseDomain string) error {
	if !style.Valid() {
		return ErrEndpointStyleUnknown
	}
	if style == TenantEndpointStyleSubdomain && strings.TrimSpace(baseDomain) == "" {
		return ErrSubdomainStyleNoBase
	}
	return nil
}

// EffectiveEndpointStyle はゼロ値を Path として読む。永続化前の行や、この列を知らない
// 呼び出し元が組み立てた Tenant が、暗黙に到達不能な subdomain 側へ倒れないようにする。
func (t Tenant) EffectiveEndpointStyle() TenantEndpointStyle {
	if t.EndpointStyle == "" {
		return TenantEndpointStylePath
	}
	return t.EndpointStyle
}

// localeTagPattern は DefaultLocale の形 (2 文字の言語タグ)。
var localeTagPattern = regexp.MustCompile(`^[a-z]{2}$`)

var tenantSchema = z.Struct(z.Shape{
	"ID": z.String().Min(1).Required(),
	"Realm": z.String().Min(1).Max(63).TestFunc(
		func(value *string, _ z.Ctx) bool {
			return value != nil && tenantIDPattern.MatchString(*value) && *value != "admin"
		},
		z.Message("tenant realm must be a URL-safe slug and must not be admin"),
	).Required(),
	"DisplayName": z.String().Min(1).Max(200).Required(),
	"Status": z.StringLike[TenantStatus]().TestFunc(
		func(value *TenantStatus, _ z.Ctx) bool { return value.Valid() },
		z.Message("tenant status is not in enum"),
	).Required(),
	"DefaultLocale": z.Ptr(z.String().TestFunc(
		func(value *string, _ z.Ctx) bool { return value != nil && localeTagPattern.MatchString(*value) },
		z.Message("tenant default locale must be a two-letter language tag"),
	)),
	// ゼロ値は Path として読むので許す (EffectiveEndpointStyle)。
	"EndpointStyle": z.StringLike[TenantEndpointStyle]().TestFunc(
		func(value *TenantEndpointStyle, _ z.Ctx) bool { return *value == "" || value.Valid() },
		z.Message("tenant endpoint style is not in enum"),
	),
	"CreatedAt": z.Time().Required(),
	"UpdatedAt": z.Time().Required(),
})

// TenantBrandingAssetKind は TenantBranding が持つ画像アセットの種別 (wi-89, ADR-096)。
// SCL `TenantBrandingAssetKind` の双子定義。
type TenantBrandingAssetKind string

const (
	TenantBrandingAssetKindLogo    TenantBrandingAssetKind = "logo"
	TenantBrandingAssetKindFavicon TenantBrandingAssetKind = "favicon"
)

func (k TenantBrandingAssetKind) Valid() bool {
	return k == TenantBrandingAssetKindLogo || k == TenantBrandingAssetKindFavicon
}

// TenantFooterLink は hosted UI footer に表示する、順序固定の安全な外部リンク。
// ラベルは描画時にプレーンテキストとして扱い、URL は HTTPS だけを許可する。
type TenantFooterLink struct {
	Label string `json:"label,omitempty"`
	URL   string `json:"url,omitempty"`
}

func (l TenantFooterLink) IsSet() bool { return l.Label != "" || l.URL != "" }

// TenantBranding はテナント単位の hosted UI ブランディング設定 (wi-89, ADR-096)。SCL
// `TenantBranding` の双子定義。Tenant aggregate には埋め込まず、TenantUserAttributeSchema
// と同じ理由で独立 entity として持つ。全フィールドは空文字列 (ゼロ値) を「未設定」として扱う。
type TenantBranding struct {
	TenantID         string           `json:"tenant_id"`
	ProductName      string           `json:"product_name,omitempty"`
	LogoObjectKey    string           `json:"logo_object_key,omitempty"`
	LogoURL          string           `json:"logo_url,omitempty"`
	FaviconObjectKey string           `json:"favicon_object_key,omitempty"`
	FaviconURL       string           `json:"favicon_url,omitempty"`
	PrimaryColor     string           `json:"primary_color,omitempty"`
	AccentColor      string           `json:"accent_color,omitempty"`
	FooterLink1      TenantFooterLink `json:"footer_link_1"`
	FooterLink2      TenantFooterLink `json:"footer_link_2"`
	FooterText       string           `json:"footer_text,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

func (b TenantBranding) Validate() error {
	if err := spec.Validate(tenantBrandingSchema, &b); err != nil {
		return err
	}
	if !validTenantFooterLink(b.FooterLink1) || !validTenantFooterLink(b.FooterLink2) {
		return errors.New("footer links must be complete plaintext label and https URL pairs")
	}
	return nil
}

// IsConfigured は branding が presentational に意味のある値を 1 つでも持つかを返す。
// 全フィールドが未設定 (ゼロ値) なら GetTenantBranding はシステム既定にフォールバックする
// (ADR-096 決定 8)。
func (b TenantBranding) IsConfigured() bool {
	return b.ProductName != "" || b.LogoURL != "" || b.FaviconURL != "" ||
		b.PrimaryColor != "" || b.AccentColor != "" || b.FooterLink1.IsSet() ||
		b.FooterLink2.IsSet() || b.FooterText != ""
}

var (
	tenantBrandingHexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	tenantBrandingHTTPSPattern    = regexp.MustCompile(`^https://`)
)

// validTenantBrandingColor は空文字列 (未設定) を許容しつつ、値がある場合は `#rrggbb`
// 形式であることを要求する。コントラスト比は保存制約ではない (ADR-097)。
func validTenantBrandingColor(value string) bool {
	return value == "" || tenantBrandingHexColorPattern.MatchString(value)
}

// validTenantBrandingLink は空文字列 (未設定) を許容しつつ、値がある場合は https scheme
// のみを allowlist する (ADR-096 決定 5)。
func validTenantBrandingLink(value string) bool {
	return value == "" || tenantBrandingHTTPSPattern.MatchString(value)
}

var tenantBrandingSchema = z.Struct(z.Shape{
	"TenantID":         z.String().Min(1).Required(),
	"ProductName":      z.String().Max(80),
	"LogoObjectKey":    z.String(),
	"LogoURL":          z.String(),
	"FaviconObjectKey": z.String(),
	"FaviconURL":       z.String(),
	"PrimaryColor": z.String().TestFunc(
		func(value *string, _ z.Ctx) bool { return value != nil && validTenantBrandingColor(*value) },
		z.Message("primary_color must be #rrggbb"),
	),
	"AccentColor": z.String().TestFunc(
		func(value *string, _ z.Ctx) bool { return value != nil && validTenantBrandingColor(*value) },
		z.Message("accent_color must be #rrggbb"),
	),
	"FooterLink1": tenantFooterLinkSchema,
	"FooterLink2": tenantFooterLinkSchema,
	"FooterText":  z.String().Max(280),
	"CreatedAt":   z.Time().Required(),
	"UpdatedAt":   z.Time().Required(),
})

var tenantFooterLinkSchema = z.Struct(z.Shape{
	"Label": z.String().Max(80),
	"URL":   z.String().Max(2048),
})

func validTenantFooterLink(link TenantFooterLink) bool {
	if !link.IsSet() {
		return true
	}
	return len(link.Label) <= 80 && link.Label != "" && len(link.URL) <= 2048 && link.URL != "" && validTenantBrandingLink(link.URL)
}

// TenantBrandingAsset はテナントの branding ロゴ / favicon の保存済み blob (wi-89,
// ADR-096)。ADR-073 の Application icon 保存パターンを再利用するが、専用テーブル・専用
// object_key 空間に分離する。
type TenantBrandingAsset struct {
	TenantID    string                  `json:"tenant_id"`
	Kind        TenantBrandingAssetKind `json:"kind"`
	ID          string                  `json:"id"`
	ContentType string                  `json:"content_type"`
	SizeBytes   int                     `json:"size_bytes"`
	Data        []byte                  `json:"-"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}
