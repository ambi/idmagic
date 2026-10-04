# Feature: ライフサイクルワークフローの定義の例

## Rule: REQ-IDGOVERNANCE-001 管理者はライフサイクルワークフローを作成できる

### Example: EX-IDGOVERNANCE-001-01 通常経路

- Given 管理者が認証済みである
- When 管理者が一覧画面の新規作成操作から専用の作成画面へ移動する
- Then トリガー種別とアクション種別が日本語の説明付きで表示される
- When 管理者がトリガー種別とアクション種別を選択する
- When 管理者がアクションごとの必須設定を入力し、複数アクションの実行順を並べ替える
- When 管理者がワークフローを作成する
- Then 選択したトリガーと順序付きアクションがリビジョン 1 の `draft` として返される
- Then UI はトリガー、アクション、`draft`、`revision`、`current_revision` などの内部識別子を露出せず、日本語の表示名と意味を表示する

### Example: EX-IDGOVERNANCE-001-02 グループまたはアプリケーションを使うアクションに選択可能な参照先がない

- Given 管理者が認証済みである
- When 管理者が一覧画面の新規作成操作から専用の作成画面へ移動する
- Then トリガー種別とアクション種別が日本語の説明付きで表示される
- When 管理者がトリガー種別とアクション種別を選択する
- But グループまたはアプリケーションを使うアクションに選択可能な参照先がない
- Then 作成は拒否され、先に参照先を作成する必要があることが日本語で表示される

### Example: EX-IDGOVERNANCE-001-03 トリガーまたはアクションの必須設定が不足している

- Given 管理者が認証済みである
- When 管理者が一覧画面の新規作成操作から専用の作成画面へ移動する
- Then トリガー種別とアクション種別が日本語の説明付きで表示される
- When 管理者がトリガー種別とアクション種別を選択する
- When 管理者がアクションごとの必須設定を入力し、複数アクションの実行順を並べ替える
- But トリガーまたはアクションの必須設定が不足している
- Then 作成操作は無効になり、不足している設定が日本語で表示される

## Rule: REQ-IDGOVERNANCE-002 管理者は既存ライフサイクルワークフローの定義を編集できる

### Example: EX-IDGOVERNANCE-002-01 通常経路

- Given 管理者が既存ワークフローを選択している
- When 管理者が一覧画面の編集操作から専用の編集画面へ移動する
- Then 現在のトリガーと順序付きアクションが日本語の表示名と説明付きでフォームに復元される
- When 管理者がトリガーまたはアクションを変更して保存する
- Then `current_revision` が増え、変更した定義が一覧と編集フォームに反映される

## Rule: REQ-IDGOVERNANCE-007 未知のフィールドや別テナントのリソースを参照するワークフローは有効化できない

### Example: EX-IDGOVERNANCE-007-01 通常経路

- Given `tenant_id` "acme" の管理者がワークフローを編集している
- When TenantUserAttributeSchema に存在しないフィールドをフィルターに指定して保存する
- Then エラー "InvalidRequestError"
- When `tenant_id` "default" の `group_id` をアクションに指定して有効化する
- Then エラー "InvalidRequestError"

## Rule: REQ-IDGOVERNANCE-011 無効化すると未開始の WorkflowRun はキャンセルされ、実行中の WorkflowRun はステップ境界で止まる

### Example: EX-IDGOVERNANCE-011-01 通常経路

- Given ワークフローが `enabled` で、`running` の WorkflowRun "run-1" と `queued` の WorkflowRun "run-2" が存在する
- When 管理者がワークフローを無効化する
- Then "run-2" は直ちに `canceled` になる
- Then "run-1" は現在のステップのチェックポイント後、次のステップの前に `canceled` になる
- Then 無効化後に新しいトリガーは WorkflowRun を作らない

### Example: EX-IDGOVERNANCE-011-02 無効化と再試行が競合する

- Given ワークフローが `enabled` で、`running` の WorkflowRun "run-1" と `queued` の WorkflowRun "run-2" が存在する
- When 管理者がワークフローを無効化する
- But 無効化と再試行が競合する
- Then 無効化済みワークフローの WorkflowRun を再試行しても新しいステップを開始しない
- And エラー "InvalidRequestError"

## Rule: REQ-IDGOVERNANCE-012 別テナントのワークフローとリソースはテナント境界を越えない

### Example: EX-IDGOVERNANCE-012-01 通常経路

- Given `tenant_id` "default" にワークフローが存在する
- When `tenant_id` "acme" の管理者がその `workflow_id` を指定して取得する
- Then ワークフローは存在しないものとして扱われる
- When `tenant_id` "acme" のワークフローアクションが `tenant_id` "default" の `group_id` を参照する
- Then 保存時または WorkflowRun の実行時に InvalidRequestError で拒否され、別テナントの同名リソースへフォールバックしない

## Rule: REQ-IDGOVERNANCE-013 プレビューではアクションの結果を試算するが WorkflowRun や Job を作成しない

### Example: EX-IDGOVERNANCE-013-01 通常経路

- Given ワークフローが `enabled` で、対象 User は一部のアクションについてすでに目的の状態（例: 対象グループのメンバー）である
- When 管理者が `dry_run` を呼び出す
- Then 対象 User の現在のグループメンバーシップ、アプリケーションの割り当て、必須操作、ステータス、メールアドレスの検証状態が実際に読み取られる
- Then すでに目的の状態にあるアクションは `no_op`、状態が変わるアクションは `would_change`、リソースが存在しないなど実行不能なアクションは `blocked` と理由を返す
- Then WorkflowRun、Job、メンバーシップ、割り当て、必須操作、ステータス、メールアドレスは一切作成・変更されない

### Example: EX-IDGOVERNANCE-013-02 有効化後の下書き編集が `current_revision` にあり、`enabled_revision` は編集前の内容を指す

- Given ワークフローが `enabled` で、対象 User は一部のアクションについてすでに目的の状態（例: 対象グループのメンバー）である
- When 管理者が `dry_run` を呼び出す
- But 有効化後の下書き編集が `current_revision` にあり、`enabled_revision` は編集前の内容を指す
- Then プレビューでは `enabled_revision` のアクションとトリガーを評価し、下書きの変更を反映しない

### Example: EX-IDGOVERNANCE-013-03 ワークフローのトリガーフィルターが対象 User の現在の属性に一致しない

- Given ワークフローが `enabled` で、対象 User は一部のアクションについてすでに目的の状態（例: 対象グループのメンバー）である
- When 管理者が `dry_run` を呼び出す
- Then ワークフローのトリガーフィルターが対象 User の現在の属性に一致しない
- Then レスポンスはすべてのアクションを `blocked`、理由を `trigger_not_matched` として返す

## Rule: REQ-IDGOVERNANCE-014 ライフサイクルワークフローの管理は管理者に限られる

### Example: EX-IDGOVERNANCE-014-01 通常経路

- Given "alice" は認証済みだが `admin` ロールを持たない
- When "alice" がワークフローの作成、更新、有効化、無効化、削除、プレビュー、または WorkflowRun の再試行を要求する
- Then AccessDeniedError で拒否される
- Then ワークフローは作成も変更もされず、WorkflowRun と Job は生成されない

### Example: EX-IDGOVERNANCE-014-02 "alice" がワークフローの一覧または詳細を要求する

- Given "alice" は認証済みだが `admin` ロールを持たない
- When "alice" がワークフローの作成、更新、有効化、無効化、削除、プレビュー、または WorkflowRun の再試行を要求する
- But "alice" がワークフローの一覧または詳細を要求する
- Then AccessDeniedError で拒否される
