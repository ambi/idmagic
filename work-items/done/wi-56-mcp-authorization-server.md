---
status: completed
archived_from: 05db9a31be1bac966b00a5c404f6e93a18f88173
---

# Model Context Protocol (MCP) の認可サーバーとして振る舞う

ADR-055 を `suggested` → `accepted` に確定し、`spec/contexts/oauth2.yaml` へ RFC 9728
standard、`McpResourceServer`/`ResourceIndicator`/`ProtectedResourceMetadata`/
`InvalidTargetError` models、新規 events (`ProtectedResourceMetadataServed`/
`ResourceScopedTokenIssued`/`ResourceAudienceRejected`)、interfaces
(`GetProtectedResourceMetadata` + McpResourceServer admin CRUD 5 本、既存
`Authorize`/`PushAuthorizationRequest`/`Token` への resource 関連 requires/emits/errors)
を追加した。Go 側は Domain (`McpResourceServer` + state enum)、Ports、Memory/Postgres
永続化 (`mcp_resource_servers` テーブル新設、sqlc 再生成)、共通 resource indicator 検証
(`ResolveResourceIndicator`)、RFC 9728 PRM builder + `/.well-known/oauth-protected-resource`
handler、McpResourceServer admin CRUD (`/api/admin/mcp-resource-servers`)、
Authorize・PushAuthorizationRequest・Token(authorization_code redemption)・
Token(token-exchange、既存 TOFU コメントを実 registry 検証へ置換) への resource indicator
配線、bootstrap (memory/postgres_valkey) と central routes.go への DI 配線を全層
test-first (RED 確認 → GREEN) で実装した。

全文は `git show 05db9a31be1bac966b00a5c404f6e93a18f88173:<このファイルのパス>` で読む。
