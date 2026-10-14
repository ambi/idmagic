---
status: completed
archived_from: 854f521daf0113319b826044107f04189d1f24c1
---

# ApiTokens が宣言する具体例 16 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は `main` に対して、規則 `REQ-AUTHENTICATION-037` と TypeSpec の `IdentityProviderConnectionInUseError` の追加を返した。
ApiTokens が台帳に残していた 16 件の具体例をすべて実装と照合し、`//spec:covers` を付けたテストへ結び付けて台帳から外した。
UI の 4 件（`EX-APITOKENS-001-01` から `-04`）は `ApiTokensTab.test.tsx` へ新しく書いた。見出しの階層、3 つの Base URL と用途、API の種類とリソースごとのまとまり、閉じた状態で始まるグループ、変更系の無いリソース、変更系だけのリソースの列の位置を観測する。
認証、発行と失効、粒度認可の 11 件は `backend/apitoken/handlers_http/scenario_examples_test.go` を新設し、`testing_stack` の実署名器と製品と同じ組み立てで観測した。認証の拒否は JWT と記録の一方の 1 か所だけを壊し、既定のままのトークンが通ることと対にしている。拒否の効果はトークンの記録とグループの保存先から読み直す。
`EX-APITOKENS-004-03` は、契約がすべての管理 operation に宣言を持つため製品の入口から再現できず、契約を差し替えた `admin_scope_test.go` の単体テストで観測した。
照合で分かった欠陥は 3 件で、利用者の判断によりすべて本項目で直した。発行、一覧、失効が `system_admin` を拒否していた（`decisions.md` と食い違い）。発行と失効が Origin と CSRF を検証していなかった。外部 IdP 接続の削除が連携の残りを見ず、PostgreSQL では 500、メモリでは連携を残したまま削除していた。削除の拒否は新しい規則 `REQ-AUTHENTICATION-037` として定め、409 `connection_in_use` を返し、管理画面は拒否を表示言語で示す。
ApiTokens の前提判定を改名したことで、`check-status-drift` が同名の定義を読まないために隠れていた、ApiTokens と外部 IdP 接続の 11 operation の契約のずれが表に出た。実装が正しいものは契約へ 401 と 403 の問題コードを宣言し、どの経路も返さない 400 を外してリリースベースラインへ反映した。

全文は `git show 854f521daf0113319b826044107f04189d1f24c1:<このファイルのパス>` で読む。
