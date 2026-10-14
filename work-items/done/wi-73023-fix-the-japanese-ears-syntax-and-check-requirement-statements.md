---
status: completed
archived_from: bb8b8175200d963c2b68946eff1c6aa4d4a57935
---

# 要件文を、応答の主体と前置きの標識を固定した日本語の EARS で書き、`check-spec` で検査する

`mise run spec-diff -- main` の結果、規範の差分は REQ-IDMANAGEMENT と REQ-TENANCY の 131 要件の本文の書き直しだけであり、要件の追加、削除、ID とタイトルの変更はなく、`affected_spec` と一致した。
`SPECIFICATION_FORMAT.md` の EARS 形式の節を、前置き、主体、応答の順の日本語の構文に改めた。主体は Context の `README.md` の H1、前置きの標識は「では、」「の間、」「とき、」「場合、」に固定し、「場合」を望まない入力に限った。応答の中の条件と、能力、許可、推奨、定義の文末を禁じ、一行に一文だけを書く。理由と参考にした資料は `docs/development/format-rationale.md` に書いた。
`check-spec` に要件文の構文の検査を加え、Tenancy と IdManagement を検査の一覧に載せた。残りの Context は wi-35451 が書き直して加える。
対象の 20 文書は 2,017 行から 2,060 行になった。前置きを繰り返して行を分けた分だけ増えた。

全文は `git show bb8b8175200d963c2b68946eff1c6aa4d4a57935:<このファイルのパス>` で読む。
