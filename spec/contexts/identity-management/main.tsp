import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Identity Management")
namespace IdMagic.IdManagement {

namespace Operations {
model GetEmailVerificationContextHttpResponse {
  csrf_token: string;
}

model GetEmailVerificationContextSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetEmailVerificationContextHttpResponse;
}

@doc("未認証で開かれる email verification 画面が確認 POST を保護する CSRF token を取得する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetEmailVerificationContext")
@route("/api/account/v1/email/verify_context")
@get
op GetEmailVerificationContext(

): GetEmailVerificationContextSuccess_200;

model GetAccountSummaryHttpResponse {
  summary: IdMagic.Contract.AccountSummary;
}

model GetAccountSummarySuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAccountSummaryHttpResponse;
}

union GetAccountSummaryError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAccountSummaryError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAccountSummaryError403Body;
}

@doc("認証済みユーザーが自身のアカウント概要を取得する (self-service portal home)。\n対象は subject.id に固定。roles を含まず、最終ログイン・パスワード変更日時・\nMFA 状態・未対応 required actions を返す。admin shell の AccountContext とは\n別契約。")
@TypeSpec.OpenAPI.operationId("GetAccountSummary")
@route("/api/account/v1/summary")
@get
op GetAccountSummary(

): GetAccountSummarySuccess_200 | GetAccountSummaryError403;

model GetUserProfileHttpResponse {
  profile: IdMagic.Contract.AccountProfileResponse;
}

model GetUserProfileSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetUserProfileHttpResponse;
}

union GetUserProfileError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetUserProfileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetUserProfileError403Body;
}

@doc("認証済みユーザーが自身のプロフィールを取得する (self-service)。対象は\nsubject.id に固定するため cross-user 参照は構造的に発生しない。attributes は\nself が読める分のみ、editable_attributes に編集可能定義を併せて返す。")
@TypeSpec.OpenAPI.operationId("GetUserProfile")
@route("/api/account/v1/profile")
@get
op GetUserProfile(

): GetUserProfileSuccess_200 | GetUserProfileError403;

model UpdateUserProfileHttpRequest {
  request: IdMagic.Contract.AccountProfileUpdateRequest;
}

model UpdateUserProfileHttpResponse {
  profile: IdMagic.Contract.AccountProfileResponse;
}

model UpdateUserProfileSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateUserProfileHttpResponse;
}

union UpdateUserProfileError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateUserProfileError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateUserProfileError400Body;
}

union UpdateUserProfileError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateUserProfileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateUserProfileError403Body;
}

@doc("認証済みユーザーが自身の表示名と editable_by_user=true の属性を更新する\n(self-service)。subject.id == resource.id に固定。属性は key 単位の merge とし、\neditable_by_user=false の key は拒否する。status / roles / organization の変更は\nadmin 専用で本経路では扱わない。")
@TypeSpec.OpenAPI.operationId("UpdateUserProfile")
@route("/api/account/v1/profile")
@patch
op UpdateUserProfile(
  @body body: UpdateUserProfileHttpRequest
): UpdateUserProfileSuccess_200 | UpdateUserProfileError400 | UpdateUserProfileError403;

model RequestEmailChangeHttpRequest {
  request: IdMagic.Contract.AccountEmailChangeRequest;
}

model RequestEmailChangeSuccess_200 {
  @statusCode statusCode: 200;
}

union RequestEmailChangeError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RequestEmailChangeError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RequestEmailChangeError400Body;
}

union RequestEmailChangeError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RequestEmailChangeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RequestEmailChangeError403Body;
}

union RequestEmailChangeError422Body {
  IdMagic.Contract.InvalidEmailError,
  IdMagic.Contract.EmailUnchangedError,
}

model RequestEmailChangeError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: RequestEmailChangeError422Body;
}

@doc("認証済みユーザーが自身の primary email 変更を起票する (self-service)。新アドレスへ\nワンタイムリンクを送るだけで User.email は変えない。subject.id に固定し、tenant 内で\n既に使われているアドレスは拒否する。\n確認メールの文面は Tenancy の通知テンプレートカタログ\n(template_key=EmailChangeConfirmation) から解決し、locale は受信者 User の locale 属性 →\nテナントの default_locale → システム既定 locale の順に決める。テナント上書きがあれば\n組込み既定より優先する。プレーンテキストと HTML の両方を必ず生成し、確認リンクは\nリクエストの発行元 URL から組み立てる。")
@TypeSpec.OpenAPI.operationId("RequestEmailChange")
@route("/api/account/v1/email/change_request")
@post
op RequestEmailChange(
  @body body: RequestEmailChangeHttpRequest
): RequestEmailChangeSuccess_200 | RequestEmailChangeError400 | RequestEmailChangeError403 | RequestEmailChangeError422;

model ConfirmEmailChangeHttpRequest {
  request: IdMagic.Contract.AccountEmailChangeConfirm;
}

model ConfirmEmailChangeSuccess_200 {
  @statusCode statusCode: 200;
}

union ConfirmEmailChangeError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ConfirmEmailChangeError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ConfirmEmailChangeError400Body;
}

@doc("新アドレスへ送ったワンタイムリンクを消費して primary email を確定する。\ntoken が新アドレス所有の証左なので認証済みセッションは要求しない。token の\n所持そのものが認可の根拠であり (self-service セッションではなく\ntoken-possession-authenticated)、AuthenticatedSelf/session ベースの access 分類は\n適用しない。確定時に email_verified=true とし、verify_email の required action が\nあれば自動解除する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ConfirmEmailChange")
@route("/api/account/v1/email/verify")
@post
op ConfirmEmailChange(
  @body body: ConfirmEmailChangeHttpRequest
): ConfirmEmailChangeSuccess_200 | ConfirmEmailChangeError400;

model ExportAccountDataHttpResponse {
  export: IdMagic.Contract.AccountDataExport;
}

model ExportAccountDataSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ExportAccountDataHttpResponse;
}

union ExportAccountDataError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ExportAccountDataError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ExportAccountDataError403Body;
}

@doc("認証済みユーザーが自身の個人データを JSON で取得する (self-service, GDPR access)。\n対象は subject.id に固定。本ステージは同期生成で profile と consents を含む。")
@TypeSpec.OpenAPI.operationId("ExportAccountData")
@route("/api/account/v1/data_export")
@get
op ExportAccountData(

): ExportAccountDataSuccess_200 | ExportAccountDataError403;

model ListAdminUsersHttpResponse {
  response: IdMagic.Contract.AdminUserListResponse;
}

model ListAdminUsersSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListAdminUsersHttpResponse;
}

union ListAdminUsersError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminUsersError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminUsersError403Body;
}

union ListAdminUsersError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListAdminUsersError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListAdminUsersError400Body;
}

@doc("管理者が削除されていないユーザーを preferred_username 昇順 (同値は id で tie-break) の\n双方向 keyset pagination で一覧する。query は username、name、email、id、roles を tenant 全体で\ncase-insensitive contains 検索し、status とともに絞り込む。cursor は応答の Link response header\n(rel=\"prev\" / rel=\"next\") から取得する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("ListAdminUsers")
@route("/api/admin/v1/users")
@get
op ListAdminUsers(
  @query query?: string,
  @query status?: IdMagic.Contract.UserStatus,
  @query cursor?: string,
  @query limit?: int64
): ListAdminUsersSuccess_200 | ListAdminUsersError403 | ListAdminUsersError400;

model GetAdminUserHttpResponse {
  user: IdMagic.Contract.AdminUserResponse;
}

model GetAdminUserSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAdminUserHttpResponse;
}

union GetAdminUserError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetAdminUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetAdminUserError400Body;
}

union GetAdminUserError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminUserError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminUserError403Body;
}

@doc("管理者が id を指定してユーザーを取得する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("GetAdminUser")
@route("/api/admin/v1/users/{user_id}")
@get
op GetAdminUser(
  @path user_id: string
): GetAdminUserSuccess_200 | GetAdminUserError400 | GetAdminUserError403;

model CreateAdminUserHttpRequest {
  request: IdMagic.Contract.AdminUserCreateRequest;
}

model CreateAdminUserHttpResponse {
  user: IdMagic.Contract.AdminUserResponse;
}

model CreateAdminUserSuccess_201 {
  @statusCode statusCode: 201;
  @body body: CreateAdminUserHttpResponse;
}

union CreateAdminUserError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateAdminUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateAdminUserError400Body;
}

union CreateAdminUserError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CreateAdminUserError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateAdminUserError403Body;
}

union CreateAdminUserError422Body {
  IdMagic.Contract.QuotaExceededError,
  IdMagic.Contract.InvalidRoleError,
}

model CreateAdminUserError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: CreateAdminUserError422Body;
}

@doc("管理者が Authentication context の PasswordPolicy を満たす初期パスワードでユーザーを作成する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("CreateAdminUser")
@route("/api/admin/v1/users")
@post
op CreateAdminUser(
  @body body: CreateAdminUserHttpRequest
): CreateAdminUserSuccess_201 | CreateAdminUserError400 | CreateAdminUserError403 | CreateAdminUserError422;

model ImportAdminUsersHttpRequest {
  csv: string;
  @doc("旧 JSON クライアント移行用。新しい streaming preview では Content-Type と endpoint semantics が preview を固定する。")
  mode?: IdMagic.Contract.UserImportMode;
}

model ImportAdminUsersHttpResponse {
  job: IdMagic.Contract.UserImportJob;
}

model ImportAdminUsersSuccess_202 {
  @statusCode statusCode: 202;
  @body body: ImportAdminUsersHttpResponse;
}

union ImportAdminUsersError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ImportAdminUsersError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ImportAdminUsersError400Body;
}

union ImportAdminUsersError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ImportAdminUsersError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ImportAdminUsersError403Body;
}

@doc("管理者が UTF-8 (BOM 可) CSV を streaming upload して副作用のない preview ジョブへ投入する。User export と共有する configurable transfer policy の既定は 100,000 data rows・64 MiB/file・64 KiB/field。payload は immutable artifact store に保存し、job params には tenant-scoped reference・server-computed SHA-256・size だけを保持する。ヘッダーは安定した機械キーの任意順・任意部分集合を許可し、未知・重複・password/hash 列をファイル単位で拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("ImportAdminUsers")
@route("/api/admin/v1/users/imports")
@post
op ImportAdminUsers(
  @body body: ImportAdminUsersHttpRequest
): ImportAdminUsersSuccess_202 | ImportAdminUsersError400 | ImportAdminUsersError403;

model ApplyAdminUserImportHttpRequest {
  preview_job_id: string;
}

model ApplyAdminUserImportHttpResponse {
  job: IdMagic.Contract.UserImportJob;
}

model ApplyAdminUserImportSuccess_202 {
  @statusCode statusCode: 202;
  @body body: ApplyAdminUserImportHttpResponse;
}

union ApplyAdminUserImportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ApplyAdminUserImportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ApplyAdminUserImportError400Body;
}

union ApplyAdminUserImportError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ApplyAdminUserImportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ApplyAdminUserImportError403Body;
}

@doc("管理者が同一 tenant の成功済み preview job を指定して CSV を適用する。CSV は再送せず、preview payload と source_sha256 を読み、現在の repository 状態に対して同じ deterministic planner で再計画する。ファイル全体は行単位部分成功、各行の profile・roles・required actions・custom attributes は単一 aggregate mutation として原子的に確定する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("ApplyAdminUserImport")
@route("/api/admin/v1/users/imports/{preview_job_id}/apply")
@post
op ApplyAdminUserImport(
  @path preview_job_id: string,
  @body body: ApplyAdminUserImportHttpRequest
): ApplyAdminUserImportSuccess_202 | ApplyAdminUserImportError400 | ApplyAdminUserImportError403;

model GetAdminUserImportHttpResponse {
  job: IdMagic.Contract.UserImportJob;
}

model GetAdminUserImportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAdminUserImportHttpResponse;
}

union GetAdminUserImportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetAdminUserImportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetAdminUserImportError400Body;
}

union GetAdminUserImportError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminUserImportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminUserImportError403Body;
}

@doc("管理者が同一 tenant の CSV インポートジョブと行ごとの検証・適用結果を取得する。row error は immutable result artifact に固定件数の page chunk として保存し、tenant・job・error ordinal に結合した署名付き opaque cursor と limit（既定 100、最大 200）で双方向取得する。job result JSON は操作別件数、result artifact metadata、error_total だけを保持する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("GetAdminUserImport")
@route("/api/admin/v1/users/imports/{job_id}")
@get
op GetAdminUserImport(
  @path job_id: string,
  @query cursor?: string,
  @query limit?: int64
): GetAdminUserImportSuccess_200 | GetAdminUserImportError400 | GetAdminUserImportError403;

model StartUserCsvExportHttpRequest {
  request: IdMagic.Contract.DataExportRequest;
}

model StartUserCsvExportHttpResponse {
  job: IdMagic.Contract.DataExportJob;
}

model StartUserCsvExportSuccess_202 {
  @statusCode statusCode: 202;
  @body body: StartUserCsvExportHttpResponse;
}

union StartUserCsvExportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model StartUserCsvExportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: StartUserCsvExportError400Body;
}

union StartUserCsvExportError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model StartUserCsvExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartUserCsvExportError403Body;
}

union StartUserCsvExportError422Body {
  IdMagic.Contract.InvalidExportColumnsError,
  IdMagic.Contract.InvalidExportTargetError,
  IdMagic.Contract.InvalidExportFilterError,
}

model StartUserCsvExportError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: StartUserCsvExportError422Body;
}

@doc("管理者が User 一覧の CSV エクスポートを非同期ジョブとして開始する。CSV ヘッダーは表示 label ではなく要求した machine key と一致する。列は組み込み User allowlist (13 個のコア列 + attr:<key> の組み込み拡張属性 27 種) と実効 TenantUserAttributeSchema の custom:<key> (テナント定義属性) の部分集合に限り、status フィルタを引き継げる。生成 payload は import と同じ UserCsvTransferPolicy と immutable artifact store を使うため、成功した User export は無編集で preview できる。202 とエクスポート id を返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("StartUserCsvExport")
@route("/api/admin/v1/users/exports")
@post
op StartUserCsvExport(
  @body body: StartUserCsvExportHttpRequest
): StartUserCsvExportSuccess_202 | StartUserCsvExportError400 | StartUserCsvExportError403 | StartUserCsvExportError422;

model ListUserExportsHttpResponse {
  exports: IdMagic.Contract.DataExportJob[];
}

model ListUserExportsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListUserExportsHttpResponse;
}

union ListUserExportsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListUserExportsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListUserExportsError403Body;
}

@doc("管理者が User エクスポート一覧を、状態・件数・期限・ダウンロード可否とともに取得する。CSV 本体や PII 値は含まない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("ListUserExports")
@route("/api/admin/v1/users/exports")
@get
op ListUserExports(

): ListUserExportsSuccess_200 | ListUserExportsError403;

model GetUserExportHttpResponse {
  job: IdMagic.Contract.DataExportJob;
}

model GetUserExportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetUserExportHttpResponse;
}

union GetUserExportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetUserExportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetUserExportError400Body;
}

union GetUserExportError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetUserExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetUserExportError403Body;
}

@doc("管理者が User エクスポート 1 件の状態・進捗・失敗理由・期限を取得する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("GetUserExport")
@route("/api/admin/v1/users/exports/{export_id}")
@get
op GetUserExport(
  @path export_id: string
): GetUserExportSuccess_200 | GetUserExportError400 | GetUserExportError403;

model DownloadUserExportFileHttpResponse {
  file: IdMagic.Contract.DataExportFile;
}

model DownloadUserExportFileSuccess_200 {
  @statusCode statusCode: 200;
  @body body: DownloadUserExportFileHttpResponse;
}

union DownloadUserExportFileError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DownloadUserExportFileError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DownloadUserExportFileError400Body;
}

union DownloadUserExportFileError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DownloadUserExportFileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DownloadUserExportFileError403Body;
}

@doc("管理者が完成済み User エクスポートを content-disposition attachment でダウンロードする。succeeded かつ未期限のジョブに限る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("DownloadUserExportFile")
@route("/api/admin/v1/users/exports/{export_id}/file")
@get
op DownloadUserExportFile(
  @path export_id: string
): DownloadUserExportFileSuccess_200 | DownloadUserExportFileError400 | DownloadUserExportFileError403;

model CancelUserExportHttpRequest {
  export_id: string;
}

model CancelUserExportHttpResponse {
  job: IdMagic.Contract.DataExportJob;
}

model CancelUserExportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: CancelUserExportHttpResponse;
}

union CancelUserExportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CancelUserExportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CancelUserExportError400Body;
}

union CancelUserExportError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CancelUserExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CancelUserExportError403Body;
}

@doc("管理者が終端前の User エクスポートを取消す。既に終端の場合は拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("CancelUserExport")
@route("/api/admin/v1/users/exports/{export_id}/cancel")
@post
op CancelUserExport(
  @path export_id: string,
  @body body: CancelUserExportHttpRequest
): CancelUserExportSuccess_200 | CancelUserExportError400 | CancelUserExportError403;

model StartGroupCsvExportHttpRequest {
  request: IdMagic.Contract.DataExportRequest;
}

model StartGroupCsvExportHttpResponse {
  job: IdMagic.Contract.DataExportJob;
}

model StartGroupCsvExportSuccess_202 {
  @statusCode statusCode: 202;
  @body body: StartGroupCsvExportHttpResponse;
}

union StartGroupCsvExportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model StartGroupCsvExportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: StartGroupCsvExportError400Body;
}

union StartGroupCsvExportError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model StartGroupCsvExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartGroupCsvExportError403Body;
}

union StartGroupCsvExportError422Body {
  IdMagic.Contract.InvalidExportColumnsError,
  IdMagic.Contract.InvalidExportTargetError,
  IdMagic.Contract.InvalidExportFilterError,
}

model StartGroupCsvExportError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: StartGroupCsvExportError422Body;
}

@doc("管理者が Group 一覧の CSV エクスポートを非同期ジョブとして開始する。列は Group allowlist の部分集合に限る。202 とエクスポート id を返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("StartGroupCsvExport")
@route("/api/admin/v1/groups/exports")
@post
op StartGroupCsvExport(
  @body body: StartGroupCsvExportHttpRequest
): StartGroupCsvExportSuccess_202 | StartGroupCsvExportError400 | StartGroupCsvExportError403 | StartGroupCsvExportError422;

model ListGroupExportsHttpResponse {
  exports: IdMagic.Contract.DataExportJob[];
}

model ListGroupExportsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListGroupExportsHttpResponse;
}

union ListGroupExportsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListGroupExportsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListGroupExportsError403Body;
}

@doc("管理者が Group エクスポート一覧を、状態・件数・期限・ダウンロード可否とともに取得する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("ListGroupExports")
@route("/api/admin/v1/groups/exports")
@get
op ListGroupExports(

): ListGroupExportsSuccess_200 | ListGroupExportsError403;

model GetGroupExportHttpResponse {
  job: IdMagic.Contract.DataExportJob;
}

model GetGroupExportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetGroupExportHttpResponse;
}

union GetGroupExportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetGroupExportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetGroupExportError400Body;
}

union GetGroupExportError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetGroupExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetGroupExportError403Body;
}

@doc("管理者が Group エクスポート 1 件の状態・進捗・失敗理由・期限を取得する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("GetGroupExport")
@route("/api/admin/v1/groups/exports/{export_id}")
@get
op GetGroupExport(
  @path export_id: string
): GetGroupExportSuccess_200 | GetGroupExportError400 | GetGroupExportError403;

model DownloadGroupExportFileHttpResponse {
  file: IdMagic.Contract.DataExportFile;
}

model DownloadGroupExportFileSuccess_200 {
  @statusCode statusCode: 200;
  @body body: DownloadGroupExportFileHttpResponse;
}

union DownloadGroupExportFileError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DownloadGroupExportFileError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DownloadGroupExportFileError400Body;
}

union DownloadGroupExportFileError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DownloadGroupExportFileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DownloadGroupExportFileError403Body;
}

@doc("管理者が完成済み Group エクスポートを content-disposition attachment でダウンロードする。succeeded かつ未期限のジョブに限る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("DownloadGroupExportFile")
@route("/api/admin/v1/groups/exports/{export_id}/file")
@get
op DownloadGroupExportFile(
  @path export_id: string
): DownloadGroupExportFileSuccess_200 | DownloadGroupExportFileError400 | DownloadGroupExportFileError403;

model CancelGroupExportHttpRequest {
  export_id: string;
}

model CancelGroupExportHttpResponse {
  job: IdMagic.Contract.DataExportJob;
}

model CancelGroupExportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: CancelGroupExportHttpResponse;
}

union CancelGroupExportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CancelGroupExportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CancelGroupExportError400Body;
}

union CancelGroupExportError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CancelGroupExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CancelGroupExportError403Body;
}

@doc("管理者が終端前の Group エクスポートを取消す。既に終端の場合は拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("CancelGroupExport")
@route("/api/admin/v1/groups/exports/{export_id}/cancel")
@post
op CancelGroupExport(
  @path export_id: string,
  @body body: CancelGroupExportHttpRequest
): CancelGroupExportSuccess_200 | CancelGroupExportError400 | CancelGroupExportError403;

model StartGroupMemberCsvExportHttpRequest {
  group_id: string;
  request: IdMagic.Contract.DataExportRequest;
}

model StartGroupMemberCsvExportHttpResponse {
  job: IdMagic.Contract.DataExportJob;
}

model StartGroupMemberCsvExportSuccess_202 {
  @statusCode statusCode: 202;
  @body body: StartGroupMemberCsvExportHttpResponse;
}

union StartGroupMemberCsvExportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model StartGroupMemberCsvExportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: StartGroupMemberCsvExportError400Body;
}

union StartGroupMemberCsvExportError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model StartGroupMemberCsvExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartGroupMemberCsvExportError403Body;
}

union StartGroupMemberCsvExportError422Body {
  IdMagic.Contract.InvalidExportColumnsError,
  IdMagic.Contract.InvalidExportTargetError,
  IdMagic.Contract.InvalidExportFilterError,
}

model StartGroupMemberCsvExportError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: StartGroupMemberCsvExportError422Body;
}

@doc("管理者が特定 Group のメンバー一覧の CSV エクスポートを開始する。group_id は path から取り、そのグループのメンバーだけを対象とする (Entra / Okta / Google と同じ per-group)。202 とエクスポート id を返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("StartGroupMemberCsvExport")
@route("/api/admin/v1/groups/{group_id}/members/exports")
@post
op StartGroupMemberCsvExport(
  @path group_id: string,
  @body body: StartGroupMemberCsvExportHttpRequest
): StartGroupMemberCsvExportSuccess_202 | StartGroupMemberCsvExportError400 | StartGroupMemberCsvExportError403 | StartGroupMemberCsvExportError422;

model ListGroupMemberExportsHttpResponse {
  exports: IdMagic.Contract.DataExportJob[];
}

model ListGroupMemberExportsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListGroupMemberExportsHttpResponse;
}

union ListGroupMemberExportsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListGroupMemberExportsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListGroupMemberExportsError403Body;
}

@doc("管理者が特定 Group のメンバーエクスポート一覧を取得する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("ListGroupMemberExports")
@route("/api/admin/v1/groups/{group_id}/members/exports")
@get
op ListGroupMemberExports(
  @path group_id: string
): ListGroupMemberExportsSuccess_200 | ListGroupMemberExportsError403;

model GetGroupMemberExportHttpResponse {
  job: IdMagic.Contract.DataExportJob;
}

model GetGroupMemberExportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetGroupMemberExportHttpResponse;
}

union GetGroupMemberExportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetGroupMemberExportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetGroupMemberExportError400Body;
}

union GetGroupMemberExportError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetGroupMemberExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetGroupMemberExportError403Body;
}

@doc("管理者が特定 Group のメンバーエクスポート 1 件の状態を取得する。別グループの export id は解決しない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("GetGroupMemberExport")
@route("/api/admin/v1/groups/{group_id}/members/exports/{export_id}")
@get
op GetGroupMemberExport(
  @path group_id: string,
  @path export_id: string
): GetGroupMemberExportSuccess_200 | GetGroupMemberExportError400 | GetGroupMemberExportError403;

model DownloadGroupMemberExportFileHttpResponse {
  file: IdMagic.Contract.DataExportFile;
}

model DownloadGroupMemberExportFileSuccess_200 {
  @statusCode statusCode: 200;
  @body body: DownloadGroupMemberExportFileHttpResponse;
}

union DownloadGroupMemberExportFileError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DownloadGroupMemberExportFileError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DownloadGroupMemberExportFileError400Body;
}

union DownloadGroupMemberExportFileError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DownloadGroupMemberExportFileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DownloadGroupMemberExportFileError403Body;
}

@doc("管理者が完成済みメンバーエクスポートを content-disposition attachment でダウンロードする。succeeded かつ未期限のジョブに限る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("DownloadGroupMemberExportFile")
@route("/api/admin/v1/groups/{group_id}/members/exports/{export_id}/file")
@get
op DownloadGroupMemberExportFile(
  @path group_id: string,
  @path export_id: string
): DownloadGroupMemberExportFileSuccess_200 | DownloadGroupMemberExportFileError400 | DownloadGroupMemberExportFileError403;

model CancelGroupMemberExportHttpRequest {
  group_id: string;
  export_id: string;
}

model CancelGroupMemberExportHttpResponse {
  job: IdMagic.Contract.DataExportJob;
}

model CancelGroupMemberExportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: CancelGroupMemberExportHttpResponse;
}

union CancelGroupMemberExportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CancelGroupMemberExportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CancelGroupMemberExportError400Body;
}

union CancelGroupMemberExportError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CancelGroupMemberExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CancelGroupMemberExportError403Body;
}

@doc("管理者が終端前のメンバーエクスポートを取消す。既に終端の場合は拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("CancelGroupMemberExport")
@route("/api/admin/v1/groups/{group_id}/members/exports/{export_id}/cancel")
@post
op CancelGroupMemberExport(
  @path group_id: string,
  @path export_id: string,
  @body body: CancelGroupMemberExportHttpRequest
): CancelGroupMemberExportSuccess_200 | CancelGroupMemberExportError400 | CancelGroupMemberExportError403;

model UpdateAdminUserHttpRequest {
  user_id: string;
  request: IdMagic.Contract.AdminUserUpdateRequest;
}

model UpdateAdminUserHttpResponse {
  user: IdMagic.Contract.AdminUserResponse;
}

model UpdateAdminUserSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateAdminUserHttpResponse;
}

union UpdateAdminUserError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateAdminUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminUserError400Body;
}

union UpdateAdminUserError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAdminUserError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminUserError403Body;
}

union UpdateAdminUserError422Body {
  IdMagic.Contract.InvalidRoleError,
  IdMagic.Contract.InvalidUserAttributeError,
}

model UpdateAdminUserError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminUserError422Body;
}

@doc("管理者がユーザーのプロフィール、email_verified、roles を更新する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("UpdateAdminUser")
@route("/api/admin/v1/users/{user_id}")
@patch
op UpdateAdminUser(
  @path user_id: string,
  @body body: UpdateAdminUserHttpRequest
): UpdateAdminUserSuccess_200 | UpdateAdminUserError400 | UpdateAdminUserError403 | UpdateAdminUserError422;

model DisableAdminUserHttpRequest {
  user_id: string;
}

model DisableAdminUserSuccess_204 {
  @statusCode statusCode: 204;
}

union DisableAdminUserError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DisableAdminUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DisableAdminUserError400Body;
}

union DisableAdminUserError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DisableAdminUserError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableAdminUserError403Body;
}

union DisableAdminUserError422Body {
  IdMagic.Contract.SelfDisableForbiddenError,
}

model DisableAdminUserError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: DisableAdminUserError422Body;
}

@doc("管理者がユーザーを無効化し、新規ログインと既存セッション利用を拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("DisableAdminUser")
@route("/api/admin/v1/users/{user_id}/disable")
@post
op DisableAdminUser(
  @path user_id: string,
  @body body: DisableAdminUserHttpRequest
): DisableAdminUserSuccess_204 | DisableAdminUserError400 | DisableAdminUserError403 | DisableAdminUserError422;

model EnableAdminUserHttpRequest {
  user_id: string;
}

model EnableAdminUserSuccess_204 {
  @statusCode statusCode: 204;
}

union EnableAdminUserError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model EnableAdminUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: EnableAdminUserError400Body;
}

union EnableAdminUserError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model EnableAdminUserError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableAdminUserError403Body;
}

@doc("管理者が無効化済みユーザーを再有効化する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("EnableAdminUser")
@route("/api/admin/v1/users/{user_id}/enable")
@post
op EnableAdminUser(
  @path user_id: string,
  @body body: EnableAdminUserHttpRequest
): EnableAdminUserSuccess_204 | EnableAdminUserError400 | EnableAdminUserError403;

model DeleteAdminUserHttpRequest {
  user_id: string;
  @doc("true で即時 anonymize cascade (完全削除)。省略時は soft-delete。")
  purge?: boolean;
  @doc("監査用の任意自由記述。")
  reason?: string;
}

model DeleteAdminUserSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteAdminUserError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteAdminUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminUserError400Body;
}

union DeleteAdminUserError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAdminUserError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminUserError403Body;
}

union DeleteAdminUserError422Body {
  IdMagic.Contract.SelfDeleteForbiddenError,
}

model DeleteAdminUserError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminUserError422Body;
}

@doc("管理者がユーザーを削除する。既定は SoftDelete で、status を\nPendingDeletion に遷移させ PII / Consent / RefreshToken / Session を温存する\n(猶予期間内は Restore 可能)。`purge=true` の場合は anonymize\ncascade を即時実施し tombstone 化する。いずれも冪等で、既に同じ終着状態の\nuser への再呼び出しは no-op として 204 を返す。自分自身は削除できない\n(admin / system_admin の自爆防止)。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("DeleteAdminUser")
@route("/api/admin/v1/users/{user_id}")
@delete
op DeleteAdminUser(
  @path user_id: string,
  @body body: DeleteAdminUserHttpRequest
): DeleteAdminUserSuccess_204 | DeleteAdminUserError400 | DeleteAdminUserError403 | DeleteAdminUserError422;

model RestoreAdminUserHttpRequest {
  user_id: string;
}

model RestoreAdminUserHttpResponse {
  user: IdMagic.Contract.AdminUserResponse;
}

model RestoreAdminUserSuccess_200 {
  @statusCode statusCode: 200;
  @body body: RestoreAdminUserHttpResponse;
}

union RestoreAdminUserError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RestoreAdminUserError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RestoreAdminUserError400Body;
}

union RestoreAdminUserError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RestoreAdminUserError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RestoreAdminUserError403Body;
}

union RestoreAdminUserError409Body {
  IdMagic.Contract.UserNotPendingDeletionError,
  IdMagic.Contract.RestoreGracePeriodExpiredError,
}

model RestoreAdminUserError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: RestoreAdminUserError409Body;
}

@doc("管理者が PendingDeletion のユーザーを Restore する。status を Active\nに戻し PII / credential を温存したまま復元する。猶予期間を過ぎている、または\nPendingDeletion でない user に対しては InvalidRequestError を返す。復元後の\nuser を返す。自分自身への操作は use case 側で reject する (自爆防止と対称の\n扱い)。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("RestoreAdminUser")
@route("/api/admin/v1/users/{user_id}/restore")
@post
op RestoreAdminUser(
  @path user_id: string,
  @body body: RestoreAdminUserHttpRequest
): RestoreAdminUserSuccess_200 | RestoreAdminUserError400 | RestoreAdminUserError403 | RestoreAdminUserError409;

model SetUserRequiredActionHttpRequest {
  user_id: string;
  request: IdMagic.Contract.AdminRequiredActionRequest;
}

model SetUserRequiredActionHttpResponse {
  user: IdMagic.Contract.AdminUserResponse;
}

model SetUserRequiredActionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: SetUserRequiredActionHttpResponse;
}

union SetUserRequiredActionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model SetUserRequiredActionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: SetUserRequiredActionError400Body;
}

union SetUserRequiredActionError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model SetUserRequiredActionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SetUserRequiredActionError403Body;
}

union SetUserRequiredActionError422Body {
  IdMagic.Contract.InvalidRequiredActionError,
}

model SetUserRequiredActionError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: SetUserRequiredActionError422Body;
}

@doc("管理者が対象ユーザーに次回ログイン時の強制アクション (RequiredAction) を\n付与する。update_password を付与されたユーザーは次回ログイン時に\nchange-password 画面へ強制誘導される (本人の変更で自動解除)。冪等で、既に\n付与済みなら現状を返す。admin 専用。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("SetUserRequiredAction")
@route("/api/admin/v1/users/{user_id}/required_actions")
@post
op SetUserRequiredAction(
  @path user_id: string,
  @body body: SetUserRequiredActionHttpRequest
): SetUserRequiredActionSuccess_200 | SetUserRequiredActionError400 | SetUserRequiredActionError403 | SetUserRequiredActionError422;

model ClearUserRequiredActionHttpResponse {
  user: IdMagic.Contract.AdminUserResponse;
}

model ClearUserRequiredActionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ClearUserRequiredActionHttpResponse;
}

union ClearUserRequiredActionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ClearUserRequiredActionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ClearUserRequiredActionError400Body;
}

union ClearUserRequiredActionError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ClearUserRequiredActionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ClearUserRequiredActionError403Body;
}

@doc("管理者が対象ユーザーの強制アクションを解除する。冪等で、未付与なら\n現状を返す。admin 専用。本人のパスワード変更に伴う update_password の自動\n解除は ChangePassword / ResetPasswordWithToken 側で発火する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("ClearUserRequiredAction")
@route("/api/admin/v1/users/{user_id}/required_actions/{action}")
@delete
op ClearUserRequiredAction(
  @path user_id: string,
  @path action: string
): ClearUserRequiredActionSuccess_200 | ClearUserRequiredActionError400 | ClearUserRequiredActionError403;

model ListGroupsHttpResponse {
  response: IdMagic.Contract.GroupListResponse;
}

model ListGroupsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListGroupsHttpResponse;
}

union ListGroupsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListGroupsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListGroupsError403Body;
}

union ListGroupsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListGroupsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListGroupsError400Body;
}

@doc("管理者が所属テナントのグループを name 昇順 (同値は id で tie-break) の双方向 keyset\npagination で一覧する。各グループは member_count を含む。cursor は応答の Link response header\n(rel=\"prev\" / rel=\"next\") から取得する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("ListGroups")
@route("/api/admin/v1/groups")
@get
op ListGroups(
  @query cursor?: string,
  @query limit?: int64
): ListGroupsSuccess_200 | ListGroupsError403 | ListGroupsError400;

model GetGroupHttpResponse {
  group: IdMagic.Contract.GroupSummaryResponse;
  members: IdMagic.Contract.GroupMemberResponse[];
}

model GetGroupSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetGroupHttpResponse;
}

union GetGroupError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetGroupError400Body;
}

union GetGroupError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetGroupError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetGroupError403Body;
}

@doc("管理者が単一グループとそのメンバー一覧を取得する。別テナントのグループは未存在として扱う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("GetGroup")
@route("/api/admin/v1/groups/{group_id}")
@get
op GetGroup(
  @path group_id: string
): GetGroupSuccess_200 | GetGroupError400 | GetGroupError403;

model CreateGroupHttpRequest {
  request: IdMagic.Contract.GroupCreateRequest;
}

model CreateGroupHttpResponse {
  group: IdMagic.Contract.GroupSummaryResponse;
}

model CreateGroupSuccess_201 {
  @statusCode statusCode: 201;
  @body body: CreateGroupHttpResponse;
}

union CreateGroupError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateGroupError400Body;
}

union CreateGroupError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CreateGroupError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateGroupError403Body;
}

union CreateGroupError422Body {
  IdMagic.Contract.GroupNameRequiredError,
  IdMagic.Contract.InvalidRoleError,
  IdMagic.Contract.InvalidDynamicGroupRuleError,
  IdMagic.Contract.QuotaExceededError,
  IdMagic.Contract.InvalidEmailError,
  IdMagic.Contract.InvalidGroupAttributeError,
}

model CreateGroupError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: CreateGroupError422Body;
}

@doc("管理者が所属テナントにグループを作成する。name はテナント内で一意。email は形式検証のみ (一意制約なし)、attributes はテナントの TenantGroupAttributeSchema に対して検証する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("CreateGroup")
@route("/api/admin/v1/groups")
@post
op CreateGroup(
  @body body: CreateGroupHttpRequest
): CreateGroupSuccess_201 | CreateGroupError400 | CreateGroupError403 | CreateGroupError422;

model UpdateGroupHttpRequest {
  group_id: string;
  request: IdMagic.Contract.GroupUpdateRequest;
}

model UpdateGroupHttpResponse {
  group: IdMagic.Contract.GroupSummaryResponse;
}

model UpdateGroupSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateGroupHttpResponse;
}

union UpdateGroupError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateGroupError400Body;
}

union UpdateGroupError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateGroupError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateGroupError403Body;
}

union UpdateGroupError422Body {
  IdMagic.Contract.GroupNameRequiredError,
  IdMagic.Contract.InvalidRoleError,
  IdMagic.Contract.InvalidEmailError,
  IdMagic.Contract.InvalidGroupAttributeError,
}

model UpdateGroupError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateGroupError422Body;
}

@doc("管理者がグループの name / description / email / attributes / roles を更新する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("UpdateGroup")
@route("/api/admin/v1/groups/{group_id}")
@patch
op UpdateGroup(
  @path group_id: string,
  @body body: UpdateGroupHttpRequest
): UpdateGroupSuccess_200 | UpdateGroupError400 | UpdateGroupError403 | UpdateGroupError422;

model DeleteGroupSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteGroupError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteGroupError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteGroupError400Body;
}

union DeleteGroupError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteGroupError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteGroupError403Body;
}

@doc("管理者がグループを削除する。所属 membership は cascade で解除する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("DeleteGroup")
@route("/api/admin/v1/groups/{group_id}")
@delete
op DeleteGroup(
  @path group_id: string
): DeleteGroupSuccess_204 | DeleteGroupError400 | DeleteGroupError403;

model AddGroupMemberHttpRequest {
  group_id: string;
  user_id: string;
}

model AddGroupMemberSuccess_204 {
  @statusCode statusCode: 204;
}

union AddGroupMemberError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model AddGroupMemberError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: AddGroupMemberError400Body;
}

union AddGroupMemberError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model AddGroupMemberError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: AddGroupMemberError403Body;
}

@doc("管理者が User をグループに所属させる。同一テナントの User のみ許可し、既所属なら no-op (冪等)。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("AddGroupMember")
@route("/api/admin/v1/groups/{group_id}/members/{user_id}")
@post
op AddGroupMember(
  @path group_id: string,
  @path user_id: string,
  @body body: AddGroupMemberHttpRequest
): AddGroupMemberSuccess_204 | AddGroupMemberError400 | AddGroupMemberError403;

model RemoveGroupMemberSuccess_204 {
  @statusCode statusCode: 204;
}

union RemoveGroupMemberError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RemoveGroupMemberError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RemoveGroupMemberError400Body;
}

union RemoveGroupMemberError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RemoveGroupMemberError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RemoveGroupMemberError403Body;
}

@doc("管理者が User をグループから外す。非所属なら no-op (冪等)。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("RemoveGroupMember")
@route("/api/admin/v1/groups/{group_id}/members/{user_id}")
@delete
op RemoveGroupMember(
  @path group_id: string,
  @path user_id: string
): RemoveGroupMemberSuccess_204 | RemoveGroupMemberError400 | RemoveGroupMemberError403;

model UpdateDynamicGroupRuleHttpRequest {
  group_id: string;
  request: IdMagic.Contract.DynamicGroupRuleRequest;
}

model UpdateDynamicGroupRuleHttpResponse {
  rule: IdMagic.Contract.DynamicGroupRule;
}

model UpdateDynamicGroupRuleSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateDynamicGroupRuleHttpResponse;
}

union UpdateDynamicGroupRuleError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateDynamicGroupRuleError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateDynamicGroupRuleError400Body;
}

union UpdateDynamicGroupRuleError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateDynamicGroupRuleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateDynamicGroupRuleError403Body;
}

union UpdateDynamicGroupRuleError422Body {
  IdMagic.Contract.InvalidDynamicGroupRuleError,
}

model UpdateDynamicGroupRuleError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateDynamicGroupRuleError422Body;
}

@doc("管理者が dynamic group の制限 CEL rule を検証して保存する。未知属性、型不一致、非 Boolean、許可外構文、上限超過は拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("UpdateDynamicGroupRule")
@route("/api/admin/v1/groups/{group_id}/dynamic-rule")
@put
op UpdateDynamicGroupRule(
  @path group_id: string,
  @body body: UpdateDynamicGroupRuleHttpRequest
): UpdateDynamicGroupRuleSuccess_200 | UpdateDynamicGroupRuleError400 | UpdateDynamicGroupRuleError403 | UpdateDynamicGroupRuleError422;

model PreviewDynamicGroupRuleHttpRequest {
  group_id: string;
  request: IdMagic.Contract.DynamicGroupPreviewRequest;
}

model PreviewDynamicGroupRuleHttpResponse {
  results: IdMagic.Contract.DynamicGroupPreviewResult[];
}

model PreviewDynamicGroupRuleSuccess_200 {
  @statusCode statusCode: 200;
  @body body: PreviewDynamicGroupRuleHttpResponse;
}

union PreviewDynamicGroupRuleError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model PreviewDynamicGroupRuleError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: PreviewDynamicGroupRuleError400Body;
}

union PreviewDynamicGroupRuleError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model PreviewDynamicGroupRuleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: PreviewDynamicGroupRuleError403Body;
}

union PreviewDynamicGroupRuleError422Body {
  IdMagic.Contract.InvalidDynamicGroupRuleError,
}

model PreviewDynamicGroupRuleError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: PreviewDynamicGroupRuleError422Body;
}

@doc("最大100 Userに未保存 CEL を副作用なしで評価し、現在所属との差分だけを返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("PreviewDynamicGroupRule")
@route("/api/admin/v1/groups/{group_id}/dynamic-rule/preview")
@post
op PreviewDynamicGroupRule(
  @path group_id: string,
  @body body: PreviewDynamicGroupRuleHttpRequest
): PreviewDynamicGroupRuleSuccess_200 | PreviewDynamicGroupRuleError400 | PreviewDynamicGroupRuleError403 | PreviewDynamicGroupRuleError422;

model EnableDynamicGroupRuleHttpRequest {
  group_id: string;
}

model EnableDynamicGroupRuleHttpResponse {
  rule: IdMagic.Contract.DynamicGroupRule;
}

model EnableDynamicGroupRuleSuccess_200 {
  @statusCode statusCode: 200;
  @body body: EnableDynamicGroupRuleHttpResponse;
}

union EnableDynamicGroupRuleError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model EnableDynamicGroupRuleError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: EnableDynamicGroupRuleError400Body;
}

union EnableDynamicGroupRuleError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model EnableDynamicGroupRuleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableDynamicGroupRuleError403Body;
}

@doc("検証済み rule を有効化し、旧 version membership を即時無効化して全件再評価 Job を予約する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("EnableDynamicGroupRule")
@route("/api/admin/v1/groups/{group_id}/dynamic-rule/enable")
@post
op EnableDynamicGroupRule(
  @path group_id: string,
  @body body: EnableDynamicGroupRuleHttpRequest
): EnableDynamicGroupRuleSuccess_200 | EnableDynamicGroupRuleError400 | EnableDynamicGroupRuleError403;

model DisableDynamicGroupRuleHttpRequest {
  group_id: string;
}

model DisableDynamicGroupRuleHttpResponse {
  rule: IdMagic.Contract.DynamicGroupRule;
}

model DisableDynamicGroupRuleSuccess_200 {
  @statusCode statusCode: 200;
  @body body: DisableDynamicGroupRuleHttpResponse;
}

union DisableDynamicGroupRuleError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DisableDynamicGroupRuleError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DisableDynamicGroupRuleError400Body;
}

union DisableDynamicGroupRuleError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DisableDynamicGroupRuleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableDynamicGroupRuleError403Body;
}

@doc("rule を無効化して、全 dynamic membership を認可・割当判定から即時除外する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("DisableDynamicGroupRule")
@route("/api/admin/v1/groups/{group_id}/dynamic-rule/disable")
@post
op DisableDynamicGroupRule(
  @path group_id: string,
  @body body: DisableDynamicGroupRuleHttpRequest
): DisableDynamicGroupRuleSuccess_200 | DisableDynamicGroupRuleError400 | DisableDynamicGroupRuleError403;

model ListUserGroupsHttpResponse {
  response: IdMagic.Contract.UserGroupsResponse;
}

model ListUserGroupsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListUserGroupsHttpResponse;
}

union ListUserGroupsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListUserGroupsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListUserGroupsError400Body;
}

union ListUserGroupsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListUserGroupsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListUserGroupsError403Body;
}

@doc("管理者が指定 User の所属グループと有効ロール内訳 (明示 / グループ由来 / union) を取得する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("ListUserGroups")
@route("/api/admin/v1/users/{user_id}/groups")
@get
op ListUserGroups(
  @path user_id: string
): ListUserGroupsSuccess_200 | ListUserGroupsError400 | ListUserGroupsError403;

model ListAgentsHttpResponse {
  response: IdMagic.Contract.AgentListResponse;
}

model ListAgentsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListAgentsHttpResponse;
}

union ListAgentsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAgentsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAgentsError403Body;
}

union ListAgentsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListAgentsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListAgentsError400Body;
}

@doc("管理者が所属テナントの Agent を name 昇順 (同値は id で tie-break) の双方向 keyset\npagination で一覧する。cursor は応答の Link response header (rel=\"prev\" / rel=\"next\") から取得する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:read"])
@TypeSpec.OpenAPI.operationId("ListAgents")
@route("/api/admin/v1/agents")
@get
op ListAgents(
  @query cursor?: string,
  @query limit?: int64
): ListAgentsSuccess_200 | ListAgentsError403 | ListAgentsError400;

model GetAgentHttpResponse {
  response: IdMagic.Contract.AgentSummaryResponse;
}

model GetAgentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAgentHttpResponse;
}

union GetAgentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetAgentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetAgentError400Body;
}

union GetAgentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAgentError403Body;
}

@doc("管理者が単一 Agent を取得する。別テナントの Agent は未存在として扱う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:read"])
@TypeSpec.OpenAPI.operationId("GetAgent")
@route("/api/admin/v1/agents/{agent_id}")
@get
op GetAgent(
  @path agent_id: string
): GetAgentSuccess_200 | GetAgentError400 | GetAgentError403;

model RegisterAgentHttpRequest {
  request: IdMagic.Contract.AgentRegisterRequest;
}

model RegisterAgentHttpResponse {
  response: IdMagic.Contract.AgentSummaryResponse;
}

model RegisterAgentSuccess_201 {
  @statusCode statusCode: 201;
  @body body: RegisterAgentHttpResponse;
}

union RegisterAgentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RegisterAgentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RegisterAgentError400Body;
}

union RegisterAgentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RegisterAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RegisterAgentError403Body;
}

union RegisterAgentError422Body {
  IdMagic.Contract.QuotaExceededError,
  IdMagic.Contract.AgentNameRequiredError,
  IdMagic.Contract.AgentOwnerRequiredError,
  IdMagic.Contract.AgentOwnerNotFoundError,
  IdMagic.Contract.InvalidRoleError,
}

model RegisterAgentError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: RegisterAgentError422Body;
}

@doc("管理者が所属テナントに Agent を登録する。name はテナント内で一意、owner_user_id 省略時は actor を所有者とする。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("RegisterAgent")
@route("/api/admin/v1/agents")
@post
op RegisterAgent(
  @body body: RegisterAgentHttpRequest
): RegisterAgentSuccess_201 | RegisterAgentError400 | RegisterAgentError403 | RegisterAgentError422;

model UpdateAgentHttpRequest {
  agent_id: string;
  request: IdMagic.Contract.AgentUpdateRequest;
}

model UpdateAgentHttpResponse {
  response: IdMagic.Contract.AgentSummaryResponse;
}

model UpdateAgentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateAgentHttpResponse;
}

union UpdateAgentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateAgentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateAgentError400Body;
}

union UpdateAgentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAgentError403Body;
}

union UpdateAgentError422Body {
  IdMagic.Contract.AgentNameRequiredError,
  IdMagic.Contract.AgentOwnerRequiredError,
  IdMagic.Contract.AgentOwnerNotFoundError,
  IdMagic.Contract.InvalidRoleError,
}

model UpdateAgentError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateAgentError422Body;
}

@doc("管理者が Agent の name / description / kind / owner_user_id / roles を更新する。owner_user_id の変更は所有権移転。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("UpdateAgent")
@route("/api/admin/v1/agents/{agent_id}")
@patch
op UpdateAgent(
  @path agent_id: string,
  @body body: UpdateAgentHttpRequest
): UpdateAgentSuccess_200 | UpdateAgentError400 | UpdateAgentError403 | UpdateAgentError422;

model DisableAgentHttpRequest {
  agent_id: string;
}

model DisableAgentSuccess_204 {
  @statusCode statusCode: 204;
}

union DisableAgentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DisableAgentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DisableAgentError400Body;
}

union DisableAgentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DisableAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableAgentError403Body;
}

@doc("管理者が Agent を可逆に運用停止する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("DisableAgent")
@route("/api/admin/v1/agents/{agent_id}/disable")
@post
op DisableAgent(
  @path agent_id: string,
  @body body: DisableAgentHttpRequest
): DisableAgentSuccess_204 | DisableAgentError400 | DisableAgentError403;

model EnableAgentHttpRequest {
  agent_id: string;
}

model EnableAgentSuccess_204 {
  @statusCode statusCode: 204;
}

union EnableAgentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model EnableAgentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: EnableAgentError400Body;
}

union EnableAgentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model EnableAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableAgentError403Body;
}

@doc("管理者が運用停止中の Agent を再稼働する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("EnableAgent")
@route("/api/admin/v1/agents/{agent_id}/enable")
@post
op EnableAgent(
  @path agent_id: string,
  @body body: EnableAgentHttpRequest
): EnableAgentSuccess_204 | EnableAgentError400 | EnableAgentError403;

model KillAgentHttpRequest {
  agent_id: string;
}

model KillAgentSuccess_204 {
  @statusCode statusCode: 204;
}

union KillAgentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model KillAgentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: KillAgentError400Body;
}

union KillAgentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model KillAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: KillAgentError403Body;
}

@doc("管理者が Agent を緊急停止する (一方向終端、復帰不能)。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("KillAgent")
@route("/api/admin/v1/agents/{agent_id}/kill")
@post
op KillAgent(
  @path agent_id: string,
  @body body: KillAgentHttpRequest
): KillAgentSuccess_204 | KillAgentError400 | KillAgentError403;

model DeleteAgentSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteAgentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteAgentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteAgentError400Body;
}

union DeleteAgentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAgentError403Body;
}

@doc("管理者が Agent を削除する。束縛は cascade で解除する。Killed の Agent は削除できない (fail-closed。削除により束縛先 OAuth2Client が client_credentials で無制限に token を再取得できてしまうため)。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("DeleteAgent")
@route("/api/admin/v1/agents/{agent_id}")
@delete
op DeleteAgent(
  @path agent_id: string
): DeleteAgentSuccess_204 | DeleteAgentError400 | DeleteAgentError403;

model BindAgentCredentialHttpRequest {
  agent_id: string;
  request: IdMagic.Contract.AgentCredentialBindRequest;
}

model BindAgentCredentialSuccess_204 {
  @statusCode statusCode: 204;
}

union BindAgentCredentialError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model BindAgentCredentialError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: BindAgentCredentialError400Body;
}

union BindAgentCredentialError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model BindAgentCredentialError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: BindAgentCredentialError403Body;
}

@doc("管理者が Agent に OAuth2Client を束縛する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("BindAgentCredential")
@route("/api/admin/v1/agents/{agent_id}/credentials")
@post
op BindAgentCredential(
  @path agent_id: string,
  @body body: BindAgentCredentialHttpRequest
): BindAgentCredentialSuccess_204 | BindAgentCredentialError400 | BindAgentCredentialError403;

model UnbindAgentCredentialSuccess_204 {
  @statusCode statusCode: 204;
}

union UnbindAgentCredentialError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UnbindAgentCredentialError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UnbindAgentCredentialError400Body;
}

union UnbindAgentCredentialError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UnbindAgentCredentialError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UnbindAgentCredentialError403Body;
}

@doc("管理者が Agent から OAuth2Client の束縛を解除する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("UnbindAgentCredential")
@route("/api/admin/v1/agents/{agent_id}/credentials/{client_id}")
@delete
op UnbindAgentCredential(
  @path agent_id: string,
  @path client_id: string
): UnbindAgentCredentialSuccess_204 | UnbindAgentCredentialError400 | UnbindAgentCredentialError403;
}

}
