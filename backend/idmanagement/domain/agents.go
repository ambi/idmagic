package domain

import (
	"time"

	z "github.com/Oudwins/zog"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// ===============================================================
// Agent 集約 (ADR-048)
// ===============================================================

// Agent は tenant-scoped な非人間 (non-human) identity principal。自身の資格情報は
// 持たず、AgentCredentialBinding で既存 OAuth2Client に束縛してトークンを得る。
// owner_sub (所有者 User の sub) は必須。Status は Active / Disabled / Killed の
// 三状態で、Killed は一方向終端 (緊急停止)。Active 以外は新規トークンを発行しない
// (fail-closed)。
type Agent struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	Name        string      `json:"name"`
	Description *string     `json:"description,omitempty"`
	Kind        AgentKind   `json:"kind"`
	OwnerUserID string      `json:"owner_user_id"`
	Status      AgentStatus `json:"status"`
	Roles       []string    `json:"roles"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	DisabledAt  *time.Time  `json:"disabled_at,omitempty"`
	KilledAt    *time.Time  `json:"killed_at,omitempty"`
}

var agentSchema = z.Struct(z.Shape{
	"ID":          z.String().Min(1).Max(64).Required(),
	"TenantID":    z.String().Min(1).Required(),
	"Name":        z.String().Min(1).Max(100).Required(),
	"Description": z.Ptr(z.String().Max(500)),
	"Kind": z.StringLike[AgentKind]().TestFunc(
		func(value *AgentKind, _ z.Ctx) bool { return value.Valid() },
		z.Message("agent kind is not in enum"),
	).Required(),
	"OwnerUserID": z.String().Min(1).Required(),
	"Status": z.StringLike[AgentStatus]().TestFunc(
		func(value *AgentStatus, _ z.Ctx) bool { return value.Valid() },
		z.Message("agent status is not in enum"),
	).Required(),
	"Roles":     z.Slice(z.String().Min(1)),
	"CreatedAt": z.Time().Required(),
	"UpdatedAt": z.Time().Required(),
})

func (a Agent) Validate() error {
	return spec.Validate(agentSchema, &a)
}

// IsActive は Agent が新規トークン発行可能な状態かを返す (ADR-048)。Status が Active
// かつ disabled_at / killed_at がいずれも未設定の場合のみ true。
func (a Agent) IsActive() bool {
	return a.Status == AgentStatusActive && a.DisabledAt == nil && a.KilledAt == nil
}

// AgentCredentialBinding は Agent と OAuth2Client の束縛関係 (ADR-048)。
// agent_id × client_id で一意。
type AgentCredentialBinding struct {
	AgentID   string    `json:"agent_id"`
	ClientID  string    `json:"client_id"`
	CreatedAt time.Time `json:"created_at"`
}

var agentCredentialBindingSchema = z.Struct(z.Shape{
	"AgentID":   z.String().Min(1).Required(),
	"ClientID":  z.String().Min(1).Required(),
	"CreatedAt": z.Time().Required(),
})

func (b AgentCredentialBinding) Validate() error {
	return spec.Validate(agentCredentialBindingSchema, &b)
}

// NewAgentID は不変の Agent 識別子 (UUID v4) を生成する。
func NewAgentID() (string, error) {
	return spec.NewUUIDv4()
}
