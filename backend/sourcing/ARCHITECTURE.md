---
context: sourcing
updated_at: 2026-08-09
---

# Architecture: sourcing

## Overview

The `Sourcing` context owns identity ingestion from upstream systems that hold durable authority
over an identity population: source binding, external-id correlation, ingestion runs, attribute
mapping, and deletion authority. The classification axis is authority and durable binding, not
transport direction or runtime shape — a distinction that ruled out an `Inbound` context name and
keeps admin CSV import and login-time federation out of this context. The context root stays thin
(facade and composition only); shared ingestion mechanics get pulled up once a second source slice
exists, not speculated in advance. Today the only member is the `scim` slice
(`scim/domain`, `scim/ports`, `scim/usecases`, `scim/handlers_http`, `scim/db_memory`,
`scim/db_postgres`), a SCIM 2.0 (RFC 7643/7644) inbound server.

## SCIM 2.0 inbound provisioning

Each tenant mounts `/realms/{realm_id}/scim/v2` and authenticates with a per-tenant Bearer token
that resolves tenant identity; a global shared token was rejected as a violation of tenant
isolation. The server implements `/Users` and `/Groups` (GET, POST, GET/{id}, PUT/{id},
PATCH/{id}, DELETE/{id}), `/ServiceProviderConfig`, `/ResourceTypes`, and `/Schemas`.

Attributes map directly onto the User/Group aggregates:

| SCIM | idmagic |
| --- | --- |
| `Users.id` | `User.sub` |
| `Users.userName` | `User.preferred_username` |
| `Users.name.formatted` / `displayName` | `User.name` |
| `Users.emails[type=work].value` | `User.email` |
| `Users.active` | `UserLifecycle.status == Active` |
| `Groups.id` | `Group.id` |
| `Groups.displayName` | `Group.name` |
| `Groups.members` | `GroupMember` memberships |

A PATCH/PUT toggling `active` to `false` transitions `User.lifecycle.status` to `Disabled`; toggling
it to `true` transitions it back to `Active`. `DELETE /Users/{id}` does not purge: it performs the
same soft-delete (`PendingDeletion`, 30-day grace period, then anonymize-cascade purge) as the rest
of the platform, so a misconfigured or erroneous external sync cannot cause unrecoverable PII loss —
this integrates SCIM deletion into the existing soft-delete policy rather than bypassing it.
`DELETE /Groups/{id}` is immediate and complete, since groups carry no PII.

## Design Decisions

- Inbound identity intake is grouped into `Sourcing` by whether there is an upstream authority with
  a durable source binding, not by transport direction or runtime shape — a distinction that keeps
  admin CSV import and login-time federation out of this context and rules out naming it `Inbound`
  ([ADR-141](../../decisions/ADR-141-inbound-identity-sourcing-taxonomy.md)).
- SCIM `DELETE /Users/{id}` integrates into the platform's existing soft-delete policy
  (`PendingDeletion`, 30-day grace period, then anonymize-cascade purge) rather than purging
  immediately, so a misconfigured or erroneous external sync cannot cause unrecoverable PII loss
  ([ADR-080](../../decisions/ADR-080-scim2-inbound-provisioning.md)).
