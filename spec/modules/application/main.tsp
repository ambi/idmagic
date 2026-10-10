import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Application")
namespace IdMagic.Application {

namespace Operations {
model ListAdminApplicationsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminApplicationListResponse;
}

union ListAdminApplicationsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminApplicationsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminApplicationsError403Body;
}

union ListAdminApplicationsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListAdminApplicationsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListAdminApplicationsError400Body;
}

@doc("管理者が、自身のテナントの Application を、名前の昇順（同じなら id の順）に、双方向のキーセットページングで一覧する。カーソルはレスポンスの Link レスポンスヘッダー（`rel=\"prev\"` と `rel=\"next\"`）から得る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:read"])
@TypeSpec.OpenAPI.operationId("ListAdminApplications")
@route("/api/admin/v1/applications")
@get
op ListAdminApplications(
  @query cursor?: string,
  @query limit?: int64
): ListAdminApplicationsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListAdminApplicationsError403 | ListAdminApplicationsError400;

model GetAdminApplicationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminApplicationDetailResponse;
}

union GetAdminApplicationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetAdminApplicationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetAdminApplicationError400Body;
}

union GetAdminApplicationError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminApplicationError403Body;
}

union GetAdminApplicationError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
}

model GetAdminApplicationError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAdminApplicationError404Body;
}

@doc("管理者が、プロトコルの関係を通じて解決した実際の設定を含めて Application を読む。OIDC または SAML のアプリケーションの詳細画面は、登録したこれらの相手側の設定を、Tenancy.GetAdminIntegrationEndpoints が返す IdMagic の設定の案内と並べて示し、クライアントシークレットを再び表示することはない。他のテナントのものは存在しないものとして扱う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:read"])
@TypeSpec.OpenAPI.operationId("GetAdminApplication")
@route("/api/admin/v1/applications/{id}")
@get
op GetAdminApplication(
  @path id: string
): GetAdminApplicationSuccess_200 | GetAdminApplicationError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminApplicationError403 | GetAdminApplicationError404;

model CreateAdminApplicationSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.AdminApplicationCreateResponse;
}

union CreateAdminApplicationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateAdminApplicationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateAdminApplicationError400Body;
}

union CreateAdminApplicationError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CreateAdminApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateAdminApplicationError403Body;
}

union CreateAdminApplicationError422Body {
  IdMagic.Contract.QuotaExceededError,
}

model CreateAdminApplicationError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: CreateAdminApplicationError422Body;
}

@doc("管理者が、一回の呼び出しで Application を作成する。種類に応じたプロトコルの行を用意し、Application と `application_id` の関係を一つのトランザクションでカタログへ公開する。プロトコルの種類は後から変えられない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("CreateAdminApplication")
@route("/api/admin/v1/applications")
@post
op CreateAdminApplication(
  @body body: IdMagic.Contract.AdminApplicationCreateRequest
): CreateAdminApplicationSuccess_201 | CreateAdminApplicationError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateAdminApplicationError403 | CreateAdminApplicationError422;

model UpdateAdminApplicationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminApplicationResponse;
}

union UpdateAdminApplicationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateAdminApplicationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminApplicationError400Body;
}

union UpdateAdminApplicationError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAdminApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminApplicationError403Body;
}

union UpdateAdminApplicationError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
}

model UpdateAdminApplicationError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminApplicationError404Body;
}

@doc("管理者が、Application のメタデータを更新する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UpdateAdminApplication")
@route("/api/admin/v1/applications/{id}")
@patch
op UpdateAdminApplication(
  @path id: string,
  @body body: IdMagic.Contract.AdminApplicationUpdateRequest
): UpdateAdminApplicationSuccess_200 | UpdateAdminApplicationError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateAdminApplicationError403 | UpdateAdminApplicationError404;

model UploadApplicationIconSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.ApplicationIconUploadResponse;
}

union UploadApplicationIconError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UploadApplicationIconError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UploadApplicationIconError400Body;
}

union UploadApplicationIconError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UploadApplicationIconError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UploadApplicationIconError403Body;
}

union UploadApplicationIconError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
}

model UploadApplicationIconError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UploadApplicationIconError404Body;
}

@doc("管理者が、Application のアイコンの画像を multipart でアップロードする。PNG、JPEG、WebP、GIF を 256 KiB まで受け付け、形式はマジックバイトで確かめる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UploadApplicationIcon")
@route("/api/admin/v1/applications/{id}/icon")
@post
op UploadApplicationIcon(
  @path id: string,
  @header contentType: "multipart/form-data",
  @multipartBody body: {
    @doc("保存する画像。フォームのフィールド名は `file` である。")
    file: HttpPart<File>,
  }
): UploadApplicationIconSuccess_200 | UploadApplicationIconError400 | IdMagic.Contract.AuthenticationRequiredResponse | UploadApplicationIconError403 | UploadApplicationIconError404;

model DeleteApplicationIconSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.ApplicationIconUploadResponse;
}

union DeleteApplicationIconError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteApplicationIconError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteApplicationIconError400Body;
}

union DeleteApplicationIconError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteApplicationIconError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteApplicationIconError403Body;
}

union DeleteApplicationIconError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
}

model DeleteApplicationIconError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DeleteApplicationIconError404Body;
}

@doc("管理者が、保存した Application のアイコンを削除する。以降は `icon_url` を返さない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("DeleteApplicationIcon")
@route("/api/admin/v1/applications/{id}/icon")
@delete
op DeleteApplicationIcon(
  @path id: string
): DeleteApplicationIconSuccess_200 | DeleteApplicationIconError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteApplicationIconError403 | DeleteApplicationIconError404;

model GetApplicationIconSuccess_200 {
  @statusCode statusCode: 200;
  @doc("保存したアイコンの、検証済みの形式に固定する。`X-Content-Type-Options: nosniff` を付けて返す。")
  @header contentType: "image/png" | "image/jpeg" | "image/webp" | "image/gif";
  @doc("アイコンのバイト列。")
  @body body: bytes;
}

union GetApplicationIconError404Body {
  IdMagic.Contract.ApplicationIconNotFoundError,
}

model GetApplicationIconError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetApplicationIconError404Body;
}

@doc("保存した Application のアイコンを返す公開インターフェース。Content-Type は検証済みの形式に固定し、`X-Content-Type-Options: nosniff` を設定する。他のテナントのオブジェクトと削除済みのオブジェクトは、どちらも存在しないものとして扱う。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetApplicationIcon")
@route("/application-icons/{application_id}/{id}")
@get
op GetApplicationIcon(
  @path application_id: string,
  @path id: string
): GetApplicationIconSuccess_200 | GetApplicationIconError404;

model DeleteAdminApplicationSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteAdminApplicationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteAdminApplicationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminApplicationError400Body;
}

union DeleteAdminApplicationError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAdminApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminApplicationError403Body;
}

union DeleteAdminApplicationError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
}

model DeleteAdminApplicationError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminApplicationError404Body;
}

@doc("管理者が、Application を冪等に削除する。関連するプロトコルの行は、同じトランザクションで連鎖して削除する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("DeleteAdminApplication")
@route("/api/admin/v1/applications/{id}")
@delete
op DeleteAdminApplication(
  @path id: string
): DeleteAdminApplicationSuccess_204 | DeleteAdminApplicationError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteAdminApplicationError403 | DeleteAdminApplicationError404;

model UpdateApplicationOidcConfigSuccess_204 {
  @statusCode statusCode: 204;
}

union UpdateApplicationOidcConfigError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateApplicationOidcConfigError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationOidcConfigError400Body;
}

union UpdateApplicationOidcConfigError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateApplicationOidcConfigError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationOidcConfigError403Body;
}

union UpdateApplicationOidcConfigError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
  IdMagic.Contract.ApplicationProtocolTargetNotFoundError,
}

model UpdateApplicationOidcConfigError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationOidcConfigError404Body;
}

@doc("管理者が、Application の OIDC のプロトコルが指す OAuth2 のクライアントの設定を更新する。`rules` と `sub_source_attribute` はクレームの開示を上書きするので、公開範囲が `private` の属性や予約済みのクレームの種類を参照する規則は拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UpdateApplicationOidcConfig")
@route("/api/admin/v1/applications/{id}/oidc")
@patch
op UpdateApplicationOidcConfig(
  @path id: string,
  @body body: IdMagic.Contract.ApplicationOidcConfigUpdateRequest
): UpdateApplicationOidcConfigSuccess_204 | UpdateApplicationOidcConfigError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateApplicationOidcConfigError403 | UpdateApplicationOidcConfigError404;

model RotateApplicationClientSecretSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.RotateApplicationClientSecretResponse;
}

union RotateApplicationClientSecretError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RotateApplicationClientSecretError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RotateApplicationClientSecretError400Body;
}

union RotateApplicationClientSecretError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RotateApplicationClientSecretError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RotateApplicationClientSecretError403Body;
}

union RotateApplicationClientSecretError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
}

model RotateApplicationClientSecretError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: RotateApplicationClientSecretError404Body;
}

@doc("互換性のために、管理者が Application の OIDC のプロトコルのクライアントシークレットを従来の方法でローテーションする。それまでの資格情報は指定した重複期間の間有効で、0 日は即座に失効させることを意味する。新しい管理 UI は、代わりに IssueApplicationClientSecret と RevokeApplicationClientSecret を使う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("RotateApplicationClientSecret")
@route("/api/admin/v1/applications/{id}/oidc/rotate-secret")
@post
op RotateApplicationClientSecret(
  @path id: string,
  @body body: IdMagic.Contract.RotateApplicationClientSecretRequest
): RotateApplicationClientSecretSuccess_200 | RotateApplicationClientSecretError400 | IdMagic.Contract.AuthenticationRequiredResponse | RotateApplicationClientSecretError403 | RotateApplicationClientSecretError404;

model IssueApplicationClientSecretSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.IssueApplicationClientSecretResponse;
}

union IssueApplicationClientSecretError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model IssueApplicationClientSecretError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: IssueApplicationClientSecretError400Body;
}

union IssueApplicationClientSecretError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model IssueApplicationClientSecretError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: IssueApplicationClientSecretError403Body;
}

union IssueApplicationClientSecretError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
}

model IssueApplicationClientSecretError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: IssueApplicationClientSecretError404Body;
}

union IssueApplicationClientSecretError422Body {
  IdMagic.Contract.ClientSecretLimitExceededError,
}

model IssueApplicationClientSecretError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: IssueApplicationClientSecretError422Body;
}

@doc("管理者が、Application のシークレット方式の OIDC のクライアントに、有効期限付きの追加の資格情報を発行する。既存の資格情報は変えず、すでに Active の資格情報が二つあるときは原子的にリクエストを拒否する。シークレット方式でないクライアントは拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("IssueApplicationClientSecret")
@route("/api/admin/v1/applications/{id}/oidc/client-secrets")
@post
op IssueApplicationClientSecret(
  @path id: string,
  @body body: IdMagic.Contract.IssueApplicationClientSecretRequest
): IssueApplicationClientSecretSuccess_201 | IssueApplicationClientSecretError400 | IdMagic.Contract.AuthenticationRequiredResponse | IssueApplicationClientSecretError403 | IssueApplicationClientSecretError422 | IssueApplicationClientSecretError404;

model RevokeApplicationClientSecretSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.RevokeApplicationClientSecretResponse;
}

union RevokeApplicationClientSecretError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RevokeApplicationClientSecretError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RevokeApplicationClientSecretError400Body;
}

union RevokeApplicationClientSecretError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RevokeApplicationClientSecretError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeApplicationClientSecretError403Body;
}

union RevokeApplicationClientSecretError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
}

model RevokeApplicationClientSecretError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: RevokeApplicationClientSecretError404Body;
}

@doc("管理者が、Application のシークレット方式の OIDC のクライアントの資格情報を一つ指定して、即座に失効させる。失効済みの資格情報の失効は冪等に成功し、別のクライアントの資格情報と存在しない資格情報は拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("RevokeApplicationClientSecret")
@route("/api/admin/v1/applications/{id}/oidc/client-secrets/{credential_id}")
@delete
op RevokeApplicationClientSecret(
  @path id: string,
  @path credential_id: string
): RevokeApplicationClientSecretSuccess_200 | RevokeApplicationClientSecretError400 | IdMagic.Contract.AuthenticationRequiredResponse | RevokeApplicationClientSecretError403 | RevokeApplicationClientSecretError404;

model UpdateApplicationWsFedConfigSuccess_204 {
  @statusCode statusCode: 204;
}

union UpdateApplicationWsFedConfigError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateApplicationWsFedConfigError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationWsFedConfigError400Body;
}

union UpdateApplicationWsFedConfigError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateApplicationWsFedConfigError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationWsFedConfigError403Body;
}

union UpdateApplicationWsFedConfigError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
  IdMagic.Contract.ApplicationProtocolTargetNotFoundError,
}

model UpdateApplicationWsFedConfigError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationWsFedConfigError404Body;
}

@doc("管理者が、Application の WS-Fed のプロトコルが指す Relying Party の設定を更新する。`rules` と `name_id_source` はクレームの開示を上書きするので、公開範囲が `private` の属性や予約済みのクレームの種類を参照する規則は拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UpdateApplicationWsFedConfig")
@route("/api/admin/v1/applications/{id}/wsfed")
@patch
op UpdateApplicationWsFedConfig(
  @path id: string,
  @body body: IdMagic.Contract.ApplicationWsFedConfigUpdateRequest
): UpdateApplicationWsFedConfigSuccess_204 | UpdateApplicationWsFedConfigError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateApplicationWsFedConfigError403 | UpdateApplicationWsFedConfigError404;

model UpdateApplicationSamlConfigSuccess_204 {
  @statusCode statusCode: 204;
}

union UpdateApplicationSamlConfigError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateApplicationSamlConfigError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationSamlConfigError400Body;
}

union UpdateApplicationSamlConfigError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateApplicationSamlConfigError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationSamlConfigError403Body;
}

union UpdateApplicationSamlConfigError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
  IdMagic.Contract.ApplicationProtocolTargetNotFoundError,
}

model UpdateApplicationSamlConfigError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationSamlConfigError404Body;
}

@doc("管理者が、Application の SAML のプロトコルが指すサービスプロバイダーの設定と IdP プロファイルの結び付きを部分的に更新する。署名の要求を有効にすると、検証できる証明書が必須になる。`rules` と `name_id_source` はクレームの開示を上書きするので、公開範囲が `private` の属性や予約済みのクレームの種類を参照する規則は拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UpdateApplicationSamlConfig")
@route("/api/admin/v1/applications/{id}/saml")
@patch
op UpdateApplicationSamlConfig(
  @path id: string,
  @body body: IdMagic.Contract.ApplicationSamlConfigUpdateRequest
): UpdateApplicationSamlConfigSuccess_204 | UpdateApplicationSamlConfigError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateApplicationSamlConfigError403 | UpdateApplicationSamlConfigError404;

model ListApplicationAssignmentsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.ApplicationAssignmentListResponse;
}

union ListApplicationAssignmentsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListApplicationAssignmentsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListApplicationAssignmentsError400Body;
}

union ListApplicationAssignmentsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListApplicationAssignmentsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListApplicationAssignmentsError403Body;
}

union ListApplicationAssignmentsError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
}

model ListApplicationAssignmentsError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: ListApplicationAssignmentsError404Body;
}

@doc("管理者が、Application の割り当てを、`subject_type` と `subject_id` の昇順に、双方向のキーセットページングで一覧する。カーソルはレスポンスの Link レスポンスヘッダー（`rel=\"prev\"` と `rel=\"next\"`）から得る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:read"])
@TypeSpec.OpenAPI.operationId("ListApplicationAssignments")
@route("/api/admin/v1/applications/{id}/assignments")
@get
op ListApplicationAssignments(
  @path id: string,
  @query cursor?: string,
  @query limit?: int64
): ListApplicationAssignmentsSuccess_200 | ListApplicationAssignmentsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ListApplicationAssignmentsError403 | ListApplicationAssignmentsError404;

model AssignApplicationSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.ApplicationAssignmentResponse;
}

union AssignApplicationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model AssignApplicationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: AssignApplicationError400Body;
}

union AssignApplicationError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model AssignApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: AssignApplicationError403Body;
}

union AssignApplicationError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
}

model AssignApplicationError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: AssignApplicationError404Body;
}

@doc("管理者が、ユーザーまたはグループを Application に割り当てる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("AssignApplication")
@route("/api/admin/v1/applications/{id}/assignments")
@post
op AssignApplication(
  @path id: string,
  @body body: IdMagic.Contract.ApplicationAssignmentRequest
): AssignApplicationSuccess_201 | AssignApplicationError400 | IdMagic.Contract.AuthenticationRequiredResponse | AssignApplicationError403 | AssignApplicationError404;

model UnassignApplicationSuccess_204 {
  @statusCode statusCode: 204;
}

union UnassignApplicationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UnassignApplicationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UnassignApplicationError400Body;
}

union UnassignApplicationError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UnassignApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UnassignApplicationError403Body;
}

@doc("管理者が、Application から割り当てを冪等に外す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UnassignApplication")
@route("/api/admin/v1/applications/{id}/assignments/{subject_type}/{subject_id}")
@delete
op UnassignApplication(
  @path id: string,
  @path subject_type: string,
  @path subject_id: string
): UnassignApplicationSuccess_204 | UnassignApplicationError400 | IdMagic.Contract.AuthenticationRequiredResponse | UnassignApplicationError403;

model GetAppSignInPolicySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AppSignInPolicyResponse;
}

union GetAppSignInPolicyError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetAppSignInPolicyError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetAppSignInPolicyError400Body;
}

union GetAppSignInPolicyError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAppSignInPolicyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAppSignInPolicyError403Body;
}

union GetAppSignInPolicyError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
}

model GetAppSignInPolicyError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAppSignInPolicyError404Body;
}

@doc("管理者が、Application のサインインのポリシーを読む。アプリケーション自身のポリシー、テナントのデフォルト、上書きの後に実際に適用される規則の並び、デフォルトより弱いポリシーの警告のフラグを返す。何も設定していないときは空の規則を返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:read"])
@TypeSpec.OpenAPI.operationId("GetAppSignInPolicy")
@route("/api/admin/v1/applications/{id}/sign-in-policy")
@get
op GetAppSignInPolicy(
  @path id: string
): GetAppSignInPolicySuccess_200 | GetAppSignInPolicyError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetAppSignInPolicyError403 | GetAppSignInPolicyError404;

model UpdateAppSignInPolicySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AppSignInPolicyResponse;
}

union UpdateAppSignInPolicyError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateAppSignInPolicyError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateAppSignInPolicyError400Body;
}

union UpdateAppSignInPolicyError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAppSignInPolicyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAppSignInPolicyError403Body;
}

union UpdateAppSignInPolicyError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
}

model UpdateAppSignInPolicyError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateAppSignInPolicyError404Body;
}

@doc("管理者が、Application のサインインのポリシーを置き換える。不正な CIDR を含む条件は拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UpdateAppSignInPolicy")
@route("/api/admin/v1/applications/{id}/sign-in-policy")
@put
op UpdateAppSignInPolicy(
  @path id: string,
  @body body: IdMagic.Contract.AppSignInPolicyRequest
): UpdateAppSignInPolicySuccess_200 | UpdateAppSignInPolicyError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateAppSignInPolicyError403 | UpdateAppSignInPolicyError404;

model GetTenantDefaultSignInPolicySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantDefaultSignInPolicyResponse;
}

union GetTenantDefaultSignInPolicyError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetTenantDefaultSignInPolicyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetTenantDefaultSignInPolicyError403Body;
}

@doc("管理者が、テナントのデフォルトのサインインのポリシーを、MFA を登録していない有効なユーザーの数とともに読む。何も設定していないときは、空の規則のポリシーを返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("GetTenantDefaultSignInPolicy")
@route("/api/admin/v1/default-sign-in-policy")
@get
op GetTenantDefaultSignInPolicy(

): GetTenantDefaultSignInPolicySuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetTenantDefaultSignInPolicyError403;

model UpdateTenantDefaultSignInPolicySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantDefaultSignInPolicyResponse;
}

union UpdateTenantDefaultSignInPolicyError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateTenantDefaultSignInPolicyError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantDefaultSignInPolicyError400Body;
}

union UpdateTenantDefaultSignInPolicyError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantDefaultSignInPolicyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantDefaultSignInPolicyError403Body;
}

@doc("管理者が、テナントのデフォルトのサインインのポリシーを置き換える。MFA を必須にするときは、強制の開始時刻、登録を回避できる猶予、管理者の承認を許すかどうかを示す。不正な CIDR、過去の強制の開始時刻、正でない猶予は、いずれも拒否する。空の規則を保存すると、デフォルトはすべて許可に戻る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UpdateTenantDefaultSignInPolicy")
@route("/api/admin/v1/default-sign-in-policy")
@put
op UpdateTenantDefaultSignInPolicy(
  @body body: IdMagic.Contract.TenantDefaultSignInPolicyRequest
): UpdateTenantDefaultSignInPolicySuccess_200 | UpdateTenantDefaultSignInPolicyError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateTenantDefaultSignInPolicyError403;

model ListApplicationCategoriesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminApplicationCategoryListResponse;
}

union ListApplicationCategoriesError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListApplicationCategoriesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListApplicationCategoriesError403Body;
}

@doc("管理者が、自身のテナントの ApplicationCategory を位置の昇順で一覧する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:read"])
@TypeSpec.OpenAPI.operationId("ListApplicationCategories")
@route("/api/admin/v1/application-categories")
@get
op ListApplicationCategories(

): ListApplicationCategoriesSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListApplicationCategoriesError403;

model CreateApplicationCategoryHttpResponse {
  category: IdMagic.Contract.AdminApplicationCategoryResponse;
}

model CreateApplicationCategorySuccess_201 {
  @statusCode statusCode: 201;
  @body body: CreateApplicationCategoryHttpResponse;
}

union CreateApplicationCategoryError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateApplicationCategoryError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateApplicationCategoryError400Body;
}

union CreateApplicationCategoryError422Body {
  IdMagic.Contract.CategoryNameRequiredError,
}

model CreateApplicationCategoryError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: CreateApplicationCategoryError422Body;
}

union CreateApplicationCategoryError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CreateApplicationCategoryError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateApplicationCategoryError403Body;
}

@doc("管理者が ApplicationCategory を作成する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("CreateApplicationCategory")
@route("/api/admin/v1/application-categories")
@post
op CreateApplicationCategory(
  @body body: IdMagic.Contract.AdminApplicationCategoryRequest
): CreateApplicationCategorySuccess_201 | CreateApplicationCategoryError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateApplicationCategoryError422 | CreateApplicationCategoryError403;

model UpdateApplicationCategoryHttpResponse {
  category: IdMagic.Contract.AdminApplicationCategoryResponse;
}

model UpdateApplicationCategorySuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateApplicationCategoryHttpResponse;
}

union UpdateApplicationCategoryError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateApplicationCategoryError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationCategoryError400Body;
}

union UpdateApplicationCategoryError422Body {
  IdMagic.Contract.CategoryNameRequiredError,
}

model UpdateApplicationCategoryError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationCategoryError422Body;
}

union UpdateApplicationCategoryError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateApplicationCategoryError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationCategoryError403Body;
}

union UpdateApplicationCategoryError404Body {
  IdMagic.Contract.ApplicationCategoryNotFoundError,
}

model UpdateApplicationCategoryError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationCategoryError404Body;
}

@doc("管理者が、ApplicationCategory の名前または位置を更新する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UpdateApplicationCategory")
@route("/api/admin/v1/application-categories/{id}")
@patch
op UpdateApplicationCategory(
  @path id: string,
  @body body: IdMagic.Contract.AdminApplicationCategoryRequest
): UpdateApplicationCategorySuccess_200 | UpdateApplicationCategoryError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateApplicationCategoryError422 | UpdateApplicationCategoryError403 | UpdateApplicationCategoryError404;

model DeleteApplicationCategorySuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteApplicationCategoryError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteApplicationCategoryError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteApplicationCategoryError400Body;
}

union DeleteApplicationCategoryError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteApplicationCategoryError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteApplicationCategoryError403Body;
}

union DeleteApplicationCategoryError404Body {
  IdMagic.Contract.ApplicationCategoryNotFoundError,
}

model DeleteApplicationCategoryError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DeleteApplicationCategoryError404Body;
}

@doc("管理者が、ApplicationCategory を冪等に削除する。割り当てていた Application からも外れる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("DeleteApplicationCategory")
@route("/api/admin/v1/application-categories/{id}")
@delete
op DeleteApplicationCategory(
  @path id: string
): DeleteApplicationCategorySuccess_204 | DeleteApplicationCategoryError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteApplicationCategoryError403 | DeleteApplicationCategoryError404;

model SetApplicationCategoriesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminApplicationResponse;
}

union SetApplicationCategoriesError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model SetApplicationCategoriesError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: SetApplicationCategoriesError400Body;
}

union SetApplicationCategoriesError422Body {
  IdMagic.Contract.UnknownCategoryError,
}

model SetApplicationCategoriesError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: SetApplicationCategoriesError422Body;
}

union SetApplicationCategoriesError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model SetApplicationCategoriesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SetApplicationCategoriesError403Body;
}

union SetApplicationCategoriesError404Body {
  IdMagic.Contract.ApplicationNotFoundError,
}

model SetApplicationCategoriesError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: SetApplicationCategoriesError404Body;
}

@doc("管理者が、Application に割り当てるカテゴリーの集合を置き換える。受け付けるのは、自身のテナントにすでに存在するカテゴリーだけである。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("SetApplicationCategories")
@route("/api/admin/v1/applications/{id}/categories")
@put
op SetApplicationCategories(
  @path id: string,
  @body body: IdMagic.Contract.ApplicationCategoriesRequest
): SetApplicationCategoriesSuccess_200 | SetApplicationCategoriesError400 | IdMagic.Contract.AuthenticationRequiredResponse | SetApplicationCategoriesError422 | SetApplicationCategoriesError403 | SetApplicationCategoriesError404;

model ListMyApplicationsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.MyApplicationListResponse;
}

union ListMyApplicationsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListMyApplicationsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListMyApplicationsError403Body;
}

@doc("ユーザーが、自身に割り当てられ表示される Application を一覧する。非表示の割り当ては除く。手動の並び順（ApplicationOrdering）を保存していればそれを優先し、なければ名前の昇順にする。レスポンスの `categories` はテナントのカテゴリーの定義を位置の昇順で含み、各アプリケーションはその `category_ids` によって節へ振り分ける。")
@TypeSpec.OpenAPI.operationId("ListMyApplications")
@route("/api/account/v1/applications")
@get
op ListMyApplications(

): ListMyApplicationsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListMyApplicationsError403;

model GetMyApplicationOrderSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.MyApplicationOrderResponse;
}

union GetMyApplicationOrderError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetMyApplicationOrderError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetMyApplicationOrderError403Body;
}

@doc("ユーザーが、保存したポータルの手動の並び順を読む。何も保存していないときは空の配列を返す。")
@TypeSpec.OpenAPI.operationId("GetMyApplicationOrder")
@route("/api/account/v1/applications/order")
@get
op GetMyApplicationOrder(

): GetMyApplicationOrderSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetMyApplicationOrderError403;

model ReorderMyApplicationsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.MyApplicationOrderResponse;
}

union ReorderMyApplicationsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ReorderMyApplicationsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ReorderMyApplicationsError400Body;
}

union ReorderMyApplicationsError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ReorderMyApplicationsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ReorderMyApplicationsError403Body;
}

@doc("ユーザーが、自身のポータルの手動の並び順を保存する。`application_ids` に含められるのは、自身に割り当てられ表示されるアプリケーションだけで、それ以外は拒否する。")
@TypeSpec.OpenAPI.operationId("ReorderMyApplications")
@route("/api/account/v1/applications/order")
@put
op ReorderMyApplications(
  @body body: IdMagic.Contract.ReorderMyApplicationsRequest
): ReorderMyApplicationsSuccess_200 | ReorderMyApplicationsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ReorderMyApplicationsError403;
}

}
