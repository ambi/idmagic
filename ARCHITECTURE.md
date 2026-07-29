---
context: repo
updated_at: 2026-07-29
---

# Architecture: repo

## Overview

This document is the cross-cutting design record for `idmagic`: how the system is currently built and why
it took that shape, in prose a human can read. The machine-checked module ledger lives beside it in
`architecture.yaml`; design that belongs to a single bounded context lives in that context's own
`ARCHITECTURE.md` ([ADR-143](decisions/ADR-143-second-layer-design-ledger-decision-split.md)).

Normative requirements are in SCL, rejected options and the premises they were weighed against are in
ADRs, and one-off implementation records are in work items. Design statements here carry a short reason
for their shape and link to the ADR that holds the full comparison. See
[Documentation Policy](#documentation-policy) for the routing table.

Lists that churn — endpoints, fields, screens — are not kept here. Code, `spec/contexts/*.yaml`, and the
UI documents are authoritative for those.

### Reading order

For a feature change, read in this order.

1. `spec/scl.yaml` `context_map`, to locate the bounded context and its dependencies.
2. That context's `spec/contexts/<context>.yaml`. Feature and behavior changes are SCL-first.
3. The context's `ARCHITECTURE.md` if it has one; otherwise the relevant section of this document.
4. The ADR, only when the background of a decision is needed. Search `decisions/` by filename rather
   than trusting an old work item's summary.
5. Go implementation, in the order `domain/`, `usecases/`, `ports/`, then whichever
   `<role>_<technology>/` adapter is involved.
6. `backend/shared/` and `backend/cmd/internal/bootstrap/` only when touching cross-cutting HTTP or
   persistence behavior.
7. `frontend/ARCHITECTURE.md` and `frontend/src/features/README.md` first when touching the UI.

Going the other way — implementation back to specification — package names correspond closely to SCL
context names. The exceptions are collected under `backend/shared/`.

## Structure

```text
.
├── architecture.yaml  # cross-cutting module ledger (machine-checked)
├── backend/           # Go bounded contexts, shared, cmd/
│   └── <context>/
│       ├── architecture.yaml   # this context's module ledger
│       └── ARCHITECTURE.md     # this context's design (only where one is warranted)
├── frontend/          # React UI and gateway
├── spec/              # SCL and derived contracts
├── infra/             # container, local runtime, and database schema assets
├── load/k6/           # tenant-local OAuth SLO smoke
├── tools/             # RA/SCL CLI, renderer, schema validator
├── verification/      # traceability manifest and revision-stamped evidence
├── decisions/         # Architecture Decision Records
└── work-items/        # units of work and completion records
```

Dependencies run from `spec` toward the implementations and derived artifacts. `backend` domain and
usecase packages never depend back on adapters or runtime.

### RA layer mapping

`idmagic` expresses the Regenerative Architecture rings as Go package boundaries.

| RA layer | Location | How to read it |
| --- | --- | --- |
| Specification Core | `spec/scl.yaml`, `spec/contexts/*.yaml` | Normative specification. Changes start here. |
| Decision Record | `decisions/*.md` | Rejected options and the premises at the time. |
| Architecture | `ARCHITECTURE.md`, `architecture.yaml` (root and per context) | The current design, and the ledger that is machine-checked against the tree. |
| Application Logic | `backend/<context>/domain`, `backend/<context>/usecases`, `backend/shared/spec` | Framework-independent domain, usecases, and SCL bindings. |
| Adapter Layer | `backend/<context>/{handlers_http,db_postgres,...}`, `backend/shared/<capability>/<role>_<technology>` | HTTP, persistence, crypto, policy, notification — the connections outward. |
| Runtime & Infrastructure | `backend/cmd/`, `backend/cmd/internal/bootstrap`, `infra/`, `frontend/`, `docker compose` | Startup, DI, delivery, process boundaries. |

`backend/shared/spec` holds Go bindings of SCL and their derived checks; it is not the specification
core itself. Do not adjust the Go binding in place of changing SCL.

## Stack

- Go, React/TypeScript, Bun, PostgreSQL, Docker Compose, Kubernetes, Prometheus, Grafana, Loki, Promtail,
  k6.
- Dynamic Group membership expressions over User attributes evaluate through a restricted CEL
  environment (`cel-go`). The environment is narrowed so unsafe expressions cannot be accepted, and a
  rule-version mismatch fails closed
  ([ADR-111](decisions/ADR-111-cel-dynamic-group-membership-rules.md)).

## Context Map

The main correspondence between SCL contexts and Go packages.

| SCL context | Go package | Responsibility |
| --- | --- | --- |
| `System` | `backend/cmd/internal/bootstrap`, `backend/shared/http/server_http`, `frontend/` | Cross-cutting UX, startup, routing composition, health. |
| `Tenancy` | `backend/tenancy` | Tenant / realm, tenant-scoped settings, user attribute schema, control-plane tenant administration. |
| `IdManagement` | `backend/idmanagement` | User, Group, Agent, self profile, identity lifecycle, CEL dynamic membership rules and re-evaluation. |
| `IdGovernance` | `backend/idgovernance` | LifecycleWorkflow policy and orchestration; the record of truth stays in IdManagement ([ADR-117](decisions/ADR-117-extract-identity-governance-context.md)). |
| `Authentication` | `backend/authentication` | Credential verification, MFA, login sessions, step-up, password change and reset, authentication events. |
| `OAuth2` | `backend/oauth2` | OAuth 2.0 / OIDC protocol endpoints, clients, consent, tokens, role policy. |
| `Application` | `backend/application` | Application catalog, protocol bindings, assignment, portal ordering and categories. |
| `Audit` | `backend/audit` | Read model of audit events across every context. Owns the search-attribute registry, PII transformation, admin API, and retention. |
| `ClaimMapping` | `backend/claimmapping` | Protocol-neutral claim release policy, identity attribute projection, fail-closed validation. |
| `Provisioning` | `backend/provisioning` | SCIM 2.0 outbound provisioning: the push lifecycle toward downstream SaaS. idmagic's User/Group is the source of truth and the downstream is a mirror ([ADR-128](decisions/ADR-128-extract-provisioning-context-and-transactional-delivery-capture.md)). |
| `Sourcing` | `backend/sourcing` | Inbound identity intake from an upstream authority. Owns source bindings, correlation with the external immutable id, and deletion/deactivation that follows the upstream authority. Organized as one feature slice per source; currently `sourcing/scim` only ([ADR-141](decisions/ADR-141-inbound-identity-sourcing-taxonomy.md)). |
| `ApiTokens` | `backend/apitoken` | Tenant-scoped API access tokens (`idmagic_pat_` prefix) that authenticate the management and SCIM APIs: issuance, revocation, listing, and the scope vocabulary ([ADR-135](decisions/ADR-135-unify-scim-and-management-api-tokens.md)). |
| `Jobs` | `backend/jobs` | Generic asynchronous job infrastructure that preserves the tenant boundary. Design: [`backend/jobs/ARCHITECTURE.md`](backend/jobs/ARCHITECTURE.md). |
| `Seeding` | `backend/seeding` | Environment profiles, dry-run, redacted plans, and apply policy. Business data and its persistence stay in each record context ([ADR-118](decisions/ADR-118-extract-environment-aware-seeding-context.md)). |
| `SigningKeys` | `backend/signingkeys` | Tenant-and-usage-scoped key metadata, X.509 credentials, rotation, repository port, admin/JWKS HTTP, and memory/PostgreSQL/Vault adapters. JWT and XML wire signers stay in the protocol adapters. Design: [`backend/signingkeys/ARCHITECTURE.md`](backend/signingkeys/ARCHITECTURE.md). |
| `DataKeys` | `backend/datakeys` | Per-tenant `DataEncryptionKey` (DEK) metadata and lifecycle (bootstrap/rotate/disable/destroy) for reversible secrets left in the app DB (e.g. MFA TOTP seeds). Does not own signing keys (`SigningKeys`) or the `EnvelopeCrypto` port itself, which lives in `backend/shared/security` as a technical shared adapter ([ADR-148](decisions/ADR-148-envelope-encryption-and-datakeys-context.md)). |
| `WsFederation` | `backend/wsfederation` | WS-Fed passive, WS-Trust active STS, federation metadata, MEX, RP trust, and request-tenant XML signing. Design: [`backend/wsfederation/ARCHITECTURE.md`](backend/wsfederation/ARCHITECTURE.md). |
| `Saml` | `backend/saml` | SAML 2.0 IdP, SP trust, metadata, SSO/SLO, and request-tenant XML signing. Design: [`backend/saml/ARCHITECTURE.md`](backend/saml/ARCHITECTURE.md). |

The published vocabulary and dependencies between contexts are authoritative in `spec/scl.yaml`
`context_map`. Before adding a direct import, revisit `depends_on` there.

## Conventions

A bounded context normally takes this shape.

```text
backend/<context>/
  architecture.yaml  # this context's module ledger
  domain/            # entities, value objects, state machines, pure validation
  usecases/          # application logic that performs the specified operations
  ports/             # abstractions over repositories, stores, external services
  handlers_http/     # inbound HTTP adapter
  db_memory/         # memory repository adapter
  db_postgres/       # PostgreSQL repository adapter
```

`domain/` knows nothing of Echo, PostgreSQL, or HTTP request/response. `usecases/` depends on `ports/`
and never on a concrete adapter. `handlers_http` owns wire conversion, HTTP status, cookies and headers,
and boundary concerns such as CSRF and Origin. The rule that `usecases/` never imports an adapter holds
in every context: outward capabilities (signing, assignment gating, authentication resolution) arrive as
a `ports/` abstraction or an interface declared inside the usecase package, and the adapter injects the
concrete implementation — for example `oauth2`'s `ports.TokenIssuer`, or the `ApplicationGate` interface
in `saml` and `wsfederation`.

Adapters sit directly under the context or feature that owns them, named `<role>_<technology>` in
snake_case, with no `adapters/` or `persistence/` classification directory in between. A package name
alone should reveal whether the role is handler, repository, publisher, or client, and whether they represent technical adapters—HTTP, PostgreSQL, S3, or SCIM. A classification directory destroys exactly that: the
package name stops saying what the thing does
([ADR-133](decisions/ADR-133-flat-wikipedia-architecture.md)).

Whether a context has `domain/` and `usecases/` follows from whether it has logic of its own; the
packages are not placed mechanically. Shared SCL Go bindings stay in `backend/shared/spec` (ADR-070),
while context-specific business types are owned by that context's `domain/` (ADR-089). A context such as
`tenancy`, which has no domain logic beyond the bindings, has no per-context `domain/`. Contexts that do
have their own logic — `idmanagement` (User/Group/Agent aggregates, attribute schema, field validation)
or `saml` / `wsfederation` (protocol-specific parsing and claim mapping) — have `domain/`, and contexts
that orchestrate SSO/sign-in (SP/RP resolution, signature verification, assignment gating, claim
issuance) have `usecases/`. Every issuance decision in browser federation lives in `usecases/`;
`handlers_http` stays closed around wire format and the HTTP boundary.

`backend/shared/` is for technical capabilities that several contexts genuinely share. Putting a
context-specific concept there because it is convenient widens the reading surface of the next change.
Concrete domain event structs belong to the owning context's `domain/events.go`, and
`backend/shared/spec/events.go` carries only the event envelope interface and its wire marshalling.
Audit classifies on a stable event type discriminator rather than a registry of concrete types.

### Feature vertical slices

A context with two or more independent sub-domains (features) may add a feature vertical slice layer to
the four-layer grid: `backend/<context>/<feature>/{domain,ports,usecases,<role>_<technology>}/`
([ADR-130](decisions/ADR-130-idmanagement-feature-vertical-slice.md)). A single-feature context does not
get one — a `<context>/<context>/` stutter stops the directory structure from screaming its purpose, and
is harmful for that reason. The pilot is `idmanagement`, split into `user`, `group`, and `agent`:

```text
backend/idmanagement/
  module.go                 # one per context (the DI bundle is not split by feature)
  domain/                   # only types shared across features (enums, DomainEvent)
  usecases/                 # only cross-feature usecase helpers and error values
  deps_http/                # the Deps type definition itself (leaf package, see below)
  handlers_http/            # route registration and cross-feature integration tests
  user/  group/  agent/     # each feature's domain/ ports/ usecases/ and adapters
```

`handlers_http` and `db_postgres` could not be split naively, because of Go's language rules and the
unit of code generation; they needed a different design from `domain`/`ports`/`usecases` (ADR-130).

- **handlers_http**: handlers were originally methods on a `Deps` struct (`func (d Deps) handleX`). Go
  only allows a method to be defined in the same package as its receiver type, so splitting `Deps` into
  per-feature embedded sub-structs would force the same field to be duplicated across them wherever a
  feature reaches across (group handlers use `UserRepo`; agent handlers use `UserRepo` and
  `ClientRepo`). Instead the `Deps` type definition was extracted into a standalone leaf package
  `deps_http`, and the handlers were converted into **free functions**
  (`func handleX(d Deps, c *echo.Context) error`) and moved into the feature packages. A free function
  need not share a package with the receiver type, so the implementation could be separated per feature
  without splitting `Deps` at all. `routes.go` re-exports it as `type Deps = httpdeps.Deps` (a type
  alias through an import alias), so external construction sites (`idmagic.Deps{...}` in bootstrap and
  tests) were untouched.
- **db_postgres**: the idmanagement entry in `sqlc.yaml` was split into several per-feature entries, and
  `queries/*.sql` together with the generated `sqlcgen/` moved into the feature directories.
  Cross-feature test fixture helpers (`seedTenant`, `seedUser`) were duplicated into each feature
  package, because Go's `_test.go` files cannot span packages. The `lifecycle_workflows` queries and
  `sqlcgen` belong to the IdGovernance context and therefore live in `backend/idgovernance/db_postgres/`
  (ADR-090, ADR-117).

Core package names stay as the layer names (`domain`/`ports`/`usecases`); adapter package names stay
`<role>_<technology>` (`handlers_http`, `db_memory`, `db_postgres`). Where several features of one
context are imported together, named imports (`userDomain`, `groupDomain`) disambiguate them. Resolving
the collision with an import alias preserves the shared vocabulary of the core packages, which
lengthening the directory names would not (ADR-133).

### Frontend Component Structure

A UI context boundary aligned with an RA/SCL feature is placed in `frontend/src/features/<feature>/`.
Views, local components, helpers, tests, and localized dictionaries (`*.i18n.ts`) for that feature must reside in its directory. The alias `slices/` is not used.
Cross-cutting reusable components not tied to a specific feature boundary are placed in `frontend/src/components/`.

## Cross-cutting Concerns

### HTTP routing

Routes are composed in `backend/shared/http/server_http/routes.go`. It registers the tenant-scoped
routes under both the default tenant and `/realms/:tenant_id`, and separates only control-plane tenant
administration under `/realms/default/admin/tenants`.

Each context's routes live in `backend/<context>/handlers_http/routes.go`; read that file for the exact
endpoint list. A new HTTP API is registered in the owning context's `routes.go` with its handler under
the same `handlers_http`. Context-specific repositories and route wiring are collected in
`backend/<context>/module.go` so the central router only calls the Module (ADR-091).

### Request correlation

Every request is assigned a `request_id`, returned in the `X-Request-ID` response header and attached to
every application log line for that request (alongside `trace_id` / `span_id` when
`OBSERVABILITY=otel`).

Because `X-Request-ID` is attacker-controllable, the default is to **self-generate the id and ignore any
inbound value** — secure by default, so a directly reachable client cannot spoof or collide correlation
ids. Set `REQUEST_ID_TRUST_INBOUND=true` only when a trusted edge proxy generates (and thereby
sanitizes) the header, which is what makes a single id shared across the proxy and application tiers
worth having. A proxy that passes the client value through untouched must not be trusted. Either way, a
reused inbound value is bounded in length and character set as defense in depth against header and log
injection.

### Metrics

`GET /metrics` exposes Prometheus/OpenMetrics-format metrics: HTTP RED (count, error rate via
`status_code`, duration, in-flight) for every route template, plus authentication golden signals for
SLO and alerting.

| Metric | Labels | Verifies |
| --- | --- | --- |
| `http_requests_total`, `http_request_duration_seconds`, `http_requests_in_flight` | `route`, `method`, `status_code` | per-interface latency and error-rate objectives |
| `authn_login_attempts_total` | `outcome`, `reason_class`, `method` | login success/failure golden signal |
| `authn_login_throttle_total` | `policy`, `outcome` | login throttle hit rate |
| `oauth2_token_issuance_total`, `oauth2_token_issuance_duration_seconds` | `grant_type`, `outcome` | `/token` issuance rate and latency by grant |
| `http_request_aborts_total`, `operation_detached_completion_failures_total` | `kind` | cancellation policy |

Every label is a bounded, finite set. `tenant_id`, `user_id`, `client_id`, and resolved request paths are
never labels, because their cardinality is unbounded; that is also why the endpoint is scraped outside
the tenant-resolution middleware and kept separate from the application API. It is always registered but
returns `503` until the process finishes constructing its Prometheus registry at startup, and works
independently of `OBSERVABILITY`, because a pull-based scrape needs no collector configured. Expose it
only on a loopback/management network or behind an authenticating proxy.

### Logging

Application logs are structured JSON Lines on stdout (`timestamp`, `level`, `service`, `message`, plus
`trace_id` / `span_id` / `request_id` for correlation — `backend/shared/logging`, ADR-018). This process
never writes them anywhere else; aggregating and searching them across replicas and nodes is a separate,
externally-observing concern, kept independent of the OpenTelemetry Collector so a logging outage cannot
affect trace/metric export.

**Local** (`infra/docker/docker-compose.dev.yaml`): Promtail discovers every container through the Docker
Engine API (`docker_sd_configs`) and ships its logs to Loki, so no host log directory needs to be
bind-mounted — only the Docker socket. Grafana is provisioned on first boot with both Prometheus (wi-11)
and Loki as datasources and with the existing golden-signals dashboard, so `docker compose up` is enough
to browse metrics and logs together.

**Kubernetes** (`infra/k8s/monitoring/loki/`): Promtail runs as a DaemonSet, discovering pods via
`kubernetes_sd_configs` and tailing `/var/log/pods`; Loki runs as a single-replica StatefulSet with PVC
storage (ADR-102 placement; filesystem storage is a dev-shaped default, replaced by an overlay for
object-store-backed retention in a real production cluster, per the work item's Risk Notes). Grafana
itself is not deployed by this repo — the Loki datasource is registered against whatever Grafana instance
already exists, using the same ConfigMap-sidecar convention `grafana-dashboard.yaml` already relies on
(a label the cluster's Grafana sidecar watches), rather than the `grafana-dashboard.yaml` dashboard content
itself.

| Field | Loki treatment | Why |
| --- | --- | --- |
| `service`, `level` | index label | bounded, finite sets |
| `trace_id`, `span_id`, `request_id` | structured metadata (not a label) | unbounded — an index label here would blow up cardinality, the same reasoning [Metrics](#metrics) applies to `tenant_id` / `user_id` |

### HTTP server hardening

The boundary HTTP server applies production-safe timeouts and a request body limit so a single slow or
oversized client cannot exhaust connections or memory (`gosec G112` / CWE-400). Bodies over the limit are
rejected with `413`.

| Variable | Default | Purpose |
| --- | --- | --- |
| `HTTP_READ_HEADER_TIMEOUT` | `10s` | max time to read request headers (slowloris bound) |
| `HTTP_READ_TIMEOUT` | `30s` | max time to read the full request |
| `HTTP_WRITE_TIMEOUT` | `60s` | max time to write the response |
| `HTTP_IDLE_TIMEOUT` | `120s` | keep-alive idle connection timeout |
| `HTTP_MAX_BODY_BYTES` | `1048576` | max request body size in bytes (1 MiB) |

This is defense in depth, not a substitute for an edge proxy. The primary line against volumetric floods
and TLS-handshake slowloris is the fronting reverse proxy, which sees total traffic and can stop abuse
cheaply. idmagic still enforces its own timeouts and body limit so it stays safe when run without a
proxy, and so the proxy-to-app hop and any in-cluster direct access are covered.

### Security response headers

A boundary middleware applies `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`,
`X-Frame-Options: DENY`, and a strict `Content-Security-Policy`
(`default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'`) to every backend
response. `frame-ancestors 'none'` together with `X-Frame-Options: DENY` forbids framing, so the login,
consent, and portal surfaces cannot be clickjacked. The CSP does not use `'unsafe-inline'`: the only
inline script idmagic renders is the fixed auto-submit of the SAML ACS / WS-Fed POST-binding form, which
is pinned by a `script-src 'sha256-…'` hash on that response with its `form-action` narrowed to the
destination endpoint.

**Header ownership.** CSP and `frame-ancestors` require per-route decisions and are owned by idmagic, so
they hold even behind a minimal or absent proxy. The SPA is served by the gateway, which sets its own
`script-src 'self'` CSP for the static HTML.

**HSTS belongs to the TLS terminator.** `Strict-Transport-Security` is off by default so development
over plain `http` is not poisoned. Enable it only when TLS is terminated at or ahead of this hop: leave
it to the edge proxy in the usual setup (`HSTS_ENABLED=false`), or set `HSTS_ENABLED=true` when the app
itself should assert it (tuning `HSTS_MAX_AGE_SECONDS` / `HSTS_INCLUDE_SUBDOMAINS`).

To tighten CSP without breaking a page, set `CSP_REPORT_ONLY=true` to emit
`Content-Security-Policy-Report-Only` and `CSP_REPORT_URI=<url>` to collect violations, observe, then
switch back to enforce.

### Persistence

Persistence ports and repository implementations belong to the owning context. Context-specific memory
and PostgreSQL adapters go in `backend/<context>/{db_memory,db_postgres}`; the shared DB pool, row
scanner, and transaction helpers go in `backend/shared/storage/db_postgres` (ADR-090, ADR-133).
Ephemeral state is consolidated into PostgreSQL as well, so no second class of datastore is operated
([ADR-139](decisions/ADR-139-consolidate-ephemeral-state-into-postgresql.md)).

All static SQL statements in `db_postgres` must use `sqlc` to generate type-safe queries. Raw `Pool.Query`/`Pool.Exec` with strings are only permitted for highly dynamic queries where `sqlc` provides no benefit, serving as an escape hatch rather than the default (ADR-090).

To add structure to PostgreSQL, first update the current-state schema in `infra/schema/postgres.sql`.
Structural diffs are inspected with a `psqldef` dry-run and applied by a pre-deploy job (the procedure is
in `infra/schema/README.md`). Changes a structural diff cannot express — backfills, value conversions,
saving data before a drop — are stated explicitly in the work item's runbook or a dedicated SQL script.
There is no migration runner at application startup. The memory adapter is also the reference for tests
and the local demo, so never update only the postgres side.

`postgres.sql` carries no SQL comments. Design rationale for a table or column goes here (or the owning
context's own `ARCHITECTURE.md`) instead, both because a second copy of "why" drifts from this record and
because `psqldef`'s dependency-ordering pass has shown comment-content-sensitive statement-ordering bugs
on an empty database (`infra/schema/README.md` Rules has the detail and upstream references).

### Database design policy

#### 1. Column type selection

The selection rules are fixed so the judgement is reproducible each time a table is added
([ADR-084](decisions/ADR-084-postgres-column-type-policy.md)).

- **Free-form strings, unbounded**: use `TEXT`. Never use unconstrained `varchar`.
- **Bounded strings**: `TEXT` + `CHECK (char_length(col) <= N)`, or `varchar(N)`, consistently. Which one
  and the specific `N` follow `wi-128-string-length-limits-policy`. Fixed-format identifiers are guarded
  with `CHECK (... ~ regex)`.
- **Internally generated ids**: columns idmagic generates with `spec.NewUUIDv4()` are `UUID`. Go keeps
  them as `string`; the pgx text codec registration (`RegisterUUIDAsText`) bridges the two.
- **Externally decided ids**: ids whose value an external party decides (`entity_id`, `wtrealm`,
  `scim_id`, `kid`) stay `TEXT`, because idmagic does not assign them and they are not UUIDs.
- **Time**: `TIMESTAMPTZ` throughout, with microsecond precision as the source of truth. Do not round in
  the schema.
- **Finite value sets**: `TEXT` + `CHECK (col IN (...))`. PostgreSQL enums are avoided, because adding a
  value requires `ALTER TYPE` and sits badly with declarative schema diffing.
- **JSONB**: for external-spec-derived metadata, claim/policy configuration, and append-only payloads
  only. Values that need joins or filters, FK or uniqueness constraints, or that participate in a state
  machine are not kept inside JSONB.

#### 2. tenant_id retention classes

`users.id` and `oauth2_clients.client_id` are system-generated, globally unique identifiers, so child
rows reference their parent by that global key and **tenant-scoped composite foreign keys are not used**
([ADR-082](decisions/ADR-082-user-domain-id-and-tenant-key-policy.md), simplified by
[ADR-083](decisions/ADR-083-globally-unique-client-id.md)). Do not add `tenant_id` merely because the
tenant is reachable through a globally unique parent; add it when it serves search, a constraint,
retention, or audit.

- **Tenant-owned aggregate / tenant-scoped config**: carries `tenant_id`, usually as part of the primary
  or a unique key (`users`, `groups`, `oauth2_clients`, `applications`, `agents`, `signing_keys`,
  `application_categories`, `saml_service_providers`, `wsfed_relying_parties`, `*_sign_in_policies`).
- **External tenant-scoped natural key**: `tenant_id` is part of the primary key because the external id
  is only unique within a tenant (`scim_user_refs`, `scim_group_refs` on `(tenant_id, scim_id)`).
- **Child of a globally unique parent**: identified by the global key (`user_id` / `client_id`), with no
  `tenant_id` unless per-tenant search or retention needs it (`consents`, `application_orderings`,
  `mfa_factors`, `password_history`, `password_reset_tokens`, `email_change_tokens`, `group_members`).
  Two exceptions keep it. `authentication_sessions`: the session id is an opaque cookie value resolved on
  every request, so `tenant_id` is a fail-closed defense-in-depth predicate on that lookup as well as the
  per-tenant active-session listing index
  ([ADR-126](decisions/ADR-126-postgresql-as-login-session-source-of-truth.md)). The ephemeral
  auth/OAuth2 stores keyed by an opaque token, code, or challenge
  (`oauth2_authorization_requests`, `oauth2_authorization_codes`, `oauth2_par_requests`,
  `oauth2_device_codes`, `oauth2_replay_jtis`, `oauth2_access_token_denylist`, `webauthn_sessions`,
  `login_throttle_counters`, `saml_authnrequest_replays`) keep it for the same reason: each lookup is a
  high-frequency, fail-closed resolution of an attacker-influenced opaque key, so the tenant boundary is
  enforced in the DB layer (ADR-139; the ADR-082 §4 exception).
- **Append-only / audit / throttling**: decided by emit-time tenant, query boundary, and
  retention (`audit_events`, `authentication_event_buckets`).

#### 3. Envelope encryption for reversible secrets

Reversible secrets that must remain in the app DB (MFA TOTP seeds today; future Token Vault upstream
tokens) are never stored as plaintext. The scheme is two-tier: a master key, held by a swappable
`EnvelopeCrypto` provider, wraps a per-tenant `DataEncryptionKey` (DEK); the DEK then AEAD-encrypts each
secret directly. AEAD/keyset handling is delegated entirely to [Tink](https://developers.google.com/tink)
— no hand-rolled nonce, tag, or AAD assembly. Every ciphertext binds an AAD of
`(tenant, context, table, record id, field)` plus the DEK version, so a ciphertext copied across tenant,
table, or field boundaries fails to decrypt rather than silently succeeding.

- `EnvelopeCrypto` (the Tink-backed AEAD/keyset port, plus the OpenBao/cleartext-keyset master-key
  provider adapter) lives in `backend/shared/security` alongside `certificates_mtls` /
  `passwords_argon2id` / `tokens_jose` — it is a technical capability, not a business aggregate.
- `backend/datakeys` (`DataKeys` context) owns only the wrapped-DEK metadata and lifecycle (bootstrap,
  rotate, disable, destroy) — never the `EnvelopeCrypto` port itself, mirroring how `SigningKeys` keeps
  `transit/sign` separate from this `encrypt`/`decrypt`/`datakey` capability.
- Rotation activates a new DEK version for new writes while the previous version stays `retiring` (still
  decryptable) until a resumable re-encryption job, registered through `backend/jobs`'
  `JobKind`/`HandlerRegistry` (`wi-126-async-job-runner`), migrates every reference; only then can the old
  version be destroyed. A `FieldMigrator` port (`backend/datakeys/ports`) lets each owning context
  register its own batch re-encryption/pending-count logic without `DataKeys` depending on any consumer's
  schema, mirroring how `Jobs`' `HandlerRegistry` stays decoupled from consumer business logic; Rotate
  auto-enqueues the job per registered migrator, and Destroy refuses to erase a wrapped DEK while any
  migrator still reports pending rows.
- Decrypt failure (unwrap failure, provider unreachable, AAD/tamper mismatch) is fail-closed: the caller
  denies access rather than falling back to plaintext or skipping the field.
- The initial master-key provider is OpenBao (Vault Transit-compatible HTTP API); dev/local uses a Tink
  cleartext keyset so OpenBao is not required to develop. The provider is swappable by design.
- The only HTTP surface is a read-only, `system_admin`-gated `GET /api/admin/data-keys/health`
  (`backend/datakeys/handlers_http`) reporting each tenant's active DEK version/status and master-key
  provider name/reachability — never key material. There is no rotate/disable/destroy admin endpoint;
  those lifecycle operations are internal-only for now.

The full rationale — including why OpenBao over HashiCorp Vault CE, and why this is not merged into the
`SigningKeys` `KeyStore` port — is in
[ADR-148](decisions/ADR-148-envelope-encryption-and-datakeys-context.md).

`tenant_data_encryption_keys.wrapped_dek` is erased (set `NULL`) on destroy rather than the row being
deleted — crypto-shredding — so DEK lifecycle history (`status` moving through `active` / `retiring` /
`disabled` / `destroyed`) stays queryable after the key material itself is gone.

#### 4. Per-context schema notes without a dedicated `ARCHITECTURE.md`

The contexts below have no `<context>/ARCHITECTURE.md` of their own (see [Context Map](#context-map)), so
schema-level design rationale that does not fit the policies above lives here instead of in
`infra/schema/postgres.sql` (which carries no comments, see [Persistence](#persistence)).

- **Tenancy** — `tenant_brandings` (wi-89, ADR-096) is 1:1 hosted UI branding config kept in its own
  table rather than columns on `tenants` so per-feature config growth does not bloat the core tenant row;
  `tenant_branding_assets` and `tenant_user_attribute_schemas` follow the same reasoning. Absence of a
  `tenant_brandings` row, or all-`NULL` columns, means branding is unset and callers fall back to system
  defaults. `tenant_branding_assets` (wi-89, ADR-096) validates and stores logo/favicon blobs in the same
  shape as `application_icons` (ADR-073), kept in a separate table and `object_key` space so branding
  asset ownership never crosses with Application icon storage. `notification_templates` (wi-288,
  ADR-142) holds tenant overrides of the notification email catalog keyed by
  `(tenant_id, template_key, locale)`; absence of a row means the builtin default for that key/locale
  applies, and deleting the row is how "reset to default" works (no version history, ADR-142 §1).
  Individual columns rather than JSONB keep per-column length limits as `CHECK` constraints (ADR-084);
  `subject`/`body_text`/`body_html` are `NOT NULL` together so a half-overridden template cannot exist
  (ADR-142 §4), while `from_display_name` is nullable because the system default sender name is a valid
  choice. `tenants.default_locale` (wi-288, ADR-142 §7) is the tenant tier of notification locale
  resolution (recipient → tenant → system); `NULL` means "use the system default".
  `tenants.endpoint_style` (wi-285, ADR-144) fixes the shape of a tenant's canonical location (its
  issuer, cookie scope, and WebAuthn RP ID all derive from it); `'path'` is the default because it needs
  neither wildcard DNS nor a wildcard certificate.
- **OAuth2** — `oauth2_client_secrets` (wi-25) separates the `client_secret` credential from the client
  row itself; `oauth2_clients.client_secret_hash` is kept only for dual-read/backfill during that
  rollout. `refresh_tokens.sid` (ADR-127) is the OIDC session id, equal to
  `authentication_sessions.id`, `NULL` when issuance has no browser session (`client_credentials` etc.);
  there is no FK to `authentication_sessions` because housekeeping retention's physical deletes there run
  independent of a refresh token's own revoke state (the ADR-082 §4 opaque-cross-context-reference
  exception). `refresh_tokens.resource` (RFC 8707, ADR-055, wi-262) is the resource indicator bound at
  authorization-code redemption, retained across rotation; `NULL` means no resource was specified.
  `mcp_resource_servers` (ADR-055) is the tenant-scoped registration of an MCP resource server (a
  tool/data source); `resource` is the tenant-unique canonical resource URI that Protected Resource
  Metadata (RFC 9728) and resource-indicator (RFC 8707) verification are checked against. Within the
  ephemeral protocol-state tables already listed under [tenant_id retention
  classes](#2-tenant_id-retention-classes): `oauth2_authorization_requests` keeps the whole `/authorize`
  mid-flow state in `payload` JSONB and serializes its transitions with `SELECT ... FOR UPDATE` in one
  transaction (ADR-139 §3); `oauth2_authorization_codes` is single-redeem, promoting `state` to a CAS
  predicate (`UPDATE ... WHERE state = 'issued' RETURNING`); `oauth2_par_requests` is likewise
  single-consume through a `used` CAS predicate; `oauth2_device_codes` keys on `device_code_hash` with
  `user_code` as a tenant-unique secondary lookup, approval sets `user_id`, and `state` is the Exchange
  CAS predicate; `oauth2_replay_jtis.kind` distinguishes `dpop` from `client_assertion` replay guards,
  recorded via `INSERT ... ON CONFLICT DO NOTHING` (a new row means new-use); `oauth2_access_token_denylist`
  stays `LOGGED` (replicated to a physical standby) because losing a revocation on failover would be a
  defense-in-depth regression.
- **Audit** — `audit_event_search_attributes` (wi-145) is a sidecar search index: one row per
  `(event, attr_name, transformed value)`, where `attr_name` is a `Field` from the `AuditSearchRegistry`.
  PII attributes are hashed or rounded before being stored here; plaintext exists only in
  `audit_events.payload`, and only for failure events under short-lived retention. It cascades on
  `audit_events` deletion, and its lookup index orders `(tenant_id, attr_name, attr_value)` for equality
  matches with `occurred_at DESC` for the scan.
- **ApiTokens** — `api_tokens` (wi-273, wi-275) holds lifecycle records for managed RFC 9068 JWT access
  tokens; JWT bodies are never stored, `jti` is the lookup key. `scopes` lists the granted
  `<resource>:<action>` permissions (`ApiTokenScope` in `spec/contexts/api-tokens.yaml`); the table's
  `CHECK` mirrors that enum as defense in depth alongside Go-side validation.
- **IdGovernance** — `lifecycle_workflow_revisions` are append-only; execution records
  (`lifecycle_workflow_runs`/`_steps`) reference the revision they expand rather than mutable JSON.
- **Provisioning** — `provisioning_connections.credential_secret` (ADR-128) is a dev/test-grade plaintext
  column with no envelope encryption yet (wi-97); production deployments must not rely on it as-is,
  matching the `SigningKeys` PostgreSQL `KeyStore`'s own dev/test disclaimer.

## Runtime Composition

The main package in `backend/cmd/idmagic/` performs startup, and `backend/cmd/internal/bootstrap` owns
startup-time DI. `backend/cmd/idmagic-worker/` only claims durable
jobs and runs handlers, scaling horizontally independently of the API
([ADR-099](decisions/ADR-099-job-worker-execution-model-and-fault-tolerance.md)).
`backend/cmd/idmagic-batch/` is started one-shot by an external scheduler, performs a single retention
sweep or signing-key lifecycle pass, and exits
([ADR-124](decisions/ADR-124-scheduled-batch-execution-boundary.md)). Every runtime unit reuses the same
Go module and bounded context implementations. The ledger of these units is `runtime_units` in
`architecture.yaml`.

This shape — every bounded context implementation in a single Go module, with several runtime units as
thin entry points reusing that shared implementation — is currently a **modular monolith**. Context
boundaries are kept strict as logical boundaries (contexts couple through published language and ports,
ADR-091), and by default several contexts compose into one process. The runtime splits that do exist
keep the synchronous dependencies of authentication and OAuth2 inside the API process, and are limited
to what the triggers in [REGENERATIVE_ARCHITECTURE.md §3.9](REGENERATIVE_ARCHITECTURE.md) justify:
resource and latency characteristics (per-lane workers,
[ADR-129](decisions/ADR-129-job-execution-lanes.md)), and the execution boundary of cross-cutting batches
(ADR-124). The organizational
trigger has not fired, so no service split happens until independent data ownership, teams, and SLOs
exist (ADR-099). This describes the present state; it does not prescribe a future style.

`Dependencies` in `backend/cmd/internal/bootstrap/deps.go` is the boundary aggregate handed to the HTTP
layer, absorbing runtime choices such as memory / postgres / console / otel. Context-specific
repositories are bundled into each `Module`, and the central `Dependencies` and server `Deps` receive
the Module. After adding a port, check at least the context's `ports/`, the memory adapter, whether the
postgres adapter and a schema diff are needed, `bootstrap.Dependencies`, `assembleMemory` /
`assemblePostgres`, `support.Deps`, and the constructor of the HTTP handler or usecase involved.

### Availability and shared state

Running more than one replica requires the `postgres` runtime (`PERSISTENCE=postgres`, `DATABASE_URL`).
All shared state, durable and ephemeral alike, lives in PostgreSQL rather than in per-replica process
memory (ADR-139).

- **Durable**: refresh tokens, audit events, authentication-event aggregation buckets, and **login
  sessions**. A logged-in browser session has `authentication_sessions` as its single source of truth,
  so restarting or rolling API replicas does not invalidate active sessions. Revocation (self-service,
  logout, or an account being disabled) tombstones the row (`revoked_at` / `revoke_reason`) instead of
  deleting it, so a repeated revoke request is a safe no-op.
- **Ephemeral** (short-lived auth/OAuth2 rows): authorization request, authorization code, PAR, device
  code, DPoP and client-assertion replay guards, access-token denylist, WebAuthn ceremony challenges,
  and the login brute-force throttle. All are short-lived and retry-safe. Every row carries `expires_at`
  and every read filters on `expires_at > now()`, so TTL correctness is independent of the best-effort
  GC sweep that `idmagic-worker` runs to reclaim space.

A cutover onto this runtime abandons any **in-flight** ephemeral state (an `/authorize` mid-flow, a
pending PAR or device request, a throttle counter). Those simply restart and recover, and no durable
state is affected.

The login throttle in particular *must* be shared. With per-replica counters an attacker's failed
attempts split across `N` replicas, so the per-account and per-IP lockout thresholds effectively loosen
by up to `N×` cluster-wide — a silent security regression. In the shared PostgreSQL counter they are
counted cluster-wide with a serialized `SELECT ... FOR UPDATE` update, and the account and IP
identifiers are SHA-256 hashed so no plaintext username or IP is stored.

Because the throttle sits on the critical path, its degradation is **fail-closed**: if the store is
unreachable, a login attempt whose throttle state cannot be verified is rejected rather than let
through. Run PostgreSQL in a highly available configuration (REGIONAL / synchronous standby) for
multi-replica deployments so this path stays up.

The `memory` runtime keeps this state in process and is therefore **single-replica / test only**.

## Structural Decisions

- The artifact boundary between `backend/` and `frontend/`, and the placement of Go entry points, follow
  [ADR-092](decisions/ADR-092-backend-and-frontend-top-level-directories.md).
- The placement of runtime and database infrastructure assets follows
  [ADR-102](decisions/ADR-102-infrastructure-root-for-runtime-and-database-assets.md).
- Technical shared context is separated from context-owned adapters, and context-specific persistence
  adapters live with their context, because a shared package that accumulates context-specific concepts
  widens the reading surface of every later change
  ([ADR-070](decisions/ADR-070-technical-shared-context-for-cross-context-adapters.md),
  [ADR-090](decisions/ADR-090-context-local-persistence-and-sqlc.md)).
- SCL normative elements, Architecture modules, declared checks, and revision-stamped evidence are
  directly traceable — not only for audit, but so an AI can fetch the minimum context a change needs
  ([ADR-115](decisions/ADR-115-direct-workspace-traceability-graph.md)).
- Keeping the structure as an executable, checkable declaration is
  [ADR-116](decisions/ADR-116-executable-architecture-map.md). Moving that ledger into
  `architecture.yaml` and making this document the prose design record is
  [ADR-143](decisions/ADR-143-second-layer-design-ledger-decision-split.md).
- Separating LifecycleWorkflow from IdManagement's record of truth into IdGovernance's policy and
  orchestration follows [ADR-117](decisions/ADR-117-extract-identity-governance-context.md).
- Environment-specific seed policy and execution orchestration are separated from the record contexts and
  applied through each context's published command surface
  ([ADR-118](decisions/ADR-118-extract-environment-aware-seeding-context.md)).
- Outbound provisioning (the SCIM client) is a separate context from the inbound SCIM server because the
  direction of truth is reversed and the lifecycles differ. Delivery does not observe the existing
  queue; it is committed by a same-transaction capture that writes `ProvisioningDelivery` inside the
  caller's Postgres transaction
  ([ADR-128](decisions/ADR-128-extract-provisioning-context-and-transactional-delivery-capture.md)).
- Inbound identity intake is grouped by whether there is an authority with a durable source binding —
  not by direction or runtime shape — into a single `Sourcing` context with one feature slice per
  source. A source-independent core is not built until a second source lands (thin root)
  ([ADR-141](decisions/ADR-141-inbound-identity-sourcing-taxonomy.md)).
- Envelope encryption for DB-resident reversible secrets splits the technical `EnvelopeCrypto` port
  (Tink AEAD/keyset, master-key provider) from the business-facing per-tenant DEK lifecycle, the same
  split `SigningKeys` uses for `transit/sign`; the port lives in `backend/shared/security`, the lifecycle
  in the new `DataKeys` context, and neither is merged into `SigningKeys`, whose `KeyStore` port has a
  different operation shape and lifecycle ([ADR-148](decisions/ADR-148-envelope-encryption-and-datakeys-context.md)).

## Documentation Policy

Once you know what you want to write, this table decides where it goes. The axis is the question each
document answers.

| What you want to write | Where it goes | Question it answers |
| --- | --- | --- |
| Normative requirement, behavior, contract, data shape | `spec/contexts/*.yaml` | What must hold |
| Current design of one context | `<context>/ARCHITECTURE.md` | How it is now, and why |
| Cross-cutting design, conventions, cross-cutting policy | this document | The same, for what spans contexts |
| Machine-checked module ledger | `architecture.yaml` (root / context) | How the structure is checked |
| Rejected options, premises at the time, revisit conditions | `decisions/ADR-NNN-*.md` | What was rejected, and why this was chosen |
| How to use or run something | the `README.md` of that directory | How to use it / how to run it |
| What to do when something happens | `infra/runbooks/*.md` | What to do in an incident |
| A one-off implementation record | `work-items/` | What was done this time |

Do not open an ADR merely to describe a design. An ADR is written when there was a real fork and a
rejected option actually exists. Conversely, when design is written here, add a sentence or two of
reason and link to the ADR — never transcribe the ADR body, because a second copy always drifts.

Do not hand-copy lists that can be read mechanically from code or schema. No exhaustive endpoint tables,
test inventories, or environment-variable tables.

Prose in `ARCHITECTURE.md` is written in English, matching `README.md`.
