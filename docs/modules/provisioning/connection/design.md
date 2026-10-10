# 接続の管理の設計

この文書は、[接続の管理](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

`StartFullResync` は、接続のスコープの中の User と、`push_groups` の対象となる Group を列挙し、重複を除いた件数を対象の数とする `FullResync` を保存する。
続いて、対象ごとに `update` のタスクを作り、`FullResync` へ関連付ける。
タスクの挿入と関連付けは、一つの SQL の文で行う。

| 状態 | 意味 | 遷移の契機 |
| --- | --- | --- |
| `running` | 対象のタスクの決着を待っている | `StartFullResync` |
| `completed` | 対象のタスクがすべて `succeeded` または `dead_letter` になった | 最後のタスクを終端にしたジョブ、または対象が 0 件の `StartFullResync` |

完了の判定は、関連付いたタスクのうち `succeeded` と `dead_letter` を数え直して行い、その合計が対象の数に達したときだけ完了とする。
関連付けは一件ずつで対象の数を超えないので、合計が対象の数に達したことは、全件が作られて終端になったことを表す。
`StartFullResync` がタスクを作り終える前に先のタスクが終わっても、完了しない。

## 信頼性

ジョブのハンドラーは、タスクを `succeeded` または `dead_letter` として保存した後に、完了を判定する。
`completed` への更新は `running` のときだけ成り立つ条件付きの更新であり、成り立った呼び出しだけが `FullResyncCompleted` を発行する。
同じジョブの再実行や、最後の二件を同時に終えた二つのジョブがあっても、イベントは一度だけ発行される。
完了の判定が失敗したジョブは Jobs が再実行し、終端済みのタスクについて判定だけをやり直す。

`FullResyncCompleted` の `totalSubjects` は対象の数、`succeededCount` と `failedCount` は完了の時点の `succeeded` と `dead_letter` の件数である。
完了の後に管理者が `dead_letter` のタスクを再試行しても、完了済みの `FullResync` は変わらない。
