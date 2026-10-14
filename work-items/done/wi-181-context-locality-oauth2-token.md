---
status: completed
archived_from: a881a8c2b46c98c2b43b3502d6d987b01ad9a0ea
---

# oauth2 コンテキストの token/grant 系型・永続化をコンテキストローカリティ化する

token/grant domain 型、refresh token の sqlc-backed persistence、Valkey
adapter 窓口、OAuth2 Module の DI 集約を context 内へ配置した。authorize handler は
consent details と transaction/Cookie 操作を feature ファイルへ分離した。

全文は `git show a881a8c2b46c98c2b43b3502d6d987b01ad9a0ea:<このファイルのパス>` で読む。
