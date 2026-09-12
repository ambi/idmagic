# Feature: IdGovernance のシナリオ

## Rule: REQ-IDGOVERNANCE-001 管理者はライフサイクルワークフローを作成できる

Primary actor: `TenantAdministrator`

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

Primary actor: `TenantAdministrator`

### Example: EX-IDGOVERNANCE-002-01 通常経路

- Given 管理者が既存ワークフローを選択している
- When 管理者が一覧画面の編集操作から専用の編集画面へ移動する
- Then 現在のトリガーと順序付きアクションが日本語の表示名と説明付きでフォームに復元される
- When 管理者がトリガーまたはアクションを変更して保存する
- Then `current_revision` が増え、変更した定義が一覧と編集フォームに反映される

## Rule: REQ-IDGOVERNANCE-003 管理者が定義したワークフローが部署変更に応じてグループとアプリケーションの割り当てを更新する

Primary actor: `TenantAdministrator`

### Example: EX-IDGOVERNANCE-003-01 通常経路

- Given ロール=["admin"] の管理者 "operator" が LifecycleWorkflow "engineering-onboarding"（`trigger=user_attributes_changed department`、`action=add_group_member+assign_application`）を有効化済みである
- And ユーザー "alice" の `department` は "Sales" である
- When 管理者 "operator" がユーザー "alice" の `department` を "Engineering" に更新する
- Then WorkflowRun が作成され、トリガーのスナップショットに `changed_fields=["department"]` を保持する
- Then `worker` が WorkflowRun を実行し、`add_group_member` と `assign_application` のステップが `changed` になる
- Then WorkflowRun のステータスは `succeeded` である
- Then "LifecycleWorkflowRunSucceeded" が発行される

## Rule: REQ-IDGOVERNANCE-004 User の変更と WorkflowRun の捕捉は同じ整合性境界で確定し、キュー投入の障害から回復する

Primary actor: `TenantAdministrator`

### Example: EX-IDGOVERNANCE-004-01 通常経路

- Given `user_created` / `user_attributes_changed` / `user_status_changed` のいずれかに一致する有効なワークフローがある
- When ワークフローに一致する User の変更が実行される
- Then User の変更と、重複排除キー (`tenant_id`, `workflow_id`, `revision`, `source_occurrence_id`, `target_user_id`) を持つ `queued` の WorkflowRun および `pending` の WorkflowStep が同一トランザクションで確定する
- Then 同じ発火事象の再配信は既存の WorkflowRun に収束し、WorkflowStep を重複して作成しない
- When `lifecycle_workflow_run` Job をキューへ投入する
- Then Job が WorkflowRun に関連付けられる

### Example: EX-IDGOVERNANCE-004-02 キューへの投入が一時的に失敗する

- Given `user_created` / `user_attributes_changed` / `user_status_changed` のいずれかに一致する有効なワークフローがある
- When ワークフローに一致する User の変更が実行される
- Then User の変更と、重複排除キー (`tenant_id`, `workflow_id`, `revision`, `source_occurrence_id`, `target_user_id`) を持つ `queued` の WorkflowRun および `pending` の WorkflowStep が同一トランザクションで確定する
- Then 同じ発火事象の再配信は既存の WorkflowRun に収束し、WorkflowStep を重複して作成しない
- When `lifecycle_workflow_run` Job をキューへ投入する
- But キューへの投入が一時的に失敗する
- Then WorkflowRun は `job_id` が未設定の `queued` 状態で残り、`worker` の定期ディスパッチャーが再走査して重複しない Job を関連付ける

## Rule: REQ-IDGOVERNANCE-005 すでに目的の状態にある操作は `no_op` として扱われる

Primary actor: `TenantAdministrator`

### Example: EX-IDGOVERNANCE-005-01 通常経路

- Given ユーザー "alice" は既に対象 Group のメンバーである
- And 有効化済みのワークフローは "alice" の `department` 変更をトリガーとし、`add_group_member` アクションを定義している
- When ワークフローが "alice" の `add_group_member` ステップを実行する
- Then ステップの結果は `no_op` である
- Then メンバーシップの重複行は作成されない
- Then WorkflowRun のステータスは `succeeded` である

## Rule: REQ-IDGOVERNANCE-006 値が変わらない属性更新と動的グループのアクション指定は境界条件として扱われる

Primary actor: `TenantAdministrator`

### Example: EX-IDGOVERNANCE-006-01 通常経路

- Given "alice" の department は既に "Engineering" である
- When 管理者が "alice" の department に同じ値 "Engineering" を指定して更新する
- Then `user_attributes_changed` トリガーは発火せず WorkflowRun は作成されない
- When 管理者がワークフローの `add_group_member` アクションに動的グループを指定して保存する
- Then 保存は InvalidRequestError で拒否される

## Rule: REQ-IDGOVERNANCE-007 未知のフィールドや別テナントのリソースを参照するワークフローは有効化できない

Primary actor: `TenantAdministrator`

### Example: EX-IDGOVERNANCE-007-01 通常経路

- Given `tenant_id` "acme" の管理者がワークフローを編集している
- When TenantUserAttributeSchema に存在しないフィールドをフィルターに指定して保存する
- Then エラー "InvalidRequestError"
- When `tenant_id` "default" の `group_id` をアクションに指定して有効化する
- Then エラー "InvalidRequestError"

## Rule: REQ-IDGOVERNANCE-008 通知に失敗してもアクセス剥奪操作は前進する

Primary actor: `TenantAdministrator`

### Example: EX-IDGOVERNANCE-008-01 通常経路

- Given 有効化済みの退職者ワークフローが `disable_user`、`remove_group_member`、`send_email` を定義順に持つ
- And 対象 User に検証済みのプライマリメールアドレスがない
- When 対象 User に退職相当のステータス変更が発生する
- Then WorkflowRun が作成される
- Then `disable_user` と `remove_group_member` のステップは `changed` になる
- Then `send_email` のステップはブロックされた失敗になる
- Then WorkflowRun のステータスは `partially_failed` である
- Then "LifecycleWorkflowRunPartiallyFailed" と "LifecycleWorkflowStepFailed" が発行される

## Rule: REQ-IDGOVERNANCE-009 一時的な失敗は再試行時に成功済みのステップを飛ばして収束する

Primary actor: `TenantAdministrator`

### Example: EX-IDGOVERNANCE-009-01 通常経路

- Given 有効化済みワークフローの WorkflowRun は、1 回目の試行で Repository がタイムアウトし、一部のステップが未完了である
- When ハンドラーが Jobs に再試行可能なエラーを返す
- Then Jobs がバックオフ後に同じ WorkflowRun の Job を再試行する
- Then 再試行では `changed` または `no_op` のステップを再実行せず、`failed` のステップだけを再実行する
- When 管理者が WorkflowRun の詳細を確認する
- Then ステータスは `succeeded` である

## Rule: REQ-IDGOVERNANCE-010 同一ユーザーの WorkflowRun は発火順に直列化され、副作用の前に再検証される

Primary actor: `TenantAdministrator`

### Example: EX-IDGOVERNANCE-010-01 通常経路

- Given 同じ対象ユーザーに対する `queued` の WorkflowRun が `triggered_at` 順に 2 件ある
- When `worker` が同じ対象ユーザーの `queued` の WorkflowRun を取得する
- Then 先行する WorkflowRun が終端状態になるまで、後続の WorkflowRun のアクションを開始しない
- Then 各アクションの直前に、対象ユーザーとグループまたはアプリケーションのリソースを同一テナント内で再取得する
- Then 削除済みまたは別テナントのリソースを参照するステップは、機密情報を除いたエラーコードと `failed` の結果をチェックポイントに記録する

## Rule: REQ-IDGOVERNANCE-011 無効化すると未開始の WorkflowRun はキャンセルされ、実行中の WorkflowRun はステップ境界で止まる

Primary actor: `TenantAdministrator`

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

Primary actor: `TenantAdministrator`

### Example: EX-IDGOVERNANCE-012-01 通常経路

- Given `tenant_id` "default" にワークフローが存在する
- When `tenant_id` "acme" の管理者がその `workflow_id` を指定して取得する
- Then ワークフローは存在しないものとして扱われる
- When `tenant_id` "acme" のワークフローアクションが `tenant_id` "default" の `group_id` を参照する
- Then 保存時または WorkflowRun の実行時に InvalidRequestError で拒否され、別テナントの同名リソースへフォールバックしない

## Rule: REQ-IDGOVERNANCE-013 プレビューではアクションの結果を試算するが WorkflowRun や Job を作成しない

Primary actor: `TenantAdministrator`

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

Primary actor: `TenantAdministrator`

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
