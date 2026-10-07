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
  IdMagic.Contract.InsufficientScopeError,
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

@doc("テナントの最新の公開済みの認可モデルを読む。`version` を指定すると、そのリビジョンを読む。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("GetAuthorizationModel")
@route("/api/admin/v1/authorization/model")
@get
op GetAuthorizationModel(
  @query version?: int32
): GetAuthorizationModelSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetAuthorizationModelError403 | GetAuthorizationModelError404;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("テナントの認可モデルを新しいリビジョンとして公開する。既存のリビジョンには追記するだけで、書き換えない。未知のリレーションへの参照と、書き換え規則の循環は 422 で拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("PutAuthorizationModel")
@route("/api/admin/v1/authorization/model")
@post
op PutAuthorizationModel(
  @body body: PutAuthorizationModelHttpRequest
): PutAuthorizationModelSuccess_201 | PutAuthorizationModelError400 | IdMagic.Contract.AuthenticationRequiredResponse | PutAuthorizationModelError403 | PutAuthorizationModelError422;

model ListRelationTuplesHttpResponse {
  tuples: IdMagic.Contract.RelationTuple[];
  consistency: string;
}

model ListRelationTuplesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListRelationTuplesHttpResponse;
}

union ListRelationTuplesError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListRelationTuplesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListRelationTuplesError403Body;
}

@doc("テナントの RelationTuple を絞り込み条件で一覧する。どの絞り込み条件も任意で、レスポンスはページの上限で止まる。")
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
): ListRelationTuplesSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListRelationTuplesError403;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("テナントの RelationTuple を一つのトランザクションで追加および削除する。登録済みの認可モデルに合わない変更は 422 で拒否し、何も適用しない。レスポンスの一貫性トークンは、後の CheckAccess に渡せる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("WriteRelationTuples")
@route("/api/admin/v1/authorization/relation-tuples")
@post
op WriteRelationTuples(
  @body body: IdMagic.Contract.RelationTupleWriteRequest
): WriteRelationTuplesSuccess_200 | WriteRelationTuplesError400 | IdMagic.Contract.AuthenticationRequiredResponse | WriteRelationTuplesError403 | WriteRelationTuplesError422;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("サブジェクトがその委任の連鎖とともに、対象のリソースに対してリレーションを満たすかを判定する。診断と内部向けのもので、許可も拒否も 200 で返す。判定できないチェックも許可せず、拒否として返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("CheckAccess")
@route("/api/admin/v1/authorization/check")
@post
op CheckAccess(
  @body body: IdMagic.Contract.FgaCheckRequest
): CheckAccessSuccess_200 | CheckAccessError400 | IdMagic.Contract.AuthenticationRequiredResponse | CheckAccessError403 | CheckAccessError422;

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
  @doc("呼び出し元のアクセストークンが含むスコープ。操作が求めるものを満たさないとき、チェックはフェイルクローズで拒否する。")
  granted_scopes?: string[];
  @doc("操作が求めるスコープ。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("サブジェクトとその委任の連鎖が指定したリレーションを満たすリソースの識別子を、上限付きの走査で列挙する。上限に達すると `truncated` を設定する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ListAccessibleResources")
@route("/api/admin/v1/authorization/list-accessible-resources")
@post
op ListAccessibleResources(
  @body body: ListAccessibleResourcesHttpRequest
): ListAccessibleResourcesSuccess_200 | ListAccessibleResourcesError400 | IdMagic.Contract.AuthenticationRequiredResponse | ListAccessibleResourcesError403 | ListAccessibleResourcesError422;
}

}
