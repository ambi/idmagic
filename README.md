# IdMagic

**A compact identity provider for serious protocol experiments.**

IdMagic is a Go-based Identity Provider for experimenting with OAuth 2.0,
OpenID Connect, SAML, WS-Federation, tenant isolation, application portals, and
identity administration. It is built with Regenerative Architecture practices:
the durable product model lives in SCL, architectural reasoning lives in ADRs,
and implementation is kept close to bounded contexts.

The project is useful as a readable reference implementation, a protocol lab,
and a foundation for identity platform experiments.

## Highlights

- OAuth 2.0 / OpenID Connect authorization server with PKCE, PAR, device flow,
  token introspection, revocation, dynamic client registration, DPoP,
  `private_key_jwt`, refresh-token rotation, and userinfo.
- Enterprise federation surface: SAML 2.0 IdP, WS-Federation passive profile,
  WS-Trust username/mixed STS, federation metadata, and Microsoft Entra domain
  federation presets.
- Multi-tenant identity model with realm-scoped routes, per-tenant signing keys,
  admin console, account portal, groups, roles, application catalog, consent
  management, audit views, and per-tenant hosted login/account branding.
- Adapter-oriented runtime: in-memory local mode, PostgreSQL, Valkey, Kafka
  outbox relay, OpenTelemetry, SMTP, AuthZEN, and Vault Transit signing.
- React admin/account/auth UI built with Vite, TanStack Router, Tailwind CSS,
  Radix UI, and local shadcn-style components.
- SCL-first documentation flow: canonical specification in `spec/`, ADRs in
  `decisions/`, and change records in `work-items/`.

## Quick Start

Run the API and UI together in local memory mode:

```bash
just dev
```

Open <http://localhost:5173/> and choose the local demo authentication entry.
Use:

| User | Password | Notes |
| --- | --- | --- |
| `alice` | `demo-password-1234` | tenant admin demo user |
| `root` | `demo-password-1234` | tenant admin + system admin |

Do not open `/login` directly. The login screen expects an active
authorization transaction.

### Manual Local Run

If you prefer separate terminals:

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

## Docker Development Stack

The compose stack starts PostgreSQL, Valkey, Redpanda/Kafka, OpenTelemetry
Collector, the Go API, the UI gateway, and the outbox relay. Caddy exposes the
combined app at <http://localhost:8080/>.

```bash
just dev-compose
```

Re-apply only the declarative PostgreSQL schema:

```bash
docker compose -f deploy/docker/docker-compose.dev.yaml run --rm schema
```

Run the OAuth/OIDC demo script against the compose stack:

```bash
BASE=http://localhost:8080 ./demo.sh
```

## Common Commands

This repository uses `just` as the command map:

```bash
just --list
just setup
just verify
just dev-api
just dev-ui
just dev-compose
```

Useful checks:

```bash
just verify-go
just verify-ui
just test-ui-e2e
```

## Configuration

Local defaults use in-memory persistence and console email output. Production
adapters are selected with environment variables:

| Variable | Values | Purpose |
| --- | --- | --- |
| `PERSISTENCE` | `memory`, `postgres_valkey` | storage backend |
| `DATABASE_URL` | connection string | PostgreSQL database connection |
| `VALKEY_URL` | connection string | Valkey connection for volatile state |
| `EVENT_SINK` | `console`, `outbox` | domain event destination |
| `KAFKA_BROKERS` | broker list | outbox relay broker list |
| `OBSERVABILITY` | `noop`, `otel` | OpenTelemetry tracing/metrics |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | endpoint URL | OTLP/HTTP collector endpoint |
| `AUTHZEN` | `local`, `remote` | authorization policy mode |
| `AUTHZEN_URL` | remote policy service URL | remote authorization policy endpoint |
| `EMAIL_SENDER` | `console`, `smtp` | password reset and notification delivery |
| `KEY_PROVIDER` | `local`, `vault` | signing key provider |
| `VAULT_ADDR`, `VAULT_TOKEN` | Vault configuration | Vault Transit configuration |
| `BREACHED_PASSWORD_CHECKER` | `noop`, `hibp` | breached password checker |
| `REQUEST_ID_TRUST_INBOUND` | `false`, `true` | reuse an edge proxy's inbound `X-Request-ID` (see Request Correlation) |
| `HSTS_ENABLED` | `false`, `true` | emit `Strict-Transport-Security` (only when TLS is terminated at/ahead of this hop; see Security Response Headers) |
| `HSTS_MAX_AGE_SECONDS` | `31536000` | HSTS `max-age` when enabled |
| `HSTS_INCLUDE_SUBDOMAINS` | `true`, `false` | add `includeSubDomains` to HSTS |
| `CSP_REPORT_ONLY` | `false`, `true` | send CSP as `Content-Security-Policy-Report-Only` for staged rollout |
| `CSP_REPORT_URI` | URL/path | CSP `report-uri` for violation collection |
| `WEBAUTHN_RP_ID` | domain, e.g. `localhost` | WebAuthn relying-party ID; WebAuthn/passkeys are disabled when unset |
| `WEBAUTHN_RP_ORIGINS` | comma-separated origins | Allowed browser origins for WebAuthn ceremonies, e.g. `http://localhost:5173` |
| `WEBAUTHN_RP_DISPLAY_NAME` | display name | WebAuthn relying-party display name shown by authenticators |
| `SKIP_DEMO_SEED` | `true` | disable demo seed data |

### Tenant Branding

Tenant admins can customize the hosted login / consent / device / account
portal surfaces from Admin Console → Settings → Branding (ADR-096): product
name, logo, favicon, primary/accent colors, a support link, a legal link, and
footer text. This is intentionally a *safe subset*, not a general theming
system:

- Colors are limited to two `#rrggbb` tokens injected only as CSS custom
  properties (`--tenant-brand-primary` / `--tenant-brand-accent`); arbitrary
  CSS/HTML/JS is never accepted. Any value in that format can be saved; each
  tenant is responsible for choosing a readable color combination.
- Links (support / legal) accept `https://` only; other schemes
  (`javascript:`, `data:`, plain `http://`) are rejected.
- Logo/favicon uploads reuse the same validated-blob pipeline as Application
  icons (ADR-073): PNG/JPEG/WebP/GIF only, verified by magic byte (not file
  extension or declared content-type), 256 KiB max, no SVG. Delivery
  responses pin `Content-Type` and set `X-Content-Type-Options: nosniff`.
- An unconfigured or partially-configured tenant falls back to IdMagic's
  default branding; a branding read failure never blocks the login endpoint.
- The admin console's own UI chrome is unaffected — only the public-facing
  hosted surfaces (`GET /api/branding`, `GET
  /tenant-branding-assets/{kind}/{object_key}`) read tenant branding.
  Email-template branding and custom CSS/HTML injection are out of scope.

### WebAuthn / Passkeys

WebAuthn binds passkeys to the browser origin and relying-party ID. For local
development, `localhost` is allowed by browsers over plain HTTP; non-local
deployments must use HTTPS and set `WEBAUTHN_RP_ID` to the registrable domain
that users visit. `WEBAUTHN_RP_ORIGINS` must include every public origin used by
the UI, such as `https://login.example.com`. The Docker development stack sets
`WEBAUTHN_RP_ID=localhost` and `WEBAUTHN_RP_ORIGINS=http://localhost:8080`.

For SMTP testing, Mailpit works well:

```bash
mailpit --smtp 127.0.0.1:1025 --listen 127.0.0.1:8025

EMAIL_SENDER=smtp \
SMTP_HOST=127.0.0.1 \
SMTP_PORT=1025 \
SMTP_TLS=none \
SMTP_FROM=noreply@idmagic.test \
./dev.sh
```

Open Mailpit at <http://127.0.0.1:8025/>.

### High Availability & Shared State

Running more than one replica requires the `postgres_valkey` runtime with a shared
Valkey (`PERSISTENCE=postgres_valkey`, `DATABASE_URL`, `VALKEY_URL`). All ephemeral /
short-lived state is then kept in a store shared across replicas rather than in
per-replica process memory:

- **Valkey** holds the authorization request / authorization code / PAR / device
  code / login session / DPoP & client-assertion replay guards / access-token
  denylist, and the **login brute-force throttle**.
- **PostgreSQL** owns the durable shared state: refresh tokens, audit events, and
  auth-event aggregation buckets.

The login throttle in particular *must* be shared: with per-replica counters an
attacker's failed attempts split across `N` replicas, so the per-account /
per-IP lockout thresholds (ADR-029) would effectively loosen up to `N×`
cluster-wide — a silent security regression. On the shared Valkey they are
counted cluster-wide with atomic increments, and the account / IP identifiers are
SHA-256 hashed so no plaintext username or IP is stored (ADR-077).

Because the throttle is on the critical path, its degradation is **fail-closed**:
if the shared store is unreachable, a login attempt whose throttle state cannot be
verified is rejected rather than let through (it never fails open into an
un-throttled state). Run Valkey in a highly-available configuration
(replication / failover) for multi-replica deployments so this path stays up.

The `memory` runtime keeps this state in process and is therefore **single-replica
/ test only** — do not run multiple replicas against it.

### Request Correlation (`X-Request-ID`)

Every request is assigned a `request_id`. It is returned in the `X-Request-ID`
response header and attached to every application log line for the request
(alongside `trace_id` / `span_id` when `OBSERVABILITY=otel`), so a single request
can be correlated across logs and with a client report.

Correlation-id generation belongs at the edge. Because `X-Request-ID` is
attacker-controllable, IdMagic is **secure by default**: it self-generates the id
and ignores any inbound `X-Request-ID`, so a directly reachable client cannot
spoof or collide correlation ids. Choose one of two setups:

- **Trusted edge proxy owns the header.** If a proxy in front of IdMagic
  generates (and thereby sanitizes) `X-Request-ID` for external traffic, set
  `REQUEST_ID_TRUST_INBOUND=true` so that id flows into IdMagic's logs — giving a
  single id shared across the proxy and application tiers. Only enable this when
  the proxy actually sets/regenerates the header; a proxy that passes the client
  value through untouched must not be trusted. Examples:
  - Envoy / Istio regenerate `x-request-id` at the edge by default.
  - nginx (≥ 1.11.0): `proxy_set_header X-Request-ID $request_id;`
  - Caddy v2: `reverse_proxy` with `header_up X-Request-ID {http.request.uuid}`
- **No proxy, or a proxy that cannot set the header.** Leave
  `REQUEST_ID_TRUST_INBOUND=false` (the default); IdMagic generates its own id and
  the inbound value is ignored. No proxy header configuration is required.

Regardless of the setting, a reused inbound value is sanitized (bounded length,
restricted character set) as defense in depth against header/log injection.

### HTTP Server Hardening

The boundary HTTP server applies production-safe timeouts and a request body
limit so a single slow or oversized client cannot exhaust connections or memory
(`gosec G112` / CWE-400). Bodies over the limit are rejected with `413`. Defaults
are conservative and can be overridden per deployment:

| Variable | Default | Purpose |
| --- | --- | --- |
| `HTTP_READ_HEADER_TIMEOUT` | `10s` | max time to read request headers (slowloris bound) |
| `HTTP_READ_TIMEOUT` | `30s` | max time to read the full request |
| `HTTP_WRITE_TIMEOUT` | `60s` | max time to write the response |
| `HTTP_IDLE_TIMEOUT` | `120s` | keep-alive idle connection timeout |
| `HTTP_MAX_BODY_BYTES` | `1048576` | max request body size in bytes (1 MiB) |

This is defense in depth, not a substitute for an edge proxy. The **primary**
line against volumetric floods and TLS-handshake slowloris is the fronting
reverse proxy (Envoy / Nginx / Caddy / HAProxy), which sees total traffic and can
stop abuse cheaply at the edge. IdMagic still enforces its own timeouts and body
limit so it stays safe when run without a proxy, and so the proxy↔app hop and
any in-cluster direct access are covered. Tune the proxy's own timeouts and
connection limits alongside these values.

### Security Response Headers

Every backend response carries security headers applied by a boundary middleware
(ADR-076): `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`,
`X-Frame-Options: DENY`, and a strict `Content-Security-Policy`
(`default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'`).
`frame-ancestors 'none'` plus `X-Frame-Options: DENY` forbid framing so the
login / consent / portal surfaces cannot be clickjacked. The CSP does not use
`'unsafe-inline'`: the only inline script IdMagic renders is the fixed
auto-submit of the SAML ACS / WS-Fed POST-binding form, which is pinned by a
`script-src 'sha256-…'` hash on that response, and its `form-action` is narrowed
to the destination endpoint.

**Header ownership (app vs edge).** CSP and `frame-ancestors` require per-route
decisions and are owned by IdMagic so they hold even behind a minimal or absent
proxy (secure by default). The SPA is served by the gateway (see `frontend/Caddyfile`),
which sets its own `script-src 'self'` CSP for the static HTML — IdMagic's
middleware covers the backend responses the gateway reverse-proxies.

**HSTS is owned by the TLS terminator.** `Strict-Transport-Security` is off by
default so development over plain `http` is not poisoned. Enable it only when TLS
is terminated at or ahead of this hop:

- Terminating TLS at the edge proxy (typical): leave HSTS to the proxy, keep
  `HSTS_ENABLED=false`.
- Terminating TLS at/for the app, or wanting the app to assert it: set
  `HSTS_ENABLED=true` (tune `HSTS_MAX_AGE_SECONDS` / `HSTS_INCLUDE_SUBDOMAINS`).

**Staged rollout / reporting.** To tighten CSP without breaking a page, set
`CSP_REPORT_ONLY=true` to emit `Content-Security-Policy-Report-Only` and
`CSP_REPORT_URI=<url>` to collect violations, observe, then switch back to
enforce (`CSP_REPORT_ONLY=false`).

## Repository Map

```text
spec/             SCL source and generated specification artifacts
decisions/        Architecture Decision Records
work-items/       planned and completed change records
backend/cmd/              process entry points
backend/         Go bounded contexts, use cases, ports, and adapters
frontend/               React SPA for auth, account, admin, and system flows
deploy/           Docker, schema, and runtime infrastructure assets
```

The main bounded contexts are `tenancy`, `identitymanagement`,
`authentication`, `oauth2`, `application`, `wsfederation`, and `saml`.
Shared adapter code lives under `backend/shared`; runtime composition lives in
`backend/bootstrap`.

## Architecture

IdMagic follows Regenerative Architecture:

| Layer | Location |
| --- | --- |
| Specification Core | `spec/scl.yaml`, `spec/contexts/*.yaml` |
| Decisions | `decisions/*.md` |
| Application logic | `backend/<context>/domain`, `backend/<context>/usecases` |
| Ports and adapters | `backend/<context>/ports`, `backend/<context>/adapters`, `backend/shared/adapters` |
| Runtime and infrastructure | `backend/cmd/`, `backend/bootstrap`, `deploy/`, `frontend/` |

Start with [ARCHITECTURE.md](ARCHITECTURE.md) when changing code. It is the
small, stable index for navigating the implementation. Use the generated SCL
artifacts in `spec/` for the canonical product model, and ADRs for the reasoning
behind protocol and infrastructure choices.

## PostgreSQL Schema

`deploy/schema/postgres.sql` is the declarative current-state schema. The app
does not run migrations on startup; deployment applies schema changes with
`psqldef`.

```bash
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --dry-run < deploy/schema/postgres.sql

psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --apply < deploy/schema/postgres.sql
```

See [deploy/schema/README.md](deploy/schema/README.md) for the full workflow.

## Documentation Guide

- Product specification: [spec/scl.yaml](spec/scl.yaml)
- Implementation index: [ARCHITECTURE.md](ARCHITECTURE.md)
- UI design and test policy: [frontend/README.md](frontend/README.md)
- PostgreSQL workflow: [deploy/schema/README.md](deploy/schema/README.md)
- Architecture decisions: [decisions/](decisions/)
- Work items: [work-items/](work-items/)

When a change affects behavior, update SCL first, regenerate derived artifacts,
and keep README focused on the stable project overview.

## Build and Versioning

IdMagic supports injecting build version metadata at build time using Go `-ldflags`.

### Local Build via Just

You can inject a specific version when building locally by setting the `VERSION` environment variable:

```bash
VERSION=1.0.0 just build-go
```

If `VERSION` is not specified, it defaults to `0.0.0-dev`. The build process automatically extracts the current Git commit hash and build date, injecting them into the binary.

If the binary is built without `just` (e.g., direct `go build`), version metadata falls back to using Go's `runtime/debug.BuildInfo` (if VCS info is present in the build environment).

### Docker Build

You can pass the version metadata as build arguments:

```bash
docker build \
  --build-arg VERSION=1.0.0 \
  --build-arg GIT_COMMIT=$(git rev-parse HEAD) \
  --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  -t idmagic:1.0.0 -f deploy/docker/Dockerfile .
```

### Checking Active Version

The running version can be verified via:
1. **Startup Logs**: The server prints its version details on launch:
   `idmagic 1.0.0 (commit=..., date=...) listening on :8080`
