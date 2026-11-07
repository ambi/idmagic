package domain

// AuthorizationDetailType（RFC 9396 / 登録 type 定義）の双子定義。
// internal/shared/spec/oauth2.go から移設 (wi-173)。実行時インスタンスの
// AuthorizationDetail は [[wi-181]] 側の型からも参照されるため shared に残置しており、
// 本ファイルの型（登録スキーマ）とは依存関係がない。

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// AuthorizationDetailFieldSemantics は authorization_details の登録スキーマで
// 各フィールドが担うダウンスコープ半順序を表す (RFC 9396)。
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

// AuthorizationDetailTypeState は登録 type の運用状態を表す。
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

// AuthorizationDetailType はテナントが登録する authorization_details の type 定義。
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

func (t AuthorizationDetailType) Validate() error {
	return spec.All(
		spec.Required("TenantID", t.TenantID),
		spec.Required("Type", t.Type),
		spec.Required("DisplayTemplate", t.DisplayTemplate),
		spec.Required("State", t.State,
			spec.Must(AuthorizationDetailTypeState.Valid, "authorization detail type state is not in enum")),
		spec.Nested("Schema", append(
			spec.NonEmpty("Rules", t.Schema.Rules),
			spec.EachWith("Rules", t.Schema.Rules, func(rule AuthorizationDetailFieldRule) spec.Issues {
				return append(
					spec.Required("Name", rule.Name),
					spec.Required("Semantics", rule.Semantics, spec.Must(AuthorizationDetailFieldSemantics.Valid,
						"authorization detail field semantics is not in enum"))...,
				)
			})...,
		)),
		spec.Required("CreatedAt", t.CreatedAt),
		spec.Required("UpdatedAt", t.UpdatedAt),
	)
}
