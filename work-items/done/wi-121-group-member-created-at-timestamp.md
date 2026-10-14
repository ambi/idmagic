---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# GroupMember の所属追加時刻を created_at に統合する

GroupMember の所属追加時刻を `created_at` に統合し、`added_at` を SCL model、Postgres schema、
Go model/repository/usecase、SCIM 同期経路、admin HTTP response、UI 型から削除した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
