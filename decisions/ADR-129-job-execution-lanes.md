---
status: accepted
authors: [tn]
created_at: 2026-07-19
supersedes: [ADR-099]
---

# ADR-129: JobKind を lane に固定割当し、lane 別 claim filter と worker topology で実行枠を隔離する

## コンテキスト

[[ADR-098-durable-job-queue-skip-locked-lease]] は単一 `jobs` テーブル上の
`SKIP LOCKED` durable queue を、[[ADR-099-job-worker-execution-model-and-fault-tolerance]]
は `idmagic-worker` 1 プロセス・単一 `Concurrency` semaphore・`ClaimBatch` が
`JobKind` を絞らないことを決めた。現状は `backchannel_logout_delivery` /
`user_import_preview` / `user_import_apply` / `dynamic_group_reconcile` /
`noop_echo` / `lifecycle_workflow_run` の全 `JobKind` が同じ `run_at` 順
backlog・同じ実行枠を共有する。

CSV import・動的 Group 再計算・full resync のような長時間・大量ジョブが
backlog を占有すると、低遅延が必要な配送ジョブが同じ backlog の後ろで
待たされる。worker を水平スケールしても全 instance が全 `JobKind` を
処理するため、特定の負荷クラスへ実行枠を予約できない。単一 queue・
at-least-once・lease・retry/dead-letter という ADR-098/099 の基盤を
崩さずに、レイテンシ・資源特性が異なる `JobKind` 間で障害と容量を
隔離する必要がある。

## 決定

1. **lane は 3 種、`JobKind` に 1 対 1 固定**: `latency_sensitive` /
   `default` / `bulk`。enqueue 呼び出し元は lane を指定できない
   （lane は `JobKind` 登録情報からのみ決まる）。初期割当は
   `backchannel_logout_delivery=latency_sensitive`、
   `user_import_preview` / `user_import_apply` /
   `dynamic_group_reconcile=bulk`、`noop_echo` / `lifecycle_workflow_run` /
   `provisioning_delivery=default`。`provisioning_delivery` は
   Motivation が低遅延要件を明示するのは logout notification 配送だけ
   であり、SCIM 配送は `backchannel_logout_delivery` ほどの遅延要件を
   持たないため `default` とする。新規 `JobKind` は登録時に lane を
   同時に宣言しなければならず、未割当・重複登録は startup/test で拒否する。
2. **共有 `jobs` テーブルを維持し `lane TEXT NOT NULL` を追加**する。
   claim は対象 lane を必須条件とする。claimable index を
   `(lane, run_at) WHERE status='queued'` と
   `(lane, lease_expires_at) WHERE status='running'` の lane-prefixed
   partial index に置き換える。同一 lane 内は現行どおり due `run_at` の
   古い順で、lane を跨いだ厳密な開始順・完了順は保証しない
   （ADR-098 の SKIP LOCKED 順序保証をそのまま lane 内に閉じ込める）。
3. **`Runner` を lane スコープの実行単位にする**: 1 `Runner` インスタンスは
   1 lane だけを claim・実行し、自分の `Concurrency` semaphore を持つ。
   1 process は複数 `Runner` を同時に起動できる（compat mode = 1 process が
   3 lane 全部の `Runner` を持つ）し、1 lane だけを持つ dedicated deployment
   にもできる。`JobRepository.ClaimBatch` は lane 引数を必須にする。
4. **worker topology は lane 別 deployment とし、`latency_sensitive` に
   専用の最小実行枠を予約する**: production は `idmagic-worker` を lane 別
   3 Deployment（`idmagic-worker-latency-sensitive` /
   `idmagic-worker-default` / `idmagic-worker-bulk`）に分け、各々
   独立した `JOB_WORKER_LANE` 環境変数・concurrency・replica 数を持つ。
   `latency_sensitive` deployment は bulk backlog の有無に関わらず常時
   稼働する専用 replica を持ち、他 lane の実行枠を消費しない。development
   （`just dev` の Docker なし標準環境、`just dev-compose`）は compat mode
   の単一 worker process が 3 lane 全部を claim し、[[ADR-107-audit-retention-and-jobs-dev-environment-topology]]
   の「API と worker は同じ実データ store を共有する」制約をそのまま
   維持する。
5. **無停止 rollout は 4 段階を順に適用**し、どの段階でも claim
   されないジョブを作らない: (a) `lane TEXT NOT NULL DEFAULT 'default'` で
   既存行を backfill できる schema を先に適用する（旧 binary が
   lane 非対応のまま行を insert しても `default` へ落ちる）、
   (b) 新 binary は compat mode（1 process が全 lane を claim）で
   デプロイし、lane 列を書ける状態と全 lane 処理を両立させる、
   (c) lane 別 worker deployment へ切り替える、(d) `JobKind` の
   最終 lane 割当を適用する。各段階の gate は lane 別 queued 件数の
   監視とする。

## 却下した代替案

- **`JobKind` ごとの物理テーブル**: `JobRepository`・claim query
  ([[ADR-098-durable-job-queue-skip-locked-lease]]) を `JobKind` の数だけ
  複製することになり、SKIP LOCKED を単一 queue に閉じ込めた ADR-098 の
  単純さを失う。lane が増減するたびに migration も増える。
- **Kafka/NATS/SQS 等の別 broker への移行**: ADR-098 で却下した理由
  （運用コスト、durable queue を PostgreSQL の外に持ち出す複雑さ）が
  そのまま当てはまる。lane 3 種の隔離だけなら broker 追加は過剰。
- **`JobKind` ごとの専用 worker binary**: ADR-099 で決めた
  「`idmagic-api` / `idmagic-relay` / `idmagic-worker` の 3 プロセス境界を
  超えてサービス分割しない」方針に反する。同一 binary・同一
  `RunnerConfig` の lane 引数で十分に隔離できる。
- **lane 内の数値 priority・重み付きスケジューリング**: 厳密優先度は
  低優先度ジョブを starvation させる（Risk Notes 参照）。lane による
  hard isolation の方が、bulk backlog がどれだけ滞留しても
  `latency_sensitive` の実行枠を奪わないという保証を単純に持てる。
- **enqueue 呼び出し元が lane を指定できるようにする**: 任意の
  呼び出し元が `latency_sensitive` を自称できると実行枠予約の保証が
  崩れる。lane は `JobKind` 登録情報だけが決める。
- **consumer 固有の順序保証・排他・rate limit（例: SCIM connection 単位）
  を lane 機構に含める**: lane は負荷クラス間の容量隔離が目的で、
  consumer 内の fairness とは別の関心事。[[wi-157-job-admin-operations-surface]]
  など個別の work item に委ねる。

## 影響

- `spec/contexts/jobs.yaml` の `models.Job`（`lane` field 追加）、
  `models.JobKind`（enum の実装ドリフト是正を含む——`provisioning_delivery`
  は `backend/provisioning/usecases/job_handler.go` が
  `jobsdomain.RegisterKind` で既に実行時登録している稼働中の `JobKind`
  だが spec の enum に反映されていなかった。`backchannel_logout_delivery`
  は [[wi-257-oidc-front-back-channel-logout-notifications]]
  （`status: pending`、未実装）向けの正しい先行宣言であり変更しない。
  この work item で `provisioning_delivery` を spec 側に追加し実装へ
  合わせる）、`interfaces.ClaimJobs`（lane 引数・lane 単位の `ensures`）に
  反映する。
- `infra/schema/postgres.sql` の `jobs` テーブルへ `lane` 列と
  lane-prefixed partial index を追加する。
- `backend/jobs/ports/repository.go` の `ClaimBatch` シグネチャ、
  `backend/jobs/usecases/runner.go` の `RunnerConfig`（lane 追加）、
  `backend/jobs/domain/job.go` の `JobKind` registry（lane metadata 検証）
  を変更する。
- `infra/k8s/` に `idmagic-worker` の Deployment が現状存在しないため、
  lane 別 3 Deployment を新設する（base + dev/prod overlay）。
- [[ADR-099-job-worker-execution-model-and-fault-tolerance]] の
  concurrency model（項目 3: 単一 `Concurrency` semaphore、
  環境変数 `JOB_WORKER_CONCURRENCY`）を lane 単位に拡張する。ADR-099 の
  他の決定（プロセス境界、at-least-once、lease/heartbeat、backoff/
  dead-letter、graceful drain、poll-only、Docker なし標準開発環境）は
  そのまま維持する。
- `ARCHITECTURE.md` の「Durable Job Worker」節（lane 非対応の現状記述）を
  実装後の現在形に同期する。
