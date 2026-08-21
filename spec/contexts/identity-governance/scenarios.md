# IdGovernance Scenarios

### REQ-IDGOVERNANCE-001: 管理者はライフサイクルワークフローを作成できる
- ACTOR TenantAdministrator
- GIVEN 管理者が認証済みである
- WHEN 管理者が一覧画面の新規作成操作から専用の作成画面へ移動する
- THEN トリガー種別とアクション種別が日本語の説明付きで表示される
- WHEN 管理者がトリガー種別とアクション種別を選択する
  - ALT グループまたはアプリケーションを使うアクションに選択可能な参照先がない → 作成は拒否され、先に参照先を作成する必要があることが日本語で表示される
- WHEN 管理者がアクションごとの必須設定を入力し、複数アクションの実行順を並べ替える
  - ALT トリガーまたはアクションの必須設定が不足している → 作成操作は無効になり、不足している設定が日本語で表示される
- WHEN 管理者がワークフローを作成する
- THEN 選択したトリガーと順序付きアクションがリビジョン 1 の `draft` として返される
- THEN UI はトリガー、アクション、`draft`、`revision`、`current_revision` などの内部識別子を露出せず、日本語の表示名と意味を表示する

### REQ-IDGOVERNANCE-002: 管理者は既存ライフサイクルワークフローの定義を編集できる
- ACTOR TenantAdministrator
- GIVEN 管理者が既存ワークフローを選択している
- WHEN 管理者が一覧画面の編集操作から専用の編集画面へ移動する
- THEN 現在のトリガーと順序付きアクションが日本語の表示名と説明付きでフォームに復元される
- WHEN 管理者がトリガーまたはアクションを変更して保存する
- THEN `current_revision` が増え、変更した定義が一覧と編集フォームに反映される

### REQ-IDGOVERNANCE-003: 管理者が定義したワークフローが部署変更に応じてグループとアプリケーションの割り当てを更新する
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] の管理者 "operator" が LifecycleWorkflow "engineering-onboarding"（`trigger=user_attributes_changed department`、`action=add_group_member+assign_application`）を有効化済みである
- GIVEN ユーザー "alice" の `department` は "Sales" である
- WHEN 管理者 "operator" がユーザー "alice" の `department` を "Engineering" に更新する
- THEN WorkflowRun が作成され、トリガーのスナップショットに `changed_fields=["department"]` を保持する
- THEN `worker` が WorkflowRun を実行し、`add_group_member` と `assign_application` のステップが `changed` になる
- THEN WorkflowRun のステータスは `succeeded` である
- THEN "LifecycleWorkflowRunSucceeded" が発行される

### REQ-IDGOVERNANCE-004: User の変更と WorkflowRun の捕捉は同じ整合性境界で確定し、キュー投入の障害から回復する
- ACTOR TenantAdministrator
- GIVEN `user_created` / `user_attributes_changed` / `user_status_changed` のいずれかに一致する有効なワークフローがある
- WHEN ワークフローに一致する User の変更が実行される
- THEN User の変更と、重複排除キー (`tenant_id`, `workflow_id`, `revision`, `source_occurrence_id`, `target_user_id`) を持つ `queued` の WorkflowRun および `pending` の WorkflowStep が同一トランザクションで確定する
- THEN 同じ発火事象の再配信は既存の WorkflowRun に収束し、WorkflowStep を重複して作成しない
- WHEN `lifecycle_workflow_run` Job をキューへ投入する
  - ALT キューへの投入が一時的に失敗する → WorkflowRun は `job_id` が未設定の `queued` 状態で残り、`worker` の定期ディスパッチャーが再走査して重複しない Job を関連付ける
- THEN Job が WorkflowRun に関連付けられる

### REQ-IDGOVERNANCE-005: すでに目的の状態にある操作は `no_op` として扱われる
- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" は既に対象 Group のメンバーである
- GIVEN 有効化済みのワークフローは "alice" の `department` 変更をトリガーとし、`add_group_member` アクションを定義している
- WHEN ワークフローが "alice" の `add_group_member` ステップを実行する
- THEN ステップの結果は `no_op` である
- THEN メンバーシップの重複行は作成されない
- THEN WorkflowRun のステータスは `succeeded` である

### REQ-IDGOVERNANCE-006: 値が変わらない属性更新と動的グループのアクション指定は境界条件として扱われる
- ACTOR TenantAdministrator
- GIVEN "alice" の department は既に "Engineering" である
- WHEN 管理者が "alice" の department に同じ値 "Engineering" を指定して更新する
- THEN `user_attributes_changed` トリガーは発火せず WorkflowRun は作成されない
- WHEN 管理者がワークフローの `add_group_member` アクションに動的グループを指定して保存する
- THEN 保存は InvalidRequestError で拒否される

### REQ-IDGOVERNANCE-007: 未知のフィールドや別テナントのリソースを参照するワークフローは有効化できない
- ACTOR TenantAdministrator
- GIVEN `tenant_id` "acme" の管理者がワークフローを編集している
- WHEN TenantUserAttributeSchema に存在しないフィールドをフィルターに指定して保存する
- THEN エラー "InvalidRequestError"
- WHEN `tenant_id` "default" の `group_id` をアクションに指定して有効化する
- THEN エラー "InvalidRequestError"

### REQ-IDGOVERNANCE-008: 通知に失敗してもアクセス剥奪操作は前進する
- ACTOR TenantAdministrator
- GIVEN 有効化済みの退職者ワークフローが `disable_user`、`remove_group_member`、`send_email` を定義順に持つ
- GIVEN 対象 User に検証済みのプライマリメールアドレスがない
- WHEN 対象 User に退職相当のステータス変更が発生する
- THEN WorkflowRun が作成される
- THEN `disable_user` と `remove_group_member` のステップは `changed` になる
- THEN `send_email` のステップはブロックされた失敗になる
- THEN WorkflowRun のステータスは `partially_failed` である
- THEN "LifecycleWorkflowRunPartiallyFailed" と "LifecycleWorkflowStepFailed" が発行される

### REQ-IDGOVERNANCE-009: 一時的な失敗は再試行時に成功済みのステップを飛ばして収束する
- ACTOR TenantAdministrator
- GIVEN 有効化済みワークフローの WorkflowRun は、1 回目の試行で Repository がタイムアウトし、一部のステップが未完了である
- WHEN ハンドラーが Jobs に再試行可能なエラーを返す
- THEN Jobs がバックオフ後に同じ WorkflowRun の Job を再試行する
- THEN 再試行では `changed` または `no_op` のステップを再実行せず、`failed` のステップだけを再実行する
- WHEN 管理者が WorkflowRun の詳細を確認する
- THEN ステータスは `succeeded` である

### REQ-IDGOVERNANCE-010: 同一ユーザーの WorkflowRun は発火順に直列化され、副作用の前に再検証される
- ACTOR TenantAdministrator
- GIVEN 同じ対象ユーザーに対する `queued` の WorkflowRun が `triggered_at` 順に 2 件ある
- WHEN `worker` が同じ対象ユーザーの `queued` の WorkflowRun を取得する
- THEN 先行する WorkflowRun が終端状態になるまで、後続の WorkflowRun のアクションを開始しない
- THEN 各アクションの直前に、対象ユーザーとグループまたはアプリケーションのリソースを同一テナント内で再取得する
- THEN 削除済みまたは別テナントのリソースを参照するステップは、機密情報を除いたエラーコードと `failed` の結果をチェックポイントに記録する

### REQ-IDGOVERNANCE-011: 無効化すると未開始の WorkflowRun はキャンセルされ、実行中の WorkflowRun はステップ境界で止まる
- ACTOR TenantAdministrator
- GIVEN ワークフローが `enabled` で、`running` の WorkflowRun "run-1" と `queued` の WorkflowRun "run-2" が存在する
- WHEN 管理者がワークフローを無効化する
  - ALT 無効化と再試行が競合する → 無効化済みワークフローの WorkflowRun を再試行しても新しいステップを開始しない → エラー "InvalidRequestError"
- THEN "run-2" は直ちに `canceled` になる
- THEN "run-1" は現在のステップのチェックポイント後、次のステップの前に `canceled` になる
- THEN 無効化後に新しいトリガーは WorkflowRun を作らない

### REQ-IDGOVERNANCE-012: 別テナントのワークフローとリソースはテナント境界を越えない
- ACTOR TenantAdministrator
- GIVEN `tenant_id` "default" にワークフローが存在する
- WHEN `tenant_id` "acme" の管理者がその `workflow_id` を指定して取得する
- THEN ワークフローは存在しないものとして扱われる
- WHEN `tenant_id` "acme" のワークフローアクションが `tenant_id` "default" の `group_id` を参照する
- THEN 保存時または WorkflowRun の実行時に InvalidRequestError で拒否され、別テナントの同名リソースへフォールバックしない

### REQ-IDGOVERNANCE-013: プレビューではアクションの結果を試算するが WorkflowRun や Job を作成しない
- ACTOR TenantAdministrator
- GIVEN ワークフローが `enabled` で、対象 User は一部のアクションについてすでに目的の状態（例: 対象グループのメンバー）である
- WHEN 管理者が `dry_run` を呼び出す
  - ALT 有効化後の下書き編集が `current_revision` にあり、`enabled_revision` は編集前の内容を指す → プレビューでは `enabled_revision` のアクションとトリガーを評価し、下書きの変更を反映しない
- THEN 対象 User の現在のグループメンバーシップ、アプリケーションの割り当て、必須操作、ステータス、メールアドレスの検証状態が実際に読み取られる
  - ALT ワークフローのトリガーフィルターが対象 User の現在の属性に一致しない → レスポンスはすべてのアクションを `blocked`、理由を `trigger_not_matched` として返す
- THEN すでに目的の状態にあるアクションは `no_op`、状態が変わるアクションは `would_change`、リソースが存在しないなど実行不能なアクションは `blocked` と理由を返す
- THEN WorkflowRun、Job、メンバーシップ、割り当て、必須操作、ステータス、メールアドレスは一切作成・変更されない
