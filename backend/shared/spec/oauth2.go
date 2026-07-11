package spec

// OAuth2 bounded context の双子定義のうち、client / consent / 認可詳細タイプ定義
// （OAuth2Client / Consent / AuthorizationDetailType 系）は internal/oauth2/domain へ
// 移設済み（wi-173, ADR-089）。本ファイルには authorization request / code / refresh /
// PAR / device / token claims（wi-181 で移設予定）と、それらが参照する実行時
// AuthorizationDetail（複数 context 予定の型から参照されるため shared に残置）が残る。

// AuthorizationDetail は RFC 9396 authorization_details の 1 要素。type で識別される
// 構造化された細粒度権限を表し、登録済み AuthorizationDetailType に対し fail-closed に検証する。
// AuthorizationDetailType 自体は internal/oauth2/domain へ移設済み (wi-173)。
type AuthorizationDetail struct {
	Type       string         `json:"type"`
	Locations  []string       `json:"locations,omitempty"`
	Actions    []string       `json:"actions,omitempty"`
	Datatypes  []string       `json:"datatypes,omitempty"`
	Identifier string         `json:"identifier,omitempty"`
	Privileges []string       `json:"privileges,omitempty"`
	Fields     map[string]any `json:"fields,omitempty"`
}
