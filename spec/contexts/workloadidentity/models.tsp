namespace IdMagic.Contract {

@doc("Published-language stub for the Agent principal the IdManagement context publishes, carrying only the attributes binding verification needs.")
model AgentRef {
  id: string;
  tenant_id: string;
  @doc("Active, Disabled, or Killed. Binding to anything but Active is refused fail-closed.")
  status: string;
}


@doc("Verification of an external attestation token failed fail-closed: an unregistered issuer, a bad signature, an expired token, an audience mismatch, a TTL over the ceiling, a pattern mismatch, an ambiguous match, or a bound Agent that is not active. A domain condition; over HTTP it reaches the client as the token exchange InvalidGrantError.")
model WorkloadAttestationRejectedError {

}


@doc("Neither `jwks_uri` nor `jwks` was given. RFC 9457 Problem Details with `type` = `urn:idmagic:error:workload_trust_bundle_jwks_required`.")
model WorkloadTrustBundleJwksRequiredError is ProblemDetails;


@doc("`name` is empty. RFC 9457 Problem Details with `type` = `urn:idmagic:error:workload_trust_bundle_name_required`.")
model WorkloadTrustBundleNameRequiredError is ProblemDetails;


@doc("`issuer` is empty. RFC 9457 Problem Details with `type` = `urn:idmagic:error:workload_trust_bundle_issuer_required`.")
model WorkloadTrustBundleIssuerRequiredError is ProblemDetails;


@doc("`accepted_audiences` is empty. RFC 9457 Problem Details with `type` = `urn:idmagic:error:workload_trust_bundle_audiences_required`.")
model WorkloadTrustBundleAudiencesRequiredError is ProblemDetails;


@doc("`max_subject_token_ttl_seconds` is not a positive value. RFC 9457 Problem Details with `type` = `urn:idmagic:error:workload_trust_bundle_invalid_ttl`.")
model WorkloadTrustBundleInvalidTtlError is ProblemDetails;


@doc("The `agent_id` given does not resolve to an Agent in the tenant. RFC 9457 Problem Details with `type` = `urn:idmagic:error:agent_workload_binding_agent_not_found`.")
model AgentWorkloadBindingAgentNotFoundError is ProblemDetails;


@doc("`subject_pattern` is empty. RFC 9457 Problem Details with `type` = `urn:idmagic:error:agent_workload_binding_pattern_required`.")
model AgentWorkloadBindingPatternRequiredError is ProblemDetails;


@doc("The states of WorkloadTrustBundleLifecycle.")
enum WorkloadTrustBundleStatus {
  enabled: "enabled",
  disabled: "disabled",
}


@doc("The states of AgentWorkloadBindingLifecycle.")
enum AgentWorkloadBindingStatus {
  enabled: "enabled",
  disabled: "disabled",
}


@doc("A tenant-scoped trust setting for an external attestation issuer. `issuer` is unique within the tenant, and at least one of `jwks_uri` and inline `jwks` is required.")
model WorkloadTrustBundle {
  @doc("The generated identifier (UUID v4). Immutable.")
  @minLength(1)
  @maxLength(64)
  id: string;
  @doc("The tenant it belongs to. Cross-tenant references are forbidden.")
  tenant_id: string = "default";
  @doc("The administrative display name, unique within the tenant.")
  @minLength(1)
  @maxLength(100)
  name: string;
  @doc("The SPIFFE trust domain, or a label the operator gives an issuer group.")
  @minLength(1)
  @maxLength(255)
  trust_domain: string;
  @doc("The issuer URL the `iss` claim of the external attestation token must match. Unique within the tenant.")
  issuer: url;
  @doc("Where the JWKS is fetched from. It must be https.")
  jwks_uri?: url;
  @doc("An inline JWKS, as an alternative to `jwks_uri`.")
  jwks?: JWKS;
  @doc("The accepted audiences; the `aud` claim of the external attestation token must match at least one of them.")
  accepted_audiences: string[];
  @doc("The ceiling on the external attestation token's own `exp - iat`. Exceeding it is refused fail-closed.")
  max_subject_token_ttl_seconds: int64 = 3600;
  status: WorkloadTrustBundleStatus = WorkloadTrustBundleStatus.enabled;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  @doc("When the JWKS was last fetched successfully. Used to judge how fresh the last-known-good cache is.")
  jwks_cached_at?: utcDateTime;
}


@doc("A mapping under a WorkloadTrustBundle that projects a glob pattern over external subjects onto an Agent of the same tenant. `agent_id` must name an existing Agent whose `tenant_id` matches the WorkloadTrustBundle's (WorkloadIdentityReferencesStayTenantLocal).")
model AgentWorkloadBinding {
  @doc("The generated identifier (UUID v4). Immutable.")
  @minLength(1)
  @maxLength(64)
  id: string;
  @doc("The tenant it belongs to. It matches `WorkloadTrustBundle.tenant_id`.")
  tenant_id: string = "default";
  @doc("The id of the WorkloadTrustBundle it belongs to.")
  trust_bundle_id: string;
  @doc("A glob pattern matched against the `sub` claim of the external attestation token, such as `spiffe://example.org/ns/*/sa/agent-*` or `system:serviceaccount:prod-agents:*`. Duplicates within one `trust_bundle_id` are forbidden.")
  @minLength(1)
  @maxLength(500)
  subject_pattern: string;
  @doc("The Agent to project onto. It must be an existing Agent whose `tenant_id` matches the WorkloadTrustBundle's.")
  agent_id: string;
  status: AgentWorkloadBindingStatus = AgentWorkloadBindingStatus.enabled;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  disabled_at?: utcDateTime;
}


@doc("The JSON request an administrator sends to register a WorkloadTrustBundle.")
model WorkloadTrustBundleRegisterRequest {
  @minLength(1)
  @maxLength(100)
  name: string;
  @minLength(1)
  @maxLength(255)
  trust_domain: string;
  issuer: url;
  jwks_uri?: url;
  jwks?: JWKS;
  accepted_audiences: string[];
  max_subject_token_ttl_seconds?: int64;
}


@doc("The JSON request an administrator sends to update the `name`, `jwks_uri`, `jwks`, `accepted_audiences`, or `max_subject_token_ttl_seconds` of a WorkloadTrustBundle. A field left out keeps its current value. `issuer` and `trust_domain` are immutable; changing them means registering again.")
model WorkloadTrustBundleUpdateRequest {
  @minLength(1)
  @maxLength(100)
  name?: string;
  jwks_uri?: url;
  jwks?: JWKS;
  accepted_audiences?: string[];
  max_subject_token_ttl_seconds?: int64;
}


@doc("How a WorkloadTrustBundle is represented. The list and the detail endpoints of the administration API return the same representation. It carries no secret from `jwks`, only public key material.")
model WorkloadTrustBundleResponse {
  id: string;
  tenant_id: string;
  name: string;
  trust_domain: string;
  issuer: url;
  jwks_uri?: url;
  @doc("Whether an inline `jwks` is set. The `jwks` itself is not returned.")
  has_inline_jwks: boolean;
  accepted_audiences: string[];
  max_subject_token_ttl_seconds: int64;
  status: WorkloadTrustBundleStatus;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  jwks_cached_at?: utcDateTime;
}


@doc("The WorkloadTrustBundle list of the administration API.")
model WorkloadTrustBundleListResponse {
  trust_bundles: WorkloadTrustBundleResponse[];
}


@doc("The response of RefreshWorkloadTrustBundleJWKS: whether the JWKS could actually be fetched. It is a test operation, so that a tenant finds a misconfiguration before it matters.")
model WorkloadTrustBundleRefreshResponse {
  reachable: boolean;
  key_count?: int64;
  jwks_cached_at?: utcDateTime;
}


@doc("The JSON request an administrator sends to create an AgentWorkloadBinding.")
model AgentWorkloadBindingCreateRequest {
  @minLength(1)
  @maxLength(500)
  subject_pattern: string;
  @minLength(1)
  agent_id: string;
}


@doc("How an AgentWorkloadBinding is represented. The list and the detail endpoints of the administration API return the same representation.")
model AgentWorkloadBindingResponse {
  id: string;
  tenant_id: string;
  trust_bundle_id: string;
  subject_pattern: string;
  agent_id: string;
  status: AgentWorkloadBindingStatus;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  disabled_at?: utcDateTime;
}


@doc("The AgentWorkloadBinding list of the administration API.")
model AgentWorkloadBindingListResponse {
  bindings: AgentWorkloadBindingResponse[];
}


@doc("The raw external attestation token and its verification context, handed over by the OAuth 2.0 token-exchange grant. The input of VerifyWorkloadAttestation.")
model WorkloadAttestation {
  @doc("The external attestation token as a raw string, a JWT-SVID for instance.")
  subject_token: string;
  @doc("The `resource` parameter of the token exchange (RFC 8707). It is kept in the audit event.")
  requested_audience: string;
}


@doc("The IdMagic principal an external attestation was projected onto once it passed verification. The OAuth 2.0 token-exchange use case treats it as the subject and resolves the bound OAuth2Client that `client_id` names through its own repository, to get the downscoping ceiling on scopes and grant types; WorkloadIdentity itself knows nothing about OAuth2Client.")
model WorkloadIdentityGrant {
  agent_id: string;
  @doc("The `client_id` of the AgentCredentialBinding the bound Agent holds. It becomes the `sub` of the issued token.")
  client_id: string;
  trust_bundle_id: string;
  binding_id: string;
}


@doc("An administrator registered a new WorkloadTrustBundle.")
model WorkloadTrustBundleConfigured {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
  issuer: string;
}


@doc("An administrator updated the metadata of a WorkloadTrustBundle.")
model WorkloadTrustBundleUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
}


@doc("An administrator disabled a WorkloadTrustBundle. No binding under it can be used for an exchange afterwards.")
model WorkloadTrustBundleDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
}


@doc("An administrator re-enabled a disabled WorkloadTrustBundle.")
model WorkloadTrustBundleEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
}


@doc("An administrator deleted a WorkloadTrustBundle. The AgentWorkloadBindings under it were deleted with it.")
model WorkloadTrustBundleDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
}


@doc("An administrator called RefreshWorkloadTrustBundleJWKS and tried to fetch the JWKS again.")
model WorkloadTrustBundleJWKSRefreshed {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
  reachable: boolean;
}


@doc("An administrator created an AgentWorkloadBinding.")
model AgentWorkloadBindingCreated {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
  bindingId: string;
  agentId: string;
}


@doc("An administrator disabled an AgentWorkloadBinding.")
model AgentWorkloadBindingDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  bindingId: string;
}


@doc("An administrator re-enabled a disabled AgentWorkloadBinding.")
model AgentWorkloadBindingEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  bindingId: string;
}


@doc("An administrator deleted an AgentWorkloadBinding.")
model AgentWorkloadBindingDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  bindingId: string;
}


@doc("An external attestation passed verification and an IdMagic token was issued as the credential of the bound Agent.")
model WorkloadTokenExchanged {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
  bindingId: string;
  agentId: string;
  audience: string;
}


@doc("Verification of an external attestation token was refused fail-closed.")
model WorkloadAttestationRejected {
  occurredAt: utcDateTime;
  tenantId: string;
  @doc("One of unregistered_issuer, trust_bundle_disabled, jwks_unavailable, invalid_signature, expired, audience_mismatch, ttl_exceeded, no_binding_match, ambiguous_match, agent_not_active, or agent_unbound.")
  reason: string;
  @doc("Set only when the issuer was registered.")
  trustBundleId?: string;
}

}
