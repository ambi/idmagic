---
status: completed
archived_from: 0f76d6c089956fd2b893393377fb0f439872e026
---

# PostgreSQL の識別子と Application 子テーブルの tenant key を正規化する

Application の識別子を global UUID primary key に正規化し、icon、assignment、sign-in policy は
親 Application の foreign key のみで参照するようにした。リポジトリの tenant-scoped read / delete は
親 Application への join / exists guard を通し、冗長な child tenant key に依存しない fail-closed
境界へ変更した。RefreshToken の永続 tenant_id とその索引を除去し、EventLog の内部 primary key を
`id` に変更しながら、Kafka / EventDelivery の `event_id` 値は同一 UUID のまま維持した。Password
history は UUID primary key を DB 生成し、既存の `(user_id, created_at DESC, id DESC)` 順序を維持した。

全文は `git show 0f76d6c089956fd2b893393377fb0f439872e026:<このファイルのパス>` で読む。
