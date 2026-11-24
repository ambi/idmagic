---
status: completed
---

# 表記の検査が拒否する語を、書く前に文書から分かるようにする

`docs/formats/documentation-guide.md` の用語の箇条に、`mise run check-terminology` が採らない表記を検査すること、組の一次情報が `tools/check/src/terminology.ts` の `TERMINOLOGY_RULES` であること、検査の対象、照合が文脈を読まない部分文字列の一致であることを書き、規則のファイルへリンクした。
一覧は文書へ写さず、コードを一次情報のままにした。規則の追加時に片方だけが古くなることを避けるためである。
規範仕様の差分はない（`mise run spec-diff`）。
