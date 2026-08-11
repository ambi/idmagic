---
context: jobs
updated_at: 2026-08-11
---

# Jobs Specification

## Overview

テナント境界を保つ汎用非同期ジョブ基盤の durable queue と worker runtime を所有する。
各ジョブの業務ロジックは呼び出し元の bounded context の usecase に残り、Jobs は
enqueue・claim・リース・リトライ・進捗可視化という技術的能力だけを提供する。管理者向け
一覧/詳細/キャンセル API と運用 metrics は別 feature slice が所有する。

The `Jobs` context owns no business processing. It owns the execution substrate common to any
tenant-owned asynchronous work: enqueue, durability, claim, lease, heartbeat, retry, dead-letter, and
cancel. Interpreting a JobKind's params and performing its side effects stays in the consumer context's
usecase; `backend/cmd/idmagic-worker/worker.go` composes those handlers into the registry at startup.
The API process enqueues jobs but never runs them, and the worker process runs jobs but never serves
HTTP.

Business logic is kept out of the substrate so that touching the job infrastructure does not require
re-reading every consumer context. Module boundaries are inferred from paths and checked for forbidden imports.

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

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| JobLease | worker が Job を Running にする際に確保する排他権。lease_owner (worker 識別子) と lease_expires_at を持ち、heartbeat で更新される。heartbeat が途絶えて期限切れになると別 worker が再取得できる。 | lease, リース |
| DeadLetter | attempts が max_attempts を超えて Failed に確定した Job。再試行されず、error を保持したまま調査対象として残る。 | dead-letter |
| Queued | Job が enqueue され、まだ worker に claim されていない初期状態。run_at を過ぎれば claim 可能。 | queued |
| Running | worker がリースを確保し、ハンドラを実行中の状態。 | running |
| Succeeded | ハンドラが正常終了した終端状態。 | succeeded |
| Failed | attempts が max_attempts を超えて dead-letter に確定した終端状態。 | failed |
| Canceled | 終端状態に達する前に取消された終端状態。 | canceled |
| Claim | worker が claim 可能な Job を取得し、自らを lease_owner として Running へ遷移させる。 | claim |
| Complete | worker がハンドラの正常終了を報告し、Job を Succeeded に確定する。 | complete |
| Fail | worker がハンドラの失敗を報告する。attempts が max_attempts 未満なら Retry、以上なら dead-letter (Failed) に確定する。 | fail |
| Retry | 失敗後、backoff を経て再試行のため Queued に戻す遷移。 | retry |
| Cancel | 終端状態に達していない Job を取消す。 | cancel |
| ExecutionLane | JobKind の登録情報が一意に決める実行枠の区分。異なる<br>レイテンシ・資源特性を持つ JobKind 間で claim と worker 実行枠を隔離するために使う。<br>enqueue 呼び出し元は lane を指定できない。latency_sensitive / default / bulk の3種。 | lane, 実行レーン |
| Developer | 標準開発環境でこのリポジトリを動かす開発者。 |  |
| System | Jobs の durable queue と worker runtime そのもの。人間の操作者を伴わない技術的な主体を指す。 |  |

## State Transitions

### JobLifecycle

Job の状態機械。queued → running は worker の claim、running → queued は
リトライ (backoff 待ち)、succeeded / failed / canceled は終端で不可逆。running → failed は
attempts が max_attempts に達した dead-letter 確定、running → queued (retry) は達していない場合。

Initial: `queued`
Terminal: `succeeded`, `failed`, `canceled`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | JobStarted | — | running |  |
| running | JobSucceeded | — | succeeded |  |
| running | JobFailed | attempts >= max_attempts | failed |  |
| running | JobRetried | attempts < max_attempts | queued |  |
| queued | JobCanceled | — | canceled |  |
| running | JobCanceled | — | canceled |  |

## Design

### Internal Interfaces

#### EnqueueJob
呼び出し元 bounded context が非同期処理を queue に投入する内部インターフェース。
HTTP には公開せず、同一プロセス内の Go 呼び出しとして各 context の usecase から使う。
dedup_key を指定すると (tenant_id, dedup_key) が既存の未終端 Job と一致する場合は
新規作成せず既存 JobRef を返す。作成される Job の lane は kind の登録情報から一意に
決まり、呼び出し元は指定できない。

#### ClaimJobs
worker が指定した lane 内で claim 可能な (status=queued かつ run_at <= now、
またはリース失効済みの running) Job をバッチで取得し、同一トランザクション内で Running に
遷移させ、自分を lease_owner として lease_expires_at を設定する内部インターフェース。
対象 lane 以外の Job は取得しない。
- Result invariant: all_jobs_leased_by(output.jobs, input.worker_id)
- Result invariant: single_worker_holds_active_lease(output.jobs)
- Result invariant: all_jobs_in_lane(output.jobs, input.lane)

#### HeartbeatJob
worker が実行中の Job のリースを延長する。

#### CompleteJob
worker がハンドラの正常終了を報告し、Job を Succeeded に確定する。

#### FailJob
worker がハンドラの失敗を報告する。attempts < max_attempts なら backoff 後の
run_at を設定して Queued (Retry) へ、attempts >= max_attempts なら Failed
(dead-letter) へ確定する。
- Result invariant: output.status == 'failed' || output.status == 'queued'

#### CancelJob
終端状態に達していない Job を取消す。既に終端状態の Job には作用しない (不可逆)。

### Execution lanes

A `JobKind` carries exactly one `ExecutionLane` (`latency_sensitive`, `default`, or `bulk`), fixed at
registration by `domain.RegisterKind(kind, lane)`. The enqueue caller cannot choose it: `Job.Lane` is
derived from the kind's registration, and a claim only considers jobs within its own lane. The caller is
deliberately excluded because a lane is a unit of capacity isolation, not a per-call priority.

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

### Claim and lease

`ClaimBatch` picks up, within its lane only, the `queued` jobs that are due and the `running` jobs whose
lease has expired. On PostgreSQL, a
`WHERE lane = $lane AND (...) ORDER BY run_at FOR UPDATE SKIP LOCKED` combined with the `running` update
in the same statement prevents two workers from holding a valid lease on the same job. The durable queue
lives in PostgreSQL rather than a dedicated broker, reusing the same `FOR UPDATE SKIP LOCKED` claim
pattern already used elsewhere in the codebase for safe concurrent claiming instead of introducing
additional middleware.

The execution guarantee is **at-least-once**. Each claim increments `attempts` and sets `lease_owner`
and `lease_expires_at`. A running handler heartbeats every third of the lease period, and only the lease
owner may complete. If a process crashes or is killed and its heartbeat stops, another worker may
re-claim the job once the lease expires — so **handlers must be idempotent**, using a dedup key and the
consumer's own consistency boundary.

### Retry and dead-letter

On failure the job returns to `queued` with an exponentially backed-off `run_at`, and settles as
`failed` once `max_attempts` is reached. Poll interval, in-process concurrency, lease, and retry backoff
are process-wide settings: `JOB_POLL_INTERVAL`, `JOB_WORKER_CONCURRENCY`, `JOB_LEASE_DURATION`,
`JOB_BACKOFF_BASE`, `JOB_BACKOFF_CAP`. They currently offer no per-JobKind QoS, nor consumer-specific
ordering or rate limits.

On SIGTERM/SIGINT the worker stops claiming and waits for in-flight handlers up to the drain grace
period. After the grace period it exits and recovery happens through natural lease expiry rather than an
explicit re-enqueue — explicit re-enqueue is avoided because it would open a double-execution window
when drain cutoff races handler completion.

### Metrics

`idmagic-worker` exposes `/metrics` (`MetricsExposition`, system.yaml) on a separate management-only
HTTP listener — a different process and instance from the idmagic-api `/metrics`. It carries the
per-lane `jobs_claim_latency_seconds`, `jobs_outcome_total`, `jobs_retry_total`, and `jobs_queue_depth`.
`tenant_id` and `job_id` are excluded from the labels to keep cardinality finite.

### Boundary with scheduled batch

Periodic all-tenant retention and signing-key lifecycle work is not mixed into durable jobs; an external
scheduler starts `idmagic-batch` one-shot instead. A cross-cutting sweep has no per-tenant unit of work,
so pushing it through a tenant-owned queue would make both queue depth and lane isolation meaningless.

The lifecycle workflow dispatcher inside the worker is not a periodic batch that performs business work
directly. It is a recovery path: it re-scans WorkflowRuns that were committed in the same transaction
but never associated with a Job, and hands them off safely to the durable queue.

### Schema notes

`jobs.tenant_id` is required even though a `Job` has no natural-key parent, following the root
the repository specification's [`tenant_id` retention classes](../../SPECIFICATION.md#2-tenant_id-retention-classes)
tenant-owned-aggregate category. `status` and `kind` are closed vocabularies normative in
this document; the `CHECK` constraints are defense in depth alongside that, not the source of
truth. `params`/`result` are opaque per-`JobKind` JSONB payloads stored without at-rest encryption; terminal rows
are purged after a TTL by the worker's retention sweep, not by a dedicated Job. `lane`'s
`DEFAULT 'default'` backfills pre-lane rows in place when the column was added to an existing table, as
part of a zero-downtime rollout that needed no migration outage. `dedup_key` backs
`JobHandlerIdempotency`: the
partial unique index allows at most one non-terminal Job per `(tenant_id, dedup_key)`.

### Design Decisions

- The durable queue uses a PostgreSQL `FOR UPDATE SKIP LOCKED` lease so concurrent workers claim distinct
  rows without adding a second queue datastore.
- `idmagic-worker` is a process boundary separate from the API. Delivery is at least once, handlers own
  idempotency, and shutdown drains already-claimed work before the lease expires.
- Scheduled one-shot retention and key-lifecycle work belongs to `idmagic-batch`; Jobs owns durable,
  retryable asynchronous application work rather than every scheduled command.
- Execution lanes isolate concurrency for latency-sensitive, default, and bulk work without creating
  separate queue implementations.
- `params` and `result` are not encrypted at rest. They are opaque per `JobKind`, must not contain secrets,
  and terminal rows are purged by the worker retention sweep rather than by a recursively enqueued Job.

## Scenarios

### REQ-JOBS-001: Dockerなしの標準開発環境でworkerがジョブを完了する
- ACTOR Developer
- GIVEN embedded PostgreSQL binary が取得済み、または初回取得可能である
- GIVEN 開発用ポートが利用可能である
- WHEN 開発者が標準開発コマンドを実行する
- THEN embedded PostgreSQL が起動し schema が適用される
  - ALT binary 取得、ポート確保、または schema 適用に失敗する → 標準開発環境は API と UI を起動せず fail-fast する
- THEN API、worker、UI が起動する
- THEN API が Job を enqueue する
- THEN worker が同じ Job を claim して Succeeded にする
- THEN API と worker は同じ PostgreSQL queue を共有する

### REQ-JOBS-002: ジョブをenqueueするとworkerが実行して成功する
- ACTOR System
- GIVEN テナント "tenant-a" が存在する
- GIVEN worker プロセスが起動している
- WHEN テナント "tenant-a" に kind="noop_echo" の Job を enqueue する
- THEN Job の状態が queued である
- THEN worker が Job を claim し状態が running になる
- THEN ハンドラが正常終了し状態が succeeded になり result を保持する

### REQ-JOBS-003: 同じJobがリトライされてもハンドラの副作用は1回分しか観測されない
- ACTOR System
- GIVEN dedup_key を指定して Job が enqueue 済みである
- WHEN ハンドラが1回目の実行で外部への通知を送信する
- THEN Job は succeeded になる
- WHEN at-least-once 配信により同じ Job が再配送される
- THEN ハンドラは dedup_key を用いて冪等に判定し重複通知を送らない

### REQ-JOBS-004: workerがクラッシュしてもリース失効後に別workerが再取得する
- ACTOR System
- GIVEN テナント "tenant-a" の Job が worker-1 に claim され running である
- WHEN worker-1 が heartbeat を送らないまま停止する
- THEN lease_expires_at を過ぎる
- THEN worker-2 が同じ Job を claim し running を継続する

### REQ-JOBS-005: ハンドラが失敗し続けるとmax_attempts超過でdead-letterになる
- ACTOR System
- GIVEN max_attempts=3 の Job が running で FailJob を呼ばれた回数が既に2回である
- WHEN ハンドラが3回目も失敗し FailJob を呼ぶ
- THEN attempts が max_attempts に達している
- THEN Job の状態が failed になり error が保持される
- THEN Job は二度と running にならない

### REQ-JOBS-006: 他テナントのJobはworkerのテナント境界を跨がない
- ACTOR System
- GIVEN テナント "tenant-a" の Job "job-a" が running で worker-1 に claim されている
- WHEN worker-1 が "job-a" のハンドラを実行する
- THEN ハンドラ実行 context の tenant_id が "tenant-a" と一致する
  - ALT ハンドラが誤って他テナントの集約 ID を渡された → handler_execution_context.tenant_id と対象集約の tenant_id が不一致のため操作が拒否される

### REQ-JOBS-007: 同じdedup_keyのlifecycle_workflow_runは重複enqueueされない
- ACTOR System
- GIVEN テナント "tenant-a" で IdGovernance が dedup_key="lifecycle-workflow-run:run-1" の
        kind="lifecycle_workflow_run" Job を EnqueueJob 済みである

- WHEN IdGovernance が同じ dedup_key で再度 EnqueueJob を呼ぶ (at-least-once 配信の再送や dispatcher の重複実行を模す)
- THEN 新規 Job は作成されず既存 Job の JobRef が返る

### REQ-JOBS-008: API processのenqueueが失敗してもperiodicなdispatcherが未関連付けのrunを回収する
- ACTOR System
- GIVEN IdGovernance が User ライフサイクルイベントの購読から WorkflowRun (job_id 未設定、status=queued) を確定したが、API process の即時 EnqueueJob 呼び出しには失敗した
- WHEN worker process の periodic dispatcher が job_id が未関連付けの queued run を再走査する
- THEN dispatcher が dedup_key=lifecycle-workflow-run:{run_id} で EnqueueJob を呼び job_id を関連付ける
- THEN worker が Job を claim してハンドラを実行する

### REQ-JOBS-009: bulk laneのbacklogが滞留してもlatency_sensitiveジョブは専用実行枠でclaimされる
- ACTOR System
- GIVEN bulk lane に worker の concurrency を超える件数の長時間 Job が queued で滞留している
- GIVEN latency_sensitive lane を claim する worker が別途稼働している
- WHEN テナント "tenant-a" に kind="backchannel_logout_delivery" (lane=latency_sensitive) の Job を enqueue する
- THEN latency_sensitive lane の worker が bulk lane の backlog に関わらず即座に claim する
- THEN Job が running に遷移し bulk lane の滞留から実行が妨げられない

### REQ-JOBS-010: lane未登録のJobKindはworker起動時に拒否される
- ACTOR Developer
- GIVEN lane が登録されていない JobKind が handler registry に登録されている
- WHEN worker を起動する
  - ALT 同一 JobKind に複数の異なる lane が重複登録されようとした → worker の起動処理が重複登録を検出し起動を失敗させる
- THEN 起動処理が lane 未登録を検出し起動を失敗させる

### REQ-JOBS-011: lane列未設定の既存行はdefault laneへbackfillされclaim対象になる
- ACTOR System
- GIVEN lane 導入前に enqueue され lane が未設定 (schema 上の default) の queued Job "job-legacy" が存在する
- WHEN schema の backfill により \"job-legacy\" の lane が default になる
- THEN default lane を claim する worker が \"job-legacy\" を claim できる
