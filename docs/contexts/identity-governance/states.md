# IdGovernance State Transitions

## WorkflowDefinitionLifecycle

`LifecycleWorkflow` は `draft` で作成する。完全な検証に成功したリビジョンを有効化すると、新しいトリガーの評価対象になる。無効化すると新しいトリガーを止め、後から再び有効化できる。管理者は `draft`、`enabled`、`disabled` のどの状態からも削除できる。`enabled` から削除した場合は新しいトリガーを止め、`queued` の WorkflowRun をキャンセルする。削除済みの定義は参照整合性を保つため、内部では終端状態の `archived` として保持するが、管理画面と通常の API には公開しない。実行履歴と `LifecycleWorkflowDeleted` 監査イベントは保持する。

| State | Kind | Meaning |
|---|---|---|
| draft | initial | 作成直後。トリガーの評価対象にならない |
| enabled | — | 新しいトリガーの評価対象になる |
| disabled | — | 新しいトリガーを止めている。再び有効化できる |
| archived | terminal | 削除済み。参照整合性のために保持し、管理画面と通常の API には公開しない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| draft | LifecycleWorkflowEnabled | — | enabled |  |
| enabled | LifecycleWorkflowDisabled | — | disabled |  |
| disabled | LifecycleWorkflowEnabled | — | enabled |  |
| draft | LifecycleWorkflowDeleted | — | archived |  |
| enabled | LifecycleWorkflowDeleted | — | archived |  |
| disabled | LifecycleWorkflowDeleted | — | archived |  |

## WorkflowRunLifecycle

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
