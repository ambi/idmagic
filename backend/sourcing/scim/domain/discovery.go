package domain

// UserCoreSchema describes the RFC7643-CORE-RESOURCES adoption:partial
// User attribute subset this server implements, for GetScimSchemas.
func UserCoreSchema() Schema {
	return Schema{
		Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Schema"},
		ID:          "urn:ietf:params:scim:schemas:core:2.0:User",
		Name:        "User",
		Description: "User Account",
		Attributes: []SchemaAttribute{
			{Name: "id", Type: "string", Mutability: "readOnly", Returned: "always", Uniqueness: "server"},
			{Name: "userName", Type: "string", Required: true, CaseExact: false, Mutability: "readWrite", Returned: "default", Uniqueness: "server"},
			{
				Name: "name", Type: "complex", Mutability: "readWrite", Returned: "default",
				SubAttributes: []SchemaAttribute{
					{Name: "formatted", Type: "string", Mutability: "readWrite", Returned: "default"},
					{Name: "givenName", Type: "string", Mutability: "readWrite", Returned: "default"},
					{Name: "familyName", Type: "string", Mutability: "readWrite", Returned: "default"},
				},
			},
			{
				Name: "emails", Type: "complex", MultiValued: true, Mutability: "readWrite", Returned: "default",
				Description: "Projected to one canonical email by primary, work, then wire order; responses contain one work/primary entry (RFC7643-CORE-RESOURCES adoption:partial).",
				SubAttributes: []SchemaAttribute{
					{Name: "value", Type: "string", Mutability: "readWrite", Returned: "default"},
					{Name: "type", Type: "string", Mutability: "readWrite", Returned: "default"},
					{Name: "primary", Type: "boolean", Mutability: "readWrite", Returned: "default"},
				},
			},
			{Name: "active", Type: "boolean", Mutability: "readWrite", Returned: "default"},
		},
	}
}

// GroupCoreSchema describes the RFC7643-CORE-RESOURCES adoption:partial
// Group attribute subset this server implements, for GetScimSchemas.
func GroupCoreSchema() Schema {
	return Schema{
		Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Schema"},
		ID:          "urn:ietf:params:scim:schemas:core:2.0:Group",
		Name:        "Group",
		Description: "Group",
		Attributes: []SchemaAttribute{
			{Name: "id", Type: "string", Mutability: "readOnly", Returned: "always", Uniqueness: "server"},
			{Name: "displayName", Type: "string", Required: true, Mutability: "readWrite", Returned: "default", Uniqueness: "server"},
			{
				Name: "members", Type: "complex", MultiValued: true, Mutability: "readWrite", Returned: "default",
				Description: "Only User members are supported. Nested group members (type=Group) are not supported (RFC7643-CORE-RESOURCES adoption:partial).",
				SubAttributes: []SchemaAttribute{
					{Name: "value", Type: "string", Mutability: "immutable", Returned: "default"},
					{Name: "$ref", Type: "reference", Mutability: "immutable", Returned: "default", ReferenceTypes: []string{"User"}},
					{Name: "display", Type: "string", Mutability: "readOnly", Returned: "default"},
					{Name: "type", Type: "string", Mutability: "immutable", Returned: "default"},
				},
			},
		},
	}
}
