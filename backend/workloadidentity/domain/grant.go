package domain

import "time"

// WorkloadIdentityGrant は検証を通過した外部 attestation を写した idmagic principal。
// OAuth2 の token-exchange usecase はこれを subject として扱い、ClientID が
// 指す束縛済み OAuth2Client を自身の repository で解決して scope / grant_types の
// ダウンスコープ上限を得る (WorkloadIdentity は OAuth2Client の詳細を持たない)。
type WorkloadIdentityGrant struct {
	AgentID       string
	ClientID      string
	TrustBundleID string
	BindingID     string
}

// WorkloadClientAuthentication は、クライアントアサーションとして提示された JWT-SVID の
// 検証を通過し、要求のクライアントが対応先の Agent に束縛されていることを確かめた結果
// (REQ-WORKLOADIDENTITY-017)。OAuth2 はこれで `spiffe_jwt` のクライアントを認証し、
// JTI があれば有効期限まで再利用を拒否する。
type WorkloadClientAuthentication struct {
	AgentID       string
	ClientID      string
	TrustBundleID string
	BindingID     string
	SpiffeID      string
	// JTI は JWT-SVID の `jti` で、なければ空である。
	JTI       string
	ExpiresAt time.Time
}
