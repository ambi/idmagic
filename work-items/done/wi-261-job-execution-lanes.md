---
status: completed
archived_from: 61f5d0fe18c17eadc4a23f1b9048389f098c55ca
---

# ジョブ実行レーンごとに処理能力を分離する

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

全文は `git show 61f5d0fe18c17eadc4a23f1b9048389f098c55ca:<このファイルのパス>` で読む。
