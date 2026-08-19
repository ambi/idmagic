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

@doc("An administrator lists the Applications of their tenant by name ascending, with the id breaking ties, through bidirectional keyset\npagination. The cursor comes from the Link response header of the response (`rel=\"prev\"` and `rel=\"next\"`).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:read"])
@TypeSpec.OpenAPI.operationId("ListAdminApplications")
@route("/api/admin/v1/applications")
@get
op ListAdminApplications(
  @query cursor?: string,
  @query limit?: int64
): ListAdminApplicationsSuccess_200 | ListAdminApplicationsError403 | ListAdminApplicationsError400;

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
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminApplicationError403Body;
}

@doc("An administrator reads an Application, including the real settings resolved through its protocol relation. The detail screen of an OIDC or SAML application shows these registered counterpart settings side by side with the IdMagic setup guidance Tenancy.GetAdminIntegrationEndpoints returns, and never shows a client secret again. Another tenant's is treated as nonexistent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:read"])
@TypeSpec.OpenAPI.operationId("GetAdminApplication")
@route("/api/admin/v1/applications/{id}")
@get
op GetAdminApplication(
  @path id: string
): GetAdminApplicationSuccess_200 | GetAdminApplicationError400 | GetAdminApplicationError403;

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

@doc("An administrator creates an Application in one call. The protocol row for its kind is prepared, and the Application together with the `application_id` relation is published to the catalogue in a single transaction. The protocol kind cannot be changed afterwards.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("CreateAdminApplication")
@route("/api/admin/v1/applications")
@post
op CreateAdminApplication(
  @body body: IdMagic.Contract.AdminApplicationCreateRequest
): CreateAdminApplicationSuccess_201 | CreateAdminApplicationError400 | CreateAdminApplicationError403 | CreateAdminApplicationError422;

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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAdminApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminApplicationError403Body;
}

@doc("An administrator updates the metadata of an Application.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UpdateAdminApplication")
@route("/api/admin/v1/applications/{id}")
@patch
op UpdateAdminApplication(
  @path id: string,
  @body body: IdMagic.Contract.AdminApplicationUpdateRequest
): UpdateAdminApplicationSuccess_200 | UpdateAdminApplicationError400 | UpdateAdminApplicationError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model UploadApplicationIconError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UploadApplicationIconError403Body;
}

@doc("An administrator uploads the icon image of an Application as multipart. PNG, JPEG, WebP, and GIF are accepted, up to 256 KiB, and the format is checked against the magic bytes.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UploadApplicationIcon")
@route("/api/admin/v1/applications/{id}/icon")
@post
op UploadApplicationIcon(
  @path id: string,
  @header contentType: "multipart/form-data",
  @multipartBody body: {
    @doc("The image to store, under the form field name `file`.")
    file: HttpPart<File>,
  }
): UploadApplicationIconSuccess_200 | UploadApplicationIconError400 | UploadApplicationIconError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model DeleteApplicationIconError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteApplicationIconError403Body;
}

@doc("An administrator deletes the stored icon of an Application. `icon_url` is no longer returned afterwards.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("DeleteApplicationIcon")
@route("/api/admin/v1/applications/{id}/icon")
@delete
op DeleteApplicationIcon(
  @path id: string
): DeleteApplicationIconSuccess_200 | DeleteApplicationIconError400 | DeleteApplicationIconError403;

model GetApplicationIconSuccess_200 {
  @statusCode statusCode: 200;
  @doc("Fixed to the validated format of the stored icon, and served with `X-Content-Type-Options: nosniff`.")
  @header contentType: "image/png" | "image/jpeg" | "image/webp" | "image/gif";
  @doc("The icon bytes.")
  @body body: bytes;
}

union GetApplicationIconError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetApplicationIconError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetApplicationIconError400Body;
}

@doc("Public interface serving a stored Application icon. The Content-Type is fixed to the validated format and `X-Content-Type-Options: nosniff` is set. An object of another tenant, and a deleted one, are both treated as nonexistent.")
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

@doc("An administrator deletes an Application, idempotently. The related protocol rows are cascade-deleted in the same transaction.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("DeleteAdminApplication")
@route("/api/admin/v1/applications/{id}")
@delete
op DeleteAdminApplication(
  @path id: string
): DeleteAdminApplicationSuccess_204 | DeleteAdminApplicationError400 | DeleteAdminApplicationError403;

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

@doc("An administrator updates the settings of the OAuth2 client the OIDC protocol of an Application points at. `rules` and\n`sub_source_attribute` override claim release, so a rule referring to an attribute whose visibility is `private`, or to a\nreserved claim type, is refused.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UpdateApplicationOidcConfig")
@route("/api/admin/v1/applications/{id}/oidc")
@patch
op UpdateApplicationOidcConfig(
  @path id: string,
  @body body: IdMagic.Contract.ApplicationOidcConfigUpdateRequest
): UpdateApplicationOidcConfigSuccess_204 | UpdateApplicationOidcConfigError400 | UpdateApplicationOidcConfigError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model RotateApplicationClientSecretError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RotateApplicationClientSecretError403Body;
}

@doc("For compatibility, an administrator rotates the client secret of an Application's OIDC protocol the old way. The previous credential stays valid for the overlap given, and zero days means revoking it at once. The newer administration UI uses IssueApplicationClientSecret and RevokeApplicationClientSecret instead.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("RotateApplicationClientSecret")
@route("/api/admin/v1/applications/{id}/oidc/rotate-secret")
@post
op RotateApplicationClientSecret(
  @path id: string,
  @body body: IdMagic.Contract.RotateApplicationClientSecretRequest
): RotateApplicationClientSecretSuccess_200 | RotateApplicationClientSecretError400 | RotateApplicationClientSecretError403;

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

@doc("An administrator issues an additional credential with an expiry on the secret-based OIDC client of an Application. Existing credentials are left alone, and when two Active credentials already exist the request is refused atomically. A client that is not secret-based is refused.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("IssueApplicationClientSecret")
@route("/api/admin/v1/applications/{id}/oidc/client-secrets")
@post
op IssueApplicationClientSecret(
  @path id: string,
  @body body: IdMagic.Contract.IssueApplicationClientSecretRequest
): IssueApplicationClientSecretSuccess_201 | IssueApplicationClientSecretError400 | IssueApplicationClientSecretError403 | IssueApplicationClientSecretError422;

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
  IdMagic.Contract.AccessDeniedError,
}

model RevokeApplicationClientSecretError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeApplicationClientSecretError403Body;
}

@doc("An administrator revokes one named credential of an Application's secret-based OIDC client, at once. Revoking an already-revoked credential succeeds idempotently, and a credential of another client, or one that does not exist, is refused.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("RevokeApplicationClientSecret")
@route("/api/admin/v1/applications/{id}/oidc/client-secrets/{credential_id}")
@delete
op RevokeApplicationClientSecret(
  @path id: string,
  @path credential_id: string
): RevokeApplicationClientSecretSuccess_200 | RevokeApplicationClientSecretError400 | RevokeApplicationClientSecretError403;

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

@doc("An administrator updates the settings of the relying party the WS-Fed protocol of an Application points at. `rules` and\n`name_id_source` override claim release, so a rule referring to an attribute whose visibility is `private`, or to a\nreserved claim type, is refused.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UpdateApplicationWsFedConfig")
@route("/api/admin/v1/applications/{id}/wsfed")
@patch
op UpdateApplicationWsFedConfig(
  @path id: string,
  @body body: IdMagic.Contract.ApplicationWsFedConfigUpdateRequest
): UpdateApplicationWsFedConfigSuccess_204 | UpdateApplicationWsFedConfigError400 | UpdateApplicationWsFedConfigError403;

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

@doc("An administrator partially updates the service provider settings and the IdP profile binding the SAML protocol of an Application\npoints at. Turning on the signature requirement makes a verifiable certificate mandatory. `rules` and `name_id_source` override\nclaim release, so a rule referring to an attribute whose visibility is `private`, or to a reserved claim type, is refused.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UpdateApplicationSamlConfig")
@route("/api/admin/v1/applications/{id}/saml")
@patch
op UpdateApplicationSamlConfig(
  @path id: string,
  @body body: IdMagic.Contract.ApplicationSamlConfigUpdateRequest
): UpdateApplicationSamlConfigSuccess_204 | UpdateApplicationSamlConfigError400 | UpdateApplicationSamlConfigError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model ListApplicationAssignmentsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListApplicationAssignmentsError403Body;
}

@doc("An administrator lists the assignments of an Application by `subject_type` and `subject_id` ascending, through bidirectional keyset\npagination. The cursor comes from the Link response header of the response (`rel=\"prev\"` and `rel=\"next\"`).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:read"])
@TypeSpec.OpenAPI.operationId("ListApplicationAssignments")
@route("/api/admin/v1/applications/{id}/assignments")
@get
op ListApplicationAssignments(
  @path id: string,
  @query cursor?: string,
  @query limit?: int64
): ListApplicationAssignmentsSuccess_200 | ListApplicationAssignmentsError400 | ListApplicationAssignmentsError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model AssignApplicationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: AssignApplicationError403Body;
}

@doc("An administrator assigns a user or a group to an Application.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("AssignApplication")
@route("/api/admin/v1/applications/{id}/assignments")
@post
op AssignApplication(
  @path id: string,
  @body body: IdMagic.Contract.ApplicationAssignmentRequest
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

@doc("An administrator removes an assignment from an Application, idempotently.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UnassignApplication")
@route("/api/admin/v1/applications/{id}/assignments/{subject_type}/{subject_id}")
@delete
op UnassignApplication(
  @path id: string,
  @path subject_type: string,
  @path subject_id: string
): UnassignApplicationSuccess_204 | UnassignApplicationError400 | UnassignApplicationError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model GetAppSignInPolicyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAppSignInPolicyError403Body;
}

@doc("An administrator reads the sign-in policy of an Application. It returns the application's own policy, the tenant default, the effective rule sequence that actually applies after the override, and a warning flag for a policy weaker than the default. When nothing is set it returns empty rules.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:read"])
@TypeSpec.OpenAPI.operationId("GetAppSignInPolicy")
@route("/api/admin/v1/applications/{id}/sign-in-policy")
@get
op GetAppSignInPolicy(
  @path id: string
): GetAppSignInPolicySuccess_200 | GetAppSignInPolicyError400 | GetAppSignInPolicyError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAppSignInPolicyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAppSignInPolicyError403Body;
}

@doc("An administrator replaces the sign-in policy of an Application. A condition holding an invalid CIDR is refused.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UpdateAppSignInPolicy")
@route("/api/admin/v1/applications/{id}/sign-in-policy")
@put
op UpdateAppSignInPolicy(
  @path id: string,
  @body body: IdMagic.Contract.AppSignInPolicyRequest
): UpdateAppSignInPolicySuccess_200 | UpdateAppSignInPolicyError400 | UpdateAppSignInPolicyError403;

model GetTenantDefaultSignInPolicySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantDefaultSignInPolicyResponse;
}

union GetTenantDefaultSignInPolicyError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetTenantDefaultSignInPolicyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetTenantDefaultSignInPolicyError403Body;
}

@doc("An administrator reads the tenant default sign-in policy together with the number of active users not enrolled in MFA. When nothing is set it returns a policy with empty rules.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("GetTenantDefaultSignInPolicy")
@route("/api/admin/v1/default-sign-in-policy")
@get
op GetTenantDefaultSignInPolicy(

): GetTenantDefaultSignInPolicySuccess_200 | GetTenantDefaultSignInPolicyError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantDefaultSignInPolicyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantDefaultSignInPolicyError403Body;
}

@doc("An administrator replaces the tenant default sign-in policy. Making MFA mandatory states the enforcement start time, the grace during which an enrollment bypass may be used, and whether administrator approval is allowed. An invalid CIDR, an enforcement start time in the past, and a non-positive grace are all refused. Saving empty rules returns the default to allow-all.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UpdateTenantDefaultSignInPolicy")
@route("/api/admin/v1/default-sign-in-policy")
@put
op UpdateTenantDefaultSignInPolicy(
  @body body: IdMagic.Contract.TenantDefaultSignInPolicyRequest
): UpdateTenantDefaultSignInPolicySuccess_200 | UpdateTenantDefaultSignInPolicyError400 | UpdateTenantDefaultSignInPolicyError403;

model ListApplicationCategoriesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminApplicationCategoryListResponse;
}

union ListApplicationCategoriesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListApplicationCategoriesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListApplicationCategoriesError403Body;
}

@doc("An administrator lists the ApplicationCategories of their tenant by position ascending.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:read"])
@TypeSpec.OpenAPI.operationId("ListApplicationCategories")
@route("/api/admin/v1/application-categories")
@get
op ListApplicationCategories(

): ListApplicationCategoriesSuccess_200 | ListApplicationCategoriesError403;

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

@doc("An administrator creates an ApplicationCategory.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("CreateApplicationCategory")
@route("/api/admin/v1/application-categories")
@post
op CreateApplicationCategory(
  @body body: IdMagic.Contract.AdminApplicationCategoryRequest
): CreateApplicationCategorySuccess_201 | CreateApplicationCategoryError400 | CreateApplicationCategoryError422 | CreateApplicationCategoryError403;

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

@doc("An administrator updates the name or the position of an ApplicationCategory.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("UpdateApplicationCategory")
@route("/api/admin/v1/application-categories/{id}")
@patch
op UpdateApplicationCategory(
  @path id: string,
  @body body: IdMagic.Contract.AdminApplicationCategoryRequest
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

@doc("An administrator deletes an ApplicationCategory, idempotently. It is removed from the Applications it was assigned to as well.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("DeleteApplicationCategory")
@route("/api/admin/v1/application-categories/{id}")
@delete
op DeleteApplicationCategory(
  @path id: string
): DeleteApplicationCategorySuccess_204 | DeleteApplicationCategoryError400 | DeleteApplicationCategoryError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model SetApplicationCategoriesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SetApplicationCategoriesError403Body;
}

@doc("An administrator replaces the set of categories assigned to an Application. Only categories that already exist in their tenant are accepted.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["applications:write"])
@TypeSpec.OpenAPI.operationId("SetApplicationCategories")
@route("/api/admin/v1/applications/{id}/categories")
@put
op SetApplicationCategories(
  @path id: string,
  @body body: IdMagic.Contract.ApplicationCategoriesRequest
): SetApplicationCategoriesSuccess_200 | SetApplicationCategoriesError400 | SetApplicationCategoriesError422 | SetApplicationCategoriesError403;

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

@doc("A user lists the Applications assigned to them and visible; hidden assignments are excluded.\nA manual order (ApplicationOrdering) takes precedence when one is saved, and name ascending is used otherwise. The `categories`\nof the response carry the tenant's category definitions by position ascending, and each app is sorted into a section by its `category_ids`.")
@TypeSpec.OpenAPI.operationId("ListMyApplications")
@route("/api/account/v1/applications")
@get
op ListMyApplications(

): ListMyApplicationsSuccess_200 | ListMyApplicationsError403;

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

@doc("A user reads the manual portal order they saved. When nothing is saved it returns an empty array.")
@TypeSpec.OpenAPI.operationId("GetMyApplicationOrder")
@route("/api/account/v1/applications/order")
@get
op GetMyApplicationOrder(

): GetMyApplicationOrderSuccess_200 | GetMyApplicationOrderError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model ReorderMyApplicationsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ReorderMyApplicationsError403Body;
}

@doc("A user saves their own manual portal order. `application_ids` may hold only apps assigned to them and visible; anything else is refused.")
@TypeSpec.OpenAPI.operationId("ReorderMyApplications")
@route("/api/account/v1/applications/order")
@put
op ReorderMyApplications(
  @body body: IdMagic.Contract.ReorderMyApplicationsRequest
): ReorderMyApplicationsSuccess_200 | ReorderMyApplicationsError400 | ReorderMyApplicationsError403;
}

}
