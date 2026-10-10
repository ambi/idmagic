package domain

import (
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// MergeTenantAttributeDefs merges builtin attribute definitions with a tenant's
// custom schema (if any). The result is the attribute_defs input IssueClaimsWithFloor
// uses to enforce the visibility floor. A nil schema yields builtin defs only.
func MergeTenantAttributeDefs(schema *userdomain.TenantUserAttributeSchema) []userdomain.UserAttributeDef {
	defs := userdomain.BuiltinUserAttributeDefs()
	if schema != nil {
		defs = append(defs, schema.Attributes...)
	}
	return defs
}
