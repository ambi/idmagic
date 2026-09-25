# wi-22987-track-full-resync-completion

Provisioning が、管理者の開始した Full Resync の完了を `FullResyncCompleted` として記録するようになった（[`REQ-PROVISIONING-013`](../../domain/provisioning/scenarios.feature.md)）。

- Full Resync が作ったプロビジョニングタスクがすべて `succeeded` または `dead_letter` になった時点で、`FullResyncCompleted` を一度だけ発行する。これまでは Full Resync を開始してもこのイベントを発行していなかった。
- イベントの `totalSubjects` は開始時の対象数、`succeededCount` と `failedCount` は完了時点で成功したプロビジョニングタスクと再試行を使い切ったプロビジョニングタスクの件数である。
- 対象が 0 件の Full Resync は、開始と同時に完了する。
- 完了後に `dead_letter` のプロビジョニングタスクを再試行しても、完了済みの Full Resync の件数は変わらず、イベントも再び発行しない。
- `StartFullResync` の応答（`enqueued_count`）は変わらない。
