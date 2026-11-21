# IdGovernance の用語集

| 用語 | 定義 | 別名 |
|---|---|---|
| アクセスの見直し | 付与済みの所属、割り当て、Agent の継続利用の必要性を、確認担当者が判断すること。 | access review |
| 見直し計画 | アクセスの見直しの対象、確認担当者、実施時期、未回答時の処置を定めたもの。 | access review plan |
| 見直し | 見直し計画に基づいて行う一回分の確認。 | access review instance |
| 確認担当者 | 見直しの対象項目について、維持または失効を判断する利用者。 | reviewer |
| LifecycleWorkflow | テナント管理者が事前に定義し、User の作成、属性変更、`status` の遷移をトリガーとして構造化されたアクションを実行する。`draft` / `enabled` / `disabled` / `archived` のライフサイクルを持ち、有効化したリビジョンだけが新しい WorkflowRun を生成する。DAG や条件分岐は持たず、アクションは定義順の線形リストとする。 | lifecycle workflow, ライフサイクルワークフロー |
| WorkflowTrigger | LifecycleWorkflow のリビジョンに固定する発火条件。`user_created` / `user_attributes_changed` / `user_status_changed` のいずれかの種別と、0〜20 件の型付きフィルター（`field` / `operator` / `value` の AND）からなる。User を変更した後の状態に対して評価し、属性変更の前後比較だけは変更差分を使う。 | workflow trigger, トリガー |
| WorkflowAction | WorkflowRun が展開する望ましい状態への操作 1 件。`add_group_member` / `remove_group_member` / `assign_application` / `unassign_application` / `set_required_action` / `clear_required_action` / `enable_user` / `disable_user` / `send_email` のいずれかで、すでに目的の状態なら `no_op` とする冪等な操作。 | workflow action, アクション |
| WorkflowRun | WorkflowTrigger の 1 回の発火から生成する実行単位。作成時の `workflow_id`、`revision`、発火事象、対象ユーザー、展開済みのアクションリストを固定するため、後から定義が変わっても意味は変わらない。Jobs の `lifecycle_workflow_run` Job に関連付けて実行する。 | workflow run, ワークフロー実行 |
| WorkflowStep | WorkflowRun 内のアクション 1 件分の実行記録。結果（`changed` / `no_op` / `failed` / `canceled`）、機密情報を除いたエラーコード、実行時刻をチェックポイントとして記録する。再試行では失敗したステップだけを再実行し、`changed` または `no_op` になったステップは飛ばす。 | workflow step, ステップ |
