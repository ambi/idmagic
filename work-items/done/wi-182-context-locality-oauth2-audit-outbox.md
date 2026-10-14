---
status: completed
archived_from: f7e4de53c260a8378c473682e8056e5d1d8e0ff6
---

# oauth2 コンテキストの audit event / outbox 永続化をコンテキストローカリティ化する

audit event と outbox の memory/PostgreSQL adapter を oauth2 context に
同居させ、固定 PostgreSQL query を sqlc 生成へ移行した。DI は `oauth2.Module` が
`AuditEventRepo` と `EventSink` を所有する構成に完結した。

全文は `git show f7e4de53c260a8378c473682e8056e5d1d8e0ff6:<このファイルのパス>` で読む。
