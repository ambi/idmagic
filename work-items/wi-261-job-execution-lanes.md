---
status: pending
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
  `noop_echo/lifecycle_workflow_run=default` とする。将来の JobKind は
  SCL-first で lane を同時に宣言する。
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

- [ ] T001 [ADR] execution lane、共有テーブル、JobKind 固定割当、worker topology、
      無停止 rollout と却下案を決定する。
- [ ] T002 [SCL] Jobs の glossary/models/interfaces/objectives/scenarios を先に更新し、
      `just yaml-check-scl` と派生成果物の同期を行う。
- [ ] T003 [Domain] RED: JobKind ごとの lane 決定、未割当・不正 lane 拒否、既定割当の
      test を先に失敗させる（`models.Job` / `models.JobKind`）→ GREEN。
- [ ] T004 [Usecase/Adapter] RED: 対象 lane 以外を claim しないこと、lane 内の
      lease 排他、複数 lane worker の容量隔離、旧行 backfill を memory/PostgreSQL
      test で先に失敗させる（`interfaces.EnqueueJob` / `interfaces.ClaimJobs`）→ GREEN。
- [ ] T005 [Runtime/Infra] lane 選択設定、lane 別 worker process/deployment、
      development topology、graceful drain を実装する。
- [ ] T006 [Obs] lane 別 queue latency、depth、active、outcome、retry metrics と
      structured log を追加し、高 cardinality/PII label がないことを検証する。
- [ ] T007 [Docs] ADR、`ARCHITECTURE.md`、運用手順を実装後の現在形へ同期する。
- [ ] T008 [Verify] bulk backlog 中も latency-sensitive job が予約枠で実行される
      integration testを含め、全検証を通す。

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
