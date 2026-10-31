package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"time"
)

// AgentGuardrailViolationAction は超過の扱いである (REQ-OAUTH2-074)。
type AgentGuardrailViolationAction string

const (
	AgentGuardrailDeny            AgentGuardrailViolationAction = "deny"
	AgentGuardrailRequireApproval AgentGuardrailViolationAction = "require_approval"
)

// 評価の理由。監査にだけ残し、応答へ返さない。
const (
	AgentGuardrailReasonResource        = "allowed_resources"
	AgentGuardrailReasonDetailType      = "allowed_authorization_detail_types"
	AgentGuardrailReasonIssuanceRate    = "issuance_rate"
	AgentGuardrailReasonExpired         = "expires_at"
	AgentGuardrailReasonDelegationDepth = "max_delegation_depth"
	AgentGuardrailReasonApproval        = "require_approval"
	AgentGuardrailReasonUnavailable     = "guardrail_unavailable"
)

type AgentGuardrailIssuanceRate struct {
	MaxIssuances  int `json:"max_issuances"`
	WindowSeconds int `json:"window_seconds"`
}

type AgentGuardrailOnViolation struct {
	AllowedResources                AgentGuardrailViolationAction `json:"allowed_resources,omitempty"`
	AllowedAuthorizationDetailTypes AgentGuardrailViolationAction `json:"allowed_authorization_detail_types,omitempty"`
	IssuanceRate                    AgentGuardrailViolationAction `json:"issuance_rate,omitempty"`
	ExpiresAt                       AgentGuardrailViolationAction `json:"expires_at,omitempty"`
}

// AgentGuardrailLimits は宣言の内容である。nil は制限せず、空の配列は何も許さない。
type AgentGuardrailLimits struct {
	AllowedResources                []string                    `json:"allowed_resources,omitzero"`
	AllowedAuthorizationDetailTypes []string                    `json:"allowed_authorization_detail_types,omitzero"`
	IssuanceRate                    *AgentGuardrailIssuanceRate `json:"issuance_rate,omitempty"`
	MaxTokenTTLSeconds              *int                        `json:"max_token_ttl_seconds,omitempty"`
	ExpiresAt                       *time.Time                  `json:"expires_at,omitempty"`
	MaxDelegationDepth              *int                        `json:"max_delegation_depth,omitempty"`
	RequireApproval                 bool                        `json:"require_approval,omitempty"`
	OnViolation                     AgentGuardrailOnViolation   `json:"on_violation,omitzero"`
}

// AgentGuardrailPolicy は一つの層である。AgentID が空ならテナントのデフォルトを表す。
type AgentGuardrailPolicy struct {
	TenantID string
	AgentID  string
	AgentGuardrailLimits
	UpdatedAt time.Time
}

const (
	agentGuardrailMaxItems         = 100
	agentGuardrailMaxWindowSeconds = 86400
	agentGuardrailMaxIssuances     = 1000000
	agentGuardrailMaxDepth         = 16
)

// ErrInvalidAgentGuardrail は宣言が検証を通らないことを表す (REQ-OAUTH2-077)。
var ErrInvalidAgentGuardrail = errors.New("invalid agent guardrail")

func (l AgentGuardrailLimits) Validate() error {
	if len(l.AllowedResources) > agentGuardrailMaxItems || len(l.AllowedAuthorizationDetailTypes) > agentGuardrailMaxItems {
		return ErrInvalidAgentGuardrail
	}
	if slices.Contains(l.AllowedResources, "") || slices.Contains(l.AllowedAuthorizationDetailTypes, "") {
		return ErrInvalidAgentGuardrail
	}
	if r := l.IssuanceRate; r != nil && (r.MaxIssuances < 1 || r.MaxIssuances > agentGuardrailMaxIssuances || r.WindowSeconds < 1 || r.WindowSeconds > agentGuardrailMaxWindowSeconds) {
		return ErrInvalidAgentGuardrail
	}
	if t := l.MaxTokenTTLSeconds; t != nil && (*t < 1 || *t > agentGuardrailMaxWindowSeconds) {
		return ErrInvalidAgentGuardrail
	}
	if d := l.MaxDelegationDepth; d != nil && (*d < 0 || *d > agentGuardrailMaxDepth) {
		return ErrInvalidAgentGuardrail
	}
	for _, action := range []AgentGuardrailViolationAction{
		l.OnViolation.AllowedResources, l.OnViolation.AllowedAuthorizationDetailTypes, l.OnViolation.IssuanceRate, l.OnViolation.ExpiresAt,
	} {
		if action != "" && action != AgentGuardrailDeny && action != AgentGuardrailRequireApproval {
			return ErrInvalidAgentGuardrail
		}
	}
	return nil
}

// Version は内容から導くバージョンである。管理者の上げ忘れで監査が偽にならない。
func (l AgentGuardrailLimits) Version() string {
	raw, _ := json.Marshal(l) //nolint:errchkjson // 固定の型の直列化は失敗しない。
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:16])
}

// AgentGuardrailRequest は発行から判定に渡す値である。
type AgentGuardrailRequest struct {
	Resources       []string
	DetailTypes     []string
	DelegationDepth int
}

// AgentGuardrailFinding は一つの層が超過と判定した項目である。
type AgentGuardrailFinding struct {
	Reason string
	Action AgentGuardrailViolationAction
}

func effectiveAction(action AgentGuardrailViolationAction) AgentGuardrailViolationAction {
	if action == "" {
		return AgentGuardrailDeny
	}
	return action
}

// Evaluate は計数を除く項目を判定する。計数は作用なのでユースケースが編成する。
// aud を指定しない発行は許可した資源に収まると示せないので超過とする。
func (l AgentGuardrailLimits) Evaluate(req AgentGuardrailRequest, now time.Time) []AgentGuardrailFinding {
	var findings []AgentGuardrailFinding
	if l.AllowedResources != nil && !allContained(req.Resources, l.AllowedResources, true) {
		findings = append(findings, AgentGuardrailFinding{AgentGuardrailReasonResource, effectiveAction(l.OnViolation.AllowedResources)})
	}
	if l.AllowedAuthorizationDetailTypes != nil && !allContained(req.DetailTypes, l.AllowedAuthorizationDetailTypes, false) {
		findings = append(findings, AgentGuardrailFinding{AgentGuardrailReasonDetailType, effectiveAction(l.OnViolation.AllowedAuthorizationDetailTypes)})
	}
	if l.ExpiresAt != nil && !now.Before(*l.ExpiresAt) {
		findings = append(findings, AgentGuardrailFinding{AgentGuardrailReasonExpired, effectiveAction(l.OnViolation.ExpiresAt)})
	}
	if l.MaxDelegationDepth != nil && req.DelegationDepth > *l.MaxDelegationDepth {
		findings = append(findings, AgentGuardrailFinding{AgentGuardrailReasonDelegationDepth, AgentGuardrailDeny})
	}
	if l.RequireApproval {
		findings = append(findings, AgentGuardrailFinding{AgentGuardrailReasonApproval, AgentGuardrailRequireApproval})
	}
	return findings
}

func allContained(values, allowed []string, requireAny bool) bool {
	if requireAny && len(values) == 0 {
		return false
	}
	for _, value := range values {
		if !slices.Contains(allowed, value) {
			return false
		}
	}
	return true
}

// TokenLifetimeSeconds は defaultSeconds をこの層の上限で短くした有効期間を返す。期限を過ぎた
// 層は承認で免除された発行にだけ届くので、期限では短くしない。
func (l AgentGuardrailLimits) TokenLifetimeSeconds(defaultSeconds int, now time.Time) int {
	lifetime := defaultSeconds
	if l.MaxTokenTTLSeconds != nil {
		lifetime = min(lifetime, *l.MaxTokenTTLSeconds)
	}
	if l.ExpiresAt != nil && now.Before(*l.ExpiresAt) {
		lifetime = min(lifetime, max(1, int(l.ExpiresAt.Sub(now).Seconds())))
	}
	return lifetime
}
