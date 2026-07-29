---
context: jobs
updated_at: 2026-07-26
---

# Architecture: jobs

## Overview

The `Jobs` context owns no business processing. It owns the execution substrate common to any
tenant-owned asynchronous work: enqueue, durability, claim, lease, heartbeat, retry, dead-letter, and
cancel. Interpreting a JobKind's params and performing its side effects stays in the consumer context's
usecase; `backend/cmd/idmagic-worker/worker.go` composes those handlers into the registry at startup.
The API process enqueues jobs but never runs them, and the worker process runs jobs but never serves
HTTP.

Business logic is kept out of the substrate so that touching the job infrastructure does not require
re-reading every consumer context. The module ledger is in `architecture.yaml` beside this file.

```text
API / consumer usecase
  └─ EnqueueJob (lane is derived from the kind)
       └─ JobRepository ──> PostgreSQL jobs (lane column, lane-prefixed index)
                                  │
idmagic-worker                    │ poll (independent per lane)
  ├─ lifecycle workflow dispatcher│ (recovers runs that were never enqueued)
  ├─ Runner (lane=latency_sensitive) ── ClaimBatch(lane) <───┘
  ├─ Runner (lane=default)          ── ClaimBatch(lane) <───┘
  ├─ Runner (lane=bulk)             ── ClaimBatch(lane) <───┘
  │    ├─ HandlerRegistry ──> consumer context usecase (shared registry)
  │    ├─ Heartbeat ────────> extends lease_expires_at
  │    └─ Complete / Fail ──> succeeded / queued(retry) / failed
  └─ jobsQueueDepthSamplingLoop ──> per-lane queue depth/active gauges (10s interval)
```

## Execution lanes

A `JobKind` carries exactly one `ExecutionLane` (`latency_sensitive`, `default`, or `bulk`), fixed at
registration by `domain.RegisterKind(kind, lane)`. The enqueue caller cannot choose it: `Job.Lane` is
derived from the kind's registration, and a claim only considers jobs within its own lane. The caller is
deliberately excluded because a lane is a unit of capacity isolation, not a per-call priority
([ADR-129](../../decisions/ADR-129-job-execution-lanes.md)).

Each lane has an independent `Runner` with its own concurrency semaphore, batch-claiming as many jobs as
it has free slots and running handlers concurrently (four slots by default). One process can start
`Runner`s for several lanes at once — the compat mode when `JOB_WORKER_LANES` is unset, and the default
for development and docker-compose — or run a dedicated deployment holding a single lane's `Runner`,
which is the production default (`infra/k8s/base/worker.yaml` defines the three deployments
`idmagic-worker-{latency-sensitive,default,bulk}`, with per-lane concurrency in
`JOB_WORKER_CONCURRENCY_<LANE>`).

Ordering across lanes is not a goal. The point is capacity isolation — however far a bulk backlog
builds up, it never takes execution slots from `latency_sensitive` — so no numeric priority exists
within a lane either. Claim candidates within a lane are roughly oldest-`run_at` first, but with
concurrent execution, multiple processes, and jobs sharing a timestamp, neither start nor completion
order is guaranteed.

## Claim and lease

`ClaimBatch` picks up, within its lane only, the `queued` jobs that are due and the `running` jobs whose
lease has expired. On PostgreSQL, a
`WHERE lane = $lane AND (...) ORDER BY run_at FOR UPDATE SKIP LOCKED` combined with the `running` update
in the same statement prevents two workers from holding a valid lease on the same job. Putting the
durable queue in PostgreSQL rather than a dedicated broker is
[ADR-098](../../decisions/ADR-098-durable-job-queue-skip-locked-lease.md).

The execution guarantee is **at-least-once**. Each claim increments `attempts` and sets `lease_owner`
and `lease_expires_at`. A running handler heartbeats every third of the lease period, and only the lease
owner may complete. If a process crashes or is killed and its heartbeat stops, another worker may
re-claim the job once the lease expires — so **handlers must be idempotent**, using a dedup key and the
consumer's own consistency boundary.

## Retry and dead-letter

On failure the job returns to `queued` with an exponentially backed-off `run_at`, and settles as
`failed` once `max_attempts` is reached. Poll interval, in-process concurrency, lease, and retry backoff
are process-wide settings: `JOB_POLL_INTERVAL`, `JOB_WORKER_CONCURRENCY`, `JOB_LEASE_DURATION`,
`JOB_BACKOFF_BASE`, `JOB_BACKOFF_CAP`. They currently offer no per-JobKind QoS, nor consumer-specific
ordering or rate limits.

On SIGTERM/SIGINT the worker stops claiming and waits for in-flight handlers up to the drain grace
period. After the grace period it exits and recovery happens through natural lease expiry rather than an
explicit re-enqueue — explicit re-enqueue is avoided because it would open a double-execution window
when drain cutoff races handler completion (ADR-099).

## Metrics

`idmagic-worker` exposes `/metrics` (`MetricsExposition`, system.yaml) on a separate management-only
HTTP listener — a different process and instance from the idmagic-api `/metrics`. It carries the
per-lane `jobs_claim_latency_seconds`, `jobs_outcome_total`, `jobs_retry_total`, and `jobs_queue_depth`.
`tenant_id` and `job_id` are excluded from the labels to keep cardinality finite.

## Boundary with scheduled batch

Periodic all-tenant retention and signing-key lifecycle work is not mixed into durable jobs; an external
scheduler starts `idmagic-batch` one-shot instead. A cross-cutting sweep has no per-tenant unit of work,
so pushing it through a tenant-owned queue would make both queue depth and lane isolation meaningless
([ADR-124](../../decisions/ADR-124-scheduled-batch-execution-boundary.md)).

The lifecycle workflow dispatcher inside the worker is not a periodic batch that performs business work
directly. It is a recovery path: it re-scans WorkflowRuns that were committed in the same transaction
but never associated with a Job, and hands them off safely to the durable queue.

## Schema notes

`jobs.tenant_id` is required even though a `Job` has no natural-key parent, following the root
`ARCHITECTURE.md` [`tenant_id` retention classes](../../ARCHITECTURE.md#2-tenant_id-retention-classes)
tenant-owned-aggregate category. `status` and `kind` are closed vocabularies normative in
`spec/contexts/jobs.yaml`; the `CHECK` constraints are defense in depth alongside that, not the source of
truth. `params`/`result` are opaque per-`JobKind` JSONB payloads (ADR-100: no at-rest encryption in this
WI); terminal rows are purged after a TTL by the worker's retention sweep, not by a dedicated Job.
`lane`'s `DEFAULT 'default'` backfills pre-lane rows in place when the column was added to an existing
table, part of ADR-129 decision 5's zero-downtime rollout. `dedup_key` backs `JobHandlerIdempotency`: the
partial unique index allows at most one non-terminal Job per `(tenant_id, dedup_key)`.

## Design Decisions

- Durable queue implemented as a PostgreSQL `FOR UPDATE SKIP LOCKED` lease:
  [ADR-098](../../decisions/ADR-098-durable-job-queue-skip-locked-lease.md)
- Process separation, delivery guarantee, and drain:
  [ADR-099](../../decisions/ADR-099-job-worker-execution-model-and-fault-tolerance.md)
- Boundary with scheduled batch:
  [ADR-124](../../decisions/ADR-124-scheduled-batch-execution-boundary.md)
- Capacity isolation through execution lanes:
  [ADR-129](../../decisions/ADR-129-job-execution-lanes.md)
