package ports

import "slices"

// 監査イベントの汎用検索属性 registry (wi-145)。SCL models の
// AuditEventSearchAttribute / AuditEventFilterExpression / AuditEventFilterOperator /
// AuditEventSearchTransform の双子定義。
//
// この registry が field / operator / transform の allowlist の単一の正であり、
// 任意 SQL / JSONPath ではなく閉じた検索文法を構造的に強制する。registry に無い field は
// filter parse の時点で拒否され、SQL 生成には到達しない。

// AuditSearchTransform は検索属性を sidecar に保存 / 検索する前に平文値へ適用する変換。
type AuditSearchTransform int

const (
	// TransformNone は平文をそのまま保存する (非 PII)。
	TransformNone AuditSearchTransform = iota
	// TransformHash は tenant salt 付き SHA-256 で hash する (ADR-046)。
	TransformHash
	// TransformIPTruncate は IPv4 /24・IPv6 /48 へ丸める (ADR-046)。
	TransformIPTruncate
)

// AuditFilterOperator は filter 式で使える比較演算子。SCL AuditEventFilterOperator の双子。
type AuditFilterOperator string

const (
	// OpEq は完全一致 (値 1 個)。
	OpEq AuditFilterOperator = "eq"
	// OpIn は値集合のいずれか一致 (値 N 個)。
	OpIn AuditFilterOperator = "in"
	// OpContains は raw text の部分一致 (q 用、値 1 個)。
	OpContains AuditFilterOperator = "contains"
	// OpTimeRange は occurred_at の期間指定 (値 2 個: after, before)。
	OpTimeRange AuditFilterOperator = "time_range"
)

// AuditSearchAttribute は検索で許可される 1 検索軸 (registry エントリ)。
// SCL AuditEventSearchAttribute の双子。Field を sidecar の attr_name として使う。
type AuditSearchAttribute struct {
	// Field は検索軸の正準ドット名。sidecar の attr_name でもある。例: actor.id / client.ip。
	Field string
	// RawStorable は平文をそのまま sidecar に保存してよいか。PII 属性は false。
	RawStorable bool
	// Transform は保存 / 検索前に平文値へ適用する変換。
	Transform AuditSearchTransform
	// TenantSaltReq は Transform に tenant salt を要するか (hash 属性で true)。
	TenantSaltReq bool
	// AllowedOperators はこの属性に許可される operator の allowlist。
	AllowedOperators []AuditFilterOperator
	// UIVisible は admin 検索ビルダーの UI プリセットに出すか。
	UIVisible bool
}

// AllowsOperator は op がこの属性で許可されているかを返す。
func (a AuditSearchAttribute) AllowsOperator(op AuditFilterOperator) bool {
	return slices.Contains(a.AllowedOperators, op)
}

// AuditFilterExpression は AuditEventQuery.filter の 1 項。registry allowlist の field と
// operator、値の並びからなる連言 (AND) の 1 要素。SCL AuditEventFilterExpression の双子。
// PII 属性の Values は平文入力をサーバ側で transform してから照合する。
type AuditFilterExpression struct {
	Field    string
	Operator AuditFilterOperator
	Values   []string
}

// AuditSearchRegistry は許可される検索属性の allowlist。
//
// 非 PII raw id 属性と、emit payload 上で transform 済みの PII-safe 属性を sidecar に載せる。
var AuditSearchRegistry = map[string]AuditSearchAttribute{
	"event.type": {
		Field:            "event.type",
		RawStorable:      true,
		Transform:        TransformNone,
		AllowedOperators: []AuditFilterOperator{OpEq, OpIn, OpContains},
		UIVisible:        true,
	},
	"outcome": {
		Field:            "outcome",
		RawStorable:      true,
		Transform:        TransformNone,
		AllowedOperators: []AuditFilterOperator{OpEq, OpIn},
		UIVisible:        true,
	},
	"actor.id": {
		Field:            "actor.id",
		RawStorable:      true,
		Transform:        TransformNone,
		AllowedOperators: []AuditFilterOperator{OpEq, OpIn},
		UIVisible:        true,
	},
	"actor.username": {
		Field:            "actor.username",
		RawStorable:      false,
		Transform:        TransformHash,
		TenantSaltReq:    true,
		AllowedOperators: []AuditFilterOperator{OpEq, OpIn},
		UIVisible:        true,
	},
	"target.id": {
		Field:            "target.id",
		RawStorable:      true,
		Transform:        TransformNone,
		AllowedOperators: []AuditFilterOperator{OpEq, OpIn},
		UIVisible:        true,
	},
	"client.id": {
		Field:            "client.id",
		RawStorable:      true,
		Transform:        TransformNone,
		AllowedOperators: []AuditFilterOperator{OpEq, OpIn},
		UIVisible:        true,
	},
	"client.ip": {
		Field:            "client.ip",
		RawStorable:      false,
		Transform:        TransformIPTruncate,
		AllowedOperators: []AuditFilterOperator{OpEq, OpIn},
		UIVisible:        true,
	},
	"session.id": {
		Field:            "session.id",
		RawStorable:      true,
		Transform:        TransformNone,
		AllowedOperators: []AuditFilterOperator{OpEq, OpIn},
		UIVisible:        true,
	},
	"transaction.id": {
		Field:            "transaction.id",
		RawStorable:      true,
		Transform:        TransformNone,
		AllowedOperators: []AuditFilterOperator{OpEq, OpIn},
		UIVisible:        true,
	},
	"correlation.id": {
		Field:            "correlation.id",
		RawStorable:      true,
		Transform:        TransformNone,
		AllowedOperators: []AuditFilterOperator{OpEq, OpIn},
		UIVisible:        true,
	},
	"request.id": {
		Field:            "request.id",
		RawStorable:      true,
		Transform:        TransformNone,
		AllowedOperators: []AuditFilterOperator{OpEq, OpIn},
		UIVisible:        true,
	},
}

// LookupSearchAttribute は field 名で registry を引く。第 2 戻り値は存在有無。
func LookupSearchAttribute(field string) (AuditSearchAttribute, bool) {
	attr, ok := AuditSearchRegistry[field]
	return attr, ok
}
