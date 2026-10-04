# Feature: 永続キューの例

## Rule: REQ-JOBS-002 ジョブを投入すると `worker` が実行して成功する

### Example: EX-JOBS-002-01 通常経路

- Given テナント "tenant-a" が存在する
- And `worker` プロセスが起動している
- When テナント "tenant-a" に `kind="noop_echo"` の Job を投入する
- Then Job の状態が queued である
- Then `worker` が Job を取得し状態が `running` になる
- Then ハンドラーが正常終了し状態が `succeeded` になり `result` を保持する

## Rule: REQ-JOBS-003 同じ Job を再試行してもハンドラーの副作用は 1 回分だけ観測される

### Example: EX-JOBS-003-01 通常経路

- Given `dedup_key` を指定して Job が投入済みである
- When ハンドラーが 1 回目の実行で外部への通知を送信する
- Then Job は `succeeded` になる
- When 少なくとも 1 回の配送保証により同じ Job が再配送される
- Then ハンドラーは `dedup_key` を用いて冪等に判定し、重複した通知を送らない

## Rule: REQ-JOBS-004 `worker` が異常終了してもリース失効後に別の `worker` が再取得する

### Example: EX-JOBS-004-01 通常経路

- Given テナント "tenant-a" の Job が `worker-1` に取得され `running` である
- When `worker-1` がハートビートを送らないまま停止する
- Then `lease_expires_at` を過ぎる
- Then `worker-2` が同じ Job を取得し `running` を継続する

## Rule: REQ-JOBS-005 ハンドラーが失敗し続けると `max_attempts` 到達時に配信不能となる

### Example: EX-JOBS-005-01 通常経路

- Given `max_attempts=3` の Job が `running` で、`FailJob` を呼ばれた回数がすでに 2 回である
- When ハンドラーが 3 回目も失敗し `FailJob` を呼ぶ
- Then `attempts` が `max_attempts` に達している
- Then Job の状態が `failed` になりエラーが保持される
- Then Job は二度と `running` にならない

## Rule: REQ-JOBS-006 他テナントの Job は `worker` のテナント境界を越えない

### Example: EX-JOBS-006-01 通常経路

- Given テナント "tenant-a" の Job "job-a" が `running` で `worker-1` に取得されている
- When `worker-1` が "job-a" のハンドラーを実行する
- Then ハンドラー実行コンテキストの `tenant_id` が "tenant-a" と一致する

### Example: EX-JOBS-006-02 ハンドラーが誤って他テナントの Aggregate ID を渡された

- Given テナント "tenant-a" の Job "job-a" が `running` で `worker-1` に取得されている
- When `worker-1` が "job-a" のハンドラーを実行する
- Then ハンドラーが誤って他テナントの Aggregate ID を渡された
- Then `handler_execution_context.tenant_id` と対象 Aggregate の `tenant_id` が不一致のため操作が拒否される

## Rule: REQ-JOBS-007 同じ dedup_key の lifecycle_workflow_run は重複して投入されない

### Example: EX-JOBS-007-01 通常経路

- Given テナント "tenant-a" で IdGovernance が `dedup_key="lifecycle-workflow-run:run-1"`、`kind="lifecycle_workflow_run"` の Job を `EnqueueJob` で投入済みである
- When IdGovernance が同じ `dedup_key` で再度 `EnqueueJob` を呼ぶ（再送やディスパッチャーの重複実行を模す）
- Then 新規 Job は作成されず既存 Job の JobRef が返る

## Rule: REQ-JOBS-008 API プロセスでの投入に失敗しても定期ディスパッチャーが未関連付けの実行を回収する

### Example: EX-JOBS-008-01 通常経路

- Given IdGovernance が User のライフサイクルイベントの購読から WorkflowRun（`job_id` 未設定、`status=queued`）を確定したが、API プロセスからの即時 `EnqueueJob` 呼び出しには失敗した
- When `worker` プロセスの定期ディスパッチャーが `job_id` の関連付いていない `queued` の実行を再走査する
- Then ディスパッチャーが `dedup_key=lifecycle-workflow-run:{run_id}` で `EnqueueJob` を呼び、`job_id` を関連付ける
- Then `worker` が Job を取得してハンドラーを実行する

## Rule: REQ-JOBS-009 bulk レーンに未処理ジョブが滞留しても latency_sensitive ジョブは専用実行枠で取得される

### Example: EX-JOBS-009-01 通常経路

- Given `bulk` レーンに `worker` の並行数を超える件数の長時間 Job が `queued` で滞留している
- And `latency_sensitive` レーンから取得する `worker` が別途稼働している
- When テナント "tenant-a" に `kind="backchannel_logout_delivery"`（`lane=latency_sensitive`）の Job を投入する
- Then `latency_sensitive` レーンの `worker` が `bulk` レーンの滞留量にかかわらず即座に取得する
- Then Job が `running` に遷移し、`bulk` レーンの滞留によって実行を妨げられない

## Rule: REQ-JOBS-010 レーン未登録の JobKind は `worker` 起動時に拒否される

### Example: EX-JOBS-010-01 通常経路

- Given レーンが登録されていない `JobKind` がハンドラー一覧に登録されている
- When `worker` を起動する
- Then 起動処理がレーン未登録を検出して起動を失敗させる

### Example: EX-JOBS-010-02 同一 JobKind に複数の異なるレーンが重複登録されようとした

- Given レーンが登録されていない `JobKind` がハンドラー一覧に登録されている
- When `worker` を起動する
- But 同一 JobKind に複数の異なるレーンが重複登録されようとした
- Then `worker` の起動処理が重複登録を検出して起動を失敗させる

## Rule: REQ-JOBS-011 レーンのカラムを省略したレコードは `default` レーンで補完され取得対象になる

### Example: EX-JOBS-011-01 通常経路

- Given `lane` カラムを省略して作成された `queued` Job "job-default" が存在する
- When スキーマの `DEFAULT 'default'` により "job-default" の `lane` が補完される
- Then `default` レーンから取得する `worker` が "job-default" を取得できる
