package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

var ErrInvalidDynamicGroupRule = errors.New("invalid dynamic group rule")

type DynamicGroupDeps struct {
	GroupRepo  idmports.GroupRepository
	UserRepo   idmports.UserRepository
	SchemaRepo tenantports.TenantUserAttributeSchemaRepository
	JobRepo    jobsports.JobRepository
	Emit       func(spec.DomainEvent) error
}

type DynamicGroupPreview struct {
	UserID    string  `json:"user_id"`
	Matched   bool    `json:"matched"`
	Change    string  `json:"change"`
	ErrorCode *string `json:"error_code,omitempty"`
}

func effectiveRuleDefs(ctx context.Context, deps DynamicGroupDeps, tenantID string) ([]idmdomain.UserAttributeDef, error) {
	defs := idmdomain.BuiltinUserAttributeDefs()
	if deps.SchemaRepo == nil {
		return defs, nil
	}
	schema, err := deps.SchemaRepo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if schema != nil {
		defs = schema.EffectiveDefs()
	}
	return defs, nil
}

func UpdateDynamicGroupRule(ctx context.Context, deps DynamicGroupDeps, actorUserID, groupID, expression string, now time.Time) (*idmdomain.DynamicGroupRule, error) {
	tenantID := tenancy.TenantID(ctx)
	group, err := deps.GroupRepo.FindByID(ctx, tenantID, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}
	if group.MembershipType.Effective() != idmdomain.GroupMembershipDynamic {
		return nil, ErrDynamicMembershipManaged
	}
	defs, err := effectiveRuleDefs(ctx, deps, tenantID)
	if err != nil {
		return nil, err
	}
	compiled, err := idmdomain.CompileDynamicGroupRule(expression, defs)
	if err != nil {
		return nil, errors.Join(ErrInvalidDynamicGroupRule, err)
	}
	existing, err := deps.GroupRepo.FindDynamicRule(ctx, tenantID, groupID)
	if err != nil {
		return nil, err
	}
	version := int64(1)
	createdAt := normalizedNow(now)
	enabled := false
	if existing != nil {
		version = existing.Version + 1
		createdAt = existing.CreatedAt
		enabled = existing.Enabled
	}
	rule := &idmdomain.DynamicGroupRule{GroupID: groupID, TenantID: tenantID, Expression: expression, Enabled: enabled, Version: version, ReferencedAttributes: compiled.References(), CreatedAt: createdAt, UpdatedAt: normalizedNow(now)}
	if err := rule.Validate(); err != nil {
		return nil, errors.Join(ErrInvalidDynamicGroupRule, err)
	}
	if err := deps.GroupRepo.SaveDynamicRule(ctx, rule); err != nil {
		return nil, err
	}
	if err := adminEmit(deps.Emit, &spec.DynamicGroupRuleUpdated{At: normalizedNow(now), TenantID: tenantID, ActorUserID: actorUserID, GroupID: groupID, RuleVersion: rule.Version}); err != nil {
		return nil, err
	}
	if rule.Enabled {
		err = scheduleDynamicGroupReconcile(ctx, deps, rule, normalizedNow(now))
	}
	return rule, err
}

func SetDynamicGroupRuleEnabled(ctx context.Context, deps DynamicGroupDeps, actorUserID, groupID string, enabled bool, now time.Time) (*idmdomain.DynamicGroupRule, error) {
	tenantID := tenancy.TenantID(ctx)
	rule, err := deps.GroupRepo.FindDynamicRule(ctx, tenantID, groupID)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, ErrInvalidDynamicGroupRule
	}
	if rule.Enabled == enabled {
		return rule, nil
	}
	rule.Enabled = enabled
	rule.Version++
	rule.UpdatedAt = normalizedNow(now)
	if err := deps.GroupRepo.SaveDynamicRule(ctx, rule); err != nil {
		return nil, err
	}
	var event spec.DomainEvent = &spec.DynamicGroupRuleDisabled{At: normalizedNow(now), TenantID: tenantID, ActorUserID: actorUserID, GroupID: groupID, RuleVersion: rule.Version}
	if enabled {
		event = &spec.DynamicGroupRuleEnabled{At: normalizedNow(now), TenantID: tenantID, ActorUserID: actorUserID, GroupID: groupID, RuleVersion: rule.Version}
	}
	if err := adminEmit(deps.Emit, event); err != nil {
		return nil, err
	}
	if enabled {
		err = scheduleDynamicGroupReconcile(ctx, deps, rule, normalizedNow(now))
	} else {
		_, err = ReconcileDynamicGroup(ctx, deps, rule, normalizedNow(now))
	}
	return rule, err
}

type DynamicGroupReconcileParams struct {
	GroupID     string `json:"group_id"`
	RuleVersion int64  `json:"rule_version"`
}

func scheduleDynamicGroupReconcile(ctx context.Context, deps DynamicGroupDeps, rule *idmdomain.DynamicGroupRule, now time.Time) error {
	if deps.JobRepo == nil {
		result, err := ReconcileDynamicGroup(ctx, deps, rule, now)
		if err != nil {
			return err
		}
		return adminEmit(deps.Emit, &spec.DynamicMembershipEvaluated{At: now, TenantID: rule.TenantID, GroupID: rule.GroupID, RuleVersion: rule.Version, AddedCount: result.Added, RemovedCount: result.Removed, UnchangedCount: result.Unchanged, ErrorCount: result.Errors})
	}
	params, err := json.Marshal(DynamicGroupReconcileParams{GroupID: rule.GroupID, RuleVersion: rule.Version})
	if err != nil {
		return err
	}
	dedupKey := fmt.Sprintf("dynamic-group:%s:v%d", rule.GroupID, rule.Version)
	_, err = jobsusecases.Enqueue(ctx, jobsusecases.EnqueueDeps{Repo: deps.JobRepo}, jobsports.EnqueueInput{
		TenantID: rule.TenantID, Kind: jobsdomain.KindDynamicGroupReconcile, Params: params, DedupKey: &dedupKey,
	}, now)
	return err
}

func DynamicGroupReconcileHandler(deps DynamicGroupDeps) func(context.Context, *jobsdomain.Job) (json.RawMessage, error) {
	return func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
		var params DynamicGroupReconcileParams
		if err := json.Unmarshal(job.Params, &params); err != nil {
			return nil, err
		}
		ctx = tenancy.WithTenant(ctx, &tenancydomain.Tenant{ID: job.TenantID}, "", "")
		rule, err := deps.GroupRepo.FindDynamicRule(ctx, job.TenantID, params.GroupID)
		if err != nil {
			return nil, err
		}
		// 古いジョブは新しいルールの所属を上書きせず成功扱いで終了する。
		if rule == nil || !rule.Enabled || rule.Version != params.RuleVersion {
			return json.Marshal(DynamicReconcileResult{})
		}
		result, err := ReconcileDynamicGroup(ctx, deps, rule, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		if err := adminEmit(deps.Emit, &spec.DynamicMembershipEvaluated{At: time.Now().UTC(), TenantID: rule.TenantID, GroupID: rule.GroupID, RuleVersion: rule.Version, AddedCount: result.Added, RemovedCount: result.Removed, UnchangedCount: result.Unchanged, ErrorCount: result.Errors}); err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}

func PreviewDynamicGroupRule(ctx context.Context, deps DynamicGroupDeps, groupID, expression string, userIDs []string) ([]DynamicGroupPreview, error) {
	if len(userIDs) > 100 {
		return nil, errors.Join(ErrInvalidDynamicGroupRule, fmt.Errorf("preview supports at most 100 users"))
	}
	tenantID := tenancy.TenantID(ctx)
	group, err := deps.GroupRepo.FindByID(ctx, tenantID, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}
	defs, err := effectiveRuleDefs(ctx, deps, tenantID)
	if err != nil {
		return nil, err
	}
	compiled, err := idmdomain.CompileDynamicGroupRule(expression, defs)
	if err != nil {
		return nil, errors.Join(ErrInvalidDynamicGroupRule, err)
	}
	members, err := deps.GroupRepo.ListMembersByGroup(ctx, tenantID, groupID)
	if err != nil {
		return nil, err
	}
	current := map[string]bool{}
	for _, member := range members {
		current[member.UserID] = true
	}
	out := make([]DynamicGroupPreview, 0, len(userIDs))
	for _, id := range userIDs {
		user, err := deps.UserRepo.FindBySub(ctx, id)
		if err != nil {
			return nil, err
		}
		if user == nil || user.TenantID != tenantID {
			return nil, ErrUserNotFound
		}
		matched, evalErr := compiled.Evaluate(*user)
		change := "unchanged"
		if matched && !current[id] {
			change = "add"
		}
		if !matched && current[id] {
			change = "remove"
		}
		item := DynamicGroupPreview{UserID: id, Matched: matched, Change: change}
		if evalErr != nil {
			code := "evaluation_error"
			item.ErrorCode = &code
		}
		out = append(out, item)
	}
	return out, nil
}

type DynamicReconcileResult struct{ Added, Removed, Unchanged, Errors int }

func SyncDynamicGroupsForUser(ctx context.Context, deps DynamicGroupDeps, user *idmdomain.User, now time.Time) error {
	rules, err := deps.GroupRepo.ListDynamicRules(ctx, user.TenantID)
	if err != nil {
		return err
	}
	defs, err := effectiveRuleDefs(ctx, deps, user.TenantID)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		members, err := deps.GroupRepo.ListMembersByGroup(ctx, user.TenantID, rule.GroupID)
		if err != nil {
			return err
		}
		var existing *idmdomain.GroupMember
		for _, member := range members {
			if member.UserID == user.ID {
				existing = member
				break
			}
		}
		matched := false
		if rule.Enabled {
			compiled, compileErr := idmdomain.CompileDynamicGroupRule(rule.Expression, defs)
			if compileErr == nil {
				matched, _ = compiled.Evaluate(*user)
			}
		}
		valid := existing != nil && existing.Source == idmdomain.MembershipSourceDynamicRule && existing.RuleVersion != nil && *existing.RuleVersion == rule.Version
		if matched && valid {
			continue
		}
		if existing != nil {
			if _, err := deps.GroupRepo.RemoveMember(ctx, user.TenantID, rule.GroupID, user.ID); err != nil {
				return err
			}
		}
		if matched {
			version := rule.Version
			if _, err := deps.GroupRepo.AddMember(ctx, &idmdomain.GroupMember{GroupID: rule.GroupID, UserID: user.ID, Source: idmdomain.MembershipSourceDynamicRule, RuleVersion: &version, CreatedAt: normalizedNow(now)}); err != nil {
				return err
			}
		}
	}
	return nil
}

func ReconcileDynamicGroup(ctx context.Context, deps DynamicGroupDeps, rule *idmdomain.DynamicGroupRule, now time.Time) (DynamicReconcileResult, error) {
	var result DynamicReconcileResult
	members, err := deps.GroupRepo.ListMembersByGroup(ctx, rule.TenantID, rule.GroupID)
	if err != nil {
		return result, err
	}
	current := map[string]*idmdomain.GroupMember{}
	for _, member := range members {
		current[member.UserID] = member
	}
	if !rule.Enabled {
		for id := range current {
			removed, err := deps.GroupRepo.RemoveMember(ctx, rule.TenantID, rule.GroupID, id)
			if err != nil {
				return result, err
			}
			if removed {
				result.Removed++
			}
		}
		return result, nil
	}
	defs, err := effectiveRuleDefs(ctx, deps, rule.TenantID)
	if err != nil {
		return result, err
	}
	compiled, err := idmdomain.CompileDynamicGroupRule(rule.Expression, defs)
	if err != nil {
		return result, err
	}
	users, err := deps.UserRepo.FindAll(ctx, rule.TenantID)
	if err != nil {
		return result, err
	}
	seen := map[string]bool{}
	for _, user := range users {
		matched, evalErr := compiled.Evaluate(*user)
		if evalErr != nil {
			result.Errors++
			matched = false
		}
		seen[user.ID] = true
		existing := current[user.ID]
		valid := existing != nil && existing.Source == idmdomain.MembershipSourceDynamicRule && existing.RuleVersion != nil && *existing.RuleVersion == rule.Version
		if matched && valid {
			result.Unchanged++
			continue
		}
		if existing != nil {
			if _, err := deps.GroupRepo.RemoveMember(ctx, rule.TenantID, rule.GroupID, user.ID); err != nil {
				return result, err
			}
			result.Removed++
		}
		if matched {
			version := rule.Version
			if _, err := deps.GroupRepo.AddMember(ctx, &idmdomain.GroupMember{GroupID: rule.GroupID, UserID: user.ID, Source: idmdomain.MembershipSourceDynamicRule, RuleVersion: &version, CreatedAt: now}); err != nil {
				return result, err
			}
			result.Added++
		}
	}
	for id := range current {
		if !seen[id] {
			if _, err := deps.GroupRepo.RemoveMember(ctx, rule.TenantID, rule.GroupID, id); err != nil {
				return result, err
			}
			result.Removed++
		}
	}
	return result, nil
}
