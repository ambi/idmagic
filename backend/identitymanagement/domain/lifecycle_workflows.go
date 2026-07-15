package domain

// Identity lifecycle workflow の純粋な domain model。永続化や他 context への
// 副作用はここに置かず、定義の妥当性、状態遷移、trigger 評価と revision 固定だけを
// 扱う。これにより at-least-once の worker が同じ plan を再実行しても意味が変わらない。

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"
)

type LifecycleWorkflowStatus string

const (
	LifecycleWorkflowDraft    LifecycleWorkflowStatus = "draft"
	LifecycleWorkflowEnabled  LifecycleWorkflowStatus = "enabled"
	LifecycleWorkflowDisabled LifecycleWorkflowStatus = "disabled"
	LifecycleWorkflowArchived LifecycleWorkflowStatus = "archived"
)

func (s LifecycleWorkflowStatus) Valid() bool {
	return s == LifecycleWorkflowDraft || s == LifecycleWorkflowEnabled || s == LifecycleWorkflowDisabled || s == LifecycleWorkflowArchived
}

type WorkflowTriggerKind string

const (
	WorkflowTriggerUserCreated           WorkflowTriggerKind = "user_created"
	WorkflowTriggerUserAttributesChanged WorkflowTriggerKind = "user_attributes_changed"
	WorkflowTriggerUserStatusChanged     WorkflowTriggerKind = "user_status_changed"
)

func (k WorkflowTriggerKind) Valid() bool {
	return k == WorkflowTriggerUserCreated || k == WorkflowTriggerUserAttributesChanged || k == WorkflowTriggerUserStatusChanged
}

type WorkflowFilterOperator string

const (
	WorkflowFilterEqual    WorkflowFilterOperator = "eq"
	WorkflowFilterNotEqual WorkflowFilterOperator = "not_eq"
	WorkflowFilterIn       WorkflowFilterOperator = "in"
	WorkflowFilterExists   WorkflowFilterOperator = "exists"
)

func (o WorkflowFilterOperator) Valid() bool {
	return o == WorkflowFilterEqual || o == WorkflowFilterNotEqual || o == WorkflowFilterIn || o == WorkflowFilterExists
}

type WorkflowActionKind string

const (
	WorkflowActionAddGroupMember      WorkflowActionKind = "add_group_member"
	WorkflowActionRemoveGroupMember   WorkflowActionKind = "remove_group_member"
	WorkflowActionAssignApplication   WorkflowActionKind = "assign_application"
	WorkflowActionUnassignApplication WorkflowActionKind = "unassign_application"
	WorkflowActionSetRequiredAction   WorkflowActionKind = "set_required_action"
	WorkflowActionClearRequiredAction WorkflowActionKind = "clear_required_action"
	WorkflowActionEnableUser          WorkflowActionKind = "enable_user"
	WorkflowActionDisableUser         WorkflowActionKind = "disable_user"
	WorkflowActionSendEmail           WorkflowActionKind = "send_email"
)

func (k WorkflowActionKind) Valid() bool {
	switch k {
	case WorkflowActionAddGroupMember, WorkflowActionRemoveGroupMember,
		WorkflowActionAssignApplication, WorkflowActionUnassignApplication,
		WorkflowActionSetRequiredAction, WorkflowActionClearRequiredAction,
		WorkflowActionEnableUser, WorkflowActionDisableUser, WorkflowActionSendEmail:
		return true
	}
	return false
}

type WorkflowRunStatus string

const (
	WorkflowRunQueued          WorkflowRunStatus = "queued"
	WorkflowRunRunning         WorkflowRunStatus = "running"
	WorkflowRunSucceeded       WorkflowRunStatus = "succeeded"
	WorkflowRunPartiallyFailed WorkflowRunStatus = "partially_failed"
	WorkflowRunFailed          WorkflowRunStatus = "failed"
	WorkflowRunCanceled        WorkflowRunStatus = "canceled"
)

func (s WorkflowRunStatus) Terminal() bool {
	return s == WorkflowRunSucceeded || s == WorkflowRunPartiallyFailed || s == WorkflowRunFailed || s == WorkflowRunCanceled
}

type WorkflowStepOutcome string

const (
	WorkflowStepPending  WorkflowStepOutcome = "pending"
	WorkflowStepChanged  WorkflowStepOutcome = "changed"
	WorkflowStepNoop     WorkflowStepOutcome = "no_op"
	WorkflowStepFailed   WorkflowStepOutcome = "failed"
	WorkflowStepCanceled WorkflowStepOutcome = "canceled"
)

func (o WorkflowStepOutcome) Complete() bool {
	return o == WorkflowStepChanged || o == WorkflowStepNoop || o == WorkflowStepCanceled
}

// WorkflowFilter.Value は AttributeValue と同じ JSON native value を取る。exists は
// value を持たず、他の operator は value を必須にする。
type WorkflowFilter struct {
	Field    string                 `json:"field"`
	Operator WorkflowFilterOperator `json:"operator"`
	Value    any                    `json:"value,omitempty"`
}

func (f WorkflowFilter) Validate() error {
	if strings.TrimSpace(f.Field) == "" || !f.Operator.Valid() {
		return errors.New("workflow filter field and operator are required")
	}
	if f.Operator == WorkflowFilterExists && f.Value != nil {
		return errors.New("exists workflow filter must not have a value")
	}
	if f.Operator != WorkflowFilterExists && f.Value == nil {
		return errors.New("workflow filter value is required")
	}
	return nil
}

type WorkflowTrigger struct {
	Kind              WorkflowTriggerKind `json:"kind"`
	WatchedAttributes []string            `json:"watched_attributes,omitempty"`
	FromStatus        *UserStatus         `json:"from_status,omitempty"`
	ToStatus          *UserStatus         `json:"to_status,omitempty"`
	Filters           []WorkflowFilter    `json:"filters,omitempty"`
}

func (t WorkflowTrigger) Validate() error {
	if !t.Kind.Valid() || len(t.Filters) > 20 {
		return errors.New("invalid workflow trigger")
	}
	for _, f := range t.Filters {
		if err := f.Validate(); err != nil {
			return err
		}
	}
	switch t.Kind {
	case WorkflowTriggerUserAttributesChanged:
		if len(t.WatchedAttributes) == 0 {
			return errors.New("attribute trigger requires watched attributes")
		}
		for _, field := range t.WatchedAttributes {
			if strings.TrimSpace(field) == "" {
				return errors.New("watched attribute must not be empty")
			}
		}
	case WorkflowTriggerUserStatusChanged:
		if t.FromStatus == nil || t.ToStatus == nil || !t.FromStatus.Valid() || !t.ToStatus.Valid() {
			return errors.New("status trigger requires valid from and to status")
		}
	default:
		if len(t.WatchedAttributes) != 0 || t.FromStatus != nil || t.ToStatus != nil {
			return errors.New("trigger has fields not applicable to its kind")
		}
	}
	return nil
}

type WorkflowAction struct {
	Kind           WorkflowActionKind `json:"kind"`
	GroupID        string             `json:"group_id,omitempty"`
	ApplicationID  string             `json:"application_id,omitempty"`
	Visibility     string             `json:"visibility,omitempty"`
	RequiredAction RequiredAction     `json:"required_action,omitempty"`
	Reason         string             `json:"reason,omitempty"`
	TemplateKey    string             `json:"template_key,omitempty"`
}

func (a WorkflowAction) Validate() error {
	if !a.Kind.Valid() {
		return errors.New("invalid workflow action kind")
	}
	requires := func(v string) error {
		if strings.TrimSpace(v) == "" {
			return errors.New("workflow action required value is missing")
		}
		return nil
	}
	switch a.Kind {
	case WorkflowActionAddGroupMember, WorkflowActionRemoveGroupMember:
		return requires(a.GroupID)
	case WorkflowActionAssignApplication, WorkflowActionUnassignApplication:
		return requires(a.ApplicationID)
	case WorkflowActionSetRequiredAction, WorkflowActionClearRequiredAction:
		if !a.RequiredAction.Valid() {
			return errors.New("workflow action required_action is invalid")
		}
	case WorkflowActionSendEmail:
		return requires(a.TemplateKey)
	}
	return nil
}

type LifecycleWorkflowRevision struct {
	WorkflowID string           `json:"workflow_id"`
	TenantID   string           `json:"tenant_id"`
	Revision   int64            `json:"revision"`
	Trigger    WorkflowTrigger  `json:"trigger"`
	Actions    []WorkflowAction `json:"actions"`
	CreatedAt  time.Time        `json:"created_at"`
}

func (r LifecycleWorkflowRevision) Validate() error {
	if r.WorkflowID == "" || r.TenantID == "" || r.Revision < 1 || r.CreatedAt.IsZero() || len(r.Actions) < 1 || len(r.Actions) > 20 {
		return errors.New("invalid workflow revision")
	}
	if err := r.Trigger.Validate(); err != nil {
		return err
	}
	for _, action := range r.Actions {
		if err := action.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type LifecycleWorkflow struct {
	ID              string                  `json:"id"`
	TenantID        string                  `json:"tenant_id"`
	Name            string                  `json:"name"`
	Description     *string                 `json:"description,omitempty"`
	Status          LifecycleWorkflowStatus `json:"status"`
	CurrentRevision int64                   `json:"current_revision"`
	EnabledRevision *int64                  `json:"enabled_revision,omitempty"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

func (w LifecycleWorkflow) Validate() error {
	if w.ID == "" || w.TenantID == "" || strings.TrimSpace(w.Name) == "" || w.CurrentRevision < 1 || w.CreatedAt.IsZero() || w.UpdatedAt.IsZero() || !w.Status.Valid() {
		return errors.New("invalid lifecycle workflow")
	}
	if w.Description != nil && len(*w.Description) > 500 {
		return errors.New("workflow description is too long")
	}
	if w.Status == LifecycleWorkflowEnabled && (w.EnabledRevision == nil || *w.EnabledRevision < 1 || *w.EnabledRevision > w.CurrentRevision) {
		return errors.New("enabled workflow must select an existing revision")
	}
	if w.Status != LifecycleWorkflowEnabled && w.EnabledRevision != nil {
		return errors.New("only enabled workflow may have enabled revision")
	}
	return nil
}

func (w *LifecycleWorkflow) Enable(revision int64, now time.Time) error {
	if w.Status == LifecycleWorkflowArchived {
		return errors.New("archived workflow cannot be enabled")
	}
	if revision < 1 || revision > w.CurrentRevision {
		return errors.New("workflow revision does not exist")
	}
	w.Status, w.EnabledRevision, w.UpdatedAt = LifecycleWorkflowEnabled, &revision, now.UTC()
	return w.Validate()
}

func (w *LifecycleWorkflow) Disable(now time.Time) error {
	if w.Status != LifecycleWorkflowEnabled {
		return errors.New("only enabled workflow can be disabled")
	}
	w.Status, w.EnabledRevision, w.UpdatedAt = LifecycleWorkflowDisabled, nil, now.UTC()
	return nil
}

func (w *LifecycleWorkflow) Delete(now time.Time) error {
	if w.Status == LifecycleWorkflowArchived {
		return errors.New("workflow is already deleted")
	}
	w.Status, w.EnabledRevision, w.UpdatedAt = LifecycleWorkflowArchived, nil, now.UTC()
	return nil
}

type WorkflowRun struct {
	ID                 string              `json:"id"`
	TenantID           string              `json:"tenant_id"`
	WorkflowID         string              `json:"workflow_id"`
	Revision           int64               `json:"revision"`
	SourceOccurrenceID string              `json:"source_occurrence_id"`
	TargetUserID       string              `json:"target_user_id"`
	TriggerKind        WorkflowTriggerKind `json:"trigger_kind"`
	ChangedFields      []string            `json:"changed_fields,omitempty"`
	Actions            []WorkflowAction    `json:"actions"`
	Status             WorkflowRunStatus   `json:"status"`
	JobID              *string             `json:"job_id,omitempty"`
	TriggeredAt        time.Time           `json:"triggered_at"`
}

func (r WorkflowRun) Validate() error {
	if r.ID == "" || r.TenantID == "" || r.WorkflowID == "" || r.Revision < 1 || r.SourceOccurrenceID == "" || r.TargetUserID == "" || !r.TriggerKind.Valid() || !r.Status.Valid() || r.TriggeredAt.IsZero() || len(r.Actions) == 0 {
		return errors.New("invalid workflow run")
	}
	for _, action := range r.Actions {
		if err := action.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (s WorkflowRunStatus) Valid() bool {
	return s == WorkflowRunQueued || s == WorkflowRunRunning || s == WorkflowRunSucceeded || s == WorkflowRunPartiallyFailed || s == WorkflowRunFailed || s == WorkflowRunCanceled
}

type WorkflowStep struct {
	RunID       string              `json:"run_id"`
	Index       int                 `json:"index"`
	Action      WorkflowAction      `json:"action"`
	Outcome     WorkflowStepOutcome `json:"outcome"`
	ErrorCode   string              `json:"error_code,omitempty"`
	CompletedAt *time.Time          `json:"completed_at,omitempty"`
}

func (s WorkflowStep) Validate() error {
	if s.RunID == "" || s.Index < 0 || !s.Outcome.Valid() {
		return errors.New("invalid workflow step")
	}
	return s.Action.Validate()
}

func (o WorkflowStepOutcome) Valid() bool {
	return o == WorkflowStepPending || o == WorkflowStepChanged || o == WorkflowStepNoop || o == WorkflowStepFailed || o == WorkflowStepCanceled
}

// TriggerMatch は User mutation が trigger を発火させたときに run に残す最小情報。
// 値自体を保存しないため PII を durable history に持ち込まない。
type TriggerMatch struct {
	Kind          WorkflowTriggerKind
	ChangedFields []string
}

func EvaluateWorkflowTrigger(trigger WorkflowTrigger, before, after *User, changedFields []string, originRunID string) (TriggerMatch, bool) {
	if originRunID != "" || after == nil || trigger.Validate() != nil {
		return TriggerMatch{}, false
	}
	matched := false
	switch trigger.Kind {
	case WorkflowTriggerUserCreated:
		matched = before == nil
	case WorkflowTriggerUserAttributesChanged:
		if before == nil {
			return TriggerMatch{}, false
		}
		for _, field := range trigger.WatchedAttributes {
			if slices.Contains(changedFields, field) {
				matched = true
				break
			}
		}
	case WorkflowTriggerUserStatusChanged:
		if before == nil {
			return TriggerMatch{}, false
		}
		matched = before.Lifecycle.EffectiveStatus() == *trigger.FromStatus && after.Lifecycle.EffectiveStatus() == *trigger.ToStatus
	}
	if !matched || !workflowFiltersMatch(trigger.Filters, after) {
		return TriggerMatch{}, false
	}
	return TriggerMatch{Kind: trigger.Kind, ChangedFields: slices.Clone(changedFields)}, true
}

func workflowFiltersMatch(filters []WorkflowFilter, user *User) bool {
	for _, filter := range filters {
		value, exists := workflowUserField(user, filter.Field)
		switch filter.Operator {
		case WorkflowFilterExists:
			if !exists {
				return false
			}
		case WorkflowFilterEqual:
			if !exists || !reflect.DeepEqual(value, filter.Value) {
				return false
			}
		case WorkflowFilterNotEqual:
			if exists && reflect.DeepEqual(value, filter.Value) {
				return false
			}
		case WorkflowFilterIn:
			values, ok := filter.Value.([]any)
			if !ok {
				return false
			}
			if !exists || !slices.ContainsFunc(values, func(v any) bool { return reflect.DeepEqual(value, v) }) {
				return false
			}
		}
	}
	return true
}

func workflowUserField(user *User, field string) (any, bool) {
	switch field {
	case "preferred_username":
		return user.PreferredUsername, true
	case "email":
		if user.Email != nil {
			return *user.Email, true
		}
	case "status":
		return string(user.Lifecycle.EffectiveStatus()), true
	}
	value, ok := user.Attributes[field]
	if !ok {
		return nil, false
	}
	return value.JSONValue(), value.JSONValue() != nil
}

func PlanWorkflowRun(id string, revision LifecycleWorkflowRevision, targetUserID, occurrenceID string, match TriggerMatch, now time.Time) (*WorkflowRun, []WorkflowStep, error) {
	if err := revision.Validate(); err != nil {
		return nil, nil, err
	}
	if id == "" || targetUserID == "" || occurrenceID == "" {
		return nil, nil, fmt.Errorf("workflow run id, user and occurrence are required")
	}
	run := &WorkflowRun{ID: id, TenantID: revision.TenantID, WorkflowID: revision.WorkflowID, Revision: revision.Revision, SourceOccurrenceID: occurrenceID, TargetUserID: targetUserID, TriggerKind: match.Kind, ChangedFields: slices.Clone(match.ChangedFields), Actions: slices.Clone(revision.Actions), Status: WorkflowRunQueued, TriggeredAt: now.UTC()}
	steps := make([]WorkflowStep, len(revision.Actions))
	for i, action := range revision.Actions {
		steps[i] = WorkflowStep{RunID: id, Index: i, Action: action, Outcome: WorkflowStepPending}
	}
	return run, steps, nil
}
