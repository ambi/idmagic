# wi-33641-notify-quarantine-to-the-notification-email

プロビジョニング接続を隔離すると、接続の `notification_email` へメールで知らせるようになった（[`REQ-PROVISIONING-011`](../../modules/provisioning/synchronization/README.md)）。

- 連続失敗による隔離と、インクリメンタル同期の誤削除ガードによる隔離の両方で送る。これまではどちらも `ConnectionQuarantined` を発行するだけで、メールは送っていなかった。
- `notification_email` が未設定なら送らない。隔離に気付けるよう、宛先を設定しておく。
- 本文には、アプリケーション ID、隔離の理由、隔離した時刻（UTC）を載せる。言語はテナントのデフォルト言語で決まる。
- 通知テンプレートに `provisioning_connection_quarantined`（[`NotificationTemplateKey`](../../../spec/contexts/tenancy/models.tsp)）を追加した。管理画面の通知テンプレート一覧で文面を上書きでき、`application_id`、`quarantine_reason`、`quarantined_at` を差し込める。
- 送信に失敗しても隔離は保たれ、再送はしない。
