---
status: completed
---

# 人間が承認した操作を実行時の対象と引数に結び付ける

規則の `requires_approved_operation` を持つ操作を、CIBA で人が承認した明細と同じ対象と引数でだけ許可し、承認の使用権を `operation_key` 一つに限る REQ-OAUTH2-065 を追加した。
`POST /resource-access/evaluate` に `operation_key`、`operation_parameters`、応答の `approved_operation` を、`ResourceAccessRule` に `requires_approved_operation` を、`ResourceAccessEvaluated` に相関の項目を足した（REQ-OAUTH2-062、REQ-OAUTH2-064）。
CIBA のトークンは承認の識別子を持ち、明細の `locations` が登録済みの `Active` なリソースを指せば `aud` をそのリソースにする（REQ-OAUTH2-041）。
`supervised` を強制しないとする旧記述は、REQ-OAUTH2-050 に合わせて直した。

短い予約と確定は、タイムアウト後の再試行で二重に実行され得るので採らず、同じ鍵の再送だけを許して重複の排除を資源の冪等性へ委ねた。
IdMagic は実行の結果を持たず、資源は応答の `approval_request_id` と `operation_digest` を自身の結果とともに記録する。
承認の識別子は交換の派生へ継承せず、一般のトークンの再利用可能性は変えていない。

テストは実装の後に書いたので、RED は故障注入（鍵の比較、引数の比較、`aud`、承認の識別子、ストアの配線）で観測した。

残るリスクは次のとおりである。
冪等性のない資源では一回性は成り立たない。
承認の `expires_at` の後は同じ鍵の再送も拒否するので、資源は先に自身の記録を読む必要がある。
UNLOGGED の表の消失では、使用中の承認も不許可になる。
`operation_digest` は引数の平文の推測に耐えるシークレットではなく、`fields` にシークレットを入れない前提である。
管理 UI の規則は JSON の入力のままで、`requires_approved_operation` の専用の入力はない。
