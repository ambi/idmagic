---
status: completed
archived_from: 4635e55fbc6c09ae4eb2489ab59701fb581bc52e
---

# client_secret rotation を use case + endpoint + admin UI として導入する

Application editor から secret-based OAuth2 client の credential を rotation / 即時 revoke できるようにし、旧 credential の overlap と既存 hash の dual-read 移行を追加した。

全文は `git show 4635e55fbc6c09ae4eb2489ab59701fb581bc52e:<このファイルのパス>` で読む。
