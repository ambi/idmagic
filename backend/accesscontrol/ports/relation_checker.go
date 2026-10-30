package ports

import "context"

// PrincipalRef は判定の主体または行為者の 1 段である。Type は認可モデルの主体型を指す。
type PrincipalRef struct {
	Type string
	ID   string
}

// RelationCheck は検証済みの主体と行為者についての 1 回の判定である。主体と行為者は
// 呼び出し側がトークンから導いたものだけを渡す。
type RelationCheck struct {
	TenantID       string
	ResourceType   string
	ResourceID     string
	Relation       string
	Subject        PrincipalRef
	Actors         []PrincipalRef
	GrantedScopes  []string
	RequiredScopes []string
}

// RelationDecision は判定の結果である。Reasons は識別子を含まない。
type RelationDecision struct {
	Permitted bool
	Reasons   []string
}

// RelationChecker は、管理 API を通らずに関係の判定を求める入口である。error は判定不能を
// 表し、呼び出し側はこれを許可へ退避してはならない。
type RelationChecker interface {
	CheckRelation(ctx context.Context, check RelationCheck) (RelationDecision, error)
}
