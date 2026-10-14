---
status: completed
archived_from: a8e03c729bc0cb0163e004b8abe40c9d40181425
---

# すべての API に共通する外部の振る舞いを、規範 ID を持つ要件として宣言する

`mise run spec-diff` の結果は、REQ-SYSTEM-022 から REQ-SYSTEM-030 の追加と、REQ-AUDIT-004 の変更である。
汎用 API に共通する振る舞いを、新しい機能スライス「HTTP API の共通の振る舞い」の要件として宣言した。
宣言したのは、Problem Details の形式、エラーコードのない失敗の 500、ページサイズのデフォルトと上限、カーソルのテナントと絞り込みへの束縛、ページ送りの `Link` と `Pagination-*` ヘッダー、`HTTP_MAX_BODY_BYTES` による 413、コードポイントで数える文字列長と 422、セキュリティヘッダー、レートリミットの拒否の形式である。
REQ-AUDIT-004 は、要件文を変えずに上位の要件の欄で REQ-SYSTEM-024 の例外として位置付けた。
`api-guidelines.md` と `authorization.md` は、宣言済みの要件へリンクし、契約の書き方、目的、担保手段、適用状況を書く文書になった。
デフォルトのページサイズは `support_http.DefaultPageLimit` と `MaxPageLimit` へ集め、値が 50 件と 200 件の 8 つのコレクションのハンドラーがこれを参照する。
`SPECIFICATION_FORMAT.md` は、システム全体にかかる要件も共有の仕組みの機能スライスで宣言すると定めた。

全文は `git show a8e03c729bc0cb0163e004b8abe40c9d40181425:<このファイルのパス>` で読む。
