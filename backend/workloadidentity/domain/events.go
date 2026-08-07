package domain

import "time"

type WorkloadTrustBundleConfigured struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	TrustBundleID string    `json:"trustBundleId"`
	Issuer        string    `json:"issuer"`
}

func (e *WorkloadTrustBundleConfigured) EventType() string     { return "WorkloadTrustBundleConfigured" }
func (e *WorkloadTrustBundleConfigured) OccurredAt() time.Time { return e.At }

type WorkloadTrustBundleUpdated struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	TrustBundleID string    `json:"trustBundleId"`
}

func (e *WorkloadTrustBundleUpdated) EventType() string     { return "WorkloadTrustBundleUpdated" }
func (e *WorkloadTrustBundleUpdated) OccurredAt() time.Time { return e.At }

type WorkloadTrustBundleDisabled struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	TrustBundleID string    `json:"trustBundleId"`
}

func (e *WorkloadTrustBundleDisabled) EventType() string     { return "WorkloadTrustBundleDisabled" }
func (e *WorkloadTrustBundleDisabled) OccurredAt() time.Time { return e.At }

type WorkloadTrustBundleEnabled struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	TrustBundleID string    `json:"trustBundleId"`
}

func (e *WorkloadTrustBundleEnabled) EventType() string     { return "WorkloadTrustBundleEnabled" }
func (e *WorkloadTrustBundleEnabled) OccurredAt() time.Time { return e.At }

type WorkloadTrustBundleDeleted struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	TrustBundleID string    `json:"trustBundleId"`
}

func (e *WorkloadTrustBundleDeleted) EventType() string     { return "WorkloadTrustBundleDeleted" }
func (e *WorkloadTrustBundleDeleted) OccurredAt() time.Time { return e.At }

type WorkloadTrustBundleJWKSRefreshed struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	TrustBundleID string    `json:"trustBundleId"`
	Reachable     bool      `json:"reachable"`
}

func (e *WorkloadTrustBundleJWKSRefreshed) EventType() string {
	return "WorkloadTrustBundleJWKSRefreshed"
}
func (e *WorkloadTrustBundleJWKSRefreshed) OccurredAt() time.Time { return e.At }

type AgentWorkloadBindingCreated struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	TrustBundleID string    `json:"trustBundleId"`
	BindingID     string    `json:"bindingId"`
	AgentID       string    `json:"agentId"`
}

func (e *AgentWorkloadBindingCreated) EventType() string     { return "AgentWorkloadBindingCreated" }
func (e *AgentWorkloadBindingCreated) OccurredAt() time.Time { return e.At }

type AgentWorkloadBindingDisabled struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	BindingID string    `json:"bindingId"`
}

func (e *AgentWorkloadBindingDisabled) EventType() string     { return "AgentWorkloadBindingDisabled" }
func (e *AgentWorkloadBindingDisabled) OccurredAt() time.Time { return e.At }

type AgentWorkloadBindingEnabled struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	BindingID string    `json:"bindingId"`
}

func (e *AgentWorkloadBindingEnabled) EventType() string     { return "AgentWorkloadBindingEnabled" }
func (e *AgentWorkloadBindingEnabled) OccurredAt() time.Time { return e.At }

type AgentWorkloadBindingDeleted struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	BindingID string    `json:"bindingId"`
}

func (e *AgentWorkloadBindingDeleted) EventType() string     { return "AgentWorkloadBindingDeleted" }
func (e *AgentWorkloadBindingDeleted) OccurredAt() time.Time { return e.At }

// WorkloadTokenExchanged は外部 attestation の検証を通過し、束縛先 Agent の資格情報として
// idmagic token を発行したことを表す (ADR-053)。OAuth2 の token-exchange usecase が
// 実際の発行成功後に emit する (検証成功だけでは emit しない)。
type WorkloadTokenExchanged struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	TrustBundleID string    `json:"trustBundleId"`
	BindingID     string    `json:"bindingId"`
	AgentID       string    `json:"agentId"`
	Audience      string    `json:"audience"`
}

func (e *WorkloadTokenExchanged) EventType() string     { return "WorkloadTokenExchanged" }
func (e *WorkloadTokenExchanged) OccurredAt() time.Time { return e.At }

// WorkloadAttestationRejected は外部 attestation token の検証が fail-closed で拒否
// されたことを表す (ADR-053)。TrustBundleID は issuer が登録済みだった場合のみ設定する。
type WorkloadAttestationRejected struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	Reason        string    `json:"reason"`
	TrustBundleID string    `json:"trustBundleId,omitempty"`
}

func (e *WorkloadAttestationRejected) EventType() string     { return "WorkloadAttestationRejected" }
func (e *WorkloadAttestationRejected) OccurredAt() time.Time { return e.At }
