---
status: completed
archived_from: fd0469b83ceb080f82ba74b83e030ff9c5c6b408
---

# テナントをまたぐ Application 参照の拒否契約を実装と一致させる

`mise run spec-diff` は REQ-APPLICATION-007 と REQ-APPLICATION-008 のシナリオ変更と、TypeSpec 宣言 `ApplicationNotFoundError` と `ApplicationCategoryNotFoundError` の追加を報告した。
`EX-APPLICATION-007-03` と `EX-APPLICATION-008-03` は、別テナントの id による取得を `InvalidRequestError` で拒否するという記述から、存在しない id を指定したときと同じ 404（`application_not_found`、`not_found`）で応答し、名称、OIDC 設定、画像の内容を含まないという記述へ変わった。
Application 管理 API 18 件に、実際に返していた 404 を宣言した。Application を指定する 16 件は `ApplicationNotFoundError`、カテゴリーを指定する 2 件は `ApplicationCategoryNotFoundError` である。
製品コードは変更していない。`tools/check/example-coverage-debt.json` から 2 件の項目を削除し、API ガイドラインの適用状況とリリースノートを追記した。

全文は `git show fd0469b83ceb080f82ba74b83e030ff9c5c6b408:<このファイルのパス>` で読む。
