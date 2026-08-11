import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Sourcing")
namespace IdMagic.Sourcing {

namespace Operations {
model GetScimServiceProviderConfigHttpResponse {
  config: unknown;
}

model GetScimServiceProviderConfigSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetScimServiceProviderConfigHttpResponse;
}

union GetScimServiceProviderConfigError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model GetScimServiceProviderConfigError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: GetScimServiceProviderConfigError400Body;
}

@doc("SCIM 2.0 /ServiceProviderConfig を取得する。\nレスポンスは受け付ける認証方式を authenticationSchemes で申告する。同属性は\nRFC 7643 §5 で REQUIRED の多値属性であり、空や欠落は許されない。SCIM API は\nBearer スキームで提示するアクセストークンで認証するため、少なくとも\noauthbearertoken 方式を 1 件含める。")
@TypeSpec.OpenAPI.operationId("GetScimServiceProviderConfig")
@route("/scim/v2/ServiceProviderConfig")
@get
op GetScimServiceProviderConfig(
  @query tenant_id: string
): GetScimServiceProviderConfigSuccess_200 | GetScimServiceProviderConfigError400;

model GetScimResourceTypesHttpResponse {
  types: unknown[];
}

model GetScimResourceTypesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetScimResourceTypesHttpResponse;
}

union GetScimResourceTypesError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model GetScimResourceTypesError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: GetScimResourceTypesError400Body;
}

@doc("SCIM 2.0 /ResourceTypes を取得する。実装済みの User と Group の2 resource type\nだけを返す(schema URN、endpoint、schema extension は空)。")
@TypeSpec.OpenAPI.operationId("GetScimResourceTypes")
@route("/scim/v2/ResourceTypes")
@get
op GetScimResourceTypes(
  @query tenant_id: string
): GetScimResourceTypesSuccess_200 | GetScimResourceTypesError400;

model GetScimSchemasHttpResponse {
  schemas: unknown[];
}

model GetScimSchemasSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetScimSchemasHttpResponse;
}

union GetScimSchemasError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model GetScimSchemasError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: GetScimSchemasError400Body;
}

@doc("SCIM 2.0 /Schemas を取得する。RFC7643-CORE-RESOURCES が対応する User と Group の\n属性だけを、各属性の type、multiValued、required、mutability(readOnly/readWrite)、\nreturned、uniqueness とともに広告する。User emails は単一 canonical email への\nprojection、Group members の referenceTypes は User のみと明記し、未対応の\nphoneNumbers、addresses、Group member は広告しない。空の attribute 配列を返さない。")
@TypeSpec.OpenAPI.operationId("GetScimSchemas")
@route("/scim/v2/Schemas")
@get
op GetScimSchemas(
  @query tenant_id: string
): GetScimSchemasSuccess_200 | GetScimSchemasError400;

model CreateScimUserHttpRequest {
  tenant_id: string;
  body: unknown;
}

model CreateScimUserHttpResponse {
  response: unknown;
}

model CreateScimUserSuccess_200 {
  @statusCode statusCode: 200;
  @body body: CreateScimUserHttpResponse;
}

union CreateScimUserError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model CreateScimUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: CreateScimUserError400Body;
}

@doc("SCIM 2.0 /Users に対する POST 要求。内部 User を作成する。`userName` は必須で、\n欠落は invalidValue の ScimProtocolError にする。emails は全 element を検証後、\nprimary=true、type=work(case-insensitive)、wire order の順で単一 email へ投影する。\n不正 emails、phoneNumbers、addresses は invalidValue にする。`id` は常に server が\n生成し、body に含まれていても無視する(readOnly)。`active` 省略時は true。作成した\nresource は canonical email がある場合だけ単一 type=work、primary=true entry と、\n`id`、`meta.resourceType`、`meta.created`、`meta.lastModified`、`meta.location` を返す。\n同一 tenant 内で `userName` が既存 User と重複する場合は 409 `uniqueness` にする。")
@TypeSpec.OpenAPI.operationId("CreateScimUser")
@route("/scim/v2/Users")
@post
op CreateScimUser(
  @body body: CreateScimUserHttpRequest
): CreateScimUserSuccess_200 | CreateScimUserError400;

model ListScimUsersHttpResponse {
  response: unknown;
}

model ListScimUsersSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListScimUsersHttpResponse;
}

union ListScimUsersError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model ListScimUsersError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: ListScimUsersError400Body;
}

@doc("SCIM 2.0 /Users collection を filter、startIndex、count に従って検索する。\nfilter は RFC 7644 §3.4.2.2 grammar のうち、User の許可属性\n(userName, active, name.formatted, name.givenName, name.familyName,\nemails.value, id への eq, ne, co, sw, ew, pr。meta.created,\nmeta.lastModified への eq, ne, gt, ge, lt, le, pr。dateTime 比較は\nRFC3339 実時刻で行う) への比較演算子と論理演算子 (and, or, not, 丸括弧に\nよるグループ化) だけを受け付ける。属性名は schema URN プレフィックス\n(urn:ietf:params:scim:schemas:core:2.0:User: など) を付けた表記でも\n解決できる。許可外の属性・演算子・未知の URN プレフィックス・構文エラー・\n入力長やネスト深さの上限超過は invalidFilter の ScimProtocolError とし、\n内部エラーには落とさない。")
@TypeSpec.OpenAPI.operationId("ListScimUsers")
@route("/scim/v2/Users")
@get
op ListScimUsers(
  @query tenant_id: string,
  @query filter?: string,
  @query startIndex?: int64,
  @query count?: int64
): ListScimUsersSuccess_200 | ListScimUsersError400;

model GetScimUserHttpResponse {
  response: unknown;
}

model GetScimUserSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetScimUserHttpResponse;
}

union GetScimUserError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model GetScimUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: GetScimUserError400Body;
}

@doc("SCIM 2.0 /Users/{id} に対する GET 要求。canonical email がある場合だけ単一の\ntype=work、primary=true entry として返す。`id`、`meta.resourceType`、`meta.created`、\n`meta.lastModified`、`meta.location` を含む resource を返す。")
@TypeSpec.OpenAPI.operationId("GetScimUser")
@route("/scim/v2/Users/{id}")
@get
op GetScimUser(
  @query tenant_id: string,
  @path id: string
): GetScimUserSuccess_200 | GetScimUserError400;

model PatchScimUserHttpRequest {
  tenant_id: string;
  id: string;
  body: unknown;
}

model PatchScimUserHttpResponse {
  response: unknown;
}

model PatchScimUserSuccess_200 {
  @statusCode statusCode: 200;
  @body body: PatchScimUserHttpResponse;
}

union PatchScimUserError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model PatchScimUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: PatchScimUserError400Body;
}

@doc("SCIM 2.0 /Users/{id} に対する PATCH 要求(RFC 7644 §3.5.2)。`Operations` の\n各要素は `op`(add/replace/remove のいずれか)と `path` を持つ。`path` は\nRFC7644-PATCH が対応する User 属性(userName、name、name.formatted、\nname.givenName、name.familyName、emails、active)のいずれかでなければならず、\nphoneNumbers、addresses を含むそれ以外は invalidPath にする。emails の\nadd/replace は全 element を検証後、primary=true、type=work(case-insensitive)、\nwire order の順で単一 email へ投影し、不正値は invalidValue にする。`op` が\nadd/replace/remove 以外、または value の型が対象属性と不整合な場合も\ninvalidValue にする。`id`、`meta`、`schemas` への書込みは mutability エラーにする。")
@TypeSpec.OpenAPI.operationId("PatchScimUser")
@route("/scim/v2/Users/{id}")
@patch
op PatchScimUser(
  @path id: string,
  @body body: PatchScimUserHttpRequest
): PatchScimUserSuccess_200 | PatchScimUserError400;

model UpdateScimUserHttpRequest {
  tenant_id: string;
  id: string;
  body: unknown;
}

model UpdateScimUserHttpResponse {
  response: unknown;
}

model UpdateScimUserSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateScimUserHttpResponse;
}

union UpdateScimUserError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model UpdateScimUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: UpdateScimUserError400Body;
}

@doc("SCIM 2.0 /Users/{id} に対する PUT 要求。RFC7643-CORE-RESOURCES が対応する\nmutable 属性を**完全に置換**する(部分更新ではない)。`userName` は必須で、\n欠落は invalidValue の ScimProtocolError にする。省略された他の mutable\n属性(name.*、emails、active)は既定値にリセットする(name/emails は空、\nactive は true)。emails は全 element を検証後、primary=true、type=work\n(case-insensitive)、wire order の順で単一 email へ投影する。不正 emails、\nphoneNumbers、addresses は invalidValue にする。`id`、`meta` への書込みは\n無視する(readOnly)。")
@TypeSpec.OpenAPI.operationId("UpdateScimUser")
@route("/scim/v2/Users/{id}")
@put
op UpdateScimUser(
  @path id: string,
  @body body: UpdateScimUserHttpRequest
): UpdateScimUserSuccess_200 | UpdateScimUserError400;

model DeleteScimUserSuccess_200 {
  @statusCode statusCode: 200;
}

union DeleteScimUserError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model DeleteScimUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: DeleteScimUserError400Body;
}

@doc("SCIM 2.0 /Users/{id} に対する DELETE 要求。内部 User を soft-delete する。")
@TypeSpec.OpenAPI.operationId("DeleteScimUser")
@route("/scim/v2/Users/{id}")
@delete
op DeleteScimUser(
  @query tenant_id: string,
  @path id: string
): DeleteScimUserSuccess_200 | DeleteScimUserError400;

model CreateScimGroupHttpRequest {
  tenant_id: string;
  body: unknown;
}

model CreateScimGroupHttpResponse {
  response: unknown;
}

model CreateScimGroupSuccess_200 {
  @statusCode statusCode: 200;
  @body body: CreateScimGroupHttpResponse;
}

union CreateScimGroupError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model CreateScimGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: CreateScimGroupError400Body;
}

@doc("SCIM 2.0 /Groups に対する POST 要求。`displayName` は必須で、欠落は\ninvalidValue の ScimProtocolError にする。`id` は常に server が生成し、\nbody に含まれていても無視する(readOnly)。`members` の各要素は type が省略または\nUser(case-insensitive)で、同一 tenant 内で解決可能な User の SCIM id でなければ\nならない。type=Group または解決できない member は invalidValue にして group を\n作成しない。作成した resource の各 member は type=User とし、`id`、\n`meta.resourceType`、`meta.created`、`meta.lastModified`、`meta.location`\nを含めて返す。同一 tenant 内で `displayName` が既存 Group と重複する場合は\n409 `uniqueness` にする。")
@TypeSpec.OpenAPI.operationId("CreateScimGroup")
@route("/scim/v2/Groups")
@post
op CreateScimGroup(
  @body body: CreateScimGroupHttpRequest
): CreateScimGroupSuccess_200 | CreateScimGroupError400;

model ListScimGroupsHttpResponse {
  response: unknown;
}

model ListScimGroupsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListScimGroupsHttpResponse;
}

union ListScimGroupsError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model ListScimGroupsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: ListScimGroupsError400Body;
}

@doc("SCIM 2.0 /Groups collection を filter、startIndex、count に従って検索する。\nfilter は RFC 7644 §3.4.2.2 grammar のうち、Group の許可属性\n(displayName, id への eq, ne, co, sw, ew, pr。meta.created,\nmeta.lastModified への eq, ne, gt, ge, lt, le, pr。dateTime 比較は\nRFC3339 実時刻で行う) への比較演算子と論理演算子 (and, or, not, 丸括弧に\nよるグループ化) だけを受け付ける。属性名は schema URN プレフィックス\n(urn:ietf:params:scim:schemas:core:2.0:Group: など) を付けた表記でも\n解決できる。許可外の属性・演算子・未知の URN プレフィックス・構文エラー・\n入力長やネスト深さの上限超過は invalidFilter の ScimProtocolError とし、\n内部エラーには落とさない。")
@TypeSpec.OpenAPI.operationId("ListScimGroups")
@route("/scim/v2/Groups")
@get
op ListScimGroups(
  @query tenant_id: string,
  @query filter?: string,
  @query startIndex?: int64,
  @query count?: int64
): ListScimGroupsSuccess_200 | ListScimGroupsError400;

model GetScimGroupHttpResponse {
  response: unknown;
}

model GetScimGroupSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetScimGroupHttpResponse;
}

union GetScimGroupError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model GetScimGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: GetScimGroupError400Body;
}

@doc("SCIM 2.0 /Groups/{id} に対する GET 要求。各 member は type=User として返す。\n`id`、`meta.resourceType`、`meta.created`、`meta.lastModified`、`meta.location` を\n含む resource を返す。")
@TypeSpec.OpenAPI.operationId("GetScimGroup")
@route("/scim/v2/Groups/{id}")
@get
op GetScimGroup(
  @query tenant_id: string,
  @path id: string
): GetScimGroupSuccess_200 | GetScimGroupError400;

model PatchScimGroupHttpRequest {
  tenant_id: string;
  id: string;
  body: unknown;
}

model PatchScimGroupHttpResponse {
  response: unknown;
}

model PatchScimGroupSuccess_200 {
  @statusCode statusCode: 200;
  @body body: PatchScimGroupHttpResponse;
}

union PatchScimGroupError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model PatchScimGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: PatchScimGroupError400Body;
}

@doc("SCIM 2.0 /Groups/{id} に対する PATCH 要求(RFC 7644 §3.5.2)。`path` は\nRFC7644-PATCH が対応する Group 属性(displayName、members)のいずれかで\nなければならず、それ以外は invalidPath の ScimProtocolError にする。`op` が\nadd/replace/remove 以外、または value の型が対象属性と不整合な場合は\ninvalidValue にする。`members` は type が省略または User(case-insensitive)の member\nだけを許可する。type=Group または解決できない member を指定した場合も\ninvalidValue にし、group を変更しない。`id`、`meta`、`schemas` への書込みは\nmutability エラーにする。")
@TypeSpec.OpenAPI.operationId("PatchScimGroup")
@route("/scim/v2/Groups/{id}")
@patch
op PatchScimGroup(
  @path id: string,
  @body body: PatchScimGroupHttpRequest
): PatchScimGroupSuccess_200 | PatchScimGroupError400;

model UpdateScimGroupHttpRequest {
  tenant_id: string;
  id: string;
  body: unknown;
}

model UpdateScimGroupHttpResponse {
  response: unknown;
}

model UpdateScimGroupSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateScimGroupHttpResponse;
}

union UpdateScimGroupError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model UpdateScimGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: UpdateScimGroupError400Body;
}

@doc("SCIM 2.0 /Groups/{id} に対する PUT 要求。RFC7643-CORE-RESOURCES が対応する\nmutable 属性を**完全に置換**する(部分更新ではない)。`displayName` は必須で、\n欠落は invalidValue の ScimProtocolError にする。`members` を省略した場合は\n既存メンバーを全て削除する(空集合への置換)。`members` は type が省略または\nUser(case-insensitive)の member だけを許可する。type=Group または解決できない\nmember を指定した場合は invalidValue にし、group を変更しない。`id`、`meta` への\n書込みは無視する(readOnly)。")
@TypeSpec.OpenAPI.operationId("UpdateScimGroup")
@route("/scim/v2/Groups/{id}")
@put
op UpdateScimGroup(
  @path id: string,
  @body body: UpdateScimGroupHttpRequest
): UpdateScimGroupSuccess_200 | UpdateScimGroupError400;

model DeleteScimGroupSuccess_200 {
  @statusCode statusCode: 200;
}

union DeleteScimGroupError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model DeleteScimGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: DeleteScimGroupError400Body;
}

@doc("SCIM 2.0 /Groups/{id} に対する DELETE 要求。")
@TypeSpec.OpenAPI.operationId("DeleteScimGroup")
@route("/scim/v2/Groups/{id}")
@delete
op DeleteScimGroup(
  @query tenant_id: string,
  @path id: string
): DeleteScimGroupSuccess_200 | DeleteScimGroupError400;
}

}
