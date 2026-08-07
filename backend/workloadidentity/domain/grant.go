package domain

// WorkloadIdentityGrant は検証を通過した外部 attestation を写した idmagic principal
// (ADR-053)。OAuth2 の token-exchange usecase はこれを subject として扱い、ClientID が
// 指す束縛済み OAuth2Client を自身の repository で解決して scope / grant_types の
// ダウンスコープ上限を得る (WorkloadIdentity は OAuth2Client の詳細を持たない)。
type WorkloadIdentityGrant struct {
	AgentID       string
	ClientID      string
	TrustBundleID string
	BindingID     string
}
