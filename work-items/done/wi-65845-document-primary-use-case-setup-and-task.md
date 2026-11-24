---
status: completed
---

# 主要ユースケースの証拠を、形式の文書だけで検査を通るように書けるようにする

`docs/formats/work-item-format.md` だけで、検査を通る `primary_use_cases` を書けるようにした。
「基本形式」のテンプレートの例に `setup` を足した。
「主要ユースケースの証拠」の節からはテンプレートへリンクした。
`requirement` は同じ記録の `affected_spec` から選ぶこと、`test.task` は `verify` が実行するタスクか CI が `mise run` で呼ぶタスク（Go なら `test-go`）であることを明記した。
あわせて、どの条件を着手中の `check-work-items` が検査し、どの条件を完了時の検査が検査するかも書いた。
例は一か所に置き、二重に書いて片方が古くなることを避けた。
