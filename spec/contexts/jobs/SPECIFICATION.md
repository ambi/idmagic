---
context: jobs
updated_at: 2026-08-11
---

# Jobs Specification

## Overview

テナント境界を保つ汎用非同期ジョブ基盤の永続キューと `worker` の実行環境を所有する。各ジョブの業務ロジックは呼び出し元の Bounded Context のユースケースに残し、Jobs はキュー投入、取得、リース、再試行、進捗表示という技術的能力だけを提供する。管理者向けの一覧・詳細・キャンセル API と運用メトリクスは別の機能単位が所有する。

業務処理は一切所有しない。所有するのは、テナントが持つあらゆる非同期作業に共通する実行基盤である。投入、永続化、取得、リース、ハートビート、再試行、配信不能への退避、キャンセルがこれに当たる。JobKind のパラメーターを解釈して副作用を起こす処理は、利用側 Context のユースケースに残す。`backend/cmd/idmagic-worker/worker.go` は起動時にそれらのハンドラーを一覧へ登録する。API プロセスはジョブを投入するが実行せず、`worker` プロセスはジョブを実行するが HTTP を提供しない。

業務ロジックを基盤の外に置くのは、ジョブ基盤に触れるたびに利用側のすべての Context を読み直さずに済むようにするためである。モジュール境界はパスから推論し、禁止されたインポートがないかを検査する。

```text
API / 利用側のユースケース
  └─ EnqueueJob（レーンは種別から導出）
       └─ JobRepository ──> PostgreSQL の jobs（lane 列、レーンを先頭にしたインデックス）
                                  │
idmagic-worker                    │ レーンごとに独立してポーリング
  ├─ ライフサイクルワークフローのディスパッチャー │（未投入の実行を回収）
  ├─ Runner (lane=latency_sensitive) ── ClaimBatch(lane) <───┘
  ├─ Runner (lane=default)          ── ClaimBatch(lane) <───┘
  ├─ Runner (lane=bulk)             ── ClaimBatch(lane) <───┘
  │    ├─ HandlerRegistry ──> 利用側 Context のユースケース（共有するハンドラー一覧）
  │    ├─ Heartbeat ────────> lease_expires_at を延長
  │    └─ Complete / Fail ──> succeeded / queued（再試行）/ failed
  └─ jobsQueueDepthSamplingLoop ──> レーンごとのキュー深度・実行中ゲージ（10 秒間隔）
```

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| JobLease | `worker` が Job を Running にする際に確保する排他権。`lease_owner`（`worker` 識別子）と `lease_expires_at` を持ち、ハートビートで更新する。ハートビートが途絶えて期限切れになると、別の `worker` が再取得できる。 | lease, リース |
| DeadLetter | `attempts` が `max_attempts` を超えて `Failed` に確定した Job。再試行されず、エラーを保持したまま調査対象として残る。 | 配信不能 |
| Queued | Job が投入され、まだ `worker` に取得されていない初期状態。`run_at` を過ぎると取得できる。 | queued |
| Running | `worker` がリースを確保し、ハンドラーを実行中の状態。 | running |
| Succeeded | ハンドラが正常終了した終端状態。 | succeeded |
| Failed | `attempts` が `max_attempts` を超えて配信不能に確定した終端状態。 | failed |
| Canceled | 終端状態に達する前に取消された終端状態。 | canceled |
| Claim | `worker` が実行可能な Job を取得し、自身を `lease_owner` として Running へ遷移させる。 | claim |
| Complete | `worker` がハンドラーの正常終了を報告し、Job を Succeeded に確定する。 | complete |
| Fail | `worker` がハンドラーの失敗を報告する。`attempts` が `max_attempts` 未満なら Retry、以上なら配信不能（Failed）に確定する。 | fail |
| Retry | 失敗後、バックオフを経て再試行のため Queued に戻す遷移。 | retry |
| Cancel | 終端状態に達していない Job を取消す。 | cancel |
| ExecutionLane | JobKind の登録情報が一意に決める実行枠の区分。レイテンシーや資源特性が異なる JobKind 間で、取得処理と `worker` の実行枠を分離するために使う。投入元はレーンを指定できない。`latency_sensitive` / `default` / `bulk` の 3 種類がある。 | lane, 実行レーン |
| Developer | 標準開発環境でこのリポジトリを動かす開発者。 |  |
| System | Jobs の永続キューと `worker` の実行環境そのもの。人間の操作者を伴わない技術的な主体を指す。 |  |

## State Transitions

### JobLifecycle

Job の状態遷移を表す。`queued` から `running` へは `worker` による取得で遷移し、`running` から `queued` へはバックオフを伴う再試行で戻る。`succeeded`、`failed`、`canceled` は不可逆の終端状態である。`attempts` が `max_attempts` に達した場合は `running` から配信不能を表す `failed` へ、未達の場合は再試行を表す `queued` へ遷移する。

Initial: `queued` Terminal: `succeeded`, `failed`, `canceled`

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
呼び出し元の Bounded Context が非同期処理をキューへ投入するための内部インターフェース。HTTP には公開せず、同一プロセス内の Go 呼び出しとして各 Context のユースケースから使う。`dedup_key` を指定した場合、`(tenant_id, dedup_key)` が既存の未終端 Job と一致すれば新しく作成せず、既存の JobRef を返す。作成する Job のレーンは JobKind の登録情報から一意に決まり、呼び出し元は指定できない。

#### ClaimJobs
`worker` が指定したレーン内で取得可能な Job、すなわち `status=queued` かつ `run_at <= now` の Job、またはリースが失効した `running` の Job を一括取得する内部インターフェース。同一トランザクション内で `Running` に遷移させ、自身を `lease_owner` として `lease_expires_at` を設定する。対象レーン以外の Job は取得しない。
- Result invariant: all_jobs_leased_by(output.jobs, input.worker_id)
- Result invariant: single_worker_holds_active_lease(output.jobs)
- Result invariant: all_jobs_in_lane(output.jobs, input.lane)

#### HeartbeatJob
`worker` が実行中の Job のリースを延長する。

#### CompleteJob
`worker` がハンドラーの正常終了を報告し、Job を Succeeded に確定する。

#### FailJob
`worker` がハンドラーの失敗を報告する。`attempts < max_attempts` ならバックオフ後の `run_at` を設定して Queued（Retry）へ、`attempts >= max_attempts` なら Failed（配信不能）へ確定する。
- Result invariant: output.status == 'failed' || output.status == 'queued'

#### CancelJob
終端状態に達していない Job を取消す。既に終端状態の Job には作用しない (不可逆)。

### Execution lanes

`JobKind` はちょうど 1 つの `ExecutionLane` (`latency_sensitive`、`default`、`bulk`) を持ち、`domain.RegisterKind(kind, lane)` による登録の時点で固定される。投入する呼び出し元がこれを選ぶことはできない。`Job.Lane` は種別の登録から導かれ、取得は自分の系統の中のジョブだけを対象にする。呼び出し元を意図的に排除しているのは、系統が呼び出しごとの優先度ではなく容量を隔離する単位だからである。

各レーンは独自の並行数上限を持つ独立した `Runner` を備え、空いている実行枠の数だけジョブを一括取得して、ハンドラーを並行実行する（デフォルトは 4 枠）。1 つのプロセスで複数レーンの `Runner` を同時に起動することもできる。これは `JOB_WORKER_LANES` が未設定の場合の互換動作であり、開発環境と Docker Compose のデフォルトでもある。1 つのレーンの `Runner` だけを持つ専用の Deployment を動かすこともでき、こちらを本番のデフォルトとする。`infra/k8s/base/worker.yaml` が `idmagic-worker-{latency-sensitive,default,bulk}` の 3 つの Deployment を定義し、レーンごとの並行数は `JOB_WORKER_CONCURRENCY_<LANE>` で与える。

系統をまたぐ順序は目的ではない。狙いは容量の隔離であり、`bulk` の滞留がどれだけ積み上がっても `latency_sensitive` の実行の枠を奪うことはない。同じ理由で系統の中にも数値の優先度は存在しない。系統の中の取得の候補はおおむね `run_at` の古い順だが、並行実行、複数プロセス、同じ時刻を持つジョブがあるため、開始の順序も完了の順序も保証しない。

### Claim and lease

`ClaimBatch` は自分のレーンに限り、実行時刻に達した `queued` のジョブと、リースが切れた `running` のジョブを取得する。PostgreSQL では `WHERE lane = $lane AND (...) ORDER BY run_at FOR UPDATE SKIP LOCKED` を同じ文の `running` への更新と組み合わせることで、2 つの `worker` が同じジョブに対して有効なリースを同時に持つことを防ぐ。永続キューは専用の仲介サービスではなく PostgreSQL に置き、ミドルウェアを増やす代わりに、安全な並行取得のためにコードベースの他の箇所ですでに使っている `FOR UPDATE SKIP LOCKED` と同じ方式を再利用する。

実行の保証は **少なくとも 1 回** である。取得のたびに `attempts` を増やし、`lease_owner` と `lease_expires_at` を設定する。実行中のハンドラーはリース期間の 3 分の 1 ごとにハートビートを送り、完了させられるのはリースの所有者だけである。プロセスが異常終了したり停止させられたりしてハートビートが止まると、リースの期限が切れた後に別の `worker` がそのジョブを再取得できる。したがって **ハンドラーは冪等でなければならない**。重複排除キーと、利用側自身の一貫性境界を使うこと。

### Retry and dead-letter

失敗したジョブは指数的に後ろ倒しした `run_at` とともに `queued` へ戻り、`max_attempts` に達すると `failed` として確定する。取得の間隔、プロセス内の並行度、リース、再試行の間隔はプロセス全体の設定である。`JOB_POLL_INTERVAL`、`JOB_WORKER_CONCURRENCY`、`JOB_LEASE_DURATION`、`JOB_BACKOFF_BASE`、`JOB_BACKOFF_CAP` がそれにあたる。現時点では JobKind ごとの品質の制御も、利用側ごとの順序や流量の制限も提供しない。

SIGTERM や SIGINT を受けると `worker` は取得をやめ、停止猶予期間まで実行中のハンドラーを待つ。猶予期間の後は終了し、回復は明示的な再投入ではなくリースの自然な期限切れによって起こる。明示的な再投入を避けるのは、停止処理の打ち切りとハンドラーの完了が競合したときに二重実行の余地が生じるからである。

### Metrics

`idmagic-worker` は `/metrics` (`MetricsExposition`、system.yaml) を管理専用の別の HTTP の待ち受けで公開する。idmagic-api の `/metrics` とは別のプロセスかつ別の実体である。系統ごとの `jobs_claim_latency_seconds`、`jobs_outcome_total`、`jobs_retry_total`、`jobs_queue_depth` を持つ。`tenant_id` と `job_id` は取りうる値を有限に保つためラベルから除外する。

### Boundary with scheduled batch

全テナントを対象とする定期的な保持期間の処理と署名鍵のライフサイクルの処理は、永続的なジョブに混ぜない。代わりに外部の予定表が `idmagic-batch` を 1 回限りで起動する。横断的な一掃にはテナントごとの作業の単位が存在しないので、テナントが持つ待ち行列へ流し込むと待ち行列の深さも系統の隔離も意味を失う。

`worker` 内のライフサイクルワークフローの割り当て処理は、業務処理を直接行う定期バッチではなく、回復経路である。同じトランザクションで確定したものの Job と関連付けられなかった WorkflowRun を再走査し、永続キューへ安全に引き渡す。

### Schema notes

`Job` は自然キーの親を持たないが、`jobs.tenant_id` は必須である。リポジトリ全体の仕様にある [`tenant_id` retention classes](../../SPECIFICATION.md#2-tenant_id-retention-classes) のうち、テナントが所有する Aggregate の分類に従う。`status` と `kind` は本書が規範として定める閉じた語彙であり、`CHECK` 制約はそれと並ぶ多層防御であって正ではない。`params` と `result` は `JobKind` ごとに解釈する不透明な JSONB であり、保存時に暗号化しない。終端行は専用の Job ではなく、`worker` の保持期間スイープが有効期間の経過後に削除する。`lane` を省略した行には `DEFAULT 'default'` を適用するが、通常の投入処理は `JobKind` の登録情報からレーンを明示する。`dedup_key` は `JobHandlerIdempotency` を支える。部分一意インデックスにより、`(tenant_id, dedup_key)` ごとに未終端の Job は最大 1 つに限られる。

### Design Decisions

- 永続キューは PostgreSQL の `FOR UPDATE SKIP LOCKED` によるリースを使う。第二のキューデータストアを増やさずに、並行する `worker` が別々の行を取得できるようにするためである。
- `idmagic-worker` は API とは別のプロセスの境界である。配送は少なくとも 1 回であり、冪等性はハンドラーが担い、停止時にはリースが切れる前に取得済みの作業を捌き切る。
- 予定実行の 1 回限りの保持期間と鍵のライフサイクルの処理は `idmagic-batch` に属する。Jobs が所有するのは、あらゆる予定実行の命令ではなく、永続的で再試行できる非同期のアプリケーションの作業である。
- 実行の系統は、待ち行列の実装を別々に作ることなく、遅延に敏感な作業、通常の作業、大量の作業の並行度を隔離する。
- `params` と `result` は保存時に暗号化しない。`JobKind` ごとに中身の見えないものであり、シークレットを含めてはならない。終端の行は再帰的に投入される Job ではなく、`worker` の保持期間スイープが削除する。

## Scenarios

### REQ-JOBS-001: Docker なしの標準開発環境で `worker` ジョブを完了する
- ACTOR Developer
- GIVEN 組込み PostgreSQL バイナリが取得済み、または初回取得可能である
- GIVEN 開発用ポートが利用可能である
- WHEN 開発者が標準開発コマンドを実行する
- THEN 組込み PostgreSQL が起動しスキーマが適用される
  - ALT バイナリの取得、ポートの確保、またはスキーマの適用に失敗する → 標準開発環境は API と UI を起動せずフェイルファストする
- THEN API、`worker`、UI が起動する
- THEN API が Job をキューへ投入する
- THEN `worker` が同じ Job を取得して Succeeded にする
- THEN API と `worker` は同じ PostgreSQL キューを共有する

### REQ-JOBS-002: ジョブを投入すると `worker` が実行して成功する
- ACTOR System
- GIVEN テナント "tenant-a" が存在する
- GIVEN `worker` プロセスが起動している
- WHEN テナント "tenant-a" に `kind="noop_echo"` の Job を投入する
- THEN Job の状態が queued である
- THEN `worker` が Job を取得し状態が `running` になる
- THEN ハンドラーが正常終了し状態が `succeeded` になり `result` を保持する

### REQ-JOBS-003: 同じ Job を再試行してもハンドラーの副作用は 1 回分だけ観測される
- ACTOR System
- GIVEN `dedup_key` を指定して Job が投入済みである
- WHEN ハンドラが1回目の実行で外部への通知を送信する
- THEN Job は succeeded になる
- WHEN at-least-once 配信により同じ Job が再配送される
- THEN ハンドラは dedup_key を用いて冪等に判定し重複通知を送らない

### REQ-JOBS-004: `worker` が異常終了してもリース失効後に別の `worker` が再取得する
- ACTOR System
- GIVEN テナント "tenant-a" の Job が `worker-1` に取得され `running` である
- WHEN `worker-1` がハートビートを送らないまま停止する
- THEN `lease_expires_at` を過ぎる
- THEN `worker-2` が同じ Job を取得し `running` を継続する

### REQ-JOBS-005: ハンドラーが失敗し続けると `max_attempts` 到達時に配信不能となる
- ACTOR System
- GIVEN `max_attempts=3` の Job が `running` で `FailJob` を呼ばれた回数がすでに 2 回である
- WHEN ハンドラが3回目も失敗し FailJob を呼ぶ
- THEN `attempts` が `max_attempts` に達している
- THEN Job の状態が `failed` になりエラーが保持される
- THEN Job は二度と `running` にならない

### REQ-JOBS-006: 他テナントの Job は `worker` のテナント境界を越えない
- ACTOR System
- GIVEN テナント "tenant-a" の Job "job-a" が `running` で `worker-1` に取得されている
- WHEN `worker-1` が "job-a" のハンドラーを実行する
- THEN ハンドラー実行コンテキストの `tenant_id` が "tenant-a" と一致する
  - ALT ハンドラーが誤って他テナントの Aggregate ID を渡された → `handler_execution_context.tenant_id` と対象 Aggregate の `tenant_id` が不一致のため操作が拒否される

### REQ-JOBS-007: 同じ dedup_key の lifecycle_workflow_run は重複して投入されない
- ACTOR System
- GIVEN テナント "tenant-a" で IdGovernance が `dedup_key="lifecycle-workflow-run:run-1"`、`kind="lifecycle_workflow_run"` の Job を `EnqueueJob` で投入済みである

- WHEN IdGovernance が同じ `dedup_key` で再度 `EnqueueJob` を呼ぶ（少なくとも 1 回の配信による再送やディスパッチャーの重複実行を模す）
- THEN 新規 Job は作成されず既存 Job の JobRef が返る

### REQ-JOBS-008: API プロセスでの投入に失敗しても定期ディスパッチャーが未関連付けの実行を回収する
- ACTOR System
- GIVEN IdGovernance が User のライフサイクルイベントの購読から WorkflowRun（`job_id` 未設定、`status=queued`）を確定したが、API プロセスからの即時 `EnqueueJob` 呼び出しには失敗した
- WHEN `worker` プロセスの定期ディスパッチャーが `job_id` の関連付いていない `queued` の実行を再走査する
- THEN ディスパッチャーが `dedup_key=lifecycle-workflow-run:{run_id}` で `EnqueueJob` を呼び、`job_id` を関連付ける
- THEN `worker` が Job を取得してハンドラーを実行する

### REQ-JOBS-009: bulk レーンに未処理ジョブが滞留しても latency_sensitive ジョブは専用実行枠で取得される
- ACTOR System
- GIVEN `bulk` レーンに `worker` の並行数を超える件数の長時間 Job が `queued` で滞留している
- GIVEN `latency_sensitive` レーンから取得する `worker` が別途稼働している
- WHEN テナント "tenant-a" に `kind="backchannel_logout_delivery"`（`lane=latency_sensitive`）の Job を投入する
- THEN `latency_sensitive` レーンの `worker` が `bulk` レーンの滞留量にかかわらず即座に取得する
- THEN Job が `running` に遷移し、`bulk` レーンの滞留によって実行を妨げられない

### REQ-JOBS-010: レーン未登録の JobKind は `worker` 起動時に拒否される
- ACTOR Developer
- GIVEN レーンが登録されていない `JobKind` がハンドラー一覧に登録されている
- WHEN `worker` を起動する
  - ALT 同一 JobKind に複数の異なるレーンが重複登録されようとした → `worker` の起動処理が重複登録を検出して起動を失敗させる
- THEN 起動処理がレーン未登録を検出して起動を失敗させる

### REQ-JOBS-011: レーン列を省略した行は `default` レーンで補完され取得対象になる
- ACTOR System
- GIVEN `lane` 列を省略して作成された `queued` Job "job-default" が存在する
- WHEN スキーマの `DEFAULT 'default'` により "job-default" の `lane` が補完される
- THEN `default` レーンから取得する `worker` が "job-default" を取得できる
