---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-06-22
---

# Model Context Protocol (MCP) の認可サーバーとして振る舞う

## Motivation
Model Context Protocol (MCP) は AI エージェントが外部ツール / データソース (MCP
サーバー) へ接続する事実上の標準になった。MCP の認可仕様は OAuth 2.1 を基盤とし、
リモート MCP サーバーは OAuth Resource Server として扱われる。2025-11 改訂以降、
インターネット公開の MCP サーバーは OAuth 2.1 + PKCE(S256) を必須とし、Protected
Resource Metadata (RFC 9728) で対応する認可サーバーを広告し、MCP クライアントは
Resource Indicators (RFC 8707) で audience を限定し、AS メタデータ (RFC 8414) と
Dynamic Client Registration (RFC 7591) で自動接続する。

idmagic は OAuth 2.1 相当 (PKCE 必須・DPoP・PAR)、Discovery、JWKS、DCR を既に
備えており、MCP の認可サーバーになる素地が揃っている。本 WI は MCP エコシステムの
認可サーバーとして、Protected Resource Metadata の発行、Resource Indicators による
audience 限定 ([[wi-50-token-exchange-delegation-actor-chain]] と整合)、MCP 向けの
discovery / DCR を整備し、エージェントと MCP クライアントが idmagic 経由でツール
サーバー用の audience 限定トークンを取得できるようにする。

## Scope
- **decision**:
  - 新規 ADR [[ADR-055]]: MCP authorization 仕様への準拠範囲 (対象改訂)、Protected Resource Metadata (RFC 9728) の提供方法、Resource Indicators (RFC 8707) による audience 限定の 必須化、DCR / DPoP / PKCE の MCP 文脈での既定、MCP サーバー (resource) の登録モデルを確定する。
- **scl**:
  - 新規 model: ProtectedResourceMetadata / McpResourceServer / ResourceIndicator。 AccessTokenClaims の audience を resource 単位に厳格化する。
  - 新規 event: ProtectedResourceMetadataServed / ResourceScopedTokenIssued / ResourceAudienceRejected。
  - 新規 interface: GetProtectedResourceMetadata、MCP resource server の管理 CRUD。permission AdminMcpResourcesManage。
- **go**:
  - RFC 9728 メタデータ生成、Resource Indicators による audience 限定発行、別 resource での token 拒否を fail-closed で実装する。既存 DCR / discovery / DPoP / PKCE を MCP 文脈へ接続する。
- **http**:
  - /.well-known/oauth-protected-resource (RFC 9728)、resource 限定トークン発行、MCP resource 管理 API。

## Out of Scope
- MCP サーバー (ツール側) 実装そのもの (idmagic は認可サーバー / metadata 提供側)。
- Cross-App Access / Enterprise-Managed Authorization 拡張 ([[wi-57-cross-app-access-identity-assertion-grant]])。
- MCP transport (stdio / HTTP streaming) の実装。

## Verification
- `just test-go`
  - reason: RFC 9728 メタデータ整合、resource indicator による audience 限定、別 resource 拒否、PKCE 強制の境界。
- `just lint-go`
- `just build-go`
- 手動: MCP クライアントが PRM を発見 → DCR → resource 指定で token 取得 → 別 resource では拒否されることを確認する。

## Risk Notes
既存の OAuth 2.1 / Discovery / DCR 資産を流用するため新規暗号要素は少ないが、
Resource Indicators による audience 限定を緩めると MCP ツール間で token が再利用され
権限越境を招く。audience は resource 単位に厳格化し、別 resource では fail-closed で拒否する。
仕様改訂の版差は ADR で対象改訂を固定する。
