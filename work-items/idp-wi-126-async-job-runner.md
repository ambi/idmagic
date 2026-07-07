---
id: idp-wi-126-async-job-runner
title: "汎用非同期ジョブ基盤 (durable queue + worker) を導入する"
created_at: 2026-07-08
authors: ["tn"]
status: pending
risk: high
---

# テナント境界を保つ汎用非同期ジョブ基盤 (durable queue + worker) を導入する

## Motivation
現状、時間のかかる処理はすべて HTTP リクエスト内で同期実行するしかなく、単発の
CSV 一括インポート ([[idp-wi-96-bulk-user-import-csv]]) はサイズ上限で回避している。
しかし今後、非同期・バックグラウンド実行を要する処理は継続的に増える見込みがある:

- 大容量 CSV / ストリーミングでのユーザ一括インポート ([[idp-wi-96-bulk-user-import-csv]])
- outbound SCIM プロビジョニングの再送・一括同期 ([[idp-wi-45-outbound-scim-provisioning]])
- バックアップ / リストア・DR 実行 ([[idp-wi-101-backup-restore-and-disaster-recovery]])
- soft-delete の猶予期間経過後の purge を lazy-on-access から定期実行へ
- retention / 監査ログの集約・エクスポート

これらを機能ごとに個別実装すると、リトライ・耐障害性・テナント分離・可視化 (進捗 /
失敗) がバラバラになり、二重管理と障害時の不整合を招く。本 WI は「1 つの汎用非同期
ジョブ基盤」を導入し、各機能はジョブ種別を登録するだけで durable なキューイング・
at-least-once 実行・リトライ・進捗可視化・水平スケールを共有できるようにする。

本 WI は基盤のみを提供し、既存の同期処理の載せ替えは各機能側 WI で行う。最初の
consumer は [[idp-wi-96-bulk-user-import-csv]] を想定する。

## Scope
- **decision**:
  - 新規 ADR (キュー基盤): durable job queue を PostgreSQL の `SELECT ... FOR UPDATE
    SKIP LOCKED` によるリース方式で実装する判断。却下案 (Valkey Streams / 外部
    ブローカー NATS・Kafka / cron のみ) と、その理由を記録する。
  - 新規 ADR (実行モデル): worker 実行モデル (同一バイナリの run mode 分離 +
    in-process worker pool、水平スケールは worker replica 増設)、配信保証
    (at-least-once + ハンドラ冪等性)、リース + heartbeat による失効ジョブ再取得、
    指数バックオフ付きリトライと max_attempts 超過時の dead-letter、
    graceful drain ([[idp-ADR-078-kubernetes-health-probes-and-graceful-drain]]
    と整合) を記録する。
  - 新規 ADR (ジョブデータ保持): params / result / error に載る PII の at-rest
    方針と保持期間 (TTL / purge) を記録する。
- **scl**:
  - 新規 bounded context (仮称 `Jobs`) を追加する想定。context 追加の是非
    (専用 context か技術共有 context
    [[idp-ADR-070-technical-shared-context-for-cross-context-adapters]] への相乗りか)
    は Plan の未決定事項とする。
  - §3.2 models: `Job` (集約)、`JobStatus` / `JobKind` enum、`JobProgress`、
    `JobRef` (published language) を追加する。
  - §3.3 interfaces: 管理者向け `ListJobs` / `GetJob` / `CancelJob` (admin API) と、
    内部 enqueue capability (`EnqueueJob`) を追加する。enqueue は他 context が
    published language 経由で呼ぶ内部 interface として定義する。
  - §3.4 states/events: 状態機械 `JobLifecycle` (Queued → Running →
    Succeeded / Failed / Canceled、Running → Queued の再試行遷移) と、
    `JobEnqueued` / `JobStarted` / `JobSucceeded` / `JobFailed` / `JobRetried` /
    `JobCanceled` を追加する。
  - §3.5 invariants: (a) リース排他 (1 ジョブは同時に 1 worker のみが Running)、
    (b) at-least-once + 冪等 (再試行で副作用が重複しない)、(c) テナント分離
    (worker は job.tenant_id の境界内でのみ副作用を起こす)、(d) 終端状態の不可逆性、
    (e) max_attempts 超過で dead-letter へ確定、を明示する。
  - `permissions`: `AdminJobsRead` / `AdminJobsCancel` を追加する。
- **architecture**:
  - 新規 context / worker プロセス / ディレクトリ規約を追加するため
    [ARCHITECTURE.md](../ARCHITECTURE.md) の map と details を同期する。
- **go**:
  - `Job` domain、`JobRepository` port、queue claim/lease/heartbeat/complete/retry の
    usecase、`JobKind` ごとの handler registry、worker pool (poll / concurrency /
    backoff / drain) を追加する。memory ランタイム用の in-process 実装と
    postgres_valkey ランタイム用の SKIP LOCKED 実装を用意する。
- **http**:
  - admin の `GET /api/admin/jobs` / `GET /api/admin/jobs/{job_id}` /
    `POST /api/admin/jobs/{job_id}/cancel` を追加する。
- **infrastructure / deploy**:
  - `MODE=worker` 相当の run mode を bootstrap に追加し、worker を API と別プロセス /
    別レプリカで起動できるようにする。compose / Ansible / K8s manifest に worker
    デプロイ単位を追加する。PostgreSQL schema に `jobs` テーブルを追加する。
- **ui**:
  - admin にジョブ一覧 / 詳細 (進捗・失敗理由・リトライ状況・キャンセル) 画面を追加する。
- **documentation**:
  - README に worker プロセスの起動方法とジョブ運用 (スケール・監視) を追記する。

## Out of Scope
- 個別機能の非同期化そのもの (CSV インポート = [[idp-wi-96-bulk-user-import-csv]]、
  outbound SCIM = [[idp-wi-45-outbound-scim-provisioning]] 等) は各機能側 WI で行う。
  本 WI は基盤 + 最小の疎通確認用ジョブ種別 (no-op / echo) までとする。
- cron / スケジュール実行 (定期起動) は本 WI では最小に留め、必要なら別 WI で拡張する。
- 外部メッセージブローカー (Kafka / NATS / SQS) への差し替え。将来 JobRepository port の
  別実装として追加可能にするが、本 WI では PostgreSQL 実装のみ。
- ワークフロー / DAG・ジョブ間依存・fan-out/fan-in などの上位オーケストレーション。

## Plan
- **キュー方式**: 既存の永続ランタイム (postgres_valkey) に合わせ、外部ブローカーを
  増やさず PostgreSQL の `FOR UPDATE SKIP LOCKED` でリース取得する durable queue と
  する。memory ランタイムは同一 port の in-process 実装で疎通させる。
- **実行モデル**: 同一バイナリに run mode を追加し、worker を API と別プロセス /
  別レプリカで水平スケールさせる。単一ノード / dev では in-process worker も可能にする。
- **耐障害性**: リース + `lease_expires_at` + heartbeat。worker がクラッシュしても
  リース失効後に別 worker が再取得する。at-least-once のためハンドラは冪等必須とし、
  ジョブに任意の dedup key を持たせる。指数バックオフでリトライし、max_attempts 超過は
  dead-letter (Failed 終端 + error 保存)。shutdown 時は新規 claim を止め、in-flight は
  完了待ちか再キュー ([[idp-ADR-078-kubernetes-health-probes-and-graceful-drain]] と整合)。
- **テナント分離**: `job.tenant_id` を必須とし、handler 実行 context を当該 tenant に
  固定する。worker は他テナントの集約に触れない。
- **却下した代替案**:
  - Valkey Streams / Redis: 既に Valkey は ephemeral state 用途 (throttle / session)
    で使うが、ジョブは durable な監査対象であり、消えては困る。永続ストアである
    PostgreSQL を真実源にする方が耐障害性の推論が容易。
  - 外部ブローカー (Kafka / NATS / SQS): 運用要素が増え、デモ IdP の起動容易性を落とす。
    将来 port の別実装として追加可能にするに留める。
  - cron のみ / 同期のまま: リトライ・進捗可視化・水平スケールを個別実装する羽目になる。
- **未決定事項**:
  1. 新規 bounded context (`Jobs`) を切るか、技術共有 context に相乗りさせるか。
  2. ジョブ params / result の PII 方針 (暗号化
     [[idp-wi-97-envelope-encryption-at-rest]] との関係、保持期間)。
  3. worker のリース poll 間隔 / concurrency / backoff の既定値と設定注入点。
  4. `LISTEN/NOTIFY` によるプッシュ起動を初期から入れるか、poll のみで始めるか。
  5. admin ジョブ画面をどこまで作るか (一覧 + 詳細 + キャンセルの最小か、リアルタイム
     進捗まで含めるか)。

## Tasks
- [ ] T001 [Decision] キュー基盤 ADR (PostgreSQL SKIP LOCKED リース) を書く。
- [ ] T002 [Decision] 実行モデル ADR (worker 分離 / at-least-once + 冪等 / リース +
      heartbeat / backoff + dead-letter / graceful drain) を書く。
- [ ] T003 [Decision] ジョブデータ保持 ADR (params/result の PII / TTL) を書く。
- [ ] T004 [SCL] `Jobs` context を追加し、models / interfaces / states / events /
      invariants / permissions を定義する。`just yaml-check` を通す。
- [ ] T005 [Architecture] 新規 context / worker プロセス / ディレクトリ規約を
      ARCHITECTURE.md に同期する。
- [ ] T006 [Go domain] `Job` 集約・`JobStatus`/`JobKind`・状態遷移・イベントを実装する。
- [ ] T007 [Go ports] `JobRepository` (enqueue / claim-with-lease / heartbeat /
      complete / fail / retry / cancel / list / get) を定義する。
- [ ] T008 [Go usecase] enqueue・claim・handler registry・worker pool
      (poll / concurrency / backoff / drain) を実装し単体テストする。
- [ ] T009 [Adapter] memory (in-process) と postgres (SKIP LOCKED) の JobRepository を
      実装し、リース排他・失効再取得・冪等をテストする。
- [ ] T010 [Schema] `jobs` テーブルを deploy/schema/postgres.sql に追加する。
- [ ] T011 [HTTP] admin `ListJobs` / `GetJob` / `CancelJob` を実装しテストする。
- [ ] T012 [Infra] bootstrap に worker run mode を追加し、compose / K8s に worker
      デプロイ単位を追加する。
- [ ] T013 [UI] admin ジョブ一覧 / 詳細 (進捗・失敗・キャンセル) 画面を追加する。
- [ ] T014 [Docs] README に worker 起動とジョブ運用を追記する。
- [ ] T015 [Verify] `just verify` を green にする。

## Verification
- `just yaml-check`
- `just verify-go` (lint + race テスト)。特に claim のリース排他・失効再取得・
  冪等再試行を並行テストで担保する。
- `just verify-ui`
- 手動: worker を API と別プロセスで起動し、疎通用ジョブ (no-op) を enqueue →
  Running → Succeeded まで遷移すること、worker を kill してリース失効後に別 worker が
  再取得すること、max_attempts 超過で dead-letter へ確定することを確認する。

## Risk Notes
基盤ゆえに影響範囲が広く、並行制御 (リース排他) とテナント分離を誤ると、ジョブの
二重実行・別テナントへの副作用・失効ジョブの取りこぼしといった重大な不整合を招く。
`FOR UPDATE SKIP LOCKED` の排他とリース失効再取得を並行テストで担保し、ハンドラ冪等性を
契約として明示する。params / result に PII が載るため、保持期間と at-rest 方針
([[idp-wi-97-envelope-encryption-at-rest]]) を ADR で確定してから実装する。worker
プロセス分離はデプロイ構成を増やすため、単一プロセス (in-process worker) でも動作する
縮退経路を残す。
