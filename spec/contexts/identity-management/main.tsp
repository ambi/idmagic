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

@doc("The email verification screen, which opens unauthenticated, reads the CSRF token that protects its confirmation POST.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetEmailVerificationContext")
@route("/api/account/v1/email/verify_context")
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

@doc("An authenticated user reads their own account summary, the home of the self-service portal.\nThe target is fixed to `subject.id`. It carries no roles, and returns the last sign-in, the password change time,\nthe MFA state, and the required actions still outstanding. It is a separate contract from the AccountContext of the\nadministration shell.")
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

@doc("An authenticated user reads their own profile (self-service). The target is fixed to\n`subject.id`, so a cross-user read cannot arise structurally. `attributes` carries only what the user may read of\ntheir own, and `editable_attributes` carries the definitions they may edit.")
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

@doc("An authenticated user updates their own display name and the attributes marked `editable_by_user=true`\n(self-service). `subject.id` is fixed equal to `resource.id`. Attributes merge per key, and a key with\n`editable_by_user=false` is refused. Changing status, roles, or organization is for administrators and is not handled here.")
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

@doc("An authenticated user raises a change of their own primary email (self-service). It only sends a one-time link to the\nnew address; `User.email` is left alone. The target is fixed to `subject.id`, and an address already in use inside the\ntenant is refused.\nThe wording of the confirmation message is resolved from the Tenancy notification template catalogue\n(`template_key=EmailChangeConfirmation`), and the locale is decided in the order: the recipient User's locale attribute,\nthe tenant's `default_locale`, the system default locale. A tenant override takes precedence over the built-in default.\nBoth a plain text and an HTML body are always produced, and the confirmation link is built from the issuer URL of the request.")
@TypeSpec.OpenAPI.operationId("RequestEmailChange")
@route("/api/account/v1/email/change_request")
@post
op RequestEmailChange(
  @body body: IdMagic.Contract.AccountEmailChangeRequest
): RequestEmailChangeSuccess_204 | RequestEmailChangeError400 | RequestEmailChangeError403 | RequestEmailChangeError422;

model ConfirmEmailChangeHttpResponse {
  @doc("Always `ok`.")
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
  IdMagic.Contract.AccessDeniedError,
}

model ConfirmEmailChangeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ConfirmEmailChangeError403Body;
}

@doc("Consumes the one-time link sent to the new address and settles the primary email.\nThe token is itself the proof that the new address is owned, so no authenticated session is required. Possession of the\ntoken is the ground of authorization -- this is token-possession-authenticated rather than a self-service session -- so\nthe AuthenticatedSelf and session-based access classification does not apply. On settling, `email_verified` becomes true\nand a `verify_email` required action, if there is one, is cleared automatically.")
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

@doc("An authenticated user reads their own personal data as JSON (self-service, GDPR access).\nThe target is fixed to `subject.id`. At this stage it is generated synchronously and carries the profile and the consents.")
@TypeSpec.OpenAPI.operationId("ExportAccountData")
@route("/api/account/v1/data_export")
@get
op ExportAccountData(

): ExportAccountDataSuccess_200 | ExportAccountDataError403;

model ListAdminUsersSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminUserListResponse;
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

@doc("An administrator lists the users that have not been deleted by `preferred_username` ascending, with the id breaking ties,\nthrough bidirectional keyset pagination. `query` searches username, name, email, id, and roles case-insensitively as a\nsubstring across the tenant, and narrows together with `status`. The cursor comes from the Link response header of the\nresponse (`rel=\"prev\"` and `rel=\"next\"`).")
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

@doc("An administrator reads a user by id.")
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

@doc("An administrator creates a user with an initial password that satisfies the PasswordPolicy of the Authentication context.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("CreateAdminUser")
@route("/api/admin/v1/users")
@post
op CreateAdminUser(
  @body body: IdMagic.Contract.AdminUserCreateRequest
): CreateAdminUserSuccess_201 | CreateAdminUserError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateAdminUserError403 | CreateAdminUserError422;

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

@doc("An administrator streams a UTF-8 CSV upload, with or without a BOM, into a preview job that causes no side effects. The configurable transfer policy shared with User export defaults to 100,000 data rows, 64 MiB per file, and 64 KiB per field. The payload is kept in the immutable artifact store, and the job params hold only a tenant-scoped reference, the server-computed SHA-256, and the size. The header may carry the stable machine keys in any order and any subset, while an unknown column, a duplicate one, and a password or hash column each refuse the whole file.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("ImportAdminUsers")
@route("/api/admin/v1/users/imports")
@post
op ImportAdminUsers(
  @header contentType: "text/csv",
  @doc("The CSV document itself, streamed as the request body. There is no JSON envelope; this endpoint always runs a preview.")
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

@doc("An administrator applies a CSV by naming a succeeded preview job of the same tenant. The CSV is not sent again: the preview payload and `source_sha256` are read, and the same deterministic planner replans against the current repository state. The file as a whole succeeds row by row, while the profile, roles, required actions, and custom attributes of each row settle atomically as a single aggregate mutation.")
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

@doc("An administrator reads a CSV import job of the same tenant together with the per-row validation and application results. Row errors are kept in an immutable result artifact as page chunks of a fixed size, and are read in both directions through a signed opaque cursor bound to the tenant, the job, and the error ordinal, with a limit that defaults to 100 and tops out at 200. The job result JSON holds only the per-operation counts, the result artifact metadata, and `error_total`.")
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

@doc("An administrator starts a CSV export of the user list as an asynchronous job. The CSV header matches the machine keys that were requested, not display labels. The columns are confined to a subset of the built-in User allow list (13 core columns plus 27 built-in extension attributes as `attr:<key>`) and the `custom:<key>` tenant-defined attributes of the effective TenantUserAttributeSchema, and a status filter can be carried over. The generated payload uses the same UserCsvTransferPolicy and immutable artifact store as import, so a successful User export can be previewed without editing. It returns 202 and the export id.")
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

@doc("An administrator lists the User exports with their state, counts, expiry, and whether they can be downloaded. Neither the CSV itself nor any PII value is included.")
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

@doc("An administrator reads the state, progress, failure reason, and expiry of one User export.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:read"])
@TypeSpec.OpenAPI.operationId("GetUserExport")
@route("/api/admin/v1/users/exports/{export_id}")
@get
op GetUserExport(
  @path export_id: string
): GetUserExportSuccess_200 | GetUserExportError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetUserExportError403 | GetUserExportError503;

model DownloadUserExportFileSuccess_200 {
  @statusCode statusCode: 200;
  @doc("The generated export is CSV.")
  @header contentType: "text/csv; charset=utf-8";
  @doc("Filename the browser saves the download under: `attachment; filename=\"<name>.csv\"`.")
  @header("Content-Disposition") contentDisposition: string;
  @doc("The CSV bytes.")
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

@doc("An administrator downloads a finished User export as a `Content-Disposition: attachment`. Only a job that has succeeded and not yet expired can be downloaded.")
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

@doc("An administrator cancels a User export before it terminates. One that has already terminated is refused.")
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

@doc("An administrator starts a CSV export of the group list as an asynchronous job. The CSV header matches the machine keys that were requested, not display labels, and the columns are confined to a subset of the Group allow list: the built-in import-compatible columns plus the `custom:<key>` tenant-defined attributes of the TenantGroupAttributeSchema. The generated payload uses the same CsvTransferPolicy and immutable artifact store as import, so a successful Group export can be previewed without editing; `lifecycle_action` is written empty on every row, which is what keeps an unedited round trip at all rows unchanged. It returns 202 and the export id.")
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

@doc("An administrator streams a UTF-8 CSV upload, with or without a BOM, into a preview job that causes no side effects. The configurable transfer policy shared with Group export defaults to 100,000 data rows, 64 MiB per file, and 64 KiB per field. The payload is kept in the immutable artifact store, and the job params hold only a tenant-scoped reference, the server-computed SHA-256, and the size. The header may carry the stable machine keys in any order and any subset, while an unknown column, a duplicate one, and a password or hash column each refuse the whole file. A row resolves its target by `id` first and by `name` second; `membership_type` may be chosen at creation but never changed afterwards, the two dynamic rule columns are validated as the final state they combine into, a `custom:<key>` cell is parsed in the canonical lexical form its declared type fixes, and a Group an external source owns is refused.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("ImportAdminGroups")
@route("/api/admin/v1/groups/imports")
@post
op ImportAdminGroups(
  @header contentType: "text/csv",
  @doc("The CSV document itself, streamed as the request body. There is no JSON envelope; this endpoint always runs a preview.")
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

@doc("An administrator applies a CSV by naming a succeeded preview job of the same tenant. The CSV is not sent again: the preview payload and `source_sha256` are read, and the same deterministic planner replans against the current repository state. The file as a whole succeeds row by row, while the name, description, contact address, roles, custom attributes, and dynamic rule of each row settle atomically as a single aggregate mutation. A row that carries `lifecycle_action=delete` deletes the Group, releases every membership in it by cascade, and writes its audit record in that same transaction; the deletion is irreversible, so a Group the CSV does not name is never deleted.")
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

@doc("An administrator reads a CSV group import job of the same tenant together with the per-row validation and application results. Row errors are kept in an immutable result artifact as page chunks of a fixed size, and are read in both directions through a signed opaque cursor bound to the tenant, the job, and the error ordinal, with a limit that defaults to 100 and tops out at 200. The job result JSON holds only the per-operation counts, the deletion counts kept apart from them, the result artifact metadata, and `error_total`.")
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

@doc("An administrator lists the Group exports with their state, counts, expiry, and whether they can be downloaded.")
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

@doc("An administrator reads the state, progress, failure reason, and expiry of one Group export.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("GetGroupExport")
@route("/api/admin/v1/groups/exports/{export_id}")
@get
op GetGroupExport(
  @path export_id: string
): GetGroupExportSuccess_200 | GetGroupExportError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetGroupExportError403 | GetGroupExportError503;

model DownloadGroupExportFileSuccess_200 {
  @statusCode statusCode: 200;
  @doc("The generated export is CSV.")
  @header contentType: "text/csv; charset=utf-8";
  @doc("Filename the browser saves the download under: `attachment; filename=\"<name>.csv\"`.")
  @header("Content-Disposition") contentDisposition: string;
  @doc("The CSV bytes.")
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

@doc("An administrator downloads a finished Group export as a `Content-Disposition: attachment`. Only a job that has succeeded and not yet expired can be downloaded.")
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

@doc("An administrator cancels a Group export before it terminates. One that has already terminated is refused.")
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

@doc("An administrator starts a CSV export of the members of one Group. The `group_id` comes from the path and only that group's members are exported, the same per-group shape Entra, Okta, and Google use. It returns 202 and the export id.")
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

@doc("An administrator lists the member exports of one Group.")
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

@doc("An administrator reads the state of one member export of a Group. An export id belonging to another group does not resolve.")
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
  @doc("The generated export is CSV.")
  @header contentType: "text/csv; charset=utf-8";
  @doc("Filename the browser saves the download under: `attachment; filename=\"<name>.csv\"`.")
  @header("Content-Disposition") contentDisposition: string;
  @doc("The CSV bytes.")
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

@doc("An administrator downloads a finished member export as a `Content-Disposition: attachment`. Only a job that has succeeded and not yet expired can be downloaded.")
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

@doc("An administrator cancels a member export before it terminates. One that has already terminated is refused.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:read"])
@TypeSpec.OpenAPI.operationId("CancelGroupMemberExport")
@route("/api/admin/v1/groups/{group_id}/members/exports/{export_id}/cancel")
@post
op CancelGroupMemberExport(
  @path group_id: string,
  @path export_id: string,): CancelGroupMemberExportSuccess_200 | CancelGroupMemberExportError400 | IdMagic.Contract.AuthenticationRequiredResponse | CancelGroupMemberExportError403 | CancelGroupMemberExportError503;

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

@doc("An administrator updates a user's profile, `email_verified`, and roles.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("UpdateAdminUser")
@route("/api/admin/v1/users/{user_id}")
@patch
op UpdateAdminUser(
  @path user_id: string,
  @body body: IdMagic.Contract.AdminUserUpdateRequest
): UpdateAdminUserSuccess_200 | UpdateAdminUserError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateAdminUserError403 | UpdateAdminUserError422;

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

@doc("An administrator disables a user, refusing both new sign-ins and the use of existing sessions.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("DisableAdminUser")
@route("/api/admin/v1/users/{user_id}/disable")
@post
op DisableAdminUser(
  @path user_id: string,): DisableAdminUserSuccess_204 | DisableAdminUserError400 | IdMagic.Contract.AuthenticationRequiredResponse | DisableAdminUserError403 | DisableAdminUserError422;

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

@doc("An administrator re-enables a disabled user.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("EnableAdminUser")
@route("/api/admin/v1/users/{user_id}/enable")
@post
op EnableAdminUser(
  @path user_id: string,): EnableAdminUserSuccess_204 | EnableAdminUserError400 | IdMagic.Contract.AuthenticationRequiredResponse | EnableAdminUserError403;

model DeleteAdminUserHttpRequest {
  @doc("Same meaning as `?purge=true`: skip the soft delete and anonymize immediately. Omitted means a soft delete.")
  force?: boolean;
  @doc("Free-text note kept for the audit trail.")
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

@doc("An administrator deletes a user. By default this is a soft delete: the status moves to\n`pending_deletion` and the PII, Consents, RefreshTokens, and Sessions are preserved, so a Restore is possible\nwithin the grace period. With `purge=true` the anonymize cascade runs at once and leaves a tombstone. Both are\nidempotent: calling again for a user already in the same end state is a no-op that returns 204. A caller cannot\ndelete themselves, which keeps an admin or system_admin from destroying their own access.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("DeleteAdminUser")
@route("/api/admin/v1/users/{user_id}")
@delete
op DeleteAdminUser(
  @path user_id: string,
  @doc("`true` skips the soft delete and anonymizes immediately, the same as `force` in the body. The body itself is optional.")
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

@doc("An administrator restores a user in `pending_deletion`. The status returns to `active`\nand the user is restored with their PII and credentials intact. A user past the grace period, or one not in\n`pending_deletion`, gives InvalidRequestError. It returns the restored user. Acting on oneself is rejected in the\nuse case, mirroring how self-deletion is treated.")
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

@doc("An administrator gives a target user an action to be forced at their next sign-in (a RequiredAction).\nA user given `update_password` is sent to the change-password screen at their next sign-in, and changing it clears\nthe action automatically. It is idempotent: if the action is already there, the current state is returned. Administrators only.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("SetUserRequiredAction")
@route("/api/admin/v1/users/{user_id}/required_actions")
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
  IdMagic.Contract.AccessDeniedError,
}

model ClearUserRequiredActionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ClearUserRequiredActionError403Body;
}

@doc("An administrator clears a forced action from a target user. It is idempotent: if the action was not there,\nthe current state is returned. Administrators only. The automatic clearing of `update_password` when the user changes\ntheir own password happens in ChangePassword and ResetPasswordWithToken instead.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("ClearUserRequiredAction")
@route("/api/admin/v1/users/{user_id}/required_actions/{action}")
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

@doc("An administrator lists the groups of their tenant by name ascending, with the id breaking ties, through bidirectional keyset\npagination. Each group carries its `member_count`. The cursor comes from the Link response header of the response\n(`rel=\"prev\"` and `rel=\"next\"`).")
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
  IdMagic.Contract.AccessDeniedError,
}

model GetGroupError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetGroupError403Body;
}

@doc("An administrator reads one group together with its member list. A group of another tenant is treated as nonexistent.")
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

@doc("An administrator creates a group in their tenant. The name is unique within the tenant. The email is only checked for its form and carries no uniqueness constraint, and the attributes are validated against the tenant's TenantGroupAttributeSchema.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("CreateGroup")
@route("/api/admin/v1/groups")
@post
op CreateGroup(
  @body body: IdMagic.Contract.GroupCreateRequest
): CreateGroupSuccess_201 | CreateGroupError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateGroupError403 | CreateGroupError422;

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

@doc("An administrator updates the name, description, email, attributes, or roles of a group.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["groups:write"])
@TypeSpec.OpenAPI.operationId("UpdateGroup")
@route("/api/admin/v1/groups/{group_id}")
@patch
op UpdateGroup(
  @path group_id: string,
  @body body: IdMagic.Contract.GroupUpdateRequest
): UpdateGroupSuccess_200 | UpdateGroupError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateGroupError403 | UpdateGroupError422;

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

@doc("An administrator deletes a group. The memberships in it are released by cascade, each as its own GroupMemberRemoved before the final GroupDeleted, and the group quota the group held is released. The deletion is immediate and irreversible; there is no reservation state to restore from. The CSV import path reaches the same deletion through the explicit `lifecycle_action=delete` column and settles the release, the deletion, and the audit record in one transaction per row.")
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
  IdMagic.Contract.AccessDeniedError,
}

model AddGroupMemberError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: AddGroupMemberError403Body;
}

@doc("An administrator makes a User a member of a group. Only a User of the same tenant is permitted, and an existing membership is a no-op, so the call is idempotent.")
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
  IdMagic.Contract.AccessDeniedError,
}

model RemoveGroupMemberError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RemoveGroupMemberError403Body;
}

@doc("An administrator removes a User from a group. A user who is not a member is a no-op, so the call is idempotent.")
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

@doc("An administrator validates and saves the restricted CEL rule of a dynamic group. An unknown attribute, a type mismatch, a non-boolean result, syntax outside the permitted subset, and exceeding a ceiling are all refused.")
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

@doc("Evaluates an unsaved CEL against at most 100 Users without any side effect, and returns only what differs from the current membership.")
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
  IdMagic.Contract.AccessDeniedError,
}

model EnableDynamicGroupRuleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableDynamicGroupRuleError403Body;
}

@doc("Activates a validated rule, invalidates the membership of the old version at once, and schedules a job that re-evaluates everything.")
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
  IdMagic.Contract.AccessDeniedError,
}

model DisableDynamicGroupRuleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableDynamicGroupRuleError403Body;
}

@doc("Deactivates a rule, so every dynamic membership is excluded from authorization and assignment decisions at once.")
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
  IdMagic.Contract.AccessDeniedError,
}

model ListUserGroupsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListUserGroupsError403Body;
}

@doc("An administrator reads the groups a named User belongs to and the breakdown of their effective roles: direct, inherited from groups, and the union.")
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

@doc("An administrator lists the Agents of their tenant by name ascending, with the id breaking ties, through bidirectional keyset\npagination. The cursor comes from the Link response header of the response (`rel=\"prev\"` and `rel=\"next\"`).")
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
  IdMagic.Contract.AccessDeniedError,
}

model GetAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAgentError403Body;
}

@doc("An administrator reads one Agent. An Agent of another tenant is treated as nonexistent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:read"])
@TypeSpec.OpenAPI.operationId("GetAgent")
@route("/api/admin/v1/agents/{agent_id}")
@get
op GetAgent(
  @path agent_id: string
): GetAgentSuccess_200 | GetAgentError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetAgentError403;

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

@doc("An administrator registers an Agent in their tenant. The name is unique within the tenant, and leaving out `owner_user_id` makes the caller the owner.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("RegisterAgent")
@route("/api/admin/v1/agents")
@post
op RegisterAgent(
  @body body: IdMagic.Contract.AgentRegisterRequest
): RegisterAgentSuccess_201 | RegisterAgentError400 | IdMagic.Contract.AuthenticationRequiredResponse | RegisterAgentError403 | RegisterAgentError422;

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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAgentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAgentError403Body;
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

@doc("An administrator updates the name, description, kind, `owner_user_id`, or roles of an Agent. Changing `owner_user_id` transfers ownership.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("UpdateAgent")
@route("/api/admin/v1/agents/{agent_id}")
@patch
op UpdateAgent(
  @path agent_id: string,
  @body body: IdMagic.Contract.AgentUpdateRequest
): UpdateAgentSuccess_200 | UpdateAgentError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateAgentError403 | UpdateAgentError422;

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

@doc("An administrator suspends an Agent reversibly.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("DisableAgent")
@route("/api/admin/v1/agents/{agent_id}/disable")
@post
op DisableAgent(
  @path agent_id: string,): DisableAgentSuccess_204 | DisableAgentError400 | DisableAgentError403;

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

@doc("An administrator returns a suspended Agent to service.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("EnableAgent")
@route("/api/admin/v1/agents/{agent_id}/enable")
@post
op EnableAgent(
  @path agent_id: string,): EnableAgentSuccess_204 | EnableAgentError400 | EnableAgentError403;

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

@doc("An administrator kills an Agent. This is one-way and terminal; there is no way back.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("KillAgent")
@route("/api/admin/v1/agents/{agent_id}/kill")
@post
op KillAgent(
  @path agent_id: string,): KillAgentSuccess_204 | KillAgentError400 | KillAgentError403;

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

@doc("An administrator deletes an Agent. Its bindings are released by cascade. A killed Agent cannot be deleted, fail-closed: deleting it would let the OAuth2Client it was bound to fetch tokens again through client_credentials without limit.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("DeleteAgent")
@route("/api/admin/v1/agents/{agent_id}")
@delete
op DeleteAgent(
  @path agent_id: string
): DeleteAgentSuccess_204 | DeleteAgentError400 | DeleteAgentError403;

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

@doc("An administrator binds an OAuth2Client to an Agent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("BindAgentCredential")
@route("/api/admin/v1/agents/{agent_id}/credentials")
@post
op BindAgentCredential(
  @path agent_id: string,
  @body body: IdMagic.Contract.AgentCredentialBindRequest
): BindAgentCredentialSuccess_204 | BindAgentCredentialError400 | IdMagic.Contract.AuthenticationRequiredResponse | BindAgentCredentialError403;

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

@doc("An administrator unbinds an OAuth2Client from an Agent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["agents:write"])
@TypeSpec.OpenAPI.operationId("UnbindAgentCredential")
@route("/api/admin/v1/agents/{agent_id}/credentials/{client_id}")
@delete
op UnbindAgentCredential(
  @path agent_id: string,
  @path client_id: string
): UnbindAgentCredentialSuccess_204 | UnbindAgentCredentialError400 | IdMagic.Contract.AuthenticationRequiredResponse | UnbindAgentCredentialError403;
}

}
