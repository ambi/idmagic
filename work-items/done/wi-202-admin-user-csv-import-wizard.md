---
status: completed
archived_from: 4235bc48d42453a705ffb0e208002255bc95e364
---

# 管理画面に CSV ユーザーインポートウィザードを追加する

`spec/contexts/identity-management.yaml` の AdminUsers 画面契約に import wizard の状態を追加し、
既存の CSV インポート受け入れシナリオの error code を実装 (`invalid_header` / `csv_too_large` /
`too_many_rows` / `field_too_large`) に合わせて訂正した。`frontend/src/features/admin-users/` に
`AdminUserImportPage` (CSV 選択 → テンプレート DL → dry-run → 行エラープレビュー →
明示確認付き apply → 結果表示) を実装し、`/admin/users/import` ルートと一覧ページからの導線を追加した。
`importAdminUsers` / `getAdminUserImport` の API クライアントと型、stable error code → ja/en
翻訳文言、job polling とタイムアウト処理、コンポーネントテストを追加した。

全文は `git show 4235bc48d42453a705ffb0e208002255bc95e364:<このファイルのパス>` で読む。
