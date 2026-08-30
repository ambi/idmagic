namespace IdMagic.Contract {

@doc("Published-language stub for the client kind owned by the OAuth2 context. Wire values follow RFC 6749 section 2.1.")
enum ClientType {
  Public: "public",
  Confidential: "confidential",
}


@doc("Published-language stub for the client secret credential state the OAuth2 context publishes.")
enum ClientSecretCredentialStatus {
  Active: "Active",
  Expired: "Expired",
  Revoked: "Revoked",
}


@doc("Published-language stub for the grant types owned by the OAuth2 context. Wire values are the `grant_type` values registered by RFC 6749 section 4, RFC 8628 section 3.4 (device code), RFC 8693 section 2.1 (token exchange), and OpenID Connect CIBA Core section 4.")
enum GrantType {
  AuthorizationCode: "authorization_code",
  RefreshToken: "refresh_token",
  ClientCredentials: "client_credentials",
  DeviceCode: "urn:ietf:params:oauth:grant-type:device_code",
  TokenExchange: "urn:ietf:params:oauth:grant-type:token-exchange",
  Ciba: "urn:openid:params:grant-type:ciba",
}


@doc("Published-language stub for the token endpoint client authentication methods owned by the OAuth2 context. Wire values are the `token_endpoint_auth_method` values of OpenID Connect Discovery 1.0 section 3 and RFC 8705 section 2.")
enum TokenEndpointAuthMethod {
  ClientSecretBasic: "client_secret_basic",
  ClientSecretPost: "client_secret_post",
  PrivateKeyJwt: "private_key_jwt",
  TlsClientAuth: "tls_client_auth",
  None: "none",
}


@doc("Published-language stub for the authorization response type owned by the OAuth2 context. Wire value follows RFC 6749 section 3.1.1; only the authorization code flow is supported.")
enum ResponseType {
  Code: "code",
}


@doc("Published-language stub for the FAPI security profile owned by the OAuth2 context. Wire values are IdMagic-defined (backend/oauth2/client/domain/client.go).")
enum FapiProfile {
  None: "none",
  Fapi2SecurityProfile: "fapi_2_security_profile",
}


@doc("An OAuth2Client already holds the maximum of two Active client secret credentials, so another one cannot be issued. Published-language stub for the limit error the OAuth2 context defines. RFC 9457 Problem Details with `type` = `urn:idmagic:error:client_secret_limit_exceeded`.")
model ClientSecretLimitExceededError is ProblemDetails;


@doc("Published-language stub for the event the OAuth2 context publishes when a client secret is additionally issued. It carries neither the secret nor its hash.")
model ClientSecretIssued {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  clientId: string;
  credentialId: string;
  expiresAt: utcDateTime;
}


@doc("Published-language stub for the event the OAuth2 context publishes when one client secret is revoked. It carries neither the secret nor its hash.")
model ClientSecretRevoked {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  clientId: string;
  credentialId: string;
}


@doc("The kind of an Application. `federated` is a business application tied together by SSO federation, `weblink` is an external link with no SSO, and `service` is a machine-to-machine client_credentials API client -- Okta's API Services, with neither a portal tile nor user assignments.")
enum ApplicationKind {
  federated: "federated",
  weblink: "weblink",
  service: "service",
}


@doc("Whether an Application is enabled.")
enum ApplicationStatus {
  active: "active",
  disabled: "disabled",
}


@doc("The single protocol an Application uses. It cannot be changed after creation.")
enum ApplicationProtocolType {
  oidc: "oidc",
  saml: "saml",
  wsfed: "wsfed",
}


@doc("The application kind chosen at bulk creation. `oidc`, `saml`, and `wsfed` create a federated Application together with the matching protocol settings and publish both to the catalogue atomically; `weblink` creates an external link with no SSO; and `service` creates a machine-to-machine client_credentials OAuth2 client of the `service` kind.")
enum ApplicationProvisionType {
  oidc: "oidc",
  saml: "saml",
  wsfed: "wsfed",
  weblink: "weblink",
  service: "service",
}


@doc("A reference to the single protocol configuration an Application is tied to. It carries only the key that matches `type`, and the protocol row is related back to the Application through its `application_id` as an ordinary foreign key.")
model ApplicationProtocol {
  @doc("The protocol kind.")
  type: ApplicationProtocolType;
  @doc("The OAuth2 `client_id` that `oidc` points at.")
  client_id?: string;
  @doc("The SAML `entity_id` that `saml` points at.")
  @maxLength(2048)
  entity_id?: string;
  @doc("The `wtrealm` of the WS-Fed RP that `wsfed` points at.")
  @maxLength(2048)
  wtrealm?: string;
}


@doc("A portal section an administrator defines per tenant. An Application carries zero or more of them, and\nthe user portal groups its tiles into sections by category. It never crosses the assignment boundary: a section holds\nonly the apps assigned to that user.")
model ApplicationCategory {
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  @doc("The category identifier.")
  id: string;
  @doc("The section display name.")
  @minLength(1)
  name: string;
  @doc("The display order of the section; smaller comes first, and ties break by name ascending.")
  position: int64;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The authentication strength the sign-in policy demands. `Password` asks for nothing more (acr includes urn:idmagic:acr:pwd), and `Mfa` requires a second factor (acr equals urn:idmagic:acr:mfa, and amr holds one of otp, webauthn, hwk, or swk).")
enum RequiredAuthnStrength {
  Password: "Password",
  Mfa: "Mfa",
}


@doc("The authentication strength the sign-in policy demands. It maps the constrained choices, Password and Mfa, onto the internal acr and amr. When Mfa is not satisfied, the user is nudged to step up.")
model RequiredAuthnLevel {
  strength: RequiredAuthnStrength = RequiredAuthnStrength.Password;
}


@doc("Governs the pre-enrollment period before MFA becomes mandatory, and the administrator-approved onboarding after enforcement starts. Before `enforcement_start_at` an ordinary sign-in is allowed, which nudges the user to enroll through self-service; after it, an unenrolled user does not complete sign-in, and only when `allow_admin_bypass` is true and a valid single-use enrollment bypass exists do they proceed into the enrollment-only flow. `grace_period_seconds` is how long a bypass may be used after enforcement starts, not a period of free enrollment.")
model MfaEnrollmentPolicy {
  enforcement_start_at?: utcDateTime;
  grace_period_seconds?: int64;
  allow_admin_bypass: boolean = false;
}


@doc("The static access conditions of a sign-in policy. Every one of them is a structured condition the initial implementation can actually evaluate. `network_allow_cidrs` matches the client IP against the permitted CIDRs, and `reauth_max_age_seconds` is evaluated as the recency of an authentication or a step-up. The old free-text network and device fields have been dropped.")
model AccessCondition {
  @doc("The CIDRs the client IP is permitted to fall in. Empty means no restriction. When it is non-empty and the client IP falls in none of them, or the IP cannot be determined at all, the request is refused fail-closed.")
  network_allow_cidrs: string[];
  reauth_max_age_seconds?: int64;
}


@doc("One rule of a sign-in policy.")
model SignInRule {
  rule_id: string;
  name: string;
  enabled: boolean;
  required_authn: RequiredAuthnLevel;
  condition: AccessCondition;
  mfa_enrollment?: MfaEnrollmentPolicy;
  @doc("When `required_authn` is `Mfa`, whether a remembered trusted device (amr holding `tdev`) may satisfy the MFA\nrequirement. False means MFA every time, admitting nothing but a genuine second factor.\nIt defaults to true, but as long as the tenant leaves `trusted_device_max_age_seconds` at zero, no trusted device is\never issued or evaluated, so the effective default is not to remember anything.")
  allow_trusted_device: boolean = true;
}


@doc("The sign-in policy of one Application, owned by the ApplicationCatalog. When it holds any rule it replaces the tenant default outright for that application; when it holds none, the default applies. This is an override model. A setting weaker than the default can still be saved, and the UI warns about it.")
model AppSignInPolicy {
  @doc("The global UUID foreign key to the parent Application. The tenant boundary follows from that parent.")
  application_id: string;
  rules: SignInRule[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The tenant default sign-in policy. The ApplicationCatalog owns one per tenant, and it is the baseline applied when federation starts for every Application that has no policy of its own. Setting a policy on an application overrides the default for that application.")
model TenantDefaultSignInPolicy {
  tenant_id: string = "default";
  rules: SignInRule[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("How the administration API represents an Application.")
model AdminApplicationResponse {
  id: string;
  name: string;
  kind: ApplicationKind;
  status: ApplicationStatus;
  icon_url?: string;
  icon_object_key?: string;
  launch_url?: string;
  protocol?: ApplicationProtocol;
  category_ids: string[];
  category_names: string[];
  protocol_summary?: string;
  assigned_subject_count: int64;
  sign_in_policy_summary: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The Application list response.")
model AdminApplicationListResponse {
  applications: AdminApplicationResponse[];
}


@doc("The request to create or update an ApplicationCategory. Leaving `position` out places it last.")
model AdminApplicationCategoryRequest {
  @minLength(1)
  name: string;
  position?: int64;
}


@doc("How the administration API represents an ApplicationCategory.")
model AdminApplicationCategoryResponse {
  id: string;
  name: string;
  position: int64;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The ApplicationCategory list response, by position ascending with ties breaking by name ascending.")
model AdminApplicationCategoryListResponse {
  categories: AdminApplicationCategoryResponse[];
}


@doc("The request replacing the set of categories assigned to an Application. `category_ids` may hold only categories that already exist in the tenant.")
model ApplicationCategoriesRequest {
  category_ids: string[];
}


@doc("The request that creates an Application in one call. It names the kind and carries the matching protocol settings with it. The backend prepares the protocol row outside the catalogue first, then inserts the Application and sets `row.application_id` in a single transaction. A protocol row whose second half failed is a valid configuration outside the catalogue, so no meaningless Application is left behind.")
model AdminApplicationCreateRequest {
  @minLength(1)
  name: string;
  @doc("One of oidc, saml, wsfed, weblink, or service.")
  type: ApplicationProvisionType;
  @doc("The launch URL a portal tile opens. Required for `weblink`.")
  launch_url?: string;
  @doc("Required when `type` is `oidc`: the redirect URIs of the OAuth2 client to create.")
  redirect_uris?: string[];
  @doc("The scopes of the OAuth2 client to create.")
  scope?: string;
  @doc("The kind of client to create when `type` is `oidc`; `confidential` by default.")
  client_type?: ClientType;
  @doc("The client authentication method of the client created when `type` is `oidc`. It is settled at creation and immutable afterwards.")
  token_endpoint_auth_method?: TokenEndpointAuthMethod;
  @doc("The public key JWKS URI for `token_endpoint_auth_method=private_key_jwt`.")
  jwks_uri?: url;
  @doc("The certificate Subject DN for `token_endpoint_auth_method=tls_client_auth`.")
  @maxLength(2048)
  tls_client_auth_subject_dn?: string;
  @doc("Required when `type` is `wsfed`: the `wtrealm` of the WS-Fed RP to create.")
  @maxLength(2048)
  wtrealm?: string;
  @doc("Required when `type` is `wsfed`: the permitted `wreply` targets.")
  reply_urls?: string[];
  @doc("The NameID format for `type=wsfed`.")
  name_id_format?: string;
  @doc("The NameID source attribute for `type=wsfed`.")
  name_id_source?: string;
  @doc("Required when `type` is `saml`: the entity ID of the SAML SP to create.")
  @maxLength(2048)
  entity_id?: string;
  @doc("Required when `type` is `saml`: the Assertion Consumer Service URL.")
  acs_urls?: string[];
  @doc("For `type=saml`: the Single Logout Service URL of the service provider.")
  @maxLength(2048)
  slo_url?: string;
  @doc("For `type=saml`: whether the whole SAML response is signed as well as the assertion.")
  sign_response?: boolean;
  @doc("For `type=saml`: whether the service provider's AuthnRequests must be signed. When true, `authn_request_signing_certificate_pem` supplies the key that verifies them.")
  want_authn_requests_signed?: boolean;
  @doc("For `type=saml`: the PEM certificate that verifies a signed AuthnRequest.")
  authn_request_signing_certificate_pem?: string;
  @doc("For `type=saml` and `type=wsfed`: the identity provider profile whose issuer and signing key the assertions are issued under.")
  idp_profile_id?: string;
}


@doc("The result of a bulk creation. For `type=oidc` it returns the `client_id` and `client_secret` of the OAuth2 client it created, once; the secret cannot be read again.")
model AdminApplicationCreateResponse {
  application: AdminApplicationResponse;
  @doc("The generated client id, returned when `type` is `oidc`.")
  client_id?: string;
  @doc("The secret, returned once when `type` is `oidc`.")
  client_secret?: string;
}


@doc("The real settings of the OAuth2 client the OIDC protocol of an Application points at, for the application detail screen. Advanced fields are included. `token_endpoint_auth_method`, `client_type`, and `fapi_profile` are immutable under the update contract and are shown for reference only.")
model ApplicationOidcConfig {
  client_id: string;
  client_type: ClientType;
  redirect_uris: string[];
  grant_types: GrantType[];
  response_types: ResponseType[];
  token_endpoint_auth_method: TokenEndpointAuthMethod;
  scope: string;
  require_pushed_authorization_requests: boolean;
  dpop_bound_access_tokens: boolean;
  fapi_profile: FapiProfile;
  @doc("True only for a confidential client using client_secret_basic or client_secret_post. False for private_key_jwt, mTLS, and a public client.")
  client_secret_rotatable: boolean;
  @doc("Credential metadata carrying neither a hash nor a plaintext. The editor shows the creation time, the expiry, and whether the credential is Active, Expired, or Revoked.")
  secret_credentials: ApplicationClientSecretCredentialMetadata[];
  @doc("The source attribute used for the `sub` claim of the ID token and UserInfo. Left unset, the\nuser's internal id is used, which is the default behaviour. It overrides the subject identifier the same way\n`name_id_source` does for WS-Fed and SAML.")
  sub_source_attribute?: string;
  @doc("The claim release override for this application. Independently of what the scope-driven\nstandard claims (profile, email, and so on) would disclose, it narrows or remaps what is released.\nAn attribute whose visibility is `private`, and a reserved claim type, are refused by the floor.")
  rules: ClaimMappingRule[];
}


@doc("Non-sensitive metadata projecting an OAuth2 credential for the Application editor. It exposes neither the OAuth2 hash nor the plaintext, and derives the state from `revoked_at`, `expires_at`, and the time of evaluation.")
model ApplicationClientSecretCredentialMetadata {
  credential_id: string;
  created_at: utcDateTime;
  expires_at?: utcDateTime;
  revoked_at?: utcDateTime;
  status: ClientSecretCredentialStatus;
}


@doc("The real settings of the relying party the WS-Fed protocol of an Application points at, for the application detail screen. The per-RP advanced fields -- audience, token type, claim rules -- are included. Entra domain federation is handled separately, as a per-tenant setting.")
model ApplicationWsFedConfig {
  wtrealm: string;
  reply_urls: string[];
  audience: string;
  token_type: WsFedTokenType;
  name_id_format: string;
  name_id_source: string;
  rules: ClaimMappingRule[];
}


@doc("The real settings of the service provider the SAML protocol of an Application points at, for the application detail screen.")
model ApplicationSamlConfig {
  @doc("The id of the IdP profile in the same tenant that every SAML SP is assigned to.")
  idp_profile_id: string;
  entity_id: string;
  acs_urls: string[];
  slo_url?: string;
  audience?: string;
  name_id_format: string;
  name_id_source: string;
  sign_assertion: boolean;
  sign_response: boolean;
  want_authn_requests_signed: boolean;
  authn_request_signing_certificate_pem?: string;
  rules: ClaimMappingRule[];
}


@doc("The application detail response. Besides the Application itself it carries the real settings resolved through the protocol relation. At most one of `oidc`, `saml`, and `wsfed` is set, the one matching `Application.protocol.type`.")
model AdminApplicationDetailResponse {
  application: AdminApplicationResponse;
  oidc?: ApplicationOidcConfig;
  wsfed?: ApplicationWsFedConfig;
  saml?: ApplicationSamlConfig;
  @doc("The same view the dedicated sign-in policy endpoint returns, so the detail screen can render the policy without a second request.")
  sign_in_policy: AppSignInPolicyResponse;
}


@doc("The request updating the OIDC protocol settings. Only the fields given are changed. Advanced fields are included, while\nfields immutable under the update contract, `token_endpoint_auth_method` among them, are not.")
model ApplicationOidcConfigUpdateRequest {
  redirect_uris?: string[];
  grant_types?: GrantType[];
  response_types?: ResponseType[];
  scope?: string;
  require_pushed_authorization_requests?: boolean;
  dpop_bound_access_tokens?: boolean;
  @doc("Left out, the existing setting is kept. Sending an empty string returns it to the default, the user's internal id.")
  sub_source_attribute?: string;
  @doc("Left out, the existing claim release override is kept. Sending an empty array returns it to no override, meaning the standard claims alone.")
  rules?: ClaimMappingRule[];
}


@doc("The request to rotate the secret of an Application's OIDC protocol. `grace_days` is 7 when left out, and only 0 -- revoke at once -- or 1 to 30 days is accepted.")
model RotateApplicationClientSecretRequest {
  grace_days?: int64;
}


@doc("Returns the new secret once. It carries no hash, no existing secret, and no way to read the plaintext again.")
model RotateApplicationClientSecretResponse {
  client_secret: string;
  grace_until?: utcDateTime;
  credentials: ApplicationClientSecretCredentialMetadata[];
}


@doc("The request that issues an additional secret with an expiry on an Application's OIDC protocol, leaving existing credentials alone. `expires_in_days` is 90 when left out, and only 1 to 730 days is accepted.")
model IssueApplicationClientSecretRequest {
  expires_in_days?: int64;
}


@doc("Returns the additionally issued secret once. It carries no hash, no existing secret, and no way to read the plaintext again.")
model IssueApplicationClientSecretResponse {
  client_secret: string;
  credential: ApplicationClientSecretCredentialMetadata;
  credentials: ApplicationClientSecretCredentialMetadata[];
}


@doc("The list of non-sensitive metadata returned after one named credential has been revoked.")
model RevokeApplicationClientSecretResponse {
  credentials: ApplicationClientSecretCredentialMetadata[];
}


@doc("The request updating the WS-Federation protocol settings. Only the fields given are changed. The per-RP advanced fields -- audience, token type, claim rules -- are included. Entra domain federation is out of scope here.")
model ApplicationWsFedConfigUpdateRequest {
  reply_urls?: string[];
  audience?: string;
  token_type?: WsFedTokenType;
  name_id_format?: string;
  name_id_source?: string;
  rules?: ClaimMappingRule[];
}


@doc("The request partially updating the SAML protocol settings. It updates the ACS and SLO URLs, the audience, the NameID, the signature policy, and the claim rules.")
model ApplicationSamlConfigUpdateRequest {
  @doc("The IdP profile in the same tenant to assign to the SP. When a dedicated profile is already in use by another SP, the update is refused.")
  idp_profile_id?: string;
  acs_urls?: string[];
  slo_url?: string;
  audience?: string;
  name_id_format?: string;
  name_id_source?: string;
  sign_assertion?: boolean;
  sign_response?: boolean;
  want_authn_requests_signed?: boolean;
  authn_request_signing_certificate_pem?: string;
  rules?: ClaimMappingRule[];
}


@doc("The request updating an Application. Only the fields given are changed.")
model AdminApplicationUpdateRequest {
  name?: string;
  status?: ApplicationStatus;
  launch_url?: string;
}


@doc("How an Application is represented after its icon has been uploaded or deleted.")
model ApplicationIconUploadResponse {
  application: AdminApplicationResponse;
}


@doc("The request assigning an Application. Leaving `visibility` out means `visible`.")
model ApplicationAssignmentRequest {
  subject_type: AssignmentSubjectType;
  @minLength(1)
  subject_id: string;
  visibility?: AssignmentVisibility;
}


@doc("How an assignment is represented.")
model ApplicationAssignmentResponse {
  subject_type: AssignmentSubjectType;
  subject_id: string;
  visibility: AssignmentVisibility;
  created_at: utcDateTime;
}


@doc("The assignment list response.")
model ApplicationAssignmentListResponse {
  assignments: ApplicationAssignmentResponse[];
}


@doc("The request in which an administrator replaces the sign-in policy of an Application. Sending any rule overrides the tenant default; sending none returns it to applying the default.")
model AppSignInPolicyRequest {
  rules: SignInRule[];
}


@doc("How the sign-in policy of an Application is represented. It lets the application detail screen tell apart the application's\nown override (`policy`), the tenant default (`tenant_default`), and the effective rule sequence that actually applies.")
model AppSignInPolicyResponse {
  policy: AppSignInPolicy;
  tenant_default: TenantDefaultSignInPolicy;
  @doc("The rule sequence actually evaluated under the override model: `policy.rules` when the application holds any rule, and `tenant_default.rules` when it does not.")
  effective_rules: SignInRule[];
  @doc("True when the application's own policy is weaker than the default, that is, when it loosens the authentication strength, the reauthentication window, or the permitted networks. It is for a UI warning and does not stop evaluation.")
  weaker_than_default: boolean;
}


@doc("The request in which an administrator replaces the tenant default sign-in policy.")
model TenantDefaultSignInPolicyRequest {
  rules: SignInRule[];
}


@doc("How the tenant default sign-in policy is represented.")
model TenantDefaultSignInPolicyResponse {
  policy: TenantDefaultSignInPolicy;
  @doc("How many active users of the tenant hold no TOTP or WebAuthn factor at all. It is used to gauge the impact before making MFA mandatory.")
  unenrolled_user_count: int64;
}


@doc("One assigned application shown on the user portal.")
model MyApplication {
  id: string;
  name: string;
  kind: ApplicationKind;
  icon_url?: string;
  launch_url?: string;
  @doc("The ids of the categories it carries. The portal sorts its tiles into sections by these.")
  category_ids: string[];
}


@doc("One section heading of the user portal, representing the tenant's category definitions by position ascending.")
model PortalApplicationCategory {
  id: string;
  name: string;
}


@doc("The list of assigned applications for the user portal. A manual order takes precedence when one is saved, and name ascending is used otherwise.\n`categories` carries the tenant's category definitions by position ascending, and the portal uses them as section headings.")
model MyApplicationListResponse {
  applications: MyApplication[];
  categories: PortalApplicationCategory[];
}


@doc("The manual order on the user portal. Keyed by `user_id`, it holds the display order of Applications as a\nsequence of `application_id` values. `user_id` names an existing, globally unique User.\nAn application in the sequence that is no longer assigned drops out of the list, and an assigned app absent from the\nsequence is appended by name ascending. `application_ids` is kept as an array rather than a database foreign key, so\nthe use case validation of ReorderMyApplications is what confines it to Applications assigned to that user and visible.")
model ApplicationOrdering {
  user_id: string;
  application_ids: string[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("A user saves their own manual portal order. `application_ids` may hold only apps assigned to them and visible.")
model ReorderMyApplicationsRequest {
  application_ids: string[];
}


@doc("The manual portal order a user has saved. With nothing saved, `application_ids` is empty.")
model MyApplicationOrderResponse {
  application_ids: string[];
}


@doc("An Application was created.")
model ApplicationCreated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
}


@doc("The metadata of an Application was updated.")
model ApplicationUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
  changedFields: string[];
}


@doc("The stored icon of an Application was uploaded or deleted.")
model ApplicationIconUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
  action: string;
}


@doc("An Application was deleted.")
model ApplicationDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
}


@doc("A user or a group was assigned to an Application.")
model ApplicationAssigned {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
  subjectType: string;
  subjectId: string;
}


@doc("An assignment was removed from an Application.")
model ApplicationUnassigned {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
  subjectType: string;
  subjectId: string;
}


@doc("An ApplicationCategory was created.")
model ApplicationCategoryCreated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  categoryId: string;
}


@doc("An ApplicationCategory was updated.")
model ApplicationCategoryUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  categoryId: string;
}


@doc("An ApplicationCategory was deleted. It was removed from the Applications it was assigned to as well.")
model ApplicationCategoryDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  categoryId: string;
}


@doc("The sign-in policy of an Application was updated.")
model AppSignInPolicyUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
}


@doc("The claim release override of an Application was updated: the rules, and the source attribute of `sub` or of the NameID.")
model ApplicationClaimMappingUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
  protocol: string;
}


@doc("The tenant default sign-in policy was updated, clearing it included.")
model TenantDefaultSignInPolicyUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
}


@doc("Federation was refused by an Application sign-in policy.")
model AppAccessDeniedByPolicy {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  protocol: string;
  subject: string;
  reason: string;
}


@doc("A step-up became necessary to satisfy the authentication strength of an Application sign-in policy.")
model AppStepUpRequired {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  protocol: string;
  subject: string;
}


@doc("The `name` of an ApplicationCategory is empty. RFC 9457 Problem Details with `type` = `urn:idmagic:error:category_name_required`.")
model CategoryNameRequiredError is ProblemDetails;


@doc("A `category_id` assigned to an Application does not exist among the ApplicationCategory records of its tenant. RFC 9457 Problem Details with `type` = `urn:idmagic:error:unknown_category`.")
model UnknownCategoryError is ProblemDetails;

}
