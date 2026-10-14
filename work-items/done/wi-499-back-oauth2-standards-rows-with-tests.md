---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# OAuth2 が宣言する標準 80 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
本項目は 2 つのことをした。1 つは `docs/domain/oauth2/standards.md` の
`OAuth Client ID Metadata Document` 節が宣言する **7 行の消化**である。7 行とも、その行の `Statement` を
区別できる入力と観測を持つテストを得て `tools/check/standards-coverage-debt.json` から消え、台帳は
97 件から 90 件になった。テストは 1 ファイル（`backend/oauth2/client/cimd_http/standards_test.go`、
6 テスト）で、文書を配る TLS サーバーの上に製品の `Fetcher` と `ClientRepositoryWithCIMD` を組み、
そこへ `client_id` を通す。製品コードは 1 行も変わっていない。7 行とも宣言した採用を満たしており、
欠陥の切り出しは無い。
もう 1 つは **T002 の分割**である。残る 73 行を、節でも機能領域でもなく**行が共有する製品の入口**を
単位に 7 件へ割り、[[wi-516-back-oauth2-authorization-request-standards-rows-with-tests]] から
[[wi-522-back-oauth2-client-profile-standards-rows-with-tests]] までを起票して
[[wi-495-burn-down-the-standards-coverage-debt]] の `depends_on` へ足した。この単位を選んだ根拠は
T001 の測定にある。7 行の所要を支配したのは行数ではなく、7 行が共有する入口にハーネスを 1 つ組むこと
だった。節で割ると同じハーネスを何度も組み直すことになり、実際
`RFC6749-AUTHORIZATION-CODE`、`RFC7636-VERIFY`、`RFC9207-ISS`、`RFC9700-REDIRECT-MATCH` は 4 つの別々の
節にありながら同じ観測をする。割り当ては網羅かつ排他であることを台帳と突き合わせて確かめた。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
