---
status: completed
archived_from: e50ccb1f62ec962e4c3e592b991b0b096589a9d2
---

# グループ単位のMembership CSVを安全に往復できるUIまで実装する

`mise run spec-diff` が示す規範上の差分は次のとおり。追加された scenario は
`REQ-IDMANAGEMENT-029` (1 つの Group に固定したメンバーシップ CSV の事前検証と適用、
`present|absent` の閉じた語彙、`user_id` 優先の識別子解決、`group_id`/`group_name` の
照合、同一 tenant かつ同一 group のプレビュー結合、現在状態からの再計画、行原子性)、
`REQ-IDMANAGEMENT-030` (無編集の export→preview が全行 unchanged になる往復と、
分割ファイルでも欠落行を解除しない安全性、10,000 membership の容量契約)、
`REQ-IDMANAGEMENT-031` (dynamic Group、dynamic_rule 由来 membership、source-managed
Group/User、対象 Group 不在の fail-closed 拒否)。
追加された TypeSpec 宣言は `ImportAdminGroupMembers` / `ApplyAdminGroupMemberImport` /
`GetAdminGroupMemberImport`、`GroupMembershipCsvState` / `GroupMembershipImportJob` /
`GroupMembershipImportJobRef` / `GroupMembershipImportResult` /
`GroupMembershipImportRowError` / `GroupMembershipImportMode` /
`GroupMembershipImportNotFoundError` / `GroupMembershipImportUnavailableError`。
削除された宣言は無く、`mise run check-api-compat` は破壊的変更なしと判定した。
`GroupMember` / `GroupMembershipSource` / `StartGroupMemberCsvExport` の doc は、
CSV 上の読み取り専用性と機械キーへの移行を述べるよう変更した。
Jobs 側では `JobKind` に `group_membership_import_preview` /
`group_membership_import_apply` を追加した。

全文は `git show e50ccb1f62ec962e4c3e592b991b0b096589a9d2:<このファイルのパス>` で読む。
