import "@typespec/http";

using TypeSpec.Http;

namespace IdMagic.Contract {

enum FeatureMaturity {
  experimental,
  preview,
  supported,
  deprecated,
}

enum FeatureUpdatePolicy {
  rolling,
  recreate_on_version_change,
  recreate_always,
}

@doc("One runtime-selectable feature enabled for this process. The registry, not configuration or persistence, owns these fields.")
model RuntimeFeatureMetadata {
  id: string;
  version: string;
  maturity: FeatureMaturity;
  update_policy: FeatureUpdatePolicy;
}

@doc("Versioned operational metadata for the runtime-selectable features enabled for this process.")
model FeatureRuntimeMetadata {
  schema_version: "1";
  enabled: RuntimeFeatureMetadata[];
}

@doc("RFC 9457 Problem Details, the default error body for generic APIs. `support_http.WriteProblem` emits it with `Content-Type: application/problem+json`. Contact points whose shape a standard fixes — OAuth 2.0 and OpenID Connect, Dynamic Client Registration (RFC 7591), SCIM (RFC 7644), and the SharedSignals receive endpoint (RFC 8935) — return that standard's error body instead; see the HTTP error responses section of the root specification. Every `*Error` model declared as `is ProblemDetails` shares this body and is told apart by its `type` URN suffix.")
model ProblemDetails {
  @doc("Problem type URI. IdMagic problem types are `urn:idmagic:error:<code>`, where `<code>` is the stable error code the server reports. The URI is not dereferenceable; clients switch on the URN suffix rather than fetching it.")
  type: string;
  @doc("Short human-readable summary derived from the error code: `invalid_role` becomes `Invalid role`. Stable for a given `type`.")
  title: string;
  @doc("HTTP status code of the response, repeated in the body (RFC 9457 section 3.1).")
  status: int32;
  @doc("Explanation specific to this occurrence. Always English and never localized; a UI that localizes maps the `type` URN suffix to its own message instead.")
  detail?: string;
  @doc("Request correlation id, the same value the response carries in the `X-Request-ID` header. Absent when the request had no correlation id.")
  instance?: string;
}


@doc("The 401 an operation answers when the guard in front of it finds no authenticated session. It is one shared response rather than one model per operation because the body never varies: 401 and 403 are the two branches of a single guard, and only the 403 differs from operation to operation (`AccessDeniedError`, `InsufficientScopeError`, `MfaEnrollmentNotAllowedError`). See the Declared status codes section of the root specification.")
model AuthenticationRequiredResponse {
  @statusCode statusCode: 401;
  @header contentType: "application/problem+json";
  @body body: AuthenticationRequiredError;
}


@doc("Reachability of one configured dependency, as the readiness probe reports it when `verbose` is set.")
model DependencyStatus {
  @doc("`healthy` when the dependency answered, `unavailable` when it did not.")
  status: string;
  @doc("Why the dependency is unavailable. Absent when it is healthy.")
  message?: string;
}


@doc("""
The wire form every domain event takes. `spec.MarshalDomainEvent` produces it, the audit store records it, the account
security notification dispatcher subscribes to it, and `AdminAuditEventResponse` publishes it to administrators. It is
not an HTTP request or response body of its own: no operation accepts or returns it, and it reaches a caller only
through the audit read model.
""")
model DomainEventEnvelope {
  @doc("The event type name, the value the emitting context's `EventType()` returns. It is the vocabulary `AuditEventQuery.type` filters on.")
  type: string;

  @doc("When the event occurred, as RFC 3339 with nanosecond precision in UTC. It is the emitting context's clock, not the time the record was appended.")
  occurredAt: utcDateTime;

  @doc("The event body. Only the published fields below are contract; the rest belongs to the emitting context.")
  payload: DomainEventPayload;
}


@doc("""
The published language of domain event payloads: the fields another context reads by name. The audit search attribute
extractor projects them into search axes and the account security notification dispatcher resolves the recipient and the
guard from them, so renaming one leaves the emitting context compiling while an audit filter silently returns nothing.
Every field is optional because no event carries all of them, and an event that omits a field simply does not appear
under that axis. A payload also carries fields private to the emitting context; those are not contract and are not
declared here. `mise run check-event-contract` holds this declaration and the reading points to the same set.
""")
model DomainEventPayload {
  @doc("The tenant the event belongs to. Its absence is what leaves an event out of a tenant-scoped audit search.")
  tenantId?: string;

  @doc("The subject the event is about. It names the actor when `actorUserId` is absent and the event is not one an agent performs.")
  userId?: string;

  @doc("The administrator who performed an administrative operation, as distinct from the user it acted on.")
  actorUserId?: string;

  @doc("The user an administrative operation acted on. `actor.id` and `target.id` answer different questions and are never collapsed.")
  targetUserId?: string;

  @doc("The agent. On the token exchange and approval events an agent performs, it names the actor; elsewhere the agent is the target.")
  agentId?: string;

  @doc("The OAuth 2.0 client.")
  clientId?: string;

  @doc("The login session.")
  sessionId?: string;

  @doc("The authorization transaction.")
  transactionId?: string;

  @doc("The correlation id shared across the records of one logical operation.")
  correlationId?: string;

  @doc("The request correlation id, the same value the response carries in `X-Request-ID`.")
  requestId?: string;

  @doc("The account name as it was submitted. Kept in plaintext and reserved for authentication failures, where an account that never existed still has to be traceable.")
  username?: string;

  @doc("The client IP, matched in plaintext, neither truncated nor hashed.")
  ip?: string;

  @doc("The country the client IP resolved to, rendered into the account security notification body.")
  countryCode?: string;

  @doc("The client user agent. The notification dispatcher derives the device label and the per-tenant salted device hash from it.")
  userAgent?: string;

  @doc("Why the transition happened. Session revocations notify the account holder only for `self_revoke` and `admin_revoke`.")
  reason?: string;

  @doc("The lifecycle workflow.")
  workflowId?: string;

  @doc("The lifecycle workflow run.")
  runId?: string;

  @doc("The zero-based position of the step within a lifecycle workflow run.")
  stepIndex?: int32;

  @doc("The principals appearing in the delegation chain, so an event is reachable from any participant, not only its ends.")
  actorChain?: string[];

  @doc("How many steps of delegation a token exchange carried.")
  delegationDepth?: int32;

  @doc("The same depth under the name an authorization decision records it. Both are projected onto one search axis.")
  actorChainDepth?: int32;

  @doc("The delegation mode, the value REQ-OAUTH2-049 derives, so introspection and the audit record name the same mode.")
  delegationMode?: string;

  ...Record<unknown>;
}


}
