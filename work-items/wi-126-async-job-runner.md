---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-08
---

# テナント境界を保つ汎用非同期ジョブ基盤の core runtime を導入する

## Motivation
現状、時間のかかる処理はすべて HTTP リクエスト内で同期実行するしかなく、単発の
CSV 一括インポート ([[wi-96-bulk-user-import-csv]]) はサイズ上限で回避している。
しかし今後、非同期・バックグラウンド実行を要する処理は継続的に増える見込みがある:

- 大容量 CSV / ストリーミングでのユーザ一括インポート ([[wi-96-bulk-user-import-csv]])
- outbound SCIM プロビジョニングの再送・一括同期 ([[wi-45-outbound-scim-provisioning]])
- バックアップ / リストア・DR 実行 ([[wi-101-backup-restore-and-disaster-recovery]])
- soft-delete の猶予期間経過後の purge を lazy-on-access から定期実行へ
- retention / 監査ログの集約・エクスポート

これらを機能ごとに個別実装すると、リトライ・耐障害性・テナント分離・可視化 (進捗 /
失敗) がバラバラになり、二重管理と障害時の不整合を招く。本 WI は「1 つの汎用非同期
ジョブ基盤」を導入し、各機能はジョブ種別を登録するだけで durable なキューイング・
at-least-once 実行・リトライ・進捗可視化・水平スケールを共有できるようにする。

本 WI は core runtime のみに絞る。管理者向けジョブ一覧 / 詳細 / キャンセル UI、
運用メトリクス、runbook は [[wi-157-job-admin-operations-surface]] に分離する。
既存の同期処理の載せ替えは各機能側 WI で行う。最初の consumer は
[[wi-96-bulk-user-import-csv]] または [[wi-148-admin-resource-csv-export]] を想定する。

## Scope
- **decision**:
  - 新規 ADR: durable job queue を PostgreSQL の `SELECT ... FOR UPDATE
    SKIP LOCKED` によるリース方式で実装する判断。却下案 (Valkey Streams / 外部
    ブローカー NATS・Kafka / cron のみ) と、その理由を記録する。
  - 新規 ADR: worker 実行モデル（`idmagic-api`、既存の `idmagic-relay`、
    `idmagic-worker` のワークロード別プロセス分離。業務 bounded context は
    モジュラーモノリスとして維持し、水平スケールは worker replica を増設）と、配信保証
    (at-least-once + ハンドラ冪等性)、リース + heartbeat による失効ジョブ再取得、
    指数バックオフ付きリトライと max_attempts 超過時の dead-letter、
    graceful drain ([[ADR-078-kubernetes-health-probes-and-graceful-drain]]
    と整合) を記録する。
  - 新規 ADR: params / result / error に載る PII の at-rest
    方針と保持期間 (TTL / purge) を記録する。
- **scl**:
  - 新規 bounded context `Jobs` を追加する。`Jobs` は durable queue と worker runtime
    という技術的能力を所有し、各ジョブの業務ロジックは既存の所有 bounded context の
    usecase に残す。
  - §3.2 models: `Job` (集約)、`JobStatus` / `JobKind` enum、`JobProgress`、
    `JobRef` (published language) を追加する。
  - §3.3 interfaces: 内部 enqueue capability (`EnqueueJob`) と worker 内部の
    claim / heartbeat / complete / retry contract を定義する。管理者向け
    `ListJobs` / `GetJob` / `CancelJob` は [[wi-157-job-admin-operations-surface]]
    で扱う。
  - §3.4 states/events: 状態機械 `JobLifecycle` (Queued → Running →
    Succeeded / Failed / Canceled、Running → Queued の再試行遷移) と、
    `JobEnqueued` / `JobStarted` / `JobSucceeded` / `JobFailed` / `JobRetried` /
    `JobCanceled` を追加する。
  - §3.5 invariants: (a) リース排他 (1 ジョブは同時に 1 worker のみが Running)、
    (b) at-least-once + 冪等 (再試行で副作用が重複しない)、(c) テナント分離
    (worker は job.tenant_id の境界内でのみ副作用を起こす)、(d) 終端状態の不可逆性、
    (e) max_attempts 超過で dead-letter へ確定、を明示する。
  - 管理者 permission は本 WI では追加しない。`AdminJobsRead` / `AdminJobsCancel` は
    [[wi-157-job-admin-operations-surface]] の範囲。
- **architecture**:
  - 新規 context / worker プロセス / ディレクトリ規約を追加するため
    [ARCHITECTURE.md](../ARCHITECTURE.md) の map と details を同期する。
- **go**:
  - `Job` domain、`JobRepository` port、queue claim/lease/heartbeat/complete/retry の
    usecase、`JobKind` ごとの handler registry、worker pool (poll / concurrency /
    backoff / drain) を追加する。memory ランタイム用の in-process 実装と
    postgres_valkey ランタイム用の SKIP LOCKED 実装を用意する。
- **infrastructure / deploy**:
  - `idmagic-worker` の起動 entry point を追加し、worker を API と別プロセス /
    別レプリカで起動できるようにする。API、relay、worker は同じリポジトリと bounded
    context module を共有するが、API に worker を同居させない。ローカル compose に worker
    デプロイ単位を追加し、
    Kubernetes / Ansible の本格運用設定は [[wi-157-job-admin-operations-surface]] に送る。
    PostgreSQL schema に `jobs` テーブルを追加する。
  - 疎通確認用の no-op / echo job を追加し、外部 feature への載せ替えなしで worker
    lifecycle を検証できるようにする。

## Out of Scope
- 個別機能の非同期化そのもの (CSV インポート = [[wi-96-bulk-user-import-csv]]、
  outbound SCIM = [[wi-45-outbound-scim-provisioning]] 等) は各機能側 WI で行う。
  本 WI は基盤 + 最小の疎通確認用ジョブ種別 (no-op / echo) までとする。
- 管理者向けジョブ一覧 / 詳細 / キャンセル API、管理 UI、運用 runbook、メトリクス、
  Kubernetes / Ansible の本格運用設定。これらは
  [[wi-157-job-admin-operations-surface]] で扱う。
- cron / スケジュール実行 (定期起動) は本 WI では最小に留め、必要なら別 WI で拡張する。
- 外部メッセージブローカー (Kafka / NATS / SQS) への差し替え。将来 JobRepository port の
  別実装として追加可能にするが、本 WI では PostgreSQL 実装のみ。
- ワークフロー / DAG・ジョブ間依存・fan-out/fan-in などの上位オーケストレーション。

## Plan
- **キュー方式**: 既存の永続ランタイム (postgres_valkey) に合わせ、外部ブローカーを
  増やさず PostgreSQL の `FOR UPDATE SKIP LOCKED` でリース取得する durable queue と
  する。memory ランタイムは同一 port の in-process 実装で疎通させる。
- **プロセス境界**: `idmagic-api` は外部 HTTP API と同期の認証・認可・フェデレーション
  応答を担当し、`idmagic-relay` は outbox → Kafka の配送だけを担当する。
  `idmagic-worker` は durable job の実行だけを担当し、API から独立して水平スケールする。
  3 プロセスは同一の Go module と bounded context の実装を再利用する。認証、OAuth2、SAML、
  WS-Fed、Application 割当の同期依存はネットワーク RPC に置き換えず、API プロセス内に残す。
- **worker の責務**: worker は handler registry、スケジュール、claim、再試行、drain、観測を
  所有する。各 handler の業務ロジックは当該 bounded context の usecase を呼ぶだけにし、
  `Jobs` に業務規則を移さない。API プロセス内の定期 goroutine は worker へ移管する。
- **耐障害性**: リース + `lease_expires_at` + heartbeat。worker がクラッシュしても
  リース失効後に別 worker が再取得する。at-least-once のためハンドラは冪等必須とし、
  ジョブに任意の dedup key を持たせる。指数バックオフでリトライし、max_attempts 超過は
  dead-letter (Failed 終端 + error 保存)。shutdown 時は新規 claim を止め、in-flight は
  完了待ちか再キュー ([[ADR-078-kubernetes-health-probes-and-graceful-drain]] と整合)。
- **テナント分離**: `job.tenant_id` を必須とし、handler 実行 context を当該 tenant に
  固定する。worker は他テナントの集約に触れない。
- **却下した代替案**:
  - bounded context ごとのマイクロサービス化: 現在の認証 → OAuth2 / SAML / WS-Fed →
    Application 割当は同期の fail-closed 認可経路である。これを RPC 化すると可用性、
    レイテンシ、整合性の負担を増やすため、独立したデータ所有・チーム・SLO が成立するまで
    分割しない。
  - Valkey Streams / Redis: 既に Valkey は ephemeral state 用途 (throttle / session)
    で使うが、ジョブは durable な監査対象であり、消えては困る。永続ストアである
    PostgreSQL を真実源にする方が耐障害性の推論が容易。
  - 外部ブローカー (Kafka / NATS / SQS): 運用要素が増え、デモ IdP の起動容易性を落とす。
    将来 port の別実装として追加可能にするに留める。
  - cron のみ / 同期のまま: リトライ・進捗可視化・水平スケールを個別実装する羽目になる。
- **将来のサービス抽出基準**: 特定ワークロードに API と異なる SLO・スケール・障害隔離が
  継続的に必要であり、かつ独立したデータ所有と契約イベントだけで完結できる場合に限り、
  worker 化で安定させた後にサービス抽出を検討する。
- **未決定事項**:
  1. ジョブ params / result の PII 方針 (暗号化
     [[wi-97-envelope-encryption-at-rest]] との関係、保持期間)。
  2. worker のリース poll 間隔 / concurrency / backoff の既定値と設定注入点。
  3. `LISTEN/NOTIFY` によるプッシュ起動を初期から入れるか、poll のみで始めるか。
  4. admin ジョブ画面の詳細は [[wi-157-job-admin-operations-surface]] で決める。

## Tasks
- [x] T001 [Decision] キュー基盤 ADR (PostgreSQL SKIP LOCKED リース) を書く。
- [x] T002 [Decision] 実行モデル ADR (worker 分離 / at-least-once + 冪等 / リース +
      heartbeat / backoff + dead-letter / graceful drain) を書く。
- [x] T003 [Decision] ジョブデータ保持 ADR (params/result の PII / TTL) を書く。
- [x] T004 [SCL] `Jobs` context を追加し、models / internal interfaces / states /
      events / invariants と、API・relay・worker のプロセス責務を定義する。`just yaml-check`
      を通す。
- [x] T005 [Architecture] 新規 context / worker プロセス / ディレクトリ規約を
      ARCHITECTURE.md に同期する。
- [x] T006 [Go domain] `Job` 集約・`JobStatus`/`JobKind`・状態遷移・イベントを実装する。
- [x] T007 [Go ports] `JobRepository` (enqueue / claim-with-lease / heartbeat /
      complete / fail / retry / cancel / list / get) を定義する。
- [ ] T008 [Go usecase] enqueue・claim・handler registry・worker pool
      (poll / concurrency / backoff / drain) を実装し単体テストする。
- [ ] T009 [Adapter] memory (in-process) と postgres (SKIP LOCKED) の JobRepository を
      実装し、リース排他・失効再取得・冪等をテストする。
- [ ] T010 [Schema] `jobs` テーブルを deploy/schema/postgres.sql に追加する。
- [ ] T011 [Infra] `idmagic-worker` entry point を追加し、ローカル compose で API・relay・
      worker を分離起動できるようにする。API から retention sweep の goroutine を除去する。
- [ ] T012 [Smoke] retention sweep を最初の worker consumer として移管し、no-op / echo job
      を enqueue して Queued → Running → Succeeded と
      worker kill 後のリース失効再取得を確認できるテストを追加する。
- [ ] T013 [Verify] `just verify` を green にする。

## Verification
- `just yaml-check`
- `just verify-go` (lint + race テスト)。特に claim のリース排他・失効再取得・
  冪等再試行を並行テストで担保する。
- `just verify-ui`
- 手動: worker を API と別プロセスで起動し、疎通用ジョブ (no-op / echo) を enqueue →
  Running → Succeeded まで遷移すること、worker を kill してリース失効後に別 worker が
  再取得すること、max_attempts 超過で dead-letter へ確定することを確認する。

## Risk Notes
基盤ゆえに影響範囲が広く、並行制御 (リース排他) とテナント分離を誤ると、ジョブの
二重実行・別テナントへの副作用・失効ジョブの取りこぼしといった重大な不整合を招く。
`FOR UPDATE SKIP LOCKED` の排他とリース失効再取得を並行テストで担保し、ハンドラ冪等性を
契約として明示する。params / result に PII が載るため、保持期間と at-rest 方針
([[wi-97-envelope-encryption-at-rest]]) を ADR で確定してから実装する。worker
プロセス分離はデプロイ構成を増やすため、単一プロセス (in-process worker) でも動作する
縮退経路を残す。

## Progress

本 WI は risk: high の基盤 WI のため、RA の内層から外層へ層ごとに区切って段階実装する。
各フェーズ完了時にここへ記録し、全フェーズ完了時に `status: completed` + `Completion` を
追記して `done/` へ移す。

## 2026-07-12 — Phase A (Decision + SCL + Architecture) 完了

T001〜T005 を実施。

- **T001〜T003 ADR**: `decisions/ADR-098-durable-job-queue-skip-locked-lease.md`
  （PostgreSQL `FOR UPDATE SKIP LOCKED` によるリース方式、既存の
  `backend/shared/adapters/eventsink/kafka_relay.go` の claim パターンを踏襲する決定と、
  Valkey Streams / 外部ブローカー / cron のみの却下理由）、
  `decisions/ADR-099-job-worker-execution-model-and-fault-tolerance.md`
  （`idmagic-worker` プロセス分離、at-least-once + 冪等、リース + heartbeat 失効再取得、
  backoff + max_attempts 超過 dead-letter、[[ADR-078-kubernetes-health-probes-and-graceful-drain]]
  整合の graceful drain、poll-only の既定値、retention sweep の移管方法）、
  `decisions/ADR-100-job-data-retention-and-pii.md`
  （params/result は本 WI では暗号化せず平文 JSONB、終端ジョブは既定 30 日 TTL purge）を作成。
- **設計判断**: 既存の `backend/bootstrap/retention.go` の `startRetentionSweep` は
  テナント横断の一括処理であり、`jobs` テーブルの tenant_id 必須方針（tenant-owned
  aggregate）と相容れないため、`jobs` テーブルを経由する Job にはせず、
  `idmagic-worker` プロセスへそのまま再配置する方針とした（ADR-099）。Jobs の queue を
  実際に通す最初の consumer は疎通確認用の no-op/echo `JobKind` とする。
- **T004 SCL**: `spec/contexts/jobs.yaml` を新規作成し `Jobs` context を追加
  （`models`: `Job` (entity)・`JobStatus`/`JobKind` (enum)・`JobProgress`
  (value_object)・`JobRef` (published language)・6 種の event、`states`:
  `JobLifecycle`、`interfaces`: `EnqueueJob`/`ClaimJobs`/`HeartbeatJob`/`CompleteJob`/
  `FailJob`/`CancelJob` (いずれも内部インターフェースで HTTP binding なし、
  `Tenancy.ResolveTenant` と同型)、`invariants`: リース排他・ハンドラ冪等・テナント分離・
  終端状態不可逆・dead-letter 確定の 5 件、`objectives`: `JobRecordRetention`
  (delete_after 30d)、`scenarios`: 正常系・lease 失効再取得・dead-letter・テナント境界の
  4 件）。`spec/scl.yaml` の `context_map` に `Jobs`（`publishes: [JobRef]`、
  `depends_on: Tenancy`）を追加。
- **T005 Architecture**: `ARCHITECTURE.md` の Context Map テーブルに `Jobs` 行、
  Structural Decisions に ADR-098/099 への参照、Bootstrap And Adapters 節に
  `backend/cmd/idmagic-worker/main.go` の記述を追加。
- **検証**: `just yaml-check`（scl / work-items / ids / architecture）全て green。
- **対象外 (今回は触れない)**: T006 以降 (Go domain/ports/usecase/adapters、schema、
  `idmagic-worker` 実装、smoke test、`just verify`) は次フェーズ。

## 2026-07-12 — Phase B (Go domain + ports) 完了

T006〜T007 を実施。

- **T006 domain** (`backend/jobs/domain/job.go`, `events.go`): `Job` entity
  (`identity: id`)、`JobStatus`/`JobKind` (typed string + `Valid()`、既存の
  `backend/shared/spec/enums.go` 命名慣例に合わせ `StatusQueued` 等の短縮定数名)、
  `JobLifecycleEvent` と遷移表 `jobTransitions` + `TransitionJobLifecycle` /
  `IsJobLifecycleTerminal`（`backend/shared/spec/authorization_code_machine.go`
  の遷移表パターンを踏襲）。`NextRetryRunAt`（ADR-099 既定値: base 30s 指数
  バックオフ、cap 30分）をドメイン層の純粋関数として追加。6 種のドメインイベント
  struct (`JobEnqueued`/`JobStarted`/`JobSucceeded`/`JobFailed`/`JobRetried`/
  `JobCanceled`) は `backend/shared/spec.DomainEvent`
  (`EventType() string`/`OccurredAt() time.Time`) を構造的に満たす形で
  `backend/jobs/domain` に定義し、`shared/spec` への import は行わない
  （直近の慣例 (wi-178 の `AgentStatus`/`UserStatus` 移設等) に合わせ、
  context 固有の型は各 context の domain に置く方針、ADR-089 と整合）。
- **T007 ports** (`backend/jobs/ports/repository.go`): `JobRepository`
  interface (`Enqueue`/`ClaimBatch`/`Heartbeat`/`Complete`/`Fail`/`Cancel`/`Get`)。
  `FailOutcome`（retry か dead-letter かは usecase が `NextRetryRunAt` で計算して
  渡す）、sentinel error (`ErrJobNotFound`/`ErrJobLeaseLost`/`ErrJobAlreadyTerminal`)
  を ports 層に定義（lease 喪失は usecase の事前検証では検出できず、
  atomic な UPDATE の影響行数でしか判定できないための例外的配置、理由をコード
  コメントに明記）。WI 記載の "list" は SCL 側に admin 向け List interface が
  無く（wi-157 の範囲）、現時点で consumer が無いため YAGNI で見送った
  （Phase F の smoke test は `Get` で十分）。
- **検証**: `GOCACHE=/tmp/idmagic-cache go build ./...`、
  `GOCACHE=/tmp/idmagic-cache go test -race ./backend/jobs/...`
  （状態機械の全 (status, event) 総当たり invariant テストを含む）、
  `just lint-go` (0 issues) すべて green。
- **対象外 (今回は触れない)**: T008 以降 (usecase の enqueue/claim/handler
  registry/worker pool、memory/postgres adapter、schema、`idmagic-worker`、
  smoke test) は次フェーズ。

## 2026-07-12 — Phase B 追補: SCL vocabulary 漏れの修正

前回の Phase B 検証はパッケージスコープ (`./backend/jobs/...`) のみで、
リポジトリ全体の `just verify-go` は未実行だった。今回それを実行したところ
`backend/shared/spec` の `TestCurrentSCLIsInternallyCoherent` が
`state JobLifecycle: event Claim is missing from vocabulary` で red だった。

- **原因**: `spec/contexts/jobs.yaml` の `states.JobLifecycle` は
  `transitions[].from/event/to` に `Queued`/`Running`/`Claim` 等の PascalCase
  識別子を使うが、対応する `glossary` エントリが無かった。coherence
  検証 (`backend/shared/spec/coherence.go` `validateStates`) は状態機械の
  全 state / event 識別子が `glossary`（`vocabulary` として集約）に
  登録されていることを要求する（`backend/spec/contexts/oauth2.yaml` の
  `states.AuthorizationCodeFlow` も同様に `Received`/`StartAuthentication` 等を
  glossary へ個別登録している）。
- **修正**: `spec/contexts/jobs.yaml` の `glossary` に状態 5 件
  (`Queued`/`Running`/`Succeeded`/`Failed`/`Canceled`) とイベント 5 件
  (`Claim`/`Complete`/`Fail`/`Retry`/`Cancel`) を追加（各 `definition` +
  snake_case の `aliases`、oauth2.yaml の書式に合わせた）。Go 側
  (`backend/jobs/domain/job.go`) の変更は無し。
- **検証**: `just yaml-check`、`just verify-go`（lint 0 issues、
  `go test -race ./...` 全パッケージ green、`backend/shared/spec` の
  coherence テスト含む）、`just verify`（UI ビルド含む）すべて green。

残り T008〜T013 (Phase C〜F) は pending。
