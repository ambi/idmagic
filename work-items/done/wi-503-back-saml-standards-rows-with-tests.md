---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# SAML が宣言する標準 6 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
差分は `docs/domain/saml/standards.md` の 7 行のうち、名指しを持たなかった 6 行に対する被覆の状態で
ある。6 行すべてがその行の `Statement` を区別できる入力と観測を持つテストを得て
`tools/check/standards-coverage-debt.json` から消え、台帳は 110 件から 104 件になった。
新設したテストは `backend/saml/handlers_http/saml_standards_test.go` の 6 件で、既存の 2 件へ注記を
足した。製品コードは 1 行も変わっていない。欠陥は 1 件も見つからなかった。
`excluded` の観測の型を 1 つ決めた（Design の該当節）。**この型は既に決まっていた 2 つの型のどちらとも
違う。** 標準側の機能を実装していない行は、拒否ではなく「入口が無いこと」を、公開している契約と実際の
応答の両方から読む。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
