---
status: completed
archived_from: 63d431214e955ed3676abf0d07a767b89de320be
---

# IdManagement の実装を状態遷移表に合わせる

`mise run spec-diff -- work-item/wi-21670` は、追加した規則 REQ-IDMANAGEMENT-079 と 080、変更した規則 REQ-IDMANAGEMENT-046、追加した TypeSpec の `UserPendingDeletionError`、変更した `DisableAdminUser` と `EnableAdminUser` を報告する。
IdManagement の実装を `UserLifecycle` と `DataExportLifecycle` の状態遷移表に合わせた。
削除予約中の User の無効化と再有効化は、409 と `user_pending_deletion` で拒否し、User を変えない。
データエクスポートの保持期限を、作成の時刻ではなく完了の時刻から 30 日にし、`expires_at` は `succeeded` のエクスポートにだけ返す。
Batch の `retention-sweep` が、作成から 30 日を過ぎた CSV の成果物（エクスポートのファイル、インポートのペイロード、行のエラーのページ）を分割片ごと消す。

全文は `git show 63d431214e955ed3676abf0d07a767b89de320be:<このファイルのパス>` で読む。
