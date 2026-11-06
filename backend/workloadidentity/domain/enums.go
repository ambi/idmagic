// Package domain は WorkloadIdentity モジュールの業務型を所有する。
package domain

// WorkloadTrustBundleStatus は WorkloadTrustBundle の運用状態。Enabled は配下の
// binding が交換に使える通常稼働、Disabled は可逆な運用停止で以後の交換を
// fail-closed で拒否する。
type WorkloadTrustBundleStatus string

const (
	WorkloadTrustBundleStatusEnabled  WorkloadTrustBundleStatus = "enabled"
	WorkloadTrustBundleStatusDisabled WorkloadTrustBundleStatus = "disabled"
)

func (s WorkloadTrustBundleStatus) Valid() bool {
	switch s {
	case WorkloadTrustBundleStatusEnabled, WorkloadTrustBundleStatusDisabled:
		return true
	}
	return false
}

// AgentWorkloadBindingStatus は AgentWorkloadBinding の運用状態。
type AgentWorkloadBindingStatus string

const (
	AgentWorkloadBindingStatusEnabled  AgentWorkloadBindingStatus = "enabled"
	AgentWorkloadBindingStatusDisabled AgentWorkloadBindingStatus = "disabled"
)

func (s AgentWorkloadBindingStatus) Valid() bool {
	switch s {
	case AgentWorkloadBindingStatusEnabled, AgentWorkloadBindingStatusDisabled:
		return true
	}
	return false
}

// WorkloadTrustBundleProfile は WorkloadTrustBundle が掛ける検証の種別。OIDC は
// OIDC 互換の JWT として検証し、SPIFFE は SPIFFE の JWT-SVID の規則を足す
// (REQ-WORKLOADIDENTITY-011)。登録の後は変えない。
type WorkloadTrustBundleProfile string

const (
	WorkloadTrustBundleProfileOIDC   WorkloadTrustBundleProfile = "oidc"
	WorkloadTrustBundleProfileSPIFFE WorkloadTrustBundleProfile = "spiffe"
)

func (p WorkloadTrustBundleProfile) Valid() bool {
	switch p {
	case WorkloadTrustBundleProfileOIDC, WorkloadTrustBundleProfileSPIFFE:
		return true
	}
	return false
}
