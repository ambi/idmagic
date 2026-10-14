---
status: completed
archived_from: bb8b8175200d963c2b68946eff1c6aa4d4a57935
---

# API 規則を、基礎から観点の揃った API ガイドラインへ書き直す

API ガイドラインが、`DOCUMENTATION_GUIDE.md` §4.5 の全観点をルール単位の小見出しで持ち、各ルールに目的、担保手段、適用状況を記載するようになった。
当初は 4 列の表で書いたが、ルール本文が長く表では横に伸びて読めなかったため、小見出しと 3 項目の箇条書きに改めた。「接点」「強制点」「ワイヤ本体」などの造語も、「API 区分」「担保手段」「HTTP ボディの宣言」などの用語に改めた。
リソースの命名、値の表現、HTTP メソッドの割り当て、コレクション操作、冪等性と再試行、条件付きリクエスト、長時間実行操作、カスタムメソッド、上限とレートリミットを新しく書き、既存の 8 節は意味を変えずに組み直した。
決めた論点は次のとおりである。
パラメーターとプロパティはスネークケース、静的パスセグメントはケバブケース、日時は RFC 3339 の UTC、期間は単位付きの整数、列挙値は小文字のスネークケース、値の不在は省略で表す。
管理 API の作成、長時間実行操作の開始、クレデンシャルを発行する `POST` は、任意指定の `Idempotency-Key` を受け付ける。自動化クライアントのタイムアウト後の再送で、管理されないクレデンシャルが二重に作成されることを防ぐためである。
管理 API の更新は `ETag` と `If-Match` で競合を検出し、不一致を 412 で返す。`PUT` で全体を置換するセキュリティ設定が、同時編集で気付かれないまま失われることを防ぐためである。
長時間実行操作は、AIP-151 の汎用 `Operation` ではなく、操作ごとのリソースを 202 で返す。
レートリミットの残量ヘッダーは返さない。制限の対象が認証とトークン発行の経路であり、残量は攻撃者に制限を回避する送信間隔を教えるためである。
実装が従っていない箇所は適用状況に記載し、すべてに work item を起票した。
[[wi-593-kebab-case-static-path-segments]]、[[wi-594-lower-snake-case-enum-values]]、[[wi-595-declare-uniqueness-conflict-responses]]、[[wi-596-paginate-remaining-collections-with-shared-cursor]]、[[wi-597-declare-pagination-response-headers]]、[[wi-598-declare-audit-event-query-parameters]]、[[wi-599-decode-admin-request-bodies-strictly]]、[[wi-600-bound-relation-tuple-write-batch-size]]、[[wi-601-declare-date-format-for-attribute-values]]、[[wi-602-carry-deprecation-dates-into-runtime-contract]]、[[wi-603-declare-interface-stability-in-typespec]]、[[wi-604-accept-idempotency-key-on-admin-post]]、[[wi-605-detect-lost-updates-with-etag-and-if-match]]、[[wi-606-verify-null-omission-in-responses-and-patch]] である。
コードのコメントが英語の旧節名で指していた箇所を、現在の見出しに合わせた。
外部ガイドラインのページは取得しておらず、観点の点検は §4.5 の観点表と §11 が挙げる出典の既知の観点で行った。
`mise run spec-diff` は規範の変更を報告しない。

全文は `git show bb8b8175200d963c2b68946eff1c6aa4d4a57935:<このファイルのパス>` で読む。
