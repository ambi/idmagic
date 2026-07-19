package domain

import (
	"errors"
	"strings"
)

// AttributeSourceKind is the supply source for an AttributeMappingRule value
// (spec/contexts/provisioning.yaml models.AttributeSourceKind). expression is
// intentionally out of scope for the initial version (kept out of this enum).
type AttributeSourceKind string

const (
	SourceKindAttribute AttributeSourceKind = "attribute"
	SourceKindConstant  AttributeSourceKind = "constant"
)

func (k AttributeSourceKind) Valid() bool {
	return k == SourceKindAttribute || k == SourceKindConstant
}

// AttributeApplyOn controls when an AttributeMappingRule applies
// (spec/contexts/provisioning.yaml models.AttributeApplyOn).
type AttributeApplyOn string

const (
	ApplyCreateAndUpdate AttributeApplyOn = "create_and_update"
	ApplyCreateOnly      AttributeApplyOn = "create_only"
)

func (a AttributeApplyOn) Valid() bool {
	return a == ApplyCreateAndUpdate || a == ApplyCreateOnly
}

// AttributeMappingRule is a declarative mapping for one downstream attribute
// (spec/contexts/provisioning.yaml models.AttributeMappingRule).
type AttributeMappingRule struct {
	TargetPath    string              `json:"target_path"`
	SourceKind    AttributeSourceKind `json:"source_kind"`
	SourceKey     string              `json:"source_key,omitempty"`
	ConstantValue any                 `json:"constant_value,omitempty"`
	ApplyOn       AttributeApplyOn    `json:"apply_on"`
	Required      bool                `json:"required"`
	DefaultValue  any                 `json:"default_value,omitempty"`
}

func (r AttributeMappingRule) Validate() error {
	if strings.TrimSpace(r.TargetPath) == "" {
		return errors.New("provisioning: attribute mapping rule target_path is required")
	}
	if !r.SourceKind.Valid() {
		return errors.New("provisioning: invalid attribute mapping rule source_kind")
	}
	if !r.ApplyOn.Valid() {
		return errors.New("provisioning: invalid attribute mapping rule apply_on")
	}
	switch r.SourceKind {
	case SourceKindAttribute:
		if strings.TrimSpace(r.SourceKey) == "" {
			return errors.New("provisioning: attribute mapping rule requires source_key when source_kind=attribute")
		}
	case SourceKindConstant:
		if r.ConstantValue == nil {
			return errors.New("provisioning: attribute mapping rule requires constant_value when source_kind=constant")
		}
	}
	return nil
}

// MatchingRule is the 409-conflict correlation fallback used when externalId is
// unset on the downstream side (spec/contexts/provisioning.yaml models.MatchingRule).
type MatchingRule struct {
	ConflictMatchAttribute string `json:"conflict_match_attribute"`
}

// ProvisioningDeprovisionAction is how a deprovision trigger translates downstream
// (spec/contexts/provisioning.yaml models.ProvisioningDeprovisionAction).
type ProvisioningDeprovisionAction string

const (
	DeprovisionDeactivate ProvisioningDeprovisionAction = "deactivate"
	DeprovisionDelete     ProvisioningDeprovisionAction = "delete"
	DeprovisionNone       ProvisioningDeprovisionAction = "none"
)

func (a ProvisioningDeprovisionAction) Valid() bool {
	return a == DeprovisionDeactivate || a == DeprovisionDelete || a == DeprovisionNone
}

// DeprovisionPolicy translates internal deprovision triggers to downstream actions
// (spec/contexts/provisioning.yaml models.DeprovisionPolicy). user disable and group
// membership removal have no configurable field: they are fixed to deactivate and
// PATCH members remove respectively, enforced by the delivery engine rather than
// this policy.
type DeprovisionPolicy struct {
	OnUnassign                         ProvisioningDeprovisionAction `json:"on_unassign"`
	OnDelete                           ProvisioningDeprovisionAction `json:"on_delete"`
	OnGroupDeletedOrUnassigned         ProvisioningDeprovisionAction `json:"on_group_deleted_or_unassigned,omitempty"`
	GracePeriodDays                    int                           `json:"grace_period_days"`
	AccidentalDeletionCountThreshold   *int                          `json:"accidental_deletion_count_threshold,omitempty"`
	AccidentalDeletionPercentThreshold *int                          `json:"accidental_deletion_percent_threshold,omitempty"`
}

func (p DeprovisionPolicy) Validate() error {
	if !p.OnUnassign.Valid() {
		return errors.New("provisioning: invalid deprovision policy on_unassign")
	}
	if !p.OnDelete.Valid() {
		return errors.New("provisioning: invalid deprovision policy on_delete")
	}
	if p.OnGroupDeletedOrUnassigned != "" {
		if !p.OnGroupDeletedOrUnassigned.Valid() {
			return errors.New("provisioning: invalid deprovision policy on_group_deleted_or_unassigned")
		}
		if p.OnGroupDeletedOrUnassigned == DeprovisionDeactivate {
			return errors.New("provisioning: on_group_deleted_or_unassigned must not be deactivate (groups have no deactivate equivalent)")
		}
	}
	if p.GracePeriodDays < 0 {
		return errors.New("provisioning: grace_period_days must not be negative")
	}
	if p.AccidentalDeletionPercentThreshold != nil {
		if *p.AccidentalDeletionPercentThreshold < 1 || *p.AccidentalDeletionPercentThreshold > 100 {
			return errors.New("provisioning: accidental_deletion_percentage_threshold must be in [1, 100]")
		}
	}
	return nil
}
