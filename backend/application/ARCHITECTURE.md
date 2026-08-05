---
context: application
updated_at: 2026-08-06
---

# Architecture: application

## Overview

`Application` owns the ApplicationCatalog: the tenant-scoped registry of applications an identity
provider issues tokens or assertions for, the sign-in policy (per-application and tenant-default) that
gates each login, and the relation between a catalog entry and its concrete protocol configuration
(OAuth2 client, SAML service provider, or WS-Fed relying party). `domain` holds the aggregate and policy
evaluation rules, `ports`/`usecases` the catalog and policy operations, and `handlers_http` the HTTP
adapter; `module.go` composes them for the router.

## Sign-in policy evaluation

`AppSignInPolicy` is a tenant/application-scoped ordered set of `SignInRule`s that ApplicationCatalog
owns and evaluates on every federation start (OIDC authorize, SAML SSO, WS-Fed sign-in) before a token or
assertion is issued — the same gate point as the existing per-application protocol-binding check, so
policy evaluation cannot be bypassed by choosing a different protocol entry point (ADR-079). An earlier
version accepted free-text ACR/factor strings and free-text network/device conditions but only ever
enforced them as a fail-closed rejection, producing configuration fields that looked functional but were
never actually evaluated; the model was replaced with values the evaluator can and does check (ADR-079).

Required authentication strength is a constrained `RequiredAuthnStrength` enum — `Password` or `Mfa` —
mapped 1:1 to internal ACR URNs and AMR values, rather than free text, since only two ACR values exist in
practice and an unconstrained string invites misconfiguration. Of the original free-text conditions,
only the two that the evaluator can actually check were kept structured: `reauth_max_age_seconds`
(evaluated against authentication/step-up recency) and `network_allow_cidrs` (the request's client IP
checked against admin-supplied, save-time-validated CIDRs); free-text device conditions were dropped
entirely rather than kept as an unenforced input (ADR-079).

Evaluation is fail-closed throughout. OIDC can route an insufficient-strength result to the existing
step-up flow; SAML and WS-Fed instead halt the protocol transaction outright with an explicit rejection
reason, since neither has a step-up mechanism to redirect to yet. A non-empty CIDR allowlist that the
client IP doesn't match, or a request where the client IP can't be determined at all, is a hard rejection
rather than a step-up opportunity (ADR-079).

## Tenant default policy composition

`TenantDefaultSignInPolicy` lets a tenant set one baseline sign-in policy for every application that
doesn't define its own, using the same `SignInRule` vocabulary and evaluator as per-application policy so
no second policy language exists. It is owned by ApplicationCatalog rather than `Tenancy`, since it is
conceptually about how applications are signed into, not about the tenant aggregate itself, keeping
sign-in policy ownership in one place (ADR-081).

The relationship between default and per-application policy is **override, not composition**: if an
application defines any enabled rules, those rules entirely replace the tenant default for evaluation
purposes; otherwise the default applies as-is. `EffectiveSignInRules(default, app)` selects one side or
the other before handing rules to the same fail-closed evaluator ADR-079 defined. An initial design
composed the two as a floor the application couldn't weaken, but that made the effective policy hard for
admins to read at a glance and required a separate exemption flag for legitimate low-risk relaxation;
override was chosen for a single, directly-inspectable effective policy per application, consistent with
ADR-079's principle that the per-application policy has final say (ADR-081).

Because override lets an application go below the tenant default, `AppSignInPolicyResponse` carries a
`weaker_than_default` flag — set when the override reduces required strength, loosens or drops
re-auth recency, or widens the allowed network — computed by `AppPolicyWeakerThanDefault(default, app)`
and surfaced as a UI warning rather than a block, since the underlying design goal was to make deliberate
relaxation easy, not to forbid it (ADR-081). New tenants start with an empty (allow-all) default so
introducing this feature changes no existing tenant's behavior until an admin opts in; auto-applying a
strict default such as mandatory MFA at migration time was rejected as too likely to cause a mass
lockout (ADR-081). Because the default lives as an ordinary table row, clearing its rules or deleting the
row is an immediate, reversible rollback to allow-all with no schema change involved.

## Application/protocol relation

An Application has at most one protocol configuration, fixed at creation time: a `weblink` application
has none, and a `federated`/`service` application has exactly one of OAuth2 client, SAML service
provider, or WS-Fed relying party. Reconnecting, detaching, or changing protocol type afterward is not
supported, reflecting that no real creation/edit flow ever needed an application to carry more than one
protocol binding, even though the original JSON-array binding model was built to allow it (ADR-138).

Each protocol table (`oauth2_clients`, `saml_service_providers`, `wsfed_relying_parties`) keeps its own
existing primary key and gains a nullable, unique `application_id`; a non-`NULL` value is a composite
foreign key that also pins tenant and a fixed protocol discriminator, so the database itself rejects two
protocol rows claiming the same Application, a cross-table duplicate claim, or a tenant/type mismatch —
none of which the prior JSON-array-of-bindings representation could express as a constraint, which had
instead required a full per-tenant scan to resolve. `NULL` represents a legitimate catalog-external
record: protocol configurations created through Dynamic Client Registration or trust-management APIs
that were never meant to appear in the Application catalog, which is why every protocol config isn't
required to carry an Application (ADR-138).

Because catalog creation spans two records (the Application row and the protocol row's `application_id`
relation), both commit in one transaction: if the second half fails, no orphaned catalog-visible
Application is left behind, and the protocol row that does exist is still valid as a catalog-external
record. Deleting an Application cascades to delete its owned protocol configuration, but a protocol
config that's Application-owned rejects direct deletion through the lower-level protocol management API
as a conflict — deletion has to go through the Application it belongs to (ADR-138).

The OAuth2 protocol table was renamed from the generic `clients` to `oauth2_clients` to match the SAML
and WS-Fed table names (`saml_service_providers`, `wsfed_relying_parties`), which already used
protocol-specific, domain-standard terms rather than a name generic enough to be confused with any other
kind of client (ADR-138).
