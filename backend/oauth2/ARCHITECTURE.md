---
context: oauth2
updated_at: 2026-08-10
---

# Architecture: oauth2

## Overview

The `oauth2` context implements an OAuth 2.0 / OIDC authorization server as a set of feature
slices — `authorization/`, `client/`, `consent/`, `device/`, `token/` — each owning its own
`domain`, `ports`, `usecases`, and `db_memory`/`db_postgres` adapters. The context-level `domain`,
`ports`, and `usecases` packages are compatibility facades over the slices, `handlers_http` is the
shared HTTP/persistence adapter, and `module.go` is the single composition root. Read this document
mechanism by mechanism: authorization/device lifecycles, PKCE/PAR, client authentication, token
formats and rotation, sender constraints, consent, authorization policy, discovery, the device
grant, lifetime/security configuration, agent principals and delegation, rich authorization
requests, and session/logout binding.

## Authorization and device lifecycles as declarative state machines

The `AuthorizationRequest` and device-code lifecycles are expressed as declarative
state-transition tables in `spec/flows/` (states, events, transitions) instead of being scattered
across `if`/`switch` logic, so that regenerating the adapter layer cannot silently drift the set of
transitions a client is allowed to make. Refresh token families are
deliberately excluded from this treatment: their state space is effectively just
`{active, revoked, rotated}`, and the parent/child rotation graph matters more than transition
legality, so they are expressed as record fields and revocation rules instead (see Refresh token
rotation below). `authorization/usecases` and `device/usecases` consume these tables directly
rather than re-implementing transition logic.

## PKCE and Pushed Authorization Requests

PKCE requirement is per-client (`require_pkce` client metadata), defaulting to required for public
and FAPI 2.0 clients and opt-in for legacy confidential clients, so RFC 6749-era confidential
deployments are not forced to migrate while public/FAPI clients keep the strongest default. Only
`S256` is supported as
`code_challenge_method`; `plain` is rejected because it lets an interceptor recover the verifier
from logs. Authorization codes stay single-use and short-lived (≤60s) independent of PKCE, since
both reuse detection and replay-window minimization depend on that.

Pushed Authorization Requests (`/par`) let a client submit authorization parameters over an
authenticated back channel and reference them from `/authorize` via `request_uri`, closing off URL
tampering, open-redirect abuse, and unauthenticated request forgery at `/authorize`. FAPI 2.0 clients
(`require_pushed_authorization_requests`) must use PAR; other clients may use either path.
`request_uri` values are single-use with a ≤600s TTL, and once `/authorize` resolves a
`request_uri`, any additional query parameters on that request are ignored in favor of the pushed
ones (RFC 9126 §4) — otherwise an attacker could reattach parameters to a legitimate pushed
request.

## Client authentication

Five `token_endpoint_auth_method`s are supported — `private_key_jwt`, `tls_client_auth`, `none`,
`client_secret_post`, `client_secret_basic` — spanning FAPI-grade asymmetric authentication down to
legacy shared-secret methods, so new clients can be steered toward the strongest options while
existing deployments keep migrating. `client_secret_jwt` (HMAC) is deliberately not implemented: once `private_key_jwt` is available, a
symmetric-key alternative only adds risk without adding capability. A failed client authentication
always returns `401 invalid_client` without revealing whether the `client_id` is registered, and an
unregistered `client_id` still pays the same verification round-trip cost, to avoid a timing oracle.

`private_key_jwt` verification in `handlers_http` is pinned to a fixed rule set so that what
Discovery advertises is what the server actually checks: signature algorithm restricted to
`PS256`/`ES256` (never `none` or HMAC), `iss == sub == client_id`, audience matching this server's
issuer or endpoint URLs, signing keys resolved from the client's registered inline `jwks` or
`jwks_uri`, a bounded assertion lifetime, and single-use `jti` replay protection kept in its own
store separate from DPoP's replay store because the two have different TTLs and audit semantics.
Combining `client_assertion`
with Basic/secret authentication on the same request is rejected as `invalid_request` per RFC 6749
§2.3.

## Client ID Metadata Documents (CIMD): registry-less client resolution

Alongside RFC 7591 Dynamic Client Registration, `client_id` values shaped as an `https` URL with a
path resolve live from a client-hosted Client ID Metadata Document instead of the `OAuth2Client`
repository. Resolution is
never persisted — the document is fetched (and cached 5 minutes) each time a `client_id` isn't
found in the repository, then mapped into the same `OAuth2Client` shape every other code path
already understands, so redirect_uri matching, consent rendering, PKCE, and scope handling need no
CIMD-specific branches. The integration point is `client/cimd_http.ClientRepositoryWithCIMD`, a
decorator that embeds `OAuth2ClientRepository` and overrides only `FindByID`: a repository hit
short-circuits before any fetch, and every other method (`Save`, `Delete`, `FindAll`, credential
listing) passes straight through untouched. It is wired once at the composition root
(`cmd/internal/bootstrap`), so `authorize.go`, `push_authorization_request.go`, and
`client_auth.go` require no changes.

The fetch itself goes through `shared/security/safehttp`, the same SSRF-hardened dialer
`tokens_jose.JWKResolver` uses for `jwks_uri` (https-only, DNS-resolved-then-public-IP-only,
validated-IP direct dialing, no environment proxy, capped redirects, short timeouts, and a body
size cap). Direct dialing is required because a proxy would resolve and connect to the final target
outside the checked dial path, bypassing the transport's SSRF boundary. The shared package keeps
both fetchers behind one hardened implementation rather than two. MVP only accepts documents that
omit `token_endpoint_auth_method` or declare `none`; anything else is rejected fail-closed, and a
document's `client_id` field must match the URL it was fetched from exactly. A resolved client's
`scope` is whatever the document self-declares (default `openid`) — the same self-declared trust
model RFC 7591 DCR already uses, not a new admin-curated catalog. A CIMD-resolved client is never
linked to an `Application`, matching the existing behavior for self-registered DCR clients: the
`ApplicationGate` already treats "no Application record" as allowed, not fail-closed.

## Token formats: JWT access tokens, opaque refresh tokens

Access tokens are issued as self-contained JWTs (RFC 9068) by default, refresh tokens as opaque,
database-backed references. The asymmetry is deliberate: with many resource servers, an `/introspect` round-trip per request
doesn't scale, so a JWT a resource server can verify from JWKS alone is the better default, with a
short (600s) TTL bounding the exposure of not being able to revoke it instantly. Refresh tokens need
rotation and family-wide revocation (see below), which is naturally a database-record operation, so
opacity avoids keeping a JWT and a revocation record in sync for no benefit; refresh tokens are
stored as SHA-256 hashes, never plaintext. `/introspect` is still exposed for resource servers that
want to confirm sender-constraint (`cnf`) or real-time revocation state, but it is not the default
verification path for JWT access tokens.

## Refresh token rotation and reuse detection

Every refresh token use rotates it: the presented token is marked `rotated`, a new one is issued
carrying `parent_id`, and every token descending from one authorization code exchange shares a
`family_id`. Presenting an already-rotated token is treated as reuse: the request is rejected, every token in that `family_id`
is revoked, and a `RefreshTokenReuseDetected` audit event fires. This applies uniformly to public
and confidential clients, to keep the operational and audit story simple. Genuinely concurrent
legitimate use (e.g., two open tabs) is not distinguished from replay — one succeeds and the other
is treated as reuse — trading an occasional forced re-login for avoiding the complexity and
external-state cost of a grace-period window. `absolute_expires_at` is fixed at issuance (30 days)
and rotation never extends it.

## Sender-constrained tokens: DPoP and mTLS

DPoP (RFC 9449) is the default sender-constraint mechanism because it works identically for web
apps, SPAs, and native clients without requiring any change to a TLS-terminating proxy; mTLS
(RFC 8705) is offered as an option for organizations that already run client PKI, particularly
FAPI/banking clients. Clients declaring the FAPI 2.0 profile must use at least one of the two;
general-profile clients opt
in via `dpop_bound_access_tokens`. DPoP proof validation checks the `jwk`-header signature plus
`htm`/`htu`/`iat`/`jti`, with a bounded clock skew and a replay window on `jti`, and issued tokens
carry the JWK thumbprint in `cnf.jkt`. mTLS validation trusts a TLS-terminating proxy to pass a
verified client certificate, matches the registered `tls_client_auth_subject_dn`, and binds issued
tokens via `cnf.x5t#S256`; `/userinfo` requires the presented certificate's thumbprint to match the
token's `cnf` before accepting it. Both access and refresh tokens carry the sender constraint —
refresh tokens via a `sender_constraint` field on the store record — so proof-of-possession survives
rotation, and `/introspect` responses include `cnf` so resource servers can re-verify DPoP proofs at
the request level.

## Consent

Consent is persisted per `(subject, client_id)` as a set of granted scopes — not per-client and not
per-interaction. A per-client grant would
silently extend to newly requested high-privilege scopes added later, which conflicts with
purpose-specific consent; asking on every interaction would create consent fatigue and pressure
toward ad hoc "remember me" shortcuts. The consent UI is skipped only when every requested scope is
already covered by an unexpired, unrevoked grant; new scopes trigger a UI that highlights only the
delta. Grants expire 365 days after being granted, aligned with periodic re-consent expectations,
and `prompt=consent` forces the UI regardless of an existing grant. Revoking consent affects future
authorizations only; revoking a client's refresh tokens at the same time is an explicit, separate
action that reuses the family-revocation mechanism from refresh token rotation, while already-issued
short-lived access tokens are left to expire naturally.

## Authorization policy (AuthZEN)

Every authorization decision in this context — client/redirect_uri checks, grant-type entitlement,
refresh token validity, sender-constraint/proof matching, `/userinfo` scope checks, `/introspect`
caller authentication — is declared in `spec/policy/client-authorization.json` and evaluated through
an AuthZEN-style `authorize({subject, action, resource, context})` interface, rather than scattered
as inline conditionals across adapters where a check could silently drop on regeneration. The
current evaluator is a local
pure-function adapter over that same policy document; swapping in an external AuthZEN service, OPA,
or Cedar later only touches the adapter, not the usecases that call `authorize()`. Every `rules[].id`
declared in the policy JSON must have a matching implementation, which an invariants test enforces
so a newly declared rule cannot silently ship unimplemented.

## Discovery

The OAuth 2.0 Authorization Server Metadata / OIDC Discovery document is a derived artifact, not a
hand-maintained one: a template in `spec/discovery.json` is read at runtime with the issuer
placeholder substituted in, and its content (supported grants, auth methods, signing algorithms,
response types, PKCE methods) is cross-checked against the other specification-core files
authoritative for those facts, such as `grants/grant-types.json`, the token schemas, and the
configured PKCE method requirements. This avoids both failure
modes of the alternatives: a hand-maintained document drifting from the implementation, and a
build-time-generated document creating a second copy that goes stale if the build step is skipped.

## Device Authorization Grant

The device flow (RFC 8628) — `POST /device_authorization`, the `/device` verification UI, and the
`device_code` grant at `/token` — consumes the state-transition table already declared in
`spec/flows/device-code-flow.json` rather than reimplementing approve/deny/exchange transitions ad
hoc. `device_code` is a 32-byte
random value stored only as a SHA-256 hash (bearer secret); `user_code` uses a reduced, unambiguous
20-character alphabet (excludes vowels and visually confusable characters) rendered in
`WDJB-MJHT`-style groups. Polling honors `authorization_pending`/`slow_down`/`access_denied`/
`expired_token` against the spec-core-owned interval and backoff increment, and an approved code
moves `approved → exchanged` before token issuance to prevent double issuance.

## Lifetime, security, and retention configuration

Protocol timing and security parameters — authorization code TTL (60s, single-use), PAR
`request_uri` TTL (600s, single-use), access token TTL (600s), ID token TTL (3600s), refresh token
TTL (14 days sliding / 30 days absolute), device and user code TTL (600s), default polling interval
(5s, +5s per `slow_down`), client-authentication and code-redemption rate limits, DPoP clock skew and
replay window, and consent record retention (7 years) — are recorded together in one place rather
than as SCL `objectives`, because they are protocol/security/operational settings, not availability
or latency SLOs with error-budget semantics. Values a single model, state, or interface can
naturally enforce are expressed there as constraints/guards/contracts; values that don't belong to
one element — a rate limit spanning multiple requests, a retention window spanning a lifecycle —
stay authoritative in that shared configuration record instead.

## Agent principals and token-exchange delegation

`Agent` is a first-class principal distinct from `User` and `OAuth2Client`: it owns identity,
ownership, purpose, and lifecycle (including a kill-switch), but deliberately holds no credential
primitives of its own — it binds to one or more existing `OAuth2Client` registrations instead, so
agent governance doesn't require a second, redundant set of credential/crypto machinery. Every
agent has a
required owner (a `User` or group), so offboarding an owner can cascade to the agent's access;
`status` (`active`/`disabled`/`killed`) is checked fail-closed on every token-issuance path, meaning
an unresolved status blocks issuance rather than allowing it. Access token claims carry an optional
principal-type marker so resource servers and the AuthZEN policy layer can distinguish agent-issued
tokens without breaking existing token consumers.

Acting on a user's behalf is implemented as OAuth 2.0 Token Exchange (RFC 8693) at `/token`. The
default outcome is delegation, not impersonation: the exchanged token keeps the original user as `sub` and
records the agent as current actor in the `act` claim, nesting prior actors inward per RFC 8693
§4.1 so a chain of sub-agent delegation stays traceable; impersonation (`act` dropped, `sub`
replaced) is available only where a client/agent is explicitly permitted, and any unresolved case
defaults to delegation because that is the side that preserves the audit trail. `may_act` and the
AuthZEN policy jointly gate which actor/audience/depth combinations are permitted, exchanges must
specify a `resource` narrowing the result to a single audience (RFC 8707), a configurable maximum
delegation depth bounds `act`-chain length, and exchanged tokens are short-lived with no refresh
token issued — continuation means re-exchanging, which keeps revocation effective. Sender
constraints on the subject token carry through to the exchanged token so proof-of-possession is not
lost in the exchange.

## Rich Authorization Requests for agent-scoped permissions

Coarse OAuth scopes cannot express "transfer up to $100 from account X," so `/authorize`, `/par`,
and `/token` (including the token-exchange grant above) accept RFC 9396 `authorization_details`,
letting a request declare structured, bounded permissions instead of a broad scope. Only `type`s
pre-registered per tenant are accepted, and each detail is schema-validated fail-closed — an
unregistered type or a schema mismatch is rejected outright rather than partially accepted. Issued
and exchanged tokens may carry only a subset of what the user consented to, and — composing with
token-exchange delegation above — a subsequent exchange may only narrow that subset further, never
widen it; the partial order used for that check is defined by the registered schema itself
(containment of targets, monotonic decrease of limits). The consent UI renders each detail from a
schema-linked, human-readable template rather than raw JSON, and resource servers treat the
IdP-issued/introspected details as the sole trust boundary — they must not reinterpret or expand
what was granted. Where a `type` and a coarse `scope` overlap the same area, the structured detail's
bound wins; a request that would let `scope` re-widen an area already bounded by
`authorization_details` is rejected.

## OIDC session binding and logout propagation

The `sid` claim is `LoginSession.id` itself — one value shared across every relying party for a
given browser session, not a per-RP value — because OIDC's `sid` semantics describe the OP session,
and a per-RP `sid` would make it impossible to walk from a single session revoke to every affected
RP. `sid` propagates once, at `authenticate_user` completion, into `AuthorizationRequest`, then straight
through `AuthorizationCodeRecord` → `RefreshTokenRecord` → `IdTokenClaims`; Authentication's
`LoginSession` stays the single source of truth and none of its attributes are duplicated into
OAuth2. `ClientSession` exists purely as a `(sid, client_id)` delivery index for logout
notification, not a second copy of session state. Because `RefreshTokenRecord.sid` survives
rotation, revoking "this browser session" can revoke every refresh token across every client/family
tied to that `sid` in one operation, rather than requiring a family-by-family walk.

`id_token_hint` on `/end_session` is verified fail-closed — signature, `iss`, `aud` (must agree with
an explicit `client_id` parameter rather than being silently ignored), `sub`, and `sid` — with `exp`
deliberately not checked, since an expired ID Token at logout time is the normal case for RPs.
Without a hint, resolution falls back to `client_id` plus the browser cookie. Back-channel logout
delivery is handed to the Jobs context as a durable, idempotent job rather than a bespoke queue, and
local session/refresh-token revocation is never rolled back if delivery fails; front-channel logout
is a same-request computed iframe-target list with no delivery guarantee, since RP-side iframe
failures are an accepted, unrecoverable-by-design condition. Access token revocation is explicitly
out of scope here: access tokens stay self-contained JWTs verified by signature alone, and immediate
revocation would require making every resource-server verification a store lookup — the 600-second
residual exposure is accepted instead, on top of immediate refresh-token-family revocation and RP
notification. `check_session_iframe` (OIDC Session Management 1.0) is implemented minimally —
discovery advertisement plus a static "is the browser cookie still a valid session" check — because
the underlying spec never reached Final status and major IdPs implement it inconsistently.

## Conventions

Protocol-critical behavior that has an unambiguous, enumerable shape is declared once in `spec/`
and consumed by usecases/adapters rather than re-implemented: state transitions, authorization
rules, discovery metadata, and device-flow transitions all follow this shape, so regenerating the
adapter layer cannot silently diverge from the specification. Feature slices
(`authorization/`, `client/`, `consent/`, `device/`, `token/`) each own their `domain`/`ports`/
`usecases`/`db_memory`/`db_postgres` layers; the context-level `domain`/`ports`/`usecases` packages
are compatibility facades over the slices, and `module.go` is the sole composition root.

## Design Decisions

- Authorization request and device-code lifecycles are expressed as declarative state-transition
  tables in `spec/flows/` rather than ad hoc conditional logic, so regeneration cannot silently
  drift the transitions a client is allowed to make
  ([ADR-001](../../decisions/ADR-001-state-machine-as-spec.md)).
- PKCE requirement is staged per client type — required by default for public and FAPI 2.0 clients,
  opt-in for legacy confidential clients — rather than mandated uniformly
  ([ADR-002](../../decisions/ADR-002-pkce-required-for-all-clients.md)).
- Pushed Authorization Requests are mandatory for FAPI 2.0 clients and optional for everyone else,
  closing off URL tampering and unauthenticated request forgery at `/authorize` for the clients that
  need the strongest guarantee
  ([ADR-006](../../decisions/ADR-006-par-mandatory-fapi-clients.md)).
- Five client authentication methods are supported, spanning FAPI-grade asymmetric authentication
  down to legacy shared-secret methods, with `client_secret_jwt` deliberately excluded
  ([ADR-008](../../decisions/ADR-008-client-authentication-methods.md)).
- `private_key_jwt` verification is pinned to a fixed rule set — algorithm allowlist, issuer/
  subject/audience checks, bounded assertion lifetime, replay protection — so what Discovery
  advertises is what the server actually enforces
  ([ADR-023](../../decisions/ADR-023-private-key-jwt-verification.md)).
- Client ID Metadata Documents are supported as a non-persisted, registry-less alternative to
  Dynamic Client Registration for resolving `client_id`s shaped as HTTPS URLs
  ([ADR-155](../../decisions/ADR-155-client-id-metadata-documents.md)).
- Access tokens are issued as self-contained JWTs by default while refresh tokens stay opaque,
  database-backed references, since the two need different revocation and verification-scaling
  properties ([ADR-012](../../decisions/ADR-012-opaque-vs-jwt-access-tokens.md)).
- Refresh tokens rotate on every use, and presenting an already-rotated token revokes the entire
  token family, uniformly for public and confidential clients
  ([ADR-004](../../decisions/ADR-004-refresh-token-rotation.md)).
- DPoP is the default sender-constraint mechanism, with mTLS offered as an option for clients that
  already run client PKI ([ADR-005](../../decisions/ADR-005-dpop-as-default-sender-constraint.md)).
- Consent is persisted per `(subject, client_id)` as a set of granted scopes, not per-client and not
  per-interaction, to avoid silent scope creep and consent fatigue
  ([ADR-007](../../decisions/ADR-007-consent-model.md)).
- Authorization decisions are declared as policy in `spec/policy/client-authorization.json` and
  evaluated through an AuthZEN-style `authorize()` interface rather than scattered as inline
  conditionals ([ADR-010](../../decisions/ADR-010-authzen-policy-as-spec.md)).
- The Discovery document is generated at runtime from `spec/discovery.json` rather than
  hand-maintained or build-time generated, so it cannot drift from the implementation
  ([ADR-011](../../decisions/ADR-011-discovery-as-derived-artifact.md)).
- The Device Authorization Grant reuses the state-transition table already declared in
  `spec/flows/device-code-flow.json` rather than reimplementing approve/deny/exchange transitions
  ad hoc ([ADR-025](../../decisions/ADR-025-device-authorization-grant.md)).
- Protocol timing and security parameters (token/code/PAR TTLs, rate limits, DPoP replay window,
  consent retention) are kept together in one place rather than modeled as SCL `objectives`, since
  they are protocol/security settings, not availability SLOs
  ([ADR-109](../../decisions/ADR-109-oauth2-lifetime-security-and-retention-policy-configuration.md)).
- `Agent` is a first-class principal that owns identity and lifecycle but holds no credentials of
  its own, binding instead to existing `OAuth2Client` registrations
  ([ADR-048](../../decisions/ADR-048-agent-as-first-class-non-human-principal.md)).
- Acting on a user's behalf is implemented as OAuth 2.0 Token Exchange, defaulting to delegation
  (original `sub`, agent recorded in `act`) rather than impersonation
  ([ADR-049](../../decisions/ADR-049-token-exchange-delegation-and-actor-chain.md)).
- Agent-scoped permissions are expressed as RFC 9396 `authorization_details` rather than coarse
  scopes, so bounds like a transfer limit can be declared and only ever narrowed, never widened, on
  a subsequent token exchange
  ([ADR-050](../../decisions/ADR-050-rich-authorization-requests-for-agent-scopes.md)).
- The `sid` claim is `LoginSession.id` itself, shared across every relying party for a browser
  session, so a single session revoke can be walked to every affected RP
  ([ADR-127](../../decisions/ADR-127-oidc-session-binding-and-logout-propagation.md)).
