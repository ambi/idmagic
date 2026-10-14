---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# トークンの発行と交換が宣言する標準 15 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
`docs/domain/oauth2/standards.md` のうち、トークンの発行と交換の入口に立つ **15 行**が、その行の
`Statement` を区別できる入力と観測を持つテストを得て `tools/check/standards-coverage-debt.json` から
消えた。台帳は 75 件から 60 件になった。テストは 1 ファイル
（`backend/shared/http/server_http/token_issuance_standards_test.go`、6 テスト）で、`Register` が
組み立てたスタックの `/token` へリクエストを出し、**返ったトークンの payload を復号して直接読む**。
製品コードは 1 行も変わっていない。15 行とも宣言した採用を満たしており、欠陥の切り出しは無い。
この 15 行が言っているのは「何を出すか」なので、応答が 200 であることや、そのトークンで保護リソースへ
到達できることでは足りない。`jti` を落とす変異は「到達できる」という観測を生き残る。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
