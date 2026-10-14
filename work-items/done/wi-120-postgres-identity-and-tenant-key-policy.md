---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# Postgres schema の user identity と tenant key 方針を明确化する

SCL Context `identity-management.yaml` で User の `identity` を `id` に定義変更しました。
また `deploy/schema/postgres.sql` を更新し、`users` テーブルの主キーを `id` とし、他テーブル（`consents`、`refresh_tokens`、`agents`、`scim_user_refs` など）からの外部キー制約を `(tenant_id, user_id) -> users(tenant_id, id)` に統一しました。
Go 実装側では旧 `sub` カラムへの参照を `user_id` および `UserID` フィールドに変更し、Kafka relay やテストコードの `sub` 参照も `userId` / `UserID` に合わせて更新しました。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
