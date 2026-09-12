# Feature: Jobs のシナリオ

## Rule: REQ-JOBS-001 Docker なしの標準開発環境で `worker` ジョブを完了する

Primary actor: `Developer`

### Example: EX-JOBS-001-01 通常経路

- Given 組込み PostgreSQL バイナリが取得済み、または初回取得可能である
- And 開発用ポートが利用可能である
- When 開発者が標準開発コマンドを実行する
- Then 組込み PostgreSQL が起動しスキーマが適用される
- Then API、`worker`、UI が起動する
- Then API が Job をキューへ投入する
- Then `worker` が同じ Job を取得して Succeeded にする
- Then API と `worker` は同じ PostgreSQL キューを共有する

### Example: EX-JOBS-001-02 バイナリの取得、ポートの確保、またはスキーマの適用に失敗する

- Given 組込み PostgreSQL バイナリが取得済み、または初回取得可能である
- And 開発用ポートが利用可能である
- When 開発者が標準開発コマンドを実行する
- Then バイナリの取得、ポートの確保、またはスキーマの適用に失敗する
- Then 標準開発環境は API と UI を起動せずフェイルファストする

## Rule: REQ-JOBS-002 ジョブを投入すると `worker` が実行して成功する

Primary actor: `System`

### Example: EX-JOBS-002-01 通常経路

- Given テナント "tenant-a" が存在する
- And `worker` プロセスが起動している
- When テナント "tenant-a" に `kind="noop_echo"` の Job を投入する
- Then Job の状態が queued である
- Then `worker` が Job を取得し状態が `running` になる
- Then ハンドラーが正常終了し状態が `succeeded` になり `result` を保持する

## Rule: REQ-JOBS-003 同じ Job を再試行してもハンドラーの副作用は 1 回分だけ観測される

Primary actor: `System`

### Example: EX-JOBS-003-01 通常経路

- Given `dedup_key` を指定して Job が投入済みである
- When ハンドラーが 1 回目の実行で外部への通知を送信する
- Then Job は `succeeded` になる
- When 少なくとも 1 回の配送保証により同じ Job が再配送される
- Then ハンドラーは `dedup_key` を用いて冪等に判定し、重複した通知を送らない

## Rule: REQ-JOBS-004 `worker` が異常終了してもリース失効後に別の `worker` が再取得する

Primary actor: `System`

### Example: EX-JOBS-004-01 通常経路

- Given テナント "tenant-a" の Job が `worker-1` に取得され `running` である
- When `worker-1` がハートビートを送らないまま停止する
- Then `lease_expires_at` を過ぎる
- Then `worker-2` が同じ Job を取得し `running` を継続する

## Rule: REQ-JOBS-005 ハンドラーが失敗し続けると `max_attempts` 到達時に配信不能となる

Primary actor: `System`

### Example: EX-JOBS-005-01 通常経路

- Given `max_attempts=3` の Job が `running` で、`FailJob` を呼ばれた回数がすでに 2 回である
- When ハンドラーが 3 回目も失敗し `FailJob` を呼ぶ
- Then `attempts` が `max_attempts` に達している
- Then Job の状態が `failed` になりエラーが保持される
- Then Job は二度と `running` にならない

## Rule: REQ-JOBS-006 他テナントの Job は `worker` のテナント境界を越えない

Primary actor: `System`

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

Primary actor: `System`

### Example: EX-JOBS-007-01 通常経路

- Given テナント "tenant-a" で IdGovernance が `dedup_key="lifecycle-workflow-run:run-1"`、`kind="lifecycle_workflow_run"` の Job を `EnqueueJob` で投入済みである
- When IdGovernance が同じ `dedup_key` で再度 `EnqueueJob` を呼ぶ（再送やディスパッチャーの重複実行を模す）
- Then 新規 Job は作成されず既存 Job の JobRef が返る

## Rule: REQ-JOBS-008 API プロセスでの投入に失敗しても定期ディスパッチャーが未関連付けの実行を回収する

Primary actor: `System`

### Example: EX-JOBS-008-01 通常経路

- Given IdGovernance が User のライフサイクルイベントの購読から WorkflowRun（`job_id` 未設定、`status=queued`）を確定したが、API プロセスからの即時 `EnqueueJob` 呼び出しには失敗した
- When `worker` プロセスの定期ディスパッチャーが `job_id` の関連付いていない `queued` の実行を再走査する
- Then ディスパッチャーが `dedup_key=lifecycle-workflow-run:{run_id}` で `EnqueueJob` を呼び、`job_id` を関連付ける
- Then `worker` が Job を取得してハンドラーを実行する

## Rule: REQ-JOBS-009 bulk レーンに未処理ジョブが滞留しても latency_sensitive ジョブは専用実行枠で取得される

Primary actor: `System`

### Example: EX-JOBS-009-01 通常経路

- Given `bulk` レーンに `worker` の並行数を超える件数の長時間 Job が `queued` で滞留している
- And `latency_sensitive` レーンから取得する `worker` が別途稼働している
- When テナント "tenant-a" に `kind="backchannel_logout_delivery"`（`lane=latency_sensitive`）の Job を投入する
- Then `latency_sensitive` レーンの `worker` が `bulk` レーンの滞留量にかかわらず即座に取得する
- Then Job が `running` に遷移し、`bulk` レーンの滞留によって実行を妨げられない

## Rule: REQ-JOBS-010 レーン未登録の JobKind は `worker` 起動時に拒否される

Primary actor: `Developer`

### Example: EX-JOBS-010-01 通常経路

- Given レーンが登録されていない `JobKind` がハンドラー一覧に登録されている
- When `worker` を起動する
- Then 起動処理がレーン未登録を検出して起動を失敗させる

### Example: EX-JOBS-010-02 同一 JobKind に複数の異なるレーンが重複登録されようとした

- Given レーンが登録されていない `JobKind` がハンドラー一覧に登録されている
- When `worker` を起動する
- But 同一 JobKind に複数の異なるレーンが重複登録されようとした
- Then `worker` の起動処理が重複登録を検出して起動を失敗させる

## Rule: REQ-JOBS-011 レーン列を省略した行は `default` レーンで補完され取得対象になる

Primary actor: `System`

### Example: EX-JOBS-011-01 通常経路

- Given `lane` 列を省略して作成された `queued` Job "job-default" が存在する
- When スキーマの `DEFAULT 'default'` により "job-default" の `lane` が補完される
- Then `default` レーンから取得する `worker` が "job-default" を取得できる

## Rule: REQ-JOBS-012 管理者は自テナントのジョブだけを一覧・参照できる

Primary actor: `TenantAdministrator`

### Example: EX-JOBS-012-01 通常経路

- Given テナント "tenant-a" と "tenant-b" にそれぞれ Job が存在する
- When "tenant-a" の管理者がジョブ一覧を要求する
- Then "tenant-a" の Job だけが新しい順に返り、"tenant-b" の Job は件数にも含まれない
- Then レスポンスは `params`、`result`、`dedup_key` を含まない
- When 管理者が "tenant-b" の Job の id を指定して 1 件を要求する
- Then 存在しないものとして扱われる

### Example: EX-JOBS-012-02 実行者が要求先テナントの admin ロールを持たない

- Given テナント "tenant-a" と "tenant-b" にそれぞれ Job が存在する
- When "tenant-a" の管理者がジョブ一覧を要求する
- But 実行者が要求先テナントの admin ロールを持たない
- Then AccessDeniedError で拒否され、応答はどのテナントの Job も含まない

### Example: EX-JOBS-012-03 状態、種別、レーンで絞り込む

- Given テナント "tenant-a" と "tenant-b" にそれぞれ Job が存在する
- When "tenant-a" の管理者がジョブ一覧を要求する
- But 状態、種別、レーンで絞り込む
- Then 絞り込みに一致する自テナントの Job だけが返る

### Example: EX-JOBS-012-04 実行者が制御面主体の資格を持ち、横断を求める入力を添える

- Given テナント "tenant-a" と "tenant-b" にそれぞれ Job が存在する
- When "tenant-a" の管理者がジョブ一覧を要求する
- But 実行者が制御面主体の資格を持ち、横断を求める入力を添える
- Then 横断は認められず要求先テナントに閉じ、"tenant-b" の Job は件数にも含まれない

## Rule: REQ-JOBS-013 管理者は終端に達していないジョブを取り消せる

Primary actor: `TenantAdministrator`

### Example: EX-JOBS-013-01 通常経路

- Given テナント "tenant-a" の Job "job-1" が `queued` である
- When 管理者が "job-1" の取り消しを要求する
- Then "job-1" の状態が `canceled` になり "JobCanceled" が発行される
- Then 取り消しは再試行を伴わず、"job-1" は二度と `running` にならない

### Example: EX-JOBS-013-02 "job-1" が `running` である

- Given テナント "tenant-a" の Job "job-1" が `queued` である
- When 管理者が "job-1" の取り消しを要求する
- But "job-1" が `running` である
- Then 取り消しは受理され、リースを失ったハンドラーは次の報告で中断する

### Example: EX-JOBS-013-03 "job-1" が既に `succeeded` / `failed` / `canceled` である

- Given テナント "tenant-a" の Job "job-1" が `queued` である
- When 管理者が "job-1" の取り消しを要求する
- But "job-1" が既に `succeeded` / `failed` / `canceled` である
- Then JobNotCancelableError で拒否され、状態は変わらない

### Example: EX-JOBS-013-04 "job-1" が他テナントの Job である

- Given テナント "tenant-a" の Job "job-1" が `queued` である
- When 管理者が "job-1" の取り消しを要求する
- But "job-1" が他テナントの Job である
- Then 存在しないものとして扱われる

### Example: EX-JOBS-013-05 実行者が制御面主体の資格を持ち、他テナントの "job-1" を指定する

- Given テナント "tenant-a" の Job "job-1" が `queued` である
- When 管理者が "job-1" の取り消しを要求する
- But 実行者が制御面主体の資格を持ち、他テナントの "job-1" を指定する
- Then 存在しないものとして扱われ、"job-1" は `queued` のまま残る

## Rule: REQ-JOBS-014 管理 API はハンドラーの入出力を返さない

Primary actor: `TenantAdministrator`

### Example: EX-JOBS-014-01 通常経路

- Given 個人情報を含みうる `params` と `result` を持つ Job が存在する
- When 管理者がその Job の詳細を要求する
- Then 進捗、試行回数、上限、リースの保有者と期限、失敗理由、レーン、状態、時刻が返る
- Then `params` と `result` は返らず、管理 API から読み出す経路は存在しない

## Rule: REQ-JOBS-015 制御面主体はシステム経路で全テナントのジョブを一覧・参照・取り消せる

Primary actor: `SystemAdministrator`

### Example: EX-JOBS-015-01 通常経路

- Given テナント "tenant-a" と "tenant-b" にそれぞれ Job が存在する
- And 制御面テナントに所属する `system_admin` の操作者が制御面テナントの経路を使う
- When 操作者がシステム経路のジョブ一覧を要求する
- Then すべてのテナントの Job が新しい順に返り、`admin` ロールを併せ持つかどうかは結果を変えない
- Then レスポンスは `params`、`result`、`dedup_key` を含まない
- When 操作者が "tenant-b" の `queued` の Job の id を指定して 1 件を要求する
- Then その Job が返る
- When 操作者が同じ Job の取り消しを要求する
- Then 状態が `canceled` になり "JobCanceled" が発行される

### Example: EX-JOBS-015-02 実行者が `system_admin` を持たない、制御面テナントの所属ではない、または制御面テナント以外の経路である

- Given テナント "tenant-a" と "tenant-b" にそれぞれ Job が存在する
- And 制御面テナントに所属する `system_admin` の操作者が制御面テナントの経路を使う
- When 操作者がシステム経路のジョブ一覧を要求する
- But 実行者が `system_admin` を持たない、制御面テナントの所属ではない、または制御面テナント以外の経路である
- Then AccessDeniedError で拒否され、応答はどのテナントの Job も含まない

### Example: EX-JOBS-015-03 取り消しがブラウザーからの要求であることを証明できない

- Given テナント "tenant-a" と "tenant-b" にそれぞれ Job が存在する
- And 制御面テナントに所属する `system_admin` の操作者が制御面テナントの経路を使う
- When 操作者がシステム経路で "tenant-b" の Job の取り消しを要求する
- But 取り消しがブラウザーからの要求であることを証明できない
- Then CsrfFailedError または InvalidOriginError で拒否され、Job の状態は変わらない

### Example: EX-JOBS-015-04 指定した Job が既に終端に達している

- Given テナント "tenant-a" と "tenant-b" にそれぞれ Job が存在する
- And 制御面テナントに所属する `system_admin` の操作者が制御面テナントの経路を使う
- When 操作者がシステム経路で "tenant-b" の Job の取り消しを要求する
- But 指定した Job が既に終端に達している
- Then JobNotCancelableError で拒否され、状態は変わらない
