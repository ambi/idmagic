# ワークフローの実行

## 概要

この文書は、User の変更を契機に WorkflowRun を作り、`worker` がそのステップを実行する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | User の変更と同じトランザクションでの WorkflowRun の捕捉、Job の関連付け、ステップの実行、再試行、同じ User の実行の直列化 |
| 行為者 | テナント管理者（User の変更の契機）、System（ディスパッチャーと `worker`） |
| 扱わないもの | ワークフローの定義と無効化は[ライフサイクルワークフローの定義](../lifecycle-workflow/README.md)が扱う |

## モデル

トリガーは、User を変更した後の状態に対して評価し、属性の変更の前後の比較だけは変更の差分を使う。
同じ発火の事象は、重複排除のキー（`tenant_id`、`workflow_id`、`revision`、`source_occurrence_id`、`target_user_id`）で一つの WorkflowRun に収束する。

1 回の試行では、未完了のステップをすべて定義順に実行し、失敗したステップがあっても止めない。
再試行では `failed` のステップだけを再実行し、`changed` または `no_op` になったステップは飛ばす。

- **判断**：失敗しても止めず補償もしない理由は、[部分的な失敗で止めず補償もしない](../design/decisions.md#部分的な失敗で止めず補償もしない)。

## 状態遷移

### WorkflowRunLifecycle

`WorkflowRun` は `queued` で作成し、ディスパッチャーが `job_id` を関連付けて最初のステップを開始すると `running` に遷移する。1 回の試行では、未完了のステップを定義順にすべて実行する。Job の試行上限に達した時点で、全ステップが成功していれば `succeeded`、成功と失敗が混在していれば `partially_failed`、成功が 1 つもなければ `failed` で終了する。`no_op` は成功として扱い、Jobs 側で再試行している間は `running` のままにする。ワークフローを無効化すると、未開始の `queued` の実行は `canceled` になり、`running` の実行は現在のステップのチェックポイント後、次のステップを始める前に `canceled` になる。

| State | Kind | Meaning |
|---|---|---|
| queued | initial | 作成直後。ディスパッチャーが `job_id` を関連付けるのを待つ |
| running | — | 未完了のステップを定義順に実行している |
| succeeded | terminal | 全ステップが成功した |
| partially_failed | terminal | 成功と失敗が混在した |
| failed | terminal | 成功したステップが 1 つもなかった |
| canceled | terminal | ワークフローの無効化により、開始前またはステップの区切りで打ち切った |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | LifecycleWorkflowRunStarted | — | running |  |
| running | LifecycleWorkflowRunSucceeded | — | succeeded |  |
| running | LifecycleWorkflowRunPartiallyFailed | — | partially_failed |  |
| running | LifecycleWorkflowRunFailed | — | failed |  |
| queued | LifecycleWorkflowRunCanceled | — | canceled |  |
| running | LifecycleWorkflowRunCanceled | — | canceled |  |

## 操作

### User の変更による実行の捕捉

#### REQ-IDGOVERNANCE-004 User の変更と WorkflowRun の捕捉は同じ整合性境界で確定し、キュー投入の障害から回復する

#### REQ-IDGOVERNANCE-006 値が変わらない属性更新と動的グループのアクション指定は境界条件として扱われる

### worker による WorkflowRun の実行

#### REQ-IDGOVERNANCE-003 管理者が定義したワークフローが部署変更に応じてグループとアプリケーションの割り当てを更新する

#### REQ-IDGOVERNANCE-005 すでに目的の状態にある操作は `no_op` として扱われる

#### REQ-IDGOVERNANCE-009 一時的な失敗は再試行時に成功済みのステップを飛ばして収束する

#### REQ-IDGOVERNANCE-010 同一ユーザーの WorkflowRun は発火順に直列化され、副作用の前に再検証される

#### REQ-IDGOVERNANCE-008 通知に失敗してもアクセス剥奪操作は前進する

## セキュリティ上の考慮

WorkflowRun の実行は、管理者の権限を借りない。
対象の User、Group、Application は、いずれも実行の時点でワークフローと同じテナントの中で再取得する。
実行までに対象が削除または移動していれば、そのステップを失敗として記録する。
ワークフローがテナントの境界を越える経路は、定義の側にも実行の側にもない。
