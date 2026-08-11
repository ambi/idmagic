# Jobs Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-JOBS-001: Dockerなしの標準開発環境でworkerがジョブを完了する
- Actor: Developer
- Given: embedded PostgreSQL binary が取得済み、または初回取得可能である
- Given: 開発用ポートが利用可能である
- Then: 開発者が標準開発コマンドを実行する
- Then: embedded PostgreSQL が起動し schema が適用される
- Then: API、worker、UI が起動する
- Then: API が Job を enqueue する
- Then: worker が同じ Job を claim して Succeeded にする
- Then: API と worker は同じ PostgreSQL queue を共有する
- Alternative (binary 取得、ポート確保、または schema 適用に失敗する): 標準開発環境は API と UI を起動せず fail-fast する

### REQ-JOBS-002: ジョブをenqueueするとworkerが実行して成功する
- Actor: System
- Given: テナント "tenant-a" が存在する
- Given: worker プロセスが起動している
- Then: テナント "tenant-a" に kind="noop_echo" の Job を enqueue する
- Then: Job の状態が queued である
- Then: worker が Job を claim し状態が running になる
- Then: ハンドラが正常終了し状態が succeeded になり result を保持する

### REQ-JOBS-003: 同じJobがリトライされてもハンドラの副作用は1回分しか観測されない
- Actor: System
- Given: dedup_key を指定して Job が enqueue 済みである
- Then: ハンドラが1回目の実行で外部への通知を送信し succeeded になる
- Then: at-least-once 配信により同じ Job が誤って再実行されても、ハンドラは dedup_key を用いて冪等に判定し重複通知を送らない

### REQ-JOBS-004: workerがクラッシュしてもリース失効後に別workerが再取得する
- Actor: System
- Given: テナント "tenant-a" の Job が worker-1 に claim され running である
- Then: worker-1 が heartbeat を送らないまま停止する
- Then: lease_expires_at を過ぎる
- Then: worker-2 が同じ Job を claim し running を継続する

### REQ-JOBS-005: ハンドラが失敗し続けるとmax_attempts超過でdead-letterになる
- Actor: System
- Given: max_attempts=3 の Job が running で FailJob を呼ばれた回数が既に2回である
- Then: ハンドラが3回目も失敗し FailJob を呼ぶ
- Then: attempts が max_attempts に達している
- Then: Job の状態が failed になり error が保持される
- Then: Job は二度と running にならない

### REQ-JOBS-006: 他テナントのJobはworkerのテナント境界を跨がない
- Actor: System
- Given: テナント "tenant-a" の Job "job-a" が running で worker-1 に claim されている
- Then: worker-1 が "job-a" のハンドラを実行する
- Then: ハンドラ実行 context の tenant_id が "tenant-a" と一致する
- Alternative (ハンドラが誤って他テナントの集約 ID を渡された): handler_execution_context.tenant_id と対象集約の tenant_id が不一致のため操作が拒否される

### REQ-JOBS-007: 同じdedup_keyのlifecycle_workflow_runは重複enqueueされない
- Actor: System
- Given: テナント "tenant-a" で IdGovernance が dedup_key="lifecycle-workflow-run:run-1" の
        kind="lifecycle_workflow_run" Job を EnqueueJob 済みである

- Then: IdGovernance が同じ dedup_key で再度 EnqueueJob を呼ぶ (at-least-once 配信の再送や dispatcher の重複実行を模す)
- Then: 新規 Job は作成されず既存 Job の JobRef が返る

### REQ-JOBS-008: API processのenqueueが失敗してもperiodicなdispatcherが未関連付けのrunを回収する
- Actor: System
- Given: IdGovernance が User ライフサイクルイベントの購読から WorkflowRun (job_id 未設定、status=queued) を確定したが、API process の即時 EnqueueJob 呼び出しには失敗した
- Then: worker process の periodic dispatcher が job_id が未関連付けの queued run を再走査する
- Then: dispatcher が dedup_key=lifecycle-workflow-run:{run_id} で EnqueueJob を呼び job_id を関連付ける
- Then: worker が Job を claim してハンドラを実行する

### REQ-JOBS-009: bulk laneのbacklogが滞留してもlatency_sensitiveジョブは専用実行枠でclaimされる
- Actor: System
- Given: bulk lane に worker の concurrency を超える件数の長時間 Job が queued で滞留している
- Given: latency_sensitive lane を claim する worker が別途稼働している
- Then: テナント "tenant-a" に kind="backchannel_logout_delivery" (lane=latency_sensitive) の Job を enqueue する
- Then: latency_sensitive lane の worker が bulk lane の backlog に関わらず即座に claim する
- Then: Job が running に遷移し bulk lane の滞留から実行が妨げられない

### REQ-JOBS-010: lane未登録のJobKindはworker起動時に拒否される
- Actor: Developer
- Given: lane が登録されていない JobKind が handler registry に登録されようとしている
- Then: worker の起動処理が lane 未登録を検出し起動を失敗させる
- Alternative (同一 JobKind に複数の異なる lane が重複登録されようとした): worker の起動処理が重複登録を検出し起動を失敗させる

### REQ-JOBS-011: lane列未設定の既存行はdefault laneへbackfillされclaim対象になる
- Actor: System
- Given: lane 導入前に enqueue され lane が未設定 (schema 上の default) の queued Job "job-legacy" が存在する
- Then: schema の backfill により \"job-legacy\" の lane が default になる
- Then: default lane を claim する worker が \"job-legacy\" を claim できる

### REQ-JOBS-012: EnqueueJob
呼び出し元 bounded context が非同期処理を queue に投入する内部インターフェース。
HTTP には公開せず、同一プロセス内の Go 呼び出しとして各 context の usecase から使う。
dedup_key を指定すると (tenant_id, dedup_key) が既存の未終端 Job と一致する場合は
新規作成せず既存 JobRef を返す。作成される Job の lane は kind の登録情報から一意に
決まり、呼び出し元は指定できない。

### REQ-JOBS-013: ClaimJobs
worker が指定した lane 内で claim 可能な (status=queued かつ run_at <= now、
またはリース失効済みの running) Job をバッチで取得し、同一トランザクション内で Running に
遷移させ、自分を lease_owner として lease_expires_at を設定する内部インターフェース。
対象 lane 以外の Job は取得しない。
- Postcondition: all_jobs_leased_by(output.jobs, input.worker_id)
- Postcondition: single_worker_holds_active_lease(output.jobs)
- Postcondition: all_jobs_in_lane(output.jobs, input.lane)

### REQ-JOBS-014: HeartbeatJob
worker が実行中の Job のリースを延長する。

### REQ-JOBS-015: CompleteJob
worker がハンドラの正常終了を報告し、Job を Succeeded に確定する。

### REQ-JOBS-016: FailJob
worker がハンドラの失敗を報告する。attempts < max_attempts なら backoff 後の
run_at を設定して Queued (Retry) へ、attempts >= max_attempts なら Failed
(dead-letter) へ確定する。
- Postcondition: output.status == 'failed' || output.status == 'queued'

### REQ-JOBS-017: CancelJob
終端状態に達していない Job を取消す。既に終端状態の Job には作用しない (不可逆)。

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

## State machines

### JobLifecycle

Job の状態機械。queued → running は worker の claim、running → queued は
リトライ (backoff 待ち)、succeeded / failed / canceled は終端で不可逆。running → failed は
attempts が max_attempts に達した dead-letter 確定、running → queued (retry) は達していない場合。

Initial: `queued`  
Terminal: `succeeded`, `failed`, `canceled`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | JobStarted | "" | running |  |
| running | JobSucceeded | "" | succeeded |  |
| running | JobFailed | attempts >= max_attempts | failed |  |
| running | JobRetried | attempts < max_attempts | queued |  |
| queued | JobCanceled | "" | canceled |  |
| running | JobCanceled | "" | canceled |  |
