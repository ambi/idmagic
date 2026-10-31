---
status: completed
---

# Agent とツールの単位で実行回数を制限する

`ResourceAccessRule` に `limits`（`by`、`max_requests`、`window_seconds`）を追加し、REQ-OAUTH2-062 の論理積の最後に実行回数の上限を加えた。
超過は 200、`decision=false`、理由 `limit_exceeded` とし、`ResourceAccessEvaluated` の `exceededLimitBy` と `exceededLimitWindowSeconds` にだけ超えた上限を残す。計数の障害と未配線は 503 で許可しない。
REQ-OAUTH2-064 は宣言の検証を、REQ-OAUTH2-065 は超過で使用権を記録しないことを追加した。

判断：流量と総回数は窓の長さの違いとして一つの型で表す。ツールは規則に当たり、宣言のないツールだけ `by` の `resource` で分ける。数える順序は宣言に依らず、狭い単位と短い窓を先にして共有の回数を守る。承認は照合、計数、使用権の記録の順に分け、超過で承認を失わせず、不一致で予算を減らさない。並行した別の鍵の敗者が減らした回数は戻さない。回数は評価で許可した数であり、業務が実行した数ではない。
保存先は `shared/ratelimit` の `AllowWithin` で、別々の組み立てが同じ PostgreSQL を共有する並行評価で上限が守られることを確かめた。メモリは単一プロセスに限る。長い窓で PostgreSQL の `int32` の計数が桁あふれして上限が解けるので、`MaxInt32` で止めた。

未対応：次に許可できる時刻の応答と管理画面の専用の入力欄は扱わない。実プロセスを分けた試験と本番規模の負荷は未検証である。巻き戻すと保存済みの `limits` を無視して上限が解けるので、先に運用の代替を確かめる。
