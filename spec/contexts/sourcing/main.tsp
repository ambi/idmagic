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
): CreateScimUserSuccess_201 | CreateScimUserError400;

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
): CreateScimGroupSuccess_201 | CreateScimGroupError400;

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
}

}
