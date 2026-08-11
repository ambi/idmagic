---
context: audit
updated_at: 2026-08-09
---

# Architecture: audit

## Overview

The `Audit` context exposes the admin-facing `AdminAuditEventResponse` view over the platform's
audit trail of `DomainEvent`s.

## Retention

Audit records are kept append-only for 7 years, on the basis of GDPR Article 30 (records of
processing activities); no deletion or archival interface exists for them today. This is a fixed
operational value rather than an SLO, so it lives here rather than as a product requirement —
`spec/contexts/audit/requirements.md` deliberately carries no retention objective and points here instead.

## Design Decisions

- Audit event retention is fixed at 7 years under GDPR Article 30 record-keeping, documented as an
  operational value here rather than as a product objective
  ([ADR-107](../../decisions/ADR-107-audit-retention-and-jobs-dev-environment-topology.md)).
