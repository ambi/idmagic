---
status: completed
archived_from: 081de747cff0adfa60602d7ac6fd3d208c5cbe4c
---

# 既存の Markdown と TypeSpec の説明文を日本語へ移行する

`mise run spec-diff` の結果は、説明文を書き換えた TypeSpec の宣言 1,238 件の一覧だけで、規則と具体例（REQ、EX）の差分はない。
TypeSpec の `@doc` 2,231 件を日本語にし、OpenAPI の `description` と仕様サイトの API リファレンスが日本語になった。
`description` を除いた OpenAPI は翻訳前と完全に一致し、`description` の件数（4,189 件）と、その中の `urn:idmagic:error:*` の多重集合も一致した。
`CONFIGURATION.md` と `ROUTE_PRIORITY.md` を生成する Go の文字列、リリース文書の英文の題名、残っていた Markdown の英文を日本語にした。
条件付きで必須の設定は、英文の代わりに `SEED_PROFILE != ""` のようなキーの式で示す。
work item の完了節のフィールド名は、日本語の表記でも英語の表記でも同じ項目として読むようにし、形式文書のテンプレートを日本語にした。
生成サイトの本文に英文が残ると、`render-docs` と `check-rendered-docs` が失敗するようにした。原表記を保つ語句は、規格名、文書名、製品名、記法、用語のいずれかの分類を付けて許容語の表に載せる。

全文は `git show 081de747cff0adfa60602d7ac6fd3d208c5cbe4c:<このファイルのパス>` で読む。
