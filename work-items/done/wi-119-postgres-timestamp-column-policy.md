---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# Postgres schema の時刻列ポリシーを統一する

Postgres schema に timestamp policy コメントを追加し、mutable aggregate / tenant-scoped configuration の
`updated_at` を `NOT NULL DEFAULT now()` に統一した。`clients` と `scim_configs` には不足していた
`updated_at` を追加し、Go model / repository / usecase / bootstrap / HTTP response の契約を合わせた。
`consents`、`refresh_tokens`、`audit_events`、history / token / credential lifecycle 系は domain 固有時刻を
維持し、汎用 `updated_at` を追加しない方針を schema test で確認した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
