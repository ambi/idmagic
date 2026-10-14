---
status: completed
archived_from: e50ccb1f62ec962e4c3e592b991b0b096589a9d2
---

# Group CSV を安全に往復できる部分 upsert と明示的な削除へ拡張する

`mise run spec-diff` が示す規範上の差分は次のとおり。追加された scenario は
`REQ-IDMANAGEMENT-026` (Group CSV の事前検証と適用、`membership_type` の immutable guard、
dynamic rule の最終状態検証、source-managed の fail-closed 拒否)、`REQ-IDMANAGEMENT-027`
(無編集の export→preview が全行 unchanged になる往復と 10,000 Group の容量契約)、
`REQ-IDMANAGEMENT-028` (`lifecycle_action=delete` による明示的な削除と cascade)。
変更された scenario は `REQ-IDMANAGEMENT-004` / `REQ-IDMANAGEMENT-007` (転送ポリシーの名前が
種別非依存の `CsvTransferPolicy` になった) と `REQ-IDMANAGEMENT-025` (`groups:read` だけの
CSV インポート要求の拒否と、`groups:write` が許すものの明示)。
追加された TypeSpec 宣言は `ImportAdminGroups` / `ApplyAdminGroupImport` / `GetAdminGroupImport`、
`GroupImportJob` / `GroupImportJobRef` / `GroupImportResult` / `GroupImportRowError` /
`GroupImportMode` / `GroupCsvLifecycleAction` / `CsvTransferPolicy`。削除されたのは
`UserCsvTransferPolicy` 1 件で、これは `CsvTransferPolicy` への改名である
(どの operation の body からも参照されておらず、`mise run check-api-compat` は破壊的変更なしと判定)。
Jobs 側では `JobKind` に `group_import_preview` / `group_import_apply` を追加した。
`REQ-IDMANAGEMENT-026` と `REQ-IDMANAGEMENT-027` は後から `email` と `custom:<key>` 列を
含むよう変更した (下記「列の追加」)。

全文は `git show e50ccb1f62ec962e4c3e592b991b0b096589a9d2:<このファイルのパス>` で読む。
