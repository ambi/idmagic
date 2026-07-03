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
  management, and audit views.
- Adapter-oriented runtime: in-memory local mode, PostgreSQL, Valkey, Kafka
  outbox relay, OpenTelemetry, SMTP, AuthZEN, and Vault Transit signing.
- React admin/account/auth UI built with Vite, TanStack Router, Tailwind CSS,
  Radix UI, and local shadcn-style components.
- SCL-first documentation flow: canonical specification in `spec/`, ADRs in
  `decisions/`, and change records in `work-items/`.

## Quick Start

Run the API and UI together in local memory mode:

```bash
./dev.sh
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
ADDR=:8081 ISSUER=http://localhost:5173 go run ./cmd/idmagic

# Terminal 2: React UI
cd ui
bun install
bun run dev
```

## Docker Development Stack

The compose stack starts PostgreSQL, Valkey, Redpanda/Kafka, OpenTelemetry
Collector, the Go API, the UI gateway, and the outbox relay. Caddy exposes the
combined app at <http://localhost:8080/>.

```bash
docker compose -f deploy/docker/docker-compose.dev.yaml up --build
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

Useful direct checks:

```bash
go test ./...
go test -race ./...

cd ui
bun run lint
bun run typecheck
bun run build
bun run test:e2e
```

## Configuration

Local defaults use in-memory persistence and console email output. Production
adapters are selected with environment variables:

| Variable | Purpose |
| --- | --- |
| `PERSISTENCE=memory|postgres` | storage backend |
| `DATABASE_URL` | PostgreSQL connection string |
| `VALKEY_URL` | Valkey connection string for volatile state |
| `EVENT_SINK=console|outbox` | domain event destination |
| `KAFKA_BROKERS` | outbox relay broker list |
| `OBSERVABILITY=noop|otel` | OpenTelemetry tracing/metrics |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP/HTTP collector endpoint |
| `AUTHZEN=local|remote` / `AUTHZEN_URL` | authorization policy mode |
| `EMAIL_SENDER=console|smtp` | password reset and notification delivery |
| `KEY_PROVIDER=local|vault` | signing key provider |
| `VAULT_ADDR`, `VAULT_TOKEN` | Vault Transit configuration |
| `BREACHED_PASSWORD_CHECKER=noop|hibp` | breached password checker |
| `SKIP_DEMO_SEED=true` | disable demo seed data |

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

## Repository Map

```text
spec/             SCL source and generated specification artifacts
decisions/        Architecture Decision Records
work-items/       planned and completed change records
cmd/              process entry points
internal/         Go bounded contexts, use cases, ports, and adapters
ui/               React SPA for auth, account, admin, and system flows
deploy/           Docker, schema, and runtime infrastructure assets
```

The main bounded contexts are `tenancy`, `identitymanagement`,
`authentication`, `oauth2`, `application`, `wsfederation`, and `saml`.
Shared adapter code lives under `internal/shared`; runtime composition lives in
`internal/bootstrap`.

## Architecture

IdMagic follows Regenerative Architecture:

| Layer | Location |
| --- | --- |
| Specification Core | `spec/scl.yaml`, `spec/contexts/*.yaml` |
| Decisions | `decisions/*.md` |
| Application logic | `internal/<context>/domain`, `internal/<context>/usecases` |
| Ports and adapters | `internal/<context>/ports`, `internal/<context>/adapters`, `internal/shared/adapters` |
| Runtime and infrastructure | `cmd/`, `internal/bootstrap`, `deploy/`, `ui/` |

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
- UI design and test policy: [ui/README.md](ui/README.md)
- PostgreSQL workflow: [deploy/schema/README.md](deploy/schema/README.md)
- Architecture decisions: [decisions/](decisions/)
- Work items: [work-items/](work-items/)

When a change affects behavior, update SCL first, regenerate derived artifacts,
and keep README focused on the stable project overview.
