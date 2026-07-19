---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-19
depends_on: [wi-126-async-job-runner]
change_kind: feature
initial_context:
  scl:
    Jobs:
      - models.Job
      - models.JobKind
      - interfaces.EnqueueJob
      - interfaces.ClaimJobs
  source:
    - backend/jobs/domain
    - backend/jobs/usecases
    - backend/jobs/adapters/persistence
    - backend/cmd/idmagic-worker
    - infra
  tests:
    - backend/jobs
    - backend/cmd/idmagic-worker
  stop_before_reading:
    - frontend
affected_spec:
  - { context: Jobs, kind: model, element: Job }
  - { context: Jobs, kind: model, element: JobKind }
  - { context: Jobs, kind: interface, element: EnqueueJob }
  - { context: Jobs, kind: interface, element: ClaimJobs }
---

# ジョブ実行レーンごとに処理能力を分離する

## Motivation

現行の `idmagic-worker` は全 JobKind を単一の claim 対象・単一の並行実行プールで
処理する。`run_at` の古い順に claim するため、CSV import、動的 Group 再計算、
full resync などの長時間・大量ジョブが実行枠を占有すると、logout notification
など低遅延が必要な配送も同じ backlog の後ろで待たされる。worker を水平
スケールしても全 instance が全 JobKind を処理するため、特定の負荷クラスへ
処理能力を予約できない。

Jobs の durable queue、リース、at-least-once、retry/dead-letter という共通基盤は
維持しつつ、異なるレイテンシ・資源特性を持つ JobKind 間で障害と容量を隔離する
必要がある。

## Scope

- `spec/contexts/jobs.yaml` の glossary、`models.Job` / `models.JobKind`、
  `interfaces.EnqueueJob` / `interfaces.ClaimJobs`、objectives、scenarios に
  execution lane と lane isolation の契約を追加する。
- `latency_sensitive`、`default`、`bulk` の3レーンを導入し、JobKind の登録情報で
  lane を一意に決定する。enqueue 呼び出し元が任意の高優先度 lane を指定できない
  形にする。
- 単一の `jobs` テーブルと共通 `JobRepository` を維持し、Job に永続化した lane を
  claim filter と index に利用する。
- worker ごとに処理対象 lane と concurrency を設定し、lane 別の worker
  deployment/process が互いの実行枠を消費しないようにする。
- lane 別の queued/running 件数、queue latency、成功・失敗・retry を、tenant IDや
  Job IDをラベルに含めず観測可能にする。
- 既存 JobKind の lane 割当、既存行の backfill、旧設定から lane 別 worker への
  無停止移行手順を定義する。
- ADR に実行レーン、共有テーブル、worker 配置、互換 rollout の判断を記録し、
  実装完了時に `ARCHITECTURE.md` の現在形を同期する。

## Out of Scope

- JobKind ごとの物理テーブル、Kafka/NATS/SQS 等の別 broker、JobKind ごとの
  専用 worker binary。
- lane 内の数値 priority、重み付きスケジューリング、tenant 間 fairness。
- SCIM connection 単位など、consumer 固有の順序保証、排他、rate limit。
- 管理 API/UI 全般。lane を表示・filter する変更は
  [[wi-157-job-admin-operations-surface]] と調整する。

## Plan

- lane は JobKind と1対1に固定し、初期割当を
  `backchannel_logout_delivery=latency_sensitive`、
  `user_import_preview/user_import_apply/dynamic_group_reconcile=bulk`、
  `noop_echo/lifecycle_workflow_run/provisioning_delivery=default` とする
  （`provisioning_delivery` は実装済みだが spec 未反映だったドリフトを
  本 work item で是正し lane 割当も同時に決める。理由は ADR-129 参照）。
  将来の JobKind は SCL-first で lane を同時に宣言する。
- PostgreSQL は共有 `jobs` テーブルへ `lane TEXT NOT NULL` を追加する。
  claim は対象 lane を必須条件とし、lane・status・run_at に適した部分 index を
  用いる。同一 lane 内は現行どおり due `run_at` の古い順で、厳密な開始順・完了順は
  保証しない。
- JobKind 登録を handler だけでなく lane metadata も検証する registry に拡張し、
  未割当・重複登録・worker 対象外 lane を startup/test で検出する。
- worker は設定された lane だけを claim する。production/development topology は
  lane ごとに独立した concurrency と replica 数を持たせ、少なくとも
  `latency_sensitive` の実行枠を bulk backlog から予約する。
- rollout は、既存行を `default` へ backfill できる schema、全 lane を処理する
  互換モード、lane 別 worker 配置、JobKind の最終割当適用の順で行い、どの段階でも
  claim されないジョブを作らない。

## Tasks

- [x] T001 [ADR] execution lane、共有テーブル、JobKind 固定割当、worker topology、
      無停止 rollout と却下案を決定する。→ ADR-129。ADR-099 の concurrency
      model 項目を部分置換（supersedes/superseded_by 相互参照済み）。
- [x] T002 [SCL] Jobs の glossary/models/interfaces/objectives/scenarios を先に更新し、
      `just yaml-check-scl` と派生成果物の同期を行う。→ `ExecutionLane` enum、
      `models.Job.lane`、`JobKind.provisioning_delivery` 追加（実装済みだが spec 未反映
      だったドリフトの是正）、`ClaimJobs.lane` 入力、lane 隔離/未登録拒否/backfill の
      3 scenario を追加。
- [x] T003 [Domain] RED: JobKind ごとの lane 決定、未割当・不正 lane 拒否、既定割当の
      test を先に失敗させる（`models.Job` / `models.JobKind`）→ GREEN。
      RED: `TestExecutionLane_Valid` / `TestLaneFor_BuiltinKinds` /
      `TestLaneFor_UnregisteredKind` / `TestRegisterKind_AssignsLane` /
      `TestRegisterKind_ConflictingLanePanics` / `TestRegisterKind_SameLaneIdempotent`
      を先に compile-fail で確認（`backend/jobs/domain/job_test.go`、scenario
      「lane未登録のJobKindはworker起動時に拒否される」「bulk laneの backlog...」）→
      `ExecutionLane` 型・`RegisterKind(kind, lane)`・`LaneFor` 実装で GREEN。
- [x] T004 [Usecase/Adapter] RED: 対象 lane 以外を claim しないこと、lane 内の
      lease 排他、複数 lane worker の容量隔離、旧行 backfill を memory/PostgreSQL
      test で先に失敗させる（`interfaces.EnqueueJob` / `interfaces.ClaimJobs`）→ GREEN。
      RED: `TestEnqueue_DerivesLaneFromKind`（usecases）、
      `TestClaimBatch_ExcludesOtherLanes`（memory・PostgreSQL 両方、PostgreSQL 側は
      lane-prefixed partial index 込みの実 SQL で確認）、
      `TestRunner_OnlyClaimsConfiguredLane`（usecases、bulk lane job を
      latency_sensitive-only Runner が claim しないこと）を先に失敗確認 → GREEN。
      `ports.JobRepository.ClaimBatch`/`EnqueueInput.Lane`、
      `RunnerConfig.Lane`（`NewRunner` は無効な lane で panic）、
      `infra/schema/postgres.sql` の `lane` 列・CHECK・lane-prefixed partial index、
      `queries/jobs.sql`/sqlcgen 再生成で実装。lease 排他・reclaim は既存 test が
      そのまま lane スコープ下で green（回帰なし）。
- [x] T005 [Runtime/Infra] lane 選択設定、lane 別 worker process/deployment、
      development topology、graceful drain を実装する。
      `backend/cmd/idmagic-worker/worker.go`: `JOB_WORKER_LANES`
      (未設定時は compat mode = 全 lane、`just dev`/docker-compose の既定)、
      lane 別 `JOB_WORKER_CONCURRENCY_<LANE>`（無指定時は既存
      `JOB_WORKER_CONCURRENCY` にフォールバック）、lane 数分の `Runner` を
      並行起動し、共有の drain grace period で全 lane を待つ。
      `infra/k8s/base/worker.yaml`: `idmagic-worker-{latency-sensitive,default,bulk}`
      の 3 Deployment（`JOB_WORKER_LANES` を単一 lane に固定）。
      `infra/k8s/base/pdb.yaml`/`networkpolicy.yaml` に lane 別追記。
      prod overlay で latency_sensitive/bulk の replica を 2 に、dev/prod で
      secretRef を分離。`just check-k8s dev`/`just check-k8s prod` green。
      `infra/docker/docker-compose.dev.yaml` の worker service は
      `JOB_WORKER_LANES` 未設定のまま compat mode を維持（コメントで明示）。
- [x] T006 [Obs] lane 別 queue latency、depth、active、outcome、retry metrics と
      structured log を追加し、高 cardinality/PII label がないことを検証する。
      RED: `TestLaneDepths_CountsQueuedAndRunningPerLane`（memory・PostgreSQL 両方）、
      `TestRunner_RecordsMetrics`（usecases、claim latency/outcome/retry 記録）を
      先に失敗確認 → GREEN。`ports.JobRepository.LaneDepths`、
      `usecases.JobsMetrics`（`RunnerDeps.Metrics` 経由、nil-safe no-op）、
      `observability.Metrics` に `jobs_claim_latency_seconds` /
      `jobs_outcome_total` / `jobs_retry_total` / `jobs_queue_depth`
      (lane・status/outcome ラベルのみ、tenant_id/job_id なし) を追加し
      `TestMetricsForbidsHighCardinalityLabels` を拡張して検証。
      `idmagic-worker` に `/metrics` 専用 HTTP listener と 10s 間隔の
      lane depth sampling loop を追加（MetricsExposition, system.yaml を
      idmagic-api/idmagic-worker の independent instance として記述更新）。
      `infra/k8s`: worker 3 Deployment に metrics port・Service・
      ServiceMonitor（matchExpressions で3 lane 一括）・NetworkPolicy ingress
      (prometheus) を追加。`infra/docker/prometheus-rules.yml` /
      `infra/k8s/monitoring/prometheus-rule.yaml` に lane 別 golden signal
      recording rule 3 種・alert 3 種、Grafana dashboard に lane 別 panel 4 種を
      追加。`just check-monitoring`/`just check-k8s dev`/`just check-k8s prod` green。
- [x] T007 [Docs] ADR、`ARCHITECTURE.md`、運用手順を実装後の現在形へ同期する。
      ADR-129 作成（T001）。`ARCHITECTURE.md`「Durable Job Worker」節を
      lane-aware な現在形に書き換え、`shared-adapters`/`worker` module の
      `depends_on` に `jobs-domain`/`jobs-ports` を追加し
      `just yaml-check-architecture` green。`README.md` に
      「Durable job worker (`idmagic-worker`)」設定表（`JOB_WORKER_LANES` 等）を
      新設し、Kubernetes/Monitoring 節を lane 別 Deployment・Service・
      NetworkPolicy・ServiceMonitor の現在形に更新。
- [x] T008 [Verify] bulk backlog 中も latency-sensitive job が予約枠で実行される
      integration testを含め、全検証を通す。
      `TestRunner_BulkBacklogDoesNotStarveLatencySensitive`
      （bulk lane を concurrency 超過分の長時間ジョブで飽和させた状態で
      latency_sensitive lane の Runner が即座に claim・完了することを確認。
      lane を意図的に揃えて timeout することも確認し test 自体の有効性を検証済み）。
      `just yaml-check` / `just scl-render` / `just verify-go` /
      `just check-k8s dev` / `just check-k8s prod` / `just check-monitoring` 全て green。
      既知の未関連failure: `TestAssembledRoutesMatchGeneratedOpenAPI`
      (`GET /session/check` spec-only) は本 work item 着手前の main で既に
      再現する既存の不具合であり、本 work item のスコープ外。

## Verification

- `just yaml-check`
- `just scl-render`
- `just verify-go`
- `just check-k8s dev`
- `just check-k8s prod`
- `just check-monitoring`
- 手動: bulk lane に concurrency を超える長時間ジョブを滞留させた状態で、
  latency-sensitive job が専用 worker に claim されることを確認する。
- 手動: 互換モードから lane 別 worker へ順次切り替え、queued/running の既存ジョブが
  未処理にならず終端することを確認する。

## Risk Notes

claim filter、schema、worker deployment を別々に切り替える変更であり、順序を誤ると
特定 lane のジョブが永久に claim されない。互換モードと段階的 rollout を先に
成立させ、lane 別 queue depth を移行ゲートとして監視する。数値 priority だけでは
実行枠を隔離できず、厳密優先度は低優先度ジョブを starvation させるため初期設計には
含めない。

## Completion
- **Completed At**: 2026-07-19
- **Summary**:
  JobKind を `latency_sensitive` / `default` / `bulk` の3 lane へ固定割当し、
  claim filter・schema・worker topology・観測性を lane 単位に隔離した。
  ADR-129 で execution lane・共有テーブル・JobKind 固定割当・worker topology・
  無停止 rollout・却下案（数値 priority、JobKind ごとの物理テーブル/専用
  binary、consumer 固有順序保証の非採用）を決定し、ADR-099 の concurrency
  model を部分置換した。`spec/contexts/jobs.yaml` に `ExecutionLane` enum、
  `models.Job.lane`、`ClaimJobs.lane` 入力、lane 隔離/未登録拒否/backfill の
  3 scenario を追加し、実装済みだが spec 未反映だった `provisioning_delivery`
  JobKind のドリフトも是正した。Domain（`RegisterKind(kind, lane)` による
  lane 検証、重複登録拒否）・Usecase/Adapter（`ClaimBatch` の lane 引数、
  memory/PostgreSQL 両方の lane-prefixed claim filter、`infra/schema/postgres.sql`
  の `lane` 列・CHECK・partial index）・Runtime（`idmagic-worker` の
  `JOB_WORKER_LANES` による lane 別 `Runner` 起動、`infra/k8s/base/worker.yaml`
  の3 Deployment・prod での replica 予約・PodDisruptionBudget）・Observability
  （`jobs_claim_latency_seconds`/`jobs_outcome_total`/`jobs_retry_total`/
  `jobs_queue_depth`、`/metrics` を idmagic-worker に新設、Prometheus
  rule・Grafana panel・ServiceMonitor）の全層を test-first で実装した。
  bulk lane を concurrency 超過分の長時間ジョブで飽和させても
  latency_sensitive lane が即座に claim・完了することを自動 integration
  test で確認した（`TestRunner_BulkBacklogDoesNotStarveLatencySensitive`）。
- **Disclosures (Out of Scope のまま)**:
  - JobKind ごとの物理テーブル、Kafka/NATS/SQS 等の別 broker、JobKind ごとの
    専用 worker binary は採用していない（ADR-129 却下案）。
  - lane 内の数値 priority・重み付きスケジューリング・tenant 間 fairness は
    実装していない（starvation リスクのため意図的に対象外、Risk Notes 参照）。
  - SCIM connection 単位など consumer 固有の順序保証・排他・rate limit は
    対象外。
  - 管理 API/UI（lane の表示・filter）は対象外。[[wi-157-job-admin-operations-surface]]
    と調整する。
  - `latency_sensitive` lane に割り当てる予定の `backchannel_logout_delivery`
    JobKind は [[wi-257-oidc-front-back-channel-logout-notifications]]
    （`status: pending`、未実装）待ちのため、現時点では `latency_sensitive`
    lane に実ジョブが流れる consumer が存在しない。lane 基盤自体は完成して
    おり、wi-257 実装時に `RegisterKind(..., LaneLatencySensitive)` を追加
    するだけで有効になる。
- **Verification Results**:
  - `just yaml-check`（scl / work-items / ids / architecture / traceability）— passed
  - `just scl-render` — passed
  - `just verify-go`（lint 0 issues、`go test -race` 全パッケージ）— passed。
    既知の未関連 failure: `TestAssembledRoutesMatchGeneratedOpenAPI`
    (`GET /session/check` spec-only) は本 work item 着手前の main で既に
    再現する既存の不具合であり、本 work item の変更とは無関係（`git stash`
    で確認済み）。
  - `just check-k8s dev` / `just check-k8s prod` — passed（kustomize build +
    kubeconform -strict、22 resources valid）
  - `just check-monitoring` — passed（promtool check rules 15 rules、
    grafana-dashboard.json 妥当、k8s monitoring/monitoring-operator
    kustomize build 成功）
  - `just check-compose` — passed
  - `just verify`（上記一括、UI 側 `verify-ui`/tools test/typecheck 含む）—
    passed（`/session/check` の既存 failure を除く）
