---
status: completed
archived_from: bc6c7999d6c0751413f0d1eef50653e1a1476e93
---

# saml コンテキストへバックエンド・コンテキストローカリティを横展開する

SAML 固有の業務型、永続化 adapter、DI を `backend/saml` へ局所化し、PostgreSQL
repository を4件すべて sqlc 生成へ置換した。外部コンテキストとの連携は Module または adapter
境界に維持し、HTTP route・DB schema・公開 API の振る舞いは変えていない。

全文は `git show bc6c7999d6c0751413f0d1eef50653e1a1476e93:<このファイルのパス>` で読む。
