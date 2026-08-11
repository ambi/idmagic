# Infrastructure Guide

This guide covers deployment, monitoring, and security configurations for IdMagic.

## Kubernetes, Monitoring, and Load Smoke

The Kubernetes base separates the API, UI gateway, and durable
job worker into independent Deployments, linked by PostgreSQL and a shared OTLP. The worker is split into one Deployment per
execution lane (`idmagic-worker-{latency-sensitive,default,bulk}`),
each with its own metrics-only Service (`/metrics`, no application HTTP
surface). Apply a rendered environment only after your platform has created
the referenced Secrets (`idmagic-<environment>-runtime-secrets`,
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
PostgreSQL egress.
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

## Design

## Tenant Subdomain Routing

Set `TENANT_BASE_DOMAIN` only after the ingress has wildcard DNS and a wildcard TLS certificate for `*.${TENANT_BASE_DOMAIN}`. A tenant configured with endpoint style `subdomain` is reachable exclusively at `{realm}.${TENANT_BASE_DOMAIN}`; path-style tenants remain exclusively under `/realms/{realm}`. The application returns `Vary: Host` for tenant responses, so any CDN or reverse proxy must retain the host in its cache key. Certificate issuance and renewal remain platform responsibilities.

Changing endpoint style changes the issuer, cookie scope, and WebAuthn RP ID. Coordinate RP metadata changes and passkey re-enrollment before switching it in the system tenant console.

The cross-cutting runtime design these assets implement — high availability and shared state, request
correlation, the metrics contract, HTTP server hardening, and security response headers — is recorded in
the repository design record, not here. See
[`spec/SPECIFICATION.md`](../spec/SPECIFICATION.md) `## Design`.
This file keeps the commands and configuration steps for running the stack (ADR-143).
