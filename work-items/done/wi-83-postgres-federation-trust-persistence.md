---
status: completed
archived_from: 0fca46fb861aad86ad8b000c8feebf5172954e84
---

# SAML と WS-Federation の trust 登録を PostgreSQL に永続化する

SAML SP / WS-Fed RP 用の PostgreSQL migration と repository adapter を追加し、
PERSISTENCE=postgres の bootstrap で memory fallback ではなく PostgreSQL adapter を
配線した。SCL の protocol 契約は変更せず、durable adapter の欠落だけを埋めた。

全文は `git show 0fca46fb861aad86ad8b000c8feebf5172954e84:<このファイルのパス>` で読む。
