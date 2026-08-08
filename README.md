# IdMagic

**A production-ready, enterprise-grade identity provider.**

IdMagic is a Go-based Identity Provider delivering robust implementations of OAuth 2.0,
OpenID Connect, SAML, WS-Federation, tenant isolation, application portals, and
identity administration. It is built with Regenerative Architecture practices:
the durable product model lives in SCL, architectural reasoning lives in ADRs,
and implementation is kept close to bounded contexts.

The project is built to serve as a highly reliable real-world identity platform
capable of handling complex enterprise authentication flows.

## Key Capabilities

- **Comprehensive Identity Protocols**: Full-featured OAuth 2.0 and OpenID Connect authorization server including PKCE, PAR, device flow, DPoP, dynamic client registration, and token rotation.
- **Enterprise Federation**: Out-of-the-box support for SAML 2.0 IdP, WS-Federation passive profile, WS-Trust STS, and Microsoft Entra domain federation presets.
- **Multi-Tenant Architecture**: Deep tenant isolation with realm-scoped routes, per-tenant signing keys (auto-rotated), tenant-specific application catalogs, and customizable branded portals.
- **High Availability & Scalability**: Built for scale with shared state in PostgreSQL, robust distributed job processing, and native OpenTelemetry integration.
- **Modern Administration Experience**: High-performance React-based admin console and account portals powered by Vite, Tailwind CSS, and Radix UI.

## Architecture & Repository Map

IdMagic follows Regenerative Architecture. The main bounded contexts are `tenancy`, `idmanagement`, `authentication`, `oauth2`, `application`, `wsfederation`, and `saml`. Shared adapter code lives under `backend/shared`; runtime composition lives in `backend/bootstrap`.

| Layer | Location |
| --- | --- |
| Specification Core | `spec/scl.yaml`, `spec/contexts/*.yaml` |
| Decisions | `decisions/*.md` |
| Application logic | `backend/<context>/domain`, `backend/<context>/usecases` |
| Core and adapters | `backend/<context>/{domain,usecases,ports}`, `backend/<context>/{handlers_http,db_postgres,...}` |
| Runtime and infra | `backend/cmd/`, `backend/bootstrap`, `infra/`, `frontend/` |

`infra/schema/postgres.sql` is the declarative current-state schema. The app does not run migrations on startup; deployment applies schema changes with `psqldef`.

## Getting Started & Development

### Local Quick Start

Run the Docker-free local stack with embedded PostgreSQL, the API, worker, and UI:

```bash
just dev
```

The first run downloads and caches an embedded PostgreSQL binary (about 190 MB).
Development data is temporary and is removed when the stack stops. The API and
worker remain separate processes and share the PostgreSQL job queue, so durable
jobs work in this mode. The local endpoint is `127.0.0.1:55432` (PostgreSQL).

For the smallest API + UI loop, without durable jobs or the background worker:

```bash
just dev-memory
```

Open <http://localhost:5173/> and choose the local demo authentication entry.
Use:
- **`alice`** (password: `demo-password-1234`): tenant admin demo user
- **`root`** (password: `demo-password-1234`): tenant admin + system admin

*(Note: Do not open `/login` directly. The login screen expects an active authorization transaction.)*

### Docker Development Stack

The compose stack starts PostgreSQL, OpenTelemetry Collector, Prometheus, the Go API, and the UI gateway. Caddy exposes the combined app at <http://localhost:8080/>.

```bash
just dev-compose       # start, detached
just logs-compose      # follow logs
just down-compose      # stop and remove
```

### Manual Local Run

If you prefer separate terminals, start a shared PostgreSQL yourself and
provide `PERSISTENCE=postgres` and `DATABASE_URL` to both the API and worker.
`just dev-api` by itself continues to use memory mode.

```bash
# Terminal 1: Go API
WEBAUTHN_RP_ID=localhost \
WEBAUTHN_RP_ORIGINS=http://localhost:5173 \
ADDR=:8081 \
ISSUER=http://localhost:5173 \
just dev-api

# Terminal 2: React UI
just dev-ui
```

### Common Commands

This repository uses `just` as the command map. Useful commands include:

```bash
just --list
just setup
just verify
just dev
just verify-go
just verify-ui
just test-ui-e2e
```

### Build and Versioning

IdMagic supports injecting build version metadata at build time using Go `-ldflags`.

```bash
VERSION=1.0.0 just build-go
```

If `VERSION` is not specified, it defaults to `0.0.0-dev`. For Docker builds, you can pass version metadata as build arguments (`VERSION`, `GIT_COMMIT`, `BUILD_DATE`).

## Configuration

Local defaults use in-memory persistence and console email output. Production adapters are selected with environment variables:

| Variable | Values | Purpose |
| --- | --- | --- |
| `PERSISTENCE` | `memory`, `postgres` | storage backend |
| `DATABASE_URL` | connection string | PostgreSQL database connection |
| `OBSERVABILITY` | `noop`, `otel` | OpenTelemetry tracing/metrics |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | endpoint URL | OTLP/HTTP collector endpoint |
| `EMAIL_SENDER` | `console`, `smtp` | password reset and notification delivery |
| `DEFAULT_LOCALE` | `ja`, `en` | last resort language for notification emails; defaults to `en`. An unsupported value fails startup |
| `KEY_PROVIDER` | `local`, `vault` | signing key provider |
| `VAULT_ADDR`, `VAULT_TOKEN` | Vault configuration | Vault Transit configuration |
| `DATA_KEY_PROVIDER` | unset, `openbao` | master-key custody for envelope-encrypted reversible secrets (MFA TOTP seeds today); unset uses an in-process Tink cleartext keyset (dev/local only) |
| `OPENBAO_ADDR`, `OPENBAO_TOKEN` | OpenBao configuration | required when `DATA_KEY_PROVIDER=openbao`; startup fails fast if either is empty |
| `OPENBAO_TRANSIT_MOUNT` | mount path, default `transit` | OpenBao Transit engine mount |
| `OPENBAO_DATA_KEY_PREFIX` | key name prefix, default `idmagic/datakeys` | per-tenant OpenBao Transit key naming (`{prefix}/{tenant_id}`) |
| `BREACHED_PASSWORD_CHECKER` | `noop`, `hibp` | breached password checker |
| `REQUEST_ID_TRUST_INBOUND` | `false`, `true` | reuse an edge proxy's inbound `X-Request-ID` |
| `HSTS_ENABLED` | `false`, `true` | emit `Strict-Transport-Security` |
| `HSTS_MAX_AGE_SECONDS` | `31536000` | HSTS `max-age` when enabled |
| `HSTS_INCLUDE_SUBDOMAINS` | `true`, `false` | add `includeSubDomains` to HSTS |
| `CSP_REPORT_ONLY` | `false`, `true` | send CSP as `Content-Security-Policy-Report-Only` for staged rollout |
| `CSP_REPORT_URI` | URL/path | CSP `report-uri` for violation collection |
| `TENANT_BASE_DOMAIN` | parent hostname, e.g. `id.example.com` | enables the `subdomain` endpoint style at `{realm}.<domain>`; unset keeps every tenant on path routing |
| `WEBAUTHN_RP_ID` | domain, e.g. `localhost` | WebAuthn relying-party ID; WebAuthn/passkeys are disabled when unset |
| `WEBAUTHN_RP_ORIGINS` | comma-separated origins | Allowed browser origins for WebAuthn ceremonies, e.g. `http://localhost:5173` |
| `WEBAUTHN_RP_DISPLAY_NAME` | display name | WebAuthn relying-party display name shown by authenticators |
| `SEED_PROFILE` | `bootstrap`, `development`, `test`, `performance` | explicit startup seed profile; unset by default |
| `SEED_ENVIRONMENT` | `development`, `test`, `staging`, `production` | required when `SEED_PROFILE` is set |
| `SEED_MANIFEST` | local YAML path | optional root manifest; defaults to `seed/manifests/<profile>.yaml` |
| `SEED_SECRET_ROOT` | local directory | root for relative `file` secret locators |
| `SEED_FIRST_PARTY_REDIRECT_URIS` | comma-separated HTTPS URIs | required for production bootstrap first-party clients |

### Job Worker & Scheduled Batches

Production splits `idmagic-worker` into one Deployment per lane using the `JOB_WORKER_LANES` variable (e.g. `latency_sensitive`, `default`, `bulk`).

`idmagic-batch` executes one operational batch and exits. External schedulers run `retention-sweep` hourly and `signing-key-lifecycle` daily; neither task is coupled to the horizontally scaled durable-job worker. 

### Envelope Encryption & Data Keys

Reversible secrets that must remain in the app DB (MFA TOTP seeds today) are envelope-encrypted at rest
(ADR-148, [ARCHITECTURE.md](ARCHITECTURE.md#3-envelope-encryption-for-reversible-secrets)): a per-tenant
`DataEncryptionKey` (DEK) directly encrypts each secret, and a master key — held by the swappable
`DATA_KEY_PROVIDER` — wraps that DEK.

- **Dev fallback**: leaving `DATA_KEY_PROVIDER` unset uses an in-process Tink cleartext keyset, so no
  external service is required to develop. This must never be selected in production.
- **Losing the master key is unrecoverable.** If OpenBao's Transit keys are lost without a backup, every
  tenant's wrapped DEKs become permanently unwrappable — this is the same crypto-shredding property that
  makes `DestroyTenantDataKey` deliberate, but applied by accident. Back up OpenBao's Transit engine
  storage using OpenBao's own backup mechanism; a PostgreSQL backup of `tenant_data_encryption_keys` alone
  is not sufficient to recover, since it only holds the wrapped (master-key-encrypted) form.
- **Key health**: `GET /api/admin/data-keys/health` (`system_admin` only) reports each tenant's active DEK
  version/status and the configured provider's name/reachability, without ever returning key material.
- **Rotation and backfill**: rotating a tenant's DEK (internal-only today, no admin endpoint yet) enqueues
  a resumable `data_key_reencryption` job (`backend/jobs`) that migrates every reference onto the new
  version; only once that job reports nothing pending can the old version be destroyed. To backfill
  legacy plaintext rows written before this migration (or to catch up any tenant whose auto-enqueued job
  failed), run `idmagic-batch data-key-reencryption-sweep` — it is idempotent and safe to re-run or put on
  a cadence.

### WebAuthn Configuration Notes

WebAuthn binds passkeys to the browser origin and relying-party ID. Non-local deployments must use HTTPS and set `WEBAUTHN_RP_ID` to the registrable domain that users visit. `WEBAUTHN_RP_ORIGINS` must include every public origin used by the UI.

### Upstream OIDC Identity Provider

Tenant administrators configure inbound OIDC and SAML connections under **Settings → External identity
providers**. An OIDC connection needs a fixed HTTPS issuer and its last-known-good authorization,
token, and JWKS endpoints. The callback URI registered at the upstream provider is:

```text
https://<idmagic-origin>/realms/<realm>/api/auth/federation/oidc/callback
```

Client secrets are not stored in the IdMagic database. Put the value in the API process environment and
save only an `env:` reference in the connection:

```bash
CONTOSO_CLIENT_SECRET=replace-with-the-provider-secret
```

Example connection input:

```json
{
  "display_name": "Contoso Workforce",
  "protocol": "oidc",
  "issuer": "https://login.contoso.example",
  "client_id": "idmagic-production",
  "secret_reference": "env:CONTOSO_CLIENT_SECRET",
  "authorization_endpoint": "https://login.contoso.example/oauth2/authorize",
  "token_endpoint": "https://login.contoso.example/oauth2/token",
  "jwks_uri": "https://login.contoso.example/oauth2/jwks",
  "claim_mapping": {
    "subject": "sub",
    "username": "preferred_username",
    "email": "email",
    "email_verified": "email_verified",
    "name": "name"
  },
  "linking_policy": "none",
  "jit_provisioning": false
}
```

Test the draft connection before activating it. Login requests use only the saved provider and endpoint
configuration; a browser request cannot supply an arbitrary discovery or token URL.

JIT provisioning is disabled by default. When enabled, the first accepted upstream identity creates an
active local user without a password credential. Use `allowed_email_domains` to narrow who can be
created. Automatic linking by verified email is a separate opt-in policy: it requires the upstream
`email_verified` claim and exactly one matching verified local email. Keep it disabled unless the
upstream provider's email-verification and account-recovery guarantees are trusted. External tokens and
SAML assertions are validated for the login and are never retained.

### Tenant Endpoint Styles

Each tenant has exactly one canonical location and issuer. The default `path` style is served at `{ISSUER}/realms/{realm}` and needs no wildcard DNS or certificate. `subdomain` is served at `https://{realm}.{TENANT_BASE_DOMAIN}` and is available only when `TENANT_BASE_DOMAIN` is configured. The ingress layer must provide wildcard DNS and a matching wildcard TLS certificate; IdMagic does not issue or renew certificates.

Changing a tenant from `path` to `subdomain` (or back) is disruptive: its issuer and protocol metadata URLs change, relying parties must be reconfigured, existing passkeys must be re-enrolled, and active browser sessions end. Plan the change as an identity migration. The current work covers IdMagic-managed subdomains only; customer-owned domains are separate work.

### Notification Email Templates

Notification emails come from a template catalog rather than being composed in code. Each message resolves in two steps: the bundled default wording for the chosen language, overridden by a per-tenant customization when one exists. Every message is sent as `multipart/alternative` with both a plain-text and an HTML part.

Template keys:

| Key | Sent when | Placeholders (in addition to `product_name`, `tenant_display_name`, `user_display_name`) |
| --- | --- | --- |
| `password_reset` | a user requests a password reset | `reset_url`, `expires_in_minutes` |
| `email_verification` | an address needs verification | `verification_url`, `expires_in_minutes` |
| `email_change_confirmation` | a user requests an email address change | `confirmation_url`, `expires_in_minutes`, `new_email` |
| `account_security_alert` | not emitted yet; the catalog entry exists so the wording can be prepared | `event_description`, `occurred_at` |
| `lifecycle_workflow_notification` | a lifecycle workflow runs a `send_email` action | `notification_key` |

Placeholders are written as `{{name}}`. Each key declares an allowed set, and the admin API returns that set alongside the template. A customization referencing anything outside the set is **rejected when saved**, not silently blanked at send time, so a template can never ship a message with a missing recovery link. Values substituted into the HTML body are escaped by the renderer; links are assembled by the server, never by the template.

Language resolution runs in three steps and picks the first language with a bundled translation: the recipient's `locale` user attribute, then the tenant's default language (Settings → General), then `DEFAULT_LOCALE` (default `en`). Bundled translations ship for `ja` and `en`.

A tenant may customize the subject, the plain-text body, the HTML body, and the sender display name. The subject and both bodies are saved as one set, so a half-overridden template cannot exist. The sender email address, the surrounding HTML document, and its base styling stay server-owned. Deleting a customization ("Reset to default") returns to the bundled wording; there is no version history.

Test messages sent from the template editor always go to the acting administrator's own verified email address. The recipient cannot be chosen, which keeps tenant administrator rights from becoming a relay for sending mail to arbitrary addresses.

Upgrade note: before this catalog existed, the three emails that already shipped (password reset, email change confirmation, and lifecycle workflow notifications) were hardcoded English plain text, and lifecycle notifications used the raw template key as both subject and body. Their subjects and bodies have changed.

### Local Email Testing (SMTP)

For SMTP testing during development, Mailpit works well:

```bash
mailpit --smtp 127.0.0.1:1025 --listen 127.0.0.1:8025

EMAIL_SENDER=smtp \
SMTP_HOST=127.0.0.1 \
SMTP_PORT=1025 \
SMTP_TLS=none \
SMTP_FROM=noreply@idmagic.test \
./dev.sh
```

## API Stability, Versioning & Deprecation

IdMagic's management API and self-service account API are external contracts: tenants build automation, provisioning, and IaC against them (see [ADR-137](decisions/ADR-137-unify-api-access-token-wire-format.md) for the API access token that authenticates these calls). [ADR-156](decisions/ADR-156-api-versioning-and-deprecation-policy.md) defines how they are versioned and deprecated; this section is the operational summary.

**Stability tiers.** Every SCL interface declares a `stability`:

- `stable` — a versioned external contract, covered by the compatibility guarantee below.
- `beta` — an external contract not yet covered by the compatibility guarantee; reserved for future endpoints.
- `internal` — not part of the external contract. This covers browser-only interactive flows (login, MFA enrollment, consent — anything reachable only by a first-party browser session, not by an API access token) and admin-console screens that currently have no API-access-token path. `internal` interfaces can change without notice.

An interface counts as `stable`/`beta` only if it is reachable by an API access token (`ManagementApiClient*`/`SelfApiClient*`/SCIM scopes), is a protocol endpoint governed by an external standard (OAuth2/OIDC, SAML, WS-Federation, SCIM, SSF — see below), or is an unauthenticated public asset/operational endpoint (health probes, metrics, branding assets).

**Compatibility definition.** Backward-compatible: adding a field, adding an optional parameter, adding a new endpoint. Breaking: removing or renaming a field, changing a field's type, making a field required, changing an error code, changing a default value. Error codes returned via `BackendErrorResponse` are part of the contract.

**Versioning.** The management (`/api/admin/v1/...`) and self-service (`/api/account/v1/...`) APIs are versioned by path. `/v1/` is the only path — there is no unversioned form. A breaking change is introduced as a new `/v2/` prefix, never by mutating an existing path. At most 2 versions are supported concurrently.

**Out of IdMagic's versioning scheme**: OAuth2/OIDC, SAML, WS-Federation, SCIM, and SharedSignals (SSF) protocol endpoints. Their compatibility and versioning is governed by the standards themselves; discovery documents (`/.well-known/...`, `/scim/v2/ServiceProviderConfig`, SAML/WS-Fed metadata) are the source of truth for those, not this scheme.

**Deprecation.** A deprecated interface is marked in SCL with `deprecated_since`, and once a removal date is set, `sunset_at` (minimum 12 months after `deprecated_since`) and `successor`. Responses from a deprecated interface carry a `Deprecation` header (and a `Sunset` header once `sunset_at` is set), added uniformly by `backend/shared/http/support_http.DeprecationHeadersMiddleware`. The same fields are reflected in the generated OpenAPI document as `deprecated: true`, `x-scl-deprecated-since`, `x-scl-sunset-at`, and `x-scl-successor`.

**Currently deprecated APIs** (machine-generated from `spec/idmagic.openapi.json`, itself generated from SCL — do not hand-maintain a separate list):

```bash
jq '[.paths[][] | select(.deprecated == true) | {operationId, "x-scl-deprecated-since", "x-scl-sunset-at", "x-scl-successor"}]' spec/idmagic.openapi.json
```

**Breaking-change detection.** `just check-api-compat` compares the generated `spec/idmagic.openapi.json` against the frozen release baseline `spec/idmagic.openapi.baseline.json` and fails on any breaking difference (field removal/rename, type change, new required field, changed default, removed error code, removed path/operation/parameter/response). It runs in CI (`.github/workflows/idmagic-ci.yaml`) and in `just verify`. **After cutting a release**, refresh the baseline so future PRs are compared against what actually shipped:

```bash
just scl-render
cp spec/idmagic.openapi.json spec/idmagic.openapi.baseline.json
```

Skipping this step lets the baseline go stale and the check stops catching real regressions; committing a baseline update without an actual release makes the check stop catching real ones too, so only refresh it as part of cutting a release.

## Documentation Guide

For deep dives into specific areas, consult the following guides:

- **Product Specification**: [spec/scl.yaml](spec/scl.yaml)
- **Implementation Index**: [ARCHITECTURE.md](ARCHITECTURE.md)
- **Infrastructure & K8s Guide**: [infra/README.md](infra/README.md)
- **Seed Profiles Guide**: [seed/README.md](seed/README.md)
- **UI Design & Localization**: [frontend/README.md](frontend/README.md)
- **PostgreSQL Workflow**: [infra/schema/README.md](infra/schema/README.md)
- **Architecture Decisions**: [decisions/](decisions/)
