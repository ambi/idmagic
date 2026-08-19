namespace IdMagic.Contract {

@doc("A message was sent through the EmailSender port. The port is fail-open, so a send failure does not\npropagate into the use case layer; whether this event exists is how a SIEM observes it.\n`to` is recorded as a SHA-256 hash.")
model EmailSent {
  occurredAt: utcDateTime;
  @doc("The SHA-256 of the recipient email, as hex.")
  toHash: string;
  @doc("`password_reset`, `email_change`, `email_verification`, and the like.")
  purpose: string;
  delivered: boolean;
}


@doc("The wire protocol of an upstream connection. A social provider preset appears as the initial settings of `oidc`.")
enum IdentityProviderProtocol {
  Oidc: "oidc",
  Saml: "saml",
}


@doc("The lifecycle state of a connection. Only an Active connection may be used for discovery and login routing. A newly created one starts Disabled.")
enum IdentityProviderConnectionStatus {
  Active: "active",
  Disabled: "disabled",
}


@doc("How an external identity is correlated with a local User. The default, `none`, correlates nothing automatically beyond an existing link. `verified_email` permits a unique match on a verified email, and only when the tenant says so explicitly.")
enum AccountLinkingPolicy {
  None: "none",
  VerifiedEmail: "verified_email",
}


@doc("The tenant-managed settings projecting verified upstream claims and attributes onto local identity fields. JIT provisioning requires a mapping for the subject and the username.")
model ClaimMapping {
  subject: string;
  username: string;
  email?: string;
  email_verified?: string;
  name?: string;
}


@doc("The tenant-scoped upstream trust settings. No URL supplied during a request is ever fetched; only the last-known-good endpoints, keys, and certificates from metadata validated at configuration time are used. The client secret arrives write-only as a `secret_reference` and is held envelope-encrypted under the per-tenant DEK of DataKeys.")
model IdentityProviderConnection {
  id: string;
  tenant_id: string;
  display_name: string;
  protocol: IdentityProviderProtocol;
  status: IdentityProviderConnectionStatus;
  issuer: string;
  client_id?: string;
  @doc("The write-only field carrying the client secret itself. Only the ciphertext, envelope-encrypted under the per-tenant DEK of DataKeys, is stored in the database, and neither the value nor the ciphertext appears in an API response. The field name is carried over from the `env:` reference scheme it replaced.")
  secret_reference?: string;
  @doc("A read-only derived field saying whether `secret_reference` is set. It is never used for a write carrying a value or a ciphertext; the detail view of the administration console uses it to show nothing more than set or unset.")
  client_secret_configured: boolean;
  authorization_endpoint?: string;
  token_endpoint?: string;
  jwks_uri?: string;
  saml_sso_url?: string;
  @maxLength(2048)
  saml_entity_id?: string;
  saml_signing_certificates: string[];
  claim_mapping: ClaimMapping;
  linking_policy: AccountLinkingPolicy;
  jit_provisioning: boolean;
  allowed_email_domains: string[];
  metadata_refreshed_at?: utcDateTime;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The structured result of a connection test. It checks that OIDC discovery and JWKS are reachable, that the SAML certificate parses, and that `secret_reference` resolves, meaning it decrypts. It carries none of the values themselves: no secret, no token, no certificate body.")
model IdentityProviderConnectionTestResult {
  success: boolean;
  failures: string[];
}


@doc("The link correlating an immutable external subject with a local User. One tenant, provider, and subject map to at most one local user, and one local user and provider to at most one subject.")
model FederatedIdentity {
  tenant_id: string;
  provider_id: string;
  external_subject: string;
  local_user_id: string;
  linked_at: utcDateTime;
  last_login_at?: utcDateTime;
}


@doc("The single-use ephemeral record correlating an upstream browser round trip. It binds the state, nonce, and PKCE verifier, or the RelayState and AuthnRequest ID, to a tenant and a provider, and is consumed once within ten minutes.")
model FederatedLoginAttempt {
  state: string;
  tenant_id: string;
  provider_id: string;
  protocol: IdentityProviderProtocol;
  nonce?: string;
  pkce_verifier?: string;
  request_id?: string;
  return_to?: string;
  created_at: utcDateTime;
  expires_at: utcDateTime;
  consumed_at?: utcDateTime;
}


@doc("An active upstream provider, in the form fit to expose to the sign-in UI. It carries no secret, no internal endpoint, and no certificate.")
model ExternalIdentityProviderSummary {
  id: string;
  display_name: string;
  protocol: IdentityProviderProtocol;
}


@doc("The validated redirect that starts upstream authentication.")
model FederatedLoginStartResponse {
  redirect_to: string;
}


@doc("The result of correlating a verified external identity with a local User and a LoginSession.")
model FederatedLoginCompletion {
  local_user_id: string;
  provider_id: string;
  session_id: string;
  return_to?: string;
}


@doc("The normalized result an administrator reads to check a mapping. No token, assertion, or secret is returned.")
model ExternalIdentityClaimsPreview {
  subject: string;
  username: string;
  email?: string;
  email_verified: boolean;
  name?: string;
}


@doc("The new password matches an entry in the recent password history. RFC 9457 Problem Details with `type` = `urn:idmagic:error:password_reuse`.")
model PasswordReuseError is ProblemDetails;


@doc("The target user does not meet the conditions for an MFA enrollment bypass. RFC 9457 Problem Details with `type` = `urn:idmagic:error:mfa_enrollment_not_allowed`.")
model MfaEnrollmentNotAllowedError is ProblemDetails;


@doc("The requested combination of reset targets is not valid for the current MFA state of the target user. RFC 9457 Problem Details with `type` = `urn:idmagic:error:authenticator_reset_not_allowed`.")
model AuthenticatorResetNotAllowedError is ProblemDetails;


@doc("The administration shell bootstrap information of an authenticated browser session. It returns the CSRF token, the subject, the current realm, and the effective roles.")
model AccountContextResponse {
  csrf_token: string;
  id: string;
  preferred_username?: string;
  tenant_id?: string;
  realm?: string;
  roles?: string[];
}


@doc("The CSRF token bootstrap information an unauthenticated browser form uses together with a SameSite cookie.")
model BrowserCsrfContext {
  csrf_token: string;
}


@doc("The kinds of second factor (MFA), a subset of the RFC 8176 amr values chosen for this application. `totp` is RFC 6238, `webauthn` is WebAuthn Level 3, and `hwk` and `swk` are reserved for later.")
enum MfaFactorType {
  Totp: "totp",
  Webauthn: "webauthn",
  Hwk: "hwk",
  Swk: "swk",
}


@doc("One second factor (MFA) a user has enrolled. For `totp` the secret is stored as base32 and evaluated at authentication under RFC 6238 (T0=0, STEP=30s, `digits` digits, a window of plus or minus one step). The algorithm is SHA1 and the secret is generated at 20 bytes. `User.mfa_enrolled` being true means exactly that this set is non-empty.")
model MfaFactor {
  @doc("The `sub` of the owning user.")
  user_id: string;
  type: MfaFactorType;
  @doc("For `totp`, the base32 secret. It is left open so another factor can put its credential JSON in a field of its own. When persisted it is envelope-encrypted under the per-tenant DEK of DataKeys; the plaintext is never stored.")
  secret?: string;
  @doc("The display name the user gives it, `iPhone Authenticator` for instance.")
  label?: string;
  created_at: utcDateTime;
  last_used_at?: utcDateTime;
}


@doc("Why a LoginSession is not yet complete. `none` is an ordinary authenticated session, `challenge` is waiting on verification of an enrolled factor, and `enrollment` is waiting on the administrator-approved enrollment of a new one.")
enum LoginPendingPurpose {
  None: "None",
  Challenge: "Challenge",
  Enrollment: "Enrollment",
}


@doc("The short-lived, single-use enrollment approval an administrator issues to a user with no MFA enrolled. It is a trust anchor consumed after a successful password step so the session can move into enrollment pending; it does not exempt anyone from the MFA requirement of signing in. One that has expired, been revoked, or been consumed cannot be used, fail-closed.")
model MfaEnrollmentBypass {
  id: string;
  tenant_id: string;
  user_id: string;
  issued_by: string;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
  consumed_at?: utcDateTime;
  revoked_at?: utcDateTime;
  expired_at?: utcDateTime;
}


@doc("What an administrator's authenticator reset may target. `totp` and `webauthn` mean deleting every MfaFactor or WebAuthnCredential; `recovery_code` means deleting the whole set of backup recovery codes.")
enum AuthenticatorResetTarget {
  Totp: "totp",
  Webauthn: "webauthn",
  RecoveryCode: "recovery_code",
}


@doc("The set an administrator names for deletion when resetting a target user's authenticators.")
model AuthenticatorResetRequest {
  targets: AuthenticatorResetTarget[];
}


@doc("The result of an authenticator reset. `reenrollment_required` says that both TOTP and WebAuthn are now gone and a single-use bypass was issued automatically, so the next sign-in enters the enrollment-only pending flow. The bypass is present only when `reenrollment_required` is true.")
model AuthenticatorResetResponse {
  mfa_enrolled: boolean;
  reenrollment_required: boolean;
  bypass?: MfaEnrollmentBypass;
}


@doc("The browser login session. PostgreSQL is its single source of truth, so a session stays valid across a\nprocess restart. The cookie carries nothing but an opaque session id, while the `sub` and `auth_time` live in the\nserver-side store. The amr and acr stay here too and decide step-up at a later `/authorize`.\n`authentication_pending=true` is an intermediate state the same LoginSession moves into only when the sign-in policy\nor a step-up demands another factor; the mere fact that an MFA factor is enrolled does not make it pending. Moving\ninto a wait for another factor must never leave a separate, completed `pwd` session behind. The `/authorize`\ncompletion path treats a pending session as unauthenticated and directs it to the matching factor challenge.\nThe cookie is issued `http_only` with `same_site=Lax`. The `tenant_id` is always part of the authentication\nresolution query, so even if a session id were guessed or replayed across tenants the database layer would not return\na row of another tenant -- the isolation is fail-closed. Revocation sets `revoked_at` and `revoke_reason` as a\ntombstone and never deletes the row, so revoking again succeeds idempotently. Authentication resolution treats only\nrows with `revoked_at IS NULL AND expires_at > now` as valid.")
model LoginSession {
  id: string;
  tenant_id: string;
  user_id: string;
  @doc("The Unix second at which authentication succeeded. It is immutable as the session's start time, and the list is sorted by it, newest first.")
  auth_time: int64;
  @doc("The RFC 8176 methods by which authentication succeeded (RFC8176-AMR-VOCABULARY).")
  amr: string[];
  @doc("The authentication context class reference URN: either `urn:idmagic:acr:pwd`, when amr holds nothing beyond `pwd`, or `urn:idmagic:acr:mfa`, when amr holds one of otp, webauthn, hwk, swk, or rc (OpenID Connect Core section 2 and RFC 8176).")
  acr: string;
  @doc("Whether it is the intermediate state waiting on verification of another factor. False once complete.")
  authentication_pending: boolean = false;
  @doc("What `authentication_pending` is for. While it is `enrollment`, the session reaches no account, administration, or application resource, and only the enrollment API is permitted.")
  pending_purpose: LoginPendingPurpose = LoginPendingPurpose.None;
  @doc("The deadline for completing a `pending_purpose=enrollment`. Settling an enrollment after it is refused, and the session is not promoted to authenticated.")
  enrollment_deadline?: utcDateTime;
  @doc("The administrator-issued bypass consumed to move into Enrollment. Kept for audit correlation.")
  enrollment_bypass_id?: string;
  @doc("The Unix second at which a step-up reauthentication by password or MFA last succeeded; 0 when none has. It decides recency for high-sensitivity self-service operations.")
  step_up_at: int64 = 0;
  expires_at: utcDateTime;
  @doc("When the session was last used, in the sense of standing as an authentication result. It is touched on every authentication resolution, but a re-touch within LoginSessionTouchInterval of the last one is not persisted, so the value is coarse-grained and the idle timeout is approximated at that granularity.")
  last_seen_at: utcDateTime;
  @doc("When it was revoked; null while it is not. Revocation is a tombstone and never deletes the row.")
  revoked_at?: utcDateTime;
  @doc("The revocation reason, set together with `revoked_at`. Null while `revoked_at` is null.")
  revoke_reason?: SessionEndReason;
}


@doc("The authentication result the Authentication bounded context hands to the OAuth 2.0 / OpenID Connect bounded context. The latter never looks at passwords, the UserRepository, or the details of the session cookie.")
model AuthenticationContext {
  user_id: string;
  auth_time: int64;
  amr: string[];
  acr?: string;
  session_id?: string;
}


@doc("The projection of one MFA factor shown on the self-service security screen. No secret is\nreturned: only the kind, the label, and when it was created and last used. Today, TOTP alone.")
model AccountMfaFactor {
  type: MfaFactorType;
  label?: string;
  created_at: utcDateTime;
  last_used_at?: utcDateTime;
}


@doc("What `/api/account/security` returns, the security overview of the end-user account portal.\nIt gathers the password change time, whether MFA (TOTP and WebAuthn) is enrolled, and how many backup recovery codes\nremain.")
model AccountSecurityResponse {
  password_changed_at?: utcDateTime;
  totp_enrolled: boolean;
  factors: AccountMfaFactor[];
  webauthn_credentials: WebAuthnCredentialSummary[];
  recovery_codes: RecoveryCodeStatus;
}


@doc("What starting a TOTP enrollment returns (self-service). The client holds the secret until it confirms, and\n`otpauth_uri` is offered for a QR code or manual entry. The secret is not persisted until it comes back with the\nconfirmation (ConfirmTotpEnrollment).")
model TOTPEnrollmentStart {
  secret: string;
  otpauth_uri: string;
  account_name: string;
  issuer: string;
}


@doc("Settles a TOTP enrollment inside the administrator-approved login enrollment flow. Neither the secret nor the code reaches the audit trail or a log.")
model BrowserMfaEnrollmentConfirmRequest {
  secret: string;
  code: string;
  return_to?: string;
}


@doc("The input with which an administrator issues a single-use enrollment bypass to a target user.")
model MfaEnrollmentBypassIssueRequest {
  expires_in_seconds: int64 = 900;
}


@doc("How an enrollment bypass is shown to an administrator. There is no secret: the server consumes it automatically when the target user's password step succeeds.")
model MfaEnrollmentBypassResponse {
  bypass: MfaEnrollmentBypass;
}


@doc("The TOTP enrollment confirmation request (self-service). The secret is the one from the start,\nand the code is the current one generated from it, as proof of possession.")
model TOTPEnrollmentConfirmRequest {
  @minLength(1)
  secret: string;
  @minLength(1)
  code: string;
}


@doc("The MFA factor removal request (self-service). The code is the current TOTP code, as proof of\npossession. Removal requires the user themselves plus that proof.")
model MfaFactorRemoveRequest {
  @minLength(1)
  code: string;
}


@doc("Hints about how a WebAuthn authenticator connects (WebAuthn Level 3 section 5.8.4, AuthenticatorTransport).")
enum WebAuthnTransport {
  Usb: "usb",
  Nfc: "nfc",
  Ble: "ble",
  Internal: "internal",
  Hybrid: "hybrid",
}


@doc("One WebAuthn or passkey credential a user has enrolled. A user may hold several, so it is a\nseparate set from MfaFactor, uniquely identified by `credential_id`. `public_key` is the COSE public key and\n`sign_count` the signature counter used to detect a clone. It is a possession factor bound to an origin and an RP, and\nis used for phishing-resistant MFA and step-up. The ceremony resolves the RP ID and origin from the deployment\nconfiguration and uses `user_verification=preferred`, `attestation=none` (enterprise attestation is not strictly\nenforced), `resident_key=discouraged` (passwordless is out of scope), a 32-byte challenge, and a 120-second timeout\n(WebAuthn Level 3 sections 5, 7, and 13).")
model WebAuthnCredential {
  @doc("The credential ID the authenticator returns, base64url-encoded. Unique within the RP.")
  credential_id: string;
  @doc("The `sub` of the owning user.")
  user_id: string;
  @doc("The public key in COSE form, base64url-encoded.")
  public_key: string;
  @doc("The signature counter of the most recent assertion. A counter that goes backwards is refused as a suspected clone (`sign_count_regression=reject`).")
  sign_count: int64;
  @doc("The transport hints the authenticator declared: usb, nfc, ble, internal, or hybrid.")
  transports: string[];
  @doc("The AAGUID of the authenticator, base64url-encoded. A hint about the model.")
  aaguid?: string;
  @doc("The display name the user gives it, `MacBook Touch ID` for instance.")
  label?: string;
  @doc("Whether the credential can be synchronized across devices (the WebAuthn BE flag).")
  backup_eligible: boolean;
  @doc("Whether the credential is currently backed up (the WebAuthn BS flag).")
  backup_state: boolean;
  created_at: utcDateTime;
  last_used_at?: utcDateTime;
}


@doc("The projection of one WebAuthn credential shown on the self-service security screen.\nNeither the public key nor `sign_count` is returned: only what identification and display need.")
model WebAuthnCredentialSummary {
  credential_id: string;
  label?: string;
  transports: string[];
  created_at: utcDateTime;
  last_used_at?: utcDateTime;
}


@doc("One backup recovery code, for when TOTP or WebAuthn is lost. The plaintext is shown once at\ngeneration and only `code_hash` (SHA-256, as hex) is stored. A non-null `consumed_at` means it has been used and\ncannot be reused. A user's valid codes are treated as one set and replaced outright on regeneration\n(`regenerate_replaces_all`). Generation makes `count=10` codes, each `code_length=10` characters, drawn from the\nalphabet 23456789abcdefghijkmnpqrstuvwxyz -- a base32-like alphabet with the confusable characters removed -- and they\nare a backup alone, never a second factor on their own (NIST SP 800-63B section 5.1.2, Look-up Secrets).")
model RecoveryCode {
  user_id: string;
  @doc("The SHA-256 of the recovery code plaintext, as hex. The plaintext is not stored.")
  code_hash: string;
  generated_at: utcDateTime;
  consumed_at?: utcDateTime;
}


@doc("The state of the recovery codes shown on the self-service security screen. The codes themselves are not\nreturned: only when they were generated and how many there are in total and remaining.")
model RecoveryCodeStatus {
  generated_at?: utcDateTime;
  total: int64;
  remaining: int64;
}


@doc("One trusted device, the \"remember this device\" credential. It is issued per user, and only when the user\nexplicitly consents right after a genuine second factor (TOTP or WebAuthn) has succeeded during sign-in. The browser\nreceives `selector.verifier` in a realm-scoped HttpOnly, SameSite=Lax cookie, while the server stores nothing but the\nselector and `verifier_hash` (SHA-256, as hex). The selector fetches one row and the verifier's hash is compared in\nconstant time, so neither the cookie plaintext nor a device fingerprint is ever stored.\nThe verifier rotates on every use, so a stolen old cookie is invalidated by the next legitimate one.\nValidity is cut by both an absolute deadline (`expires_at`) and an idle one (`last_used_at` plus the idle window).")
model TrustedDevice {
  @doc("The device identifier. It is the key for audit and for the user revoking it.")
  id: string;
  @doc("The tenant it belongs to. The cookie is realm-scoped, so it never resolves across tenants.")
  tenant_id: string = "default";
  @doc("The `sub` of the owning user.")
  user_id: string;
  @doc("The non-secret lookup key that forms the first half of the cookie, base64url-encoded. It is unique and used for nothing but locating one row.")
  selector: string;
  @doc("The SHA-256 of the second half of the cookie, the verifier, as hex. The plaintext is not stored, and it rotates on every use.")
  verifier_hash: string;
  @doc("A display label derived from the User-Agent, naming nothing beyond the browser and OS family, `Chrome / macOS` for instance.\nNeither the raw User-Agent nor the IP is stored.")
  label?: string;
  created_at: utcDateTime;
  @doc("When it was last used to skip a second factor. The idle deadline is measured from here.")
  last_used_at: utcDateTime;
  @doc("`created_at` plus the tenant's `trusted_device_max_age_seconds`. The absolute deadline.")
  expires_at: utcDateTime;
  @doc("When it was revoked; null while it is not. Revocation is a tombstone and never deletes the row.")
  revoked_at?: utcDateTime;
  @doc("The revocation reason, set together with `revoked_at`. Null while `revoked_at` is null.")
  revoke_reason?: TrustedDeviceRevokeReason;
}


@doc("Why a trusted device was revoked. `self_revoke` is the user revoking it explicitly; `password_change` and\n`mfa_change` are bulk revocations following a credential change; `admin_revoke` is an administrator's authenticator\nreset; `account_disabled` is the account being disabled; and `session_revoke` is the user or an administrator revoking\nevery session.")
enum TrustedDeviceRevokeReason {
  self_revoke: "self_revoke",
  password_change: "password_change",
  mfa_change: "mfa_change",
  admin_revoke: "admin_revoke",
  account_disabled: "account_disabled",
  session_revoke: "session_revoke",
}


@doc("The projection of one trusted device shown on the self-service security screen. Neither the selector nor\n`verifier_hash` is returned: only what identification and display need. `current` says whether it is the device\nmatching the cookie the present request offered.")
model AccountTrustedDevice {
  id: string;
  current: boolean;
  label?: string;
  created_at: utcDateTime;
  last_used_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("What ListMyTrustedDevices returns: the user's own live trusted devices, by last use, newest first.\nIt is always empty when the tenant has trusted devices turned off.")
model AccountTrustedDeviceListResponse {
  devices: AccountTrustedDevice[];
  @doc("The tenant's `trusted_device_max_age_seconds`. Zero means the feature is off and the UI offers no way to remember a device.")
  max_age_seconds: int64;
}


@doc("The kinds of account security notification. Every message belongs to exactly one kind, and each kind\ndecides whether the user may stop receiving it. `new_device_sign_in` is a sign-in from a browser not seen before;\n`credential_change` is a password change; `mfa_change` is an authentication factor, recovery code, or trusted device\nbeing added or removed; `contact_change` is an email address change being requested and settled; `session_revoked` is\nthe user or an administrator explicitly revoking a session; and `impersonation` is an administrator beginning to act\nas the user.")
enum SecurityNotificationCategory {
  new_device_sign_in: "new_device_sign_in",
  credential_change: "credential_change",
  mfa_change: "mfa_change",
  contact_change: "contact_change",
  session_revoked: "session_revoked",
  impersonation: "impersonation",
}


@doc("The user's own security notification preferences. Having no row means exactly the same as everything\nbeing on, so no row is created until the first change. Only optional kinds may be listed in `disabled_categories`, and\nan update naming a mandatory kind is refused. The first thing an attacker silences after a takeover is the\nnotifications, and being able to silence them is no different from their not existing.")
model NotificationPreference {
  @doc("The `sub` of the owning user. There is no `tenant_id`, because `users.id` is globally unique.")
  user_id: string;
  @doc("The kinds the user has stopped receiving. A mandatory kind cannot be listed.")
  disabled_categories: SecurityNotificationCategory[];
  updated_at: utcDateTime;
}


@doc("One browser the user has signed in from before. It serves both to decide `new_device_sign_in` and to\ndeduplicate it. Every sign-in upserts on `(user_id, device_hash)`, and only when a row is newly created is the sign-in\nnotified as coming from a new device. `device_hash` is the SHA-256 of the User-Agent, as hex, salted with the tenant's\ncorrelation salt; neither the raw User-Agent nor the IP is stored.")
model KnownSignInDevice {
  @doc("The `sub` of the owning user.")
  user_id: string;
  @doc("`SaltedHash(tenant salt, User-Agent)`, as hex. It does not correlate across tenants.")
  device_hash: string;
  @doc("A display label derived from the User-Agent, naming nothing beyond the browser and OS family, `Chrome / macOS` for instance.")
  label?: string;
  first_seen_at: utcDateTime;
  last_seen_at: utcDateTime;
}


@doc("How one kind of security notification is presented to the user. A kind whose `mandatory` is true always has\n`enabled` true, and an update cannot set it to false.")
model AccountNotificationCategoryPreference {
  category: SecurityNotificationCategory;
  @doc("Whether it is a kind the user may not turn off.")
  mandatory: boolean;
  @doc("Whether the user currently receives notifications of this kind.")
  enabled: boolean;
}


@doc("What GetMyNotificationPreferences returns. Every kind comes back in the stable order the specification fixes,\nso the UI need not hold the list of kinds in its own code.")
model AccountNotificationPreferences {
  categories: AccountNotificationCategoryPreference[];
}


@doc("The input of UpdateMyNotificationPreferences. It names the kinds to turn off rather than the ones to turn on,\nso a kind added later is inherited as on by an existing setting. A request naming a mandatory kind is refused\nwith 400.")
model UpdateNotificationPreferencesRequest {
  @doc("The kinds to stop receiving. Only optional kinds may be listed.")
  disabled_categories: SecurityNotificationCategory[];
}


@doc("What StartWebAuthnRegistration returns: the PublicKeyCredentialCreationOptions to hand to\n`navigator.credentials.create`, as JSON and unchanged. The challenge is short-lived and bound to the\nserver-side session.")
model WebAuthnRegistrationOptions {
  @doc("The PublicKeyCredentialCreationOptions (WebAuthn Level 3).")
  public_key: Record<unknown>;
}


@doc("The input of FinishWebAuthnRegistration: the attestation `navigator.credentials.create` returned,\nsent back as JSON, optionally with a display label.")
model WebAuthnRegistrationRequest {
  @doc("The PublicKeyCredential holding an attestation response (WebAuthn Level 3).")
  attestation: Record<unknown>;
  label?: string;
}


@doc("What StartBrowserWebAuthn returns: the PublicKeyCredentialRequestOptions to hand to\n`navigator.credentials.get`, as JSON and unchanged. The challenge is bound to the pending login\nsession.")
model WebAuthnAssertionOptions {
  @doc("The PublicKeyCredentialRequestOptions (WebAuthn Level 3).")
  public_key: Record<unknown>;
}


@doc("The input of SubmitBrowserWebAuthn: the assertion `navigator.credentials.get` returned, sent back as\nJSON. The pending login session cookie resolves the `sub`, and `X-CSRF-Token` resolves CSRF.")
model BrowserWebAuthnRequest {
  @doc("The PublicKeyCredential holding an assertion response (WebAuthn Level 3).")
  assertion: Record<unknown>;
  return_to?: string;
  @doc("The user's explicit consent to remember this device and skip the second factor from now on. It is ignored\nwhen the tenant has trusted devices turned off.")
  remember_device?: boolean;
}


@doc("The input of SubmitBrowserRecoveryCode: it consumes one recovery code as the second factor.\nThe code never reaches the audit trail or a log.")
model BrowserRecoveryCodeRequest {
  code: string;
  return_to?: string;
}


@doc("The input of RemoveWebAuthnCredential: it names the credential to remove by its `credential_id`.")
model WebAuthnCredentialRemoveRequest {
  @minLength(1)
  credential_id: string;
}


@doc("What GenerateRecoveryCodes returns: the recovery codes in plaintext, once. Afterwards only the remaining\ncount (RecoveryCodeStatus) can be read; the codes are neither stored nor shown again.")
model RecoveryCodesResponse {
  codes: string[];
  generated_at: utcDateTime;
}


@doc("The projection of one entry in a user's own sign-in history. This slice derives from the existing audit\nevent store (UserAuthenticated), so the only attributes it holds are when it happened and by what means (amr).\nAdding the IP, UA, sessionId, and clientId comes later, once the event itself has been extended.")
model UserSignInActivity {
  occurred_at: utcDateTime;
  @doc("The technical values of RFC 8176 (RFC8176-AMR-VOCABULARY). The UI translates them into names meant for users: pwd is a password, otp an authenticator app, webauthn a passkey, rc a recovery code, and so on.")
  amr: string[];
}


@doc("What ListMySignInActivity and ListUserSignInActivity return: the most recent N entries, newest first.")
model UserSignInActivityResponse {
  activities: UserSignInActivity[];
}


@doc("Why a LoginSession ended. `self_revoke` and `admin_revoke` are the user or an administrator revoking it\nexplicitly; `logout`, `idle`, and `absolute` are automatic; and `password_change` and `mfa_change` follow a credential\nchange.")
enum SessionEndReason {
  logout: "logout",
  idle: "idle",
  absolute: "absolute",
  self_revoke: "self_revoke",
  admin_revoke: "admin_revoke",
  password_change: "password_change",
  mfa_change: "mfa_change",
  other: "other",
}


@doc("The projection of one entry in the self-service session list. It holds no secret: only the identifiers and\nthe authentication details. `current` says whether it is the caller's present session. `started_at` is the\n`auth_time` of the LoginSession and `expires_at` when it is due to expire.")
model AccountSession {
  id: string;
  current: boolean;
  amr: string[];
  acr: string;
  started_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("What ListMySessions returns: the user's own live sessions, by start time, newest first.")
model AccountSessionListResponse {
  sessions: AccountSession[];
}


@doc("The projection of one entry in the administrative session list. It holds no secret: only what is needed to\nidentify and audit a live session of the target user. Unlike the self-service AccountSession it has no\n`current` marker, because the caller and the target user are different people.")
model SessionRecord {
  id: string;
  user_id: string;
  amr: string[];
  acr: string;
  started_at: utcDateTime;
  last_seen_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("What ListSessions (administrative) returns: the live sessions of the target user, by start time, newest first.")
model SessionRecordListResponse {
  sessions: SessionRecord[];
}


@doc("The projection of one authentication event aggregate (bucket) for administrative display. It is the result of\naggregating individual sign-in failures into 5-minute windows during an attack: `window_start` is when the window\nbegan, `count` is the running total inside it, and `first_seen` and `last_seen` are the first and last times inside it.\n`key_hash` is SHA-256 as hex and holds no plaintext. It stays inside the tenant boundary.")
model AuthenticationEventBucket {
  @doc("failed_login | throttled | mfa_failed")
  kind: string;
  key_hash: string;
  window_start: utcDateTime;
  count: int64;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  first_seen: utcDateTime;
  last_seen: utcDateTime;
}


@doc("What ListAuthenticationEventBuckets returns, by `window_start` descending, newest window first.")
model AuthenticationEventBucketListResponse {
  buckets: AuthenticationEventBucket[];
}


@doc("The factors usable for a step-up reauthentication. `password` always; `totp` and `webauthn` once enrolled;\n`recovery_code` only while valid codes remain.")
enum StepUpMethod {
  password: "password",
  totp: "totp",
  webauthn: "webauthn",
  recovery_code: "recovery_code",
}


@doc("What StartStepUpAuthentication returns: the reauthentication methods available, which the UI offers as choices in\nits reauthentication modal.")
model StepUpStartResponse {
  methods: StepUpMethod[];
}


@doc("The input of CompleteStepUpAuthentication. Depending on `method` it uses the password or the code.\nThe material verified never reaches the audit trail or a log.")
model StepUpCompleteRequest {
  method: StepUpMethod;
  password?: string;
  code?: string;
}


@doc("The TOTP code a browser client posts to `/api/auth/totp`. The authorization request id is resolved from the HttpOnly transaction cookie and the CSRF from the `X-CSRF-Token` header. Other factors, WebAuthn among them, have a very different HTTP shape and so are defined independently, with their own endpoint and their own value object.")
model BrowserTotpRequest {
  @doc("The numeric code the authenticator app generated: six digits (RFC 6238 section 4 and RFC 4226 section 4).")
  code: string;
  @doc("Where to go inside the same realm when a direct sign-in to the administration console continues.")
  return_to?: string;
  @doc("The user's explicit consent to remember this device and skip the second factor from now on. It is ignored\nwhen the tenant has trusted devices turned off.")
  remember_device?: boolean;
}


@doc("The JSON an SPA posts to `/api/auth/change_password`. The authenticated session cookie resolves the `sub`, and CSRF is verified through the `X-CSRF-Token` header.")
model BrowserChangePasswordRequest {
  current_password: string;
  @minLength(12)
  @maxLength(128)
  new_password: string;
}


@doc("The JSON an SPA posts to `/api/auth/forgot_password`, an unauthenticated route. To prevent enumeration it always returns 204, whether or not the email exists.")
model BrowserForgotPasswordRequest {
  email: string;
}


@doc("The JSON an SPA posts to `/api/auth/reset_password`. It takes the single-use token from the email link together with the new password.")
model BrowserResetPasswordRequest {
  token: string;
  @minLength(12)
  @maxLength(128)
  new_password: string;
}


@doc("The single-use token for a password reset.\nThe token is 32 random bytes as base64url, and only its SHA-256 hash is stored.\nIt lives 1800 seconds (30 minutes) from issuance and is consumed once; once `consumed_at` is set it cannot be reused.\nExpired and consumed rows are garbage-collected, and an INSERT also does a light sweep that invalidates any unconsumed\ntoken of the same `sub`.")
model PasswordResetTokenRecord {
  user_id: string;
  token_hash: string;
  created_at: utcDateTime;
  @doc("`created_at` plus 1800 seconds (30 minutes).")
  expires_at: utcDateTime;
  @doc("Non-null means it has been used and cannot be reused.")
  consumed_at?: utcDateTime;
}


@doc("A ResourceOwner authenticated successfully. `amr` lists the means actually used (OpenID Connect Core section 2).")
model UserAuthenticated {
  occurredAt: utcDateTime;
  @doc("The tenant the authentication happened in. The audit view narrows by it.")
  tenantId: string;
  userId: string;
  amr?: string[];
  @doc("The id of the LoginSession the authentication started.")
  sessionId?: string;
  @doc("The OAuth client that started the authentication.")
  clientId?: string;
  acr?: string;
  @doc("The client IP. It is stored in plaintext for as long as the audit event is retained.")
  ip?: string;
  @doc("The client User-Agent. It is stored in plaintext for as long as the audit event is retained.")
  userAgent?: string;
  @doc("The country code from GeoIP; empty when GeoIP is not wired up.")
  countryCode?: string;
  @doc("The device fingerprint. It is stored in plaintext for as long as the audit event is retained.")
  deviceFingerprint?: string;
  @doc("The risk score. This work item only reserves the field; the value is null.")
  riskScore?: int64;
}


@doc("A ResourceOwner failed to authenticate. It is an input event for rate limiting and fraud detection.\nAttempts against a username that does not exist are recorded too, so there is no `sub`; the username is kept in\nplaintext as the `actor.username` correlation key for audit search. The username and the other client attributes are\nnot hashed, and are kept for the same retention as the other failure events, 30 days by default.")
model AuthenticationFailed {
  occurredAt: utcDateTime;
  @doc("The tenant the authentication was attempted in, taken from the request path.")
  tenantId: string;
  username?: string;
  reason?: string;
  sessionId?: string;
  clientId?: string;
  @doc("The client IP. It is stored in plaintext for as long as the audit event is retained.")
  ip?: string;
  @doc("The client User-Agent. It is stored in plaintext for as long as the audit event is retained.")
  userAgent?: string;
  countryCode?: string;
  @doc("The device fingerprint. It is stored in plaintext for as long as the audit event is retained.")
  deviceFingerprint?: string;
  riskScore?: int64;
}


@doc("A ResourceOwner changed their own password at `/api/auth/change_password`. It is the starting point of the change-password audit trail and of the notification.")
model PasswordChanged {
  occurredAt: utcDateTime;
  @doc("The tenant the target user belongs to.")
  tenantId: string;
  userId: string;
}


@doc("A sign-in attempt reached a throttling threshold -- per account (10 attempts in a 900-second window, 900-second\nlockout) or per IP (30 attempts in a 900-second window, 900-second lockout) -- and was locked out.\nIt is an input event for credential stuffing and brute-force detection in a SIEM. `keyHash` is the SHA-256 of the\nusername in lowercase, or of the IP, so no plaintext reaches the audit log.")
model LoginThrottled {
  occurredAt: utcDateTime;
  @doc("The tenant the throttling happened in, taken from the request path.")
  tenantId: string;
  @doc("Either `account` or `ip`.")
  kind: string;
  @doc("The SHA-256 of the username in lowercase, or of the IP, as hex.")
  keyHash: string;
  retryAfterSeconds: int64;
}


@doc("Records that, during a flood of credential attempts, individual AuthenticationFailed events were aggregated\ninto a bucket keyed by `(tenantId, kind, keyHash, 5-minute window)` rather than written one row at a time. Only the\nfirst event of a window is emitted, and later ones add to the count in the bucket store. The `count` in this payload\nis therefore the value at emission, and the real number is observed through ListAuthenticationEventBuckets.\n`keyHash` is SHA-256 as hex, so no plaintext reaches the audit log, following LoginThrottled.")
model AuthenticationEventAggregated {
  occurredAt: utcDateTime;
  @doc("The tenant the aggregation happened in.")
  tenantId: string;
  @doc("failed_login | throttled | mfa_failed")
  kind: string;
  @doc("The composite key `kind:keyHash:windowStartUnix`.")
  bucketKey: string;
  @doc("The SHA-256 of the username in lowercase, or of the IP, as hex.")
  keyHash: string;
  @doc("How many fell in the window at emission; the real number keeps rising in the bucket store.")
  count: int64;
  firstSeen: utcDateTime;
  lastSeen: utcDateTime;
  @doc("The `keyHash` values seen most often in the window, at most five.")
  topKeys: string[];
}


@doc("One step of a multi-step authentication flow -- password, totp, webauthn, and so on -- succeeded. It is an input to settling the acr.")
model AuthenticationStepCompleted {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  @doc("`password`, `totp`, `webauthn`, `recovery_code`, and the like.")
  step: string;
  sessionId?: string;
}


@doc("One step of a multi-step authentication flow failed.")
model AuthenticationStepFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  step: string;
  reason?: string;
}


@doc("An MFA challenge was presented: TOTP, WebAuthn, or another.")
model MfaChallengeIssued {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  factorType: MfaFactorType;
  sessionId?: string;
}


@doc("An MFA challenge succeeded.")
model MfaChallengeSucceeded {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  factorType: MfaFactorType;
  sessionId?: string;
}


@doc("An MFA challenge failed. It is also an input to the bucket aggregation with `kind=mfa_failed`.")
model MfaChallengeFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  userId?: string;
  factorType: MfaFactorType;
  sessionId?: string;
}


@doc("Under a policy that requires MFA, an unenrolled user consumed a valid administrator approval and entered the enrollment-only pending flow. Password authentication alone does not establish an ordinary session.")
model MfaEnrollmentRequired {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  sessionId: string;
  bypassId: string;
  deadline: utcDateTime;
}


@doc("Inside the login enrollment flow, a factor was enrolled and proof of possession completed, and the same LoginSession was promoted to MFA-satisfied.")
model MfaEnrollmentCompleted {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  sessionId: string;
  factorType: MfaFactorType;
}


@doc("An administrator issued a short-lived, single-use enrollment bypass to a user with no MFA enrolled.")
model MfaEnrollmentBypassIssued {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  userId: string;
  bypassId: string;
  expiresAt: utcDateTime;
}


@doc("A target user who had authenticated with a password consumed their enrollment bypass, once, and entered the enrollment-only pending flow.")
model MfaEnrollmentBypassConsumed {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  bypassId: string;
  sessionId: string;
}


@doc("An administrator revoked an unconsumed enrollment bypass.")
model MfaEnrollmentBypassRevoked {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  userId: string;
  bypassId: string;
}


@doc("An unconsumed enrollment bypass was found to be past its deadline and unusable. It is emitted the first time the expiry is observed.")
model MfaEnrollmentBypassExpired {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  bypassId: string;
}


@doc("An administrator asked for an authenticator reset on a target user. It is emitted before anything is deleted, so what was targeted at the time of the request stays in the audit trail.")
model AuthenticatorResetRequested {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  userId: string;
  targets: AuthenticatorResetTarget[];
}


@doc("An administrator's authenticator reset completed. It records the factors deleted and the recomputed `mfa_enrolled`. When both TOTP and WebAuthn are gone, `reenrollment_required` is true and a separate MfaEnrollmentBypassIssued is emitted.")
model AuthenticatorResetCompleted {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  userId: string;
  targets: AuthenticatorResetTarget[];
  mfaEnrolled: boolean;
  reenrollmentRequired: boolean;
}


@doc("A backup recovery code was consumed, recovering access after TOTP or WebAuthn was lost.")
model BackupCodeConsumed {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  remainingCount?: int64;
}


@doc("A LoginSession began. It is the starting point of the session list and of an investigation.")
model SessionStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  sessionId: string;
  amr?: string[];
  acr?: string;
  @doc("The client IP at the start of the session. It is stored in plaintext for as long as the audit event is retained.")
  ip?: string;
  @doc("The User-Agent at the start of the session. It is stored in plaintext for as long as the audit event is retained.")
  userAgent?: string;
}


@doc("The validity of a LoginSession was extended, because it stayed active.")
model SessionRefreshed {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  sessionId: string;
}


@doc("A federated sign-in through an external IdP succeeded.")
model FederatedAuthenticated {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  @doc("The external IdP identifier, `google`, `github`, or `saml:acme` for instance.")
  provider: string;
  sessionId?: string;
}


@doc("A tenant administrator created an upstream connection as a draft. It carries no secret, token, or certificate body.")
model IdentityProviderConnectionCreated {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
  protocol: IdentityProviderProtocol;
}


@doc("A connection that passed metadata and trust validation became usable for login routing.")
model IdentityProviderConnectionActivated {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
}


@doc("A connection was taken out of login routing.")
model IdentityProviderConnectionDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
}


@doc("A link between a verified external subject and a local User was created. It carries no external token or assertion and no PII claim.")
model FederatedIdentityLinked {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
  userId: string;
  @doc("explicit | verified_email | jit")
  linkingMethod: string;
}


@doc("A ResourceOwner who had stepped up unlinked an external identity.")
model FederatedIdentityUnlinked {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
  userId: string;
}


@doc("The trust, correlation, or policy check of an upstream callback refused it fail-closed. It carries no token, assertion, state, or nonce.")
model FederatedLoginRejected {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
  reason: string;
}


@doc("An administrator began a session acting as a user (impersonation). It is an audit event that presumes the user is\nnotified, and is not subject to shortened retention.")
model SessionImpersonationStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  @doc("The `sub` of the administrator who impersonated.")
  actorUserId: string;
  @doc("The `sub` of the target user.")
  targetUserId: string;
  sessionId: string;
}


@doc("An impersonation session ended. It is not subject to shortened retention.")
model SessionImpersonationEnded {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
  sessionId: string;
}


@doc("A password reset link was requested. To prevent enumeration it records only that\nthe request arrived; whether a link was actually issued -- that is, whether a user with a verified\nemail existed -- is observed separately through the EmailSent event.\n`emailHash` is a SHA-256, so no plaintext reaches the audit log.")
model PasswordResetRequested {
  occurredAt: utcDateTime;
  @doc("The tenant the reset was requested in, taken from the request path.")
  tenantId: string;
  @doc("The SHA-256 of the submitted email, as hex.")
  emailHash: string;
}


@doc("A user enrolled an MFA factor through self-service; today, TOTP. The secret never reaches\nthe audit trail -- only the kind does. It sets `User.mfa_enrolled` to true.")
model MfaFactorEnrolled {
  occurredAt: utcDateTime;
  @doc("The tenant the target user belongs to.")
  tenantId: string;
  userId: string;
  factorType: MfaFactorType;
}


@doc("A user removed an MFA factor through self-service. Removal is accompanied by proof of\npossession through a valid TOTP code. It returns `User.mfa_enrolled` to false.")
model MfaFactorRemoved {
  occurredAt: utcDateTime;
  @doc("The tenant the target user belongs to.")
  tenantId: string;
  userId: string;
  factorType: MfaFactorType;
}


@doc("A user enrolled a WebAuthn or passkey credential through self-service. Neither the public key nor\nthe `credential_id` reaches the audit trail -- only the kind does. The first one sets `User.mfa_enrolled` to true.")
model WebAuthnCredentialRegistered {
  occurredAt: utcDateTime;
  @doc("The tenant the target user belongs to.")
  tenantId: string;
  userId: string;
}


@doc("A user removed a WebAuthn or passkey credential through self-service, having stepped up.\nWhen no second factor is left, `User.mfa_enrolled` returns to false.")
model WebAuthnCredentialRemoved {
  occurredAt: utcDateTime;
  @doc("The tenant the target user belongs to.")
  tenantId: string;
  userId: string;
}


@doc("A user generated or regenerated backup recovery codes, having stepped up. Neither the plaintext nor a hash\nreaches the audit trail -- only how many were issued. A regeneration replaces the existing set outright.")
model RecoveryCodesGenerated {
  occurredAt: utcDateTime;
  @doc("The tenant the target user belongs to.")
  tenantId: string;
  userId: string;
  @doc("How many recovery codes were issued.")
  count: int64;
}


@doc("A user explicitly invalidated their backup recovery codes, having stepped up. No recovery code works as a\nsecond factor afterwards.")
model RecoveryCodesRevoked {
  occurredAt: utcDateTime;
  @doc("The tenant the target user belongs to.")
  tenantId: string;
  userId: string;
}


@doc("Right after a genuine second factor succeeded during sign-in, the user explicitly consented and the device was\nremembered. Neither the cookie plaintext, nor the User-Agent, nor the IP is in the payload: only the device identifier\nand the timestamps revocation decisions need.")
model TrustedDeviceRegistered {
  occurredAt: utcDateTime;
  @doc("The tenant the target user belongs to.")
  tenantId: string;
  userId: string;
  deviceId: string;
  @doc("The second factor that occasioned the remembering: `otp` or `webauthn`.")
  factor: string;
  expiresAt: utcDateTime;
}


@doc("A trusted device was revoked. Besides the user revoking it explicitly, it is emitted for the bulk revocations\nthat follow a credential change, an administrator's authenticator reset, an account being disabled, and every session\nbeing revoked.")
model TrustedDeviceRevoked {
  occurredAt: utcDateTime;
  @doc("The tenant the target user belongs to.")
  tenantId: string;
  userId: string;
  deviceId: string;
  reason: TrustedDeviceRevokeReason;
}


@doc("A LoginSession ended. When the user or an administrator revoked it explicitly, `actorUserId` is whoever acted\nand `reason` is `self_revoke` or `admin_revoke`. The `sessionId` equals the OIDC `sid`, and the OAuth2 bounded context\nuses it as the key to revoke the refresh token families whose `RefreshTokenRecord.sid` matches and to notify the\nregistered relying parties through front-channel and back-channel logout.\nAccess tokens themselves are not revoked immediately by this event; their short TTL takes care of them.")
model SessionEnded {
  occurredAt: utcDateTime;
  @doc("The tenant the target user belongs to.")
  tenantId: string;
  userId: string;
  sessionId: string;
  @doc("The `sub` of whoever performed the revocation; the user themselves for a self-revoke.")
  actorUserId: string;
  reason: SessionEndReason;
}


@doc("A user began a step-up reauthentication ahead of a high-sensitivity operation.\nIt is the request to be shown the available factors; the reauthentication has not succeeded yet.")
model StepUpRequested {
  occurredAt: utcDateTime;
  @doc("The tenant the user belongs to.")
  tenantId: string;
  userId: string;
  sessionId: string;
}


@doc("One account security notification was sent, or an attempt to send it failed. It exists to keep what the user\nwas told in the audit trail, and carries neither the body nor the recipient address in its payload. The notification\ndispatcher emits it itself, and because this event type is not in the catalogue no further notification cascades from it.")
model AccountSecurityNotificationSent {
  occurredAt: utcDateTime;
  @doc("The tenant the target user belongs to.")
  tenantId: string;
  @doc("The `sub` of the user receiving the notification. Under impersonation it is the target user, not the administrator who acted.")
  userId: string;
  category: SecurityNotificationCategory;
  @doc("The type name of the event that occasioned the notification.")
  triggerEventType: string;
  @doc("Whether the EmailSender accepted it. Delivery is best-effort, so the original operation stands even when this is false.")
  delivered: boolean;
}


@doc("A step-up reauthentication succeeded. `method` is the factor used, `password` or `totp`.\nThe material verified -- the password or the code -- is not in the payload.")
model StepUpCompleted {
  occurredAt: utcDateTime;
  @doc("The tenant the user belongs to.")
  tenantId: string;
  userId: string;
  sessionId: string;
  @doc("password | totp")
  method: string;
}


@doc("The request needs an authenticated session and does not have one. RFC 9457 Problem Details with `type` = `urn:idmagic:error:authentication_required`. Distinct from AccessDeniedError, which means the session exists but the policy denies the operation.")
model AuthenticationRequiredError is ProblemDetails;


@doc("The login session named by the request does not exist, or has already ended. RFC 9457 Problem Details with `type` = `urn:idmagic:error:session_not_found`.")
model SessionNotFoundError is ProblemDetails;


@doc("The JSON request creating or updating an upstream connection. It carries only what the caller may set; the id, the tenant, the status, the timestamps, and `client_secret_configured` all stay with the server, and a new connection starts Disabled.")
model IdentityProviderConnectionRequest {
  @doc("The name shown to the user on the sign-in screen.")
  display_name: string;
  protocol: IdentityProviderProtocol;
  @doc("For OIDC, the issuer; validated at configuration time.")
  issuer?: string;
  @doc("For OIDC, the client id registered upstream.")
  client_id?: string;
  @doc("The client secret itself, write-only. Only its ciphertext is stored, and no response ever carries it back.")
  secret_reference?: string;
  authorization_endpoint?: url;
  token_endpoint?: url;
  jwks_uri?: url;
  saml_sso_url?: url;
  saml_entity_id?: string;
  saml_signing_certificates?: string[];
  claim_mapping: ClaimMapping;
  linking_policy: AccountLinkingPolicy;
  @doc("Whether a local User is provisioned just in time when no link exists.")
  jit_provisioning: boolean;
  @doc("The email domains permitted to route to this connection. Empty means no restriction.")
  allowed_email_domains?: string[];
}

}
