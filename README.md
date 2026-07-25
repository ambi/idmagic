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
- **High Availability & Scalability**: Built for scale with shared state in PostgreSQL, Kafka outbox relays, robust distributed job processing, and native OpenTelemetry integration.
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

The compose stack starts PostgreSQL, Redpanda/Kafka, OpenTelemetry Collector, Prometheus, the Go API, the UI gateway, and the outbox relay. Caddy exposes the combined app at <http://localhost:8080/>. 

```bash
just dev-compose
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
| `EVENT_SINK` | `console`, `outbox` | domain event destination |
| `KAFKA_BROKERS` | broker list | outbox relay broker list |
| `OBSERVABILITY` | `noop`, `otel` | OpenTelemetry tracing/metrics |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | endpoint URL | OTLP/HTTP collector endpoint |
| `EMAIL_SENDER` | `console`, `smtp` | password reset and notification delivery |
| `DEFAULT_LOCALE` | `ja`, `en` | last resort language for notification emails; defaults to `en`. An unsupported value fails startup |
| `KEY_PROVIDER` | `local`, `vault` | signing key provider |
| `VAULT_ADDR`, `VAULT_TOKEN` | Vault configuration | Vault Transit configuration |
| `BREACHED_PASSWORD_CHECKER` | `noop`, `hibp` | breached password checker |
| `REQUEST_ID_TRUST_INBOUND` | `false`, `true` | reuse an edge proxy's inbound `X-Request-ID` |
| `HSTS_ENABLED` | `false`, `true` | emit `Strict-Transport-Security` |
| `HSTS_MAX_AGE_SECONDS` | `31536000` | HSTS `max-age` when enabled |
| `HSTS_INCLUDE_SUBDOMAINS` | `true`, `false` | add `includeSubDomains` to HSTS |
| `CSP_REPORT_ONLY` | `false`, `true` | send CSP as `Content-Security-Policy-Report-Only` for staged rollout |
| `CSP_REPORT_URI` | URL/path | CSP `report-uri` for violation collection |
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

### WebAuthn Configuration Notes

WebAuthn binds passkeys to the browser origin and relying-party ID. Non-local deployments must use HTTPS and set `WEBAUTHN_RP_ID` to the registrable domain that users visit. `WEBAUTHN_RP_ORIGINS` must include every public origin used by the UI.

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

## Documentation Guide

For deep dives into specific areas, consult the following guides:

- **Product Specification**: [spec/scl.yaml](spec/scl.yaml)
- **Implementation Index**: [ARCHITECTURE.md](ARCHITECTURE.md)
- **Infrastructure & K8s Guide**: [infra/README.md](infra/README.md)
- **Seed Profiles Guide**: [seed/README.md](seed/README.md)
- **UI Design & Localization**: [frontend/README.md](frontend/README.md)
- **PostgreSQL Workflow**: [infra/schema/README.md](infra/schema/README.md)
- **Architecture Decisions**: [decisions/](decisions/)
