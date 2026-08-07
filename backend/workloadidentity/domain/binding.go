package domain

import (
	"path"
	"time"

	z "github.com/Oudwins/zog"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// AgentWorkloadBinding は WorkloadTrustBundle 配下で外部 subject の glob pattern を
// 同一テナントの Agent へ写す mapping (ADR-053)。agent_id は WorkloadTrustBundle と
// 同一 tenant_id の既存 Agent でなければならない (WorkloadIdentityReferencesStayTenantLocal、
// usecase 側で検証する)。
type AgentWorkloadBinding struct {
	ID             string
	TenantID       string
	TrustBundleID  string
	SubjectPattern string
	AgentID        string
	Status         AgentWorkloadBindingStatus
	CreatedAt      time.Time
	UpdatedAt      *time.Time
	DisabledAt     *time.Time
}

var agentWorkloadBindingSchema = z.Struct(z.Shape{
	"ID":             z.String().Min(1).Max(64).Required(),
	"TenantID":       z.String().Min(1).Required(),
	"TrustBundleID":  z.String().Min(1).Required(),
	"SubjectPattern": z.String().Min(1).Max(500).Required(),
	"AgentID":        z.String().Min(1).Required(),
	"Status": z.StringLike[AgentWorkloadBindingStatus]().TestFunc(
		func(value *AgentWorkloadBindingStatus, _ z.Ctx) bool { return value.Valid() },
		z.Message("agent workload binding status is not in enum"),
	).Required(),
	"CreatedAt": z.Time().Required(),
})

// Validate は構造的妥当性 (spec/contexts/workloadidentity.yaml AgentWorkloadBinding
// constraints) を検証する。subject_pattern は path.Match が受理する glob 構文でなければ
// ならない (不正な pattern は登録時点で拒否し、交換時の判定漏れを防ぐ)。
func (b AgentWorkloadBinding) Validate() error {
	if err := spec.Validate(agentWorkloadBindingSchema, &b); err != nil {
		return err
	}
	if _, err := path.Match(b.SubjectPattern, ""); err != nil {
		return errMalformedSubjectPattern
	}
	return nil
}

// IsEnabled は AgentWorkloadBinding が交換に使える状態かを返す (AgentWorkloadBindingLifecycle)。
func (b AgentWorkloadBinding) IsEnabled() bool {
	return b.Status == AgentWorkloadBindingStatusEnabled
}

// MatchAgent は Enabled な binding のうち subject に一致する pattern を持つものを
// 一意に決定する (ADR-053 fail-closed)。一致が無ければ ErrNoBindingMatch、複数の
// Enabled binding に曖昧に一致すれば ErrAmbiguousBindingMatch を返す。Disabled な
// binding は候補から除外し、ambiguity の判定にも数えない。
func MatchAgent(bindings []AgentWorkloadBinding, subject string) (*AgentWorkloadBinding, error) {
	var matches []AgentWorkloadBinding
	for _, b := range bindings {
		if !b.IsEnabled() {
			continue
		}
		ok, err := path.Match(b.SubjectPattern, subject)
		if err != nil || !ok {
			continue
		}
		matches = append(matches, b)
	}
	switch len(matches) {
	case 0:
		return nil, ErrNoBindingMatch
	case 1:
		return &matches[0], nil
	default:
		return nil, ErrAmbiguousBindingMatch
	}
}

// NewAgentWorkloadBindingID は不変の AgentWorkloadBinding 識別子 (UUID v4) を生成する。
func NewAgentWorkloadBindingID() (string, error) {
	return spec.NewUUIDv4()
}
