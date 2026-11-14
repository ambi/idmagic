---
status: completed
---

# CSV のエクスポートが転送ポリシーのどの上限を超えても `csv_transfer_limit_exceeded` で失敗させる

規範仕様の差分はない（`spec-diff`：no normative specification change）。実装を REQ-IDMANAGEMENT-007、REQ-IDMANAGEMENT-027、REQ-IDMANAGEMENT-030 に合わせた。
User、Group、メンバーシップの CSV のエクスポートのジョブは、行数、成果物の大きさ、項目の大きさのどの上限の超過でも `csv_transfer_limit_exceeded` で失敗し、`DataExportFailed` の `errorCode` も同じコードを運ぶ。これまで行数と項目の大きさの超過は `export_failed` だった。
超過の判定は domain の `CSVErrorCode.ExceedsTransferPolicy` に一つだけ置き、取り込みの上限検査とエクスポートの失敗のコードが同じ判定を共有する。
`export_failed` を条件にしていた処理向けに、告知の断片 `docs/releases/changes/wi-20633-report-every-transfer-limit-as-csv-transfer-limit-exceeded.md` を加えた。
