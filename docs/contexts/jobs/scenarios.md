# Jobs Scenarios

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
- WHEN ハンドラーが 1 回目の実行で外部への通知を送信する
- THEN Job は `succeeded` になる
- WHEN 少なくとも 1 回の配送保証により同じ Job が再配送される
- THEN ハンドラーは `dedup_key` を用いて冪等に判定し、重複した通知を送らない

### REQ-JOBS-004: `worker` が異常終了してもリース失効後に別の `worker` が再取得する
- ACTOR System
- GIVEN テナント "tenant-a" の Job が `worker-1` に取得され `running` である
- WHEN `worker-1` がハートビートを送らないまま停止する
- THEN `lease_expires_at` を過ぎる
- THEN `worker-2` が同じ Job を取得し `running` を継続する

### REQ-JOBS-005: ハンドラーが失敗し続けると `max_attempts` 到達時に配信不能となる
- ACTOR System
- GIVEN `max_attempts=3` の Job が `running` で、`FailJob` を呼ばれた回数がすでに 2 回である
- WHEN ハンドラーが 3 回目も失敗し `FailJob` を呼ぶ
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
- WHEN IdGovernance が同じ `dedup_key` で再度 `EnqueueJob` を呼ぶ（再送やディスパッチャーの重複実行を模す）
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

### REQ-JOBS-012: 管理者は自テナントのジョブだけを一覧・参照できる
- ACTOR TenantAdministrator
- GIVEN テナント "tenant-a" と "tenant-b" にそれぞれ Job が存在する
- WHEN "tenant-a" の管理者がジョブ一覧を要求する
  - ALT 実行者が admin ロールも制御面主体の資格も持たない → AccessDeniedError で拒否される
  - ALT 状態、種別、レーンで絞り込む → 絞り込みに一致する自テナントの Job だけが返る
- THEN "tenant-a" の Job だけが新しい順に返り、"tenant-b" の Job は件数にも含まれない
- THEN 応答は `params`、`result`、`dedup_key` を含まない
- WHEN 管理者が "tenant-b" の Job の id を指定して 1 件を要求する
- THEN 存在しないものとして扱われる
- WHEN 制御面テナントに所属する `system_admin` が制御面テナントの経路で全テナント横断を明示して一覧を要求する
  - ALT `system_admin` を持たない、制御面テナントの所属ではない、または制御面テナント以外の経路である → 横断は認められず自テナントに閉じる
- THEN すべてのテナントの Job が返り、`admin` ロールを併せ持つかどうかは結果を変えない
- WHEN 同じ実行者が他テナントの Job の id を指定して 1 件を要求する
- THEN その Job が返る

### REQ-JOBS-013: 管理者は終端に達していないジョブを取り消せる
- ACTOR TenantAdministrator
- GIVEN テナント "tenant-a" の Job "job-1" が `queued` である
- WHEN 管理者が "job-1" の取り消しを要求する
  - ALT "job-1" が `running` である → 取り消しは受理され、リースを失ったハンドラーは次の報告で中断する
  - ALT "job-1" が既に `succeeded` / `failed` / `canceled` である → JobNotCancelableError で拒否され、状態は変わらない
  - ALT "job-1" が他テナントの Job である → 存在しないものとして扱われる
  - ALT 制御面テナントに所属する `system_admin` が制御面テナントの経路で他テナントの "job-1" を取り消す → `admin` ロールを併せ持たなくても受理される
- THEN "job-1" の状態が `canceled` になり "JobCanceled" が発行される
- THEN 取り消しは再試行を伴わず、"job-1" は二度と `running` にならない

### REQ-JOBS-014: 管理 API はハンドラーの入出力を返さない
- ACTOR TenantAdministrator
- GIVEN 個人情報を含みうる `params` と `result` を持つ Job が存在する
- WHEN 管理者がその Job の詳細を要求する
- THEN 進捗、試行回数、上限、リースの保有者と期限、失敗理由、レーン、状態、時刻が返る
- THEN `params` と `result` は返らず、管理 API から読み出す経路は存在しない
