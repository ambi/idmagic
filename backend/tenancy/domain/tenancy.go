// Package domain は Tenancy bounded context の業務ドメイン型を所有する
// (ADR-089, wi-179)。
package domain

import (
	"regexp"
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
