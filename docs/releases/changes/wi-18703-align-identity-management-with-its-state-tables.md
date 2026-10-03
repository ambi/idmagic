# WI-18703: Align IdManagement with its state tables

作業項目は `wi-18703-align-identity-management-with-its-state-tables` である。

WI-18703 は、IdManagement の実装を、`UserLifecycle` と `DataExportLifecycle` の状態遷移表に合わせる。

管理者は、削除を予約した User を無効化も再有効化もできなくなる。
`POST /api/admin/v1/users/{user_id}/disable` と `POST /api/admin/v1/users/{user_id}/enable` は、対象が削除予約中なら 409 と `user_pending_deletion` を返し、User を変えない。
削除の予約を取り消すには、猶予期間を確かめる復元を使う。
規範上の条件は [REQ-IDMANAGEMENT-046](../../domain/identity-management/user/lifecycle.md) が定める。

データエクスポートの保持期限は、作成の時刻ではなく完了の時刻から 30 日になる。
エクスポートの参照の応答は、`succeeded` のエクスポートにだけ `expires_at` を返す。
規範上の条件は [REQ-IDMANAGEMENT-079](../../domain/identity-management/data-export/README.md#req-idmanagement-079-エクスポートの保持期限は完了の時刻から-30-日である) が定める。

Batch の `retention-sweep` は、作成から 30 日を過ぎた CSV の成果物（エクスポートのファイル、インポートのペイロード、行のエラーのページ）を消す。
30 日より前のプレビューを指定した適用は、ペイロードがないため失敗する。
規範上の条件は [REQ-IDMANAGEMENT-080](../../domain/identity-management/csv-transfer/README.md#req-idmanagement-080-csv-の成果物は作成から-30-日を過ぎると消す) が定める。
