---
status: completed
archived_from: 854f521daf0113319b826044107f04189d1f24c1
---

# `state` が一致しないフェデレーションの callback が記録を残さない

`mise run spec-diff` は main に対する規範の差分を報告しない。宣言済みの `EX-AUTHENTICATION-001-03` へ実装を合わせた。
`CompleteLogin` は、`Attempts.Consume` が `ErrAttemptNotFound` または `ErrAttemptConsumed` で `state` を拒否したとき、
`Reason` が `state_mismatch`、`ProviderID` が空の `FederatedLoginRejected` を発行してから error を返す。
照合を上流の応答の検証より先に置く順序は変えていない。保存層の障害は照合の結果ではないので記録しない。
`Reason` の値は `federationdomain.RejectionStateMismatch` と、既存の値に名前を付けた
`RejectionProtocolValidationFailed` の 2 つの定数にした。
メモリ実装の `AttemptStore.Consume` は期限切れを domain の ad hoc な error で返していたので、Postgres 実装と同じ
`ErrAttemptConsumed` にそろえた。
HTTP のテストは `state` の不一致と上流の応答の検証失敗を両方扱うので
`TestMismatchedCallbackCreatesNothingAndRecordsTheRejection` へ改名し、ディレクティブで `EX-AUTHENTICATION-001-03` を
名指した。`tools/check/example-coverage-debt.json` の被覆台帳から当該行を外した。

全文は `git show 854f521daf0113319b826044107f04189d1f24c1:<このファイルのパス>` で読む。
