---
status: completed
---

# 属性値の日付を TypeSpec の日付型として宣言する

`mise run spec-diff` は、規範の変更として TypeSpec の `AttributeValue` だけを報告する。
`AttributeValue.date` を `plainDate` で宣言し、OpenAPI は `format: date` を出力するようになった。
JSON の表現は文字列のままで、Go の検証とフロントエンドの手書きの型は変えていない。
実行時の振る舞いを変えないため、`change_kind` を `bugfix` から `docs` に改めた。
API ガイドラインの「日時と日付」は全面適用になった。
仕様の TypeSpec で、日付を形式のない `string` で宣言していたのはこのプロパティだけだった。
