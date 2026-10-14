---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# トークンの提示・内省・失効・送信者制約が宣言する標準 13 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
`docs/domain/oauth2/standards.md` のうち、発行済みトークンの提示・内省・失効・送信者制約に立つ
**13 行**が、その行の `Statement` を区別できる入力と観測を持つテストを得て
`tools/check/standards-coverage-debt.json` から消えた。台帳は 60 件から 47 件になった。テストは
1 ファイル（`backend/shared/http/server_http/token_presentation_standards_test.go`、9 テスト）で、
`Register` が組み立てたスタックの `/token` でトークンを作り、`/userinfo`、`/introspect`、`/revoke`
へ提示する。製品コードは 1 行も変わっていない。
この 13 行が言っているのは「受け取ったトークンをどう扱うか」なので、観測は必ず**同じ 1 本のトークンを
条件だけ変えて提示する**形になる。別のトークンで比べると、拒否がトークンの側の欠陥によるものか提示の
側の条件によるものか区別できない。失効と `excluded` の 2 種類は、応答に加えて**拒否が防いだ効果**を
読む。応答だけを見るテストは、200 を返しながら他クライアントのトークンを失効させる実装を見逃す。
`optional` の 6 行は 6 行とも提供されており、規範の変更として切り出したものは無い。行の外で
`RFC7662-INTROSPECT` を観測する途中に見つけた 1 件を
[[wi-523-introspection-token-type-is-not-the-rfc6749-token-type]] として切り出した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
