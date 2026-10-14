---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 認可リクエストが宣言する標準 15 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
`docs/domain/oauth2/standards.md` のうち、認可リクエストの入口に立つ **15 行**が、その行の
`Statement` を区別できる入力と観測を持つテストを得て `tools/check/standards-coverage-debt.json` から
消えた。台帳は 90 件から 75 件になった。テストは 1 ファイル
（`backend/shared/http/server_http/authorization_request_standards_test.go`、11 テスト）で、
`Register` が組み立てたスタックへ HTTP リクエストを出す。製品コードは 1 行も変わっていない。
15 行とも宣言した採用を満たしており、欠陥の切り出しは無い。
**ハーネスは 1 つで足りたが、通した経路は 4 つになった。** `/authorize` と `/par` に加えて、
`RFC6749-AUTHORIZATION-CODE`、`RFC7636-VERIFY`、`RFC9396-MONOTONIC-NARROWING` が `/token` を、
`RFC7591-REDIRECT-URI` が `/register` を名指しているからである。
[[wi-499-back-oauth2-standards-rows-with-tests]] の分割の基準は「行が共有する入口」だが、ここで効いた
単位は 1 つの HTTP 経路ではなく **1 つのハーネス**だった。経路そのものを単位にしていたら、この 15 行は
4 件へ割れていたはずである。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
