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
  `private_key_jwt`, refresh-token rotation, client-secret rotation, and userinfo.
- Enterprise federation surface: SAML 2.0 IdP, WS-Federation passive profile,
  WS-Trust username/mixed STS, federation metadata, and Microsoft Entra domain
  federation presets.
- Multi-tenant identity model with realm-scoped routes, per-tenant signing keys
  rotated every 90 days with a 7-day JWKS overlap,
  admin console, account portal, groups, roles, application catalog, consent
  management, audit views, and per-tenant hosted login/account branding.
- Adapter-oriented runtime: in-memory local mode, PostgreSQL, Valkey, Kafka
  outbox relay, OpenTelemetry, SMTP, AuthZEN, and Vault Transit signing.
- React admin/account/auth UI built with Vite, TanStack Router, Tailwind CSS,
  Radix UI, and local shadcn-style components.
- SCL-first documentation flow: canonical specification in `spec/`, ADRs in
  `decisions/`, and change records in `work-items/`.

## Quick Start

Run the Docker-free local stack with embedded PostgreSQL, a Valkey-compatible
development endpoint, the API, worker, and UI:

```bash
just dev
```

The first run downloads and caches an embedded PostgreSQL binary (about 190 MB).
Development data is temporary and is removed when the stack stops. The API and
worker remain separate processes and share the PostgreSQL job queue, so durable
jobs such as CSV user import work in this mode. The local endpoints are
`127.0.0.1:55432` (PostgreSQL) and `127.0.0.1:56379` (Valkey-compatible).

For the smallest API + UI loop, without durable jobs or the background worker:

```bash
just dev-memory
```

Open <http://localhost:5173/> and choose the local demo authentication entry.
Use:

| User | Password | Notes |
| --- | --- | --- |
| `alice` | `demo-password-1234` | tenant admin demo user |
| `root` | `demo-password-1234` | tenant admin + system admin |

Do not open `/login` directly. The login screen expects an active
authorization transaction.

### CSV user import

Tenant administrators can submit UTF-8 CSV files to `POST /api/admin/users/imports`
with `mode` set to `dry_run` or `apply`. The required header is
`preferred_username,email,name,roles`; roles are `|` separated. Files are limited
to 1 MiB, 1,000 rows, and 64 KiB per field. Password material is never accepted;
imported users are required to set a password on first sign-in. CSV jobs require
the standard `just dev` or `just dev-compose`; they are unavailable in
`just dev-memory`.

### Dynamic groups

Tenant administrators can create either manual groups or CEL-based dynamic groups.
Dynamic rules are boolean expressions over the `user` object, for example
`user.department == "Engineering" && user.email.endsWith("@example.com")`.
Attribute names come from the tenant user-attribute schema; string comparison is
case-sensitive unless the rule explicitly applies `lowerAscii()`. The supported
surface is intentionally restricted to boolean/comparison operators, list macros,
safe string helpers, timestamps, and bounded regular expressions.

Dynamic memberships are exclusive from manual membership changes. Rule edits and
enabling enqueue a full reconciliation job, while a single user's attribute or
lifecycle change is evaluated synchronously. Memberships carry the rule version;
stale versions never contribute roles or application assignments. Evaluation errors
fail closed, and an attribute referenced by a stored rule cannot be deleted or have
its type changed until the rule is updated.

### Manual Local Run

If you prefer separate terminals, start shared PostgreSQL/Valkey yourself and
provide `PERSISTENCE=postgres_valkey`, `DATABASE_URL`, and `VALKEY_URL` to both
the API and worker. `just dev-api` by itself continues to use memory mode.

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
Collector, Prometheus, the Go API, the UI gateway, and the outbox relay. Caddy
exposes the combined app at <http://localhost:8080/>. Prometheus scrapes the
Go API's `/metrics` directly on the compose network (`infra/docker/prometheus.yml`);
it is deliberately not proxied through Caddy — see
[Metrics (`/metrics`)](#metrics-metrics) below. Browse Prometheus at
<http://localhost:9090/>. `infra/docker/prometheus-rules.yml` (recording +
SLO burn-rate alert rules) and `infra/docker/grafana-dashboard.json` (a
minimal panel set) are built only from the metric catalog above; wi-11
(Kubernetes/monitoring assets) turns them into cluster
PrometheusRule/ServiceMonitor/dashboard provisioning.

```bash
just dev-compose
```

## Kubernetes, Monitoring, and Load Smoke

The Kubernetes base separates the API, UI gateway, outbox relay, and durable
job worker in `infra/k8s/base`; the relay has no HTTP surface and therefore
deliberately has no Service. The worker is split into one Deployment per
ADR-129 execution lane (`idmagic-worker-{latency-sensitive,default,bulk}`),
each with its own metrics-only Service (`/metrics`, no application HTTP
surface). Apply a rendered environment only after your platform has created
the referenced Secrets (`idmagic-<environment>-runtime-secrets`,
`idmagic-<environment>-relay-secrets`, and
`idmagic-<environment>-worker-secrets`). Secret values, image release digests,
and cloud-specific database endpoints are never stored in this repository.

```bash
just check-k8s dev
just deploy-k8s dev
```

Production starts at three API/UI replicas and two relay replicas. Replace the
zero digest placeholders in the production overlay through the release
pipeline, validate it, then apply it. Roll back by applying the preceding
release's digest overlay; Kubernetes keeps the prior ReplicaSet available for
an immediate `just rollback-k8s idmagic-api` when necessary.

The API probes `/startupz`, `/livez`, and `/readyz` directly. Its NetworkPolicy
allows only the UI gateway and Prometheus scrape traffic in, plus DNS,
PostgreSQL, and Valkey egress; relay egress is additionally limited to Kafka.
Each worker lane's NetworkPolicy allows only Prometheus scrape traffic in
(`/metrics`, port 8080), plus DNS, PostgreSQL, and Valkey egress — worker has
no readiness/liveness probes since it serves no application traffic.

`infra/k8s/monitoring` packages the same HTTP RED/authentication recording and
alert rules used by the Docker example, plus lane-scoped Jobs golden signals
(queue depth, claim latency, failure ratio, retry rate; wi-261 T006). It maps
`TokenLatency`, `TokenErrorRate`, `LoginLatency`, `LoginErrorRate`, and
availability evidence to the request-rate, error-rate, latency, login, and
token panels. Apply the `monitoring/operator` directory only when Prometheus
Operator is installed (its `idmagic-worker` `ServiceMonitor` covers all three
lane Services); otherwise configure Prometheus to scrape the `idmagic-api` and
`idmagic-worker-{latency-sensitive,default,bulk}` Services at `/metrics`.

```bash
just check-monitoring
just deploy-monitoring
just deploy-monitoring-operator # Prometheus Operator only
```

The k6 smoke covers authorization-code + S256 PKCE, refresh-token rotation,
and client credentials using one tenant-local seed fixture. Its default client
is the development seed's stable UUID; it does not create or reuse data across
tenants. Start a deliberately seeded development target first, then provide
only disposable fixture credentials through environment variables when defaults
do not apply:

```bash
just k6-smoke # default: http://host.docker.internal:8080/realms/default
# local `just dev-memory` API: just k6-smoke http://host.docker.internal:8081 http://localhost:5173
just check-k6
```

The smoke threshold is p99 token latency below 300 ms and an error ratio below
0.1%, derived from `OAuth2/objective/TokenLatency` and
`OAuth2/objective/TokenErrorRate`. CI should run the same recipe against its
isolated service URL after provisioning its fixture; it must not run against a
production tenant.

Re-apply only the declarative PostgreSQL schema:

```bash
docker compose -f infra/docker/docker-compose.dev.yaml run --rm schema
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
just dev
just dev-memory
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

## UI display languages

The hosted authentication, account, and admin UI support Japanese (`ja`) and English (`en`) only.
English is the product default. Set `VITE_DEFAULT_LOCALE=ja` or `VITE_DEFAULT_LOCALE=en` at
application startup to choose the final fallback when neither an explicit nor a browser locale is
available; an unset or invalid value falls back to English.
Add user-visible copy to the dictionary that is local to its feature (for example,
`frontend/src/features/auth-flow/LoginPage.i18n.ts`) and provide both locale values in
the same change. Use `defineDictionary` so TypeScript rejects missing or extra keys;
run `just verify-ui` before committing. Do not add another locale without a separately
specified product decision. Translate only stable backend error codes in the receiving
UI dictionary; render an unknown backend message unchanged, because backend error text
is intentionally English-only.

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
| `SEED_PROFILE` | `bootstrap`, `development`, `test`, `performance` | explicit startup seed profile; unset by default |
| `SEED_ENVIRONMENT` | `development`, `test`, `staging`, `production` | required when `SEED_PROFILE` is set |
| `SEED_MANIFEST` | local YAML path | optional root manifest; defaults to `seed/manifests/<profile>.yaml` |
| `SEED_SECRET_ROOT` | local directory | root for relative `file` secret locators |
| `SEED_FIRST_PARTY_REDIRECT_URIS` | comma-separated HTTPS URIs | required for production bootstrap first-party clients; localhost is rejected |
| `SEED_GENERATOR_SEED` | arbitrary string | deterministic namespace for performance seed identifiers |

### Durable job worker (`idmagic-worker`)

| Variable | Values | Purpose |
| --- | --- | --- |
| `WORKER_ID` | string | lease owner identifier; defaults to hostname |
| `JOB_WORKER_LANES` | comma-separated `latency_sensitive`, `default`, `bulk` | lanes this process claims (ADR-129); unset = compat mode (all lanes, one process — the `just dev`/docker-compose default) |
| `JOB_WORKER_CONCURRENCY` | integer, default `4` | fallback concurrency for any lane without a `_<LANE>` override |
| `JOB_WORKER_CONCURRENCY_LATENCY_SENSITIVE`, `_DEFAULT`, `_BULK` | integer | per-lane concurrency override, e.g. reserving `latency_sensitive` capacity independent of `bulk` |
| `JOB_POLL_INTERVAL` | duration, default `2s` | claim poll interval per lane `Runner` |
| `JOB_LEASE_DURATION` | duration, default `5m` | claim lease duration; heartbeat renews at 1/3 of this |
| `JOB_BACKOFF_BASE`, `JOB_BACKOFF_CAP` | duration, default `30s`/`30m` | retry backoff bounds (ADR-099) |
| `DRAIN_GRACE_PERIOD_SECONDS` | integer, default `5` | how long SIGTERM/SIGINT waits for in-flight jobs before exiting |

Production splits `idmagic-worker` into one Deployment per lane
(`infra/k8s/base/worker.yaml`), each with `JOB_WORKER_LANES` pinned to a
single lane so a lane's execution capacity is never consumed by another
lane's backlog. `latency_sensitive` additionally has a `PodDisruptionBudget`
(`infra/k8s/base/pdb.yaml`) keeping a reserved replica available through
voluntary disruptions.

### Scheduled batches

`idmagic-batch` executes one operational batch and exits. External schedulers
run `retention-sweep` hourly and `signing-key-lifecycle` daily; neither task is
coupled to the horizontally scaled durable-job worker. Signing keys rotate
after 90 days by default, remain in JWKS for a seven-day overlap, and are then
archived. The lifecycle subcommand accepts `--cadence-days` and `--grace-days`
and rejects invalid combinations before opening runtime dependencies.

### Environment seed profiles

Seed は server 起動時には既定で実行されない。plan/apply は `just seed <environment> <profile> <mode> [manifest]`
で明示する。`just dev` / `just dev-memory` だけは、ローカル利用のため development profile を明示して
同一プロセスで適用する。

profile の desired state は `seed/manifests/*.yaml` に置く。CLI の第 4 引数または
`SEED_MANIFEST` で別の root manifest を選べる。manifest は strict decode され、ローカル相対
include だけを許可する。未知 key、重複 logical key、循環、root 外 path、YAML anchor/alias/merge は
書き込み前に拒否される。

| Profile | Allowed environments | Contents |
| --- | --- | --- |
| `bootstrap` | development / test / staging / production | first-party client の最小設定。production では `SEED_FIRST_PARTY_REDIRECT_URIS` が必須。 |
| `development` | development / test / staging | local demo user、group、protocol sample、application。 |
| `test` | test | development と同じ決定的 fixture。 |
| `performance` | development / test / staging | 決定的 synthetic user。通常は 10,000 件まで、超過は `--allow-large` が必要。 |

`just seed development development dry_run` で変更計画を確認してから、末尾を `apply` にして投入する。
seed の出力は logical key と件数だけで、password・client secret・TOTP secret・hash・PII 全量を含まない。
秘密値は YAML に直接置かず、`provider` (`env` / `file`)、`locator`、`version` の参照として記述する。
staging/production は `file` provider のみ許可し、file locator は `SEED_SECRET_ROOT` 配下の regular file
に限定される。dry-run も参照の解決可能性を検証する。
旧 `SKIP_DEMO_SEED` は廃止されたため、起動時にデモ投入を止める設定は不要である。既存環境で
demo seed が既にある場合も、同じ development profile の apply は意味比較を行い、手動変更は conflict として保持する。

性能 profile は通常の検証に含めない。小さな件数を `just seed development performance apply` として使い、
計測時だけ `just seed-throughput development 10000 250` を実行する。10,000 件を超える場合は CLI の
`--allow-large` を明示し、100,000 件を超える値は拒否される。

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
  code / DPoP & client-assertion replay guards / access-token denylist, WebAuthn
  ceremony challenges, and the **login brute-force throttle** — all short-lived,
  retry-safe state.
- **PostgreSQL** owns the durable shared state: refresh tokens, audit events,
  auth-event aggregation buckets, and (since wi-253 / ADR-126) **login sessions**.
  A logged-in browser session is the single source of truth in `authentication_sessions`;
  losing Valkey no longer signs everyone out, and restarting or rolling API replicas
  does not invalidate active sessions. Revocation (self-service, logout, or an
  account being disabled) tombstones the row (`revoked_at`/`revoke_reason`) instead
  of deleting it, so a repeated revoke request is a safe no-op.

Because login sessions moved from Valkey to PostgreSQL, a deploy that switches an
existing `postgres_valkey` environment onto this schema **does not migrate
previously-issued Valkey sessions** — those browsers will be prompted to sign in
again once, but no existing PostgreSQL-durable state (refresh tokens, audit
history) is affected. New sessions issued after the cutover survive Valkey
restarts/evictions and API replica restarts.

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

### Metrics (`/metrics`)

`GET /metrics` exposes Prometheus/OpenMetrics-format metrics (the
`MetricsExposition` interface, `spec/contexts/system.yaml`): HTTP RED (request
count, error rate via `status_code`, duration, in-flight) for every route
template, plus authentication golden signals for SLO/alerting:

| Metric | Labels | Verifies |
| --- | --- | --- |
| `http_requests_total`, `http_request_duration_seconds`, `http_requests_in_flight` | `route`, `method`, `status_code` | per-interface latency/error-rate objectives (e.g. `oauth2.yaml` `TokenLatency`/`TokenErrorRate`, `authentication.yaml` `LoginLatency`/`LoginErrorRate`) |
| `authn_login_attempts_total` | `outcome`, `reason_class`, `method` | login success/failure golden signal (recorded once per confirmed decision, independent of audit-event aggregation under a credential-stuffing burst) |
| `authn_login_throttle_total` | `policy` (`account`/`ip`), `outcome` (`allowed`/`throttled`/`store_unavailable`) | login throttle hit rate |
| `oauth2_token_issuance_total`, `oauth2_token_issuance_duration_seconds` | `grant_type`, `outcome` | `/token` issuance rate/latency by grant |
| `http_request_aborts_total`, `operation_detached_completion_failures_total` | `kind` | `RequestFaultIsolation` / cancellation policy (ADR-074) |

Every label is a bounded, finite set (route templates, HTTP methods/status
codes, grant types, outcome/reason classes). `tenant_id`, `user_id`,
`client_id`, and resolved request paths are never labels — the endpoint is
scraped outside the tenant-resolution middleware and separated from the
application API for this reason. It is always registered but returns `503`
until the process finishes constructing its Prometheus registry at startup,
and works independently of `OBSERVABILITY` (OTLP tracing/push-metrics),
because a pull-based scrape needs no collector configured. Expose it only on
a loopback/management network, or in front of an authenticating proxy — never
on the public gateway.

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
infra/            Docker, schema, and runtime infrastructure assets
```

The main bounded contexts are `tenancy`, `idmanagement`,
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
| Runtime and infrastructure | `backend/cmd/`, `backend/bootstrap`, `infra/`, `frontend/` |

Start with [ARCHITECTURE.md](ARCHITECTURE.md) when changing code. It is the
small, stable index for navigating the implementation. Use the generated SCL
artifacts in `spec/` for the canonical product model, and ADRs for the reasoning
behind protocol and infrastructure choices.

## PostgreSQL Schema

`infra/schema/postgres.sql` is the declarative current-state schema. The app
does not run migrations on startup; deployment applies schema changes with
`psqldef`.

```bash
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --dry-run < infra/schema/postgres.sql

psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --apply < infra/schema/postgres.sql
```

See [infra/schema/README.md](infra/schema/README.md) for the full workflow.

## Documentation Guide

## Lifecycle workflow operations

Lifecycle workflow runs are delivered at least once through the durable Jobs queue. A changed or no-op step is checkpointed and is not performed again when a Job retries. When a run reaches `failed` or `partially_failed`, an administrator may retry it from the workflow run detail; only failed steps return to pending.

Use dry-run before enabling a changed definition. It evaluates the selected user without creating a WorkflowRun, Job, membership, assignment, required action, status change, or email. Treat its result as a point-in-time prediction, not a guarantee of a later run.

If a run fails, inspect the sanitized step error code and the affected resource. Correct the external dependency or definition, then retry the run. Disabling a workflow prevents new triggers and cancels queued runs; it does not undo already completed actions.

- Product specification: [spec/scl.yaml](spec/scl.yaml)
- Implementation index: [ARCHITECTURE.md](ARCHITECTURE.md)
- UI design and test policy: [frontend/README.md](frontend/README.md)
- PostgreSQL workflow: [infra/schema/README.md](infra/schema/README.md)
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
  -t idmagic:1.0.0 -f infra/docker/Dockerfile .
```

### Checking Active Version

The running version can be verified via:
1. **Startup Logs**: The server prints its version details on launch:
   `idmagic 1.0.0 (commit=..., date=...) listening on :8080`
