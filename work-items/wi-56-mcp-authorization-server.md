---
depends_on: [wi-50-token-exchange-delegation-actor-chain, wi-51-rich-authorization-requests-agent-scopes]
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

## Plan
- [[ADR-055-mcp-authorization-server]] を正本に、既存 OAuth2 authorization server の RFC 8414 metadata、PKCE、PAR、Resource Indicators、RAR、token exchange を再利用する。MCP 専用 token issuer/consent store は作らない。
- MCP Resource Server を Application/service binding として登録し、canonical resource URI、allowed scopes、authorization_details schema、audience policy を所有させる。Protected Resource Metadata から authorization server を発見できるようにする。
- authorization request/token exchange の `resource` を必須 audience に変換し、access token の `aud` と scope/RAR を対象 MCP server に限定する。MCP server 側が issuer/audience を検証できる metadata/introspection contract を提供する。
- Dynamic Client Registration は既存 RegisterClient contract の policy-controlled profile とし、software statement/redirect URI/client type 制約を満たさない登録を拒否する。Enterprise-managed authorization policy は admin が resource/client/agent 単位で設定する。

## Tasks
- [ ] T001 [ADR/SCL] ADR-055 を現行 OAuth2 interfacesへ確定し、MCP resource registration、metadata、resource-bound authorization/consent/policy scenarios を追加して再生成する。
- [ ] T002 [Application] MCP resource binding と canonical resource URI uniqueness、allowed scope/RAR schema、enterprise policy を実装する。
- [ ] T003 [Metadata] RFC 9728 Protected Resource Metadata と既存 RFC 8414/OIDC metadata の MCP 必須項目を tenant prefix 付きで公開する。
- [ ] T004 [OAuth2] authorize/PAR/token/token-exchange で resource indicator を検証し、audience-bound token と consent を発行する。
- [ ] T005 [Registration/Admin] constrained dynamic registration と MCP resource/client policy 管理 API を実装する。
- [ ] T006 [Interop/Verify] MCP SDK/client/server fixture で discovery→PKCE→token→resource access を通し、wrong resource/audience、scope escalation、public-client secret、tenant 混同を検証する。

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
