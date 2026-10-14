---
status: completed
archived_from: 11f204182a9bb7673ca42ce4f2b4ae34e9e94a2a
---

# 内部生成された代理キーの列名を id に統一する

Scope 記載の 4 対象(`identity_provider_connections.provider_id`、
`oauth2_client_secrets.credential_id`、`tenant_branding_assets` / `application_icons` の
`object_key`、`application_categories.category_id`)すべてを `id` に統一した
(SCL → Postgres schema → Go domain/usecases/db_postgres/db_memory/handlers_http →
frontend)。
開示 (逸脱): Design は `identity_provider_connections.id` の列型を `UUID PRIMARY KEY` として
いたが、実装では `TEXT` のまま維持した。SCL (`authentication.yaml`) が該当 identity を
`type: String` と定義しており、ドリフトは列名のみで型は元々一致していたため。実運用では Go
側が UUID 文字列を生成するが、型制約としては課していない。
`applications.application_id` と `mcp_resource_servers.resource_server_id` は WI 記載どおり
Out of Scope のまま未着手。`oauth2_clients.client_id` / `webauthn_credentials.credential_id` /
`federated_response_replays.response_id` 等のプロトコル由来識別子、および `application.yaml`
側の `ApplicationClientSecretCredentialMetadata.credential_id` /
`RevokeApplicationClientSecret` の `credential_id` パスパラメータ(Scope 外、`oauth2.yaml` 側の
同名 admin API のみ `id` 化)は変更していない。
test-first からの逸脱 (self-attest): 本 WI は新規振る舞いを追加しない列名/フィールド名の
リネームであり、新しい RED テストは書いていない。代わりに、リネーム後に列名・フィールド名
不一致でコンパイルエラーとなった既存テストを追従させ、`identity_provider_connections` の
PK 単独化で顕在化した cross-tenant なテスト間 ID 衝突(`testConnection` ヘルパーの固定文字列
ID)というリグレッションを検出・修正した。

全文は `git show 11f204182a9bb7673ca42ce4f2b4ae34e9e94a2a:<このファイルのパス>` で読む。
