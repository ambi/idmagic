---
status: completed
---

# リソースアクセスの評価で属性の条件を判定する

`ResourceAccessRule` に CEL の `condition`（`deny_when`、`require_approval_when`、`approval_detail_type`、読み取り専用の `version`）を追加し、REQ-OAUTH2-062 の論理積の関係の判定と承認の間に一段加えた。
評価の要求に `attributes`（16 KiB まで）を追加し、式の入力を `token`、`now`、`operation`、`attributes` に分けた。
`ResourceAccessEvaluated` に `conditionVersion`、`conditionOutcome`、`declaredAttributesDigest` を追加し、平文は残さない。
不許可の理由に `condition_denied` と `condition_indeterminate` を加え、承認要求は REQ-OAUTH2-065 の経路へ接続した。REQ-OAUTH2-064 は条件の保存の検証を、REQ-OAUTH2-065 は条件が求めた承認の扱いを追加した。

判断：`cel-go` は既存の依存である。版は内容から導き、管理者の上げ忘れで監査が偽にならないようにした。`authorization_detail_type` は常にトークンへ求めるので、条件の承認の明細は `approval_detail_type` に分けた。式は評価ごとにコンパイルし、実測は約 73 µs でキャッシュを足していない。評価不能の内訳は属性の値を含み得るので記録しない。

確認の限界：`attributes` はリソースサーバーの申告で、申告を偽るリソースサーバー自身は抑えられない。費用の実測は単体のベンチマークで、`CAP-RESOURCE-ACCESS-THROUGHPUT` の負荷測定は再実行していない。MCP ゲートウェイは `attributes` を送らないので、属性を読む条件はゲートウェイの配下で不許可になる（後続を `wi-31096` に起票した）。条件を使ったテナントでこの変更を戻すと条件が無視されて許可が広がるので、先に条件を外す。
