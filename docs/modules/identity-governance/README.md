# IdGovernance

## 責務と境界

アイデンティティガバナンス（IGA）のポリシーとオーケストレーションを扱う。
JML を自動化する `LifecycleWorkflow` の定義、トリガーの評価、`WorkflowRun` の実行がここに属する。

| 扱わないもの | 担当 |
| --- | --- |
| User と Group の記録の正 | `IdManagement` |
| Application の割り当ての記録の正 | `Application` |
| WorkflowRun を実行する Job のキューと再試行 | `Jobs` |
| メールの送信の部品 | 共有の通知の部品 |

このモジュールは、User の変更を契機にワークフローを評価し、記録の正を持つモジュールの状態を冪等に変える。

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `LifecycleWorkflow` | 名前、状態、`current_revision` と `enabled_revision`、リビジョンごとの `WorkflowTrigger` と `WorkflowAction` の列 | `Tenant` を `tenant_id` で参照する。アクションは同じテナントの Group と Application を参照する |
| `WorkflowRun` | 作成時に固定した `workflow_id`、`revision`、発火の事象、対象の User、展開したアクションの列、`WorkflowStep` の記録、関連付けた Job | `LifecycleWorkflow` と User を参照する |

| 概念 | 内容 |
| --- | --- |
| `WorkflowTrigger` | `user_created`、`user_attributes_changed`、`user_status_changed` のいずれかと、0〜20 件の型付きのフィルター（`field`、`operator`、`value` の論理積） |
| `WorkflowAction` | `add_group_member`、`remove_group_member`、`assign_application`、`unassign_application`、`set_required_action`、`clear_required_action`、`enable_user`、`disable_user`、`send_email` のいずれか。すでに目的の状態なら `no_op` とする |
| `WorkflowStep` | アクション 1 件の実行の記録。結果（`changed`、`no_op`、`failed`、`canceled`）、機密情報を除いたエラーコード、実行の時刻 |

- **判断**：アクションを線形の列に限る理由は、[アクションを定義順の線形の列に限る](design/decisions.md#アクションを定義順の線形の列に限る)。
- **判断**：WorkflowRun に展開したアクションとリビジョンを固定する理由は、[WorkflowRun の作成時に展開したアクションとリビジョンを固定する](design/decisions.md#workflowrun-の作成時に展開したアクションとリビジョンを固定する)。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `Identity Governance` のタグが定める。
次の表は、それ以外にほかのモジュールと結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `UserMutationCommitter` | `IdManagement` が定め、このモジュールが実装する | このモジュールが提供する | User の変更から WorkflowRun を計画し、User と WorkflowRun を一つのトランザクションで保存する |
| `UserLifecycle` | `IdManagement` が実装する | このモジュールが定める | `enable_user` と `disable_user` を、IdManagement の User の操作として行う |
| `ApplicationAssignments` | `Application` が実装する | このモジュールが定める | `assign_application` と `unassign_application` を、Application の割り当ての操作として行う |
| `lifecycle_workflow_run` の Job | `Jobs` が実行する | このモジュールが定める | WorkflowRun を一回試行する |
| ドメインイベント | 監査と下流が購読する | このモジュールが発行する | `LifecycleWorkflow…`、`LifecycleWorkflowRun…`、`LifecycleWorkflowStepFailed` |

## 機能

| 機能 | 内容 |
| --- | --- |
| [ライフサイクルワークフローの定義](lifecycle-workflow/README.md) | 作成、編集、有効化と無効化、削除、取得、プレビュー |
| [ワークフローの実行](workflow-run/README.md) | User の変更による実行の捕捉、Job の関連付け、ステップの実行と再試行 |

| 文書 | 内容 |
| --- | --- |
| [IdGovernance の用語集](glossary.md) | このモジュールでの語義 |
| [IdGovernance の設計](design/README.md) | 設計領域ごとの設計と重要な判断 |
