# wi-96960-defer-and-cancel-user-deprovisioning-after-grace-period

Provisioning が、接続の `DeprovisionPolicy.grace_period_days` に従って User の削除を下流へ遅らせて届けるようになった（[`REQ-PROVISIONING-006`](../../domain/provisioning/scenarios.feature.md)）。

- `on_delete=delete` で `grace_period_days` が 1 以上の接続では、User を削除しても猶予期間が経つまで下流へ DELETE を送らない。これまでは設定値にかかわらず、削除の直後に DELETE を送っていた。
- 猶予期間が経つと worker が delete の配信を作り、下流へ DELETE を送って `UserDeprovisioned`（`action=delete`）を記録する。
- 猶予期間中に User を同じ Application へ再び割り当てると、予約していた削除は取り消され、下流の User は残る。ほかの Application の予約は取り消さない。
- `grace_period_days` が 0 の接続、および `on_delete` が `deactivate` または `none` の接続の振る舞いは変わらない。
- 猶予期間中に User を復元しても予約は取り消されない。削除を取りやめる場合は、Application へ再び割り当てる。
