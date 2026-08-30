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
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminAuditEventsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminAuditEventsError403Body;
}

@doc("An administrator reads the audit DomainEvents of their tenant, newest first, through bidirectional keyset pagination.\nBuilt for SOC and compliance work, it offers a time window (`after` / `before`), a page size (`limit`), continuation\n(`cursor`, reached through the `rel=\"prev\"` and `rel=\"next\"` links of the Link response header), and filtering by\n`type`, `user_id`, and `username` (the actor). Authentication events -- login success and failure, MFA, sessions,\naggregates -- live in the same store, so `category` (authentication / success / fail / aggregated / user / group /\nclient / consent / token / tenant / key) narrows the search for a security or an operational investigation. Beyond\nthat, `q` matches raw text as a substring and `filter` takes a structured expression over the field/operator allow\nlist of the registry. Only fields the AuditEventSearchAttribute registry permits may appear in `filter`; arbitrary\nSQL or JSONPath is refused. `user_id` and the `actor.id` of `filter` name the actor, and `username` resolves the\nactor's login name to a user id at search time -- not hashed, and matching nothing when no such user exists.\n`actor.username` in `filter` is a plaintext match reserved for authentication failures, so that an account name\nthat never existed can still be traced. `target.id` in `filter` is a different concept: the user an administrative\noperation acted on (AuditActorVsTarget). A system_admin reaching the endpoint through the default tenant sees\nevery tenant; an admin stays inside their own. The query takes no tenant_id input, so an admin is confined to\ntheir tenant structurally, and only a system_admin can cross that line, through the dedicated route.")
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
  IdMagic.Contract.AccessDeniedError,
}

model ExportAdminAuditEventsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ExportAdminAuditEventsError403Body;
}

@doc("An administrator exports audit events, authentication events included, over a time window and a filter.\nStreaming to a SIEM belongs to a separate work item; this interface returns one download.\nThe filter is the same AuditEventQuery ListAdminAuditEvents takes (q / filter / category / type / time window),\nand only fields on the registry allow list may appear in `filter`.")
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

@doc("An administrator reads one audit event by id. An event of another tenant is treated as nonexistent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("GetAdminAuditEvent")
@route("/api/admin/v1/audit_events/{id}")
@get
op GetAdminAuditEvent(
  @path id: string
): GetAdminAuditEventSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminAuditEventError403 | GetAdminAuditEventError404;
}

}
