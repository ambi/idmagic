# WI-20633: CSV のエクスポートが転送ポリシーのどの上限を超えても `csv_transfer_limit_exceeded` で失敗する

作業項目は `wi-20633-report-every-transfer-limit-as-csv-transfer-limit-exceeded` である。

WI-20633 は、User、Group、メンバーシップの CSV のエクスポートで、実効の転送ポリシーの `max_rows` または `max_field_bytes` を超えたときの失敗のコードを、`export_failed` から `csv_transfer_limit_exceeded` に変える。

これまでは `max_bytes` の超過だけを `csv_transfer_limit_exceeded` にし、行数と項目長の超過を理由のない `export_failed` にしていた。
エクスポートのジョブの失敗のコードと `DataExportFailed` イベントの `errorCode` で `export_failed` を条件にしている処理は、この二つの上限の超過では分岐が変わる。

規範上の条件は[ユーザー CSV](../../modules/identity-management/features/user-csv.md)の REQ-IDMANAGEMENT-007、[グループ CSV](../../modules/identity-management/features/group-csv.md)の REQ-IDMANAGEMENT-027 と REQ-IDMANAGEMENT-030 が定める。
