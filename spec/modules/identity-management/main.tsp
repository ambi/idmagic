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

@doc("未認証で開くメールアドレスの確認の画面が、その確認の POST を守る CSRF トークンを読む。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetEmailVerificationContext")
@route("/api/account/v1/email/verify-context")
@get
op GetEmailVerificationContext(

): GetEmailVerificationContextSuccess_200;

model GetAccountSummarySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AccountSummary;
}

union GetAccountSummaryError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAccountSummaryError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAccountSummaryError403Body;
}

@doc("認証済みのユーザーが、セルフサービスのポータルのホームである自身のアカウントの概要を読む。対象は `subject.id` に固定される。ロールは含まず、最後のサインイン、パスワードの変更時刻、MFA の状態、未処理の必須のアクションを返す。管理シェルの AccountContext とは別の契約である。")
@TypeSpec.OpenAPI.operationId("GetAccountSummary")
@route("/api/account/v1/summary")
@get
op GetAccountSummary(

): GetAccountSummarySuccess_200 | GetAccountSummaryError403;

model GetUserProfileSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AccountProfileResponse;
}

union GetUserProfileError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetUserProfileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetUserProfileError403Body;
}

@doc("認証済みのユーザーが、自身のプロフィールを読む（セルフサービス）。対象は `subject.id` に固定されるので、ユーザーをまたいだ読み取りは構造上起こらない。`attributes` はユーザーが自身について読んでよいものだけを、`editable_attributes` は編集してよい定義を含む。")
@TypeSpec.OpenAPI.operationId("GetUserProfile")
@route("/api/account/v1/profile")
@get
op GetUserProfile(

): GetUserProfileSuccess_200 | GetUserProfileError403;

model UpdateUserProfileSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AccountProfileResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateUserProfileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateUserProfileError403Body;
}

union UpdateUserProfileError401Body {
  IdMagic.Contract.AuthenticationRequiredError,
}

model UpdateUserProfileError401 {
  @statusCode statusCode: 401;
  @header contentType: "application/problem+json";
  @body body: UpdateUserProfileError401Body;
}

union UpdateUserProfileError404Body {
  IdMagic.Contract.UserNotFoundError,
  IdMagic.Contract.SessionNotFoundError,
}

model UpdateUserProfileError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateUserProfileError404Body;
}

union UpdateUserProfileError422Body {
  IdMagic.Contract.InvalidUserAttributeError,
}

model UpdateUserProfileError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateUserProfileError422Body;
}

@doc("認証済みのユーザーが、自身の表示名と、`editable_by_user=true` の印を付けた属性を更新する（セルフサービス）。`subject.id` は `resource.id` と等しく固定される。属性はキーごとにマージし、`editable_by_user=false` のキーは拒否する。状態、ロール、組織の変更は管理者が行うもので、ここでは扱わない。")
@TypeSpec.OpenAPI.operationId("UpdateUserProfile")
@route("/api/account/v1/profile")
@patch
op UpdateUserProfile(
  @body body: IdMagic.Contract.AccountProfileUpdateRequest
): UpdateUserProfileSuccess_200 | UpdateUserProfileError400 | UpdateUserProfileError401 | UpdateUserProfileError403 | UpdateUserProfileError404 | UpdateUserProfileError422;

union RequestEmailChangeError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RequestEmailChangeError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RequestEmailChangeError400Body;
}

union RequestEmailChangeError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
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

model RequestEmailChangeSuccess_204 {
  @statusCode statusCode: 204;
}

@doc("認証済みのユーザーが、自身の主たるメールアドレスの変更を申し出る（セルフサービス）。新しいアドレスへ一回限りのリンクを送るだけで、`User.email` は変えない。対象は `subject.id` に固定され、テナントの中ですでに使われているアドレスは拒否する。確認のメッセージの文面は Tenancy の通知のテンプレートのカタログ（`template_key=EmailChangeConfirmation`）から解決し、ロケールは、受信者の User のロケールの属性、テナントの `default_locale`、システムのデフォルトのロケールの順に決める。テナントの上書きは組み込みのデフォルトより優先する。プレーンテキストと HTML の本文を常に両方生成し、確認のリンクはリクエストの発行者の URL から組み立てる。")
@TypeSpec.OpenAPI.operationId("RequestEmailChange")
@route("/api/account/v1/email/change-request")
@post
op RequestEmailChange(
  @body body: IdMagic.Contract.AccountEmailChangeRequest
): RequestEmailChangeSuccess_204 | RequestEmailChangeError400 | RequestEmailChangeError403 | RequestEmailChangeError422;

model ConfirmEmailChangeHttpResponse {
  @doc("常に `ok` である。")
  status: string;
}

model ConfirmEmailChangeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ConfirmEmailChangeHttpResponse;
}

union ConfirmEmailChangeError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ConfirmEmailChangeError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ConfirmEmailChangeError400Body;
}

union ConfirmEmailChangeError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ConfirmEmailChangeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ConfirmEmailChangeError403Body;
}

@doc("新しいアドレスへ送った一回限りのリンクを使い、主たるメールアドレスを確定する。トークンそのものが新しいアドレスの所有の証明なので、認証済みのセッションは不要である。トークンの所持が認可の根拠であり、セルフサービスのセッションではなくトークンの所持による認証なので、AuthenticatedSelf とセッションに基づくアクセスの分類は当てはまらない。確定すると `email_verified` が true になり、`verify_email` の必須のアクションがあれば自動で解除する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ConfirmEmailChange")
@route("/api/account/v1/email/verify")
@post
op ConfirmEmailChange(
  @body body: IdMagic.Contract.AccountEmailChangeConfirm
): ConfirmEmailChangeSuccess_200 | ConfirmEmailChangeError400 | ConfirmEmailChangeError403;

model ExportAccountDataSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AccountDataExport;
}

union ExportAccountDataError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ExportAccountDataError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ExportAccountDataError403Body;
}

@doc("認証済みのユーザーが、自身の個人データを JSON で読む（セルフサービス、GDPR のアクセス権）。対象は `subject.id` に固定される。現段階では同期的に生成し、プロフィールと同意を含む。")
@TypeSpec.OpenAPI.operationId("ExportAccountData")
@route("/api/account/v1/data-export")
@get
op ExportAccountData(

): ExportAccountDataSuccess_200 | ExportAccountDataError403;

model ListAdminUsersSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminUserListResponse;
}

union ListAdminUsersError403Body {
  IdMagic.Contract.InsufficientScopeError,
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

@doc("管理者が、削除されていないユーザーを、`preferred_username` の昇順（同じなら id の順）に、双方向のキーセットページングで一覧する。`query` はテナント全体で、ユーザー名、名前、メールアドレス、id、ロールを大文字と小文字を区別せず部分一致で検索し、`status` とともに絞り込む。カーソルはレスポンスの Link レスポンスヘッダー（`rel=\"prev\"` と `rel=\"next\"`）から得る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("ListAdminUsers")
@route("/api/admin/v1/users")
@get
op ListAdminUsers(
  @query query?: string,
  @query status?: IdMagic.Contract.UserStatus,
  @query cursor?: string,
  @query limit?: int64
): ListAdminUsersSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListAdminUsersError403 | ListAdminUsersError400;

model GetAdminUserSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminUserResponse;
}

union GetAdminUserError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminUserError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminUserError403Body;
}

union GetAdminUserError404Body {
  IdMagic.Contract.UserNotFoundError,
}

model GetAdminUserError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAdminUserError404Body;
}

@doc("管理者が、id でユーザーを読む。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("GetAdminUser")
@route("/api/admin/v1/users/{user_id}")
@get
op GetAdminUser(
  @path user_id: string
): GetAdminUserSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminUserError403 | GetAdminUserError404;

model CreateAdminUserSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.AdminUserResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CreateAdminUserError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateAdminUserError403Body;
}

union CreateAdminUserError409Body {
  IdMagic.Contract.UsernameConflictError,
  IdMagic.Contract.EmailTakenError,
}

model CreateAdminUserError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: CreateAdminUserError409Body;
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

@doc("管理者が、Authentication モジュールの PasswordPolicy を満たす初期パスワードでユーザーを作成する。`roles` に予約済みのロール `system_admin` を含められるのは、制御面テナントだけである。InvalidRoleError を参照する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("CreateAdminUser")
@route("/api/admin/v1/users")
@post
op CreateAdminUser(
  @body body: IdMagic.Contract.AdminUserCreateRequest
): CreateAdminUserSuccess_201 | CreateAdminUserError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateAdminUserError403 | CreateAdminUserError409 | CreateAdminUserError422;

model ImportAdminUsersSuccess_202 {
  @statusCode statusCode: 202;
  @body body: IdMagic.Contract.UserImportJobRef;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ImportAdminUsersError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ImportAdminUsersError403Body;
}

union ImportAdminUsersError503Body {
  IdMagic.Contract.UserImportUnavailableError,
}

model ImportAdminUsersError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: ImportAdminUsersError503Body;
}

@doc("管理者が、BOM の有無を問わない UTF-8 の CSV のアップロードを、副作用のないプレビューのジョブへストリーミングする。User のエクスポートと共有する設定可能な転送のポリシーのデフォルトは、データ行 100,000 行、ファイルあたり 64 MiB、フィールドあたり 64 KiB である。ペイロードは変更不可の成果物のストアに保持し、ジョブのパラメーターはテナントに限定した参照、サーバーが計算した SHA-256、大きさだけを保持する。ヘッダーは安定した機械用のキーを任意の順序と任意の部分集合で含めてよいが、未知の列、重複する列、パスワードまたはハッシュの列があると、それぞれファイル全体を拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("ImportAdminUsers")
@route("/api/admin/v1/users/imports")
@post
op ImportAdminUsers(
  @header contentType: "text/csv",
  @doc("リクエストボディとしてストリーミングする CSV の文書そのもの。JSON のエンベロープはなく、このエンドポイントは常にプレビューを実行する。")
  @body csv: string
): ImportAdminUsersSuccess_202 | ImportAdminUsersError400 | IdMagic.Contract.AuthenticationRequiredResponse | ImportAdminUsersError403 | ImportAdminUsersError503;

model ApplyAdminUserImportSuccess_202 {
  @statusCode statusCode: 202;
  @body body: IdMagic.Contract.UserImportJobRef;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ApplyAdminUserImportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ApplyAdminUserImportError403Body;
}

union ApplyAdminUserImportError503Body {
  IdMagic.Contract.UserImportUnavailableError,
}

model ApplyAdminUserImportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: ApplyAdminUserImportError503Body;
}

@doc("管理者が、同じテナントの成功したプレビューのジョブを指定して CSV を適用する。CSV は再び送らない。プレビューのペイロードと `source_sha256` を読み、同じ決定的な計画器がリポジトリの現在の状態に対して計画し直す。ファイル全体は行単位で成功し、各行のプロフィール、ロール、必須のアクション、カスタム属性は、一つの集約の変更として原子的に確定する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("ApplyAdminUserImport")
@route("/api/admin/v1/users/imports/{preview_job_id}/apply")
@post
op ApplyAdminUserImport(
  @path preview_job_id: string
): ApplyAdminUserImportSuccess_202 | ApplyAdminUserImportError400 | IdMagic.Contract.AuthenticationRequiredResponse | ApplyAdminUserImportError403 | ApplyAdminUserImportError503;

model GetAdminUserImportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.UserImportJob;
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminUserImportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminUserImportError403Body;
}

union GetAdminUserImportError404Body {
  IdMagic.Contract.UserImportNotFoundError,
}

model GetAdminUserImportError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAdminUserImportError404Body;
}

union GetAdminUserImportError503Body {
  IdMagic.Contract.UserImportUnavailableError,
}

model GetAdminUserImportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: GetAdminUserImportError503Body;
}

@doc("管理者が、同じテナントの CSV のインポートのジョブを、行ごとの検証と適用の結果とともに読む。行のエラーは、変更不可の結果の成果物に決まった大きさのページの塊として保持し、テナント、ジョブ、エラーの序数に結び付く署名した不透明なカーソルで、双方向に読む。上限のデフォルトは 100、最大は 200 である。ジョブの結果の JSON は、操作ごとの件数、結果の成果物のメタデータ、`error_total` だけを保持する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("GetAdminUserImport")
@route("/api/admin/v1/users/imports/{job_id}")
@get
op GetAdminUserImport(
  @path job_id: string,
  @query cursor?: string,
  @query limit?: int64
): GetAdminUserImportSuccess_200 | GetAdminUserImportError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminUserImportError403 | GetAdminUserImportError404 | GetAdminUserImportError503;

model StartUserCsvExportSuccess_202 {
  @statusCode statusCode: 202;
  @body body: IdMagic.Contract.DataExportJob;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

union StartUserCsvExportError429Body {
  IdMagic.Contract.ActiveJobQuotaExceededError,
}

model StartUserCsvExportError429 {
  @statusCode statusCode: 429;
  @header contentType: "application/problem+json";
  @body body: StartUserCsvExportError429Body;
}

union StartUserCsvExportError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model StartUserCsvExportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: StartUserCsvExportError503Body;
}

@doc("管理者が、ユーザーの一覧の CSV のエクスポートを非同期のジョブとして開始する。CSV のヘッダーは表示用のラベルではなく、求めた機械用のキーに一致する。列は、組み込みの User の許可リスト（13 のコアの列と、`attr:<key>` の形の 27 の組み込みの拡張属性）と、有効な TenantUserAttributeSchema の `custom:<key>` のテナント定義の属性の部分集合に限り、状態の絞り込みを引き継げる。生成したペイロードはインポートと同じ UserCsvTransferPolicy と変更不可の成果物のストアを使うので、成功した User のエクスポートは編集せずにプレビューできる。202 とエクスポートの id を返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("StartUserCsvExport")
@route("/api/admin/v1/users/exports")
@post
op StartUserCsvExport(
  @body body: IdMagic.Contract.DataExportRequest
): StartUserCsvExportSuccess_202 | StartUserCsvExportError400 | IdMagic.Contract.AuthenticationRequiredResponse | StartUserCsvExportError403 | StartUserCsvExportError422 | StartUserCsvExportError429 | StartUserCsvExportError503;

model ListUserExportsHttpResponse {
  exports: IdMagic.Contract.DataExportJob[];
}

model ListUserExportsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListUserExportsHttpResponse;
}

union ListUserExportsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListUserExportsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListUserExportsError403Body;
}

union ListUserExportsError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model ListUserExportsError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: ListUserExportsError503Body;
}

@doc("管理者が、User のエクスポートを、状態、件数、有効期限、ダウンロードできるかどうかとともに一覧する。CSV そのものも、個人情報の値も含まない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("ListUserExports")
@route("/api/admin/v1/users/exports")
@get
op ListUserExports(

): ListUserExportsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListUserExportsError403 | ListUserExportsError503;

model GetUserExportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DataExportJob;
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetUserExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetUserExportError403Body;
}

union GetUserExportError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model GetUserExportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: GetUserExportError503Body;
}

@doc("管理者が、一つの User のエクスポートの状態、進捗、失敗の理由、有効期限を読む。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("GetUserExport")
@route("/api/admin/v1/users/exports/{export_id}")
@get
op GetUserExport(
  @path export_id: string
): GetUserExportSuccess_200 | GetUserExportError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetUserExportError403 | GetUserExportError503;

model DownloadUserExportFileSuccess_200 {
  @statusCode statusCode: 200;
  @doc("生成したエクスポートは CSV である。")
  @header contentType: "text/csv; charset=utf-8";
  @doc("ブラウザーがダウンロードを保存するファイル名。`attachment; filename=\"<name>.csv\"` である。")
  @header("Content-Disposition") contentDisposition: string;
  @doc("CSV のバイト列。")
  @body body: bytes;
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model DownloadUserExportFileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DownloadUserExportFileError403Body;
}

union DownloadUserExportFileError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model DownloadUserExportFileError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: DownloadUserExportFileError503Body;
}

@doc("管理者が、完了した User のエクスポートを `Content-Disposition: attachment` でダウンロードする。ダウンロードできるのは、成功してまだ期限切れでないジョブだけである。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("DownloadUserExportFile")
@route("/api/admin/v1/users/exports/{export_id}/file")
@get
op DownloadUserExportFile(
  @path export_id: string
): DownloadUserExportFileSuccess_200 | DownloadUserExportFileError400 | IdMagic.Contract.AuthenticationRequiredResponse | DownloadUserExportFileError403 | DownloadUserExportFileError503;

model CancelUserExportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DataExportJob;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CancelUserExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CancelUserExportError403Body;
}

union CancelUserExportError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model CancelUserExportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: CancelUserExportError503Body;
}

@doc("管理者が、終わる前の User のエクスポートを取り消す。すでに終わったものは拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("CancelUserExport")
@route("/api/admin/v1/users/exports/{export_id}/cancel")
@post
op CancelUserExport(
  @path export_id: string,): CancelUserExportSuccess_200 | CancelUserExportError400 | IdMagic.Contract.AuthenticationRequiredResponse | CancelUserExportError403 | CancelUserExportError503;

model StartGroupCsvExportSuccess_202 {
  @statusCode statusCode: 202;
  @body body: IdMagic.Contract.DataExportJob;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

union StartGroupCsvExportError429Body {
  IdMagic.Contract.ActiveJobQuotaExceededError,
}

model StartGroupCsvExportError429 {
  @statusCode statusCode: 429;
  @header contentType: "application/problem+json";
  @body body: StartGroupCsvExportError429Body;
}

union StartGroupCsvExportError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model StartGroupCsvExportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: StartGroupCsvExportError503Body;
}

@doc("管理者が、グループの一覧の CSV のエクスポートを非同期のジョブとして開始する。CSV のヘッダーは表示用のラベルではなく、求めた機械用のキーに一致し、列は Group の許可リストの部分集合（インポートと互換の組み込みの列と、TenantGroupAttributeSchema の `custom:<key>` のテナント定義の属性）に限る。生成したペイロードはインポートと同じ CsvTransferPolicy と変更不可の成果物のストアを使うので、成功した Group のエクスポートは編集せずにプレビューできる。`lifecycle_action` はすべての行で空に書き出し、それによって編集しない往復ではすべての行が変更なしのままになる。202 とエクスポートの id を返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("StartGroupCsvExport")
@route("/api/admin/v1/groups/exports")
@post
op StartGroupCsvExport(
  @body body: IdMagic.Contract.DataExportRequest
): StartGroupCsvExportSuccess_202 | StartGroupCsvExportError400 | IdMagic.Contract.AuthenticationRequiredResponse | StartGroupCsvExportError403 | StartGroupCsvExportError422 | StartGroupCsvExportError429 | StartGroupCsvExportError503;

model ImportAdminGroupsSuccess_202 {
  @statusCode statusCode: 202;
  @body body: IdMagic.Contract.GroupImportJobRef;
}

union ImportAdminGroupsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ImportAdminGroupsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ImportAdminGroupsError400Body;
}

union ImportAdminGroupsError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ImportAdminGroupsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ImportAdminGroupsError403Body;
}

union ImportAdminGroupsError503Body {
  IdMagic.Contract.GroupImportUnavailableError,
}

model ImportAdminGroupsError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: ImportAdminGroupsError503Body;
}

@doc("管理者が、BOM の有無を問わない UTF-8 の CSV のアップロードを、副作用のないプレビューのジョブへストリーミングする。Group のエクスポートと共有する設定可能な転送のポリシーのデフォルトは、データ行 100,000 行、ファイルあたり 64 MiB、フィールドあたり 64 KiB である。ペイロードは変更不可の成果物のストアに保持し、ジョブのパラメーターはテナントに限定した参照、サーバーが計算した SHA-256、大きさだけを保持する。ヘッダーは安定した機械用のキーを任意の順序と任意の部分集合で含めてよいが、未知の列、重複する列、パスワードまたはハッシュの列があると、それぞれファイル全体を拒否する。行の対象はまず `id`、次に `name` で解決する。`membership_type` は作成時に選べるが後から変えられず、二つの動的な規則の列は組み合わさった最終の状態として検証し、`custom:<key>` のセルは宣言した型が定める正規の字句の形で解析し、外部の取得元が所有する Group は拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("ImportAdminGroups")
@route("/api/admin/v1/groups/imports")
@post
op ImportAdminGroups(
  @header contentType: "text/csv",
  @doc("リクエストボディとしてストリーミングする CSV の文書そのもの。JSON のエンベロープはなく、このエンドポイントは常にプレビューを実行する。")
  @body csv: string
): ImportAdminGroupsSuccess_202 | ImportAdminGroupsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ImportAdminGroupsError403 | ImportAdminGroupsError503;

model ApplyAdminGroupImportSuccess_202 {
  @statusCode statusCode: 202;
  @body body: IdMagic.Contract.GroupImportJobRef;
}

union ApplyAdminGroupImportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ApplyAdminGroupImportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ApplyAdminGroupImportError400Body;
}

union ApplyAdminGroupImportError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ApplyAdminGroupImportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ApplyAdminGroupImportError403Body;
}

union ApplyAdminGroupImportError503Body {
  IdMagic.Contract.GroupImportUnavailableError,
}

model ApplyAdminGroupImportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: ApplyAdminGroupImportError503Body;
}

@doc("管理者が、同じテナントの成功したプレビューのジョブを指定して CSV を適用する。CSV は再び送らない。プレビューのペイロードと `source_sha256` を読み、同じ決定的な計画器がリポジトリの現在の状態に対して計画し直す。ファイル全体は行単位で成功し、各行の名前、説明、連絡先のアドレス、ロール、カスタム属性、動的な規則は、一つの集約の変更として原子的に確定する。`lifecycle_action=delete` を含む行は、Group を削除し、その中のすべてのメンバーシップを連鎖して解除し、同じトランザクションで監査の記録を書く。削除は取り消せないので、CSV が指定しない Group を削除することはない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("ApplyAdminGroupImport")
@route("/api/admin/v1/groups/imports/{preview_job_id}/apply")
@post
op ApplyAdminGroupImport(
  @path preview_job_id: string
): ApplyAdminGroupImportSuccess_202 | ApplyAdminGroupImportError400 | IdMagic.Contract.AuthenticationRequiredResponse | ApplyAdminGroupImportError403 | ApplyAdminGroupImportError503;

model GetAdminGroupImportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.GroupImportJob;
}

union GetAdminGroupImportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetAdminGroupImportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetAdminGroupImportError400Body;
}

union GetAdminGroupImportError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminGroupImportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminGroupImportError403Body;
}

union GetAdminGroupImportError404Body {
  IdMagic.Contract.GroupImportNotFoundError,
}

model GetAdminGroupImportError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAdminGroupImportError404Body;
}

union GetAdminGroupImportError503Body {
  IdMagic.Contract.GroupImportUnavailableError,
}

model GetAdminGroupImportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: GetAdminGroupImportError503Body;
}

@doc("管理者が、同じテナントの CSV のグループのインポートのジョブを、行ごとの検証と適用の結果とともに読む。行のエラーは、変更不可の結果の成果物に決まった大きさのページの塊として保持し、テナント、ジョブ、エラーの序数に結び付く署名した不透明なカーソルで、双方向に読む。上限のデフォルトは 100、最大は 200 である。ジョブの結果の JSON は、操作ごとの件数、それとは別に数えた削除の件数、結果の成果物のメタデータ、`error_total` だけを保持する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("GetAdminGroupImport")
@route("/api/admin/v1/groups/imports/{job_id}")
@get
op GetAdminGroupImport(
  @path job_id: string,
  @query cursor?: string,
  @query limit?: int64
): GetAdminGroupImportSuccess_200 | GetAdminGroupImportError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminGroupImportError403 | GetAdminGroupImportError404 | GetAdminGroupImportError503;

model ListGroupExportsHttpResponse {
  exports: IdMagic.Contract.DataExportJob[];
}

model ListGroupExportsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListGroupExportsHttpResponse;
}

union ListGroupExportsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListGroupExportsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListGroupExportsError403Body;
}

union ListGroupExportsError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model ListGroupExportsError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: ListGroupExportsError503Body;
}

@doc("管理者が、Group のエクスポートを、状態、件数、有効期限、ダウンロードできるかどうかとともに一覧する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("ListGroupExports")
@route("/api/admin/v1/groups/exports")
@get
op ListGroupExports(

): ListGroupExportsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListGroupExportsError403 | ListGroupExportsError503;

model GetGroupExportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DataExportJob;
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetGroupExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetGroupExportError403Body;
}

union GetGroupExportError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model GetGroupExportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: GetGroupExportError503Body;
}

@doc("管理者が、一つの Group のエクスポートの状態、進捗、失敗の理由、有効期限を読む。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("GetGroupExport")
@route("/api/admin/v1/groups/exports/{export_id}")
@get
op GetGroupExport(
  @path export_id: string
): GetGroupExportSuccess_200 | GetGroupExportError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetGroupExportError403 | GetGroupExportError503;

model DownloadGroupExportFileSuccess_200 {
  @statusCode statusCode: 200;
  @doc("生成したエクスポートは CSV である。")
  @header contentType: "text/csv; charset=utf-8";
  @doc("ブラウザーがダウンロードを保存するファイル名。`attachment; filename=\"<name>.csv\"` である。")
  @header("Content-Disposition") contentDisposition: string;
  @doc("CSV のバイト列。")
  @body body: bytes;
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model DownloadGroupExportFileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DownloadGroupExportFileError403Body;
}

union DownloadGroupExportFileError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model DownloadGroupExportFileError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: DownloadGroupExportFileError503Body;
}

@doc("管理者が、完了した Group のエクスポートを `Content-Disposition: attachment` でダウンロードする。ダウンロードできるのは、成功してまだ期限切れでないジョブだけである。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("DownloadGroupExportFile")
@route("/api/admin/v1/groups/exports/{export_id}/file")
@get
op DownloadGroupExportFile(
  @path export_id: string
): DownloadGroupExportFileSuccess_200 | DownloadGroupExportFileError400 | IdMagic.Contract.AuthenticationRequiredResponse | DownloadGroupExportFileError403 | DownloadGroupExportFileError503;

model CancelGroupExportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DataExportJob;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CancelGroupExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CancelGroupExportError403Body;
}

union CancelGroupExportError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model CancelGroupExportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: CancelGroupExportError503Body;
}

@doc("管理者が、終わる前の Group のエクスポートを取り消す。すでに終わったものは拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("CancelGroupExport")
@route("/api/admin/v1/groups/exports/{export_id}/cancel")
@post
op CancelGroupExport(
  @path export_id: string,): CancelGroupExportSuccess_200 | CancelGroupExportError400 | IdMagic.Contract.AuthenticationRequiredResponse | CancelGroupExportError403 | CancelGroupExportError503;

model StartGroupMemberCsvExportSuccess_202 {
  @statusCode statusCode: 202;
  @body body: IdMagic.Contract.DataExportJob;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

union StartGroupMemberCsvExportError429Body {
  IdMagic.Contract.ActiveJobQuotaExceededError,
}

model StartGroupMemberCsvExportError429 {
  @statusCode statusCode: 429;
  @header contentType: "application/problem+json";
  @body body: StartGroupMemberCsvExportError429Body;
}

union StartGroupMemberCsvExportError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model StartGroupMemberCsvExportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: StartGroupMemberCsvExportError503Body;
}

@doc("管理者が、一つの Group のメンバーの CSV のエクスポートを開始する。`group_id` はパスから得て、そのグループのメンバーだけをエクスポートする。Entra、Okta、Google が使うのと同じ、グループごとの形である。CSV のヘッダーは表示用のラベルではなく、求めた機械用のキーに一致し、生成したペイロードはインポートと同じ CsvTransferPolicy と変更不可の成果物のストアを使うので、成功したエクスポートは編集せずにプレビューできる。`membership_state` はすべての行で `present` と書き出し、それによって編集しない往復ではすべての行が変更なしのままになる。202 とエクスポートの id を返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("StartGroupMemberCsvExport")
@route("/api/admin/v1/groups/{group_id}/members/exports")
@post
op StartGroupMemberCsvExport(
  @path group_id: string,
  @body body: IdMagic.Contract.DataExportRequest
): StartGroupMemberCsvExportSuccess_202 | StartGroupMemberCsvExportError400 | IdMagic.Contract.AuthenticationRequiredResponse | StartGroupMemberCsvExportError403 | StartGroupMemberCsvExportError422 | StartGroupMemberCsvExportError429 | StartGroupMemberCsvExportError503;

model ListGroupMemberExportsHttpResponse {
  exports: IdMagic.Contract.DataExportJob[];
}

model ListGroupMemberExportsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListGroupMemberExportsHttpResponse;
}

union ListGroupMemberExportsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListGroupMemberExportsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListGroupMemberExportsError403Body;
}

union ListGroupMemberExportsError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model ListGroupMemberExportsError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: ListGroupMemberExportsError503Body;
}

@doc("管理者が、一つの Group のメンバーのエクスポートを一覧する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("ListGroupMemberExports")
@route("/api/admin/v1/groups/{group_id}/members/exports")
@get
op ListGroupMemberExports(
  @path group_id: string
): ListGroupMemberExportsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListGroupMemberExportsError403 | ListGroupMemberExportsError503;

model GetGroupMemberExportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DataExportJob;
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetGroupMemberExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetGroupMemberExportError403Body;
}

union GetGroupMemberExportError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model GetGroupMemberExportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: GetGroupMemberExportError503Body;
}

@doc("管理者が、Group のメンバーのエクスポートの一つの状態を読む。別のグループに属するエクスポートの id は解決しない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("GetGroupMemberExport")
@route("/api/admin/v1/groups/{group_id}/members/exports/{export_id}")
@get
op GetGroupMemberExport(
  @path group_id: string,
  @path export_id: string
): GetGroupMemberExportSuccess_200 | GetGroupMemberExportError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetGroupMemberExportError403 | GetGroupMemberExportError503;

model DownloadGroupMemberExportFileSuccess_200 {
  @statusCode statusCode: 200;
  @doc("生成したエクスポートは CSV である。")
  @header contentType: "text/csv; charset=utf-8";
  @doc("ブラウザーがダウンロードを保存するファイル名。`attachment; filename=\"<name>.csv\"` である。")
  @header("Content-Disposition") contentDisposition: string;
  @doc("CSV のバイト列。")
  @body body: bytes;
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model DownloadGroupMemberExportFileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DownloadGroupMemberExportFileError403Body;
}

union DownloadGroupMemberExportFileError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model DownloadGroupMemberExportFileError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: DownloadGroupMemberExportFileError503Body;
}

@doc("管理者が、完了したメンバーのエクスポートを `Content-Disposition: attachment` でダウンロードする。ダウンロードできるのは、成功してまだ期限切れでないジョブだけである。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("DownloadGroupMemberExportFile")
@route("/api/admin/v1/groups/{group_id}/members/exports/{export_id}/file")
@get
op DownloadGroupMemberExportFile(
  @path group_id: string,
  @path export_id: string
): DownloadGroupMemberExportFileSuccess_200 | DownloadGroupMemberExportFileError400 | IdMagic.Contract.AuthenticationRequiredResponse | DownloadGroupMemberExportFileError403 | DownloadGroupMemberExportFileError503;

model CancelGroupMemberExportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DataExportJob;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CancelGroupMemberExportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CancelGroupMemberExportError403Body;
}

union CancelGroupMemberExportError503Body {
  IdMagic.Contract.JobsUnavailableError,
}

model CancelGroupMemberExportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: CancelGroupMemberExportError503Body;
}

@doc("管理者が、終わる前のメンバーのエクスポートを取り消す。すでに終わったものは拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("CancelGroupMemberExport")
@route("/api/admin/v1/groups/{group_id}/members/exports/{export_id}/cancel")
@post
op CancelGroupMemberExport(
  @path group_id: string,
  @path export_id: string,): CancelGroupMemberExportSuccess_200 | CancelGroupMemberExportError400 | IdMagic.Contract.AuthenticationRequiredResponse | CancelGroupMemberExportError403 | CancelGroupMemberExportError503;

model ImportAdminGroupMembersSuccess_202 {
  @statusCode statusCode: 202;
  @body body: IdMagic.Contract.GroupMembershipImportJobRef;
}

union ImportAdminGroupMembersError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ImportAdminGroupMembersError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ImportAdminGroupMembersError400Body;
}

union ImportAdminGroupMembersError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ImportAdminGroupMembersError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ImportAdminGroupMembersError403Body;
}

union ImportAdminGroupMembersError503Body {
  IdMagic.Contract.GroupMembershipImportUnavailableError,
}

model ImportAdminGroupMembersError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: ImportAdminGroupMembersError503Body;
}

@doc("管理者が、BOM の有無を問わない UTF-8 の CSV のアップロードを、副作用のないプレビューのジョブへストリーミングする。パスの `group_id` が、どの行も触れてよい唯一の Group である。`group_id` と `group_name` の列は読み取り専用の確認用で、別の Group を指定する行は、別の場所へ書くことなく拒否する。転送のポリシー、変更不可の成果物のストア、メタデータだけのジョブのパラメーターは、すべての CSV が共有するものである。ヘッダーは安定した機械用のキーを任意の順序と任意の部分集合で含めてよいが、`membership_state` は例外で、ファイルが表せる唯一の変更の意図なので必ず宣言しなければならない。行の User はまず `user_id`、次に `preferred_username` で解決し、`source` と `created_at` は受け付けて無視する。動的な Group、動的な規則が所有するメンバーシップ、外部の取得元が所有する Group または User は、それぞれフェイルクローズで拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("ImportAdminGroupMembers")
@route("/api/admin/v1/groups/{group_id}/members/imports")
@post
op ImportAdminGroupMembers(
  @path group_id: string,
  @header contentType: "text/csv",
  @doc("リクエストボディとしてストリーミングする CSV の文書そのもの。JSON のエンベロープはなく、このエンドポイントは常にプレビューを実行する。")
  @body csv: string
): ImportAdminGroupMembersSuccess_202 | ImportAdminGroupMembersError400 | IdMagic.Contract.AuthenticationRequiredResponse | ImportAdminGroupMembersError403 | ImportAdminGroupMembersError503;

model ApplyAdminGroupMemberImportSuccess_202 {
  @statusCode statusCode: 202;
  @body body: IdMagic.Contract.GroupMembershipImportJobRef;
}

union ApplyAdminGroupMemberImportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ApplyAdminGroupMemberImportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ApplyAdminGroupMemberImportError400Body;
}

union ApplyAdminGroupMemberImportError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ApplyAdminGroupMemberImportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ApplyAdminGroupMemberImportError403Body;
}

union ApplyAdminGroupMemberImportError404Body {
  IdMagic.Contract.GroupMembershipImportNotFoundError,
}

model ApplyAdminGroupMemberImportError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: ApplyAdminGroupMemberImportError404Body;
}

union ApplyAdminGroupMemberImportError503Body {
  IdMagic.Contract.GroupMembershipImportUnavailableError,
}

model ApplyAdminGroupMemberImportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: ApplyAdminGroupMemberImportError503Body;
}

@doc("管理者が、同じテナントと同じ Group の成功したプレビューのジョブを指定して CSV を適用する。CSV は再び送らない。プレビューのペイロードと `source_sha256` を読み、同じ決定的な計画器が現在のメンバーシップに対して計画し直すので、プレビューの後に別の操作が行った追加や解除は、再生せずに判断し直す。受け付けた各行は、メンバーシップと監査の記録を原子的に確定し、失敗した行は、すでに適用した行をそのまま残す。触れるのはファイルが指定する行だけである。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("ApplyAdminGroupMemberImport")
@route("/api/admin/v1/groups/{group_id}/members/imports/{preview_job_id}/apply")
@post
op ApplyAdminGroupMemberImport(
  @path group_id: string,
  @path preview_job_id: string
): ApplyAdminGroupMemberImportSuccess_202 | ApplyAdminGroupMemberImportError400 | IdMagic.Contract.AuthenticationRequiredResponse | ApplyAdminGroupMemberImportError403 | ApplyAdminGroupMemberImportError404 | ApplyAdminGroupMemberImportError503;

model GetAdminGroupMemberImportSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.GroupMembershipImportJob;
}

union GetAdminGroupMemberImportError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetAdminGroupMemberImportError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetAdminGroupMemberImportError400Body;
}

union GetAdminGroupMemberImportError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminGroupMemberImportError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminGroupMemberImportError403Body;
}

union GetAdminGroupMemberImportError404Body {
  IdMagic.Contract.GroupMembershipImportNotFoundError,
}

model GetAdminGroupMemberImportError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAdminGroupMemberImportError404Body;
}

union GetAdminGroupMemberImportError503Body {
  IdMagic.Contract.GroupMembershipImportUnavailableError,
}

model GetAdminGroupMemberImportError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: GetAdminGroupMemberImportError503Body;
}

@doc("管理者が、同じテナントと同じ Group の CSV のグループのメンバーシップのインポートのジョブを、行ごとの検証と適用の結果とともに読む。行のエラーは、変更不可の結果の成果物に決まった大きさのページの塊として保持し、テナント、ジョブ、エラーの序数に結び付く署名した不透明なカーソルで、双方向に読む。上限のデフォルトは 100、最大は 200 である。ジョブの結果の JSON は、操作ごとの件数、それとは別に数えた解除の件数、結果の成果物のメタデータ、`error_total` だけを保持する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("GetAdminGroupMemberImport")
@route("/api/admin/v1/groups/{group_id}/members/imports/{job_id}")
@get
op GetAdminGroupMemberImport(
  @path group_id: string,
  @path job_id: string,
  @query cursor?: string,
  @query limit?: int64
): GetAdminGroupMemberImportSuccess_200 | GetAdminGroupMemberImportError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminGroupMemberImportError403 | GetAdminGroupMemberImportError404 | GetAdminGroupMemberImportError503;

model UpdateAdminUserSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminUserResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAdminUserError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminUserError403Body;
}

union UpdateAdminUserError409Body {
  IdMagic.Contract.UsernameConflictError,
  IdMagic.Contract.EmailTakenError,
}

model UpdateAdminUserError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminUserError409Body;
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

@doc("管理者が、ユーザーのプロフィール、`email_verified`、ロールを更新する。予約済みのロール `system_admin` の追加は制御面テナントでだけ受け付け、拒否したときは同じリクエストのほかのフィールドも書き込まない。InvalidRoleError を参照する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("UpdateAdminUser")
@route("/api/admin/v1/users/{user_id}")
@patch
op UpdateAdminUser(
  @path user_id: string,
  @body body: IdMagic.Contract.AdminUserUpdateRequest
): UpdateAdminUserSuccess_200 | UpdateAdminUserError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateAdminUserError403 | UpdateAdminUserError409 | UpdateAdminUserError422;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

union DisableAdminUserError409Body {
  IdMagic.Contract.UserPendingDeletionError,
}

model DisableAdminUserError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: DisableAdminUserError409Body;
}

@doc("管理者がユーザーを無効にし、新しいサインインと既存のセッションの利用の両方を拒否する。削除を予定したユーザーは拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("DisableAdminUser")
@route("/api/admin/v1/users/{user_id}/disable")
@post
op DisableAdminUser(
  @path user_id: string,): DisableAdminUserSuccess_204 | DisableAdminUserError400 | IdMagic.Contract.AuthenticationRequiredResponse | DisableAdminUserError403 | DisableAdminUserError409 | DisableAdminUserError422;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model EnableAdminUserError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableAdminUserError403Body;
}

union EnableAdminUserError409Body {
  IdMagic.Contract.UserPendingDeletionError,
}

model EnableAdminUserError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: EnableAdminUserError409Body;
}

@doc("管理者が、無効にしたユーザーを再び有効にする。削除を予定したユーザーは拒否する。予定を取り消すのは RestoreAdminUser である。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("EnableAdminUser")
@route("/api/admin/v1/users/{user_id}/enable")
@post
op EnableAdminUser(
  @path user_id: string,): EnableAdminUserSuccess_204 | EnableAdminUserError400 | IdMagic.Contract.AuthenticationRequiredResponse | EnableAdminUserError403 | EnableAdminUserError409;

model DeleteAdminUserHttpRequest {
  @doc("`?purge=true` と同じ意味で、論理削除を飛ばして即座に匿名化する。省くと論理削除になる。")
  force?: boolean;
  @doc("監査証跡のために残す自由入力のメモ。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("管理者がユーザーを削除する。デフォルトは論理削除で、状態を `pending_deletion` へ移し、個人情報、Consent、RefreshToken、Session を残すので、猶予期間の間は Restore できる。`purge=true` では匿名化の連鎖を直ちに実行し、墓標を残す。どちらも冪等で、すでに同じ最終の状態にあるユーザーに対して再び呼ぶと、何もせずに 204 を返す。呼び出し元は自身を削除できないので、admin や system_admin が自身のアクセスを失うことはない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("DeleteAdminUser")
@route("/api/admin/v1/users/{user_id}")
@delete
op DeleteAdminUser(
  @path user_id: string,
  @doc("`true` は論理削除を飛ばして即座に匿名化する。ボディの `force` と同じである。ボディ自体は任意である。")
  @query purge?: boolean,
  @body body: DeleteAdminUserHttpRequest
): DeleteAdminUserSuccess_204 | DeleteAdminUserError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteAdminUserError403 | DeleteAdminUserError422;

model RestoreAdminUserSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminUserResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("管理者が、`pending_deletion` のユーザーを復元する。状態は `active` に戻り、個人情報と資格情報を保ったまま復元する。猶予期間を過ぎたユーザーと `pending_deletion` でないユーザーは InvalidRequestError になる。復元したユーザーを返す。自身に対する操作は、自身の削除の扱いと同じく、ユースケースで拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("RestoreAdminUser")
@route("/api/admin/v1/users/{user_id}/restore")
@post
op RestoreAdminUser(
  @path user_id: string,): RestoreAdminUserSuccess_200 | RestoreAdminUserError400 | IdMagic.Contract.AuthenticationRequiredResponse | RestoreAdminUserError403 | RestoreAdminUserError409;

model SetUserRequiredActionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminUserResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("管理者が、対象のユーザーに次のサインインで強制するアクション（RequiredAction）を与える。`update_password` を与えたユーザーは次のサインインでパスワードの変更の画面へ送られ、変更するとアクションは自動で解除される。冪等で、アクションがすでにあれば現在の状態を返す。管理者だけが行える。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("SetUserRequiredAction")
@route("/api/admin/v1/users/{user_id}/required-actions")
@post
op SetUserRequiredAction(
  @path user_id: string,
  @body body: IdMagic.Contract.AdminRequiredActionRequest
): SetUserRequiredActionSuccess_200 | SetUserRequiredActionError400 | IdMagic.Contract.AuthenticationRequiredResponse | SetUserRequiredActionError403 | SetUserRequiredActionError422;

model ClearUserRequiredActionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminUserResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ClearUserRequiredActionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ClearUserRequiredActionError403Body;
}

@doc("管理者が、対象のユーザーから強制するアクションを解除する。冪等で、アクションがなければ現在の状態を返す。管理者だけが行える。ユーザーが自身のパスワードを変更したときの `update_password` の自動の解除は、代わりに ChangePassword と ResetPasswordWithToken が行う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("ClearUserRequiredAction")
@route("/api/admin/v1/users/{user_id}/required-actions/{action}")
@delete
op ClearUserRequiredAction(
  @path user_id: string,
  @path action: string
): ClearUserRequiredActionSuccess_200 | ClearUserRequiredActionError400 | IdMagic.Contract.AuthenticationRequiredResponse | ClearUserRequiredActionError403;

model ListGroupsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.GroupListResponse;
}

union ListGroupsError403Body {
  IdMagic.Contract.InsufficientScopeError,
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

@doc("管理者が、自身のテナントのグループを、名前の昇順（同じなら id の順）に、双方向のキーセットページングで一覧する。各グループは `member_count` を含む。カーソルはレスポンスの Link レスポンスヘッダー（`rel=\"prev\"` と `rel=\"next\"`）から得る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("ListGroups")
@route("/api/admin/v1/groups")
@get
op ListGroups(
  @query cursor?: string,
  @query limit?: int64
): ListGroupsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListGroupsError403 | ListGroupsError400;

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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetGroupError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetGroupError403Body;
}

@doc("管理者が、グループを一つ、そのメンバーの一覧とともに読む。他のテナントのグループは存在しないものとして扱う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("GetGroup")
@route("/api/admin/v1/groups/{group_id}")
@get
op GetGroup(
  @path group_id: string
): GetGroupSuccess_200 | GetGroupError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetGroupError403;

model CreateGroupSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.GroupSummaryResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CreateGroupError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateGroupError403Body;
}

union CreateGroupError409Body {
  IdMagic.Contract.GroupNameConflictError,
}

model CreateGroupError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: CreateGroupError409Body;
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

@doc("管理者が、自身のテナントにグループを作成する。名前はテナントの中で一意である。メールアドレスは形式だけを検査し一意性の制約はなく、属性はテナントの TenantGroupAttributeSchema に照らして検証する。`roles` に予約済みのロール `system_admin` を含められるのは制御面テナントだけで、そこではグループがメンバーの制御面の User の有効ロールにそれを与える。InvalidRoleError を参照する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("CreateGroup")
@route("/api/admin/v1/groups")
@post
op CreateGroup(
  @body body: IdMagic.Contract.GroupCreateRequest
): CreateGroupSuccess_201 | CreateGroupError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateGroupError403 | CreateGroupError409 | CreateGroupError422;

model UpdateGroupSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.GroupSummaryResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateGroupError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateGroupError403Body;
}

union UpdateGroupError409Body {
  IdMagic.Contract.GroupNameConflictError,
}

model UpdateGroupError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: UpdateGroupError409Body;
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

@doc("管理者が、グループの名前、説明、メールアドレス、属性、またはロールを更新する。予約済みのロール `system_admin` の追加は制御面テナントでだけ受け付け、拒否したときは同じリクエストのほかのフィールドも書き込まない。InvalidRoleError を参照する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("UpdateGroup")
@route("/api/admin/v1/groups/{group_id}")
@patch
op UpdateGroup(
  @path group_id: string,
  @body body: IdMagic.Contract.GroupUpdateRequest
): UpdateGroupSuccess_200 | UpdateGroupError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateGroupError403 | UpdateGroupError409 | UpdateGroupError422;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteGroupError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteGroupError403Body;
}

@doc("管理者がグループを削除する。その中のメンバーシップは連鎖して解除し、最後の GroupDeleted の前にそれぞれ個別の GroupMemberRemoved を発行し、グループが使っていたグループのクォータを解放する。削除は即時で取り消せず、復元のための予約の状態はない。CSV のインポートの経路は、明示的な `lifecycle_action=delete` の列で同じ削除に至り、解除、削除、監査の記録を行ごとに一つのトランザクションで確定する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("DeleteGroup")
@route("/api/admin/v1/groups/{group_id}")
@delete
op DeleteGroup(
  @path group_id: string
): DeleteGroupSuccess_204 | DeleteGroupError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteGroupError403;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model AddGroupMemberError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: AddGroupMemberError403Body;
}

@doc("管理者が、User をグループのメンバーにする。許すのは同じテナントの User だけで、既存のメンバーシップに対しては何もしないので、呼び出しは冪等である。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("AddGroupMember")
@route("/api/admin/v1/groups/{group_id}/members/{user_id}")
@post
op AddGroupMember(
  @path group_id: string,
  @path user_id: string
): AddGroupMemberSuccess_204 | AddGroupMemberError400 | IdMagic.Contract.AuthenticationRequiredResponse | AddGroupMemberError403;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RemoveGroupMemberError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RemoveGroupMemberError403Body;
}

@doc("管理者が、User をグループから外す。メンバーでないユーザーに対しては何もしないので、呼び出しは冪等である。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("RemoveGroupMember")
@route("/api/admin/v1/groups/{group_id}/members/{user_id}")
@delete
op RemoveGroupMember(
  @path group_id: string,
  @path user_id: string
): RemoveGroupMemberSuccess_204 | RemoveGroupMemberError400 | IdMagic.Contract.AuthenticationRequiredResponse | RemoveGroupMemberError403;

model UpdateDynamicGroupRuleSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DynamicGroupRule;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("管理者が、動的グループの制限付きの CEL の規則を検証して保存する。未知の属性、型の不一致、真偽値でない結果、許可する部分集合の外の構文、上限の超過は、いずれも拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("UpdateDynamicGroupRule")
@route("/api/admin/v1/groups/{group_id}/dynamic-rule")
@put
op UpdateDynamicGroupRule(
  @path group_id: string,
  @body body: IdMagic.Contract.DynamicGroupRuleRequest
): UpdateDynamicGroupRuleSuccess_200 | UpdateDynamicGroupRuleError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateDynamicGroupRuleError403 | UpdateDynamicGroupRuleError422;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("保存していない CEL を、副作用なしに最大 100 の User に対して評価し、現在のメンバーシップとの差分だけを返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("PreviewDynamicGroupRule")
@route("/api/admin/v1/groups/{group_id}/dynamic-rule/preview")
@post
op PreviewDynamicGroupRule(
  @path group_id: string,
  @body body: IdMagic.Contract.DynamicGroupPreviewRequest
): PreviewDynamicGroupRuleSuccess_200 | PreviewDynamicGroupRuleError400 | IdMagic.Contract.AuthenticationRequiredResponse | PreviewDynamicGroupRuleError403 | PreviewDynamicGroupRuleError422;

model EnableDynamicGroupRuleSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DynamicGroupRule;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model EnableDynamicGroupRuleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableDynamicGroupRuleError403Body;
}

@doc("検証済みの規則を有効にし、古いバージョンのメンバーシップを即座に無効にして、すべてを評価し直すジョブを予定する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("EnableDynamicGroupRule")
@route("/api/admin/v1/groups/{group_id}/dynamic-rule/enable")
@post
op EnableDynamicGroupRule(
  @path group_id: string,): EnableDynamicGroupRuleSuccess_200 | EnableDynamicGroupRuleError400 | IdMagic.Contract.AuthenticationRequiredResponse | EnableDynamicGroupRuleError403;

model DisableDynamicGroupRuleSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DynamicGroupRule;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DisableDynamicGroupRuleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableDynamicGroupRuleError403Body;
}

@doc("規則を無効にし、すべての動的なメンバーシップを即座に認可と割り当ての判断から外す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("DisableDynamicGroupRule")
@route("/api/admin/v1/groups/{group_id}/dynamic-rule/disable")
@post
op DisableDynamicGroupRule(
  @path group_id: string,): DisableDynamicGroupRuleSuccess_200 | DisableDynamicGroupRuleError400 | IdMagic.Contract.AuthenticationRequiredResponse | DisableDynamicGroupRuleError403;

model ListUserGroupsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.UserGroupsResponse;
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListUserGroupsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListUserGroupsError403Body;
}

@doc("管理者が、指定した User が属するグループと、その有効ロールの内訳（直接、グループから継承、和集合）を読む。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("ListUserGroups")
@route("/api/admin/v1/users/{user_id}/groups")
@get
op ListUserGroups(
  @path user_id: string
): ListUserGroupsSuccess_200 | ListUserGroupsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ListUserGroupsError403;

model ListAgentsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AgentListResponse;
}

union ListAgentsError403Body {
  IdMagic.Contract.InsufficientScopeError,
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

@doc("管理者が、自身のテナントの Agent を、名前の昇順（同じなら id の順）に、双方向のキーセットページングで一覧する。カーソルはレスポンスの Link レスポンスヘッダー（`rel=\"prev\"` と `rel=\"next\"`）から得る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:read"])
@TypeSpec.OpenAPI.operationId("ListAgents")
@route("/api/admin/v1/agents")
@get
op ListAgents(
  @query cursor?: string,
  @query limit?: int64
): ListAgentsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListAgentsError403 | ListAgentsError400;

model GetAgentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AgentSummaryResponse;
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAgentError403Body;
}

union GetAgentError404Body {
  IdMagic.Contract.AgentNotFoundError,
}

model GetAgentError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAgentError404Body;
}

@doc("管理者が Agent を一つ読む。他のテナントの Agent は存在しないものとして扱う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:read"])
@TypeSpec.OpenAPI.operationId("GetAgent")
@route("/api/admin/v1/agents/{agent_id}")
@get
op GetAgent(
  @path agent_id: string
): GetAgentSuccess_200 | GetAgentError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetAgentError403 | GetAgentError404;

model RegisterAgentSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.AgentSummaryResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RegisterAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RegisterAgentError403Body;
}

union RegisterAgentError409Body {
  IdMagic.Contract.AgentNameConflictError,
}

model RegisterAgentError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: RegisterAgentError409Body;
}

union RegisterAgentError422Body {
  IdMagic.Contract.QuotaExceededError,
  IdMagic.Contract.AgentNameRequiredError,
  IdMagic.Contract.AgentKindRequiredError,
  IdMagic.Contract.InvalidAgentKindError,
  IdMagic.Contract.AgentOwnerRequiredError,
  IdMagic.Contract.AgentOwnerNotFoundError,
  IdMagic.Contract.InvalidRoleError,
}

model RegisterAgentError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: RegisterAgentError422Body;
}

@doc("管理者が、自身のテナントに Agent を登録する。名前はテナントの中で一意で、`owner_user_id` を省くと呼び出し元が所有者になる。Agent はシステムの運用者とは別の種類の主体なので、`roles` には、どのテナントでも予約済みのロール `system_admin` を含められない。InvalidRoleError を参照する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("RegisterAgent")
@route("/api/admin/v1/agents")
@post
op RegisterAgent(
  @body body: IdMagic.Contract.AgentRegisterRequest
): RegisterAgentSuccess_201 | RegisterAgentError400 | IdMagic.Contract.AuthenticationRequiredResponse | RegisterAgentError403 | RegisterAgentError409 | RegisterAgentError422;

model UpdateAgentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AgentSummaryResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAgentError403Body;
}

union UpdateAgentError404Body {
  IdMagic.Contract.AgentNotFoundError,
}

model UpdateAgentError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateAgentError404Body;
}

union UpdateAgentError409Body {
  IdMagic.Contract.AgentNameConflictError,
  IdMagic.Contract.AgentKilledError,
}

model UpdateAgentError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: UpdateAgentError409Body;
}

union UpdateAgentError422Body {
  IdMagic.Contract.AgentNameRequiredError,
  IdMagic.Contract.InvalidAgentKindError,
  IdMagic.Contract.AgentOwnerRequiredError,
  IdMagic.Contract.AgentOwnerNotFoundError,
  IdMagic.Contract.InvalidRoleError,
}

model UpdateAgentError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateAgentError422Body;
}

@doc("管理者が、Agent の名前、説明、種類、`owner_user_id`、またはロールを更新する。`owner_user_id` の変更は所有権の移転である。予約済みのロール `system_admin` の追加はどのテナントでも拒否し、拒否したときは同じリクエストのほかのフィールドも書き込まない。InvalidRoleError を参照する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("UpdateAgent")
@route("/api/admin/v1/agents/{agent_id}")
@patch
op UpdateAgent(
  @path agent_id: string,
  @body body: IdMagic.Contract.AgentUpdateRequest
): UpdateAgentSuccess_200 | UpdateAgentError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateAgentError403 | UpdateAgentError404 | UpdateAgentError409 | UpdateAgentError422;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DisableAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableAgentError403Body;
}

union DisableAgentError404Body {
  IdMagic.Contract.AgentNotFoundError,
}

model DisableAgentError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DisableAgentError404Body;
}

union DisableAgentError409Body {
  IdMagic.Contract.AgentKilledError,
}

model DisableAgentError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: DisableAgentError409Body;
}

@doc("管理者が、Agent を元に戻せる形で一時停止する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("DisableAgent")
@route("/api/admin/v1/agents/{agent_id}/disable")
@post
op DisableAgent(
  @path agent_id: string,): DisableAgentSuccess_204 | DisableAgentError400 | IdMagic.Contract.AuthenticationRequiredResponse | DisableAgentError403 | DisableAgentError404 | DisableAgentError409;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model EnableAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableAgentError403Body;
}

union EnableAgentError404Body {
  IdMagic.Contract.AgentNotFoundError,
}

model EnableAgentError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: EnableAgentError404Body;
}

union EnableAgentError409Body {
  IdMagic.Contract.AgentKilledError,
  IdMagic.Contract.AgentOwnerInactiveError,
}

model EnableAgentError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: EnableAgentError409Body;
}

@doc("管理者が、一時停止した Agent を稼働に戻す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("EnableAgent")
@route("/api/admin/v1/agents/{agent_id}/enable")
@post
op EnableAgent(
  @path agent_id: string,): EnableAgentSuccess_204 | EnableAgentError400 | IdMagic.Contract.AuthenticationRequiredResponse | EnableAgentError403 | EnableAgentError404 | EnableAgentError409;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model KillAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: KillAgentError403Body;
}

union KillAgentError404Body {
  IdMagic.Contract.AgentNotFoundError,
}

model KillAgentError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: KillAgentError404Body;
}

union KillAgentError409Body {
  IdMagic.Contract.AgentKilledError,
}

model KillAgentError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: KillAgentError409Body;
}

@doc("管理者が Agent を強制停止する。一方向の終端の操作で、元には戻せない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("KillAgent")
@route("/api/admin/v1/agents/{agent_id}/kill")
@post
op KillAgent(
  @path agent_id: string,): KillAgentSuccess_204 | KillAgentError400 | IdMagic.Contract.AuthenticationRequiredResponse | KillAgentError403 | KillAgentError404 | KillAgentError409;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAgentError403Body;
}

union DeleteAgentError404Body {
  IdMagic.Contract.AgentNotFoundError,
}

model DeleteAgentError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DeleteAgentError404Body;
}

union DeleteAgentError409Body {
  IdMagic.Contract.AgentKilledError,
}

model DeleteAgentError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: DeleteAgentError409Body;
}

@doc("管理者が Agent を削除する。そのバインディングは連鎖して解除する。強制停止した Agent はフェイルクローズで削除できない。削除すると、それに結び付いていた OAuth2Client が client_credentials で再び無制限にトークンを取得できてしまうためである。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("DeleteAgent")
@route("/api/admin/v1/agents/{agent_id}")
@delete
op DeleteAgent(
  @path agent_id: string
): DeleteAgentSuccess_204 | DeleteAgentError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteAgentError403 | DeleteAgentError404 | DeleteAgentError409;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model BindAgentCredentialError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: BindAgentCredentialError403Body;
}

union BindAgentCredentialError404Body {
  IdMagic.Contract.AgentNotFoundError,
}

model BindAgentCredentialError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: BindAgentCredentialError404Body;
}

union BindAgentCredentialError409Body {
  IdMagic.Contract.AgentKilledError,
  IdMagic.Contract.AgentClientAlreadyBoundError,
}

model BindAgentCredentialError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: BindAgentCredentialError409Body;
}

union BindAgentCredentialError422Body {
  IdMagic.Contract.OAuth2ClientNotFoundError,
}

model BindAgentCredentialError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: BindAgentCredentialError422Body;
}

@doc("管理者が、OAuth2Client を Agent に結び付ける。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("BindAgentCredential")
@route("/api/admin/v1/agents/{agent_id}/credentials")
@post
op BindAgentCredential(
  @path agent_id: string,
  @body body: IdMagic.Contract.AgentCredentialBindRequest
): BindAgentCredentialSuccess_204 | BindAgentCredentialError400 | IdMagic.Contract.AuthenticationRequiredResponse | BindAgentCredentialError403 | BindAgentCredentialError404 | BindAgentCredentialError409 | BindAgentCredentialError422;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UnbindAgentCredentialError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UnbindAgentCredentialError403Body;
}

union UnbindAgentCredentialError404Body {
  IdMagic.Contract.AgentNotFoundError,
}

model UnbindAgentCredentialError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UnbindAgentCredentialError404Body;
}

@doc("管理者が、OAuth2Client と Agent の結び付きを外す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("UnbindAgentCredential")
@route("/api/admin/v1/agents/{agent_id}/credentials/{client_id}")
@delete
op UnbindAgentCredential(
  @path agent_id: string,
  @path client_id: string
): UnbindAgentCredentialSuccess_204 | UnbindAgentCredentialError400 | IdMagic.Contract.AuthenticationRequiredResponse | UnbindAgentCredentialError403 | UnbindAgentCredentialError404;
}

}
