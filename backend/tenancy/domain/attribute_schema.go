package domain

import "time"

// UserAttributeDef は管理 API が受け渡す定義であり、型と公開範囲の意味の検証は IdManagement に委ねる。
type UserAttributeDef struct {
	Key            string  `json:"key"`
	Label          string  `json:"label,omitempty"`
	Type           string  `json:"type"`
	MultiValued    bool    `json:"multi_valued"`
	Required       bool    `json:"required"`
	EditableByUser bool    `json:"editable_by_user"`
	Unique         bool    `json:"unique"`
	ClaimName      *string `json:"claim_name,omitempty"`
	OIDCScope      *string `json:"oidc_scope,omitempty"`
	Visibility     string  `json:"visibility"`
	PII            bool    `json:"pii"`
}

type GroupAttributeDef struct {
	Key         string `json:"key"`
	Label       string `json:"label,omitempty"`
	Type        string `json:"type"`
	MultiValued bool   `json:"multi_valued"`
	Required    bool   `json:"required"`
}

type TenantUserAttributeSchema struct {
	TenantID   string             `json:"tenant_id"`
	Attributes []UserAttributeDef `json:"attributes"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

type TenantGroupAttributeSchema struct {
	TenantID   string              `json:"tenant_id"`
	Attributes []GroupAttributeDef `json:"attributes"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}
