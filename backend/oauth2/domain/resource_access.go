package domain

// リソースサーバーが評価を求められる操作の規則 (REQ-OAUTH2-062、REQ-OAUTH2-064)。
// 認可モデルの型はテナント全体で共有されるので、登録にない操作を評価させないことで、
// あるリソースサーバーが別のリソースサーバーの資源の関係を探索する経路を閉じる。

import (
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// MaxResourceAccessRules は 1 つの登録が持てる規則の上限。
const MaxResourceAccessRules = 100

const maxResourceAccessNameLength = 64

var resourceAccessNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// 評価の不許可の理由。識別子を含めず、リソースサーバーへそのまま返す。
const (
	ReasonTokenRejected               = "token_rejected"
	ReasonOperationNotRegistered      = "operation_not_registered"
	ReasonAuthorizationDetailMismatch = "authorization_detail_mismatch"
	ReasonPrincipalUnresolved         = "principal_unresolved"
	ReasonSenderProofInvalid          = "sender_proof_invalid"
	ReasonApprovalRequired            = "approval_required"
	ReasonApprovedOperationMismatch   = "approved_operation_mismatch"
	ReasonApprovalExpired             = "approval_expired"
	ReasonApprovalAlreadyUsed         = "approval_already_used"
	ReasonConditionDenied             = "condition_denied"
	ReasonConditionIndeterminate      = "condition_indeterminate"
)

// ResourceAccessRule は評価できる 1 つの操作と、その操作がトークンへ求める制約である。
type ResourceAccessRule struct {
	ResourceType            string   `json:"resource_type"`
	Relation                string   `json:"relation"`
	Scopes                  []string `json:"scopes"`
	AuthorizationDetailType string   `json:"authorization_detail_type,omitempty"`
	// RequiresApprovedOperation は、人が承認した一回の操作だけを許可する (REQ-OAUTH2-065)。
	RequiresApprovedOperation bool `json:"requires_approved_operation,omitempty"`
	// Condition は、ほかの制約を満たす操作の結果を分ける。
	Condition *ResourceAccessCondition `json:"condition,omitempty"`
}

// PermitsDetails は、規則が求める構造化された権限をトークンが持つかを判定する。
// RFC 9396 は locations と actions の省略の意味を種類に委ねるので、省略を全許可と
// 読まず、明示して一致するものだけを受ける。
func (r ResourceAccessRule) PermitsDetails(details []spec.AuthorizationDetail, resource, resourceID string) bool {
	if r.AuthorizationDetailType == "" {
		return true
	}
	return slices.ContainsFunc(details, func(d spec.AuthorizationDetail) bool {
		return d.Type == r.AuthorizationDetailType &&
			slices.Contains(d.Locations, resource) &&
			slices.Contains(d.Actions, r.Relation) &&
			(d.Identifier == "" || d.Identifier == resourceID)
	})
}

// RuleFor は型と関係の組に一致する規則を返す。
func (m McpResourceServer) RuleFor(resourceType, relation string) (ResourceAccessRule, bool) {
	for _, rule := range m.AccessRules {
		if rule.ResourceType == resourceType && rule.Relation == relation {
			return rule, true
		}
	}
	return ResourceAccessRule{}, false
}

func (m McpResourceServer) validateAccessRules() error {
	if len(m.AccessRules) > MaxResourceAccessRules {
		return fmt.Errorf("access_rules must not exceed %d", MaxResourceAccessRules)
	}
	seen := make(map[[2]string]struct{}, len(m.AccessRules))
	for _, rule := range m.AccessRules {
		if !validResourceAccessName(rule.ResourceType) || !validResourceAccessName(rule.Relation) {
			return errors.New("access_rules resource_type and relation must match ^[a-z][a-z0-9_]*$ within 64 characters")
		}
		key := [2]string{rule.ResourceType, rule.Relation}
		if _, duplicate := seen[key]; duplicate {
			return errors.New("access_rules must not repeat a resource_type and relation pair")
		}
		seen[key] = struct{}{}
		if rule.RequiresApprovedOperation && rule.AuthorizationDetailType == "" {
			return errors.New("access_rules requires_approved_operation needs authorization_detail_type")
		}
		if rule.Condition != nil {
			if err := rule.Condition.Validate(); err != nil {
				return err
			}
			if rule.RequiresApprovedOperation && rule.Condition.RequireApprovalWhen != "" {
				return errors.New("access_rules require_approval_when must not be combined with requires_approved_operation")
			}
		}
		if len(rule.Scopes) == 0 {
			return errors.New("access_rules scopes must not be empty")
		}
		for _, scope := range rule.Scopes {
			if !slices.Contains(m.Scopes, scope) {
				return errors.New("access_rules scopes must be a subset of the resource server scopes")
			}
		}
	}
	return nil
}

func validResourceAccessName(value string) bool {
	return len(value) <= maxResourceAccessNameLength && resourceAccessNamePattern.MatchString(value)
}
