package domain

import "time"

type FederationLinked struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
	Provider string    `json:"provider"`
}

func (e *FederationLinked) EventType() string     { return "FederationLinked" }
func (e *FederationLinked) OccurredAt() time.Time { return e.At }

type FederationUnlinked struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
	Provider string    `json:"provider"`
}

func (e *FederationUnlinked) EventType() string     { return "FederationUnlinked" }
func (e *FederationUnlinked) OccurredAt() time.Time { return e.At }

type EmailChangeRequested struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	UserID       string    `json:"userId"`
	NewEmailHash string    `json:"newEmailHash"`
}

func (e *EmailChangeRequested) EventType() string     { return "EmailChangeRequested" }
func (e *EmailChangeRequested) OccurredAt() time.Time { return e.At }

type EmailChanged struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
}

func (e *EmailChanged) EventType() string     { return "EmailChanged" }
func (e *EmailChanged) OccurredAt() time.Time { return e.At }

type UserCreated struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	ActorUserID  string    `json:"actorUserId"`
	TargetUserID string    `json:"targetUserId"`
}

func (e *UserCreated) EventType() string     { return "UserCreated" }
func (e *UserCreated) OccurredAt() time.Time { return e.At }

type UserUpdated struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	ActorUserID   string    `json:"actorUserId"`
	TargetUserID  string    `json:"targetUserId"`
	ChangedFields []string  `json:"changedFields"`
}

func (e *UserUpdated) EventType() string     { return "UserUpdated" }
func (e *UserUpdated) OccurredAt() time.Time { return e.At }

type UserDisabled struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	ActorUserID  string    `json:"actorUserId"`
	TargetUserID string    `json:"targetUserId"`
}

func (e *UserDisabled) EventType() string     { return "UserDisabled" }
func (e *UserDisabled) OccurredAt() time.Time { return e.At }

type UserEnabled struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	ActorUserID  string    `json:"actorUserId"`
	TargetUserID string    `json:"targetUserId"`
}

func (e *UserEnabled) EventType() string     { return "UserEnabled" }
func (e *UserEnabled) OccurredAt() time.Time { return e.At }

type UserRequiredActionSet struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	ActorUserID  string    `json:"actorUserId"`
	TargetUserID string    `json:"targetUserId"`
	Action       string    `json:"action"`
}

func (e *UserRequiredActionSet) EventType() string     { return "UserRequiredActionSet" }
func (e *UserRequiredActionSet) OccurredAt() time.Time { return e.At }

type UserRequiredActionCleared struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	ActorUserID  string    `json:"actorUserId"`
	TargetUserID string    `json:"targetUserId"`
	Action       string    `json:"action"`
}

func (e *UserRequiredActionCleared) EventType() string     { return "UserRequiredActionCleared" }
func (e *UserRequiredActionCleared) OccurredAt() time.Time { return e.At }

type UserSoftDeleted struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	ActorUserID  string    `json:"actorUserId"`
	TargetUserID string    `json:"targetUserId"`
	Reason       string    `json:"reason,omitempty"`
}

func (e *UserSoftDeleted) EventType() string     { return "UserSoftDeleted" }
func (e *UserSoftDeleted) OccurredAt() time.Time { return e.At }

type UserRestored struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	ActorUserID  string    `json:"actorUserId"`
	TargetUserID string    `json:"targetUserId"`
}

func (e *UserRestored) EventType() string     { return "UserRestored" }
func (e *UserRestored) OccurredAt() time.Time { return e.At }

type UserDeleted struct {
	At           time.Time `json:"-"`
	TenantID     string    `json:"tenantId"`
	ActorUserID  string    `json:"actorUserId"`
	TargetUserID string    `json:"targetUserId"`
	Reason       string    `json:"reason,omitempty"`
}

func (e *UserDeleted) EventType() string     { return "UserDeleted" }
func (e *UserDeleted) OccurredAt() time.Time { return e.At }

type AgentRegistered struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	AgentID     string    `json:"agentId"`
}

func (e *AgentRegistered) EventType() string     { return "AgentRegistered" }
func (e *AgentRegistered) OccurredAt() time.Time { return e.At }

type AgentUpdated struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	ActorUserID   string    `json:"actorUserId"`
	AgentID       string    `json:"agentId"`
	ChangedFields []string  `json:"changedFields"`
}

func (e *AgentUpdated) EventType() string     { return "AgentUpdated" }
func (e *AgentUpdated) OccurredAt() time.Time { return e.At }

type AgentDisabled struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	AgentID     string    `json:"agentId"`
}

func (e *AgentDisabled) EventType() string     { return "AgentDisabled" }
func (e *AgentDisabled) OccurredAt() time.Time { return e.At }

type AgentEnabled struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	AgentID     string    `json:"agentId"`
}

func (e *AgentEnabled) EventType() string     { return "AgentEnabled" }
func (e *AgentEnabled) OccurredAt() time.Time { return e.At }

type AgentKilled struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	AgentID     string    `json:"agentId"`
}

func (e *AgentKilled) EventType() string     { return "AgentKilled" }
func (e *AgentKilled) OccurredAt() time.Time { return e.At }

type AgentDeleted struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	AgentID     string    `json:"agentId"`
}

func (e *AgentDeleted) EventType() string     { return "AgentDeleted" }
func (e *AgentDeleted) OccurredAt() time.Time { return e.At }

type AgentOwnerChanged struct {
	At                  time.Time `json:"-"`
	TenantID            string    `json:"tenantId"`
	ActorUserID         string    `json:"actorUserId"`
	AgentID             string    `json:"agentId"`
	PreviousOwnerUserID string    `json:"previousOwnerUserId"`
	NewOwnerUserID      string    `json:"newOwnerUserId"`
}

func (e *AgentOwnerChanged) EventType() string     { return "AgentOwnerChanged" }
func (e *AgentOwnerChanged) OccurredAt() time.Time { return e.At }

type AgentCredentialBound struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	AgentID     string    `json:"agentId"`
	ClientID    string    `json:"clientId"`
}

func (e *AgentCredentialBound) EventType() string     { return "AgentCredentialBound" }
func (e *AgentCredentialBound) OccurredAt() time.Time { return e.At }

type AgentCredentialUnbound struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	AgentID     string    `json:"agentId"`
	ClientID    string    `json:"clientId"`
}

func (e *AgentCredentialUnbound) EventType() string     { return "AgentCredentialUnbound" }
func (e *AgentCredentialUnbound) OccurredAt() time.Time { return e.At }

type GroupCreated struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	GroupID     string    `json:"groupId"`
}

func (e *GroupCreated) EventType() string     { return "GroupCreated" }
func (e *GroupCreated) OccurredAt() time.Time { return e.At }

type GroupUpdated struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	ActorUserID   string    `json:"actorUserId"`
	GroupID       string    `json:"groupId"`
	ChangedFields []string  `json:"changedFields"`
}

func (e *GroupUpdated) EventType() string     { return "GroupUpdated" }
func (e *GroupUpdated) OccurredAt() time.Time { return e.At }

type GroupDeleted struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	GroupID     string    `json:"groupId"`
}

func (e *GroupDeleted) EventType() string     { return "GroupDeleted" }
func (e *GroupDeleted) OccurredAt() time.Time { return e.At }

type GroupMemberAdded struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	GroupID     string    `json:"groupId"`
	UserID      string    `json:"userId"`
}

func (e *GroupMemberAdded) EventType() string     { return "GroupMemberAdded" }
func (e *GroupMemberAdded) OccurredAt() time.Time { return e.At }

type GroupMemberRemoved struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	GroupID     string    `json:"groupId"`
	UserID      string    `json:"userId"`
}

func (e *GroupMemberRemoved) EventType() string     { return "GroupMemberRemoved" }
func (e *GroupMemberRemoved) OccurredAt() time.Time { return e.At }

type DynamicGroupRuleUpdated struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	GroupID     string    `json:"groupId"`
	RuleVersion int64     `json:"ruleVersion"`
}

func (e *DynamicGroupRuleUpdated) EventType() string     { return "DynamicGroupRuleUpdated" }
func (e *DynamicGroupRuleUpdated) OccurredAt() time.Time { return e.At }

type DynamicGroupRuleEnabled struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	GroupID     string    `json:"groupId"`
	RuleVersion int64     `json:"ruleVersion"`
}

func (e *DynamicGroupRuleEnabled) EventType() string     { return "DynamicGroupRuleEnabled" }
func (e *DynamicGroupRuleEnabled) OccurredAt() time.Time { return e.At }

type DynamicGroupRuleDisabled struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	GroupID     string    `json:"groupId"`
	RuleVersion int64     `json:"ruleVersion"`
}

func (e *DynamicGroupRuleDisabled) EventType() string     { return "DynamicGroupRuleDisabled" }
func (e *DynamicGroupRuleDisabled) OccurredAt() time.Time { return e.At }

type DynamicMembershipEvaluated struct {
	At             time.Time `json:"-"`
	TenantID       string    `json:"tenantId"`
	GroupID        string    `json:"groupId"`
	RuleVersion    int64     `json:"ruleVersion"`
	AddedCount     int       `json:"addedCount"`
	RemovedCount   int       `json:"removedCount"`
	UnchangedCount int       `json:"unchangedCount"`
	ErrorCount     int       `json:"errorCount"`
}

func (e *DynamicMembershipEvaluated) EventType() string     { return "DynamicMembershipEvaluated" }
func (e *DynamicMembershipEvaluated) OccurredAt() time.Time { return e.At }

// DataExport* events audit the admin CSV export lifecycle
// (spec/contexts/identity-management.yaml events). Payloads carry no
// PII values — only counts, target kind, ids, and stable codes — and every
// event carries tenantId so the shared audit projection (bootstrap.NewEmitFunc)
// keys it to the tenant.

type DataExportRequested struct {
	At               time.Time `json:"-"`
	TenantID         string    `json:"tenantId"`
	ActorUserID      string    `json:"actorUserId"`
	ExportID         string    `json:"exportId"`
	Target           string    `json:"target"`
	RequestedColumns []string  `json:"requestedColumns"`
}

func (e *DataExportRequested) EventType() string     { return "DataExportRequested" }
func (e *DataExportRequested) OccurredAt() time.Time { return e.At }

type DataExportStarted struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ExportID string    `json:"exportId"`
	Target   string    `json:"target"`
}

func (e *DataExportStarted) EventType() string     { return "DataExportStarted" }
func (e *DataExportStarted) OccurredAt() time.Time { return e.At }

type DataExportSucceeded struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	ExportID  string    `json:"exportId"`
	Target    string    `json:"target"`
	TotalRows int       `json:"totalRows"`
	ByteSize  int       `json:"byteSize"`
}

func (e *DataExportSucceeded) EventType() string     { return "DataExportSucceeded" }
func (e *DataExportSucceeded) OccurredAt() time.Time { return e.At }

type DataExportFailed struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	ExportID  string    `json:"exportId"`
	Target    string    `json:"target"`
	ErrorCode string    `json:"errorCode"`
}

func (e *DataExportFailed) EventType() string     { return "DataExportFailed" }
func (e *DataExportFailed) OccurredAt() time.Time { return e.At }

type DataExportCanceled struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	ExportID    string    `json:"exportId"`
}

func (e *DataExportCanceled) EventType() string     { return "DataExportCanceled" }
func (e *DataExportCanceled) OccurredAt() time.Time { return e.At }

type DataExportDownloaded struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	ExportID    string    `json:"exportId"`
	Target      string    `json:"target"`
}

func (e *DataExportDownloaded) EventType() string     { return "DataExportDownloaded" }
func (e *DataExportDownloaded) OccurredAt() time.Time { return e.At }
