---
context: audit
updated_at: 2026-08-06
---

# Architecture: audit

## Overview

The `Audit` context exposes the admin-facing `AdminAuditEventResponse` view over the platform's
audit trail of `DomainEvent`s.

## Retention

Audit records are kept append-only for 7 years, on the basis of GDPR Article 30 (records of
processing activities); no deletion or archival interface exists for them today. This is a fixed
operational value rather than an SLO, so it lives here rather than as an SCL `objectives` entry —
`spec/contexts/audit.yaml` deliberately carries no retention objective and points here instead.
Rationale in
[ADR-107](../../decisions/ADR-107-audit-retention-and-jobs-dev-environment-topology.md) §1.
