# IdGovernance のデータ

この文書は、IdGovernance の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `LifecycleWorkflow` | 名前、状態、`current_revision` と `enabled_revision`、リビジョンごとの `WorkflowTrigger` と `WorkflowAction` の列 | `Tenant` を `tenant_id` で参照する。アクションは同じテナントの Group と Application を参照する |
| `WorkflowRun` | 作成時に固定した `workflow_id`、`revision`、発火の事象、対象の User、展開したアクションの列、`WorkflowStep` の記録、関連付けた Job | `LifecycleWorkflow` と User を参照する |

| 概念 | 内容 |
| --- | --- |
| `WorkflowTrigger` | `user_created`、`user_attributes_changed`、`user_status_changed` のいずれかと、0〜20 件の型付きのフィルター（`field`、`operator`、`value` の論理積） |
| `WorkflowAction` | `add_group_member`、`remove_group_member`、`assign_application`、`unassign_application`、`set_required_action`、`clear_required_action`、`enable_user`、`disable_user`、`send_email` のいずれか。すでに目的の状態なら `no_op` とする |
| `WorkflowStep` | アクション 1 件の実行の記録。結果（`changed`、`no_op`、`failed`、`canceled`）、機密情報を除いたエラーコード、実行の時刻 |

- **判断**：アクションを線形の列に限る理由は、[アクションを定義順の線形の列に限る](decisions.md#アクションを定義順の線形の列に限る)。
- **判断**：WorkflowRun に展開したアクションとリビジョンを固定する理由は、[WorkflowRun の作成時に展開したアクションとリビジョンを固定する](decisions.md#workflowrun-の作成時に展開したアクションとリビジョンを固定する)。
