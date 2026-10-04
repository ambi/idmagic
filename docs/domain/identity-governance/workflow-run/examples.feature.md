# Feature: ワークフローの実行の例

## Rule: REQ-IDGOVERNANCE-003 管理者が定義したワークフローが部署変更に応じてグループとアプリケーションの割り当てを更新する

### Example: EX-IDGOVERNANCE-003-01 通常経路

- Given ロール=["admin"] の管理者 "operator" が LifecycleWorkflow "engineering-onboarding"（`trigger=user_attributes_changed department`、`action=add_group_member+assign_application`）を有効化済みである
- And ユーザー "alice" の `department` は "Sales" である
- When 管理者 "operator" がユーザー "alice" の `department` を "Engineering" に更新する
- Then WorkflowRun が作成され、トリガーのスナップショットに `changed_fields=["department"]` を保持する
- Then `worker` が WorkflowRun を実行し、`add_group_member` と `assign_application` のステップが `changed` になる
- Then WorkflowRun のステータスは `succeeded` である
- Then "LifecycleWorkflowRunSucceeded" が発行される

## Rule: REQ-IDGOVERNANCE-004 User の変更と WorkflowRun の捕捉は同じ整合性境界で確定し、キュー投入の障害から回復する

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

### Example: EX-IDGOVERNANCE-005-01 通常経路

- Given ユーザー "alice" は既に対象 Group のメンバーである
- And 有効化済みのワークフローは "alice" の `department` 変更をトリガーとし、`add_group_member` アクションを定義している
- When ワークフローが "alice" の `add_group_member` ステップを実行する
- Then ステップの結果は `no_op` である
- Then メンバーシップの重複レコードは作成されない
- Then WorkflowRun のステータスは `succeeded` である

## Rule: REQ-IDGOVERNANCE-006 値が変わらない属性更新と動的グループのアクション指定は境界条件として扱われる

### Example: EX-IDGOVERNANCE-006-01 通常経路

- Given "alice" の department は既に "Engineering" である
- When 管理者が "alice" の department に同じ値 "Engineering" を指定して更新する
- Then `user_attributes_changed` トリガーは発火せず WorkflowRun は作成されない
- When 管理者がワークフローの `add_group_member` アクションに動的グループを指定して保存する
- Then 保存は InvalidRequestError で拒否される

## Rule: REQ-IDGOVERNANCE-008 通知に失敗してもアクセス剥奪操作は前進する

### Example: EX-IDGOVERNANCE-008-01 通常経路

- Given 有効化済みの退職者ワークフローが `disable_user`、`remove_group_member`、`send_email` を定義順に持つ
- And 対象 User に検証済みのプライマリメールアドレスがない
- When 対象 User の属性が退職を表す値に変わる
- Then WorkflowRun が作成される
- Then `disable_user` と `remove_group_member` のステップは `changed` になる
- Then `send_email` のステップはブロックされた失敗になる
- Then WorkflowRun のステータスは `partially_failed` である
- Then "LifecycleWorkflowRunPartiallyFailed" と "LifecycleWorkflowStepFailed" が発行される

## Rule: REQ-IDGOVERNANCE-009 一時的な失敗は再試行時に成功済みのステップを飛ばして収束する

### Example: EX-IDGOVERNANCE-009-01 通常経路

- Given 有効化済みワークフローの WorkflowRun は、1 回目の試行で Repository がタイムアウトし、一部のステップが未完了である
- When ハンドラーが Jobs に再試行可能なエラーを返す
- Then Jobs がバックオフ後に同じ WorkflowRun の Job を再試行する
- Then 再試行では `changed` または `no_op` のステップを再実行せず、`failed` のステップだけを再実行する
- When 管理者が WorkflowRun の詳細を確認する
- Then ステータスは `succeeded` である

## Rule: REQ-IDGOVERNANCE-010 同一ユーザーの WorkflowRun は発火順に直列化され、副作用の前に再検証される

### Example: EX-IDGOVERNANCE-010-01 通常経路

- Given 同じ対象ユーザーに対する `queued` の WorkflowRun が `triggered_at` 順に 2 件ある
- When `worker` が同じ対象ユーザーの `queued` の WorkflowRun を取得する
- Then 先行する WorkflowRun が終端状態になるまで、後続の WorkflowRun のアクションを開始しない
- Then 各アクションの直前に、対象ユーザーとグループまたはアプリケーションのリソースを同一テナント内で再取得する
- Then 削除済みまたは別テナントのリソースを参照するステップは、機密情報を除いたエラーコードと `failed` の結果をチェックポイントに記録する
