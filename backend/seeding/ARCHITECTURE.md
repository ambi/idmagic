---
context: seeding
updated_at: 2026-08-09
---

# Architecture: seeding

## Overview

The `Seeding` context is an operations bounded context owning `SeedProfile`, `SeedRequest`,
`SeedPlan`, environment policy, drift policy, and application order. It does not own the meaning,
validation, or persistence of seeded resources — those stay in each record context (IdManagement,
Authentication, OAuth2, Application, Saml, WsFederation), reached through their existing idempotent
command surfaces. This split keeps environment safety and application order centralized while
avoiding duplicated invariant checks; the rejected alternative of scattering profiles across record
contexts loses that single point of cross-context safety verification.

## Environment policy and planning

A profile is never inferred from environment name — it must be given explicitly by request/CLI.
Production accepts only the `bootstrap` profile; `demo`/`test`/`performance` are rejected
fail-closed before any write, so a misrouted request cannot seed demo credentials into production.
Dry-run and apply share the same planner, and re-applying the same manifest/generator-seed/secret
version is a no-op — manual drift is a conflict by default, with explicit reconcile left as a
separate, later contract. Application uses bounded, dependency-ordered batches of idempotent
commands rather than one cross-context transaction; the `performance` profile's batch size defaults
to 250 and caps at 1,000. Rather than a dedicated checkpoint table, the same request replays
deterministically from logical keys/IDs derived from profile and generator seed, serialized by an
in-process mutex per request key and, across processes, a PostgreSQL advisory lock on the existing
connection.

## Seed manifests and secret references

`models.SeedManifest` is a versioned, strictly-decoded YAML desired state, converted by the
`manifests_yaml` adapter into domain types before reaching the existing per-resource contributors;
domain and usecases never see the parser or filesystem/env APIs directly. `include` resolves only
local relative paths under the manifest root, bounded by depth and total-count limits — YAML merge
keys, templating, remote URLs, and env-var expansion are excluded from the grammar to avoid path
traversal and injection surfaces. Secret values are never written into a manifest; they are
referenced through `models.SeedSecretReference`, whose `env` provider is available everywhere but
whose `file` provider is the only one permitted in staging/production. Dry-run validates that a
reference resolves without ever passing the materialized value into a plan, log, or error.

## Design Decisions

- `Seeding` is a separate operations context that owns environment policy, drift policy, and
  application order across record contexts through their existing idempotent command surfaces,
  rather than scattering seed profiles across each record context and losing the single point of
  cross-context safety verification
  ([ADR-118](../../decisions/ADR-118-extract-environment-aware-seeding-context.md)).
- Seed manifests are versioned, strictly-decoded YAML with a restricted `include`/secret-reference
  grammar (no merge keys, templating, remote URLs, or literal `${ENV}` expansion), and re-applying
  the same manifest/generator-seed/secret version replays deterministically rather than relying on a
  dedicated checkpoint table
  ([ADR-132](../../decisions/ADR-132-use-versioned-seed-manifests-and-secret-references.md)).
