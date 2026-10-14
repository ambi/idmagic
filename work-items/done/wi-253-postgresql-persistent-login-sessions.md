---
status: completed
archived_from: a6fa2de1d15b93d5894e0f81e044eb9183810bac
---

# ログイン済みセッションを PostgreSQL の単一正本として永続化する

ログイン済み LoginSession の単一正本を Valkey から PostgreSQL `authentication_sessions`
へ移した (ADR-126)。session id は tenant_id を含めた fail-closed な indexed lookup
(`authentication_sessions_pkey`) で解決し、失効は物理削除ではなく `revoked_at` /
`revoke_reason` の idempotent tombstone とした。`SessionStore` port を
`Find`(active-only) / `FindOwned`(所有者確認) / `Revoke`(idempotent) / `Touch`(粗粒度
last_seen_at) へ再設計し、`SessionManager.Resolve` から一括で touch することで
oauth2/idmanagement/frontend への配線を増やさずに書き込み量を抑えた。memory /
postgres 両 adapter を同じ contract に同期し、Valkey の login session 保存
(`SCAN session:*` を含む) は撤去した。production bootstrap
(`postgres_valkey.go`) を PostgreSQL 実装に切替済み。

全文は `git show a6fa2de1d15b93d5894e0f81e044eb9183810bac:<このファイルのパス>` で読む。
