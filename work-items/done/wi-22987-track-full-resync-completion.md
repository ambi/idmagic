---
status: completed
archived_from: 5ddf8cdc8b7a83e926efe613bfc2c6fac4eff5d1
---

# Full Resync のプロビジョニングタスクの完了を追跡し、全対象が収束したときに完了イベントを発行する

`mise run spec-diff` は main に対する規範仕様の変更なしを報告した（`REQ-PROVISIONING-013` と `FullResyncCompleted` の宣言は変えていない）。
`EX-PROVISIONING-013-01` の振る舞いを実装し、被覆の負債から外した。
`StartFullResync` は重複を除いた対象数を持つ `FullResync` を保存し、各プロビジョニングタスクを一つの SQL 文で挿入して関連付ける。
ジョブハンドラーはプロビジョニングタスクを `succeeded` または `dead_letter` として保存した後、終端の件数を数え直し、対象数に達していれば `running` のときだけ成立する条件付き更新で完了させ、成立したときだけ `FullResyncCompleted` を発行する。
対象が 0 件なら `StartFullResync` が開始と同時に完了させる。
`provisioning_full_resyncs` と `provisioning_full_resync_tasks` の表を追加し、`internals.md` に完了追跡の仕組みを記した。
タスク一覧 API と `StartFullResync` の応答は変わらない。

全文は `git show 5ddf8cdc8b7a83e926efe613bfc2c6fac4eff5d1:<このファイルのパス>` で読む。
