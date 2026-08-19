namespace IdMagic.Contract {

@doc("The kinds of rewrite rule an authorization model may use for a relation. `direct` reads a RelationTuple as it stands, `computed_userset` reads another relation on the same object, and `tuple_to_userset` reads a relation on the object reached through `tupleset_relation`.")
enum RelationRewriteKind {
  direct: "direct",
  computed_userset: "computed_userset",
  tuple_to_userset: "tuple_to_userset",
}


@doc("One rewrite rule making up a relation. A relation holds when the union of the rules on its RelationDefinition holds. Intersection and exclusion are not defined.")
model RelationRewrite {
  kind: RelationRewriteKind;
  @doc("The relation name evaluated by `computed_userset` and `tuple_to_userset`. Not set for `direct`.")
  @maxLength(64)
  computed_relation?: string;
  @doc("The relation `tuple_to_userset` follows to the parent object. Not set for the other kinds.")
  @maxLength(64)
  tupleset_relation?: string;
  @doc("The subject shapes `direct` accepts: `user` for an individual subject, `group#member` for a subject set, and `user:*` for a wildcard. A RelationTuple whose shape is not listed here is refused both when written and when evaluated. Not set for the other kinds.")
  direct_subject_types?: string[];
}


@doc("The definition of one relation a ResourceTypeDefinition exposes. A relation with no rewrites can never hold.")
model RelationDefinition {
  @doc("The relation name, matching `^[a-z][a-z0-9_]*$` and unique within its ResourceTypeDefinition.")
  @minLength(1)
  @maxLength(64)
  name: string;
  rewrites: RelationRewrite[];
}


@doc("A resource type the authorization model defines. The `resource_type` and `subject_type` of a RelationTuple may only be type names declared here.")
model ResourceTypeDefinition {
  @doc("The type name, matching `^[a-z][a-z0-9_]*$` and unique within the AuthorizationModel.")
  @minLength(1)
  @maxLength(64)
  name: string;
  relations: RelationDefinition[];
}


@doc("The authorization model of a tenant. Revisions are only appended, and the latest published one decides every check. A revision referring to an unknown relation or type is refused fail-closed at registration.")
model AuthorizationModel {
  @doc("The generated identifier (UUID v4). Immutable.")
  id: string;
  @doc("The tenant it belongs to. Cross-tenant references are forbidden.")
  tenant_id: string = "default";
  @doc("The revision, rising monotonically within the tenant and starting at 1.")
  version: int32;
  resource_types: ResourceTypeDefinition[];
  created_at: utcDateTime;
}


@doc("The relation fact `(resource_type:resource_id, relation, subject)`. The triple is unique within the tenant, so writing the same one again is idempotent. A subject carrying `subject_relation` denotes a subject set (`group:eng#member`). A `subject_id` of `*` is a wildcard, accepted only when a `direct` rule permits that `subject_type`.")
model RelationTuple {
  @doc("The resource type the relation is held on. A type name the authorization model declared.")
  @minLength(1)
  @maxLength(64)
  resource_type: string;
  @doc("The resource identifier. IdMagic does not mint it; it holds the id from the caller's own resource space as it is.")
  @minLength(1)
  @maxLength(256)
  resource_id: string;
  @minLength(1)
  @maxLength(64)
  relation: string;
  @doc("The subject type. A type name the authorization model declared.")
  @minLength(1)
  @maxLength(64)
  subject_type: string;
  @doc("The subject identifier. `*` denotes a wildcard.")
  @minLength(1)
  @maxLength(256)
  subject_id: string;
  @doc("The relation name that makes this a subject set. Left out, the subject is an individual one.")
  @maxLength(64)
  subject_relation?: string;
}


@doc("The change WriteRelationTuples applies in one transaction. The same triple may not appear in both `writes` and `deletes`. An object listed in `delete_objects` has its RelationTuples removed on both the resource side and the subject side, so a deletion cascades.")
model RelationTupleWriteRequest {
  writes?: RelationTuple[];
  deletes?: RelationTuple[];
  delete_objects?: FgaObjectReference[];
}


@doc("An object reference written as `type:id`.")
model FgaObjectReference {
  @minLength(1)
  @maxLength(64)
  type: string;
  @minLength(1)
  @maxLength(256)
  id: string;
}


@doc("One step of the principals acting on behalf of the subject at check time. The RFC 8693 `act` chain, ordered outermost first.")
model FgaActor {
  @doc("A subject type the authorization model declares (`agent`, `user`, and so on). Checks and validity resolution use the same vocabulary.")
  @minLength(1)
  type: string;
  @minLength(1)
  id: string;
}


@doc("The input to CheckAccess. `subject` is the principal being acted for -- usually the `sub` of the access token -- and `actor_chain` is the sequence of agents acting on its behalf. Giving `minimum_consistency` makes the check confirm the store's write revision has reached at least that point before deciding.")
model FgaCheckRequest {
  @minLength(1)
  resource_type: string;
  @minLength(1)
  resource_id: string;
  @minLength(1)
  relation: string;
  @minLength(1)
  subject_type: string;
  @minLength(1)
  subject_id: string;
  actor_chain?: FgaActor[];
  @doc("The smallest consistency token this check needs. Pass the value WriteRelationTuples returned.")
  minimum_consistency?: string;
  @doc("The scopes the caller's access token carries. The check refuses fail-closed when they do not cover what the operation requires.")
  granted_scopes?: string[];
  @doc("The scopes the operation requires.")
  required_scopes?: string[];
}


@doc("The result of CheckAccess. When `permitted` is false, `reasons` names the rules that denied it. `relation_path` summarizes only the relation names that were followed; it carries no object or subject identifier.")
model FgaCheckResult {
  permitted: boolean;
  @doc("The revision of the authorization model the check used.")
  model_version: int32;
  @doc("The store's consistency token at the moment of the check.")
  consistency: string;
  relation_path: string[];
  reasons: string[];
}


@doc("The result of ListAccessibleResources. Reaching the scan ceiling sets `truncated`, and the caller must not treat the list as complete.")
model FgaListAccessibleResourcesResult {
  resource_ids: string[];
  truncated: boolean;
  model_version: int32;
  consistency: string;
}


@doc("Registering the authorization model was rejected by validation: a reference to an unknown type or relation, a name that breaks the naming rules, a cycle in the rewrite rules, or a missing required field. RFC 9457 Problem Details with `type` = `urn:idmagic:error:authorization_model_invalid`.")
model AuthorizationModelInvalidError is ProblemDetails;


@doc("The tenant has no published authorization model. RFC 9457 Problem Details with `type` = `urn:idmagic:error:authorization_model_not_found`.")
model AuthorizationModelNotFoundError is ProblemDetails;


@doc("A RelationTuple does not fit the registered authorization model: an unknown type or relation, a subject type that is not permitted, a wildcard that is not permitted, or the same tuple in both `writes` and `deletes`. RFC 9457 Problem Details with `type` = `urn:idmagic:error:relation_tuple_invalid`.")
model RelationTupleInvalidError is ProblemDetails;


@doc("The consistency token presented is newer than the write revision the store holds, or it belongs to another tenant. The check is fail-closed and rejects. RFC 9457 Problem Details with `type` = `urn:idmagic:error:consistency_not_satisfied`.")
model ConsistencyNotSatisfiedError is ProblemDetails;


@doc("An administrator published a new revision of the tenant's authorization model.")
model AuthorizationModelPublished {
  occurredAt: utcDateTime;
  tenantId: string;
  modelId: string;
  version: int32;
  resourceTypeCount: int32;
}


@doc("RelationTuples were written. The individual tuples are not copied into the audit trail; only the count and the consistency token are kept.")
model RelationTupleWritten {
  occurredAt: utcDateTime;
  tenantId: string;
  writtenCount: int32;
  consistency: string;
}


@doc("RelationTuples were deleted. A cascading per-object deletion is recorded through this event as well.")
model RelationTupleDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  deletedCount: int32;
  consistency: string;
}


@doc("CheckAccess reached a decision. The resource identifier is reduced to the first 16 hex digits of its SHA-256, and neither the subject identifier nor the tuple contents are kept.")
model FgaCheckEvaluated {
  occurredAt: utcDateTime;
  tenantId: string;
  resourceType: string;
  resourceIdDigest: string;
  relation: string;
  permitted: boolean;
  modelVersion: int32;
  @doc("A summary of the path followed, relation names only.")
  relationPath: string[];
  @doc("The rules that denied it. Empty when `permitted` is true.")
  reasons: string[];
  @doc("How many steps the delegation chain had. Zero means direct access.")
  actorChainDepth: int32;
}


@doc("ListAccessibleResources finished its scan. The per-candidate decisions are not expanded into the audit trail; this single event keeps only the counts and whether the scan was cut short. No resource identifier is kept.")
model FgaResourcesEnumerated {
  occurredAt: utcDateTime;
  tenantId: string;
  resourceType: string;
  relation: string;
  @doc("How many candidates were scanned.")
  candidateCount: int32;
  @doc("How many were permitted.")
  permittedCount: int32;
  @doc("Whether the scan hit its ceiling and was cut short.")
  truncated: boolean;
  modelVersion: int32;
  actorChainDepth: int32;
}

}
