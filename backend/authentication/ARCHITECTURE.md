---
context: authentication
updated_at: 2026-07-27
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
not own the login-time correlation policy. The ownership and rejected sourcing-context alternative are
recorded in [ADR-146](../../decisions/ADR-146-authentication-owned-identity-broker.md).

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

`authentication_sessions` (wi-253, ADR-126) is the single source of truth for a `LoginSession`.
`tenant_id` is kept alongside the user-derived path as an exception to the usual
[`tenant_id` retention classes](../../ARCHITECTURE.md#2-tenant_id-retention-classes): a session id is an
opaque browser-cookie value re-verified on every request, a fail-closed boundary where per-tenant lookup
matters the same way it does for `refresh_tokens.sid` (ADR-082 §4). Revocation sets
`revoked_at`/`revoke_reason` rather than deleting the row, so physical removal stays a housekeeping
concern independent of revoke state, and a repeated revoke is a safe no-op. Its two indexes serve, in
order: keyset pagination over one user's non-revoked sessions ordered by `auth_time DESC`, and the
housekeeping batch's `expires_at` cleanup scan.

`mfa_factors.secret` (pre-ADR-148) is the plaintext TOTP seed column, kept only so existing rows remain
readable (dual-read); new writes populate `secret_key_version`/`secret_ciphertext` and leave `secret`
`NULL`. wi-97 T006 backfills the remaining plaintext rows, after which `secret` is dropped.

`webauthn_credentials` (wi-26, ADR-087) keys on `credential_id` because one user can register several;
it stays a separate table from `mfa_factors` for that reason. `public_key` holds the COSE public key
(base64url). `recovery_codes` (wi-26, ADR-087) never stores the plaintext code, only `code_hash`
(SHA-256 hex); a non-`NULL` `consumed_at` means the code is used and cannot be replayed, and
regeneration replaces a user's whole set at once. `webauthn_sessions` is the WebAuthn ceremony
challenge store; `GetDel` is `DELETE ... WHERE expires_at > now() RETURNING data`.

`tenant_correlation_salts` (wi-145, ADR-046) is a per-tenant secret used to compute the correlated hash
(`SaltedHash`) of usernames/IPs and the throttle/bucket `keyHash`, so correlation is never aggregated
across tenants; it is generated on first use rather than provisioned up front.

`login_throttle_counters` (ADR-077's fail-closed premise) is `LOGGED` because losing it on failover would
be a defense-in-depth regression, and uses `fillfactor = 80` to leave header room for the frequent
same-row `UPDATE`s (HOT updates) a counter takes. `identifier_hash` is a SHA-256 hex digest, so no
plaintext username or IP is retained.
