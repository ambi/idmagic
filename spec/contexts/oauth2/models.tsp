namespace IdMagic.Contract {

@doc("One connected application the end-user account portal returns (self-service).\nIt is fixed to the caller's own `sub`, so neither `sub` nor `tenant_id` appears. It returns the client_id, the display\nname, the granted scopes, and when they were granted and expire, for display.")
model AccountConsentResponse {
  client_id: string;
  @doc("The display name the client_id resolved to. It resolves in the order `OAuth2Client.client_name`, then the\nApplication catalogue name, and falls back to the client_id itself when there is neither.")
  client_name: string;
  scopes: string[];
  state: ConsentState;
  granted_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("The threshold of an EndpointRateLimitPolicy was exceeded. This is the general endpoint rate limit, counted on every arriving request, and it is separate from the login throttle, which counts failures per account and per IP. Fail-closed: the request is rejected even when the shared counter store is unreachable. Written by `support_http.WriteRateLimited` with `Content-Type: application/json`, not as Problem Details, because the body carries `retry_after_seconds`. The response also carries a `Retry-After` header with the same number of seconds.")
model RateLimitedError {
  @doc("Always `rate_limited`.")
  error: string;
  @doc("How many seconds to wait before retrying. Repeated in the `Retry-After` response header.")
  retry_after_seconds: int32;
  @doc("English fallback text for callers that do not localize by code. A UI that localizes maps `rate_limited` to its own message instead.")
  message: string;
}


@doc("The next screen, or the client redirect URI, to show after an SPA authentication step.")
model BrowserFlowResponse {
  next?: string;
  redirect_to?: url;
}


@doc("How the authorization endpoint delivers the code to the redirect_uri. Wire values follow OAuth 2.0 Multiple Response Type Encoding Practices and OAuth 2.0 Form Post Response Mode.")
enum ResponseMode {
  Query: "query",
  FormPost: "form_post",
}


@doc("Token kind carried by `token_type_hint` and by revocation audit events. Wire values follow RFC 7009 section 2.1.")
enum TokenType {
  AccessToken: "access_token",
  RefreshToken: "refresh_token",
}


@doc("How a token is bound to its holder (RFC 7800). `dpop` is the thumbprint of the DPoP public key, `mtls` the thumbprint of the mTLS client certificate.")
enum SenderConstraintKind {
  Dpop: "dpop",
  Mtls: "mtls",
}


@doc("The sender constraint applied to an issued access token. `none` means a bearer token.")
enum AccessTokenSenderConstraint {
  None: "none",
  Dpop: "dpop",
  Mtls: "mtls",
}


@doc("The hashing method of the PKCE `code_challenge` (RFC 7636). This application permits S256 alone; `plain` is refused as weak.")
enum CodeChallengeMethod {
  S256: "S256",
}


@doc("The states the AuthorizationCodeFlow state machine may take. See the state machine section as well.")
enum AuthorizationCodeState {
  Received: "received",
  AuthenticationPending: "authentication_pending",
  Authenticated: "authenticated",
  ConsentPending: "consent_pending",
  Consented: "consented",
  CodeIssued: "code_issued",
  Exchanged: "exchanged",
  Rejected: "rejected",
  Expired: "expired",
}


@doc("The states the DeviceCodeFlow state machine (RFC 8628) may take. See the state machine section as well.")
enum DeviceCodeState {
  Issued: "issued",
  UserCodeEntered: "user_code_entered",
  Approved: "approved",
  Denied: "denied",
  Exchanged: "exchanged",
  Expired: "expired",
}


@doc("The states the ApprovalRequestLifecycle state machine may take. It is held as the general form of \"a precondition is not yet satisfied\" rather than as CIBA transport vocabulary. `consumed` is a terminal state reachable only from `approved`, standing for the single exchange into tokens. See the state machine section as well.")
enum ApprovalRequestState {
  Pending: "pending",
  Approved: "approved",
  Denied: "denied",
  Expired: "expired",
  Consumed: "consumed",
}


@doc("The states the RefreshTokenLifecycle state machine may take. `rotated` has been consumed into a child token, `revoked` has been revoked, and `expired` has run out.")
enum RefreshTokenState {
  Active: "Active",
  Rotated: "Rotated",
  Revoked: "Revoked",
  Expired: "Expired",
}


@doc("The states the AuthorizationCodeRecordLifecycle state machine may take. It describes the lifecycle of the code itself, along a different axis from AuthorizationCodeFlow.")
enum AuthorizationCodeRecordState {
  Issued: "issued",
  Redeemed: "redeemed",
  Expired: "expired",
}


@doc("The states the ConsentLifecycle state machine may take. The right to withdraw under GDPR Article 7(3) makes `granted` to `revoked` possible.")
enum ConsentState {
  Granted: "granted",
  Revoked: "revoked",
  Expired: "expired",
}


@doc("The states the PARRecordLifecycle state machine may take. From `stored` it becomes `used` when `/authorize` reads it, and `expired` when it runs out.")
enum PARRecordState {
  Stored: "Stored",
  Used: "Used",
  Expired: "Expired",
}


@doc("The downscoping partial order each field carries in a registered authorization_details schema. `set` is set inclusion (the request is a subset of the consent or the source), `at_most` is monotone decrease (the request is at most the consent or the source), `enum` is an exact match against a permitted list, and `exact` is an exact match on an opaque value. Anything undecidable falls to the refusing side, fail-closed.")
enum AuthorizationDetailFieldSemantics {
  set: "set",
  at_most: "at_most",
  enum_: "enum",
  exact: "exact",
}


@doc("The operational states of an AuthorizationDetailType. Only `enabled` is accepted; `disabled` refuses new requests, and tokens already issued stay valid until they expire.")
enum AuthorizationDetailTypeState {
  Enabled: "Enabled",
  Disabled: "Disabled",
}


@doc("OAuth 2.0 / OpenID Connect protocol error codes, carried in the `error` member of an OAuthError body. Wire values are registered by RFC 6749 sections 4.1.2.1 and 5.2, RFC 6750 section 3.1 (invalid_token, insufficient_scope), RFC 8628 section 3.5 (authorization_pending, slow_down, expired_token), RFC 8707 section 2 (invalid_target), RFC 9449 section 7.1 (invalid_dpop_proof), RFC 7591 section 3.2.2 (invalid_client_metadata, invalid_redirect_uri), RFC 9126 section 2.2 (invalid_request_uri), RFC 9396 section 5 (invalid_authorization_details), OpenID Connect Core section 3.1.2.6 (login_required), and OpenID Connect CIBA Core section 13 (unknown_user_id, invalid_binding_message).")
enum OAuthErrorCode {
  InvalidRequest: "invalid_request",
  InvalidClient: "invalid_client",
  InvalidGrant: "invalid_grant",
  UnauthorizedClient: "unauthorized_client",
  UnsupportedGrantType: "unsupported_grant_type",
  InvalidScope: "invalid_scope",
  InvalidToken: "invalid_token",
  ServerError: "server_error",
  InvalidDpopProof: "invalid_dpop_proof",
  AccessDenied: "access_denied",
  ExpiredToken: "expired_token",
  SlowDown: "slow_down",
  InsufficientScope: "insufficient_scope",
  AuthorizationPending: "authorization_pending",
  InvalidTarget: "invalid_target",
  UnsupportedResponseType: "unsupported_response_type",
  LoginRequired: "login_required",
  InvalidRequestUri: "invalid_request_uri",
  InvalidClientMetadata: "invalid_client_metadata",
  InvalidRedirectUri: "invalid_redirect_uri",
  InvalidAuthorizationDetails: "invalid_authorization_details",
  UnknownUserId: "unknown_user_id",
  InvalidBindingMessage: "invalid_binding_message",
}


@doc("A client under RFC 7591 with the FAPI 2.0 extensions. A client outside the catalogue, whose `application_id` is null, is allowed; when it is not null it names an `oidc` Application of the same tenant, and at most one client is bound to one Application.")
model OAuth2Client {
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  @minLength(1)
  @maxLength(128)
  client_id: string;
  @doc("The Application it corresponds to. Ordinary CRUD leaves it alone; only the bulk creation and deletion of an Application manage the relationship.")
  application_id?: string;
  client_secret_hash?: string;
  @maxLength(200)
  client_name?: string;
  client_type: ClientType;
  @doc("Required for a redirect-based grant (authorization_code). A machine-to-machine client using client_credentials alone has none (RFC 6749 section 3.1.2).")
  redirect_uris?: url[];
  grant_types: GrantType[];
  response_types: ResponseType[];
  token_endpoint_auth_method: TokenEndpointAuthMethod;
  scope: string;
  jwks_uri?: url;
  jwks?: JWKS;
  tls_client_auth_subject_dn?: string;
  id_token_signed_response_alg: SignatureAlgorithm = SignatureAlgorithm.PS256;
  require_pushed_authorization_requests: boolean = false;
  require_pkce?: boolean;
  dpop_bound_access_tokens: boolean = false;
  fapi_profile: FapiProfile = FapiProfile.None;
  first_party: boolean = false;
  @doc("Where OpenID Connect Back-Channel Logout 1.0 notifications go. When it is set, every session revocation POSTs a\nsigned logout token to this URI.")
  backchannel_logout_uri?: url;
  @doc("Whether the RP requires a `sid` claim in the logout token. When true, nothing is delivered for a target session without a `sid`, because an issuance with no browser session has no target to notify.")
  backchannel_logout_session_required: boolean = false;
  @doc("Where OpenID Connect Front-Channel Logout 1.0 notifications go. The `/end_session` response carries an iframe\npointing at this URI.")
  frontchannel_logout_uri?: url;
  @doc("Whether to add the `iss` and `sid` query parameters to the iframe URI.")
  frontchannel_logout_session_required: boolean = false;
  @doc("The claim issuance policy for this client, the per-application release override.\nThe source attribute that plays the part of a NameID is used as the source of `sub`. Whether a ClaimExposed attribute\nmay be disclosed at all, decided by the scope and the consent through the standard scopes openid, profile, email, and\nso on, is applied first and independently of this policy; this policy narrows and remaps within that range, and merges\ninto the ID token and UserInfo.")
  claim_policy: ClaimMappingPolicy;
  created_at: utcDateTime;
  @doc("It equals `created_at` at creation and is updated on every administrative operation on the client metadata afterwards.")
  updated_at: utcDateTime;
}


@doc("The authentication credential of a secret-based OAuth2 client. The plaintext is returned only in the issuance response and appears in no storage, event, or listing. Issuing a new one leaves existing credentials alone, and they keep matching until they are revoked individually or expire. An existing credential with no expiry is tolerated so the migration needs no downtime.")
model ClientSecretCredential {
  id: string;
  client_id: string;
  secret_hash: string;
  created_at: utcDateTime;
  @doc("When the credential expires. It is required for a newly added credential; null stands for a legacy credential from before the migration, which has no expiry, or for the current credential of a compatibility rotation.")
  expires_at?: utcDateTime;
  @doc("When it was revoked outright. A credential with a `revoked_at` is not used for authentication whatever its `expires_at`.")
  revoked_at?: utcDateTime;
}


@doc("The GDPR consent record. Its state follows the ConsentLifecycle state machine. `user_id` and `client_id` name an existing User and OAuth2Client.")
model Consent {
  user_id: string;
  client_id: string;
  scopes: string[];
  state: ConsentState;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  granted_at: utcDateTime;
  @doc("Normally `granted_at` plus 365 days.")
  expires_at: utcDateTime;
  @doc("When the state became `revoked`, for audit.")
  revoked_at?: utcDateTime;
  @doc("RFC 9396: the structured details the user consented to. It is the yardstick for the subset check at issuance and exchange.")
  authorization_details?: AuthorizationDetail[];
}


@doc("One element of RFC 9396 `authorization_details`: a structured, fine-grained permission identified by its `type`. It is handled consistently through the request, the consent, and its reflection into the token, and is validated fail-closed against the registered AuthorizationDetailType.")
model AuthorizationDetail {
  @doc("The identifier of the registered AuthorizationDetailType, `payment_initiation` for instance.")
  @minLength(1)
  type: string;
  @doc("RFC 9396: the URIs of the resource servers the detail is aimed at.")
  locations?: string[];
  @doc("RFC 9396: the actions permitted against the target.")
  actions?: string[];
  @doc("RFC 9396: the kinds of data to be accessed.")
  datatypes?: string[];
  @doc("RFC 9396: the identifier of the target resource.")
  identifier?: string;
  @doc("RFC 9396: the privilege levels against the target.")
  privileges?: string[];
  @doc("The structured fields specific to the type, `instructedAmount` and `creditorAccount` for instance. The registered schema binds their meaning.")
  fields?: Record<unknown>;
}


@doc("One field rule of an AuthorizationDetailsSchema, defining its downscoping partial order and its permitted values.")
model AuthorizationDetailFieldRule {
  @doc("The field it governs: a standard field such as `actions`, or a key inside the `fields` map.")
  @minLength(1)
  name: string;
  @doc("The partial order used for the subset check.")
  semantics: AuthorizationDetailFieldSemantics;
  @doc("When true, the detail must carry this field.")
  required: boolean = false;
  @doc("The closed set of permitted values when `semantics` is `set` or `enum`; empty means unrestricted.")
  allowed?: string[];
}


@doc("The structural schema of one type. It lists the fields accepted and the downscoping partial order of each. An unregistered field, a mismatch against the schema, and a value outside the permitted set are all refused, fail-closed.")
model AuthorizationDetailsSchema {
  @doc("The field rules.")
  rules: AuthorizationDetailFieldRule[];
}


@doc("An `authorization_details` type definition a tenant registers. It bundles the type identifier, the validation schema, and the display template for the consent UI. Only an `enabled` type is accepted. Management is guarded by AdminAuthorizationDetailTypesManage.")
model AuthorizationDetailType {
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  @doc("The `authorization_details` type identifier.")
  @minLength(1)
  type: string;
  description?: string;
  schema: AuthorizationDetailsSchema;
  @doc("The template presenting a detail to a human in the consent UI, `up to {fields.instructedAmount} to {identifier}` for instance.")
  display_template: string;
  state: AuthorizationDetailTypeState = AuthorizationDetailTypeState.Enabled;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The operational states of an McpResourceServer. Only `active` passes the resource indicator check; `disabled` refuses new issuance, and tokens already issued stay valid until they expire.")
enum McpResourceServerState {
  Active: "Active",
  Disabled: "Disabled",
}


@doc("The tenant-scoped registration of an MCP resource server, a tool or a data source. It owns the canonical resource URI and the permitted scopes, and is the yardstick for Protected Resource Metadata (RFC 9728) and for the resource indicator check (RFC 8707). Management is guarded by AdminMcpResourcesManage.")
model McpResourceServer {
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  @doc("The unique identifier of the registration (a UUID), globally unique across tenants.")
  @minLength(1)
  id: string;
  @doc("The canonical resource URI strictly bound to the `aud` of the tokens issued. Unique within the tenant.")
  resource: ResourceIndicator;
  @doc("The display name in the administration UI.")
  @minLength(1)
  @maxLength(200)
  name: string;
  @doc("The allow list of scopes permitted when accessing this resource. The scopes requested alongside a `resource` have to be a subset of it.")
  scopes: string[];
  state: McpResourceServerState = McpResourceServerState.Active;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The `resource` parameter of RFC 8707: an absolute URI naming the requested audience, with no fragment. Only one is permitted per token, and naming several is refused with `invalid_target`.")
model ResourceIndicator {
  @minLength(1)
  value: url;
}


@doc("RFC 9728 Protected Resource Metadata. For the realm's IdMagic API it is derived from the ApiTokenScopes, and for an external resource from the McpResourceServer registration.")
model ProtectedResourceMetadata {
  resource: url;
  @doc("The authorization servers protecting this resource, that is, the IdMagic issuer.")
  authorization_servers: url[];
  scopes_supported: string[];
  bearer_methods_supported: string[] = #["header"];
  dpop_signing_alg_values_supported?: string[];
}


@doc("The JSON request in which an administrator registers or updates an McpResourceServer. `resource` is settable at creation only.")
model McpResourceServerRequest {
  @doc("Required at creation, and immutable on update, because changing it would change what the `aud` of already-issued tokens means.")
  resource?: string;
  @minLength(1)
  @maxLength(200)
  name: string;
  scopes: string[];
  state?: McpResourceServerState;
}


@doc("How an McpResourceServer is represented. The list and the detail views of the administration UI return the same representation.")
model McpResourceServerResponse {
  tenant_id: string;
  id: string;
  resource: url;
  name: string;
  scopes: string[];
  state: McpResourceServerState;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The McpResourceServer list of the administration API.")
model McpResourceServerListResponse {
  resource_servers: McpResourceServerResponse[];
}


@doc("The store record of a refresh token. The plain value is never stored. Its state follows the RefreshTokenLifecycle state machine. The tenant is derived from the globally identified User and OAuth2Client rather than kept redundantly on the token row, and the use case checks the tenant-local invariant of both.")
model RefreshTokenRecord {
  id: string;
  @doc("The SHA-256 hash.")
  hash: string;
  @doc("The identifier every token descended from one authorization code exchange shares.")
  family_id: string;
  parent_id?: string;
  client_id: string;
  user_id: string;
  scopes: string[];
  state: RefreshTokenState;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
  @doc("The absolute deadline, which a rotation does not extend.")
  absolute_expires_at: utcDateTime;
  sender_constraint?: RefreshTokenSenderConstraint;
  @doc("The `sid` (OIDC session id) of the AuthorizationCodeRecord it came from. A rotation carries the parent's\nvalue over unchanged. Null means an issuance with no browser session, client_credentials for instance.\nEvery family whose value matches the `sessionId` of Authentication's SessionEnded -- possibly spanning several\nclients -- is resolved through this field, so a session revocation moves them all to RevokeToken at once.")
  sid?: string;
  @doc("The resource indicator it came from (`AuthorizationCodeRecord.resource`).\nA rotation carries the parent's value over unchanged, so the audience stays strictly confined to the same\nMcpResourceServer afterwards. Null means an issuance with no `resource`.")
  resource?: ResourceIndicator;
}


@doc("The record that a given browser session (`sid`) issued an authentication result to a particular OAuth2Client\n(an RP participation). The `sid` equals Authentication's `LoginSession.id` and is shared across RPs. The session state\nitself is not copied here; this exists only as the index that resolves which RPs to notify by front-channel and\nback-channel logout when a session is revoked.")
model ClientSession {
  @doc("The OIDC session id, equal to `LoginSession.id`.")
  sid: string;
  client_id: string;
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  first_issued_at: utcDateTime;
  @doc("It is updated on every re-issuance (re-authorization) to the same RP. It has no effect on resolving where to notify.")
  last_issued_at: utcDateTime;
}


@doc("The record of one back-channel logout delivery attempt. The local session and refresh token revocation is\nalready settled before this record is created, and a failed delivery does not undo it.\nThe delivery itself is delegated to a durable job of the Jobs bounded context\n(`kind=backchannel_logout_delivery`).")
model LogoutNotification {
  id: string;
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  @doc("The OIDC session id to notify about.")
  sid: string;
  @doc("The RP to notify.")
  client_id: string;
  @doc("The `jti` of the logout token issued. The RP uses it for replay detection and IdMagic for audit correlation.")
  logout_token_jti: string;
  @doc("Where it is delivered, a snapshot of the client's `backchannel_logout_uri` taken when delivery began.")
  target_uri: url;
  state: LogoutNotificationState;
  attempts: int64 = 0;
  last_error?: string;
  @doc("The `JobRef.id` of the Jobs context, correlating the delivery run.")
  job_id?: string;
  created_at: utcDateTime;
  delivered_at?: utcDateTime;
}


@doc("The delivery state of a LogoutNotification. It does not map one-to-one onto the JobStatus of Jobs; it states only whether the delivery itself succeeded, while the job's own retries and lease stay with Jobs.")
enum LogoutNotificationState {
  Pending: "Pending",
  Delivered: "Delivered",
  Failed: "Failed",
}


@doc("The server-internal intermediate state that tracks the flow.")
model AuthorizationRequest {
  id: string;
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  state: AuthorizationCodeState;
  client_id: string;
  redirect_uri: url;
  response_type: ResponseType;
  scope: string;
  state_param?: string;
  nonce?: string;
  code_challenge: string;
  code_challenge_method: CodeChallengeMethod;
  prompt?: string;
  max_age?: int64;
  par_request_uri?: string;
  user_id?: string;
  auth_time?: int64;
  @doc("The methods by which authentication succeeded, propagated from the AuthenticationContext by `advance(authenticate_user)`.")
  amr?: string[];
  @doc("The URN derived from the amr according to the AuthenticationContextPolicy.")
  acr?: string;
  @doc("The `acr_values` parameter the client gave to `/authorize` (RFC 8176). It decides step-up.")
  acr_values?: string;
  @doc("The OIDC session id propagated from `AuthenticationContext.session_id` by `advance(authenticate_user)`,\nequal to `LoginSession.id` of the Authentication context. It is carried into the AuthorizationCodeRecord,\nthe RefreshTokenRecord, and the IdTokenClaims, and correlates both the family revocation a session revocation starts\nand the front-channel and back-channel logout.")
  sid?: string;
  @doc("RFC 9396: the structured details the client requested. Only those already validated and conforming to a registered type are kept.")
  authorization_details?: AuthorizationDetail[];
  created_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("The persistent record of an issued authorization code. The code itself is short-lived. Its state follows AuthorizationCodeRecordLifecycle, which describes the lifecycle of the code itself along a different axis from AuthorizationCodeFlow.")
model AuthorizationCodeRecord {
  code: string;
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  authorization_request_id: string;
  client_id: string;
  user_id: string;
  scopes: string[];
  redirect_uri: url;
  code_challenge: string;
  code_challenge_method: CodeChallengeMethod;
  nonce?: string;
  auth_time: int64;
  @doc("The authentication methods carried over from the authorization request. They are emitted into the ID token and the access token.")
  amr?: string[];
  @doc("The ACR URN carried over from the authorization request. It is emitted into the ID token and the access token.")
  acr?: string;
  @doc("The OIDC session id carried over from the authorization request. It propagates into the `sid` claim of the ID token and into the `sid` of the RefreshTokenRecord issued.")
  sid?: string;
  @doc("RFC 9396: the structured details consented to. The exchange reflects them into the tokens issued.")
  authorization_details?: AuthorizationDetail[];
  state: AuthorizationCodeRecordState;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
  @doc("When the state became `redeemed`, for audit.")
  redeemed_at?: utcDateTime;
  @doc("The refresh family issued by a successful exchange, used to revoke the family when a replay is detected.")
  issued_family_id?: string;
}


@doc("The stored record of a Pushed Authorization Request. Its state follows PARRecordLifecycle.")
model PARRecord {
  request_uri: string;
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  client_id: string;
  parameters: Record<string>;
  state: PARRecordState;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("The device authorization record of RFC 8628. The `device_code` is stored as a hash alone.")
model DeviceAuthorization {
  device_code_hash: string;
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  @doc("A short code that avoids confusable characters.")
  user_code: string;
  client_id: string;
  scopes: string[];
  state: DeviceCodeState;
  user_id?: string;
  auth_time?: int64;
  interval_seconds: int64;
  last_polled_at?: utcDateTime;
  issued_family_id?: string;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("The approval request that holds a client's request until a human settles it. The state and the content of the\ndecision are kept apart from the transport bookkeeping of CIBA, but they sit in the same record so that a competing\ndecision and a poll serialize inside one store boundary. The `id` is a UUID, and only the SHA-256 hash of the\n`auth_req_id` is stored, as the index.")
model ApprovalRequest {
  id: string;
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  @doc("The client that raised the approval request.")
  client_id: string;
  @doc("The Agent bound to the client, when there is one. It is used on the approval screen and in the `agent_id` claim of the tokens issued.")
  agent_id?: string;
  @doc("The `sub` of the User who decides. It is resolved from `login_hint` or `id_token_hint`, and from then on only that User may decide.")
  user_id: string;
  @doc("The scopes of the token to issue if it is approved.")
  scopes: string[];
  @doc("RFC 9396: the structured permission request being approved. The approval screen renders it for a human to read.")
  authorization_details?: AuthorizationDetail[];
  @doc("The short identifying text shown on the approval screen so the wrong request cannot be approved by mistake (CIBA Core section 7.1, `binding_message`).")
  @maxLength(64)
  binding_message?: string;
  state: ApprovalRequestState;
  @doc("The SHA-256 hash of the CIBA `auth_req_id`. It is an index of the transport layer and takes no part in the meaning of the decision.")
  auth_req_id_hash: string;
  @doc("Transport bookkeeping: the minimum polling interval in seconds the client is told to keep. It rises on every `slow_down`.")
  interval_seconds: int64;
  @doc("Transport bookkeeping: when it was last polled. It decides `slow_down`.")
  last_polled_at?: utcDateTime;
  requested_at: utcDateTime;
  expires_at: utcDateTime;
  decided_at?: utcDateTime;
  consumed_at?: utcDateTime;
}


@doc("The sender-constrained proof (RFC 7800).")
model SenderConstraintCnf {
  @doc("The SHA-256 thumbprint of the DPoP key (RFC 9449 section 6.1).")
  jkt?: string;
  @doc("The thumbprint of the mTLS client certificate (`x5t#S256`, RFC 8705).")
  x5t_s256?: string;
}


@doc("The claims of a DPoP proof JWT (RFC 9449 section 4.2). `typ` is `dpop+jwt`, and the public key is presented in the `jwk` header.")
model DpopProofClaims {
  @doc("An identifier unique to each proof. It cannot be reused within the replay window.")
  jti: string;
  @doc("The HTTP method of the request it covers.")
  htm: string;
  @doc("The HTTP URI of the request it covers, without the query and the fragment.")
  htu: url;
  @doc("When the proof was issued. Anything outside the permitted clock skew is refused.")
  iat: utcDateTime;
  @doc("The base64url of the SHA-256 of the access token being presented (RFC 9449 section 4.3). It is required when accessing a protected resource, and absent at the token endpoint, where there is no such token yet.")
  ath?: string;
}


@doc("The sender constraint attached to a refresh token. It holds what it was bound to at issuance: the DPoP key thumbprint or the mTLS certificate thumbprint.")
model RefreshTokenSenderConstraint {
  type: SenderConstraintKind;
  jkt?: string;
  x5t_s256?: string;
}


@doc("The claims of a JWT access token (RFC 9068). TTL 600 seconds.")
model AccessTokenClaims {
  iss: url;
  @doc("For `client_credentials` it equals the client_id.")
  sub: string;
  aud: Audience;
  client_id: string;
  @doc("The scopes, space-separated.")
  scope: string;
  @doc("Unix epoch seconds")
  exp: int64;
  iat: int64;
  nbf?: int64;
  jti: string;
  auth_time?: int64;
  acr?: string;
  amr?: string[];
  cnf?: SenderConstraintCnf;
  @doc("The `act` claim of RFC 8693 section 4.1, naming the current actor of the delegation. When the `subject_token` carried an `act`, they nest.")
  act?: Record<unknown>;
  @doc("The `may_act` claim of RFC 8693 section 4.4, naming the actors permitted to act on the subject's behalf.")
  may_act?: Record<unknown>;
  @doc("RFC 9396: the structured details bound to the token. It is the resource server's checkpoint. Optional, for backward compatibility.")
  authorization_details?: AuthorizationDetail[];
  @doc("The id of the Agent the token was issued for, present only when the issuing client is bound to one. It marks the token as belonging to a non-human principal, which introspection reads to derive the delegation mode (REQ-OAUTH2-049). It says nothing about the Agent's kind: a `supervised` Agent's token carries the same claim, and the approval that permitted it is traceable through the audit trail instead (REQ-OAUTH2-050).")
  agent_id?: string;
}


@doc("The claims of an OpenID Connect ID token. TTL 3600 seconds.")
model IdTokenClaims {
  iss: url;
  sub: string;
  aud: Audience;
  exp: int64;
  iat: int64;
  auth_time: int64;
  @doc("The OIDC session id (OpenID Connect Front-Channel and Back-Channel Logout 1.0). It is present only for an\nissuance bound to a browser session. The RP sends it back to `/end_session` as the `id_token_hint`, and it correlates\nfront-channel and back-channel logout.")
  sid?: string;
  nonce?: string;
  acr?: string;
  amr?: string[];
  azp?: string;
  at_hash?: string;
  c_hash?: string;
  name?: string;
  preferred_username?: string;
  email?: string;
  email_verified?: boolean;
  @doc("The additional claims formed by merging the scope-driven attribute claims (ClaimsForScopes) with the per-client\nClaimMapping claim policy. When a key collides with a standard claim, the standard claim\nwins.")
  additional_claims?: Record<unknown>;
}


@doc("A string, or an array of one or more strings (JWT, RFC 7519 section 4.1.3).")
model Audience {
  @doc("A JSON string, or an array of one or more JSON strings.")
  one_or_many: string | string[];
}


@doc("The metadata of RFC 8414 and OpenID Connect Discovery 1.0.")
model DiscoveryDocument {
  issuer: url;
  authorization_endpoint: url;
  token_endpoint: url;
  userinfo_endpoint: url;
  jwks_uri: url;
  introspection_endpoint: url;
  revocation_endpoint: url;
  pushed_authorization_request_endpoint: url;
  device_authorization_endpoint: url;
  @doc("CIBA Core section 4: the backchannel authentication endpoint. It is advertised only when CIBA is offered.")
  backchannel_authentication_endpoint?: url;
  @doc("CIBA Core section 4: the token delivery modes implemented. This implementation offers poll alone and advertises neither ping nor push.")
  backchannel_token_delivery_modes_supported?: string[] = #["poll"];
  @doc("CIBA Core section 4: the `user_code` parameter is not supported. The approval screen is protected by the user's own authenticated session and a step-up, so it carries no second shared secret.")
  backchannel_user_code_parameter_supported?: boolean = false;
  registration_endpoint: url;
  end_session_endpoint?: url;
  frontchannel_logout_supported: boolean = true;
  frontchannel_logout_session_supported: boolean = true;
  backchannel_logout_supported: boolean = true;
  backchannel_logout_session_supported: boolean = true;
  check_session_iframe?: url;
  scopes_supported: string[] = #["openid", "profile", "email", "phone", "address", "custom_attribute", "offline_access", "idmagic.admin", "idmagic.account", "account:read", "account:write", "account:mfa:write", "account:sessions:write", "account:consents:write", "account:password:write"];
  response_types_supported: ResponseType[];
  response_modes_supported?: ResponseMode[];
  grant_types_supported: GrantType[];
  token_endpoint_auth_methods_supported: TokenEndpointAuthMethod[];
  introspection_endpoint_auth_methods_supported: TokenEndpointAuthMethod[] = #[TokenEndpointAuthMethod.ClientSecretBasic, TokenEndpointAuthMethod.PrivateKeyJwt, TokenEndpointAuthMethod.TlsClientAuth];
  revocation_endpoint_auth_methods_supported: TokenEndpointAuthMethod[] = #[TokenEndpointAuthMethod.ClientSecretBasic, TokenEndpointAuthMethod.PrivateKeyJwt, TokenEndpointAuthMethod.TlsClientAuth, TokenEndpointAuthMethod.None];
  id_token_signing_alg_values_supported: SignatureAlgorithm[];
  code_challenge_methods_supported: CodeChallengeMethod[];
  require_pushed_authorization_requests: boolean;
  require_pkce: boolean;
  authorization_response_iss_parameter_supported?: boolean;
  dpop_signing_alg_values_supported?: SignatureAlgorithm[];
  tls_client_certificate_bound_access_tokens?: boolean;
  @doc("Advertises that a client_id may also be resolved on each request as an HTTPS URL, a Client ID Metadata Document.")
  client_id_metadata_document_supported: boolean = true;
  subject_types_supported: string[] = #["public"];
  claims_supported?: string[] = #["sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "acr", "amr", "azp", "name", "given_name", "family_name", "preferred_username", "email", "email_verified", "updated_at"];
  acr_values_supported?: string[] = #["urn:idmagic:acr:pwd", "urn:idmagic:acr:mfa"];
  ui_locales_supported?: string[] = #["en", "ja"];
  @doc("RFC 9396: the `authorization_details` types this tenant accepts, derived from its enabled AuthorizationDetailTypes.")
  authorization_details_types_supported?: string[];
}


@doc("The body of a dynamic client registration request to `/register` (RFC 7591). The OAuth2Client that results is derived from it.")
model ClientRegistrationRequest {
  redirect_uris: url[];
  client_name?: string;
  client_type?: ClientType;
  grant_types: GrantType[];
  response_types?: ResponseType[];
  token_endpoint_auth_method?: TokenEndpointAuthMethod;
  scope?: string;
  jwks_uri?: url;
  jwks?: JWKS;
  tls_client_auth_subject_dn?: string;
  id_token_signed_response_alg?: SignatureAlgorithm;
  fapi_profile?: FapiProfile;
  require_pkce?: boolean;
  require_pushed_authorization_requests?: boolean;
  dpop_bound_access_tokens?: boolean;
}


@doc("The Client ID Metadata Document fetched from an HTTPS URL the client hosts itself. The client_id is that fetch URL, and it has to match the client_id inside the document exactly. A document whose `token_endpoint_auth_method` is absent, or is anything other than `none`, is refused.")
model ClientIDMetadataDocument {
  @doc("It has to match the URL it was fetched from exactly.")
  client_id: url;
  client_name: string;
  client_uri?: url;
  logo_uri?: url;
  redirect_uris: url[];
  grant_types?: GrantType[];
  response_types?: ResponseType[];
  @doc("Absent is read as `none`. A document declaring any other value is refused.")
  token_endpoint_auth_method?: TokenEndpointAuthMethod;
  @doc("Absent permits `openid` alone.")
  scope?: string;
}


@doc("The JSON request in which an administrator registers or updates an `authorization_details` type (RFC 9396). `type` is settable at creation only.")
model AuthorizationDetailTypeRequest {
  @doc("Required at creation. On update the `type` from the path is used.")
  @maxLength(100)
  type?: string;
  @maxLength(500)
  description?: string;
  schema: AuthorizationDetailsSchema;
  @minLength(1)
  @maxLength(500)
  display_template: string;
  state?: AuthorizationDetailTypeState;
}


@doc("How an `authorization_details` type is represented. The list and the detail views of the administration UI return the same representation.")
model AuthorizationDetailTypeResponse {
  tenant_id: string;
  type: string;
  description?: string;
  schema: AuthorizationDetailsSchema;
  display_template: string;
  state: AuthorizationDetailTypeState;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The `authorization_details` type list of the administration API.")
model AuthorizationDetailTypeListResponse {
  types: AuthorizationDetailTypeResponse[];
}


@doc("An administrator creates an OAuth client in their tenant. The generated `client_secret` is returned once, in the creation response.")
model AdminOAuth2ClientCreateRequest {
  redirect_uris: url[];
  @maxLength(200)
  client_name?: string;
  client_type?: ClientType;
  grant_types: GrantType[];
  response_types?: ResponseType[];
  token_endpoint_auth_method?: TokenEndpointAuthMethod;
  scope?: string;
  jwks_uri?: url;
  jwks?: JWKS;
  tls_client_auth_subject_dn?: string;
  fapi_profile?: FapiProfile;
  require_pushed_authorization_requests?: boolean;
  dpop_bound_access_tokens?: boolean;
}


@doc("An administrator updates the client metadata. The client_id, the authentication method, and the secrets are left alone; secret rotation has its own endpoint.")
model AdminOAuth2ClientUpdateRequest {
  @maxLength(200)
  client_name?: string;
  redirect_uris?: url[];
  grant_types?: GrantType[];
  response_types?: ResponseType[];
  scope?: string;
  require_pushed_authorization_requests?: boolean;
  dpop_bound_access_tokens?: boolean;
}


@doc("How the administration API represents a client. `client_secret` and `client_secret_hash` are deliberately not part of the contract.")
model AdminOAuth2ClientResponse {
  tenant_id: string;
  client_id: string;
  client_name?: string;
  client_type: ClientType;
  redirect_uris: url[];
  grant_types: GrantType[];
  response_types: ResponseType[];
  token_endpoint_auth_method: TokenEndpointAuthMethod;
  scope: string;
  jwks_uri?: url;
  jwks?: JWKS;
  tls_client_auth_subject_dn?: string;
  id_token_signed_response_alg: SignatureAlgorithm;
  require_pushed_authorization_requests: boolean;
  dpop_bound_access_tokens: boolean;
  fapi_profile: FapiProfile;
  created_at: utcDateTime;
}


@doc("The client creation response. The `client_secret` is returned once, and only when one was generated.")
model AdminOAuth2ClientCreateResponse {
  client: AdminOAuth2ClientResponse;
  client_secret?: string;
}


@doc("The clients of the tenant.")
model AdminOAuth2ClientListResponse {
  clients: AdminOAuth2ClientResponse[];
}


@doc("The non-sensitive metadata of a secret credential the Application editor shows. The state is derived from `revoked_at`, then `expires_at`, then the time of evaluation.")
model ClientSecretCredentialMetadata {
  id: string;
  created_at: utcDateTime;
  expires_at?: utcDateTime;
  revoked_at?: utcDateTime;
  status: ClientSecretCredentialStatus;
}


@doc("The request that issues an additional client secret with an expiry, leaving existing credentials alone. `expires_in_days` is 90 when left out, and only 1 to 730 days is accepted.")
model IssueClientSecretRequest {
  expires_in_days?: int64;
}


@doc("The response returning the additionally issued secret, once. It carries no existing secret, no hash, and no way to read the plaintext again.")
model IssueClientSecretResponse {
  client_secret: string;
  credential: ClientSecretCredentialMetadata;
  credentials: ClientSecretCredentialMetadata[];
}


@doc("The non-sensitive credential metadata after one credential has been revoked. Neither the secret nor a hash is included.")
model RevokeClientSecretResponse {
  credentials: ClientSecretCredentialMetadata[];
}


@doc("The rotation request, naming how many days the existing secret overlaps. 0 revokes it at once, 1 to 30 gives it a grace, and leaving it out means 7 days.")
model RotateClientSecretRequest {
  grace_days?: int64;
}


@doc("The rotation response, returning the new secret once. There is no API to read it afterwards.")
model RotateClientSecretResponse {
  client_secret: string;
  grace_until?: utcDateTime;
  credentials: ClientSecretCredentialMetadata[];
}


@doc("How the administration API represents a Consent of the tenant. Because the client_id and user_id are UUIDs, the resolved human-readable names (`client_name` and `preferred_username`) are returned alongside them for display.")
model AdminConsentResponse {
  user_id: string;
  @doc("The display name the user_id resolved to. Left out when no such User is found.")
  preferred_username?: string;
  client_id: string;
  @doc("The display name the client_id resolved to. It resolves in the order `OAuth2Client.client_name`, then the\nApplication catalogue name, and falls back to the client_id itself when there is neither.")
  client_name: string;
  scopes: string[];
  state: ConsentState;
  granted_at: utcDateTime;
  expires_at: utcDateTime;
  revoked_at?: utcDateTime;
}


@doc("The Consents of the tenant.")
model AdminConsentListResponse {
  consents: AdminConsentResponse[];
}


@doc("The end user's own connected applications, that is, their active Consents (self-service).")
model AccountConsentListResponse {
  consents: AccountConsentResponse[];
}


@doc("The token endpoint request, sent as `application/x-www-form-urlencoded`.")
model TokenRequest {
  grant_type: GrantType;
  code?: string;
  redirect_uri?: string;
  code_verifier?: string;
  refresh_token?: string;
  device_code?: string;
  @doc("The `auth_req_id` of the CIBA grant (`urn:openid:params:grant-type:ciba`). Until the approval is settled it returns `authorization_pending` or `slow_down`, and once settled it is exchanged for tokens exactly once.")
  auth_req_id?: string;
  client_id?: string;
  client_secret?: string;
  client_assertion?: string;
  client_assertion_type?: string;
  scope?: string;
  @doc("The `subject_token` of the RFC 8693 token-exchange grant. It is either a self-issued access token (`urn:ietf:params:oauth:token-type:access_token`) or an external attestation JWT (`urn:ietf:params:oauth:token-type:jwt`) issued by the issuer of a registered WorkloadTrustBundle.")
  subject_token?: string;
  @doc("The URN naming the kind of `subject_token`. Left out, it is treated as a self-issued access token.")
  subject_token_type?: string;
  @doc("The optional `actor_token` of the RFC 8693 token-exchange grant. It has to be a self-issued access token.")
  actor_token?: string;
  actor_token_type?: string;
  requested_token_type?: string;
  @doc("RFC 9396: the JSON string of structured details asking for a downscope during a token exchange. Optional.")
  authorization_details?: string;
  @doc("RFC 8707: the resource indicator for an authorization_code redemption, client_credentials, device_code, or token exchange. Naming the parameter more than once gives `invalid_target`. The refresh_token grant does not name it explicitly; the resource bound at the first issuance is carried automatically across rotations.")
  resource?: ResourceIndicator;
}


@doc("The response body of a successful `/token` (RFC 6749 section 5.1). It returns the tokens issued and their kind.")
model TokenResponse {
  access_token: string;
  @doc("\"Bearer\" or \"DPoP\"")
  token_type: string;
  expires_in?: int64;
  refresh_token?: string;
  id_token?: string;
  scope?: string;
}


@doc("The `/introspect` request (RFC 7662 section 2.1), by which a resource server asks the authorization server whether a token is valid.")
model IntrospectionRequest {
  token: string;
  token_type_hint?: TokenType;
}


@doc("How the token's subject is currently acting. This derived value comes from the `act` chain and principal type and has no independent persistent state.\n`OnBehalfOf` means another principal acts for the subject (`act` contains an actor different from the subject).\n`Autonomous` means no delegation is present and the subject is a non-human principal (Agent or client).\n`Direct` means no delegation is present and the subject is a human user.")
enum DelegationMode {
  Direct: "direct",
  Autonomous: "autonomous",
  OnBehalfOf: "on_behalf_of",
}


@doc("The `/introspect` response (RFC 7662 section 2.2). It returns whether the token is active and, when it is, its metadata.")
model IntrospectionResponse {
  active: boolean;
  scope?: string;
  client_id?: string;
  sub?: string;
  aud?: Audience;
  exp?: int64;
  iat?: int64;
  token_type?: string;
  jti?: string;
  cnf?: SenderConstraintCnf;
  @doc("The `act` claim of RFC 8693 section 4.1. For a delegated token it returns the current actor and the actor chain.")
  act?: Record<unknown>;
  @doc("Whether an active token represents autonomous, on-behalf-of, or direct execution.\nThe authorization server returns the derived value so resource servers do not reimplement the `act` and principal-type rules.\nThe value is produced by the same derivation function used for audit events.")
  delegation_mode?: DelegationMode;
}


@doc("The `/revoke` request (RFC 7009 section 2.1), which revokes the named token.")
model RevokeRequest {
  token: string;
  token_type_hint?: TokenType;
}


@doc("It takes the authorize parameters as a form.")
model ParRequest {
  parameters: Record<string>;
}


@doc("The `/par` response (RFC 9126 section 2.2). It issues a short-lived `request_uri` that a later `/authorize` refers to.")
model ParResponse {
  request_uri: string;
  expires_in: int64;
}


@doc("The `/device_authorization` request (RFC 8628 section 3.1), starting the flow from a device with constrained input.")
model DeviceAuthorizationRequest {
  scope?: string;
  client_id?: string;
  client_secret?: string;
  client_assertion?: string;
  client_assertion_type?: string;
}


@doc("The `/device_authorization` response (RFC 8628 section 3.2). It returns the `device_code`, the `user_code`, and the `verification_uri`, and the device moves on to polling.")
model DeviceAuthorizationResponse {
  device_code: string;
  user_code: string;
  verification_uri: url;
  verification_uri_complete?: url;
  expires_in: int64;
  interval?: int64;
}


@doc("The `/bc-authorize` request (CIBA Core section 7.1), by which a client raises a request for human approval.\nExactly one of `login_hint` and `id_token_hint` is given, and when neither resolves the request is refused\nfail-closed with `unknown_user_id`. `user_code` is not supported and is not advertised in discovery.")
model BackchannelAuthenticationRequest {
  @doc("The hint naming the User to ask. It resolves as a username or an email address.")
  login_hint?: string;
  @doc("A valid ID token this OP issued. Its signature, `iss`, `aud`, `sub`, and `exp` are verified to resolve the User to ask.")
  id_token_hint?: string;
  @doc("The scopes of the token to issue if it is approved. It is required, has to hold `openid`, and has to be a subset of the scopes the client declared.")
  scope: string;
  @doc("RFC 9396: the structured permission request being approved, as a JSON string. Only registered types are accepted.")
  authorization_details?: string;
  @doc("The short identifying text shown on the approval screen (CIBA Core section 7.1).")
  @maxLength(64)
  binding_message?: string;
  @doc("The lifetime the client asks for, in seconds. A positive integer, at most 600. 300 when left out.")
  requested_expiry?: int64;
  client_id?: string;
  client_secret?: string;
  client_assertion?: string;
  client_assertion_type?: string;
}


@doc("The `/bc-authorize` response (CIBA Core section 7.3). The `auth_req_id` is a bearer secret, and the\nclient presents it to the token endpoint under the CIBA grant while polling.")
model BackchannelAuthenticationResponse {
  auth_req_id: string;
  expires_in: int64;
  @doc("The minimum polling interval, in seconds, the client has to keep.")
  interval?: int64;
}


@doc("One pending approval request the end-user account portal returns (self-service). It is fixed to requests\naddressed to the caller, so no `user_id` appears. It does not let `binding_message` stand in for the content: who\nasked and what they asked for are returned side by side.")
model AccountApprovalRequest {
  id: string;
  client_id: string;
  @doc("The display name the client_id resolved to. It resolves in the order `OAuth2Client.client_name`, then the\nApplication catalogue name, and falls back to the client_id itself when there is neither.")
  client_name: string;
  @doc("The display name of the Agent bound to the client, when there is one.")
  agent_name?: string;
  scopes: string[];
  authorization_details?: AuthorizationDetail[];
  binding_message?: string;
  requested_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("The approval requests pending for the end user themselves (self-service).")
model AccountApprovalRequestListResponse {
  approval_requests: AccountApprovalRequest[];
}


@doc("The end user's decision on one approval request: approve or deny.")
model AccountApprovalDecisionRequest {
  @doc("Either `approve` or `deny`.")
  decision: string;
}


@doc("The authorization parameters of `GET /authorize` or of a PAR. Every field is single-valued, and naming a parameter more than once is refused as `invalid_request`. `prompt` is a space-separated set of the unique tokens `login`, `consent`, and `none`, and `none` is never combined with another.")
model AuthorizeParameters {
  response_type: ResponseType;
  client_id: string;
  redirect_uri: string;
  scope?: string;
  state?: string;
  nonce?: string;
  code_challenge: string;
  code_challenge_method: CodeChallengeMethod;
  prompt?: string;
  max_age?: int64;
  id_token_hint?: string;
  acr_values?: string;
  request_uri?: string;
  authorization_details?: string;
  @doc("The RFC 8707 resource indicator. When it matches no registered, active McpResourceServer it is refused with `invalid_target`, and so is naming the parameter more than once.")
  resource?: ResourceIndicator;
}


@doc("The consent decision an SPA posts to `/api/auth/consent`.")
model BrowserConsentRequest {
  @doc("Either `allow` or `deny`.")
  action: string;
}


@doc("The device `user_code` decision an SPA posts to `/api/auth/device`.")
model BrowserDeviceRequest {
  user_code: string;
}


@doc("The no-store JSON an SPA uses to render the current browser authorization transaction.")
model BrowserTransactionResponse {
  @doc("Which screen to render: login, totp, consent, device, and more as factors are added.")
  kind: string;
  csrf_token: string;
  @doc("The readable name of the client shown on the consent screen. It resolves in the order `OAuth2Client.client_name`,\nthe Application catalogue name, then the client_id, which is a UUID and is avoided in raw form.")
  client_name?: string;
  scopes?: string[];
}


@doc("The no-store JSON an SPA uses to render the device `user_code` screen.")
model BrowserDeviceContextResponse {
  kind: string;
  user_code?: string;
  csrf_token: string;
}


@doc("The query or form parameters of RP-Initiated Logout v1. The `post_logout_redirect_uri` is validated\nagainst the client_id. An `id_token_hint` is accepted only when this OP issued the ID token, and its signature,\n`iss`, `aud` (refused when it contradicts the client_id), `sub`, and `sid` are all verified.\n`exp` is not verified, because sending an ID token that has already expired by the time of the logout is what a\nnormal RP implementation does.")
model EndSessionParameters {
  client_id?: string;
  id_token_hint?: string;
  post_logout_redirect_uri?: url;
  state?: string;
}


@doc("One entry of a FrontChannelLogout result, corresponding to a single iframe the EndSession response embeds.")
model FrontChannelLogoutTarget {
  client_id: string;
  @doc("The registered `frontchannel_logout_uri`, with `iss` and `sid` added when `frontchannel_logout_session_required` is set.")
  iframe_uri: url;
}


@doc("OpenID Connect Core section 5.3. It carries the mutable OIDC claims.")
model UserInfoResponse {
  sub: string;
  @doc("The additional claims formed by merging the scope-driven attribute claims (ClaimsForScopes) with the per-client\nClaimMapping claim policy. When a key collides with a standard claim, the standard claim wins.")
  claims?: Record<unknown>;
}


@doc("The request is missing a required parameter, repeats one, or carries an invalid value (RFC 6749 section 5.2, invalid_request). The OAuth 2.0 and OpenID Connect endpoints report this condition with the OAuthError body; the generic APIs report the same condition as InvalidRequestError, an RFC 9457 Problem Details body.")
model OAuthInvalidRequestError is OAuthError;


@doc("The resource owner or the authorization server denied the request (RFC 6749 section 5.2, access_denied). The OAuth 2.0 and OpenID Connect endpoints report this condition with the OAuthError body; the generic APIs report the same condition as AccessDeniedError, an RFC 9457 Problem Details body.")
model OAuthAccessDeniedError is OAuthError;


@doc("The shared OAuth 2.0 / OpenID Connect error response body (RFC 6749 section 5.2), written by `support_http.OAuthErrorBody` with `Content-Type: application/json`. Every endpoint whose error shape a standard fixes returns this body rather than RFC 9457 Problem Details, so that off-the-shelf clients can read it. The named `*Error` models declared as `is OAuthError` share this body and are told apart by the value of `error`.")
model OAuthError {
  @doc("The protocol error code. Identifies which condition occurred.")
  error: OAuthErrorCode;
  @doc("Human-readable explanation of the occurrence. Always English and never localized; the value is empty when the server has nothing safe to disclose, and the member is present either way.")
  error_description: string;
}


@doc("Client authentication failed, or the client is not registered (RFC 6749 section 5.2, invalid_client). No detail is returned, so the response does not disclose whether the `client_id` exists.")
model InvalidClientError is OAuthError;


@doc("An authorization code, refresh token, device code, or other grant is invalid, expired, replayed, or does not match the `redirect_uri` or the PKCE verifier (RFC 6749 section 5.2, invalid_grant).")
model InvalidGrantError is OAuthError;


@doc("The client is not authorized to use this `grant_type` (RFC 6749 section 5.2, unauthorized_client).")
model UnauthorizedClientError is OAuthError;


@doc("The authorization server does not support this `grant_type` (RFC 6749 section 5.2, unsupported_grant_type).")
model UnsupportedGrantTypeError is OAuthError;


@doc("The requested scope is malformed, unknown, or not permitted for the client (RFC 6749 section 5.2, invalid_scope).")
model InvalidScopeError is OAuthError;


@doc("The `resource` parameter is unregistered, disabled, or given more than once (RFC 8707 section 2, invalid_target). Rejected fail-closed.")
model InvalidTargetError is OAuthError;


@doc("The access token is expired, tampered with, malformed, or revoked (RFC 6750 section 3.1, invalid_token).")
model InvalidTokenError is OAuthError;


@doc("Verification of the DPoP proof JWT failed on its signature, `jti`, `htu`/`htm`, nonce, clock skew, or a similar check (RFC 9449 section 7.1, invalid_dpop_proof).")
model InvalidDpopProofError is OAuthError;


@doc("The scope of the presented token does not cover the requested operation (RFC 6750 section 3.1, insufficient_scope).")
model InsufficientScopeError is OAuthError;


@doc("The schema definition of an AuthorizationDetailType is invalid: it failed validation, or it references an unknown Semantics value. Returned by the administration API, not by the token endpoint. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_type`.")
model InvalidAuthorizationDetailTypeError is ProblemDetails;


@doc("In the Device Authorization Grant, the `device_code` has passed its expiry (RFC 8628 section 3.5, expired_token).")
model ExpiredTokenError is OAuthError;


@doc("The client is polling the Device Authorization Grant too fast and must add five seconds to its interval (RFC 8628 section 3.5, slow_down).")
model SlowDownError is OAuthError;


@doc("In the Device Authorization Grant, the user has not approved yet (RFC 8628 section 3.5, authorization_pending). The client waits `interval` seconds and retries.")
model AuthorizationPendingError is OAuthError;


@doc("On the CIBA /bc-authorize endpoint, neither `login_hint` nor `id_token_hint` resolves to a User to ask for approval (OpenID Connect CIBA Core section 13, unknown_user_id). A user who is not active, and a user in another tenant, fold into the same error so the difference does not disclose whether the User exists.")
model UnknownUserIdError is OAuthError;


@doc("On the CIBA /bc-authorize endpoint, the `binding_message` is too long or unsuitable for display (OpenID Connect CIBA Core section 13, invalid_binding_message).")
model InvalidBindingMessageError is OAuthError;


@doc("An unexpected internal failure on the authorization server (RFC 6749 section 4.1.2.1, server_error).")
model ServerError is OAuthError;


@doc("The fixed-window rate limit settings of one route. The operator tunes the threshold and the window through environment variables; this model fixes only the shape and embeds no concrete value in the specification.")
model EndpointRateLimitPolicy {
  @doc("The identifier of the route it covers: `token`, `authorize`, `par`, `device_authorization`, `password_reset`, `login`, and so on.")
  policy_id: string;
  max_requests: int64;
  window_seconds: int64;
  @doc("What happens when the shared counter store is unreachable. Fixed at `fail_closed`.")
  fail_mode: string;
}


@doc("A new client was registered dynamically at `/register` (RFC 7591). It is the starting point of the client registration audit trail.")
model ClientRegistered {
  occurredAt: utcDateTime;
  @doc("The tenant the client belongs to.")
  tenantId: string;
  clientId: string;
  clientType?: ClientType;
}


@doc("For a client_id in URL form, the Client ID Metadata Document was fetched and verified, and the authorization request went on. Nothing is registered persistently.")
model ClientIdMetadataDocumentResolved {
  occurredAt: utcDateTime;
  @doc("The tenant the authorization request belongs to.")
  tenantId: string;
  @doc("The URL it was fetched from.")
  clientId: string;
}


@doc("For a client_id in URL form, fetching or verifying the Client ID Metadata Document failed and the request was refused fail-closed.")
model ClientIdMetadataDocumentRejected {
  occurredAt: utcDateTime;
  @doc("The tenant the authorization request belongs to.")
  tenantId: string;
  @doc("The URL it was fetched from.")
  clientId: string;
  @doc("How the refusal is classified: a failed fetch, malformed JSON, a client_id mismatch, missing `redirect_uris`, an unsupported `token_endpoint_auth_method`, and so on.")
  reason: string;
}


@doc("An administrator created an OAuth client in their tenant. It carries no secret.")
model AdminOAuth2ClientCreated {
  occurredAt: utcDateTime;
  @doc("The tenant the client belongs to. It matches the actor.")
  tenantId: string;
  actorUserId: string;
  clientId: string;
}


@doc("An administrator updated the OAuth client metadata.")
model AdminOAuth2ClientUpdated {
  occurredAt: utcDateTime;
  @doc("The tenant the client belongs to. It matches the actor.")
  tenantId: string;
  actorUserId: string;
  clientId: string;
  changedFields: string[];
}


@doc("An administrator deleted an OAuth client from their tenant.")
model AdminOAuth2ClientDeleted {
  occurredAt: utcDateTime;
  @doc("The tenant the client belongs to. It matches the actor.")
  tenantId: string;
  actorUserId: string;
  clientId: string;
}


@doc("Through the compatibility interface, an administrator rotated or immediately revoked the credential of a secret-based client. Neither the secret nor a hash is in the payload.")
model ClientSecretRotated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  clientId: string;
  graceUntil?: utcDateTime;
  revokedImmediately: boolean;
}


@doc("A ResourceOwner consented to a scope for a client. It is the record of obtaining a legal basis under GDPR Article 6.")
model ConsentGranted {
  occurredAt: utcDateTime;
  @doc("The tenant the consent was given in.")
  tenantId: string;
  userId: string;
  clientId: string;
  scopes: string[];
}


@doc("A ResourceOwner, or an Administrator of the tenant, withdrew an existing consent, exercising the right to withdraw under GDPR Article 7(3). Further issuance is blocked from then on.")
model ConsentRevoked {
  occurredAt: utcDateTime;
  @doc("The tenant the consent belongs to.")
  tenantId: string;
  @doc("The `sub` of the Administrator, when the withdrawal came through the administration API. It may be left out when the ResourceOwner withdrew it themselves.")
  actorUserId?: string;
  userId: string;
  clientId: string;
}


@doc("RFC 9396: a client asked for `authorization_details` at `/authorize`, `/par`, or `/token`, and the request passed validation against the registered type.")
model AuthorizationDetailsRequested {
  occurredAt: utcDateTime;
  tenantId: string;
  clientId: string;
  @doc("The `sub` of the principal, when it was already settled at the time of the request.")
  userId?: string;
  @doc("The types of the details requested.")
  detailTypes: string[];
}


@doc("RFC 9396: a ResourceOwner consented to the `authorization_details` presented. It becomes the yardstick for the subset check at issuance and exchange.")
model AuthorizationDetailsConsented {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  clientId: string;
  detailTypes: string[];
}


@doc("RFC 9396: the request was refused fail-closed because the `authorization_details` were malformed, named an unregistered type, did not fit the schema, went beyond the consent, or exceeded the downscope.")
model AuthorizationDetailsRejected {
  occurredAt: utcDateTime;
  tenantId: string;
  clientId: string;
  @doc("Why it was refused, `invalid_authorization_details` for instance.")
  reason: string;
}


@doc("An AuthorizationCode was issued as `/authorize` completed. It has to be exchanged at `/token` within 60 seconds.")
model AuthorizationCodeIssued {
  occurredAt: utcDateTime;
  @doc("The tenant the authorization code was issued in.")
  tenantId: string;
  clientId: string;
  userId: string;
  scopes: string[];
  codeChallengeMethod?: string;
}


@doc("An AuthorizationCode was exchanged correctly at `/token`. Reusing the same code revokes the family.")
model AuthorizationCodeRedeemed {
  occurredAt: utcDateTime;
  @doc("The tenant the exchange happened in.")
  tenantId: string;
  clientId: string;
  userId: string;
}


@doc("An access token (a JWT) was issued. When there is a `senderConstraint`, it is bound to a proof of possession through DPoP or mTLS.")
model AccessTokenIssued {
  occurredAt: utcDateTime;
  @doc("The tenant the token was issued in.")
  tenantId: string;
  jti: string;
  clientId: string;
  userId: string;
  scopes: string[];
  senderConstraint: AccessTokenSenderConstraint = AccessTokenSenderConstraint.None;
}


@doc("A refresh token was issued. The `familyId` identifies the family descended from the original authorization code and is inherited across rotations.")
model RefreshTokenIssued {
  occurredAt: utcDateTime;
  @doc("The tenant the token was issued in.")
  tenantId: string;
  tokenId: string;
  familyId: string;
  parentId?: string;
  clientId: string;
  userId: string;
}


@doc("A refresh token was exchanged at `/token`: a new child token was issued and the parent moved to `rotated`.")
model RefreshTokenRotated {
  occurredAt: utcDateTime;
  @doc("The tenant the rotation happened in.")
  tenantId: string;
  oldTokenId: string;
  newTokenId: string;
  familyId: string;
}


@doc("A token was revoked, by `/revoke` or by an administrative operation (RFC 7009). It carries an audit and an immediacy requirement.")
model TokenRevoked {
  occurredAt: utcDateTime;
  @doc("The tenant the revocation happened in.")
  tenantId: string;
  tokenType: TokenType;
  tokenId: string;
  reason?: string;
}


@doc("`/introspect` was called and returned the token's state (RFC 7662). It is used to analyse how resource servers behave.")
model TokenIntrospected {
  occurredAt: utcDateTime;
  @doc("The tenant the introspection happened in.")
  tenantId: string;
  rsClientId: string;
  tokenId: string;
  active: boolean;
}


@doc("RFC 8693: a delegated token was issued through a token exchange. `delegationDepth` is how deeply the `act` claims of the issued token nest.")
model TokenExchanged {
  occurredAt: utcDateTime;
  @doc("The tenant the token exchange happened in.")
  tenantId: string;
  @doc("The current actor of the issued token (`act.sub`).")
  actorUserId: string;
  @doc("The subject of the issued token (`sub`). It keeps the `sub` of the `subject_token`.")
  subjectUserId: string;
  @doc("The `aud` of the issued token, taken from the `resource` parameter.")
  audience: string;
  @doc("How deeply the `act` claims of the issued token nest.")
  delegationDepth: int64;
  @doc("Maximum delegation depth applied to this decision. The system default applies when the tenant has no override.\nRecording the applied limit makes the decision reproducible from the audit event.")
  maxDelegationDepth: int64;
  @doc("Delegation mode of the issued token, produced by the same derivation function used for introspection.")
  delegationMode: DelegationMode;
  @doc("The Agent the exchanging client is bound to, when one is. It makes the actor's kind readable from the event instead of\ninferred, which is what the `actor.type` and `agent.id` audit axes need (REQ-AUDIT-005).")
  agentId?: string;
  @doc("Every principal of the issued token's `act` chain, the current actor first and the subject last, each named once. The\ndepth alone says how far the delegation went but not through whom, so an intermediate actor cannot be found from it; this\nlist is what the `delegation.actor` audit axis searches (REQ-AUDIT-006). These are token subjects, not Agent ids: only the\ncurrent actor's Agent is resolvable at exchange time, and naming that one alone would misplace where in the chain an Agent\nstood. `agentId` answers the Agent-level question instead. Identifiers only -- never a username, per\nResolvableUserEventPayloadPolicy.")
  actorChain: string[];
}


@doc("RFC 8693: a token exchange request was refused fail-closed, for a policy violation or a failed check.")
model TokenExchangeRejected {
  occurredAt: utcDateTime;
  @doc("The tenant the token exchange was refused in.")
  tenantId: string;
  @doc("The current actor, when it was known at the time of the refusal. Empty when it was not.")
  actorUserId?: string;
  @doc("Why it was refused, an OAuth error code for instance.")
  reason: string;
}


@doc("RFC 9728: `/.well-known/oauth-protected-resource` served the metadata of a registered McpResourceServer.")
model ProtectedResourceMetadataServed {
  occurredAt: utcDateTime;
  @doc("The tenant the metadata was served for.")
  tenantId: string;
  @doc("The resource URI it was served for.")
  resource: string;
}


@doc("RFC 8707: an access token was issued whose audience is strictly confined to a single McpResourceServer, following the resource indicator. It covers the authorize, PAR, and token (authorization_code) paths. Issuance confined by a resource through a token exchange is expressed by `TokenExchanged.audience`, and this event is not emitted for it.")
model ResourceScopedTokenIssued {
  occurredAt: utcDateTime;
  @doc("The tenant the token was issued in.")
  tenantId: string;
  clientId: string;
  @doc("The resource URI bound to the `aud` of the issued token.")
  resource: string;
  scopes: string[];
}


@doc("RFC 8707: the request was refused fail-closed because the resource indicator was unregistered, disabled, or given more than once. It covers the authorize, PAR, and token (authorization_code) paths. A refusal during a token exchange is expressed by `TokenExchangeRejected.reason`, and this event is not emitted for it.")
model ResourceAudienceRejected {
  occurredAt: utcDateTime;
  @doc("The tenant the refusal happened in.")
  tenantId: string;
  clientId: string;
  @doc("Why it was refused, `invalid_target` for instance.")
  reason: string;
}


@doc("A refresh token already in the `rotated` state was presented again. It triggers revocation of the whole family (RFC 9700 section 4.14).")
model RefreshTokenReuseDetected {
  occurredAt: utcDateTime;
  @doc("The tenant the reuse was detected in.")
  tenantId: string;
  familyId: string;
  tokenId: string;
  clientId?: string;
}


@doc("The authorize parameters that arrived at `/par` were stored server-side and a short-lived `request_uri` was issued (RFC 9126).")
model PARStored {
  occurredAt: utcDateTime;
  @doc("The tenant the PAR was accepted in.")
  tenantId: string;
  requestUri: string;
  clientId: string;
}


@doc("RFC 8628 section 3.1: a `device_code` and a `user_code` were issued. The `user_code` does not reach the audit log.")
model DeviceAuthorizationRequested {
  occurredAt: utcDateTime;
  @doc("The tenant the device authorization was accepted in.")
  tenantId: string;
  clientId: string;
  scopes: string[];
}


@doc("RFC 8628 section 3.3: the user approved.")
model DeviceAuthorizationApproved {
  occurredAt: utcDateTime;
  @doc("The tenant the approval happened in.")
  tenantId: string;
  clientId: string;
  userId: string;
}


@doc("RFC 8628 section 3.3: the user refused, or kept entering the wrong `user_code`.")
model DeviceAuthorizationDenied {
  occurredAt: utcDateTime;
  @doc("The tenant the refusal happened in.")
  tenantId: string;
  clientId: string;
  userId?: string;
}


@doc("Issuance was refused fail-closed because the Agent's kind requires a human approval that this path does not record (REQ-OAUTH2-050). It covers `client_credentials`, token exchange, and the workload identity federation exchange; the approval-bearing path is CIBA, whose outcome is expressed by BackchannelAuthApproved instead. Carrying the kind that was read makes the decision reproducible from the audit event alone, without resolving the Agent again.")
model AgentApprovalRequired {
  occurredAt: utcDateTime;
  @doc("The tenant the refusal happened in.")
  tenantId: string;
  agentId: string;
  @doc("The client the Agent is bound to and authenticated as.")
  clientId: string;
  @doc("The kind the decision was taken on. A value outside AgentKind is recorded as read, since it is exactly the case the fail-closed rule exists for.")
  kind: string;
  @doc("The grant type the refused request asked for, `client_credentials` for instance.")
  grantType: string;
}


@doc("CIBA Core section 7.3: an approval request was raised for a human to settle. The `auth_req_id` is a bearer secret and does not reach the audit log.")
model BackchannelAuthRequested {
  occurredAt: utcDateTime;
  @doc("The tenant the approval request was accepted in.")
  tenantId: string;
  approvalRequestId: string;
  clientId: string;
  agentId?: string;
  userId: string;
  scopes: string[];
}


@doc("CIBA Core section 7.3: the User being asked approved the request.")
model BackchannelAuthApproved {
  occurredAt: utcDateTime;
  @doc("The tenant the approval happened in.")
  tenantId: string;
  approvalRequestId: string;
  clientId: string;
  @doc("The Agent the approval was raised for, when the requesting client is bound to one. It is what makes a `supervised` Agent's issuance traceable to the human who permitted it (REQ-OAUTH2-050).")
  agentId?: string;
  userId: string;
}


@doc("CIBA Core section 7.3: the User being asked refused the request.")
model BackchannelAuthDenied {
  occurredAt: utcDateTime;
  @doc("The tenant the refusal happened in.")
  tenantId: string;
  approvalRequestId: string;
  clientId: string;
  userId: string;
}


@doc("CIBA Core section 11: an approval request reached its expiry without being decided.")
model BackchannelAuthExpired {
  occurredAt: utcDateTime;
  @doc("The tenant the expiry happened in.")
  tenantId: string;
  approvalRequestId: string;
  clientId: string;
  userId: string;
}


@doc("The lifecycle event consuming an approval request, once, into approved tokens.")
model Consume {
  occurredAt: utcDateTime;
}


@doc("The lifecycle event validating an authorization request.")
model Validate {
  occurredAt: utcDateTime;
}


@doc("The event moving the lifecycle to its refusing terminal state.")
model Reject {
  occurredAt: utcDateTime;
}


@doc("The event where resource owner authentication succeeded.")
model AuthenticateUser {
  occurredAt: utcDateTime;
}


@doc("The event asking the resource owner for consent.")
model RequestConsent {
  occurredAt: utcDateTime;
}


@doc("The event where the resource owner consented.")
model GrantConsent {
  occurredAt: utcDateTime;
}


@doc("The lifecycle event issuing an authorization code.")
model IssueCode {
  occurredAt: utcDateTime;
}


@doc("The lifecycle event redeeming an authorization code, once.")
model RedeemCode {
  occurredAt: utcDateTime;
}


@doc("The lifecycle event standing for reaching the expiry.")
model Expire {
  occurredAt: utcDateTime;
}


@doc("The event where a device user code was entered.")
model EnterUserCode {
  occurredAt: utcDateTime;
}


@doc("The event where a device authorization was approved.")
model Approve {
  occurredAt: utcDateTime;
}


@doc("The event where a device authorization was refused.")
model Deny {
  occurredAt: utcDateTime;
}


@doc("The event where a device code was exchanged for tokens.")
model Exchange {
  occurredAt: utcDateTime;
}


@doc("The event rotating a refresh token into a child token.")
model Rotate {
  occurredAt: utcDateTime;
}


@doc("The lifecycle event revoking a token.")
model RevokeToken {
  occurredAt: utcDateTime;
}


@doc("The lifecycle event withdrawing a consent logically.")
model RevokeConsent {
  occurredAt: utcDateTime;
}


@doc("The lifecycle event consuming a PAR `request_uri`, once.")
model Use {
  occurredAt: utcDateTime;
}


@doc("The lifecycle event where a LogoutNotification was delivered successfully.")
model Deliver {
  occurredAt: utcDateTime;
}


@doc("The lifecycle event where a LogoutNotification reached `max_attempts` and settled as a dead letter.")
model Exhaust {
  occurredAt: utcDateTime;
}


@doc("One administrative role and the permissions it carries, as ListAdminRolePolicies reports them. The catalogue is derived from the compiled contract, so it always matches the operations the server actually exposes.")
model AdminRolePolicy {
  @doc("Role name, the value that appears in a User's `roles`.")
  name: string;
  @doc("What the role is for. Always English and never localized.")
  description: string;
  @doc("Other names that resolve to the same role.")
  aliases: string[];
  permissions: AdminRolePermission[];
}


@doc("One permission a role carries, together with the HTTP interfaces the permission unlocks.")
model AdminRolePermission {
  @doc("Permission name.")
  name: string;
  @doc("The action the permission allows, such as `read` or `write`.")
  action: string;
  @doc("What the permission allows. Always English and never localized.")
  description: string;
  interfaces: AdminRoleInterface[];
}


@doc("One HTTP interface a permission unlocks, identified the same way the compiled contract identifies it.")
model AdminRoleInterface {
  @doc("Operation id, as declared by `@TypeSpec.OpenAPI.operationId`.")
  name: string;
  @doc("HTTP method.")
  method: string;
  @doc("Route template.")
  path: string;
}


@doc("The client information response of Dynamic Client Registration (RFC 7591 section 3.2.1). It repeats the registered metadata as the server stored it, and carries `client_secret` exactly once, at registration, for a confidential client.")
model ClientRegistrationResponse {
  client_id: string;
  @doc("Returned only for a confidential client, and only in this response. It cannot be read back afterwards.")
  client_secret?: string;
  client_type: ClientType;
  redirect_uris: url[];
  grant_types: GrantType[];
  response_types: ResponseType[];
  token_endpoint_auth_method: TokenEndpointAuthMethod;
  scope: string;
  @doc("Whether the client must use pushed authorization requests (RFC 9126).")
  require_pushed_authorization_requests: boolean;
  @doc("Whether the access tokens issued to this client are DPoP-bound (RFC 9449).")
  dpop_bound_access_tokens: boolean;
  fapi_profile: FapiProfile;
  @doc("Echoed back only when the registration supplied an inline JWK Set.")
  jwks?: JWKS;
  @doc("Echoed back only when the registration supplied a JWKS URL.")
  jwks_uri?: url;
  @doc("Echoed back only for `tls_client_auth` (RFC 8705 section 2.1).")
  tls_client_auth_subject_dn?: string;
}

}
