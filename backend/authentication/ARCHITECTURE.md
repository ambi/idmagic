---
context: authentication
updated_at: 2026-08-09
---

# Architecture: authentication

## Overview

The `Authentication` context owns credential verification, MFA, login sessions, step-up, recovery,
and login-time federation. Independent capabilities are vertical feature slices with their own domain,
ports, use cases, and adapters; `module.go` remains the context composition boundary.

## Inbound identity federation

`federation/` is the identity-broker feature slice. It owns tenant-scoped upstream OIDC and SAML
connections, external-subject links, single-use login attempts, protocol validation, linking policy,
and the handoff to an IdMagic login session. Downstream SAML IdP and WS-Federation issuance remain in
their protocol contexts. Identity Management exposes credential-less user creation for JIT, but does
not own the login-time correlation policy. Authentication owns this login-time correlation and
orchestration directly rather than delegating it to a separate sourcing context, since the broker's
linking and JIT decisions are tightly coupled to session issuance.

The broker first resolves an immutable external subject through a `FederatedIdentity`. Verified-email
linking is available only under an explicit connection policy, a verified upstream claim, and a unique
tenant-local match. JIT is separately enabled per connection and may be narrowed by an email-domain
allowlist. Explicit link and unlink operations require recent step-up, and the last usable sign-in
method cannot be removed.

Protocol adapters treat every upstream document as untrusted. OIDC uses saved HTTPS discovery
endpoints, Authorization Code with PKCE, state, nonce, issuer/audience/time checks, and a constrained
JWK algorithm. SAML uses a correlated AuthnRequest and validates XML signature, issuer, destination,
audience, subject confirmation, time, and replay. Unsolicited IdP-initiated responses and encrypted
assertions are outside the initial SAML adapter.

Client secrets are represented only by a `secret_reference`; the initial runtime resolver accepts
`env:NAME` references. Neither raw secrets nor upstream tokens/assertions are persisted or returned.
Public provider discovery exposes only active provider identifiers, display names, and protocols.

After callback validation the broker creates a normal Authentication session with `federated` in AMR.
OAuth authorization resumes through `/authorize/resume`, so application policy, required actions,
consent, and code issuance remain owned by OAuth2 rather than being duplicated in the broker.

## Persistence

`authentication_sessions` is the single source of truth for a `LoginSession`.
`tenant_id` is kept alongside the user-derived path as an exception to the usual
[`tenant_id` retention classes](../../ARCHITECTURE.md#2-tenant_id-retention-classes): a session id is an
opaque browser-cookie value re-verified on every request, a fail-closed boundary where per-tenant lookup
matters the same way it does for `refresh_tokens.sid`. Revocation sets
`revoked_at`/`revoke_reason` rather than deleting the row, so physical removal stays a housekeeping
concern independent of revoke state, and a repeated revoke is a safe no-op. Its two indexes serve, in
order: keyset pagination over one user's non-revoked sessions ordered by `auth_time DESC`, and the
housekeeping batch's `expires_at` cleanup scan.

A session's validity is a fixed 1-hour TTL set once at creation (`SessionTTLSeconds`), not a sliding
window: `expires_at` is never extended by use. Only `last_seen_at` moves, coalesced to a 5-minute
floor, because bumping `expires_at` on every touch would add write amplification, VACUUM pressure, and
lock contention that this coarse-touch/absolute-expiry combination avoids. The 90-day
retention window is a separate, later concern: it bounds how long an already-expired row is
kept for investigation before the housekeeping batch physically deletes it, not how long a session
stays valid.

`mfa_factors.secret` is the legacy plaintext TOTP seed column, kept only so existing rows remain
readable (dual-read); new writes populate `secret_key_version`/`secret_ciphertext` and leave `secret`
`NULL`. A pending backfill migrates the remaining plaintext rows, after which `secret` is dropped.

`webauthn_credentials` keys on `credential_id` because one user can register several;
it stays a separate table from `mfa_factors` for that reason. `public_key` holds the COSE public key
(base64url). `recovery_codes` never stores the plaintext code, only `code_hash`
(SHA-256 hex); a non-`NULL` `consumed_at` means the code is used and cannot be replayed, and
regeneration replaces a user's whole set at once. `webauthn_sessions` is the WebAuthn ceremony
challenge store; `GetDel` is `DELETE ... WHERE expires_at > now() RETURNING data`.

`tenant_correlation_salts` is a per-tenant secret used to compute the correlated hash
(`SaltedHash`) of usernames/IPs and the throttle/bucket `keyHash`, so correlation is never aggregated
across tenants; it is generated on first use rather than provisioned up front.

`login_throttle_counters` is `LOGGED` because losing it on failover would
be a defense-in-depth regression, and uses `fillfactor = 80` to leave header room for the frequent
same-row `UPDATE`s (HOT updates) a counter takes. `identifier_hash` is a SHA-256 hex digest, so no
plaintext username or IP is retained.

## Password lifecycle

Password rules follow NIST SP 800-63B-4 §3.1.1.2's move away from composition rules and forced
rotation: length (`min_length=12`, `max_length=128`), a case-insensitive check against the user's own
identifiers (username/email/local-part, skipped under 4 characters to avoid false positives), and a
bundled common-password dictionary are enforced, but mixed-character-class and periodic-rotation
requirements are not — both are explicitly discouraged by NIST and tend to push users toward
predictable patterns rather than raising effective entropy.

`change-password` also rejects reuse of the last 5 password hashes (`history_depth=5`), stored in
`password_histories` as the same Argon2id PHC string as `password_hash` — a separate encoding would not
raise the attack cost and would just add dual maintenance. History is written on both registration and
change-password (so a same-as-initial change is still caught) but checked only on change-password, since
a first registration has nothing to compare against.

A `BreachedPasswordChecker` port layers external knowledge (HIBP Range API, k-anonymity: only a SHA-1
prefix leaves the server) on top of the bundled offline dictionary. It fails open — an HIBP timeout or
outage returns `breached=false` rather than blocking the change — because an optional defense-in-depth
layer should not be able to take credential changes down; failures are still recorded for audit. The
default adapter is a no-op so in-memory/dev startup carries no external dependency, and
`breached_password_check_enabled=false` by default.

Forgot-password issues a single-use, 32-byte random token stored in `password_reset_tokens` only as its
SHA-256 hash (`ttl=1800s`), and redemption runs the same validation/history/breach pipeline as
change-password. Every response on this path is uniform (`204` whether the email exists, is unverified,
or was mistyped) so the recovery flow cannot become a username-enumeration oracle, and email delivery is
best-effort — a send failure is never surfaced to the caller, only logged — so an SMTP outage cannot be
probed from the unauthenticated side.

Delivery itself goes through an `EmailSender` port; the production adapter speaks SMTP only (STARTTLS by
default, PLAIN auth permitted only under TLS) rather than adding an HTTP SDK per provider, because SMTP
alone already reaches every major transactional-email provider and each additional REST SDK would
multiply dependency, credential-shape, and error-format surface for no port-level benefit. Outgoing
content is normalized before sending (CRLF/NUL stripped, HTML body escaped, subject RFC 2047-encoded) so
a user-controlled string cannot inject SMTP headers or raw HTML.

## Login throttling

Login attempts are throttled on two independent axes — per-account (10 failures/900s → 900s lock) and
per-IP (30 failures/900s → 900s lock) — because either axis alone misses an attack shape the other
catches: per-account stops a dictionary attack against one victim, per-IP stops credential stuffing
spread across many accounts from one source. Crossing either threshold is enough to return 429; counters
are cluster-wide, backed by the shared `login_throttle_counters` table in PostgreSQL, so a lock holds
across replicas.

Counters key on a SHA-256 hash of the identifier rather than plaintext, and the per-account counter is
incremented *before* checking whether the account exists — using a fixed sentinel hash for the
password-verify step when it does not — because otherwise timing (an Argon2 verify happening or not) or
the shape of the 429 response would leak which usernames exist. A successful login clears the account
counter but deliberately leaves the per-IP counter alone: one successful login from a shared office/NAT
IP does not make the rest of that IP's traffic trustworthy, and IP counters age out on their own via the
window. If the shared throttle store is unreachable, login fails closed rather than allowing unthrottled
attempts. Client IP is read from the direct peer by default; `X-Forwarded-For` is honored only under an
explicit `TRUSTED_FORWARDED_HOPS` opt-in, since trusting it unconditionally would let an attacker spoof
around the per-IP axis. Each lockout emits `LoginThrottled` with the same tenant-salted `keyHash` used
by bucket aggregation below, not plaintext.

## Authentication event logging

Authentication events are kept in two tracks so an attack traffic spike cannot take down the audit store
or bury genuine signal: `authentication_events` holds one row per individual action (success, failure,
MFA, federation, session start), while `authentication_event_buckets` folds an ongoing burst from the
same `(tenant, kind, keyHash)` into a single 5-minute-window row whose `count` increments in place
instead of emitting new rows. Once a per-account or per-IP actor crosses the login-throttle lockout
threshold above, its subsequent failures stop emitting individual `AuthenticationFailed` events and roll
into the bucket instead, sharing the same tenant-salted `keyHash` as the throttle counter so audit and
throttle correlate without exposing plaintext.

Bucket thresholds (5-minute window; 10 failures/account, 50/IP, 1000/tenant by default, tenant
overridable) balance keeping ordinary typos visible as individual events against letting a genuine flood
keep writing rows forever: set too low, normal mistakes vanish into a bucket; too high, the flood never
collapses. Each window emits exactly one `AuthenticationEventAggregated` admin event — later hits in the
same window only bump `count` — and an admin can drill from that row into up to 10 individual failure
samples for the same key. Impersonation events (`SessionImpersonationStarted`/`Ended`) are excluded from
both bucket collapsing and retention shortening, since they document an admin acting as a user and are
kept intact for that user's protection.

Retention is asymmetric by kind and enforced by an idempotent, batched hourly sweep on `occurred_at`:
successes 365 days (long enough to spot an unusual login pattern), individual failure rows 30 days,
bucket rows and session/MFA-challenge rows 90 days. Tenants may shorten or lengthen within a
`max_retention_days` global cap; impersonation events cannot be shortened below the cap. Partitioning and
cold storage were deferred — search performance instead comes from the sweep keeping row counts bounded,
a `(tenant_id, occurred_at)` index, and a query `limit`.

PII handling was reworked from an earlier hash/truncate-everything scheme once tenant-salt hashing was
found to add real wiring cost for little benefit — events tied to a confirmed account
(`UserAuthenticated`, OAuth2-flow events) already correlate by `user_id` and never needed a username
hash, and admin search instead resolves a searched username to `user_id` on the fly. The current state:
username, IP, User-Agent, and device fingerprint are stored in plaintext for the event's normal retention
window (`AuthenticationFailed` keeps plaintext username for its full 30-day retention rather than a
shorter hash-only window); location remains reduced to country code only. The one piece of the earlier
scheme that survives is the tenant-salted `keyHash` used by `LoginThrottled` and bucket aggregation —
that hash identifies a throttle/bucket key, not a stored audit PII field.

## Account portal trust boundary and step-up

Account portal APIs (`/api/account/*`) act only on the authenticated session's own `actor.sub`;
URL/body/query-supplied `sub` or `tenant_id` are never trusted, so cross-user and cross-tenant access
cannot arise structurally. This is a separate contract from the admin API (`/api/auth/account`, which
includes roles) — the portal's own summary endpoint (`/api/account/summary`) deliberately omits roles so
the self-service surface cannot leak admin metadata even by accident. Self-service can change the user's
own display name, `editable_by_user=true` attributes, and password; roles, status, org attributes, and
`editable_by_user=false` attributes stay admin-only, and `required_actions` can only be viewed — never
granted or revoked — by the user, aside from actions that clear themselves as a side effect of the user's
own action (e.g. `update_password` clearing on a successful password change). The portal UI is a distinct
shell that never surfaces admin navigation, even to a user who happens to hold an admin role.

CSRF and same-origin checks protect every self mutation, but they don't help once the session cookie
itself is stolen — an attacker holding it can still take the account over outright (change the password,
drop MFA, redirect where notifications go). High-sensitivity self-service operations therefore require a
recent re-authentication ("step-up") on top of CSRF: `ChangePassword`, `RemoveTotpFactor`,
`RequestEmailChange`, and `RevokeMyOtherSessions` are annotated `step_up: required` in SCL, with a test
(`TestStepUpAnnotatedInterfacesMatchGatedHandlers`) keeping the annotation and the gated handlers from
drifting apart. A session counts as stepped-up if `max(session.auth_time, session.step_up_at)` is within
`StepUpRecencySeconds` (5 minutes) — so a session is stepped-up immediately after login, matching the
re-auth pattern users already expect from Google/Okta-style prompts. A gate failure returns
`403 step_up_required`, not `401`, since the session is authenticated and has simply not proven recency
for this specific action; the UI reissues the original request once `POST /api/account/step_up/complete`
succeeds. Recency is stored as `step_up_at` on the `LoginSession` row itself so it cannot follow the
cookie to a different device.

## WebAuthn/passkey MFA and recovery codes

TOTP keeps the standard RFC 6238 parameters (SHA1, 30s step, 6 digits, ±1 step window, 160-bit secret).
WebAuthn credentials live in their own table (`webauthn_credentials`, keyed on `credential_id`) rather
than being squeezed into `mfa_factors`, because that table's `(user_id, type)` identity allows only one
factor per type per user, while WebAuthn's whole value is registering several authenticators per
account. Ceremony logic (CBOR/COSE parsing, signature verification) is delegated entirely to
`go-webauthn/webauthn` rather than reimplemented, since a self-written attestation/assertion verifier is
exactly the kind of code where a subtle bug is a security bypass. Registration and authentication
challenges reuse the existing ephemeral `SessionStore` (keyed by `sub` for registration, by
pending-login-session id for authentication) instead of a new store, since the challenge is already a
short-lived server-side value with the same lifecycle shape as other session data.

RP ID and allowed origins come from deployment config (`WEBAUTHN_RP_ID`/`WEBAUTHN_RP_ORIGINS`), validated
at startup and re-checked on every ceremony; attestation is `none` (privacy over device-model
enforcement), user verification `preferred`, resident keys `discouraged` (`challenge_bytes=32`,
`timeout_seconds=120`) — this stage adds WebAuthn as a phishing-resistant second factor alongside
password, not as a passwordless/discoverable-credential flow, which stays explicitly out of scope. A
returned `sign_count` at or below the stored value (0-to-0 excepted) is treated as evidence of a cloned
authenticator and the assertion is rejected outright, since a genuine authenticator's counter only moves
forward.

Recovery codes (hash-only via SHA-256, single-use via `consumed_at`, regeneration replaces the whole set
— 10 codes of 10 characters from a low-ambiguity alphabet) exist purely as a backup for a lost
TOTP/WebAuthn factor and are deliberately **not** counted toward `User.mfa_enrolled` — treating a backup
code as a standalone second factor would let a user rely on it as their only MFA, defeating the point of
having a backup. `mfa_enrolled` is instead derived from "at least one TOTP factor or WebAuthn credential
exists," recomputed whenever either is removed; generating, regenerating, or revoking recovery codes
requires step-up. On successful second-factor verification, `acr` rises to `urn:idmagic:acr:mfa` and
`amr` gains `webauthn` (an RFC 8176 registered value) or `rc` for recovery-code use — `rc` is this
application's own non-IANA value, called out explicitly since it is not a registered AMR.

## MFA enrollment bypass

Enforcing MFA at a point in time creates a chicken-and-egg problem: rejecting every unenrolled user
outright blocks legitimate new users and factor-loss recovery, but letting anyone register a factor on a
bare password success lets an attacker who merely knows the password enroll their own factor and defeat
the MFA requirement entirely. Before enforcement begins, unenrolled users get a normal password session
and are nudged toward pre-registering a factor through the already step-up-gated account security
screen. After enforcement begins, an unenrolled user can only reach a registration-only flow if an admin
has issued them an `MfaEnrollmentBypass` — a short-lived, single-use, server-side grant, not a
distributed secret — that is still unconsumed, unrevoked, and within its deadline; the enforcement date
and any grace period are operational timestamps only, never a basis for trusting who is enrolling.

A successful password login atomically consumes the bypass and moves the same `LoginSession` into
`pending_purpose=Enrollment`; that pending session is treated as unauthenticated everywhere except the
registration-only API and the original authorization transaction it was serving — it cannot reach
account, admin, or application resources. Only after the user proves possession of the new factor does
the session gain the second-factor AMR and resume as fully MFA'd; an expired, non-issuable, revoked, or
already-consumed bypass fails closed rather than falling back to any weaker path. The initial enrollment
factor is TOTP; WebAuthn is expected to plug into the same pending/bypass contract as a later adapter
without new policy or session states.

## Design Decisions

- Authentication owns the login-time identity-broker (upstream OIDC/SAML connections, external-subject
  links, linking/JIT policy) directly, rather than splitting it into a separate sourcing context
  ([ADR-146](../../decisions/ADR-146-authentication-owned-identity-broker.md)).
- PostgreSQL `authentication_sessions` is the single source of truth for `LoginSession`; revocation
  tombstones the row instead of deleting it, and Valkey holds no active-session state
  ([ADR-126](../../decisions/ADR-126-postgresql-as-login-session-source-of-truth.md)).
- `users.id` is the canonical, globally unique user identifier, with protocol `sub` claims derived from
  it rather than the reverse
  ([ADR-082](../../decisions/ADR-082-user-domain-id-and-tenant-key-policy.md)).
- Authentication event retention is asymmetric by kind (365/30/90 days), tenant-adjustable within a
  global cap, and enforced by an idempotent hourly sweep rather than partitioning or cold storage
  ([ADR-045](../../decisions/ADR-045-authentication-event-retention.md)).
- Reversible secrets kept in the app database, including the MFA TOTP seed, move to envelope encryption
  under the `DataKeys` context and an `EnvelopeCrypto` port rather than staying plaintext
  ([ADR-148](../../decisions/ADR-148-envelope-encryption-and-datakeys-context.md)).
- WebAuthn credentials and recovery codes are modeled as their own tables rather than squeezed into
  `mfa_factors`, ceremony logic is delegated to `go-webauthn/webauthn`, and recovery-code possession
  alone never counts toward `mfa_enrolled`
  ([ADR-087](../../decisions/ADR-087-webauthn-phishing-resistant-mfa.md)).
- Authentication event PII (username, IP, User-Agent, device fingerprint, location) was originally
  decided to be tenant-salt hashed or truncated rather than stored in plaintext
  ([ADR-046](../../decisions/ADR-046-authentication-event-pii-policy.md)).
- Login-throttle and other shared ephemeral state fail closed when their store is unreachable, rather
  than allowing unthrottled attempts through
  ([ADR-077](../../decisions/ADR-077-shared-login-throttle-store-and-ephemeral-state-ha.md)).
- Password policy follows NIST SP 800-63B-4: length and identifier-similarity checks plus a
  common-password dictionary, with no composition-rule or forced-rotation requirement
  ([ADR-026](../../decisions/ADR-026-password-policy.md)).
- Authentication and identity-management configuration values (password history depth, breach-check
  defaults, TOTP/WebAuthn/recovery-code parameters, reset-token TTL, login-throttle thresholds) are
  centralized in one policy-configuration ADR rather than scattered across SCL objectives
  ([ADR-106](../../decisions/ADR-106-identity-and-credential-policy-configuration.md)).
- `change-password` rejects reuse of the last 5 password hashes, checked only on change-password (not on
  initial registration) since a first registration has nothing to compare against
  ([ADR-027](../../decisions/ADR-027-password-history.md)).
- `BreachedPasswordChecker` layers an HIBP k-anonymity check on the bundled offline dictionary, fails
  open on outage, and ships a no-op default adapter so it introduces no external dependency by default
  ([ADR-028](../../decisions/ADR-028-breached-password-checker.md)).
- Forgot-password issues a single-use, hashed reset token with a uniform response and best-effort email
  delivery, so the flow cannot become a username-enumeration or SMTP-outage oracle
  ([ADR-030](../../decisions/ADR-030-password-reset-by-email.md)).
- `EmailSender`'s production adapter speaks SMTP only, not a per-provider HTTP SDK, since SMTP alone
  already reaches every major transactional-email provider
  ([ADR-035](../../decisions/ADR-035-smtp-email-sender-adapter.md)).
- Login throttling counts per-account and per-IP failures independently, keyed on hashed identifiers,
  and deliberately does not use permanent lockouts
  ([ADR-029](../../decisions/ADR-029-login-throttling.md)).
- Authentication events are split into individual rows and 5-minute bucket aggregates so a throttled
  actor's flood collapses into one row instead of growing without bound
  ([ADR-041](../../decisions/ADR-041-authentication-event-model.md)).
- The original hash-everything PII scheme for authentication events was superseded: username is no
  longer hashed since account-confirmed events already correlate by `user_id`, and admin search resolves
  a searched username to `user_id` on the fly instead
  ([ADR-104](../../decisions/ADR-104-username-search-drops-hashing.md)).
- The account self-service portal and the admin account API are separate contracts; the portal's own
  summary endpoint deliberately omits roles so it cannot leak admin metadata
  ([ADR-042](../../decisions/ADR-042-end-user-account-portal-scope.md)).
- High-sensitivity self-service operations (password change, TOTP removal, email change, revoking other
  sessions) require step-up re-authentication on top of CSRF protection
  ([ADR-043](../../decisions/ADR-043-account-portal-csrf-and-step-up.md)).
- Once MFA enforcement begins, an unenrolled user can only reach a registration-only flow through an
  admin-issued, single-use `MfaEnrollmentBypass` grant, never through a bare password success alone
  ([ADR-110](../../decisions/ADR-110-admin-authorized-mfa-enrollment-bypass.md)).
