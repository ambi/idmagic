# wi-61629-quarantine-connections-on-the-accidental-deletion-guard

接続の `DeprovisionPolicy` に設定した誤削除ガードの閾値が、定期の照合で実際に効くようになった（[`REQ-PROVISIONING-011`](../../domain/provisioning/scenarios.feature.md)）。

- 1 回の照合で計画した無効化と削除（猶予期間つき削除の予約を含む）が `accidental_deletion_count_threshold` を超えるか、下流へ反映済みの User に対する割合で `accidental_deletion_percent_threshold` を超えると、照合はその回に何も作らず、接続を隔離して `ConnectionQuarantined` を発行する。これまでは閾値を保存するだけで、どの経路も読んでいなかった。
- 閾値と等しい件数は超過ではなく、通常どおり無効化と削除を作る。
- 隔離を解除すると、次の照合があるべき状態から作り直す。原因（スコープや方針の設定の誤りなど）を正してから解除する。
- 管理者が起動する Full Resync と、割り当ての解除や User の削除の直後に作られる無効化と削除には、ガードは効かない。
- 隔離は `notification_email` へまだ通知されない。接続の `health` と `ConnectionQuarantined` で確認する。
