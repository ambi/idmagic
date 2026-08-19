namespace IdMagic.Contract {

@doc("The SCIM protocol error response of RFC 7644 section 3.12. SCIM is a contact point whose shape the standard fixes, so it does not return Problem Details; the body is `application/json`.")
model ScimProtocolError {
  @doc("Always the single element `urn:ietf:params:scim:api:messages:2.0:Error`.")
  schemas: string[];
  @doc("The HTTP status code, as a decimal string.")
  status: string;
  @doc("Human-readable explanation of the occurrence. Always English and never localized.")
  detail?: string;
  @doc("The SCIM detail error keyword of RFC 7644 section 3.12 table 9, such as `invalidFilter`, `invalidValue`, or `uniqueness`. Absent when no keyword applies.")
  scimType?: string;
}


@doc("Binds a SCIM 2.0 id to the internal User it stands for (the `sub`).")
model ScimUserRef {
  tenant_id: string;
  scim_id: string;
  user_id: string;
}


@doc("Binds a SCIM 2.0 id to the internal Group it stands for (the group id).")
model ScimGroupRef {
  tenant_id: string;
  scim_id: string;
  group_id: string;
}


@doc("A SCIM resource, sent to and returned by the /Users and /Groups endpoints (RFC 7643). The attributes present depend on which schemas the resource declares in `schemas`, so the contract records the resource as an open JSON object rather than fixing one attribute set; GetScimSchemas reports the attribute definitions the server honours. IdMagic supports the core User and Group schemas plus the enterprise User extension.")
model ScimResource is Record<unknown>;


@doc("The SCIM query result envelope of RFC 7644 section 3.4.2.")
model ScimListResponse {
  @doc("Always the single element `urn:ietf:params:scim:api:messages:2.0:ListResponse`.")
  schemas: string[];
  @doc("How many resources match the query in total, not how many this page carries.")
  totalResults: int32;
  @doc("1-based index of the first resource on this page.")
  startIndex: int32;
  @doc("How many resources this page carries.")
  itemsPerPage: int32;
  @doc("The page of matching resources. Capitalized, as RFC 7644 section 3.4.2 spells it.")
  Resources: ScimResource[];
}


@doc("Whether one optional SCIM feature is supported (RFC 7643 section 5).")
model ScimFeatureSupport {
  supported: boolean;
}


@doc("Bulk operation support, as ServiceProviderConfig declares it (RFC 7643 section 5).")
model ScimBulkConfig {
  supported: boolean;
  maxOperations: int32;
  maxPayloadSize: int32;
}


@doc("Filter support, as ServiceProviderConfig declares it (RFC 7643 section 5).")
model ScimFilterConfig {
  supported: boolean;
  @doc("The largest page the server returns, whatever `count` the query asks for.")
  maxResults: int32;
}


@doc("One authentication scheme the service provider accepts (RFC 7643 section 5).")
model ScimAuthenticationScheme {
  name: string;
  @doc("What the scheme is. Always English and never localized.")
  description: string;
  @doc("Where the scheme is specified.")
  specUri?: string;
  @doc("Scheme kind, such as `oauthbearertoken`.")
  type: string;
}


@doc("The SCIM service provider configuration of RFC 7643 section 5, reporting which optional features this deployment supports.")
model ScimServiceProviderConfig {
  @doc("Always the single element `urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig`.")
  schemas: string[];
  documentationUri?: string;
  patch: ScimFeatureSupport;
  bulk: ScimBulkConfig;
  filter: ScimFilterConfig;
  changePassword: ScimFeatureSupport;
  sort: ScimFeatureSupport;
  etag: ScimFeatureSupport;
  authenticationSchemes: ScimAuthenticationScheme[];
}


@doc("A schema a resource type extends beyond its core schema (RFC 7643 section 6), such as the enterprise User extension.")
model ScimSchemaExtension {
  @doc("URN of the extending schema.")
  schema: string;
  @doc("Whether a resource of this type must carry the extension.")
  required: boolean;
}


@doc("One resource type the service provider exposes (RFC 7643 section 6).")
model ScimResourceType {
  @doc("Always the single element `urn:ietf:params:scim:schemas:core:2.0:ResourceType`.")
  schemas: string[];
  id: string;
  name: string;
  @doc("Path the resource type is served under, relative to the SCIM base URL.")
  endpoint: string;
  @doc("What the resource type is. Always English and never localized.")
  description?: string;
  @doc("URN of the core schema for this resource type.")
  schema: string;
  schemaExtensions?: ScimSchemaExtension[];
}


@doc("One attribute definition inside a SCIM schema (RFC 7643 section 7).")
model ScimSchemaAttribute {
  name: string;
  @doc("Attribute type, such as `string`, `boolean`, `reference`, or `complex`.")
  type: string;
  multiValued: boolean;
  @doc("What the attribute holds. Always English and never localized.")
  description?: string;
  required: boolean;
  caseExact: boolean;
  @doc("One of `readOnly`, `readWrite`, `immutable`, or `writeOnly`.")
  mutability: string;
  @doc("One of `always`, `never`, `default`, or `request`.")
  returned: string;
  @doc("One of `none`, `server`, or `global`.")
  uniqueness: string;
  @doc("For `reference` attributes, which resource types the reference may point at.")
  referenceTypes?: string[];
  @doc("For `complex` attributes, the definitions of the sub-attributes.")
  subAttributes?: ScimSchemaAttribute[];
}


@doc("One schema the service provider honours (RFC 7643 section 7). IdMagic reports the core User schema, the enterprise User extension, and the core Group schema.")
model ScimSchema {
  @doc("Always the single element `urn:ietf:params:scim:schemas:core:2.0:Schema`.")
  schemas: string[];
  @doc("URN of the schema this document defines.")
  id: string;
  name: string;
  @doc("What the schema is for. Always English and never localized.")
  description?: string;
  attributes: ScimSchemaAttribute[];
}

}
