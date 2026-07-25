# Infrastructure Guide

This guide covers deployment, monitoring, and security configurations for IdMagic.

## Kubernetes, Monitoring, and Load Smoke

The Kubernetes base separates the API, UI gateway, outbox relay, and durable
job worker in `infra/k8s/base`; the relay has no HTTP surface and therefore
deliberately has no Service. The worker is split into one Deployment per
execution lane (`idmagic-worker-{latency-sensitive,default,bulk}`),
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
allows only the UI gateway and Prometheus scrape traffic in, plus DNS and
PostgreSQL egress; relay egress is additionally limited to Kafka.
Each worker lane's NetworkPolicy allows only Prometheus scrape traffic in
(`/metrics`, port 8080), plus DNS and PostgreSQL egress — worker has
no readiness/liveness probes since it serves no application traffic.

`infra/k8s/monitoring` packages the same HTTP RED/authentication recording and
alert rules used by the Docker example, plus lane-scoped Jobs golden signals
(queue depth, claim latency, failure ratio, retry rate). It maps
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

## High Availability & Shared State

Running more than one replica requires the `postgres` runtime
(`PERSISTENCE=postgres`, `DATABASE_URL`). All shared state — durable and ephemeral
alike — lives in PostgreSQL, shared across replicas rather than in per-replica
process memory:

- **Durable state**: refresh tokens, audit events, auth-event aggregation buckets,
  and **login sessions**. A logged-in browser session is
  the single source of truth in `authentication_sessions`; restarting or rolling
  API replicas does not invalidate active sessions. Revocation (self-service,
  logout, or an account being disabled) tombstones the row
  (`revoked_at`/`revoke_reason`) instead of deleting it, so a repeated revoke
  request is a safe no-op.
- **Ephemeral state** (short-lived auth/OAuth2 rows): authorization
  request / authorization code / PAR / device code / DPoP & client-assertion replay
  guards / access-token denylist, WebAuthn ceremony challenges, and the **login
  brute-force throttle** — all short-lived, retry-safe. Every row carries
  `expires_at` and every read filters `expires_at > now()`, so TTL correctness is
  independent of the best-effort GC sweep `idmagic-worker` runs to reclaim space.

A cutover that switches an environment onto this runtime abandons any **in-flight**
ephemeral state (a `/authorize` mid-flow, a pending PAR/device request, a throttle
counter); those simply restart and recover, and no durable state (refresh tokens,
audit history, login sessions) is affected.

The login throttle in particular *must* be shared: with per-replica counters an
attacker's failed attempts split across `N` replicas, so the per-account /
per-IP lockout thresholds would effectively loosen up to `N×`
cluster-wide — a silent security regression. In the shared PostgreSQL counter they
are counted cluster-wide with a serialized `SELECT ... FOR UPDATE` update, and the
account / IP identifiers are SHA-256 hashed so no plaintext username or IP is
stored.

Because the throttle is on the critical path, its degradation is **fail-closed**:
if the store is unreachable, a login attempt whose throttle state cannot be
verified is rejected rather than let through (it never fails open into an
un-throttled state). Run PostgreSQL in a highly-available configuration
(REGIONAL / synchronous standby) for multi-replica deployments so this path stays
up.

The `memory` runtime keeps this state in process and is therefore **single-replica
/ test only** — do not run multiple replicas against it.

## Request Correlation (`X-Request-ID`)

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

## Metrics (`/metrics`)

`GET /metrics` exposes Prometheus/OpenMetrics-format metrics: HTTP RED (request
count, error rate via `status_code`, duration, in-flight) for every route
template, plus authentication golden signals for SLO/alerting:

| Metric | Labels | Verifies |
| --- | --- | --- |
| `http_requests_total`, `http_request_duration_seconds`, `http_requests_in_flight` | `route`, `method`, `status_code` | per-interface latency/error-rate objectives |
| `authn_login_attempts_total` | `outcome`, `reason_class`, `method` | login success/failure golden signal |
| `authn_login_throttle_total` | `policy` (`account`/`ip`), `outcome` (`allowed`/`throttled`/`store_unavailable`) | login throttle hit rate |
| `oauth2_token_issuance_total`, `oauth2_token_issuance_duration_seconds` | `grant_type`, `outcome` | `/token` issuance rate/latency by grant |
| `http_request_aborts_total`, `operation_detached_completion_failures_total` | `kind` | cancellation policy |

Every label is a bounded, finite set. `tenant_id`, `user_id`,
`client_id`, and resolved request paths are never labels — the endpoint is
scraped outside the tenant-resolution middleware and separated from the
application API for this reason. It is always registered but returns `503`
until the process finishes constructing its Prometheus registry at startup,
and works independently of `OBSERVABILITY` (OTLP tracing/push-metrics),
because a pull-based scrape needs no collector configured. Expose it only on
a loopback/management network, or in front of an authenticating proxy — never
on the public gateway.

## HTTP Server Hardening

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

## Security Response Headers

Every backend response carries security headers applied by a boundary middleware:
`X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`,
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
proxy (secure by default). The SPA is served by the gateway,
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
