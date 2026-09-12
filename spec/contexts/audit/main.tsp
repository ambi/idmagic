import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Audit")
namespace IdMagic.Audit {

namespace Operations {
model ListAdminAuditEventsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminAuditEventListResponse;
}

union ListAdminAuditEventsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListAdminAuditEventsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListAdminAuditEventsError400Body;
}

union ListAdminAuditEventsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminAuditEventsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminAuditEventsError403Body;
}

@doc("An administrator reads the audit DomainEvents of their tenant, newest first, through bidirectional keyset pagination.\nBuilt for SOC and compliance work, it offers a time window (`after` / `before`), a page size (`limit`), continuation\n(`cursor`, reached through the `rel=\"prev\"` and `rel=\"next\"` links of the Link response header), and filtering by\n`type`, `user_id`, and `username` (the actor). Authentication events -- login success and failure, MFA, sessions,\naggregates -- live in the same store, so `category` (authentication / success / fail / aggregated / user / group /\nclient / consent / token / tenant / key) narrows the search for a security or an operational investigation. Beyond\nthat, `q` matches raw text as a substring and `filter` takes a structured expression over the field/operator allow\nlist of the registry. Only fields the AuditEventSearchAttribute registry permits may appear in `filter`; arbitrary\nSQL or JSONPath is refused. `user_id` and the `actor.id` of `filter` name the actor, and `username` resolves the\nactor's login name to a user id at search time -- not hashed, and matching nothing when no such user exists.\n`actor.username` in `filter` is a plaintext match reserved for authentication failures, so that an account name\nthat never existed can still be traced. `target.id` in `filter` is a different concept: the user an administrative\noperation acted on (AuditActorVsTarget). The scope is the caller's own tenant, fixed by the route: the query\ntakes no tenant_id input and no input that widens it, so the confinement is structural, and control-plane\ncredentials do not lift it here either -- ListSystemAuditEvents is the only search that spans tenants.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("ListAdminAuditEvents")
@route("/api/admin/v1/audit_events")
@get
op ListAdminAuditEvents(
  @query query: IdMagic.Contract.AuditEventQuery
): ListAdminAuditEventsSuccess_200 | ListAdminAuditEventsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ListAdminAuditEventsError403;

model ExportAdminAuditEventsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminAuditEventListResponse;
}

union ExportAdminAuditEventsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ExportAdminAuditEventsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ExportAdminAuditEventsError400Body;
}

union ExportAdminAuditEventsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ExportAdminAuditEventsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ExportAdminAuditEventsError403Body;
}

@doc("An administrator exports their own tenant's audit events, authentication events included, over a time window and a\nfilter. Streaming to a SIEM belongs to a separate work item; this interface returns one download.\nThe filter is the same AuditEventQuery ListAdminAuditEvents takes (q / filter / category / type / time window),\nand only fields on the registry allow list may appear in `filter`. The scope matches ListAdminAuditEvents\nexactly: the caller's own tenant, fixed by the route, control-plane credentials included.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("ExportAdminAuditEvents")
@route("/api/admin/v1/audit_events/export")
@get
op ExportAdminAuditEvents(
  @query query: IdMagic.Contract.AuditEventQuery
): ExportAdminAuditEventsSuccess_200 | ExportAdminAuditEventsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ExportAdminAuditEventsError403;

model GetAdminAuditEventSearchOptionsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AuditEventSearchOptionsResponse;
}

union GetAdminAuditEventSearchOptionsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminAuditEventSearchOptionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminAuditEventSearchOptionsError403Body;
}

@doc("An administrator reads the choices that turn the `event.type` and `outcome` fields of the audit search form into\npick lists. The values come from the same single source of truth in Go as AuditSearchRegistry and\nauditEventCategoryTypes, so a hard-coded allow list in the UI cannot drift from what the server accepts.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("GetAdminAuditEventSearchOptions")
@route("/api/admin/v1/audit_events/search_options")
@get
op GetAdminAuditEventSearchOptions(

): GetAdminAuditEventSearchOptionsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminAuditEventSearchOptionsError403;

model GetAdminAuditEventSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminAuditEventResponse;
}

union GetAdminAuditEventError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminAuditEventError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminAuditEventError403Body;
}

union GetAdminAuditEventError404Body {
  IdMagic.Contract.AuditEventNotFoundError,
}

model GetAdminAuditEventError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAdminAuditEventError404Body;
}

@doc("An administrator reads one audit event of their own tenant by id. An event of another tenant is treated as\nnonexistent, so an id alone reveals nothing about a tenant the caller cannot see, and control-plane credentials\ndo not change that -- GetSystemAuditEvent is the only read that reaches another tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("GetAdminAuditEvent")
@route("/api/admin/v1/audit_events/{id}")
@get
op GetAdminAuditEvent(
  @path id: string
): GetAdminAuditEventSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminAuditEventError403 | GetAdminAuditEventError404;

model ListSystemAuditEventsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminAuditEventListResponse;
}

union ListSystemAuditEventsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListSystemAuditEventsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListSystemAuditEventsError400Body;
}

union ListSystemAuditEventsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListSystemAuditEventsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSystemAuditEventsError403Body;
}

@doc("A control-plane operator searches every tenant's audit DomainEvents, newest first, through the same bidirectional\nkeyset pagination, time window, page size, and filter grammar ListAdminAuditEvents accepts. The scope is every\ntenant, fixed by the route rather than by an input, so a cross-tenant search is never reached by adding a\nparameter to the tenant-facing route. The caller must be an active user who belongs to the control-plane tenant,\ncarries `system_admin` among its effective roles, and calls through the control-plane tenant; such a caller needs\nno `admin` role of its own, and anyone who fails any of the three is refused with AccessDeniedError before any\nevent is read. A cursor issued by ListAdminAuditEvents is rejected rather than continued here, because the scope\nis part of what the cursor is bound to. The search form's pick lists come from\nGetAdminAuditEventSearchOptions, which carries vocabulary rather than any tenant's records (REQ-AUDIT-007).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("ListSystemAuditEvents")
@route("/api/admin/v1/system/audit_events")
@get
op ListSystemAuditEvents(
  @query query: IdMagic.Contract.AuditEventQuery
): ListSystemAuditEventsSuccess_200 | ListSystemAuditEventsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ListSystemAuditEventsError403;

model ExportSystemAuditEventsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminAuditEventListResponse;
}

union ExportSystemAuditEventsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ExportSystemAuditEventsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ExportSystemAuditEventsError400Body;
}

union ExportSystemAuditEventsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ExportSystemAuditEventsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ExportSystemAuditEventsError403Body;
}

@doc("A control-plane operator exports every tenant's audit events over the same time window and filter\nListSystemAuditEvents takes, so that what was found in the search is what the download contains. Streaming to a\nSIEM belongs to a separate work item; this interface returns one download. The caller must satisfy the same\nthree control-plane conditions ListSystemAuditEvents states (REQ-AUDIT-007).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("ExportSystemAuditEvents")
@route("/api/admin/v1/system/audit_events/export")
@get
op ExportSystemAuditEvents(
  @query query: IdMagic.Contract.AuditEventQuery
): ExportSystemAuditEventsSuccess_200 | ExportSystemAuditEventsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ExportSystemAuditEventsError403;

model GetSystemAuditEventSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminAuditEventResponse;
}

union GetSystemAuditEventError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetSystemAuditEventError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetSystemAuditEventError403Body;
}

union GetSystemAuditEventError404Body {
  IdMagic.Contract.AuditEventNotFoundError,
}

model GetSystemAuditEventError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetSystemAuditEventError404Body;
}

@doc("A control-plane operator reads one audit event of any tenant by id. The caller must satisfy the same three\ncontrol-plane conditions ListSystemAuditEvents states; anyone who fails one is refused with AccessDeniedError\nrather than told whether the event exists (REQ-AUDIT-007).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("GetSystemAuditEvent")
@route("/api/admin/v1/system/audit_events/{id}")
@get
op GetSystemAuditEvent(
  @path id: string
): GetSystemAuditEventSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetSystemAuditEventError403 | GetSystemAuditEventError404;
}

}
