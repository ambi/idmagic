---
context: provisioning
updated_at: 2026-08-09
---

# Architecture: provisioning

## Overview

The `Provisioning` context owns outbound delivery of identity changes to downstream SaaS targets —
the push counterpart to `Sourcing`'s pull/receive side. It is named for the capability
(`Provisioning`), not a direction word, so it stays correct regardless of how the inbound side is
later taxonomized; direction, authority (idmagic is source of record), and vocabulary invert
between the two contexts, so they do not share code beyond published references to `Tenancy` /
`Application` / `IdManagement` / `Jobs`.

## Protocol-agnostic core with protocol feature slices

Most outbound behavior — the connection envelope, `DeprovisionPolicy`, `AttributeMappingRule`,
and the `ProvisioningDelivery`/`RemoteResourceLink` delivery engine (queueing, retry/backoff,
quarantine, ordering, resync) — does not depend on the wire protocol. Only the wire client and
some connection setup (auth method, capability discovery, default attribute schema) are
protocol-specific. The context root (`domain`, `ports`, `usecases`, `handlers_http`) therefore
holds the protocol-agnostic core, and each protocol gets its own feature slice implementing the
`ProvisioningTargetClient` port — currently `client_scim`, with `entraid`/`googledir` expected to
follow as siblings without touching the core. This is a deliberate variant of the repo's usual
"fat slice, thin shared root" convention: here the domain shape is mostly protocol-neutral with
protocol as the driven-adapter axis, so the core is fat and the slices are thin.

No shared SCIM wire kernel exists between this context's `client_scim` slice and `Sourcing`'s
`scim` slice: inbound's filter parser/evaluator and fixed response structs serve a receiver that
evaluates incoming SCIM against its own data, while outbound needs to *build* filter strings and
serialize a broader, mapping-driven attribute set (`externalId`, enterprise extensions). The
actual overlap (discovery structs, RFC schema URNs) is small enough that sharing now would
constrain both sides prematurely; extraction is deferred to when real duplication appears.

## Same-transaction delivery capture

Delivery does not observe the existing `outbox`/Relay drain — that path only forwards to external
transports (Kafka/PubSub/log) with no in-process consumer, and its topic registration is
incomplete and non-transactional for the events this context needs. Instead, `Provisioning`
implements a published capture port that `IdManagement`'s user-mutation path and `Application`'s
assignment path call inside their own Postgres transaction, inserting one `pending`
`ProvisioningDelivery` row per matching active connection. This mirrors the same-transaction
capture pattern established for lifecycle-workflow triggers: the User/assignment commit and the
delivery row become durable atomically, so `ProvisioningDelivery` itself is this context's outbox
equivalent, without depending on the shared outbox's atomicity. Delivery idempotency uses the key
`(tenant, connection, source_type, source_id, source_version)`.

## Design Decisions

- `Provisioning` is split out as its own bounded context — named for the capability rather than a
  direction word — with a protocol-agnostic core (connection, mapping, delivery engine) and thin
  per-protocol feature slices (`client_scim` today, `entraid`/`googledir` to follow) implementing
  `ProvisioningTargetClient`, rather than a shared SCIM wire kernel with the inbound side
  ([ADR-128](../../decisions/ADR-128-extract-provisioning-context-and-transactional-delivery-capture.md)).
- `Sourcing` (inbound identity intake) does not mirror `Provisioning`'s protocol-agnostic core,
  because unlike outbound delivery it has no shared engine already implemented to extract into one
  ([ADR-141](../../decisions/ADR-141-inbound-identity-sourcing-taxonomy.md)).
- The transactional trigger-capture pattern — writing the triggering mutation and its queued
  follow-up row in the same transaction as the record context's commit — originated for
  lifecycle-workflow triggers in IdManagement and was preserved when that ownership moved to
  IdGovernance; Provisioning reuses the same pattern for its own delivery capture
  ([ADR-113](../../decisions/ADR-113-identity-lifecycle-workflow-execution-model.md),
  [ADR-117](../../decisions/ADR-117-extract-identity-governance-context.md)).
