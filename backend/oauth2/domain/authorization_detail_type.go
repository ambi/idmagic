package domain

// AuthorizationDetailType（RFC 9396 / ADR-050 の登録 type 定義）の双子定義。
// internal/shared/spec/oauth2.go から移設 (wi-173, ADR-089)。実行時インスタンスの
// AuthorizationDetail は [[wi-181]] 側の型からも参照されるため shared に残置しており、
// 本ファイルの型（登録スキーマ）とは依存関係がない。

import (
	"time"

	z "github.com/Oudwins/zog"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// AuthorizationDetailFieldSemantics は authorization_details の登録スキーマで
// 各フィールドが担うダウンスコープ半順序を表す (RFC 9396 / ADR-050)。
type AuthorizationDetailFieldSemantics string

const (
	DetailFieldSet    AuthorizationDetailFieldSemantics = "set"     // 集合包含: 要求 ⊆ 同意/元
	DetailFieldAtMost AuthorizationDetailFieldSemantics = "at_most" // 単調減少: 要求 ≤ 同意/元
	DetailFieldEnum   AuthorizationDetailFieldSemantics = "enum"    // 許可列挙からの完全一致
	DetailFieldExact  AuthorizationDetailFieldSemantics = "exact"   // 不透明値の完全一致
)

func (s AuthorizationDetailFieldSemantics) Valid() bool {
	switch s {
	case DetailFieldSet, DetailFieldAtMost, DetailFieldEnum, DetailFieldExact:
		return true
	}
	return false
}

// AuthorizationDetailTypeState は登録 type の運用状態を表す (ADR-050)。
type AuthorizationDetailTypeState string

const (
	DetailTypeEnabled  AuthorizationDetailTypeState = "Enabled"
	DetailTypeDisabled AuthorizationDetailTypeState = "Disabled"
)

func (s AuthorizationDetailTypeState) Valid() bool {
	switch s {
	case DetailTypeEnabled, DetailTypeDisabled:
		return true
	}
	return false
}

// AuthorizationDetailFieldRule は登録スキーマの 1 フィールド規則。ダウンスコープ半順序と
// 許可値を定義する。
type AuthorizationDetailFieldRule struct {
	Name      string                            `json:"name"`
	Semantics AuthorizationDetailFieldSemantics `json:"semantics"`
	Required  bool                              `json:"required"`
	Allowed   []string                          `json:"allowed,omitempty"`
}

// AuthorizationDetailsSchema はある type の構造的スキーマ。受理するフィールドと
// 各フィールドの半順序を列挙する。
type AuthorizationDetailsSchema struct {
	Rules []AuthorizationDetailFieldRule `json:"rules"`
}

// AuthorizationDetailType はテナントが登録する authorization_details の type 定義 (ADR-050)。
type AuthorizationDetailType struct {
	TenantID        string                       `json:"tenant_id"`
	Type            string                       `json:"type"`
	Description     string                       `json:"description,omitempty"`
	Schema          AuthorizationDetailsSchema   `json:"schema"`
	DisplayTemplate string                       `json:"display_template"`
	State           AuthorizationDetailTypeState `json:"state"`
	CreatedAt       time.Time                    `json:"created_at"`
	UpdatedAt       time.Time                    `json:"updated_at"`
}

var authorizationDetailTypeSchema = z.Struct(z.Shape{
	"TenantID":        z.String().Min(1).Required(),
	"Type":            z.String().Min(1).Required(),
	"DisplayTemplate": z.String().Min(1).Required(),
	"State": z.StringLike[AuthorizationDetailTypeState]().TestFunc(
		func(value *AuthorizationDetailTypeState, _ z.Ctx) bool { return value.Valid() },
		z.Message("authorization detail type state is not in enum"),
	).Required(),
	"Schema": z.Struct(z.Shape{
		"Rules": z.Slice(z.Struct(z.Shape{
			"Name": z.String().Min(1).Required(),
			"Semantics": z.StringLike[AuthorizationDetailFieldSemantics]().TestFunc(
				func(value *AuthorizationDetailFieldSemantics, _ z.Ctx) bool { return value.Valid() },
				z.Message("authorization detail field semantics is not in enum"),
			).Required(),
		})).Min(1).Required(),
	}),
	"CreatedAt": z.Time().Required(),
	"UpdatedAt": z.Time().Required(),
})

func (t AuthorizationDetailType) Validate() error {
	return spec.Validate(authorizationDetailTypeSchema, &t)
}
