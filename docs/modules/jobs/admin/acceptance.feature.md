# Feature: ジョブの管理の例

## Rule: REQ-JOBS-012 管理者は自テナントのジョブだけを一覧・参照できる

### Example: EX-JOBS-012-01 通常経路

- Given テナント "tenant-a" と "tenant-b" にそれぞれ Job が存在する
- When "tenant-a" の管理者がジョブ一覧を要求する
- Then "tenant-a" の Job だけが新しい順に返り、"tenant-b" の Job は件数にも含まれない
- Then レスポンスは `params`、`result`、`dedup_key` を含まない
- When 管理者が "tenant-b" の Job の id を指定して 1 件を要求する
- Then 存在しないものとして扱われる

### Scenario Outline: 条件ごとの結果

- Given テナント "tenant-a" と "tenant-b" にそれぞれ Job が存在する
- When "tenant-a" の管理者がジョブ一覧を要求する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-JOBS-012-02 | 実行者が要求先テナントの admin ロールを持たない | AccessDeniedError で拒否され、応答はどのテナントの Job も含まない |
  | EX-JOBS-012-03 | 状態、種別、レーンで絞り込む | 絞り込みに一致する自テナントの Job だけが返る |
  | EX-JOBS-012-04 | 実行者が制御面主体の資格を持ち、横断を求める入力を添える | 横断は認められず要求先テナントに閉じ、"tenant-b" の Job は件数にも含まれない |

## Rule: REQ-JOBS-013 管理者は終端に達していないジョブを取り消せる

### Background:

- Given テナント "tenant-a" の Job "job-1" が `queued` である

### Example: EX-JOBS-013-01 通常経路

- When 管理者が "job-1" の取り消しを要求する
- Then "job-1" の状態が `canceled` になり "JobCanceled" が発行される
- Then 取り消しは再試行を伴わず、"job-1" は二度と `running` にならない

### Scenario Outline: 条件ごとの結果

- When 管理者が "job-1" の取り消しを要求する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-JOBS-013-02 | "job-1" が `running` である | 取り消しは受理され、リースを失ったハンドラーは次の報告で中断する |
  | EX-JOBS-013-03 | "job-1" が既に `succeeded` / `failed` / `canceled` である | JobNotCancelableError で拒否され、状態は変わらない |
  | EX-JOBS-013-04 | "job-1" が他テナントの Job である | 存在しないものとして扱われる |
  | EX-JOBS-013-05 | 実行者が制御面主体の資格を持ち、他テナントの "job-1" を指定する | 存在しないものとして扱われ、"job-1" は `queued` のまま残る |

## Rule: REQ-JOBS-014 管理 API はハンドラーの入出力を返さない

### Example: EX-JOBS-014-01 通常経路

- Given 個人情報を含みうる `params` と `result` を持つ Job が存在する
- When 管理者がその Job の詳細を要求する
- Then 進捗、試行回数、上限、リースの保有者と期限、失敗理由、レーン、状態、時刻が返る
- Then `params` と `result` は返らず、管理 API から読み出す経路は存在しない

## Rule: REQ-JOBS-015 制御面主体はシステム経路で全テナントのジョブを一覧・参照・取り消せる

### Background:

- Given テナント "tenant-a" と "tenant-b" にそれぞれ Job が存在する
- And 制御面テナントに所属する `system_admin` の操作者が制御面テナントの経路を使う

### Example: EX-JOBS-015-01 通常経路

- When 操作者がシステム経路のジョブ一覧を要求する
- Then すべてのテナントの Job が新しい順に返り、`admin` ロールを併せ持つかどうかは結果を変えない
- Then レスポンスは `params`、`result`、`dedup_key` を含まない
- When 操作者が "tenant-b" の `queued` の Job の id を指定して 1 件を要求する
- Then その Job が返る
- When 操作者が同じ Job の取り消しを要求する
- Then 状態が `canceled` になり "JobCanceled" が発行される

### Scenario Outline: 条件ごとの結果

- When <condition>
- But <condition_2>
- Then <result>

#### Examples:

  | example_id | condition | condition_2 | result |
  | --- | --- | --- | --- |
  | EX-JOBS-015-02 | 操作者がシステム経路のジョブ一覧を要求する | 実行者が `system_admin` を持たない、制御面テナントの所属ではない、または制御面テナント以外の経路である | AccessDeniedError で拒否され、応答はどのテナントの Job も含まない |
  | EX-JOBS-015-03 | 操作者がシステム経路で "tenant-b" の Job の取り消しを要求する | 取り消しがブラウザーからの要求であることを証明できない | CsrfFailedError または InvalidOriginError で拒否され、Job の状態は変わらない |
  | EX-JOBS-015-04 | 操作者がシステム経路で "tenant-b" の Job の取り消しを要求する | 指定した Job が既に終端に達している | JobNotCancelableError で拒否され、状態は変わらない |
