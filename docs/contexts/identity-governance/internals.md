# IdGovernance Internals

## Trigger capture via Transactional Outbox

`IdManagement` は User のライフサイクルイベント (`UserCreated`、`UserAttributesChanged`、`UserStatusChanged`) を、ユーザーの変更と同じトランザクションで Transactional Outbox へ書く。`IdGovernance` はそのイベントを消費し、`WorkflowRun` と `WorkflowStep` の行を作る。これにより、Context をまたぐ単一トランザクションを要求せずに、User だけが更新されて対応する実行が作られない事態を防ぐ。イベントは少なくとも 1 回配信されるため、`(tenant_id, workflow_id, revision, source_occurrence_id, target_user_id)` の一意制約を使って、同じ発火事象の重複配信を 1 つの実行へ収束させる。

## Action execution via published command surface

9 種類の実行アクション（グループメンバーシップの追加と削除、アプリケーションの割り当てと解除、ユーザーの有効化と無効化、必須操作の設定と解除、メール送信）は、記録系 Context のドメイン型を直接呼び出さない。代わりに `IdManagement` と `Application` が冪等なコマンドインターフェースを公開し、実行側が Context の境界を越えて呼び出す。具体的なアダプターは Composition Root で注入する。依存方向は `IdGovernance` から `IdManagement` と `Application` への一方向であり、循環しない。

## Partial failure and loop suppression

1 回の試行では、未完了のステップをすべて定義順に実行し、失敗したステップがあっても止めない。無関係な通知の失敗がアクセス剥奪のステップを妨げないようにするためである。実行は結果の組み合わせに応じて `succeeded`、`partially_failed`、`failed` のいずれかで終了し、Context をまたぐ補償やロールバックは行わない。`WorkflowRunTriggerSnapshot` は、アクションによる User の変更について、発生元の実行とステップを記録する。トリガー評価ではこの発生元情報を持つ変更を除外するため、アクションからトリガーへの循環は実行時の防御ではなく評価器への入力時点で構造的に断たれる。`WorkflowRun`、`WorkflowStep`、`LifecycleNotificationDelivery` は `Job` の記録と同じ 30 日の保持期間に従う。
