package spec

// Tenancy bounded context の双子定義 (ADR-032 / ADR-034)。

import "time"

// DefaultTenantID は既定テナントの不変 UUID 代理キー (ADR-085)。tenant_id FK・
// 内部のテナント参照はこの値を用いる。DefaultRealm は URL `/realms/{realm}/` 等の
// 公開語彙に現れる既定 realm slug。
const (
	DefaultTenantID = "00000000-0000-4000-8000-000000000000"
	DefaultRealm    = "default"
)

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
	return validate(tenantSchema, &t)
}

// PasswordPolicyOverride はテナント固有の objectives.PasswordPolicy 上書き値。
// SCL `PasswordPolicyOverride` の双子定義。省略フィールドは global default を継承する。
type PasswordPolicyOverride struct {
	MinLength    *int `json:"min_length,omitempty"`
	MaxLength    *int `json:"max_length,omitempty"`
	HistoryDepth *int `json:"history_depth,omitempty"`
}
