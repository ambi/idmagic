---
context: authentication
updated_at: 2026-08-11
---

# Authentication Specification

## Overview

エンドユーザ (Subject) の資格情報検証、MFA、ログインセッション、パスワード変更・リセット、step-up、認証イベントを所有する。User / Group / Agent の identity ライフサイクルは IdManagement が所有する。

The `Authentication` context owns credential verification, MFA, login sessions, step-up, recovery,
and login-time federation. Independent capabilities are vertical feature slices with their own domain,
ports, use cases, and adapters; `module.go` remains the context composition boundary.

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| IdentityBroker | 外部 identity provider の認証結果を検証し、tenant 内の local User と安全に相関して LoginSession 発行へ渡す Authentication capability。 |  |
| ExternalIdentityProvider | idmagic に対して upstream authentication authority となる OIDC Provider または SAML Identity Provider。 | upstream IdP, social login provider |
| FederatedIdentity | tenant、provider、外部の不変 subject の組を local User へ一意に結び付ける identity link。 |  |
| JitProvisioning | 検証済み外部 claim と tenant の明示 policy / claim mapping に基づき、初回 federated login 中に local User を作成すること。 | JIT provisioning |
| Totp | RFC 6238 に基づく time-based one-time password。 | totp, otp |
| Webauthn | WebAuthn credential による認証。 | webauthn |
| RecoveryCode | TOTP / WebAuthn 喪失時に使う backup の使い捨て復旧コード。 | recovery_code |
| EndUser | 認証済みまたは認証を試みる一般利用者。ログイン・MFA継続・パスワードリセットなど、認証が未完了の操作の主体を指す。 |  |
| ResourceOwner | OAuth2/OIDC 認可フローでリソースの所有者として認可判断を行う利用者。EndUser と同一人物を OAuth2 文脈で指す呼称。 |  |

## Standards

### OpenID Connect Core 1.0

1.0 incorporating errata set 2 — https://openid.net/specs/openid-connect-core-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-CORE-CODE-FLOW | required | MUST | 外部 OIDC 認証は authorization code flow を使い、ID Token の署名、issuer、audience、有効時間、nonce を検証する。 |
| OIDC-CORE-CSRF | required | SHOULD | callback は login attempt に束縛された単発 state を照合し、不一致または再利用を拒否する。 |

### OpenID Connect Discovery 1.0

1.0 incorporating errata set 2 — https://openid.net/specs/openid-connect-discovery-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-DISCOVERY-ISSUER | required | MUST | discovery 文書の issuer は設定した issuer と完全一致し、endpoint と JWKS URI は事前に許可された HTTPS authority に限定する。 |

### TOTP Time-Based One-Time Password Algorithm

RFC 6238 — https://www.rfc-editor.org/rfc/rfc6238.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC6238-TOTP | optional | MUST | TOTP factor利用時は共有秘密と時間ステップからOTPを生成・検証する。 |

### Digital Identity Guidelines — Authentication and Authenticator Management

NIST SP 800-63B-4 — https://pages.nist.gov/800-63-4/sp800-63b.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| NIST63B4-PASSWORD-MINIMUM | excluded | MUST | 単一要素認証に使用するPasswordへ15文字以上の最小長を要求する。 |
| NIST63B4-NO-COMPOSITION | required | MUST NOT | 文字種混在などPassword composition ruleを課さない。 |
| NIST63B4-PASSWORD-STORAGE | required | MUST | Passwordをsaltとcost factorを持つoffline attack耐性のあるhashとして保存する。 |

### Web Authentication — An API for accessing Public Key Credentials Level 3

Candidate Recommendation Snapshot — https://www.w3.org/TR/webauthn-3/

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WEBAUTHN3-AUTHENTICATION | required | MAY | WebAuthn factor利用時はoriginとRelying Partyにscopeされた公開鍵Credentialを検証する。 |
| WEBAUTHN3-REGISTRATION | required | MUST | WebAuthn credential登録時はattestationのchallenge / RP ID / originを検証し、COSE公開鍵とsign countを保存する。 |

### Authentication Method Reference Values

RFC 8176 — https://www.rfc-editor.org/rfc/rfc8176.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8176-AMR-VOCABULARY | required | MUST | LoginSession.amr は RFC 8176 登録値 (pwd, otp, webauthn, hwk, swk) のサブセットに、本アプリ固有の非 IANA 拡張値 rc (recovery code) を加えた語彙のみを許可する。 |

## State Transitions

### IdentityProviderConnectionLifecycle

upstream connection は利用可能 Active と routing 停止 Disabled の2状態だけを遷移する。作成直後の初期状態は Disabled。metadata refresh 失敗や trust source 以外のフィールド更新は状態を変えず last-known-good を保持する。

Initial: `Disabled`
Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | IdentityProviderConnectionDisabled | — | Disabled |  |
| Disabled | IdentityProviderConnectionActivated | — | Active |  |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Inbound identity federation

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

### Persistence

`authentication_sessions` is the single source of truth for a `LoginSession`.
`tenant_id` is kept alongside the user-derived path as an exception to the usual
[`tenant_id` retention classes](../../SPECIFICATION.md#2-tenant_id-retention-classes): a session id is an
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

### Password lifecycle

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

### Login throttling

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

### Authentication event logging

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

### Account portal trust boundary and step-up

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
`RequestEmailChange`, and `RevokeMyOtherSessions` require step-up in the requirements, with a test
(`TestStepUpAnnotatedInterfacesMatchGatedHandlers`) keeping the annotation and the gated handlers from
drifting apart. A session counts as stepped-up if `max(session.auth_time, session.step_up_at)` is within
`StepUpRecencySeconds` (5 minutes) — so a session is stepped-up immediately after login, matching the
re-auth pattern users already expect from Google/Okta-style prompts. A gate failure returns
`403 step_up_required`, not `401`, since the session is authenticated and has simply not proven recency
for this specific action; the UI reissues the original request once `POST /api/account/step_up/complete`
succeeds. Recency is stored as `step_up_at` on the `LoginSession` row itself so it cannot follow the
cookie to a different device.

### WebAuthn/passkey MFA and recovery codes

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

### MFA enrollment bypass

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

### Design Decisions

- Authentication owns the login-time identity-broker (upstream OIDC/SAML connections, external-subject
  links, linking/JIT policy) directly, rather than splitting it into a separate sourcing context.
- PostgreSQL `authentication_sessions` is the single source of truth for `LoginSession`; revocation
  tombstones the row instead of deleting it, and Valkey holds no active-session state.
- `users.id` is the canonical, globally unique user identifier, with protocol `sub` claims derived from
  it rather than the reverse.
- Authentication event retention is asymmetric by kind (365/30/90 days), tenant-adjustable within a
  global cap, and enforced by an idempotent hourly sweep rather than partitioning or cold storage.
- Reversible secrets kept in the app database, including the MFA TOTP seed, move to envelope encryption
  under the `DataKeys` context and an `EnvelopeCrypto` port rather than staying plaintext.
- WebAuthn credentials and recovery codes are modeled as their own tables rather than squeezed into
  `mfa_factors`, ceremony logic is delegated to `go-webauthn/webauthn`, and recovery-code possession
  alone never counts toward `mfa_enrolled`.
- Authentication event PII (username, IP, User-Agent, device fingerprint, location) was originally
  decided to be tenant-salt hashed or truncated rather than stored in plaintext.
- Login-throttle and other shared ephemeral state fail closed when their store is unreachable, rather
  than allowing unthrottled attempts through.
- Password policy follows NIST SP 800-63B-4: length and identifier-similarity checks plus a
  common-password dictionary, with no composition-rule or forced-rotation requirement.
- Authentication and identity-management configuration values (password history depth, breach-check
  defaults, TOTP/WebAuthn/recovery-code parameters, reset-token TTL, login-throttle thresholds) are
  centralized in this policy section rather than scattered across product objectives.
- `change-password` rejects reuse of the last 5 password hashes, checked only on change-password (not on
  initial registration) since a first registration has nothing to compare against.
- `BreachedPasswordChecker` layers an HIBP k-anonymity check on the bundled offline dictionary, fails
  open on outage, and ships a no-op default adapter so it introduces no external dependency by default.
- Forgot-password issues a single-use, hashed reset token with a uniform response and best-effort email
  delivery, so the flow cannot become a username-enumeration or SMTP-outage oracle.
- `EmailSender`'s production adapter speaks SMTP only, not a per-provider HTTP SDK, since SMTP alone
  already reaches every major transactional-email provider.
- Login throttling counts per-account and per-IP failures independently, keyed on hashed identifiers,
  and deliberately does not use permanent lockouts.
- Authentication events are split into individual rows and 5-minute bucket aggregates so a throttled
  actor's flood collapses into one row instead of growing without bound.
- The original hash-everything PII scheme for authentication events was superseded: username is no
  longer hashed since account-confirmed events already correlate by `user_id`, and admin search resolves
  a searched username to `user_id` on the fly instead.
- The account self-service portal and the admin account API are separate contracts; the portal's own
  summary endpoint deliberately omits roles so it cannot leak admin metadata.
- High-sensitivity self-service operations (password change, TOTP removal, email change, revoking other
  sessions) require step-up re-authentication on top of CSRF protection.
- Once MFA enforcement begins, an unenrolled user can only reach a registration-only flow through an
  admin-issued, single-use `MfaEnrollmentBypass` grant, never through a bare password success alone.

## Scenarios

### REQ-AUTHENTICATION-001: 外部OIDC認証は検証済みsubjectを同じlocal Userへ相関する
- ACTOR EndUser
- GIVEN request tenant で OIDC connection が Active である
- GIVEN issuer、authorization endpoint、token endpoint、JWKS は管理時に検証済みである
- WHEN EndUser が StartFederatedLogin を開始する
- THEN state、nonce、PKCE を単発 attempt に保存して upstream へ遷移する
- WHEN upstream callback が code と ID Token を返す
  - ALT 同じ state または token response を再利用する → single-use attempt / replay guard が拒否する
- THEN CompleteFederatedLogin は code と ID Token の署名、issuer、audience、時刻、nonce を検証する
  - ALT state、nonce、issuer、audience、署名、時刻のいずれかが一致しない → callback を拒否し LoginSession と link を作成しない → FederatedLoginRejected を発行する
- THEN 初回は明示 JIT policy と claim mapping により local User と FederatedIdentity を作成する
- THEN 2回目は同じ tenant、provider、external subject の既存 link から同じ local User を解決する
- THEN federated AMR の LoginSession を発行する

### REQ-AUTHENTICATION-002: verified emailによる自動linkは明示policyと一意一致を要求する
- ACTOR EndUser
- GIVEN external subject の既存 link は無い
- GIVEN 同じ email の local User が tenant 内に存在する
- GIVEN provider の linking_policy が VerifiedEmail である
- GIVEN upstream email_verified claim が true であり、email は tenant 内で一意に一致する
- WHEN EndUser が未連携の external subject で federated login を完了する
  - ALT policy が None、email が未検証、または一致が曖昧である → 自動 link と LoginSession 発行を拒否する
- THEN FederatedIdentity を既存 User に作成する

### REQ-AUTHENTICATION-003: external identityの明示linkとunlinkはstep-upを要求する
- ACTOR AuthenticatedSelf
- GIVEN ResourceOwner は対象 tenant の active User である
- WHEN 直近5分以内の step-up session で provider の外部認証を完了する
  - ALT step-up が古い、または無い → link / unlink を AccessDeniedError で拒否する
- THEN external subject が未使用なら自身へ link する
- WHEN 直近5分以内の step-up session で link の解除を要求する
  - ALT password credential も他の external identity link も残らない → account lockout 防止のため unlink を拒否する
- THEN 対象の external identity link を解除する

### REQ-AUTHENTICATION-004: API token発行者はsensitive facet scope内で自身のauthentication情報だけを操作できる
- ACTOR SelfApiClient
- GIVEN client は対象 tenant の active User に固定された有効な API access token を提示している
- WHEN client が account context、security、signin activity、session、MFA factor、recovery code、password、または session の操作を要求する
  - ALT 対応しない account scope で sensitive facet の変更を要求する → 操作は AccessDeniedError で拒否される
  - ALT token の tenant または user_id が操作対象と一致しない → 操作は AccessDeniedError で拒否される
  - ALT API token で step-up endpoint を要求する → 操作は AccessDeniedError で拒否される
- THEN account:read scope は自身の account context、security、signin activity、session の参照だけを許可する
- THEN account:mfa:write scope は自身の MFA factor と recovery code の変更だけを許可する
- THEN account:sessions:write scope は自身の session の失効だけを許可する
- THEN account:password:write scope と current password は自身の password の変更だけを許可する

### REQ-AUTHENTICATION-005: browser bootstrap contextは認証状態とCSRF境界を保持する
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済み session または first-party portal の access token を持つ
- WHEN browser または API client が account context を要求する
  - ALT session が未認証または認証途中である → account context の取得は AccessDeniedError で拒否される
  - ALT Bearer token が許可された portal scope または account:read scope を一つも持たない → account context の取得は AccessDeniedError で拒否される
- THEN management portal は idmagic.admin、account portal は idmagic.account、自己管理 API client は account:read scope で同じ account context を取得できる
- THEN 応答は subject、realm、effective role、CSRF token を含む
- WHEN 未認証のパスワードリセット画面が password reset context を要求する
- THEN CSRF token を含む context が返る

### REQ-AUTHENTICATION-006: ユーザーはWebAuthnでstep-up challengeを開始できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が WebAuthn credential を登録済みで認証済み session を持つ
- WHEN ユーザー "alice" が正しい CSRF token で step-up WebAuthn challenge を要求する
  - ALT CSRF token が一致しない、または WebAuthn が利用不能である → challenge は発行されず要求は拒否される
- THEN 応答の PublicKeyCredentialRequestOptions は現在 session に束縛される

### REQ-AUTHENTICATION-007: ResourceOwnerはブラウザでパスワード認証し認可を継続する
- ACTOR ResourceOwner
- GIVEN 未認証セッションで "web-app" として認可リクエストを送信済みである
- WHEN browser login API に username "alice" と正しい password を送信する
  - ALT SameSite cookie と request token が一致しない → csrf 値を改ざんして browser login API を送信する → エラー "InvalidRequestError"
  - ALT 直近 900 秒窓で per-account の失敗回数が 10 回に達している → 正しい password で browser login API を送信する → エラー "RateLimitedError" → "LoginThrottled" が発行される
  - ALT 失敗回数に関わらず同一 IP からの login API リクエストが EndpointRateLimitPolicy の window 内で max_requests に達している → 正しい password で browser login API を送信する → エラー "RateLimitedError"
- THEN セッション Cookie が発行される
- THEN 認可コードが redirect_uri に返る
- THEN "UserAuthenticated" が発行される

### REQ-AUTHENTICATION-008: パスワードリセット要求は識別子とIPの組でrate limitされる
- ACTOR EndUser
- GIVEN 未認証である
- WHEN "alice" 宛のパスワードリセットを要求する
  - ALT 同一 identifier と IP の組で EndpointRateLimitPolicy の window 内の max_requests に達している → "alice" 宛のパスワードリセットを再度要求する → エラー "RateLimitedError"
- THEN user の存在有無に関わらず 204 を返す
- THEN "PasswordResetRequested" が発行される

### REQ-AUTHENTICATION-009: 無効化されたユーザーは新規ログインも既存セッションも拒否される
- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" が認証済みセッションを持つ
- WHEN 管理者がユーザー "alice" を無効化する
- THEN ユーザー "alice" は無効状態になる
- WHEN ユーザー "alice" が既存セッションで認証必須 API を呼ぶ
- THEN エラー "AccessDeniedError"
- WHEN ユーザー "alice" が正しい password で新規ログインを試みる
- THEN エラー "AccessDeniedError"

### REQ-AUTHENTICATION-010: ユーザーは現在のパスワードを確認して新しいパスワードへ変更できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでパスワード変更画面を開いている
- WHEN ユーザー "alice" が正しい現在のパスワードと新しいパスワードを送信する
  - ALT 新しいパスワードが 12 文字未満である → ユーザー "alice" が 12 文字未満のパスワードを送信する → エラー "InvalidRequestError"
  - ALT 新しいパスワードが直近 5 件の履歴に一致する → ユーザー "alice" が直近使用した過去のパスワードを新パスワードとして送信する → エラー "InvalidRequestError"
- THEN パスワードが変更され password_changed_at が更新される
- THEN "PasswordChanged" が発行される

### REQ-AUTHENTICATION-011: ユーザーはTOTP factorを登録して有効化できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでセキュリティ画面を開いている
- WHEN ユーザー "alice" が TOTP 登録を開始する
- THEN 応答に secret と account_name が含まれる
- WHEN ユーザー "alice" がその secret に対する正しいコードで登録を確認する
- THEN セキュリティ概要の MFA 状態が登録済みになる
- THEN "MfaFactorEnrolled" が発行される

### REQ-AUTHENTICATION-012: ユーザーはstep-up再認証のうえでTOTP factorを解除する
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が登録済み TOTP factor を持ち認証済みである
- WHEN ユーザー "alice" が step-up を成立させ現在の TOTP コードで解除する
  - ALT step-up なしで解除を試みる → ユーザー "alice" が step-up なしで TOTP factor の解除を試みる → step-up 再認証が要求される
- THEN TOTP factor が解除される
- THEN "MfaFactorRemoved" が発行される

### REQ-AUTHENTICATION-013: ユーザーは自分の有効なセッションを一覧して失効できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が複数の有効なセッションを持ち認証済みである
- WHEN ユーザー "alice" がアクティビティ画面でセッション一覧を取得する
  - ALT process 再起動を挟んでセッション一覧を取得する → サーバープロセスを再起動する → ユーザー "alice" が同じ session cookie でアクティビティ画面を開く → セッションは再起動前と同じ内容で解決できる
- THEN 自分の有効なセッションが返る
- WHEN ユーザー "alice" が現在以外のセッションを 1 件失効させる
  - ALT 既に失効済みのセッションへ同じ失効操作を再送する → ユーザー "alice" が直前に失効させた同じセッション id へ再度失効を要求する → 要求は成功として扱われ、最初の失効時刻が保持される
- THEN 失効したセッションは一覧から消える
- WHEN ユーザー "alice" が現在以外のすべてのセッションを一括失効させる
- THEN 現在のセッションだけが残る

### REQ-AUTHENTICATION-014: ユーザーは自分のサインイン履歴を確認できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでアクティビティ画面を開いている
- WHEN ユーザー "alice" が自分のサインイン履歴を取得する
  - ALT 認証手段に WebAuthn が含まれる → UI は webauthn という技術名ではなく「パスキー」と表示する
- THEN 応答に自分のサインインイベントだけが含まれる
- THEN 第二要素を使ったサインインは、pwd と第二要素の amr を含む完了後の UserAuthenticated として表示される

### REQ-AUTHENTICATION-015: MFA登録済みユーザーでもポリシーが要求しない限り第二要素を求められない
- ACTOR EndUser
- GIVEN ユーザー "alice" は TOTP または WebAuthn credential を登録済みである
- GIVEN 対象 Application の実効サインインポリシーは Password である
- WHEN ユーザー "alice" がユーザー名とパスワードを送信する
  - ALT 対象 Application の実効サインインポリシーが Mfa である → LoginSession は authentication_pending=true へ切り替わる → 利用可能な第二要素 (TOTP / パスキー / リカバリコード) の選択画面へ進む
- THEN LoginSession は authentication_pending=false で作られる
- THEN 認可フローは第二要素画面に進まず、同意または認可コード発行へ進む

### REQ-AUTHENTICATION-016: ユーザーはメールのリセットリンクでパスワードを再設定する
- ACTOR EndUser
- GIVEN ユーザー "alice" 宛に有効なパスワードリセットトークンが発行されている
- WHEN ユーザー "alice" がそのトークンと新しいパスワードを送信する
  - ALT トークンが期限切れまたは不正である → 無効なパスワードリセットトークンで新しいパスワードを送信する → エラー "InvalidRequestError"
- THEN パスワードが更新される
- WHEN ユーザー "alice" が新しいパスワードを browser login API に送信する
- THEN ログインに成功する
- WHEN EndUser が未登録のメールアドレスでパスワードリセットを要求する
- THEN 応答は登録済みアドレスに対する応答と区別できない
- WHEN EndUser が登録済みのメールアドレスでパスワードリセットを要求する
- THEN 登録済みアドレスへリセットリンクが送られる

### REQ-AUTHENTICATION-017: TOTP必須ユーザーは正しいコードで認証を継続できる
- ACTOR EndUser
- GIVEN TOTP factor が登録された authentication_pending の LoginSession が存在する
- WHEN browser TOTP API に正しいコードを送信する
  - ALT 誤った TOTP コードを送信する → browser TOTP API に誤ったコードを送信する → エラー "InvalidRequestError" → LoginSession は authentication_pending のままである
- THEN 認証が成立し認可フローが継続する
- THEN "UserAuthenticated" が発行される

### REQ-AUTHENTICATION-018: MFA未登録ユーザーは管理者承認済みオンボーディングを完了して同じ認可処理を継続できる
- ACTOR EndUser
- GIVEN 対象 Application の実効ポリシーは MFA 必須で強制開始済み、enrollment bypass を許可し猶予期限内である
- GIVEN user は TOTP / WebAuthn factor を持たない
- GIVEN 管理者が対象 user に有効な単発 enrollment bypass を発行済みである
- WHEN user が正しい password を送信する
  - ALT enrollment bypass が無い、取消済み、消費済み、または期限切れである → password が正しくてもログインを完了せず access denied にする → factor 登録 API は利用できない
- THEN bypass は消費され、同一 LoginSession は pending_purpose=Enrollment の未完了状態になる
- THEN MfaEnrollmentRequired と MfaEnrollmentBypassConsumed が発行され、登録専用画面へ進む
- WHEN user が TOTP secret に対する正しい code で登録を確定する
  - ALT enrollment deadline を過ぎている → factor を保存せず access denied にする → LoginSession を認証完了へ昇格させない
  - ALT TOTP code が不正である → factor を保存せず InvalidRequestError を返す → LoginSession は Enrollment pending のままである
- THEN factor が保存され、同一 LoginSession に otp が追加されて pending が解除される
- THEN MfaEnrollmentCompleted と UserAuthenticated が発行され、元の authorization transaction が継続する

### REQ-AUTHENTICATION-019: MFA強制開始前の未登録ユーザーはログインできるが登録を促される
- ACTOR EndUser
- GIVEN テナントデフォルトポリシーは将来時刻から MFA 必須になる
- GIVEN user は MFA factor を持たない
- WHEN user が正しい password でログインする
- THEN 強制開始前なので password session は成立する
- THEN UI は強制開始日時と事前登録を促す警告を表示する
- THEN user は通常の step-up を経た account security から factor を事前登録できる

### REQ-AUTHENTICATION-020: Enrollment pendingセッションは通常リソースへアクセスできない
- ACTOR EndUser
- GIVEN pending_purpose=Enrollment の LoginSession が存在する
- WHEN user が account、admin、Application の resource を要求する
- THEN システムは未認証として拒否する
- THEN 登録専用 start / confirm API と元の auth transaction だけを許可する

### REQ-AUTHENTICATION-021: 管理者は対象ユーザーのセッションを一覧・個別失効・全失効できる
- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" が複数の有効な LoginSession を持つ
- WHEN 管理者がユーザー "alice" の ListSessions を呼ぶ
  - ALT 他テナントの管理者が呼び出す → エラー "AccessDeniedError"
- THEN 開始時刻の降順で有効なセッション一覧が返る
- WHEN 管理者がそのうち1件の RevokeSession を呼ぶ
  - ALT 既に失効済みのセッションへ再度 RevokeSession を呼ぶ → 204 が返り revoked_at は初回の値を保持する
- THEN 対象セッションは revoke_reason=admin_revoke で失効し "SessionEnded" が発行される
- WHEN 管理者がユーザー "alice" の RevokeUserSessions を呼ぶ
- THEN 残り全セッションが失効する

### REQ-AUTHENTICATION-022: 管理者は認証器を全リセットしたユーザーに次回ログインで再登録を強制できる
- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" は TOTP factor を持ち、recovery code も生成済みである
- WHEN 管理者がユーザー "alice" の ResetUserAuthenticators を targets=[Totp, RecoveryCode] で呼ぶ
  - ALT 他テナントの管理者、または admin ロールを持たない操作者が呼び出す → エラー "AccessDeniedError" → 対象ユーザーの認証器は変更されない
- THEN "AuthenticatorResetRequested" が発行される
- THEN TOTP factor と recovery code が削除され、他に WebAuthn credential も無いため mfa_enrolled が false になる
- THEN reenrollment_required=true の応答が返り、単発 enrollment bypass が自動発行される
- THEN "AuthenticatorResetCompleted" と "MfaEnrollmentBypassIssued" が発行される
- WHEN alice が正しい password で次にログインする
- THEN 有効な bypass により同一 LoginSession が pending_purpose=Enrollment になる
- WHEN alice が新しい TOTP factor の登録を確定する
- THEN 同一 LoginSession が MFA 済みに昇格し元の authorization transaction が継続する

### REQ-AUTHENTICATION-023: 管理者が一部の認証器のみリセットした場合は残存要素でログインを継続できる
- ACTOR TenantAdministrator
- GIVEN ユーザー "bob" は TOTP factor と WebAuthn credential を両方持つ
- WHEN 管理者がユーザー "bob" の ResetUserAuthenticators を targets=[Webauthn] で呼ぶ
- THEN WebAuthn credential のみ削除され、TOTP factor は残るため mfa_enrolled は true のままである
- THEN reenrollment_required=false の応答が返り、enrollment bypass は発行されない
- WHEN bob が次回ログインで TOTP コードによる第二要素検証を完了する
- THEN ログインを完了できる
