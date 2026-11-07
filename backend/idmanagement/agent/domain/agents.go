package domain

import (
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// ===============================================================
// Agent 集約
// ===============================================================

// Agent は tenant-scoped な非人間 (non-human) identity principal。自身の資格情報は
// 持たず、AgentCredentialBinding で既存 OAuth2Client に束縛してトークンを得る。
// owner_sub (所有者 User の sub) は必須。Status は Active / Disabled / Killed の
// 三状態で、Killed は一方向終端 (緊急停止)。Active 以外は新規トークンを発行しない
// (fail-closed)。
type Agent struct {
	ID          string                `json:"id"`
	TenantID    string                `json:"tenant_id"`
	Name        string                `json:"name"`
	Description *string               `json:"description,omitempty"`
	Kind        idmdomain.AgentKind   `json:"kind"`
	OwnerUserID string                `json:"owner_user_id"`
	Status      idmdomain.AgentStatus `json:"status"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	DisabledAt  *time.Time            `json:"disabled_at,omitempty"`
	KilledAt    *time.Time            `json:"killed_at,omitempty"`
}

func (a Agent) Validate() error {
	return spec.All(
		spec.Required("ID", a.ID, spec.Chars(1, spec.LengthHandle)),
		spec.Required("TenantID", a.TenantID),
		spec.Required("Name", a.Name, spec.Chars(1, spec.LengthName)),
		spec.OptionalPtr("Description", a.Description, spec.CharsAtMost(spec.LengthDescription)),
		spec.Required("Kind", a.Kind, spec.Must(idmdomain.AgentKind.Valid, "agent kind is not in enum")),
		spec.Required("OwnerUserID", a.OwnerUserID),
		spec.Required("Status", a.Status, spec.Must(idmdomain.AgentStatus.Valid, "agent status is not in enum")),
		spec.Required("CreatedAt", a.CreatedAt),
		spec.Required("UpdatedAt", a.UpdatedAt),
	)
}

// IsActive は Agent が新規トークン発行可能な状態かを返す。Status が Active
// かつ disabled_at / killed_at がいずれも未設定の場合のみ true。
func (a Agent) IsActive() bool {
	return a.Status == idmdomain.AgentStatusActive && a.DisabledAt == nil && a.KilledAt == nil
}

// AgentCredentialBinding は Agent と OAuth2Client の束縛関係。
// agent_id × client_id で一意。
type AgentCredentialBinding struct {
	AgentID   string    `json:"agent_id"`
	ClientID  string    `json:"client_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (b AgentCredentialBinding) Validate() error {
	return spec.All(
		spec.Required("AgentID", b.AgentID),
		spec.Required("ClientID", b.ClientID),
		spec.Required("CreatedAt", b.CreatedAt),
	)
}

// NewAgentID は不変の Agent 識別子 (UUID v4) を生成する。
func NewAgentID() (string, error) {
	return spec.NewUUIDv4()
}
