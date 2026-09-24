# Provisioning の内部設計

## 同一トランザクションでのプロビジョニングタスクの記録

`Provisioning` はプロビジョニングタスクを捕捉する公開ポートを提供し、`IdManagement` のユーザー変更処理と `Application` の割り当て変更処理が、それぞれ自身の PostgreSQL トランザクション内でこのポートを呼び出す。ポートは一致する有効な接続ごとに `pending` の `ProvisioningTask` 行を 1 件挿入する。発火元の変更とプロビジョニングタスクの行は同時にコミットまたはロールバックされるため、`ProvisioningTask` がこの Context 専用の Transactional Outbox となる。プロビジョニングタスクの冪等性には `(tenant, connection, source_type, source_id, source_version)` をキーとして使う。

## 猶予期間つき削除の予約

`DeprovisionPolicy.grace_period_days` が 1 以上の接続では、`on_delete=delete` へ変換された User の削除から `ProvisioningTask` を作らず、`ScheduledDeprovision` を保存する。
予約は接続、User、削除イベントのバージョン、期限（削除時刻に猶予日数を足した時刻）を持つ。
プロビジョニングタスクの行を作らないため、worker が期限前に下流へ DELETE を送ることはない。

| 状態 | 意味 | 遷移の契機 |
| --- | --- | --- |
| `scheduled` | 期限の到来を待っている。同じ接続と User に一件だけ置き、再度の削除通知は期限を延ばさない | 猶予期間を持つ接続への User の削除 |
| `materialized` | 期限に達し、同じバージョンの delete の `ProvisioningTask` を作った | ディスパッチャーの周期処理が期限に達した予約を見つけた |
| `cancelled` | 猶予期間中に対象が適用範囲へ戻ったため取り消した | 同じ Application への User の割り当て。予約より新しいバージョンの割り当てだけが取り消す |

ディスパッチャーは、未関連付けのプロビジョニングタスクへジョブを関連付ける前に、期限に達した予約をプロビジョニングタスクへ変える。
予約を `materialized` にする更新とプロビジョニングタスクの挿入は一つの SQL 文で行い、更新は予約が `scheduled` のときだけ成立する。
このため、取消と実体化が競合しても、取り消した予約からプロビジョニングタスクが作られることはない。
実体化が失敗した予約は `scheduled` のまま残り、次の周期で再び処理される。

予約は削除時点の接続設定で判断する。
期限の到来時に接続の状態や `on_delete` を再評価せず、接続を削除した場合だけ外部キーの連鎖で予約も消える。
