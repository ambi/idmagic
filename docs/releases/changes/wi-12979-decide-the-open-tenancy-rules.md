# wi-12979-decide-the-open-tenancy-rules

Tenancy の規則に残していた未決定の点を決め、そのうち 3 点の振る舞いを改めた。

- すでに無効なテナントをもう一度無効化しても、`disabled_at` は最初に無効化した時刻のまま変わらない。これまでは二度目の要求の時刻で上書きしていた。204 と `TenantDisabled` の発行は変わらない（[`REQ-TENANCY-027`](../../domain/tenancy/lifecycle/README.md)）。
- フッターリンクのラベルの上限を、UTF-8 の 80 バイトから 80 文字に改めた。日本語のラベルも 80 文字まで保存できる。URL の上限も、バイト数ではなく 2,048 文字で数える（[`REQ-TENANCY-032`](../../domain/tenancy/branding/README.md)）。
- 負の上限を含むクォータの更新は、400 と `invalid_request` で拒否し、どの上書きも保存しない。作成を止めるには上限 0 を使う（[`REQ-TENANCY-037`](../../domain/tenancy/quota/README.md)）。
- 次の振る舞いは変えず、理由を各要件に記した。上限と使用量を読み取れないテナントは項目を省いて一覧に返す（[`REQ-TENANCY-026`](../../domain/tenancy/lifecycle/README.md)）。`TenantUpdated` の `changed_fields` には値が変わらない項目も載る（[`REQ-TENANCY-031`](../../domain/tenancy/settings/README.md)）。上書きがないテンプレートのリセットも `NotificationTemplateReset` を発行する（[`REQ-TENANCY-039`](../../domain/tenancy/notification-template/README.md)）。
