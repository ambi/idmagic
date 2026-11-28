---
status: completed
---

# 連携の再送の予約で、期限切れを検証に渡した時刻と比べる

`spec-diff` は main に対して規範の差分なしを示し、`spec_impact: none` のまま完了した。
`federationports.ReplayStore` と `negotiateports.ReplayStore` の `Reserve` を `(now time.Time, ttl time.Duration)` を受け取る形に変え、既存の予約の期限切れを実時計ではなく呼び出し側の検証の時刻で判定するようにした。
メモリ実装は `time.Now()`、PostgreSQL 実装は `DeleteExpiredReplays` の SQL の `now()` をやめ、ゼロ値の `now` と正でない `ttl` を予約せずにエラーにする。
期限の二つの時刻を並べる形ではなく `ttl` を渡す形にしたのは、型で取り違えを防ぎ、SAML の `AuthnRequestReplayStore.RecordIfNew` と同じ組にそろえるためである。
共有の `testing_contract` に、期限の直前と期限ちょうど、他テナント、無効な入力を確かめる表を足し、記録の時刻を使う Keycloak の再送テストが実行日によらず通るようになった。
ほかのストレージのアダプターが実時計で判定する箇所は wi-15471 に残る。
