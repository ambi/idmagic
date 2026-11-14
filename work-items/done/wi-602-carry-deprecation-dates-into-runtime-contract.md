---
status: cancelled
---

# TypeSpec の非推奨日をランタイムの契約へ伝達する

中止した。
監査のエクスポートをジョブへ移した変更（4c8cfdff0）が、`x-deprecated-since` と `x-sunset-at` を生成器から `RuntimeContract.Deprecations` へ運ぶ形を先に実装していた。
2026-11-14 の棚卸しで、`ExportAdminAuditEvents` と `ExportSystemAuditEvents` の日付が生成物に載り、12 か月未満の廃止日を生成器のテストが拒否することを確かめた。
この項目に残る作業はない。
