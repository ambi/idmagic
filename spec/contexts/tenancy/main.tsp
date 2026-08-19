import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Tenancy")
namespace IdMagic.Tenancy {

namespace Operations {
model ListTenantsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantListResponse;
}

union ListTenantsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListTenantsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListTenantsError403Body;
}

@doc("A system_admin reads the tenant list. It crosses tenants, so it is gated on the system_admin\nrole and called from the system console (/system). It is reached through the same handler from both the\nbare path that resolves to the default tenant and the control-plane path under /realms/default, which keeps the\ncookie session consistent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:read"])
@TypeSpec.OpenAPI.operationId("ListTenants")
@route("/api/admin/v1/tenants")
@get
op ListTenants(

): ListTenantsSuccess_200 | ListTenantsError403;

model GetTenantSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantSummaryResponse;
}

union GetTenantError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetTenantError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetTenantError400Body;
}

union GetTenantError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetTenantError403Body;
}

@doc("A system_admin reads one tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:read"])
@TypeSpec.OpenAPI.operationId("GetTenant")
@route("/api/admin/v1/tenants/{tenant_id}")
@get
op GetTenant(
  @path tenant_id: string
): GetTenantSuccess_200 | GetTenantError400 | GetTenantError403;

model CreateTenantSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.TenantSummaryResponse;
}

union CreateTenantError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateTenantError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateTenantError400Body;
}

union CreateTenantError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CreateTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateTenantError403Body;
}

@doc("A system_admin creates a tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:write"])
@TypeSpec.OpenAPI.operationId("CreateTenant")
@route("/api/admin/v1/tenants")
@post
op CreateTenant(
  @body body: IdMagic.Contract.TenantCreateRequest
): CreateTenantSuccess_201 | CreateTenantError400 | CreateTenantError403;

model UpdateTenantSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantSummaryResponse;
}

union UpdateTenantError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateTenantError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantError400Body;
}

union UpdateTenantError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantError403Body;
}

union UpdateTenantError422Body {
  IdMagic.Contract.PolicyOverrideWeakerError,
}

model UpdateTenantError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantError422Body;
}

@doc("A system_admin updates the display name of a tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:write"])
@TypeSpec.OpenAPI.operationId("UpdateTenant")
@route("/api/admin/v1/tenants/{tenant_id}")
@patch
op UpdateTenant(
  @path tenant_id: string,
  @body body: IdMagic.Contract.TenantUpdateRequest
): UpdateTenantSuccess_200 | UpdateTenantError400 | UpdateTenantError403 | UpdateTenantError422;

model DisableTenantSuccess_204 {
  @statusCode statusCode: 204;
}

union DisableTenantError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DisableTenantError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DisableTenantError400Body;
}

union DisableTenantError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DisableTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableTenantError403Body;
}

@doc("A system_admin disables a tenant other than default. /authorize, /token, /login, and the rest all give invalid_request afterwards.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:write"])
@TypeSpec.OpenAPI.operationId("DisableTenant")
@route("/api/admin/v1/tenants/{tenant_id}/disable")
@post
op DisableTenant(
  @path tenant_id: string,): DisableTenantSuccess_204 | DisableTenantError400 | DisableTenantError403;

model EnableTenantSuccess_204 {
  @statusCode statusCode: 204;
}

union EnableTenantError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model EnableTenantError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: EnableTenantError400Body;
}

union EnableTenantError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model EnableTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableTenantError403Body;
}

@doc("A system_admin re-enables a disabled tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:write"])
@TypeSpec.OpenAPI.operationId("EnableTenant")
@route("/api/admin/v1/tenants/{tenant_id}/enable")
@post
op EnableTenant(
  @path tenant_id: string,): EnableTenantSuccess_204 | EnableTenantError400 | EnableTenantError403;

model SetTenantEndpointStyleHttpRequest {
  endpoint_style: IdMagic.Contract.TenantEndpointStyle;
}

model SetTenantEndpointStyleSuccess_204 {
  @statusCode statusCode: 204;
}

union SetTenantEndpointStyleError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model SetTenantEndpointStyleError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: SetTenantEndpointStyleError400Body;
}

union SetTenantEndpointStyleError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model SetTenantEndpointStyleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SetTenantEndpointStyleError403Body;
}

@doc("A system_admin switches the shape of a tenant's canonical location.\nIt is kept apart from the ordinary attribute update (UpdateTenant) because it is a destructive operation that\nremakes the tenant's identity: the issuer changes, which breaks `iss` verification of already-issued tokens and the\nconfiguration of every relying party; the WebAuthn RP ID changes, which invalidates every existing passkey; and the\ncookie scope changes, which cuts sessions that are in progress.\n`Subdomain` can be chosen only in a deployment where `tenant_base_domain` is configured; without it the request gives\nInvalidRequestError. The realm is immutable, so the hostname after the switch follows from the existing realm.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:write"])
@TypeSpec.OpenAPI.operationId("SetTenantEndpointStyle")
@route("/api/admin/v1/tenants/{tenant_id}/endpoint_style")
@put
op SetTenantEndpointStyle(
  @path tenant_id: string,
  @body body: SetTenantEndpointStyleHttpRequest
): SetTenantEndpointStyleSuccess_204 | SetTenantEndpointStyleError400 | SetTenantEndpointStyleError403;

model UpdateTenantQuotaSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantQuota;
}

union UpdateTenantQuotaError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateTenantQuotaError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantQuotaError400Body;
}

union UpdateTenantQuotaError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantQuotaError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantQuotaError403Body;
}

@doc("A system_admin updates the Quota of one tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:write"])
@TypeSpec.OpenAPI.operationId("UpdateTenantQuota")
@route("/api/admin/v1/tenants/{tenant_id}/quota")
@put
op UpdateTenantQuota(
  @path tenant_id: string,
  @body body: IdMagic.Contract.TenantQuotaUpdateRequest
): UpdateTenantQuotaSuccess_200 | UpdateTenantQuotaError400 | UpdateTenantQuotaError403;

model GetAdminSettingsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminSettingsResponse;
}

union GetAdminSettingsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminSettingsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminSettingsError403Body;
}

@doc("A tenant administrator reads their own tenant's settings: the display name, the default locale, and the password policy override.\nThe target is fixed to `context.tenant_id`, the resolved tenant, so a cross-tenant read cannot arise structurally.\nIt is a separate route from `/api/admin/tenants`, which only a system_admin may use, and it is open to an admin as well.\nThe response carries the current values of the Authentication context's PasswordPolicy (`password_policy_defaults`) and\nthe locales the catalogue ships translations for (`supported_locales`), so the UI can show concretely what applies when\nnothing is overridden and which default locales can be chosen.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("GetAdminSettings")
@route("/api/admin/v1/settings")
@get
op GetAdminSettings(

): GetAdminSettingsSuccess_200 | GetAdminSettingsError403;

model GetAdminIntegrationEndpointsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminIntegrationEndpointCatalog;
}

union GetAdminIntegrationEndpointsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminIntegrationEndpointsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminIntegrationEndpointsError403Body;
}

@doc("A tenant administrator reads, in one call, the canonical IdMagic integration details to feed into an external OIDC RP, SAML SP, WS-Federation RP, SCIM client, or management client. The server derives every URL from the canonical issuer of the resolved request tenant, so the frontend concatenates neither realm nor origin. It carries no secret and is returned with `Cache-Control: no-store`.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("GetAdminIntegrationEndpoints")
@route("/api/admin/v1/integration-endpoints")
@get
op GetAdminIntegrationEndpoints(

): GetAdminIntegrationEndpointsSuccess_200 | GetAdminIntegrationEndpointsError403;

model UpdateAdminSettingsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminSettingsResponse;
}

union UpdateAdminSettingsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateAdminSettingsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminSettingsError400Body;
}

union UpdateAdminSettingsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAdminSettingsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminSettingsError403Body;
}

@doc("A tenant administrator updates their own tenant's settings. It takes `display_name`, `default_locale`, and\n`password_policy_override` as optional fields, fixes the target to `context.tenant_id`, and then runs the UpdateTenant\nlogic. A `password_policy_override` weaker than the standard values of the Authentication context's PasswordPolicy --\na smaller `min_length`, a larger `max_length`, or a smaller `history_depth` -- and a `max_age_days` outside the system\nceiling of 30 to 3650 days are both refused in the use case. `default_locale` accepts only a locale the catalogue ships\ntranslations for, and an empty string returns it to the system default.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UpdateAdminSettings")
@route("/api/admin/v1/settings")
@patch
op UpdateAdminSettings(
  @body body: IdMagic.Contract.TenantUpdateRequest
): UpdateAdminSettingsSuccess_200 | UpdateAdminSettingsError400 | UpdateAdminSettingsError403;

model GetTenantUserAttributeSchemaSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantUserAttributeSchemaResponse;
}

union GetTenantUserAttributeSchemaError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetTenantUserAttributeSchemaError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetTenantUserAttributeSchemaError403Body;
}

@doc("A tenant administrator reads their own tenant's custom attribute definitions.\nThe target is fixed to `context.tenant_id`, which rules out a cross-tenant read structurally.\nBesides the tenant's own custom definitions the response carries the built-in catalogue (`builtin`), so the UI can list\nevery attribute and avoid a key collision.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("GetTenantUserAttributeSchema")
@route("/api/admin/v1/tenant/user_attribute_schema")
@get
op GetTenantUserAttributeSchema(

): GetTenantUserAttributeSchemaSuccess_200 | GetTenantUserAttributeSchemaError403;

model UpdateTenantUserAttributeSchemaSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantUserAttributeSchemaResponse;
}

union UpdateTenantUserAttributeSchemaError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateTenantUserAttributeSchemaError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantUserAttributeSchemaError400Body;
}

union UpdateTenantUserAttributeSchemaError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantUserAttributeSchemaError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantUserAttributeSchemaError403Body;
}

union UpdateTenantUserAttributeSchemaError422Body {
  IdMagic.Contract.InvalidUserAttributeSchemaError,
}

model UpdateTenantUserAttributeSchemaError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantUserAttributeSchemaError422Body;
}

@doc("A tenant administrator replaces the custom attribute definitions outright. With the target fixed to `context.tenant_id`,\neach UserAttributeDef is validated and a key that collides with a built-in attribute, or a duplicate key, is refused. A\nkey a DynamicGroupRule still refers to can be neither removed nor retyped. Only the definitions are handled; no value is kept.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UpdateTenantUserAttributeSchema")
@route("/api/admin/v1/tenant/user_attribute_schema")
@put
op UpdateTenantUserAttributeSchema(
  @body body: IdMagic.Contract.TenantUserAttributeSchemaUpdateRequest
): UpdateTenantUserAttributeSchemaSuccess_200 | UpdateTenantUserAttributeSchemaError400 | UpdateTenantUserAttributeSchemaError403 | UpdateTenantUserAttributeSchemaError422;

model GetTenantGroupAttributeSchemaSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantGroupAttributeSchemaResponse;
}

union GetTenantGroupAttributeSchemaError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetTenantGroupAttributeSchemaError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetTenantGroupAttributeSchemaError403Body;
}

@doc("A tenant administrator reads their own tenant's Group custom attribute definitions.\nThe target is fixed to `context.tenant_id`, which rules out a cross-tenant read structurally.\nGroup has no built-in catalogue, so the response carries the tenant's own custom definitions alone.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("GetTenantGroupAttributeSchema")
@route("/api/admin/v1/tenant/group_attribute_schema")
@get
op GetTenantGroupAttributeSchema(

): GetTenantGroupAttributeSchemaSuccess_200 | GetTenantGroupAttributeSchemaError403;

model UpdateTenantGroupAttributeSchemaSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantGroupAttributeSchemaResponse;
}

union UpdateTenantGroupAttributeSchemaError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateTenantGroupAttributeSchemaError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantGroupAttributeSchemaError400Body;
}

union UpdateTenantGroupAttributeSchemaError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantGroupAttributeSchemaError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantGroupAttributeSchemaError403Body;
}

union UpdateTenantGroupAttributeSchemaError422Body {
  IdMagic.Contract.InvalidGroupAttributeSchemaError,
}

model UpdateTenantGroupAttributeSchemaError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantGroupAttributeSchemaError422Body;
}

@doc("A tenant administrator replaces the Group custom attribute definitions outright. With the target fixed to `context.tenant_id`,\neach GroupAttributeDef is validated and a duplicate key is refused. Only the definitions are handled; no value is kept.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UpdateTenantGroupAttributeSchema")
@route("/api/admin/v1/tenant/group_attribute_schema")
@put
op UpdateTenantGroupAttributeSchema(
  @body body: IdMagic.Contract.TenantGroupAttributeSchemaUpdateRequest
): UpdateTenantGroupAttributeSchemaSuccess_200 | UpdateTenantGroupAttributeSchemaError400 | UpdateTenantGroupAttributeSchemaError403 | UpdateTenantGroupAttributeSchemaError422;

model GetTenantBrandingSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantBrandingResponse;
}

@doc("Public interface returning the hosted UI branding of the resolved tenant. It requires no authentication, because the sign-in,\nconsent, and device screens read it before anyone has authenticated. Branding that is unset, invalid, or missing its asset each\nfall back to the system default, so the call always succeeds. It takes no input, because realm resolution (ResolveTenant) has\nalready settled the target tenant.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetTenantBranding")
@route("/api/branding")
@get
op GetTenantBranding(

): GetTenantBrandingSuccess_200;

model UpdateTenantBrandingSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantBrandingResponse;
}

union UpdateTenantBrandingError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateTenantBrandingError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantBrandingError400Body;
}

union UpdateTenantBrandingError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantBrandingError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantBrandingError403Body;
}

@doc("A tenant administrator updates their own tenant's branding: the product name, the colours, the footer links in a fixed order, and\nthe footer text. Image assets are handled by a separate interface. A colour is validated only as `#rrggbb`, and is never refused\nover its contrast ratio. Each footer link pairs a label with an https URL, and giving only one of the two is refused.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UpdateTenantBranding")
@route("/api/admin/v1/tenant/branding")
@put
op UpdateTenantBranding(
  @body body: IdMagic.Contract.TenantBrandingUpdateRequest
): UpdateTenantBrandingSuccess_200 | UpdateTenantBrandingError400 | UpdateTenantBrandingError403;

model UploadTenantBrandingAssetSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantBrandingUploadResponse;
}

union UploadTenantBrandingAssetError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UploadTenantBrandingAssetError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UploadTenantBrandingAssetError400Body;
}

union UploadTenantBrandingAssetError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UploadTenantBrandingAssetError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UploadTenantBrandingAssetError403Body;
}

@doc("An administrator uploads the branding logo or favicon as multipart. PNG, JPEG, WebP, and GIF are accepted, up to 256 KiB, and the format is checked against the magic bytes.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UploadTenantBrandingAsset")
@route("/api/admin/v1/tenant/branding/assets/{kind}")
@post
op UploadTenantBrandingAsset(
  @path kind: IdMagic.Contract.TenantBrandingAssetKind,
  @header contentType: "multipart/form-data",
  @multipartBody body: {
    @doc("The image to store, under the form field name `file`.")
    file: HttpPart<File>,
  }
): UploadTenantBrandingAssetSuccess_200 | UploadTenantBrandingAssetError400 | UploadTenantBrandingAssetError403;

model DeleteTenantBrandingAssetSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantBrandingUploadResponse;
}

union DeleteTenantBrandingAssetError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteTenantBrandingAssetError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteTenantBrandingAssetError400Body;
}

union DeleteTenantBrandingAssetError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteTenantBrandingAssetError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteTenantBrandingAssetError403Body;
}

@doc("An administrator deletes the stored branding logo or favicon. The URL of that asset is no longer returned afterwards.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("DeleteTenantBrandingAsset")
@route("/api/admin/v1/tenant/branding/assets/{kind}")
@delete
op DeleteTenantBrandingAsset(
  @path kind: IdMagic.Contract.TenantBrandingAssetKind
): DeleteTenantBrandingAssetSuccess_200 | DeleteTenantBrandingAssetError400 | DeleteTenantBrandingAssetError403;

model ListNotificationTemplatesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.NotificationTemplateListResponse;
}

union ListNotificationTemplatesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListNotificationTemplatesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListNotificationTemplatesError403Body;
}

@doc("A tenant administrator lists the notification templates. It returns every template_key of the catalogue against every supported locale, each with whether it is customized and the subject currently in force. The target is fixed to `context.tenant_id`, so a cross-tenant read cannot arise structurally.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("ListNotificationTemplates")
@route("/api/admin/v1/tenant/notification_templates")
@get
op ListNotificationTemplates(

): ListNotificationTemplatesSuccess_200 | ListNotificationTemplatesError403;

model GetNotificationTemplateSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.NotificationTemplateDetail;
}

union GetNotificationTemplateError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetNotificationTemplateError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetNotificationTemplateError400Body;
}

union GetNotificationTemplateError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetNotificationTemplateError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetNotificationTemplateError403Body;
}

@doc("A tenant administrator reads one template for editing. It returns the wording currently in force, the built-in default wording, and the permitted set of substitution variables together. A template_key or locale the catalogue does not know is refused with InvalidRequestError.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("GetNotificationTemplate")
@route("/api/admin/v1/tenant/notification_templates/{template_key}/{locale}")
@get
op GetNotificationTemplate(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string
): GetNotificationTemplateSuccess_200 | GetNotificationTemplateError400 | GetNotificationTemplateError403;

model UpdateNotificationTemplateSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.NotificationTemplateDetail;
}

union UpdateNotificationTemplateError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateNotificationTemplateError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateNotificationTemplateError400Body;
}

union UpdateNotificationTemplateError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateNotificationTemplateError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateNotificationTemplateError403Body;
}

@doc("A tenant administrator saves a template override. The subject, the text body, and the HTML body are replaced as one required set, so an override of only the text or only the HTML cannot be made. A substitution variable in the body or the subject that falls outside the permitted set of the template_key is refused at save time, before it could collapse into an empty string at run time (fail-closed). The sender email address is a server setting and cannot be overridden; only the display name can.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UpdateNotificationTemplate")
@route("/api/admin/v1/tenant/notification_templates/{template_key}/{locale}")
@put
op UpdateNotificationTemplate(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string,
  @body body: IdMagic.Contract.NotificationTemplateUpdateRequest
): UpdateNotificationTemplateSuccess_200 | UpdateNotificationTemplateError400 | UpdateNotificationTemplateError403;

model ResetNotificationTemplateSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.NotificationTemplateDetail;
}

union ResetNotificationTemplateError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ResetNotificationTemplateError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ResetNotificationTemplateError400Body;
}

union ResetNotificationTemplateError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ResetNotificationTemplateError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ResetNotificationTemplateError403Body;
}

@doc("A tenant administrator deletes a template override and returns to the built-in default. It succeeds for a template that has no override too, so it is idempotent. There is no version history, so what it returns to is always the built-in default and never the previous override.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("ResetNotificationTemplate")
@route("/api/admin/v1/tenant/notification_templates/{template_key}/{locale}")
@delete
op ResetNotificationTemplate(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string
): ResetNotificationTemplateSuccess_200 | ResetNotificationTemplateError400 | ResetNotificationTemplateError403;

model PreviewNotificationTemplateSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.NotificationTemplatePreviewResponse;
}

union PreviewNotificationTemplateError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model PreviewNotificationTemplateError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: PreviewNotificationTemplateError400Body;
}

union PreviewNotificationTemplateError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model PreviewNotificationTemplateError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: PreviewNotificationTemplateError403Body;
}

@doc("A tenant administrator renders unsaved wording against sample values. A field left out uses the wording currently in force. Substituted values on the HTML side are always escaped before expansion, and link URLs are built from the sample issuer URL. No mail is sent and no override is saved.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("PreviewNotificationTemplate")
@route("/api/admin/v1/tenant/notification_templates/{template_key}/{locale}/preview")
@post
op PreviewNotificationTemplate(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string,
  @body body: IdMagic.Contract.NotificationTemplatePreviewRequest
): PreviewNotificationTemplateSuccess_200 | PreviewNotificationTemplateError400 | PreviewNotificationTemplateError403;

model SendTestNotificationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.NotificationTemplateTestSendResponse;
}

union SendTestNotificationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model SendTestNotificationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: SendTestNotificationError400Body;
}

union SendTestNotificationError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model SendTestNotificationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SendTestNotificationError403Body;
}

@doc("A tenant administrator receives a test message using the wording currently in force. The recipient is fixed to the administrator's own verified email address and cannot be named in the request. Permitting an arbitrary recipient would make administrative privilege a relay for sending mail, so fixing it is deliberate. When the caller has no verified address, the request is refused with InvalidRequestError.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("SendTestNotification")
@route("/api/admin/v1/tenant/notification_templates/{template_key}/{locale}/test")
@post
op SendTestNotification(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string,): SendTestNotificationSuccess_200 | SendTestNotificationError400 | SendTestNotificationError403;

model GetTenantBrandingAssetSuccess_200 {
  @statusCode statusCode: 200;
  @doc("Fixed to the validated format of the stored asset, and served with `X-Content-Type-Options: nosniff`.")
  @header contentType: "image/png" | "image/jpeg" | "image/webp" | "image/gif";
  @doc("The asset bytes.")
  @body body: bytes;
}

union GetTenantBrandingAssetError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetTenantBrandingAssetError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetTenantBrandingAssetError400Body;
}

@doc("Public interface serving a stored branding asset, the logo or the favicon. The URL returned after an upload reaches this interface\nover the same origin, including the resolved realm prefix, so the administration console preview and the hosted UI can both fetch\nand render it as an image. The Content-Type is fixed to the validated format and `X-Content-Type-Options: nosniff` is set. An object\nof another tenant, and a deleted one, are both treated as nonexistent.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetTenantBrandingAsset")
@route("/tenant-branding-assets/{kind}/{id}")
@get
op GetTenantBrandingAsset(
  @path kind: IdMagic.Contract.TenantBrandingAssetKind,
  @path id: string
): GetTenantBrandingAssetSuccess_200 | GetTenantBrandingAssetError400;
}

}
