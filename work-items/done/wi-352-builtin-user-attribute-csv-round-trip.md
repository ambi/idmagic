---
status: completed
archived_from: dc0961d035553d99cd2b9ce5a472240293d05838
---

# 組み込みユーザー拡張属性(middle_name, department など)を CSV エクスポート/インポートの対象に含める

組み込み拡張属性27種 (`middle_name` / `department` など) を User CSV
エクスポート/インポートの対象に追加した。CSV 列キーは `attr:<key>`
(組み込み拡張属性) と `custom:<key>` (tenant custom 属性) を区別する形に
統一し (`NewUserCSVSchema`)、export/import の use case 層は既存の汎用実装
(`UserCSVColumn.Attribute != nil`) のままで新列を読み書きできることを確認
した。UI (`DataExportPage.tsx`) は27種を静的にハードコードせず、既存の
`getTenantUserAttributeSchema()` レスポンスの `schema.builtin` を
`attr:<key>` 列として動的に描画する方式を採用し、3箇所目の allowlist 重複
を避けた。SCL (`models.DataExportColumn` / `interfaces.StartUserCsvExport`)
を先に更新し、派生物を再生成済み。
対応していないこと (Out of Scope どおり): Group / GroupMembership の CSV
対応 (wi-350, wi-351)、新しい組み込み属性の追加、password/secret 系列の
allowlist 追加公開。手動確認はブラウザ拡張が未接続のため画面操作では
実施できておらず、`just dev` によるフル dev stack 起動確認と、実
`TenantUserCSVSchemaReader` を使った export→import unchanged 往復の自動
テストで代替した。

全文は `git show dc0961d035553d99cd2b9ce5a472240293d05838:<このファイルのパス>` で読む。
