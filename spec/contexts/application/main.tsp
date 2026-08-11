import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace IdMagic.Application {

namespace Operations {
model ListAdminApplicationsHttpResponse {
  response: IdMagic.Contract.AdminApplicationListResponse;
}

model ListAdminApplicationsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListAdminApplicationsHttpResponse;
}

union ListAdminApplicationsError403Body {
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

@doc("管理者が所属テナントの Application を name 昇順 (同値は id で tie-break) の双方向 keyset\npagination で一覧する。cursor は応答の Link response header (rel=\"prev\" / rel=\"next\") から取得する。")
@TypeSpec.OpenAPI.operationId("ListAdminApplications")
@route("/api/admin/v1/applications")
@get
op ListAdminApplications(
  @query cursor?: string,
  @query limit?: int64
): ListAdminApplicationsSuccess_200 | ListAdminApplicationsError403 | ListAdminApplicationsError400;

model GetAdminApplicationHttpResponse {
  response: IdMagic.Contract.AdminApplicationDetailResponse;
}

model GetAdminApplicationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAdminApplicationHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminApplicationError403Body;
}

@doc("管理者が Application を取得する。protocol relation から解決した実設定も含む。OIDC / SAML application の詳細画面は、この登録済み相手側設定と Tenancy.GetAdminIntegrationEndpoints が返す IdMagic 側の setup guidance を並べて表示し、client secret は再表示しない。別テナントは未存在として扱う。")
@TypeSpec.OpenAPI.operationId("GetAdminApplication")
@route("/api/admin/v1/applications/{id}")
@get
op GetAdminApplication(
  @path id: string
): GetAdminApplicationSuccess_200 | GetAdminApplicationError400 | GetAdminApplicationError403;

model CreateAdminApplicationHttpRequest {
  request: IdMagic.Contract.AdminApplicationCreateRequest;
}

model CreateAdminApplicationHttpResponse {
  response: IdMagic.Contract.AdminApplicationCreateResponse;
}

model CreateAdminApplicationSuccess_201 {
  @statusCode statusCode: 201;
  @body body: CreateAdminApplicationHttpResponse;
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

@doc("管理者が Application を一括作成する。種別に応じた protocol row を準備し、Application と application_id relation を同一 transaction で catalog へ公開する。protocol 種別は作成後に変更できない。")
@TypeSpec.OpenAPI.operationId("CreateAdminApplication")
@route("/api/admin/v1/applications")
@post
op CreateAdminApplication(
  @body body: CreateAdminApplicationHttpRequest
): CreateAdminApplicationSuccess_201 | CreateAdminApplicationError400 | CreateAdminApplicationError403 | CreateAdminApplicationError422;

model UpdateAdminApplicationHttpRequest {
  id: string;
  request: IdMagic.Contract.AdminApplicationUpdateRequest;
}

model UpdateAdminApplicationHttpResponse {
  application: IdMagic.Contract.AdminApplicationResponse;
}

model UpdateAdminApplicationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateAdminApplicationHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAdminApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminApplicationError403Body;
}

@doc("管理者が Application のメタデータを更新する。")
@TypeSpec.OpenAPI.operationId("UpdateAdminApplication")
@route("/api/admin/v1/applications/{id}")
@patch
op UpdateAdminApplication(
  @path id: string,
  @body body: UpdateAdminApplicationHttpRequest
): UpdateAdminApplicationSuccess_200 | UpdateAdminApplicationError400 | UpdateAdminApplicationError403;

model UploadApplicationIconHttpRequest {
  id: string;
  file: string;
}

model UploadApplicationIconHttpResponse {
  response: IdMagic.Contract.ApplicationIconUploadResponse;
}

model UploadApplicationIconSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UploadApplicationIconHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model UploadApplicationIconError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UploadApplicationIconError403Body;
}

@doc("管理者が Application のアイコン画像を multipart でアップロードする。受理形式は PNG / JPEG / WebP / GIF、最大 256KiB とし、magic byte で検証する。")
@TypeSpec.OpenAPI.operationId("UploadApplicationIcon")
@route("/api/admin/v1/applications/{id}/icon")
@post
op UploadApplicationIcon(
  @path id: string,
  @body body: UploadApplicationIconHttpRequest
): UploadApplicationIconSuccess_200 | UploadApplicationIconError400 | UploadApplicationIconError403;

model DeleteApplicationIconHttpResponse {
  response: IdMagic.Contract.ApplicationIconUploadResponse;
}

model DeleteApplicationIconSuccess_200 {
  @statusCode statusCode: 200;
  @body body: DeleteApplicationIconHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model DeleteApplicationIconError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteApplicationIconError403Body;
}

@doc("管理者が Application の保存済みアイコンを削除する。削除後は icon_url を返さない。")
@TypeSpec.OpenAPI.operationId("DeleteApplicationIcon")
@route("/api/admin/v1/applications/{id}/icon")
@delete
op DeleteApplicationIcon(
  @path id: string
): DeleteApplicationIconSuccess_200 | DeleteApplicationIconError400 | DeleteApplicationIconError403;

model GetApplicationIconHttpResponse {
  file: string;
}

model GetApplicationIconSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetApplicationIconHttpResponse;
}

union GetApplicationIconError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetApplicationIconError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetApplicationIconError400Body;
}

@doc("保存済み Application アイコンを配信する公開 interface。Content-Type は検証済み形式に固定し、X-Content-Type-Options: nosniff を付ける。別テナントまたは削除済み object は未存在として扱う。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetApplicationIcon")
@route("/application-icons/{application_id}/{id}")
@get
op GetApplicationIcon(
  @path application_id: string,
  @path id: string
): GetApplicationIconSuccess_200 | GetApplicationIconError400;

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
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAdminApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminApplicationError403Body;
}

@doc("管理者が Application を削除する (冪等)。関連する protocol row は同一 transaction で cascade delete される。")
@TypeSpec.OpenAPI.operationId("DeleteAdminApplication")
@route("/api/admin/v1/applications/{id}")
@delete
op DeleteAdminApplication(
  @path id: string
): DeleteAdminApplicationSuccess_204 | DeleteAdminApplicationError400 | DeleteAdminApplicationError403;

model UpdateApplicationOidcConfigHttpRequest {
  id: string;
  request: IdMagic.Contract.ApplicationOidcConfigUpdateRequest;
}

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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateApplicationOidcConfigError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationOidcConfigError403Body;
}

@doc("管理者が Application の OIDC protocol が指す OAuth2 client の設定を更新する。rules /\nsub_source_attribute は claim release 上書きであり、visibility=Private の\n属性や reserved claim type を参照する rule は拒否する。")
@TypeSpec.OpenAPI.operationId("UpdateApplicationOidcConfig")
@route("/api/admin/v1/applications/{id}/oidc")
@patch
op UpdateApplicationOidcConfig(
  @path id: string,
  @body body: UpdateApplicationOidcConfigHttpRequest
): UpdateApplicationOidcConfigSuccess_204 | UpdateApplicationOidcConfigError400 | UpdateApplicationOidcConfigError403;

model RotateApplicationClientSecretHttpRequest {
  id: string;
  request: IdMagic.Contract.RotateApplicationClientSecretRequest;
}

model RotateApplicationClientSecretHttpResponse {
  response: IdMagic.Contract.RotateApplicationClientSecretResponse;
}

model RotateApplicationClientSecretSuccess_200 {
  @statusCode statusCode: 200;
  @body body: RotateApplicationClientSecretHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model RotateApplicationClientSecretError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RotateApplicationClientSecretError403Body;
}

@doc("互換性のため、管理者が Application の OIDC protocol の client secret を従来方式で回す。旧 credential は指定した overlap 期間だけ有効にし、0日は即時 revoke とする。新しい管理 UI は IssueApplicationClientSecret と RevokeApplicationClientSecret を使用する。")
@TypeSpec.OpenAPI.operationId("RotateApplicationClientSecret")
@route("/api/admin/v1/applications/{id}/oidc/rotate-secret")
@post
op RotateApplicationClientSecret(
  @path id: string,
  @body body: RotateApplicationClientSecretHttpRequest
): RotateApplicationClientSecretSuccess_200 | RotateApplicationClientSecretError400 | RotateApplicationClientSecretError403;

model IssueApplicationClientSecretHttpRequest {
  id: string;
  request: IdMagic.Contract.IssueApplicationClientSecretRequest;
}

model IssueApplicationClientSecretHttpResponse {
  response: IdMagic.Contract.IssueApplicationClientSecretResponse;
}

model IssueApplicationClientSecretSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IssueApplicationClientSecretHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model IssueApplicationClientSecretError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: IssueApplicationClientSecretError403Body;
}

union IssueApplicationClientSecretError422Body {
  IdMagic.Contract.ClientSecretLimitExceededError,
}

model IssueApplicationClientSecretError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: IssueApplicationClientSecretError422Body;
}

@doc("管理者が Application の secret-based OIDC client に期限付き credential を追加発行する。既存 credential は変更せず、Active credential が2件なら原子的に拒否する。secret-based でない client は拒否する。")
@TypeSpec.OpenAPI.operationId("IssueApplicationClientSecret")
@route("/api/admin/v1/applications/{id}/oidc/client-secrets")
@post
op IssueApplicationClientSecret(
  @path id: string,
  @body body: IssueApplicationClientSecretHttpRequest
): IssueApplicationClientSecretSuccess_201 | IssueApplicationClientSecretError400 | IssueApplicationClientSecretError403 | IssueApplicationClientSecretError422;

model RevokeApplicationClientSecretHttpResponse {
  response: IdMagic.Contract.RevokeApplicationClientSecretResponse;
}

model RevokeApplicationClientSecretSuccess_200 {
  @statusCode statusCode: 200;
  @body body: RevokeApplicationClientSecretHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model RevokeApplicationClientSecretError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeApplicationClientSecretError403Body;
}

@doc("管理者が Application の secret-based OIDC client に属する指定 credential だけを即時失効する。既に失効済みなら冪等に成功し、別 client または存在しない credential は拒否する。")
@TypeSpec.OpenAPI.operationId("RevokeApplicationClientSecret")
@route("/api/admin/v1/applications/{id}/oidc/client-secrets/{credential_id}")
@delete
op RevokeApplicationClientSecret(
  @path id: string,
  @path credential_id: string
): RevokeApplicationClientSecretSuccess_200 | RevokeApplicationClientSecretError400 | RevokeApplicationClientSecretError403;

model UpdateApplicationWsFedConfigHttpRequest {
  id: string;
  request: IdMagic.Contract.ApplicationWsFedConfigUpdateRequest;
}

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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateApplicationWsFedConfigError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationWsFedConfigError403Body;
}

@doc("管理者が Application の WS-Fed protocol が指す relying party の設定を更新する。rules /\nname_id_source は claim release 上書きであり、visibility=Private の属性や\nreserved claim type を参照する rule は拒否する。")
@TypeSpec.OpenAPI.operationId("UpdateApplicationWsFedConfig")
@route("/api/admin/v1/applications/{id}/wsfed")
@patch
op UpdateApplicationWsFedConfig(
  @path id: string,
  @body body: UpdateApplicationWsFedConfigHttpRequest
): UpdateApplicationWsFedConfigSuccess_204 | UpdateApplicationWsFedConfigError400 | UpdateApplicationWsFedConfigError403;

model UpdateApplicationSamlConfigHttpRequest {
  id: string;
  request: IdMagic.Contract.ApplicationSamlConfigUpdateRequest;
}

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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateApplicationSamlConfigError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationSamlConfigError403Body;
}

@doc("管理者が Application の SAML protocol が指す service provider 設定とIdP profile bindingを\n部分更新する。署名要求を有効にする場合は検証可能な証明書を必須とする。rules / name_id_source は\nclaim release 上書きであり、visibility=Private の属性や reserved claim type\nを参照する rule は拒否する。")
@TypeSpec.OpenAPI.operationId("UpdateApplicationSamlConfig")
@route("/api/admin/v1/applications/{id}/saml")
@patch
op UpdateApplicationSamlConfig(
  @path id: string,
  @body body: UpdateApplicationSamlConfigHttpRequest
): UpdateApplicationSamlConfigSuccess_204 | UpdateApplicationSamlConfigError400 | UpdateApplicationSamlConfigError403;

model ListApplicationAssignmentsHttpResponse {
  response: IdMagic.Contract.ApplicationAssignmentListResponse;
}

model ListApplicationAssignmentsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListApplicationAssignmentsHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model ListApplicationAssignmentsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListApplicationAssignmentsError403Body;
}

@doc("管理者が Application の割当を subject_type, subject_id 昇順の双方向 keyset pagination で\n一覧する。cursor は応答の Link response header (rel=\"prev\" / rel=\"next\") から取得する。")
@TypeSpec.OpenAPI.operationId("ListApplicationAssignments")
@route("/api/admin/v1/applications/{id}/assignments")
@get
op ListApplicationAssignments(
  @path id: string,
  @query cursor?: string,
  @query limit?: int64
): ListApplicationAssignmentsSuccess_200 | ListApplicationAssignmentsError400 | ListApplicationAssignmentsError403;

model AssignApplicationHttpRequest {
  id: string;
  request: IdMagic.Contract.ApplicationAssignmentRequest;
}

model AssignApplicationHttpResponse {
  assignment: IdMagic.Contract.ApplicationAssignmentResponse;
}

model AssignApplicationSuccess_201 {
  @statusCode statusCode: 201;
  @body body: AssignApplicationHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model AssignApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: AssignApplicationError403Body;
}

@doc("管理者が Application にユーザー / グループを割当てる。")
@TypeSpec.OpenAPI.operationId("AssignApplication")
@route("/api/admin/v1/applications/{id}/assignments")
@post
op AssignApplication(
  @path id: string,
  @body body: AssignApplicationHttpRequest
): AssignApplicationSuccess_201 | AssignApplicationError400 | AssignApplicationError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model UnassignApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UnassignApplicationError403Body;
}

@doc("管理者が Application の割当を解除する (冪等)。")
@TypeSpec.OpenAPI.operationId("UnassignApplication")
@route("/api/admin/v1/applications/{id}/assignments/{subject_type}/{subject_id}")
@delete
op UnassignApplication(
  @path id: string,
  @path subject_type: string,
  @path subject_id: string
): UnassignApplicationSuccess_204 | UnassignApplicationError400 | UnassignApplicationError403;

model GetAppSignInPolicyHttpResponse {
  response: IdMagic.Contract.AppSignInPolicyResponse;
}

model GetAppSignInPolicySuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAppSignInPolicyHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model GetAppSignInPolicyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAppSignInPolicyError403Body;
}

@doc("管理者が Application の sign-in policy を取得する。アプリ個別ポリシー・テナントデフォルト・上書き後に実際に適用される effective ルール列と、デフォルトより弱いかの警告フラグを返す。未設定なら空 rules を返す。")
@TypeSpec.OpenAPI.operationId("GetAppSignInPolicy")
@route("/api/admin/v1/applications/{id}/sign-in-policy")
@get
op GetAppSignInPolicy(
  @path id: string
): GetAppSignInPolicySuccess_200 | GetAppSignInPolicyError400 | GetAppSignInPolicyError403;

model UpdateAppSignInPolicyHttpRequest {
  id: string;
  request: IdMagic.Contract.AppSignInPolicyRequest;
}

model UpdateAppSignInPolicyHttpResponse {
  response: IdMagic.Contract.AppSignInPolicyResponse;
}

model UpdateAppSignInPolicySuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateAppSignInPolicyHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAppSignInPolicyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAppSignInPolicyError403Body;
}

@doc("管理者が Application の sign-in policy を置き換える。無効な CIDR を含む条件は拒否する。")
@TypeSpec.OpenAPI.operationId("UpdateAppSignInPolicy")
@route("/api/admin/v1/applications/{id}/sign-in-policy")
@put
op UpdateAppSignInPolicy(
  @path id: string,
  @body body: UpdateAppSignInPolicyHttpRequest
): UpdateAppSignInPolicySuccess_200 | UpdateAppSignInPolicyError400 | UpdateAppSignInPolicyError403;

model GetTenantDefaultSignInPolicyHttpResponse {
  response: IdMagic.Contract.TenantDefaultSignInPolicyResponse;
}

model GetTenantDefaultSignInPolicySuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetTenantDefaultSignInPolicyHttpResponse;
}

union GetTenantDefaultSignInPolicyError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetTenantDefaultSignInPolicyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetTenantDefaultSignInPolicyError403Body;
}

@doc("管理者がテナントデフォルト sign-in policy と MFA 未登録 active user 数を取得する。未設定なら空 rules の policy を返す。")
@TypeSpec.OpenAPI.operationId("GetTenantDefaultSignInPolicy")
@route("/api/admin/v1/default-sign-in-policy")
@get
op GetTenantDefaultSignInPolicy(

): GetTenantDefaultSignInPolicySuccess_200 | GetTenantDefaultSignInPolicyError403;

model UpdateTenantDefaultSignInPolicyHttpRequest {
  request: IdMagic.Contract.TenantDefaultSignInPolicyRequest;
}

model UpdateTenantDefaultSignInPolicyHttpResponse {
  response: IdMagic.Contract.TenantDefaultSignInPolicyResponse;
}

model UpdateTenantDefaultSignInPolicySuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateTenantDefaultSignInPolicyHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantDefaultSignInPolicyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantDefaultSignInPolicyError403Body;
}

@doc("管理者がテナントデフォルト sign-in policy を置き換える。MFA 必須化では強制開始日時、enrollment bypass を利用できる猶予、管理者承認の可否を明示する。無効な CIDR、過去の強制開始日時、非正値の猶予は拒否する。空 rules で保存すればデフォルトは allow-all に戻る。")
@TypeSpec.OpenAPI.operationId("UpdateTenantDefaultSignInPolicy")
@route("/api/admin/v1/default-sign-in-policy")
@put
op UpdateTenantDefaultSignInPolicy(
  @body body: UpdateTenantDefaultSignInPolicyHttpRequest
): UpdateTenantDefaultSignInPolicySuccess_200 | UpdateTenantDefaultSignInPolicyError400 | UpdateTenantDefaultSignInPolicyError403;

model ListApplicationCategoriesHttpResponse {
  response: IdMagic.Contract.AdminApplicationCategoryListResponse;
}

model ListApplicationCategoriesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListApplicationCategoriesHttpResponse;
}

union ListApplicationCategoriesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListApplicationCategoriesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListApplicationCategoriesError403Body;
}

@doc("管理者が所属テナントの ApplicationCategory を position 昇順で一覧する。")
@TypeSpec.OpenAPI.operationId("ListApplicationCategories")
@route("/api/admin/v1/application-categories")
@get
op ListApplicationCategories(

): ListApplicationCategoriesSuccess_200 | ListApplicationCategoriesError403;

model CreateApplicationCategoryHttpRequest {
  request: IdMagic.Contract.AdminApplicationCategoryRequest;
}

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
  IdMagic.Contract.AccessDeniedError,
}

model CreateApplicationCategoryError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateApplicationCategoryError403Body;
}

@doc("管理者が ApplicationCategory を作成する。")
@TypeSpec.OpenAPI.operationId("CreateApplicationCategory")
@route("/api/admin/v1/application-categories")
@post
op CreateApplicationCategory(
  @body body: CreateApplicationCategoryHttpRequest
): CreateApplicationCategorySuccess_201 | CreateApplicationCategoryError400 | CreateApplicationCategoryError422 | CreateApplicationCategoryError403;

model UpdateApplicationCategoryHttpRequest {
  id: string;
  request: IdMagic.Contract.AdminApplicationCategoryRequest;
}

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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateApplicationCategoryError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateApplicationCategoryError403Body;
}

@doc("管理者が ApplicationCategory の名前 / 並び順を更新する。")
@TypeSpec.OpenAPI.operationId("UpdateApplicationCategory")
@route("/api/admin/v1/application-categories/{id}")
@patch
op UpdateApplicationCategory(
  @path id: string,
  @body body: UpdateApplicationCategoryHttpRequest
): UpdateApplicationCategorySuccess_200 | UpdateApplicationCategoryError400 | UpdateApplicationCategoryError422 | UpdateApplicationCategoryError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model DeleteApplicationCategoryError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteApplicationCategoryError403Body;
}

@doc("管理者が ApplicationCategory を削除する (冪等)。付与済みの Application からも除く。")
@TypeSpec.OpenAPI.operationId("DeleteApplicationCategory")
@route("/api/admin/v1/application-categories/{id}")
@delete
op DeleteApplicationCategory(
  @path id: string
): DeleteApplicationCategorySuccess_204 | DeleteApplicationCategoryError400 | DeleteApplicationCategoryError403;

model SetApplicationCategoriesHttpRequest {
  id: string;
  request: IdMagic.Contract.ApplicationCategoriesRequest;
}

model SetApplicationCategoriesHttpResponse {
  application: IdMagic.Contract.AdminApplicationResponse;
}

model SetApplicationCategoriesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: SetApplicationCategoriesHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model SetApplicationCategoriesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SetApplicationCategoriesError403Body;
}

@doc("管理者が Application に付与するカテゴリ集合を置き換える。所属テナントの既存カテゴリのみ受け付ける。")
@TypeSpec.OpenAPI.operationId("SetApplicationCategories")
@route("/api/admin/v1/applications/{id}/categories")
@put
op SetApplicationCategories(
  @path id: string,
  @body body: SetApplicationCategoriesHttpRequest
): SetApplicationCategoriesSuccess_200 | SetApplicationCategoriesError400 | SetApplicationCategoriesError422 | SetApplicationCategoriesError403;

model ListMyApplicationsHttpResponse {
  response: IdMagic.Contract.MyApplicationListResponse;
}

model ListMyApplicationsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListMyApplicationsHttpResponse;
}

union ListMyApplicationsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListMyApplicationsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListMyApplicationsError403Body;
}

@doc("利用者が自分に割当済み (visible) の Application を一覧する。hidden 割当は除外する。\n手動並び順 (ApplicationOrdering) があればそれを優先し、無ければ name 昇順。応答の categories は\ntenant のカテゴリ定義を position 昇順で含み、各アプリの category_ids でセクションに振り分ける。")
@TypeSpec.OpenAPI.operationId("ListMyApplications")
@route("/api/account/v1/applications")
@get
op ListMyApplications(

): ListMyApplicationsSuccess_200 | ListMyApplicationsError403;

model GetMyApplicationOrderHttpResponse {
  response: IdMagic.Contract.MyApplicationOrderResponse;
}

model GetMyApplicationOrderSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetMyApplicationOrderHttpResponse;
}

union GetMyApplicationOrderError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetMyApplicationOrderError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetMyApplicationOrderError403Body;
}

@doc("利用者が保存済みのポータル手動並び順を取得する。未保存なら空配列を返す。")
@TypeSpec.OpenAPI.operationId("GetMyApplicationOrder")
@route("/api/account/v1/applications/order")
@get
op GetMyApplicationOrder(

): GetMyApplicationOrderSuccess_200 | GetMyApplicationOrderError403;

model ReorderMyApplicationsHttpRequest {
  request: IdMagic.Contract.ReorderMyApplicationsRequest;
}

model ReorderMyApplicationsHttpResponse {
  response: IdMagic.Contract.MyApplicationOrderResponse;
}

model ReorderMyApplicationsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ReorderMyApplicationsHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model ReorderMyApplicationsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ReorderMyApplicationsError403Body;
}

@doc("利用者が自分のポータル手動並び順を保存する。application_ids は自分に割当済み visible な\nアプリのみを含み、それ以外を含むと拒否する。")
@TypeSpec.OpenAPI.operationId("ReorderMyApplications")
@route("/api/account/v1/applications/order")
@put
op ReorderMyApplications(
  @body body: ReorderMyApplicationsHttpRequest
): ReorderMyApplicationsSuccess_200 | ReorderMyApplicationsError400 | ReorderMyApplicationsError403;
}

}
