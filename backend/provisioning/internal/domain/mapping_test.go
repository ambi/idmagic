package domain

import "testing"

func TestAttributeMappingRule_Validate_AttributeSourceRequiresSourceKey(t *testing.T) {
	rule := AttributeMappingRule{TargetPath: "userName", SourceKind: SourceKindAttribute, ApplyOn: ApplyCreateAndUpdate}
	if err := rule.Validate(); err == nil {
		t.Error("Validate() should reject source_kind=attribute without source_key")
	}
	rule.SourceKey = "preferred_username"
	if err := rule.Validate(); err != nil {
		t.Errorf("Validate() with source_key set returned error: %v", err)
	}
}

func TestAttributeMappingRule_Validate_ConstantSourceRequiresConstantValue(t *testing.T) {
	rule := AttributeMappingRule{TargetPath: "active", SourceKind: SourceKindConstant, ApplyOn: ApplyCreateAndUpdate}
	if err := rule.Validate(); err == nil {
		t.Error("Validate() should reject source_kind=constant without constant_value")
	}
	rule.ConstantValue = true
	if err := rule.Validate(); err != nil {
		t.Errorf("Validate() with constant_value set returned error: %v", err)
	}
}

func TestAttributeMappingRule_Validate_RejectsEmptyTargetPathOrInvalidEnums(t *testing.T) {
	base := AttributeMappingRule{TargetPath: "userName", SourceKind: SourceKindAttribute, SourceKey: "preferred_username", ApplyOn: ApplyCreateAndUpdate}
	if err := base.Validate(); err != nil {
		t.Fatalf("baseline rule should validate, got: %v", err)
	}
	noTarget := base
	noTarget.TargetPath = ""
	if err := noTarget.Validate(); err == nil {
		t.Error("Validate() should reject empty target_path")
	}
	badSourceKind := base
	badSourceKind.SourceKind = "bogus"
	if err := badSourceKind.Validate(); err == nil {
		t.Error("Validate() should reject invalid source_kind")
	}
	badApplyOn := base
	badApplyOn.ApplyOn = "bogus"
	if err := badApplyOn.Validate(); err == nil {
		t.Error("Validate() should reject invalid apply_on")
	}
}

func TestDeprovisionPolicy_Validate_RejectsDeactivateForGroupDeletedOrUnassigned(t *testing.T) {
	p := DeprovisionPolicy{OnUnassign: DeprovisionDeactivate, OnDelete: DeprovisionDeactivate, OnGroupDeletedOrUnassigned: DeprovisionDeactivate}
	if err := p.Validate(); err == nil {
		t.Error("Validate() should reject on_group_deleted_or_unassigned=deactivate (groups have no deactivate equivalent)")
	}
	p.OnGroupDeletedOrUnassigned = DeprovisionDelete
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() with on_group_deleted_or_unassigned=delete returned error: %v", err)
	}
	p.OnGroupDeletedOrUnassigned = DeprovisionNone
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() with on_group_deleted_or_unassigned=none returned error: %v", err)
	}
}

func TestDeprovisionPolicy_Validate_RejectsNegativeGracePeriod(t *testing.T) {
	p := DeprovisionPolicy{OnUnassign: DeprovisionDeactivate, OnDelete: DeprovisionDeactivate, GracePeriodDays: -1}
	if err := p.Validate(); err == nil {
		t.Error("Validate() should reject negative grace_period_days")
	}
}

func TestDeprovisionPolicy_Validate_RejectsInvalidActions(t *testing.T) {
	p := DeprovisionPolicy{OnUnassign: "bogus", OnDelete: DeprovisionDeactivate}
	if err := p.Validate(); err == nil {
		t.Error("Validate() should reject invalid on_unassign action")
	}
}

func TestProvisioningDeprovisionAction_Valid(t *testing.T) {
	for _, a := range []ProvisioningDeprovisionAction{DeprovisionDeactivate, DeprovisionDelete, DeprovisionNone} {
		if !a.Valid() {
			t.Errorf("ProvisioningDeprovisionAction(%q).Valid() = false, want true", a)
		}
	}
	if ProvisioningDeprovisionAction("bogus").Valid() {
		t.Error(`ProvisioningDeprovisionAction("bogus").Valid() = true, want false`)
	}
}

func TestAttributeSourceKind_Valid(t *testing.T) {
	for _, k := range []AttributeSourceKind{SourceKindAttribute, SourceKindConstant} {
		if !k.Valid() {
			t.Errorf("AttributeSourceKind(%q).Valid() = false, want true", k)
		}
	}
	if AttributeSourceKind("bogus").Valid() {
		t.Error(`AttributeSourceKind("bogus").Valid() = true, want false`)
	}
}

func TestAttributeApplyOn_Valid(t *testing.T) {
	for _, a := range []AttributeApplyOn{ApplyCreateAndUpdate, ApplyCreateOnly} {
		if !a.Valid() {
			t.Errorf("AttributeApplyOn(%q).Valid() = false, want true", a)
		}
	}
	if AttributeApplyOn("bogus").Valid() {
		t.Error(`AttributeApplyOn("bogus").Valid() = true, want false`)
	}
}
