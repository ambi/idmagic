package usecases

import (
	"context"
	"slices"
	"strings"

	accessports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/internal/approval/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const mcpMethodResourceType = "idmagic_mcp_method"

type resourceOperation struct {
	Rule        oauthdomain.ResourceAccessRule
	Request     ResourceAccessRequest
	Registered  bool
	Preparation bool
	// Target は、組み込みの型で対象ごとの権限を持つツールの引数から解決した対象 ID である。
	Target string
}

func resolveResourceOperation(server *oauthdomain.McpResourceServer, in ResourceAccessRequest) resourceOperation {
	if in.ResourceType == mcpMethodResourceType && in.Relation == "access" {
		registered := slices.Contains([]string{"server/discover", "initialize", "notifications/initialized", "ping", "tools/list"}, in.ResourceID) && slices.ContainsFunc(server.AccessRules, func(r oauthdomain.ResourceAccessRule) bool { return r.Tool != nil })
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
	target := ""
	if rule.Tool.ResourceIDArgument != "" {
		args, err := approvaldomain.DecodeStrictJSONObject(in.OperationParameters)
		if err != nil {
			return resourceOperation{Request: in}
		}
		id, ok := args[rule.Tool.ResourceIDArgument].(string)
		if !ok || id == "" {
			return resourceOperation{Request: in}
		}
		target = id
	}
	if rule.ResourceType == spec.ResourceTypeMcpOperation {
		in.ResourceID = server.OperationResourceID(rule)
	} else {
		in.ResourceID = target
		target = ""
	}
	in.ResourceType, in.Relation = rule.ResourceType, rule.Relation
	return resourceOperation{Rule: rule, Request: in, Registered: rule.Tool.ID != "", Target: target}
}

// checkTargetRange は組み込みの型で対象ごとの権限を持つツールに、全対象か指定した対象の範囲を要求する。
// 範囲のない操作割当は、どの対象も許可しない (REQ-OAUTH2-083)。
func checkTargetRange(ctx context.Context, checker accessports.RelationChecker, server *oauthdomain.McpResourceServer, operation resourceOperation, subject accessports.PrincipalRef, actors []accessports.PrincipalRef, scopes []string) (accessports.RelationDecision, error) {
	rule := operation.Rule
	operationID := server.OperationResourceID(rule)
	check := accessports.RelationCheck{TenantID: server.TenantID, ResourceType: spec.ResourceTypeMcpOperation, ResourceID: operationID, Relation: spec.RelationMcpInvokeAnyTarget, Subject: subject, Actors: actors, GrantedScopes: scopes, RequiredScopes: rule.Scopes}
	decision, err := checker.CheckRelation(ctx, check)
	if err != nil || decision.Permitted {
		return decision, err
	}
	// 関係の ID に使えない対象は、範囲として保存できないので許可しない。
	if strings.ContainsAny(operation.Target, ":#") {
		return decision, nil
	}
	check.ResourceID, check.Relation = spec.McpTargetResourceID(operationID, operation.Target), spec.RelationMcpInvoke
	return checker.CheckRelation(ctx, check)
}

func operationPermission(ctx context.Context, deps ResourceAccessDeps, server *oauthdomain.McpResourceServer, operation resourceOperation, subject accessports.PrincipalRef, actors []accessports.PrincipalRef, scopes []string) (accessports.RelationDecision, error) {
	rule, in := operation.Rule, operation.Request
	check := accessports.RelationCheck{TenantID: in.TenantID, ResourceType: in.ResourceType, ResourceID: in.ResourceID, Relation: in.Relation, Subject: subject, Actors: actors, GrantedScopes: scopes, RequiredScopes: rule.Scopes}
	if rule.Tool != nil {
		decision, err := checkToolAssignment(ctx, deps.Relations, server, rule, subject, actors, scopes)
		if err != nil || !decision.Permitted {
			return decision, err
		}
		if rule.ResourceType == spec.ResourceTypeMcpOperation {
			if rule.Tool.ResourceIDArgument == "" {
				return decision, nil
			}
			return checkTargetRange(ctx, deps.Relations, server, operation, subject, actors, scopes)
		}
	}
	return deps.Relations.CheckRelation(ctx, check)
}

// assignedToolNames は一覧と接続準備に使う候補を求める。業務操作の条件、承認と回数は評価しない。
func assignedToolNames(ctx context.Context, deps ResourceAccessDeps, server *oauthdomain.McpResourceServer, subject accessports.PrincipalRef, actors []accessports.PrincipalRef, scopes []string) ([]string, error) {
	names := []string{}
	for _, rule := range server.AccessRules {
		if rule.Tool == nil {
			continue
		}
		decision, err := checkToolAssignment(ctx, deps.Relations, server, rule, subject, actors, scopes)
		if err != nil {
			return nil, err
		}
		if decision.Permitted && !slices.Contains(names, rule.Tool.Name) {
			names = append(names, rule.Tool.Name)
		}
	}
	slices.Sort(names)
	return names, nil
}

func checkToolAssignment(ctx context.Context, checker accessports.RelationChecker, server *oauthdomain.McpResourceServer, rule oauthdomain.ResourceAccessRule, subject accessports.PrincipalRef, actors []accessports.PrincipalRef, scopes []string) (accessports.RelationDecision, error) {
	if rule.Tool == nil || rule.Tool.ID == "" {
		return accessports.RelationDecision{Reasons: []string{oauthdomain.ReasonOperationNotRegistered}}, nil
	}
	if checker == nil {
		return accessports.RelationDecision{}, ErrResourceAccessNotConfigured
	}
	return checker.CheckRelation(ctx, accessports.RelationCheck{TenantID: server.TenantID, ResourceType: spec.ResourceTypeMcpOperation, ResourceID: server.OperationResourceID(rule), Relation: spec.RelationMcpInvoke, Subject: subject, Actors: actors, GrantedScopes: scopes, RequiredScopes: rule.Scopes})
}

// AssignedToolScopes は発行時も現在の割当を読み、資格情報の上限から権限を増やさない。
// 対象別の権限は実際の対象が分かる実行時に再判定する。
func AssignedToolScopes(ctx context.Context, checker accessports.RelationChecker, server *oauthdomain.McpResourceServer, agentID string, ceiling []string) ([]string, error) {
	allowed := []string{}
	for _, rule := range server.AccessRules {
		if rule.Tool == nil {
			continue
		}
		decision, err := checkToolAssignment(ctx, checker, server, rule, accessports.PrincipalRef{Type: "agent", ID: agentID}, nil, ceiling)
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
