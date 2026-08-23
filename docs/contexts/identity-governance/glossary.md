# IdGovernance Glossary

| Term | Definition | Aliases |
|---|---|---|
| LifecycleWorkflow | テナント管理者が事前に定義し、User の作成、属性変更、`status` の遷移をトリガーとして構造化されたアクションを実行する。`draft` / `enabled` / `disabled` / `archived` のライフサイクルを持ち、有効化したリビジョンだけが新しい WorkflowRun を生成する。DAG や条件分岐は持たず、アクションは定義順の線形リストとする。 | lifecycle workflow, ライフサイクルワークフロー |
| WorkflowTrigger | LifecycleWorkflow のリビジョンに固定する発火条件。`user_created` / `user_attributes_changed` / `user_status_changed` のいずれかの種別と、0〜20 件の型付きフィルター（`field` / `operator` / `value` の AND）からなる。User を変更した後の状態に対して評価し、属性変更の前後比較だけは変更差分を使う。 | workflow trigger, トリガー |
| WorkflowAction | WorkflowRun が展開する望ましい状態への操作 1 件。`add_group_member` / `remove_group_member` / `assign_application` / `unassign_application` / `set_required_action` / `clear_required_action` / `enable_user` / `disable_user` / `send_email` のいずれかで、すでに目的の状態なら `no_op` とする冪等な操作。 | workflow action, アクション |
| WorkflowRun | WorkflowTrigger の 1 回の発火から生成する実行単位。作成時の `workflow_id`、`revision`、発火事象、対象ユーザー、展開済みのアクションリストを固定するため、後から定義が変わっても意味は変わらない。Jobs の `lifecycle_workflow_run` Job に関連付けて実行する。 | workflow run, ワークフロー実行 |
| WorkflowStep | WorkflowRun 内のアクション 1 件分の実行記録。結果（`changed` / `no_op` / `failed` / `canceled`）、機密情報を除いたエラーコード、実行時刻をチェックポイントとして記録する。再試行では失敗したステップだけを再実行し、`changed` または `no_op` になったステップは飛ばす。 | workflow step, ステップ |
