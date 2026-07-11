// Package domain は Tenancy bounded context の業務ドメイン型を所有する
// (ADR-089, wi-179)。
package domain

import (
	"math"
	"regexp"
	"strconv"
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

type Tenant struct {
	ID                     string                  `json:"id"`
	Realm                  string                  `json:"realm"`
	DisplayName            string                  `json:"display_name"`
	Status                 TenantStatus            `json:"status"`
	PasswordPolicyOverride *PasswordPolicyOverride `json:"password_policy_override,omitempty"`
	CreatedAt              time.Time               `json:"created_at"`
	UpdatedAt              time.Time               `json:"updated_at"`
	DisabledAt             *time.Time              `json:"disabled_at,omitempty"`
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

var tenantIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

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

// TenantBranding はテナント単位の hosted UI ブランディング設定 (wi-89, ADR-096)。SCL
// `TenantBranding` の双子定義。Tenant aggregate には埋め込まず、TenantUserAttributeSchema
// と同じ理由で独立 entity として持つ。全フィールドは空文字列 (ゼロ値) を「未設定」として扱う。
type TenantBranding struct {
	TenantID         string    `json:"tenant_id"`
	ProductName      string    `json:"product_name,omitempty"`
	LogoObjectKey    string    `json:"logo_object_key,omitempty"`
	LogoURL          string    `json:"logo_url,omitempty"`
	FaviconObjectKey string    `json:"favicon_object_key,omitempty"`
	FaviconURL       string    `json:"favicon_url,omitempty"`
	PrimaryColor     string    `json:"primary_color,omitempty"`
	AccentColor      string    `json:"accent_color,omitempty"`
	SupportURL       string    `json:"support_url,omitempty"`
	LegalURL         string    `json:"legal_url,omitempty"`
	FooterText       string    `json:"footer_text,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (b TenantBranding) Validate() error {
	return spec.Validate(tenantBrandingSchema, &b)
}

// IsConfigured は branding が presentational に意味のある値を 1 つでも持つかを返す。
// 全フィールドが未設定 (ゼロ値) なら GetTenantBranding はシステム既定にフォールバックする
// (ADR-096 決定 8)。
func (b TenantBranding) IsConfigured() bool {
	return b.ProductName != "" || b.LogoURL != "" || b.FaviconURL != "" ||
		b.PrimaryColor != "" || b.AccentColor != "" || b.SupportURL != "" ||
		b.LegalURL != "" || b.FooterText != ""
}

var (
	tenantBrandingHexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	tenantBrandingHTTPSPattern    = regexp.MustCompile(`^https://`)
)

// tenantBrandingContrastBackground は AuthShell のカード背景色。ブランドカラーの WCAG AA
// コントラスト比 (4.5:1) をこの背景に対して検証する (ADR-096 決定 3)。
const tenantBrandingContrastBackground = "#ffffff"

// validTenantBrandingColor は空文字列 (未設定) を許容しつつ、値がある場合は `#rrggbb`
// 形式かつ既定背景に対する WCAG AA コントラスト比 (4.5:1) を満たすことを要求する。
func validTenantBrandingColor(value string) bool {
	if value == "" {
		return true
	}
	if !tenantBrandingHexColorPattern.MatchString(value) {
		return false
	}
	return contrastRatio(value, tenantBrandingContrastBackground) >= 4.5
}

// validTenantBrandingLink は空文字列 (未設定) を許容しつつ、値がある場合は https scheme
// のみを allowlist する (ADR-096 決定 5)。
func validTenantBrandingLink(value string) bool {
	return value == "" || tenantBrandingHTTPSPattern.MatchString(value)
}

// contrastRatio は 2 色の WCAG 2.x 相対輝度からコントラスト比を計算する。
func contrastRatio(hexA, hexB string) float64 {
	la, lb := relativeLuminance(hexA), relativeLuminance(hexB)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relativeLuminance(hex string) float64 {
	r, g, b := hexRGB(hex)
	return 0.2126*linearizeSRGB(r) + 0.7152*linearizeSRGB(g) + 0.0722*linearizeSRGB(b)
}

func hexRGB(hex string) (r, g, b float64) {
	v, err := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	if err != nil {
		return 0, 0, 0
	}
	return float64((v>>16)&0xff) / 255, float64((v>>8)&0xff) / 255, float64(v&0xff) / 255
}

func linearizeSRGB(c float64) float64 {
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
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
		z.Message("primary_color must be #rrggbb and meet WCAG AA contrast against the default background"),
	),
	"AccentColor": z.String().TestFunc(
		func(value *string, _ z.Ctx) bool { return value != nil && validTenantBrandingColor(*value) },
		z.Message("accent_color must be #rrggbb and meet WCAG AA contrast against the default background"),
	),
	"SupportURL": z.String().Max(2048).TestFunc(
		func(value *string, _ z.Ctx) bool { return value != nil && validTenantBrandingLink(*value) },
		z.Message("support_url must use the https scheme"),
	),
	"LegalURL": z.String().Max(2048).TestFunc(
		func(value *string, _ z.Ctx) bool { return value != nil && validTenantBrandingLink(*value) },
		z.Message("legal_url must use the https scheme"),
	),
	"FooterText": z.String().Max(280),
	"CreatedAt":  z.Time().Required(),
	"UpdatedAt":  z.Time().Required(),
})

// TenantBrandingAsset はテナントの branding ロゴ / favicon の保存済み blob (wi-89,
// ADR-096)。ADR-073 の Application icon 保存パターンを再利用するが、専用テーブル・専用
// object_key 空間に分離する。
type TenantBrandingAsset struct {
	TenantID    string                  `json:"tenant_id"`
	Kind        TenantBrandingAssetKind `json:"kind"`
	ObjectKey   string                  `json:"object_key"`
	ContentType string                  `json:"content_type"`
	SizeBytes   int                     `json:"size_bytes"`
	Data        []byte                  `json:"-"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}
