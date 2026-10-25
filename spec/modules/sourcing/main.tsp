import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Sourcing")
namespace IdMagic.Sourcing {

namespace Operations {
model GetScimServiceProviderConfigSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimServiceProviderConfig;
}

union GetScimServiceProviderConfigError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model GetScimServiceProviderConfigError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: GetScimServiceProviderConfigError400Body;
}

@doc("SCIM 2.0 の /ServiceProviderConfig を読む。レスポンスは、受け付ける認証方式を `authenticationSchemes` で宣言する。RFC 7643 の 5 節はこの属性を REQUIRED の複数値の属性と定めているので、空にも省略にもできない。SCIM API は Bearer 方式で提示したアクセストークンで認証するので、少なくとも `oauthbearertoken` の方式を載せる。")
@TypeSpec.OpenAPI.operationId("GetScimServiceProviderConfig")
@route("/scim/v2/ServiceProviderConfig")
@get
op GetScimServiceProviderConfig(
  @query tenant_id: string
): GetScimServiceProviderConfigSuccess_200 | GetScimServiceProviderConfigError400;

model GetScimResourceTypesSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimResourceType[];
}

union GetScimResourceTypesError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model GetScimResourceTypesError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: GetScimResourceTypesError400Body;
}

@doc("SCIM 2.0 の /ResourceTypes を読む。実装済みの二つのリソースタイプ、User と Group だけを、スキーマの URN とエンドポイントとともに返す。User は、対応する enterprise 拡張（urn:ietf:params:scim:schemas:extension:enterprise:2.0:User）を `schemaExtensions` で示す。")
@TypeSpec.OpenAPI.operationId("GetScimResourceTypes")
@route("/scim/v2/ResourceTypes")
@get
op GetScimResourceTypes(
  @query tenant_id: string
): GetScimResourceTypesSuccess_200 | GetScimResourceTypesError400;

model GetScimSchemasSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimSchema[];
}

union GetScimSchemasError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model GetScimSchemasError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: GetScimSchemasError400Body;
}

@doc("SCIM 2.0 の /Schemas を読む。RFC7643-CORE-RESOURCES が扱う User と Group の属性と、RFC7643-ENTERPRISE-EXTENSION が扱う enterprise 拡張（urn:ietf:params:scim:schemas:extension:enterprise:2.0:User）の `employeeNumber`、`department`、`manager` を、それぞれの `type`、`multiValued`、`required`、`mutability`（readOnly または readWrite）、`returned`、`uniqueness` とともに示す。User の emails が単一の正規のメールアドレスへ射影されることと、Group の members の `referenceTypes` が User だけであることを示し、対応していないもの（phoneNumbers、addresses、メンバーとしての Group、costCenter、division、organization）は示さない。空の属性の配列は返さない。")
@TypeSpec.OpenAPI.operationId("GetScimSchemas")
@route("/scim/v2/Schemas")
@get
op GetScimSchemas(
  @query tenant_id: string
): GetScimSchemasSuccess_200 | GetScimSchemasError400;

union CreateScimUserError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model CreateScimUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: CreateScimUserError400Body;
}

model ScimQuotaExceededResponse {
  @statusCode statusCode: 422;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimProtocolError;
}

model CreateScimUserSuccess_201 {
  @statusCode statusCode: 201;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimResource;
}

@doc("SCIM 2.0 の /Users への POST で、内部の User を作成する。`userName` は必須で、ないときは `invalidValue` の ScimProtocolError になる。`emails` の各要素を検証してから、primary=true、type=work（大文字と小文字を区別しない）、送信された順の優先順位で単一のメールアドレスへ射影する。形式の誤った `emails`、`phoneNumbers`、`addresses` は `invalidValue` になる。urn:ietf:params:scim:schemas:extension:enterprise:2.0:User の下の `employeeNumber`、`department`、`manager`（オブジェクトの `value` または単純な文字列）にも対応する。文字列でない `employeeNumber` または `department` と、同じテナントに存在しない SCIM の User を指す `manager` は、どちらも `invalidValue` になる。`id` は常にサーバーが生成し、ボディに含まれていても無視する（readOnly）。`active` のデフォルトは true である。作成したリソースは、正規のメールアドレスがあるときだけ type=work、primary=true の単一の要素を含み、拡張属性を一つ以上設定したときだけ `schemas` に enterprise 拡張の URN と対応する属性のオブジェクトを含み、常に `id`、`meta.resourceType`、`meta.created`、`meta.lastModified`、`meta.location` を返す。同じテナントの既存の User と衝突する `userName` は 409 の `uniqueness` になる。")
@TypeSpec.OpenAPI.operationId("CreateScimUser")
@route("/scim/v2/Users")
@post
op CreateScimUser(
  @header contentType: "application/scim+json",
  @body body: IdMagic.Contract.ScimResource
): CreateScimUserSuccess_201 | CreateScimUserError400 | ScimQuotaExceededResponse;

model ListScimUsersSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimListResponse;
}

union ListScimUsersError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model ListScimUsersError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: ListScimUsersError400Body;
}

@doc("`filter`、`startIndex`、`count` で SCIM 2.0 の /Users のコレクションを検索する。RFC 7644 の 3.4.2.2 節の文法のうち、`filter` が受け付けるのは、許可した User の属性に対する比較演算子（userName、active、name.formatted、name.givenName、name.familyName、emails.value、id に対する eq、ne、co、sw、ew、pr と、meta.created と meta.lastModified に対する eq、ne、gt、ge、lt、le、pr。dateTime の比較は実際の RFC 3339 の時点で行う）と、論理演算子の and、or、not と、グループ化の括弧だけである。属性名はスキーマの URN の接頭辞を付けて書いてもよい（urn:ietf:params:scim:schemas:core:2.0:User: など）。許可リストにない属性または演算子、未知の URN の接頭辞、構文の誤り、上限より長いまたは深く入れ子になった入力は、いずれも `invalidFilter` の ScimProtocolError になり、内部エラーには落ちない。")
@TypeSpec.OpenAPI.operationId("ListScimUsers")
@route("/scim/v2/Users")
@get
op ListScimUsers(
  @query tenant_id: string,
  @query filter?: string,
  @query startIndex?: int64,
  @query count?: int64
): ListScimUsersSuccess_200 | ListScimUsersError400;

model GetScimUserSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimResource;
}

union GetScimUserError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model GetScimUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: GetScimUserError400Body;
}

union GetScimUserError404Body {
  IdMagic.Contract.ScimProtocolError,
}

model GetScimUserError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/scim+json";
  @body body: GetScimUserError404Body;
}

@doc("SCIM 2.0 の /Users/{id} への GET。正規のメールアドレスは、あるときだけ type=work、primary=true の単一の要素として返す。リソースは `id`、`meta.resourceType`、`meta.created`、`meta.lastModified`、`meta.location` を含む。DeleteScimUser で削除した id は、一度も存在しなかった id と同じく 404 になる（RFC7644-DELETE-SEMANTICS）。")
@TypeSpec.OpenAPI.operationId("GetScimUser")
@route("/scim/v2/Users/{id}")
@get
op GetScimUser(
  @query tenant_id: string,
  @path id: string
): GetScimUserSuccess_200 | GetScimUserError400 | GetScimUserError404;

model PatchScimUserSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimResource;
}

union PatchScimUserError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model PatchScimUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: PatchScimUserError400Body;
}

union PatchScimUserError404Body {
  IdMagic.Contract.ScimProtocolError,
}

model PatchScimUserError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/scim+json";
  @body body: PatchScimUserError404Body;
}

@doc("SCIM 2.0 の /Users/{id} への PATCH（RFC 7644 の 3.5.2 節）。`Operations` の各要素は、add、replace、remove のいずれかの `op` と `path` を含む。`path` は RFC7644-PATCH が扱う User の属性（userName、name、name.formatted、name.givenName、name.familyName、emails、active、employeeNumber、department、manager）のいずれかでなければならず、enterprise 拡張の属性は、そのままの名前でも urn:ietf:params:scim:schemas:extension:enterprise:2.0:User で完全修飾した名前でも書ける。phoneNumbers と addresses を含め、それ以外は `invalidPath` になる。`emails` への add または replace は各要素を検証してから、primary=true、type=work（大文字と小文字を区別しない）、送信された順の優先順位で単一のメールアドレスへ射影し、形式の誤った値は `invalidValue` になる。`manager` への add または replace は値のオブジェクトまたは単純な文字列を受け付け、同じテナントに存在しない SCIM の User を指すと `invalidValue` になる。add、replace、remove 以外の `op` と、対象の属性に合わない型の値も `invalidValue` になる。`id`、`meta`、`schemas` への書き込みは mutability のエラーになる。このテナントの User へ解決しない id は、削除済みのものを含めて 404 になる（RFC7644-DELETE-SEMANTICS）。")
@TypeSpec.OpenAPI.operationId("PatchScimUser")
@route("/scim/v2/Users/{id}")
@patch
op PatchScimUser(
  @path id: string,
  @header contentType: "application/scim+json",
  @body body: IdMagic.Contract.ScimResource
): PatchScimUserSuccess_200 | PatchScimUserError400 | PatchScimUserError404;

model UpdateScimUserSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimResource;
}

union UpdateScimUserError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model UpdateScimUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: UpdateScimUserError400Body;
}

union UpdateScimUserError404Body {
  IdMagic.Contract.ScimProtocolError,
}

model UpdateScimUserError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/scim+json";
  @body body: UpdateScimUserError404Body;
}

@doc("SCIM 2.0 の /Users/{id} への PUT。RFC7643-CORE-RESOURCES が扱う変更可能な属性を丸ごと**置き換える**もので、部分更新ではない。`userName` は必須で、ないときは `invalidValue` の ScimProtocolError になる。それ以外の変更可能な属性（name.*、emails、active、enterprise 拡張の employeeNumber、department、manager）を省くと、デフォルトへ戻す。name、emails、拡張属性は空、active は true である。`emails` の各要素を検証してから、primary=true、type=work（大文字と小文字を区別しない）、送信された順の優先順位で単一のメールアドレスへ射影する。形式の誤った `emails`、`phoneNumbers`、`addresses` は `invalidValue` になり、urn:ietf:params:scim:schemas:extension:enterprise:2.0:User の下の、文字列でない `employeeNumber` または `department` と、同じテナントに存在しない SCIM の User を指す `manager` も同じである。`id` と `meta` への書き込みは無視する（readOnly）。このテナントの User へ解決しない id は、削除済みのものを含めて 404 になる（RFC7644-DELETE-SEMANTICS）。")
@TypeSpec.OpenAPI.operationId("UpdateScimUser")
@route("/scim/v2/Users/{id}")
@put
op UpdateScimUser(
  @path id: string,
  @header contentType: "application/scim+json",
  @body body: IdMagic.Contract.ScimResource
): UpdateScimUserSuccess_200 | UpdateScimUserError400 | UpdateScimUserError404;

union DeleteScimUserError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model DeleteScimUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: DeleteScimUserError400Body;
}

model DeleteScimUserSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteScimUserError404Body {
  IdMagic.Contract.ScimProtocolError,
}

model DeleteScimUserError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/scim+json";
  @body body: DeleteScimUserError404Body;
}

@doc("SCIM 2.0 の /Users/{id} への DELETE。内部の User を論理削除する。記録は残し、ライフサイクルの状態を pending_deletion にする。SCIM の側から見ると、その時点で User は消える。同じ id への以降の GET、PUT、PATCH、DELETE はすべて 404 になり、ListScimUsers は `Resources` からも `totalResults` からもこれを除き、どの Group の `members` にも、他の User の enterprise 拡張の `manager` にも現れなくなる（RFC7644-DELETE-SEMANTICS）。削除した User の `userName` は一意性の検査に残るので、同じ `userName` で作り直すと 409 の `uniqueness` になる。")
@TypeSpec.OpenAPI.operationId("DeleteScimUser")
@route("/scim/v2/Users/{id}")
@delete
op DeleteScimUser(
  @query tenant_id: string,
  @path id: string
): DeleteScimUserSuccess_204 | DeleteScimUserError400 | DeleteScimUserError404;

union CreateScimGroupError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model CreateScimGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: CreateScimGroupError400Body;
}

model CreateScimGroupSuccess_201 {
  @statusCode statusCode: 201;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimResource;
}

@doc("SCIM 2.0 の /Groups への POST。`displayName` は必須で、ないときは `invalidValue` の ScimProtocolError になる。`id` は常にサーバーが生成し、ボディに含まれていても無視する（readOnly）。`members` の各要素は、`type` が省略または User（大文字と小文字を区別しない）であり、同じテナントで解決する User の SCIM の id でなければならない。Group の種類のメンバーと、解決しないメンバーは、どちらも `invalidValue` になり、グループは作成しない。作成したリソースの各メンバーは type=User を含み、リソースは `id`、`meta.resourceType`、`meta.created`、`meta.lastModified`、`meta.location` を含む。同じテナントの既存の Group と衝突する `displayName` は 409 の `uniqueness` になる。")
@TypeSpec.OpenAPI.operationId("CreateScimGroup")
@route("/scim/v2/Groups")
@post
op CreateScimGroup(
  @header contentType: "application/scim+json",
  @body body: IdMagic.Contract.ScimResource
): CreateScimGroupSuccess_201 | CreateScimGroupError400 | ScimQuotaExceededResponse;

model ListScimGroupsSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimListResponse;
}

union ListScimGroupsError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model ListScimGroupsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: ListScimGroupsError400Body;
}

@doc("`filter`、`startIndex`、`count` で SCIM 2.0 の /Groups のコレクションを検索する。RFC 7644 の 3.4.2.2 節の文法のうち、`filter` が受け付けるのは、許可した Group の属性に対する比較演算子（displayName と id に対する eq、ne、co、sw、ew、pr と、meta.created と meta.lastModified に対する eq、ne、gt、ge、lt、le、pr。dateTime の比較は実際の RFC 3339 の時点で行う）と、論理演算子の and、or、not と、グループ化の括弧だけである。属性名はスキーマの URN の接頭辞を付けて書いてもよい（urn:ietf:params:scim:schemas:core:2.0:Group: など）。許可リストにない属性または演算子、未知の URN の接頭辞、構文の誤り、上限より長いまたは深く入れ子になった入力は、いずれも `invalidFilter` の ScimProtocolError になり、内部エラーには落ちない。")
@TypeSpec.OpenAPI.operationId("ListScimGroups")
@route("/scim/v2/Groups")
@get
op ListScimGroups(
  @query tenant_id: string,
  @query filter?: string,
  @query startIndex?: int64,
  @query count?: int64
): ListScimGroupsSuccess_200 | ListScimGroupsError400;

model GetScimGroupSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimResource;
}

union GetScimGroupError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model GetScimGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: GetScimGroupError400Body;
}

union GetScimGroupError404Body {
  IdMagic.Contract.ScimProtocolError,
}

model GetScimGroupError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/scim+json";
  @body body: GetScimGroupError404Body;
}

@doc("SCIM 2.0 の /Groups/{id} への GET。各メンバーは type=User で返す。リソースは `id`、`meta.resourceType`、`meta.created`、`meta.lastModified`、`meta.location` を含む。")
@TypeSpec.OpenAPI.operationId("GetScimGroup")
@route("/scim/v2/Groups/{id}")
@get
op GetScimGroup(
  @query tenant_id: string,
  @path id: string
): GetScimGroupSuccess_200 | GetScimGroupError400 | GetScimGroupError404;

model PatchScimGroupSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimResource;
}

union PatchScimGroupError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model PatchScimGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: PatchScimGroupError400Body;
}

@doc("SCIM 2.0 の /Groups/{id} への PATCH（RFC 7644 の 3.5.2 節）。`path` は RFC7644-PATCH が扱う Group の属性、`displayName` または `members` のいずれかでなければならず、それ以外は `invalidPath` の ScimProtocolError になる。add、replace、remove 以外の `op` と、対象の属性に合わない型の値は `invalidValue` になる。`members` は、`type` が省略または User（大文字と小文字を区別しない）のメンバーだけを受け付ける。Group の種類のメンバーと、解決しないメンバーは、どちらも `invalidValue` になり、グループは変更しない。`id`、`meta`、`schemas` への書き込みは mutability のエラーになる。")
@TypeSpec.OpenAPI.operationId("PatchScimGroup")
@route("/scim/v2/Groups/{id}")
@patch
op PatchScimGroup(
  @path id: string,
  @header contentType: "application/scim+json",
  @body body: IdMagic.Contract.ScimResource
): PatchScimGroupSuccess_200 | PatchScimGroupError400;

model UpdateScimGroupSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/scim+json";
  @body body: IdMagic.Contract.ScimResource;
}

union UpdateScimGroupError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model UpdateScimGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: UpdateScimGroupError400Body;
}

@doc("SCIM 2.0 の /Groups/{id} への PUT。RFC7643-CORE-RESOURCES が扱う変更可能な属性を丸ごと**置き換える**もので、部分更新ではない。`displayName` は必須で、ないときは `invalidValue` の ScimProtocolError になる。`members` を省くと既存のメンバーをすべて外し、集合を空に置き換える。`members` は、`type` が省略または User（大文字と小文字を区別しない）のメンバーだけを受け付ける。Group の種類のメンバーと、解決しないメンバーは、どちらも `invalidValue` になり、グループは変更しない。`id` と `meta` への書き込みは無視する（readOnly）。")
@TypeSpec.OpenAPI.operationId("UpdateScimGroup")
@route("/scim/v2/Groups/{id}")
@put
op UpdateScimGroup(
  @path id: string,
  @header contentType: "application/scim+json",
  @body body: IdMagic.Contract.ScimResource
): UpdateScimGroupSuccess_200 | UpdateScimGroupError400;

union DeleteScimGroupError400Body {
  IdMagic.Contract.ScimProtocolError,
}

model DeleteScimGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/scim+json";
  @body body: DeleteScimGroupError400Body;
}

model DeleteScimGroupSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteScimGroupError404Body {
  IdMagic.Contract.ScimProtocolError,
}

model DeleteScimGroupError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/scim+json";
  @body body: DeleteScimGroupError404Body;
}

@doc("SCIM 2.0 の /Groups/{id} への DELETE。")
@TypeSpec.OpenAPI.operationId("DeleteScimGroup")
@route("/scim/v2/Groups/{id}")
@delete
op DeleteScimGroup(
  @query tenant_id: string,
  @path id: string
): DeleteScimGroupSuccess_204 | DeleteScimGroupError400 | DeleteScimGroupError404;

union DirectoryConnectorAdminReadError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model DirectoryConnectorAdminReadError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DirectoryConnectorAdminReadError403Body;
}

union DirectoryConnectorAdminWriteError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DirectoryConnectorAdminWriteError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DirectoryConnectorAdminWriteError403Body;
}

union DirectoryConnectorNotFoundError404Body {
  IdMagic.Contract.DirectoryConnectorNotFoundError,
}

model DirectoryConnectorNotFoundError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DirectoryConnectorNotFoundError404Body;
}

model DirectoryConnectorEnrollmentGrantSuccess_201 {
  @statusCode statusCode: 201;
  @header cacheControl: "no-store";
  @body body: IdMagic.Contract.DirectoryConnectorEnrollmentGrant;
}

model DirectoryConnectorSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DirectoryConnector;
}

union CreateDirectoryConnectorError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateDirectoryConnectorError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateDirectoryConnectorError400Body;
}

@doc("テナントの管理者が Directory Connector を作成する。Connector は `awaiting_enrollment` で始まり、応答は一回だけ使える登録トークンを含む。トークンを返すのはこのときと再発行のときだけである。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("CreateDirectoryConnector")
@route("/api/admin/v1/directory-connectors")
@post
op CreateDirectoryConnector(
  @body body: IdMagic.Contract.CreateDirectoryConnectorRequest
): DirectoryConnectorEnrollmentGrantSuccess_201 | CreateDirectoryConnectorError400 | IdMagic.Contract.AuthenticationRequiredResponse | DirectoryConnectorAdminWriteError403;

model ListDirectoryConnectorsHttpResponse {
  @doc("作成時刻の昇順。")
  connectors: IdMagic.Contract.DirectoryConnector[];
}

model ListDirectoryConnectorsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListDirectoryConnectorsHttpResponse;
}

@doc("テナントの Directory Connector を一覧する。登録トークンもそのハッシュも返さない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ListDirectoryConnectors")
@route("/api/admin/v1/directory-connectors")
@get
op ListDirectoryConnectors(
): ListDirectoryConnectorsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | DirectoryConnectorAdminReadError403;

@doc("テナントの Directory Connector を一つ読む。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("GetDirectoryConnector")
@route("/api/admin/v1/directory-connectors/{connector_id}")
@get
op GetDirectoryConnector(
  @path connector_id: string
): DirectoryConnectorSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | DirectoryConnectorAdminReadError403 | DirectoryConnectorNotFoundError404;

union ReissueDirectoryConnectorEnrollmentError409Body {
  IdMagic.Contract.DirectoryConnectorDisabledError,
}

model ReissueDirectoryConnectorEnrollmentError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: ReissueDirectoryConnectorEnrollmentError409Body;
}

@doc("登録トークンを再発行する。以前の登録トークンと登録済みの公開鍵を破棄し、Connector を `awaiting_enrollment` に戻す。鍵の更新と、漏えいが疑われる鍵の失効に使う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ReissueDirectoryConnectorEnrollment")
@route("/api/admin/v1/directory-connectors/{connector_id}/enrollment-tokens")
@post
op ReissueDirectoryConnectorEnrollment(
  @path connector_id: string
): DirectoryConnectorEnrollmentGrantSuccess_201 | IdMagic.Contract.AuthenticationRequiredResponse | DirectoryConnectorAdminWriteError403 | DirectoryConnectorNotFoundError404 | ReissueDirectoryConnectorEnrollmentError409;

@doc("Directory Connector を無効化する。以後の登録と呼び出しを拒否する。無効化した Connector に繰り返しても成功し、何も変えない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("DisableDirectoryConnector")
@route("/api/admin/v1/directory-connectors/{connector_id}/disable")
@post
op DisableDirectoryConnector(
  @path connector_id: string
): DirectoryConnectorSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | DirectoryConnectorAdminWriteError403 | DirectoryConnectorNotFoundError404;

union DirectoryConnectorInvalidTokenError401Body {
  IdMagic.Contract.DirectoryConnectorInvalidTokenError,
}

model DirectoryConnectorInvalidTokenError401 {
  @statusCode statusCode: 401;
  @header contentType: "application/problem+json";
  @body body: DirectoryConnectorInvalidTokenError401Body;
}

model EnrollDirectoryConnectorSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DirectoryConnectorEnrollment;
}

union EnrollDirectoryConnectorError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model EnrollDirectoryConnectorError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: EnrollDirectoryConnectorError400Body;
}

@doc("閉域の Directory Connector が、登録トークンとともに手元の公開鍵を登録する。認証の手段は本文の登録トークンだけである。応答の `connector_id` と `audience` を、以後の JWT の `iss`、`sub`、`aud` に使う。")
@TypeSpec.OpenAPI.operationId("EnrollDirectoryConnector")
@route("/directory-connector/v1/enrollment")
@post
op EnrollDirectoryConnector(
  @body body: IdMagic.Contract.EnrollDirectoryConnectorRequest
): EnrollDirectoryConnectorSuccess_200 | EnrollDirectoryConnectorError400 | DirectoryConnectorInvalidTokenError401;

model ReportDirectoryConnectorHeartbeatSuccess_204 {
  @statusCode statusCode: 204;
}

@doc("登録済みの Directory Connector が接続を報告し、最終接続時刻を記録させる。`Authorization: Bearer` で、登録した秘密鍵で署名した JWT（`iss` と `sub` は Connector の ID、`aud` は登録の応答の `audience`、`exp` は 5 分以内、`jti` は一回限り、`alg` は ES256 か PS256）を提示する。")
@TypeSpec.OpenAPI.operationId("ReportDirectoryConnectorHeartbeat")
@route("/directory-connector/v1/heartbeat")
@post
op ReportDirectoryConnectorHeartbeat(
  @header authorization: string
): ReportDirectoryConnectorHeartbeatSuccess_204 | DirectoryConnectorInvalidTokenError401;

model IngestionRunSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.IngestionRun;
}

model IngestionRunSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.IngestionRun;
}

model IngestionRunBusyError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: IdMagic.Contract.IngestionRunBusyError;
}

@doc("Connector ごとの無効化のしきい値を更新する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("UpdateDirectoryConnectorThresholds")
@route("/api/admin/v1/directory-connectors/{connector_id}/deletion-thresholds")
@put
op UpdateDirectoryConnectorThresholds(
  @path connector_id: string,
  @body body: IdMagic.Contract.DirectoryConnectorThresholds
): DirectoryConnectorSuccess_200 | CreateDirectoryConnectorError400 | IdMagic.Contract.AuthenticationRequiredResponse | DirectoryConnectorAdminWriteError403 | DirectoryConnectorNotFoundError404;

union DirectoryIngestionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DirectoryIngestionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DirectoryIngestionError400Body;
}

union IngestionRunNotFoundError404Body {
  IdMagic.Contract.IngestionRunNotFoundError,
}

model IngestionRunNotFoundError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: IngestionRunNotFoundError404Body;
}

@doc("登録済みの Directory Connector が取り込みの実行を始める。同じ Connector の `running` の実行は `superseded` になる。認証は接続の報告と同じ JWT である。")
@TypeSpec.OpenAPI.operationId("StartDirectoryIngestionRun")
@route("/directory-connector/v1/ingestion-runs")
@post
op StartDirectoryIngestionRun(
  @header authorization: string,
  @body body: IdMagic.Contract.StartIngestionRunRequest
): IngestionRunSuccess_201 | DirectoryIngestionError400 | DirectoryConnectorInvalidTokenError401 | IngestionRunBusyError409;

model DirectoryUserBatchSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DirectoryUserBatchResult;
}

union SendDirectoryUserBatchError409Body {
  IdMagic.Contract.BatchConflictError,
  IdMagic.Contract.IngestionRunClosedError,
}

model SendDirectoryUserBatchError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: SendDirectoryUserBatchError409Body;
}

@doc("Directory Connector が `running` の実行へ User の項目のバッチを送る。項目を順に一件ずつ確定し、項目ごとの結果を返す。同じ ID と同じ項目の再送は、適用し直さずに最初の結果を返す。")
@TypeSpec.OpenAPI.operationId("SendDirectoryUserBatch")
@route("/directory-connector/v1/ingestion-runs/{run_id}/user-batches")
@post
op SendDirectoryUserBatch(
  @header authorization: string,
  @path run_id: string,
  @body body: IdMagic.Contract.DirectoryUserBatchRequest
): DirectoryUserBatchSuccess_200 | DirectoryIngestionError400 | DirectoryConnectorInvalidTokenError401 | IngestionRunNotFoundError404 | SendDirectoryUserBatchError409 | IngestionRunBusyError409;

union CompleteDirectoryIngestionRunError409Body {
  IdMagic.Contract.IngestionRunIncompleteError,
  IdMagic.Contract.IngestionRunClosedError,
}

model CompleteDirectoryIngestionRunError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: CompleteDirectoryIngestionRunError409Body;
}

@doc("Directory Connector が取り込みの実行の完了を報告する。`full` の実行では、同じ Connector の対応のうち、この実行に現れなかった `active` の User を `disabled` にする。")
@TypeSpec.OpenAPI.operationId("CompleteDirectoryIngestionRun")
@route("/directory-connector/v1/ingestion-runs/{run_id}/complete")
@post
op CompleteDirectoryIngestionRun(
  @header authorization: string,
  @path run_id: string,
  @body body: IdMagic.Contract.CompleteIngestionRunRequest
): IngestionRunSuccess_200 | DirectoryIngestionError400 | DirectoryConnectorInvalidTokenError401 | IngestionRunNotFoundError404 | CompleteDirectoryIngestionRunError409 | IngestionRunBusyError409;

@doc("保留した完全再同期の無効化を承認し、保存した候補を適用する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ApproveDirectoryIngestionRun")
@route("/api/admin/v1/directory-connectors/{connector_id}/ingestion-runs/{run_id}/approve")
@post
op ApproveDirectoryIngestionRun(
  @path connector_id: string,
  @path run_id: string
): IngestionRunSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | DirectoryConnectorAdminWriteError403 | GetDirectoryIngestionRunError404 | CompleteDirectoryIngestionRunError409 | IngestionRunBusyError409;

@doc("保留した完全再同期の無効化を破棄する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("DiscardDirectoryIngestionRun")
@route("/api/admin/v1/directory-connectors/{connector_id}/ingestion-runs/{run_id}/discard")
@post
op DiscardDirectoryIngestionRun(
  @path connector_id: string,
  @path run_id: string
): IngestionRunSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | DirectoryConnectorAdminWriteError403 | GetDirectoryIngestionRunError404 | CompleteDirectoryIngestionRunError409 | IngestionRunBusyError409;

model DirectoryIngestionCandidatesResponse {
  candidates: IdMagic.Contract.DirectoryIngestionCandidate[];
  total: int64;
}

model DirectoryIngestionCandidatesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: DirectoryIngestionCandidatesResponse;
}

@doc("実行に保存した無効化の候補を照会する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ListDirectoryIngestionCandidates")
@route("/api/admin/v1/directory-connectors/{connector_id}/ingestion-runs/{run_id}/candidates")
@get
op ListDirectoryIngestionCandidates(
  @path connector_id: string,
  @path run_id: string,
  @query @minValue(0) offset?: int64,
  @query @minValue(1) @maxValue(100) limit?: int64
): DirectoryIngestionCandidatesSuccess_200 | DirectoryIngestionError400 | IdMagic.Contract.AuthenticationRequiredResponse | DirectoryConnectorAdminReadError403 | GetDirectoryIngestionRunError404;

model ListDirectoryIngestionRunsHttpResponse {
  @doc("開始時刻の新しい順に最大 50 件。")
  runs: IdMagic.Contract.IngestionRunSummary[];
}

model ListDirectoryIngestionRunsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListDirectoryIngestionRunsHttpResponse;
}

@doc("Directory Connector の取り込みの実行を、結果ごとの項目の数とともに一覧する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ListDirectoryIngestionRuns")
@route("/api/admin/v1/directory-connectors/{connector_id}/ingestion-runs")
@get
op ListDirectoryIngestionRuns(
  @path connector_id: string
): ListDirectoryIngestionRunsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | DirectoryConnectorAdminReadError403 | DirectoryConnectorNotFoundError404;

model IngestionRunDetailSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.IngestionRunDetail;
}

union GetDirectoryIngestionRunError404Body {
  IdMagic.Contract.DirectoryConnectorNotFoundError,
  IdMagic.Contract.IngestionRunNotFoundError,
}

model GetDirectoryIngestionRunError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetDirectoryIngestionRunError404Body;
}

@doc("取り込みの実行を一つ読む。結果ごとの項目の数と、`failed` の項目を最大 100 件返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("GetDirectoryIngestionRun")
@route("/api/admin/v1/directory-connectors/{connector_id}/ingestion-runs/{run_id}")
@get
op GetDirectoryIngestionRun(
  @path connector_id: string,
  @path run_id: string
): IngestionRunDetailSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | DirectoryConnectorAdminReadError403 | GetDirectoryIngestionRunError404;
}

}
