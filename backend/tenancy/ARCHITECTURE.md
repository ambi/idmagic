---
context: tenancy
updated_at: 2026-08-09
---

# Architecture: tenancy

## Overview

`Tenancy` owns the `Tenant` aggregate and everything that resolves a request to one: path-prefix
routing, the immutable-identity/mutable-slug key split, per-tenant branding, resource quotas, and the
authorization gate that separates admin operations from ordinary user traffic. `domain` holds the
aggregate and its invariants, `ports`/`usecases` the tenant lifecycle operations, and `handlers_http`
the HTTP adapter and repositories; `module.go` is the composition boundary other contexts bind against.

## Admin authorization gate

`/admin/*` resolves the authenticated browser session's `sub` to a `User` and allows the request only
when `admin` is present in `User.roles` and the account is not `disabled_at`. `roles` stores RBAC role
names directly on `User` rather than a separate tenant-membership model, because the first admin surface
is User lifecycle management and a system_admin operates from the default control-plane tenant;
tenant-scoped roles are deferred to their own model rather than encoded into `roles` as an embedded
tenant ID. State-changing admin requests additionally verify Origin and a CSRF token on top of
session authentication, since a session cookie alone does not prove the request originated from the
admin UI.

`disabled_at` is a reversible suspension distinct from `deleted_at`: a disabled user is rejected for new
sign-in, existing sessions, token reissue, and UserInfo, but the account and its history remain intact
for reinstatement. Admin responses use a dedicated `AdminUserResponse` that never includes
`password_hash`, and every admin mutation emits a domain event carrying both the actor's and the
target's `sub` for audit traceability.

## Tenant resolution

Every protocol and admin route is mounted under `/realms/{realm}/...`; tenant CRUD, being a cross-tenant
control-plane operation, lives at `/realms/default/admin/tenants/...` so the default tenant's session
cookie path already covers it without widening cookie scope to the root path. Path-prefix
resolution was chosen over subdomain and header-based resolution because a browser flow's OIDC `iss`
claim and Discovery metadata must be derivable from the same URL the client already used: a header
cannot survive a redirect, and a subdomain-only scheme forces wildcard DNS and per-tenant TLS onto local
dev and CI.

`TenantResolver` middleware extracts the realm segment with `^/realms/([a-z0-9][a-z0-9-]{0,62})(/|$)`,
resolves it against `TenantRepository`, and attaches the resolved `Tenant` and issuer string to the
request context. An unresolvable tenant returns a generic `tenant_not_found` 404 and a disabled tenant
returns a generic `invalid_request` 400 on protocol routes — neither response leaks which case occurred,
so tenant enumeration is not possible from the resolver's response shape alone.

**Canonical location invariant.** A tenant has exactly one canonical location and one issuer, selected by
`Tenant.endpoint_style` (`Path` or `Subdomain`); the other route is treated as not found. This replaced
an earlier design that let unprefixed requests fall back to the `default` tenant and offered a
`LEGACY_BARE_ISSUER` escape hatch — both let a single tenant answer from two origins, which violates
OpenID Connect Discovery's requirement that a document's `issuer` match the URL it was fetched from.
`Subdomain` is only selectable when a deployment configures a base domain; deployments that
don't stay on `Path` and require no wildcard DNS or certificates. `realm` itself is immutable — it
appears in both the issuer and, for `Subdomain` tenants, the hostname, so renaming it would carry the
same breakage as changing `endpoint_style`.

## Tenant identity: UUID key and realm slug

`tenants` splits its former single slug primary key into an immutable `id UUID` surrogate key and a
mutable, uniquely-constrained `realm TEXT` identifier, so that a realm can be renamed later (an
operationally legitimate request — organization rename, rebrand, typo correction) without touching the
opaque key every other table's `tenant_id` FK depends on. The externally exposed vocabulary —
URL prefix, OIDC issuer, Discovery metadata — consistently uses `realm`; every internal reference
(`tenant_id` FK columns, `spec.DefaultTenantID`, context-level `TenantID`) uses the UUID. Resolution
middleware bridges the two with `FindByRealm(realm)`, and admin API addresses tenants by `realm` in the
URL while resolving to the UUID before invoking use cases.

Two default-tenant constants follow the same split: `spec.DefaultTenantID` is a fixed UUID, consistent
with idmagic-generated id columns being UUID-typed throughout, and `spec.DefaultRealm` is the string
`"default"` used only where a tenant must appear in a URL. FK columns that reference `tenants(id)` are
UUID-typed, and `tenant_id` carries no SQL default — every insert must specify tenant_id explicitly, so a
missing value fails loudly instead of silently landing in the default tenant. This is a stricter instance
of the repo-wide [`tenant_id` retention classes](../../ARCHITECTURE.md#2-tenant_id-retention-classes)
policy: append-only
or opaque-key-keyed tables that don't FK to `tenants` (`audit_events.tenant_id`,
`authentication_event_buckets.tenant_id`) keep `tenant_id` as `TEXT` rather than `UUID`, since
tenant-less audit events need a sentinel value a UUID column can't hold cleanly.

## Tenant branding

`TenantBranding` is a separate entity keyed by `tenant_id`, not a value object embedded in `Tenant` —
the same shape as `TenantUserAttributeSchema` — so that presentational, independently-updated branding
config doesn't grow the core `Tenant` aggregate that authorization and realm resolution depend on.
Its eight fields (product name, logo, favicon, two brand colors, support link, legal link,
footer text) were chosen as the common subset across Okta, Entra ID, Keycloak, and OneLogin; arbitrary
CSS, HTML, scripts, and background images were deliberately excluded to keep the input surface
constrained.

Untrusted tenant input never reaches the hosted login shell as markup or free-form styling. Brand colors
are validated as `#rrggbb` and injected only as two fixed CSS custom properties
(`--tenant-brand-primary` / `--tenant-brand-accent`); text fields render through default escaping, never
`dangerouslySetInnerHTML`; and `support_url`/`legal_url` allowlist the `https://` scheme only, rejecting
`javascript:`, `data:`, and plain `http://` at write time. An earlier version of this design
also rejected color values that failed a WCAG AA contrast check against the default background, but the
admin UI never surfaced that check to the user, leaving no way to save an otherwise-intentional low
contrast brand color — contrast is no longer a save-time constraint; only format validation applies, and
the tenant bears the readability consequence.

Logo and favicon uploads reuse the same validated-blob pipeline as application icon storage (magic-byte
check, size cap, restricted format allowlist, `nosniff` delivery), factored into a shared
`backend/shared/mediavalidation` helper so both call sites stay behaviorally identical, but persist to a
dedicated `tenant_branding_assets` table so branding storage isn't attributed to Application ownership.
`GetTenantBranding` always succeeds: missing config, invalid values, or a missing asset all
fall back to the system default brand rather than failing the hosted login page. Every branding update
bumps `updated_at`, which the public response exposes as a cache-busting version/ETag; tenant_id is
already part of the cache key (the URL), so this alone is enough to invalidate stale cached branding
without cross-tenant leakage.

## Tenant resource quotas

Resource creation is capped per tenant to bound the blast radius of a single noisy or runaway tenant on
shared infrastructure. Quotas split into two enforcement classes: **Hard** quotas
(`users`, `groups`, `agents`, `applications`, `oauth2_clients`, `active_sessions`, `consents`,
`active_jobs`) are checked synchronously inside the creating transaction and reject the operation on
breach; **Soft** quotas (`audit_events_retained`, `export_artifacts_bytes`) allow the operation to
succeed and raise an asynchronous warning/audit event instead. Rate limiting alone was rejected because
it only bounds short bursts, not sustained long-run accumulation, and soft-only enforcement was rejected
because a bug or malicious loop could still exhaust the database before any async warning fires.

New tenants receive fixed default limits (e.g. 10,000 users, 1,000 groups, 100 agents, 50 applications,
100 OAuth2 clients, 50,000 active sessions, 10,000 consents, 10 active jobs). A System Admin can override
a specific tenant's limits individually; a Tenant Admin can view usage against its own limits but cannot
change them, keeping quota authority with the operator of the shared platform rather than the tenant
itself.

Rolling quotas out onto tenants that already exist without limits risks an immediate lockout, so
migration assigns a generously large safe ceiling up front (e.g. double current usage, or the default
times ten) rather than the standard default; a background reconciliation job then reconciles usage
counters against actual row counts, after which a System Admin can tighten limits deliberately.

## Design Decisions

- Admin authorization stores RBAC role names directly on `User.roles` rather than a separate
  tenant-membership model, with tenant-scoped roles deferred to their own model instead of being
  embedded into `roles`
  ([ADR-031](../../decisions/ADR-031-admin-user-api-and-rbac.md)).
- Tenant is a first-class aggregate with a two-tier authorization boundary: an `admin` role scoped to
  its own tenant, and a `system_admin` role scoped across tenants and housed in the default
  control-plane tenant
  ([ADR-032](../../decisions/ADR-032-tenant-as-first-class-aggregate.md)).
- Tenant resolution uses path-prefix routing (`/realms/{realm}/...`) rather than subdomain or
  header-based resolution, so a browser flow's OIDC `iss` claim and Discovery metadata can be derived
  from the same URL the client already used
  ([ADR-033](../../decisions/ADR-033-tenant-resolution-via-path-prefix.md)).
- A tenant has exactly one canonical location and issuer, selected by `Tenant.endpoint_style`; this
  replaced an earlier bare-issuer fallback and `LEGACY_BARE_ISSUER` escape hatch that let a single
  tenant answer from two origins
  ([ADR-144](../../decisions/ADR-144-tenant-canonical-location-and-host-based-resolution.md)).
- `tenants` splits its primary key into an immutable UUID surrogate key and a mutable, uniquely
  constrained `realm` identifier, so a realm can be renamed without touching the opaque key every
  dependent `tenant_id` FK relies on
  ([ADR-085](../../decisions/ADR-085-tenant-uuid-key-and-realm-identifier.md)).
- idmagic-generated id columns, including seed data, are UUID-typed; ids whose values are defined by an
  external authority (e.g. SAML `entity_id`) are not
  ([ADR-084](../../decisions/ADR-084-postgres-column-type-policy.md)).
- `TenantBranding` is a separate entity keyed by `tenant_id` rather than a value object embedded in
  `Tenant`, with a constrained field set and validated-blob upload storage reused from application icon
  storage
  ([ADR-096](../../decisions/ADR-096-tenant-branding-value-and-logo-storage.md)).
- Tenant branding color values are validated by `#rrggbb` format only; a WCAG contrast check is not
  enforced as a save-time constraint, superseding an earlier version of the branding design that did
  ([ADR-097](../../decisions/ADR-097-tenant-branding-color-contrast-is-advisory.md)).
- Tenant resource quotas split into synchronously-enforced Hard quotas and asynchronously-warned Soft
  quotas, with fixed defaults, System-Admin-only limit changes, and a generous safe-ceiling migration
  for tenants that predate quotas
  ([ADR-134](../../decisions/ADR-134-tenant-resource-quotas.md)).
