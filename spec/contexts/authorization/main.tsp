import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Authorization")
namespace IdMagic.Authorization {

namespace Operations {
model GetAuthorizationModelHttpResponse {
  authorization_model: IdMagic.Contract.AuthorizationModel;
  consistency: string;
}

model GetAuthorizationModelSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAuthorizationModelHttpResponse;
}

union GetAuthorizationModelError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAuthorizationModelError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAuthorizationModelError403Body;
}

union GetAuthorizationModelError404Body {
  IdMagic.Contract.AuthorizationModelNotFoundError,
}

model GetAuthorizationModelError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAuthorizationModelError404Body;
}

@doc("Reads the latest published authorization model of the tenant. Naming a `version` reads that revision instead.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("GetAuthorizationModel")
@route("/api/admin/v1/authorization/model")
@get
op GetAuthorizationModel(
  @query version?: int32
): GetAuthorizationModelSuccess_200 | GetAuthorizationModelError403 | GetAuthorizationModelError404;

model PutAuthorizationModelHttpRequest {
  resource_types: IdMagic.Contract.ResourceTypeDefinition[];
}

model PutAuthorizationModelHttpResponse {
  authorization_model: IdMagic.Contract.AuthorizationModel;
  consistency: string;
}

model PutAuthorizationModelSuccess_201 {
  @statusCode statusCode: 201;
  @body body: PutAuthorizationModelHttpResponse;
}

union PutAuthorizationModelError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model PutAuthorizationModelError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: PutAuthorizationModelError400Body;
}

union PutAuthorizationModelError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model PutAuthorizationModelError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: PutAuthorizationModelError403Body;
}

union PutAuthorizationModelError422Body {
  IdMagic.Contract.AuthorizationModelInvalidError,
}

model PutAuthorizationModelError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: PutAuthorizationModelError422Body;
}

@doc("Publishes the authorization model of the tenant as a new revision. Existing revisions are appended to, never rewritten. A reference to an unknown relation, or a cycle in the rewrite rules, is rejected with 422.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("PutAuthorizationModel")
@route("/api/admin/v1/authorization/model")
@post
op PutAuthorizationModel(
  @body body: PutAuthorizationModelHttpRequest
): PutAuthorizationModelSuccess_201 | PutAuthorizationModelError400 | PutAuthorizationModelError403 | PutAuthorizationModelError422;

model ListRelationTuplesHttpResponse {
  tuples: IdMagic.Contract.RelationTuple[];
  consistency: string;
}

model ListRelationTuplesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListRelationTuplesHttpResponse;
}

union ListRelationTuplesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListRelationTuplesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListRelationTuplesError403Body;
}

@doc("Lists the RelationTuples of the tenant under a filter. Every filter is optional, and the response stops at the page ceiling.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ListRelationTuples")
@route("/api/admin/v1/authorization/relation-tuples")
@get
op ListRelationTuples(
  @query resource_type?: string,
  @query resource_id?: string,
  @query relation?: string,
  @query subject_type?: string,
  @query subject_id?: string,
  @query limit?: int32
): ListRelationTuplesSuccess_200 | ListRelationTuplesError403;

model WriteRelationTuplesHttpResponse {
  written_count: int32;
  deleted_count: int32;
  consistency: string;
}

model WriteRelationTuplesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: WriteRelationTuplesHttpResponse;
}

union WriteRelationTuplesError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model WriteRelationTuplesError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: WriteRelationTuplesError400Body;
}

union WriteRelationTuplesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model WriteRelationTuplesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: WriteRelationTuplesError403Body;
}

union WriteRelationTuplesError422Body {
  IdMagic.Contract.RelationTupleInvalidError,
  IdMagic.Contract.AuthorizationModelNotFoundError,
}

model WriteRelationTuplesError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: WriteRelationTuplesError422Body;
}

@doc("Adds and removes RelationTuples of the tenant in one transaction. A change that does not fit the registered authorization model is rejected with 422, and nothing is applied. The consistency token in the response can be handed to a later CheckAccess.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("WriteRelationTuples")
@route("/api/admin/v1/authorization/relation-tuples")
@post
op WriteRelationTuples(
  @body body: IdMagic.Contract.RelationTupleWriteRequest
): WriteRelationTuplesSuccess_200 | WriteRelationTuplesError400 | WriteRelationTuplesError403 | WriteRelationTuplesError422;

model CheckAccessHttpResponse {
  result: IdMagic.Contract.FgaCheckResult;
}

model CheckAccessSuccess_200 {
  @statusCode statusCode: 200;
  @body body: CheckAccessHttpResponse;
}

union CheckAccessError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CheckAccessError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CheckAccessError400Body;
}

union CheckAccessError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CheckAccessError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CheckAccessError403Body;
}

union CheckAccessError422Body {
  IdMagic.Contract.ConsistencyNotSatisfiedError,
  IdMagic.Contract.AuthorizationModelNotFoundError,
}

model CheckAccessError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: CheckAccessError422Body;
}

@doc("Decides whether a subject, together with its delegation chain, holds a relation on the target resource. Diagnostic and internal: both allow and deny come back as 200. An undecidable check is not allowed either -- it comes back as a deny.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("CheckAccess")
@route("/api/admin/v1/authorization/check")
@post
op CheckAccess(
  @body body: IdMagic.Contract.FgaCheckRequest
): CheckAccessSuccess_200 | CheckAccessError400 | CheckAccessError403 | CheckAccessError422;

model ListAccessibleResourcesHttpRequest {
  @minLength(1)
  resource_type: string;
  @minLength(1)
  relation: string;
  @minLength(1)
  subject_type: string;
  @minLength(1)
  subject_id: string;
  actor_chain?: IdMagic.Contract.FgaActor[];
  minimum_consistency?: string;
  @doc("The scopes the caller's access token carries. The check refuses fail-closed when they do not cover what the operation requires.")
  granted_scopes?: string[];
  @doc("The scopes the operation requires.")
  required_scopes?: string[];
}

model ListAccessibleResourcesHttpResponse {
  result: IdMagic.Contract.FgaListAccessibleResourcesResult;
}

model ListAccessibleResourcesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListAccessibleResourcesHttpResponse;
}

union ListAccessibleResourcesError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListAccessibleResourcesError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListAccessibleResourcesError400Body;
}

union ListAccessibleResourcesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAccessibleResourcesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAccessibleResourcesError403Body;
}

union ListAccessibleResourcesError422Body {
  IdMagic.Contract.ConsistencyNotSatisfiedError,
  IdMagic.Contract.AuthorizationModelNotFoundError,
}

model ListAccessibleResourcesError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: ListAccessibleResourcesError422Body;
}

@doc("Enumerates the resource identifiers on which a subject and its delegation chain hold the given relation, under a bounded scan. Reaching the bound sets `truncated`.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ListAccessibleResources")
@route("/api/admin/v1/authorization/list-accessible-resources")
@post
op ListAccessibleResources(
  @body body: ListAccessibleResourcesHttpRequest
): ListAccessibleResourcesSuccess_200 | ListAccessibleResourcesError400 | ListAccessibleResourcesError403 | ListAccessibleResourcesError422;
}

}
