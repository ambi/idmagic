// Package domain は WorkloadIdentity bounded context の業務型を所有する。
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
