# wi-93929-notify-restores-and-reset-verification-on-email-change

管理者が User を復元したとき、下流のプロビジョニングへ通知するようになった（[`REQ-IDMANAGEMENT-049`](../../modules/identity-management/user/lifecycle.md)、[`REQ-PROVISIONING-006`](../../modules/provisioning/synchronization/README.md)）。

- 復元は User の再有効化として下流へ届く。削除の予約で下流を無効化した接続では有効に戻し、下流から削除した接続では作り直す。これまでは復元しても下流へ何も送らず、User は下流で無効または削除されたまま残っていた。
- 猶予期間つきの削除（`grace_period_days` が 1 以上）の接続では、猶予期間中に User を復元すると、すべての接続で削除の予約を取り消す。これまでは予約が残り、期限に下流の User を削除していた。

管理者が User のメールアドレスを変えたとき、`email_verified` を `false` にするようになった（[`REQ-IDMANAGEMENT-045`](../../modules/identity-management/user/README.md)）。

- 同じ要求で `email_verified` を指定した場合は、指定した値を保存する。
- `email_verified` が変わった場合は、`UserUpdated` の `changed_fields` に `email_verified` が載る。
- CSV の取り込みと SCIM の受信によるメールアドレスの変更の振る舞いは変わらない。
