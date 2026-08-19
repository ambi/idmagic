namespace IdMagic.Contract {

@doc("The request is missing a required parameter or carries an invalid value, or the target belongs to another tenant and is treated as nonexistent. Published-language stub for the shared error the OAuth2 context defines. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_request`. The OAuth 2.0 endpoints report the same condition as OAuthInvalidRequestError instead.")
model InvalidRequestError is ProblemDetails;


@doc("The principal does not satisfy the policy for the operation. Published-language stub for the shared error the Authentication context defines. RFC 9457 Problem Details with `type` = `urn:idmagic:error:access_denied`. The OAuth 2.0 endpoints report the same condition as OAuthAccessDeniedError instead.")
model AccessDeniedError is ProblemDetails;


@doc("The tenant has reached its resource creation ceiling (Quota). Published-language stub for the shared error the Tenancy context defines. RFC 9457 Problem Details with `type` = `urn:idmagic:error:quota_exceeded`, written by the shared error handler on every path, the Dynamic Client Registration endpoint included.")
model QuotaExceededError is ProblemDetails;


@doc("The minimal stub of the Agent principal the IdManagement context publishes.")
model Agent {
  id: string;
  tenant_id: string;
  owner_user_id: string;
  status: string;
}


@doc("Verification of a received SET failed fail-closed: an unregistered issuer, a bad signature, an expired token, an audience mismatch, a duplicate `jti`, or a subject that could not be resolved. The receive endpoint is a contact point whose shape RFC 8935 fixes, so it does not return Problem Details; the body is `application/json`. IdMagic currently emits `error` and `message` rather than the `err` and `description` of RFC 8935 section 2.3, and this model records what the server sends today.")
model SecurityEventRejectedError {
  @doc("Always `security_event_rejected`.")
  error: string;
  @doc("Human-readable explanation of the rejection. Always English and never localized.")
  message: string;
}


@doc("`event_types` is empty. RFC 9457 Problem Details with `type` = `urn:idmagic:error:ssf_stream_event_types_required`.")
model SsfStreamEventTypesRequiredError is ProblemDetails;


@doc("`event_types` contains a CAEP event type the server does not know. RFC 9457 Problem Details with `type` = `urn:idmagic:error:ssf_stream_event_type_invalid`.")
model SsfStreamEventTypeInvalidError is ProblemDetails;


@doc("`delivery_endpoint` is absent or is not an https URL. RFC 9457 Problem Details with `type` = `urn:idmagic:error:ssf_transmitter_delivery_endpoint_invalid`.")
model SsfTransmitterDeliveryEndpointInvalidError is ProblemDetails;


@doc("`audience` is absent. RFC 9457 Problem Details with `type` = `urn:idmagic:error:ssf_transmitter_audience_required`.")
model SsfTransmitterAudienceRequiredError is ProblemDetails;


@doc("`trusted_issuer` is absent or is not an https URL. RFC 9457 Problem Details with `type` = `urn:idmagic:error:ssf_receiver_trusted_issuer_invalid`.")
model SsfReceiverTrustedIssuerInvalidError is ProblemDetails;


@doc("Neither `jwks_uri` nor `jwks` was given. RFC 9457 Problem Details with `type` = `urn:idmagic:error:ssf_receiver_jwks_required`.")
model SsfReceiverJwksRequiredError is ProblemDetails;


@doc("`accepted_audiences` is empty. RFC 9457 Problem Details with `type` = `urn:idmagic:error:ssf_receiver_accepted_audiences_required`.")
model SsfReceiverAcceptedAudiencesRequiredError is ProblemDetails;


@doc("Which way the stream runs, seen from IdMagic. `Transmit` means IdMagic pushes CAEP events to an external receiver; `Receive` means IdMagic accepts events from an external transmitter.")
enum SsfStreamDirection {
  Transmit: "Transmit",
  Receive: "Receive",
}


@doc("Whether the stream is running.")
enum SsfStreamStatus {
  enabled: "enabled",
  disabled: "disabled",
}


@doc("The CAEP event types that are implemented.")
enum CaepEventType {
  session_revoked: "session-revoked",
  token_claims_change: "token-claims-change",
  credential_change: "credential-change",
  assurance_level_change: "assurance-level-change",
}


@doc("What moved the revocation epoch forward.")
enum RevocationReason {
  AgentKilled: "AgentKilled",
  AgentDisabled: "AgentDisabled",
  AgentCredentialUnbound: "AgentCredentialUnbound",
  OwnerDisabled: "OwnerDisabled",
  OwnerDeleted: "OwnerDeleted",
  ManualAdmin: "ManualAdmin",
  InboundSecurityEvent: "InboundSecurityEvent",
}


@doc("The delivery state of an outbound SET.")
enum SecurityEventDeliveryStatus {
  pending: "pending",
  delivered: "delivered",
  failed: "failed",
  dead_letter: "dead_letter",
}


@doc("The verification outcome of an inbound SET.")
enum SecurityEventVerificationResult {
  accepted: "accepted",
  rejected_signature: "rejected_signature",
  rejected_unknown_issuer: "rejected_unknown_issuer",
  rejected_replay: "rejected_replay",
  rejected_audience: "rejected_audience",
  rejected_subject_unresolved: "rejected_subject_unresolved",
}


@doc("What a CAEP event is about. It maps one-to-one onto the subject representation of the access tokens IdMagic itself issues (tenant_id plus principal_id), so there is no separate subject mapping table of the kind WorkloadIdentity keeps.")
model SsfSubject {
  @doc("Either `Agent` or `User`.")
  subject_type: string;
  tenant_id: string;
  principal_id: string;
}


@doc("What a CAEP event means, before it is signed and wrapped into a SET.")
model CaepEvent {
  event_type: CaepEventType;
  subject: SsfSubject;
  reason?: RevocationReason;
  event_timestamp: utcDateTime;
  @doc("One of `policy`, `admin`, or `user`.")
  initiating_entity: string;
}


@doc("A Security Event Token (RFC 8417): the signed compact JWT together with its decoded standard claims.")
model SecurityEventToken {
  jti: string;
  iss: url;
  aud: string;
  iat: utcDateTime;
  event: CaepEvent;
  @doc("The compact serialization of the signed JWT.")
  compact: string;
}


@doc("The JSON request an administrator sends to register an SsfStream with `direction=Transmit`.")
model SsfTransmitterStreamRegisterRequest {
  event_types: CaepEventType[];
  delivery_endpoint: url;
  audience: string;
  delivery_authorization?: string;
  max_delivery_attempts?: int64;
}


@doc("The JSON request an administrator sends to register an SsfStream with `direction=Receive`.")
model SsfReceiverStreamRegisterRequest {
  event_types: CaepEventType[];
  trusted_issuer: url;
  jwks_uri?: url;
  jwks?: JWKS;
  accepted_audiences: string[];
}


@doc("The JSON request an administrator sends to update the `event_types` of an SsfStream. A field left out keeps its current value.")
model SsfStreamUpdateRequest {
  event_types?: CaepEventType[];
}


model RevocationEpochResponse {
  @doc("The epoch already advanced to. Absent for an Agent that has never been revoked.")
  epoch?: utcDateTime;
}


@doc("The one-to-one companion record holding the revocation epoch of an Agent. An access token or session whose `issued_at` precedes the epoch is treated as invalid (fail-closed). The epoch advances on KillAgent, DisableAgent, UnbindAgentCredential, a Disable or Delete of the owner (`owner_user_id`), or an accepted inbound SecurityEvent. An Agent with no row has never been revoked.")
model AgentRevocationEpoch {
  agent_id: string;
  tenant_id: string;
  epoch: utcDateTime;
  reason: RevocationReason;
  advanced_at: utcDateTime;
  @doc("A reference for tracing what advanced the epoch: the domain event, or the `jti` of the inbound SET.")
  source_event_id?: string;
}


@doc("The metadata every SSF stream a tenant registers shares. Depending on the direction, exactly one SsfTransmitterConfig or SsfReceiverConfig accompanies it.")
model SsfStream {
  id: string;
  tenant_id: string;
  direction: SsfStreamDirection;
  @doc("The subset of event types this stream sends or accepts.")
  event_types: CaepEventType[];
  status: SsfStreamStatus = SsfStreamStatus.enabled;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
}


@doc("The delivery settings accompanying an SsfStream with `direction=Transmit`, one per stream.")
model SsfTransmitterConfig {
  stream_id: string;
  @doc("The endpoint of the external receiver that accepts the push. It must be https.")
  delivery_endpoint: url;
  @doc("The `aud` claim of the SETs issued on this stream.")
  audience: string;
  @doc("An optional bearer secret presented in the Authorization header when pushing to the receiver.")
  delivery_authorization?: string;
  max_delivery_attempts: int64 = 8;
}


@doc("The trust settings for the external transmitter, accompanying an SsfStream with `direction=Receive`, one per stream. It reuses the same registered-issuer verification WorkloadTrustBundle uses. At least one of `jwks_uri` and inline `jwks` is required.")
model SsfReceiverConfig {
  stream_id: string;
  @doc("The issuer URL the `iss` claim of the external transmitter's SET must match.")
  trusted_issuer: url;
  jwks_uri?: url;
  jwks?: JWKS;
  accepted_audiences: string[];
}


@doc("The delivery state of an outbound SET. A projector creates it from RevocationEpochAdvanced, and delivery and retry are at-least-once.")
model SecurityEventDelivery {
  id: string;
  tenant_id: string;
  stream_id: string;
  set_jti: string;
  set: SecurityEventToken;
  status: SecurityEventDeliveryStatus = SecurityEventDeliveryStatus.pending;
  attempt_count: int64 = 0;
  next_attempt_at?: utcDateTime;
  last_error?: string;
  created_at: utcDateTime;
  delivered_at?: utcDateTime;
}


@doc("The record of an accepted inbound SET. `set_jti` is unique within the stream, so a replay is detected and refused.")
model ReceivedSecurityEvent {
  id: string;
  tenant_id: string;
  stream_id: string;
  set_jti: string;
  event_type: CaepEventType;
  subject: SsfSubject;
  verification_result: SecurityEventVerificationResult;
  received_at: utcDateTime;
  @doc("When the SET passed verification and was applied to the local revocation epoch.")
  reflected_at?: utcDateTime;
}


@doc("The revocation epoch of an Agent moved forward, settling a local revocation. The transmitter projector takes this event as its starting point and creates the CaepEvent and SecurityEventDelivery.")
model RevocationEpochAdvanced {
  occurredAt: utcDateTime;
  tenantId: string;
  agentId: string;
  reason: RevocationReason;
  epoch: utcDateTime;
}


@doc("A summary event for audit and governance, recording that an Agent's access was revoked at once because of a kill switch or an owner offboarding.")
model AgentAccessRevoked {
  occurredAt: utcDateTime;
  tenantId: string;
  agentId: string;
  reason: RevocationReason;
}


@doc("An administrator registered a new SsfStream.")
model SsfStreamRegistered {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
  direction: SsfStreamDirection;
}


@doc("An administrator updated the metadata of an SsfStream.")
model SsfStreamUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
}


@doc("An administrator disabled an SsfStream. It neither delivers nor accepts anything afterwards.")
model SsfStreamDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
}


@doc("An administrator re-enabled a disabled SsfStream.")
model SsfStreamEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
}


@doc("An administrator deleted an SsfStream. The TransmitterConfig or ReceiverConfig attached to it was deleted with it.")
model SsfStreamDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
}


@doc("A SecurityEventDelivery was delivered successfully.")
model SecurityEventTransmitted {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
  setJti: string;
}


@doc("A SecurityEventDelivery failed and a retry was scheduled.")
model SecurityEventDeliveryFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
  setJti: string;
  attemptCount: int64;
}


@doc("A failed SecurityEventDelivery was returned to pending for another attempt.")
model SecurityEventDeliveryRetried {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
  setJti: string;
  attemptCount: int64;
}


@doc("A SecurityEventDelivery used up `max_delivery_attempts` and moved to dead_letter.")
model SecurityEventDeliveryDeadLettered {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
  setJti: string;
}


@doc("An inbound SET was verified and applied as a local revocation.")
model SecurityEventReceived {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
  setJti: string;
  eventType: CaepEventType;
}


@doc("Verification of an inbound SET failed fail-closed and nothing was applied.")
model SecurityEventRejected {
  occurredAt: utcDateTime;
  tenantId: string;
  @doc("When the rejection happened before the issuer was resolved, the stream may not be identifiable.")
  streamId?: string;
  verificationResult: SecurityEventVerificationResult;
}


@doc("The Security Event Token in the request body is larger than the receive endpoint accepts. The receive endpoint is a contact point whose shape RFC 8935 fixes, so the body is `application/json` and carries `error` and `message` rather than Problem Details.")
model SecurityEventTokenTooLargeError {
  @doc("Always `security_event_token_too_large`.")
  error: string;
  @doc("Human-readable explanation. Always English and never localized.")
  message: string;
}


@doc("The `stream_id` in the path does not name an SsfStream of this tenant. Returned by the receive endpoint, so the body is the `application/json` shape of that contact point, not Problem Details; the administration API reports the same condition as Problem Details with `type` = `urn:idmagic:error:ssf_stream_not_found`.")
model SecurityEventStreamNotFoundError {
  @doc("Always `ssf_stream_not_found`.")
  error: string;
  @doc("Human-readable explanation. Always English and never localized.")
  message: string;
}

}
