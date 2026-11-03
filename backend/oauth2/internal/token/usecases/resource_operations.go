package usecases

import (
	"context"
	"slices"

	policyports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/internal/approval/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const mcpMethodResourceType = "idmagic_mcp_method"

type resourceOperation struct {
	Rule        domain.ResourceAccessRule
	Request     ResourceAccessRequest
	Registered  bool
	Preparation bool
}

func resolveResourceOperation(server *domain.McpResourceServer, in ResourceAccessRequest) resourceOperation {
	if in.ResourceType == mcpMethodResourceType && in.Relation == "access" {
		registered := slices.Contains([]string{"server/discover", "initialize", "notifications/initialized", "ping", "tools/list"}, in.ResourceID) && slices.ContainsFunc(server.AccessRules, func(r domain.ResourceAccessRule) bool { return r.Tool != nil })
		return resourceOperation{Request: in, Registered: registered, Preparation: true}
	}
	if in.ResourceType != spec.ResourceTypeMcpOperation || in.Relation != spec.RelationMcpInvoke {
		rule, ok := server.RuleFor(in.ResourceType, in.Relation)
		return resourceOperation{Rule: rule, Request: in, Registered: ok}
	}
	rule, ok := server.ToolRuleFor(in.ResourceID)
	if !ok {
		return resourceOperation{Request: in}
	}
	if rule.ResourceType == spec.ResourceTypeMcpOperation {
		in.ResourceID = server.OperationResourceID(rule)
	} else {
		args, err := approvaldomain.DecodeStrictJSONObject(in.OperationParameters)
		if err != nil {
			return resourceOperation{Request: in}
		}
		id, ok := args[rule.Tool.ResourceIDArgument].(string)
		if !ok || id == "" {
			return resourceOperation{Request: in}
		}
		in.ResourceID = id
	}
	in.ResourceType, in.Relation = rule.ResourceType, rule.Relation
	return resourceOperation{Rule: rule, Request: in, Registered: rule.Tool.ID != ""}
}

func operationPermission(ctx context.Context, deps ResourceAccessDeps, server *domain.McpResourceServer, operation resourceOperation, subject policyports.PrincipalRef, actors []policyports.PrincipalRef, scopes []string) (policyports.RelationDecision, error) {
	rule, in := operation.Rule, operation.Request
	check := policyports.RelationCheck{TenantID: in.TenantID, ResourceType: in.ResourceType, ResourceID: in.ResourceID, Relation: in.Relation, Subject: subject, Actors: actors, GrantedScopes: scopes, RequiredScopes: rule.Scopes}
	if operation.Preparation {
		for _, operation := range server.AccessRules {
			if operation.Tool == nil {
				continue
			}
			decision, err := checkToolAssignment(ctx, deps.Relations, server, operation, subject, actors, scopes)
			if err != nil {
				return policyports.RelationDecision{}, err
			}
			if decision.Permitted {
				return decision, nil
			}
		}
		return policyports.RelationDecision{Reasons: []string{"operation_assignment_required"}}, nil
	}
	if rule.Tool != nil {
		decision, err := checkToolAssignment(ctx, deps.Relations, server, rule, subject, actors, scopes)
		if err != nil || !decision.Permitted || rule.ResourceType == spec.ResourceTypeMcpOperation {
			return decision, err
		}
	}
	return deps.Relations.CheckRelation(ctx, check)
}

func checkToolAssignment(ctx context.Context, checker policyports.RelationChecker, server *domain.McpResourceServer, rule domain.ResourceAccessRule, subject policyports.PrincipalRef, actors []policyports.PrincipalRef, scopes []string) (policyports.RelationDecision, error) {
	if rule.Tool == nil || rule.Tool.ID == "" {
		return policyports.RelationDecision{Reasons: []string{domain.ReasonOperationNotRegistered}}, nil
	}
	if checker == nil {
		return policyports.RelationDecision{}, ErrResourceAccessNotConfigured
	}
	return checker.CheckRelation(ctx, policyports.RelationCheck{TenantID: server.TenantID, ResourceType: spec.ResourceTypeMcpOperation, ResourceID: server.OperationResourceID(rule), Relation: spec.RelationMcpInvoke, Subject: subject, Actors: actors, GrantedScopes: scopes, RequiredScopes: rule.Scopes})
}

// AssignedToolScopes は発行時も現在の割当を読み、資格情報の上限から権限を増やさない。
// 対象別の権限は実際の対象が分かる実行時に再判定する。
func AssignedToolScopes(ctx context.Context, checker policyports.RelationChecker, server *domain.McpResourceServer, agentID string, ceiling []string) ([]string, error) {
	allowed := []string{}
	for _, rule := range server.AccessRules {
		if rule.Tool == nil {
			continue
		}
		decision, err := checkToolAssignment(ctx, checker, server, rule, policyports.PrincipalRef{Type: "agent", ID: agentID}, nil, ceiling)
		if err != nil {
			return nil, err
		}
		if !decision.Permitted {
			continue
		}
		for _, scope := range rule.Scopes {
			if !slices.Contains(allowed, scope) {
				allowed = append(allowed, scope)
			}
		}
	}
	return allowed, nil
}
