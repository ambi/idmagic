---
depends_on: [wi-50-token-exchange-delegation-actor-chain, wi-51-rich-authorization-requests-agent-scopes]
status: completed
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
- **resource indicator の fail-closed 検証は Authorize / PushAuthorizationRequest /
  Token(authorization_code redemption) / Token(token-exchange) の 4 経路に限定する**
  (SCL `RFC8707-MCP-RESOURCE-BINDING` requirement, `adoption: partial`)。
  Token(client_credentials) / Token(refresh_token rotation) / DeviceAuthorization は
  resource パラメータを受理せず、常に client_id を audience とする既存動作のまま変更しない。
  M2M (client_credentials) の MCP resource-bound token、および resource 束縛済み
  access token を refresh 後も audience 保持したまま継続利用するニーズは follow-up
  work item で扱う。
- MCP resource server 管理 API のフロントエンド UI 画面 (本 WI の Scope は decision /
  scl / go / http のみで frontend 実装を含まない。管理は API 経由)。
- 実際の MCP SDK/クライアント/サーバーを用いた相互運用検証 (T006 の fixture 相当)。
  本 WI では HTTP レベルの手動検証 (DCR → PRM fail-closed → /authorize の
  resource fail-closed rejection、curl 経由) と自動テストで代替した。

## Plan
- [[ADR-055-mcp-authorization-server]] を正本に、既存 OAuth2 authorization server の RFC 8414 metadata、PKCE、PAR、Resource Indicators、RAR、token exchange を再利用する。MCP 専用 token issuer/consent store は作らない。
- MCP Resource Server を Application/service binding として登録し、canonical resource URI、allowed scopes、authorization_details schema、audience policy を所有させる。Protected Resource Metadata から authorization server を発見できるようにする。
- authorization request/token exchange の `resource` を必須 audience に変換し、access token の `aud` と scope/RAR を対象 MCP server に限定する。MCP server 側が issuer/audience を検証できる metadata/introspection contract を提供する。
- Dynamic Client Registration は既存 RegisterClient contract の policy-controlled profile とし、software statement/redirect URI/client type 制約を満たさない登録を拒否する。Enterprise-managed authorization policy は admin が resource/client/agent 単位で設定する。

## Tasks
- [x] T001 [ADR/SCL] ADR-055 を `accepted` に確定し、`spec/contexts/oauth2.yaml` に RFC 9728 standard、McpResourceServer/ResourceIndicator/ProtectedResourceMetadata/InvalidTargetError models、新規 events、interfaces (PRM + admin CRUD) を追加。`just yaml-check` green。
- [x] T002 [Domain] McpResourceServer entity (tenant-scoped, globally-unique resource_server_id, ADR-083 方針) — RED: `TestMcpResourceServerValidate_rejectsResourceWithFragment` 等を先に fail 確認 (`backend/oauth2/domain/mcp_resource_server_test.go`) → GREEN (`mcp_resource_server.go`)。canonical resource URI の tenant 内一意性は admin CRUD (T005) で強制。
- [x] T003 [Metadata] RFC 9728 Protected Resource Metadata — RED: `TestBuildProtectedResourceMetadata_unregistered_rejectedAsInvalidTarget` 等 (`usecases/protected_resource_metadata_test.go`) → GREEN (`usecases/protected_resource_metadata.go` + `adapters/http/protected_resource_metadata_handler.go`, `/.well-known/oauth-protected-resource?resource=`)。McpResourceServer 登録から導出し手書き独立管理しない (ADR-011 方針)。
- [x] T004 [OAuth2] resource indicator の共通検証 — RED: `TestResolveResourceIndicator_*` (`usecases/resource_indicator_test.go`) → GREEN (`usecases/resource_indicator.go`)。Authorize (`TestAuthorize_unregisteredResource_rejectedFailClosed`)・PushAuthorizationRequest (`TestPushAuthorizationRequest_unregisteredResource_rejectedFailClosed`)・Token(authorization_code redemption, `TestExchangeCodeForToken_tokenRequestResourceMismatch_rejectedAsInvalidTarget`)・Token(token-exchange, `TestExchangeTokenRejectsUnregisteredResource`/`RejectsDisabledResource`、既存 TOFU コメントを実 registry 検証へ置換) に配線。client_credentials / refresh_token rotation / device_code は対象外 (`## Out of Scope`、SCL `adoption: partial`)。
- [x] T005 [Registration/Admin] McpResourceServer 管理 CRUD — RED: `TestAdminCreatesAndListsMcpResourceServer` 等 (`adapters/http/admin_mcp_resource_server_handler_test.go`) → GREEN (`admin_mcp_resource_server_handler.go`, `/api/admin/mcp-resource-servers`, `TenantAdministrator` policy, prose label `AdminMcpResourcesManage`)。DCR (`RegisterClient`) は ADR-055 の決定どおり既存実装を無変更で再利用 (制約プロファイルの新規追加なし)。
- [x] T006 [Interop/Verify] MCP SDK 実機での相互運用検証は未実施 (`## Out of Scope` に開示)。代替として: 自動テスト一式 (domain/usecases/adapters 全層 TDD) に加え、`just dev-api` 相当のローカルサーバーへ curl で DCR → PRM (未登録 resource は invalid_target) → `/authorize` (未登録 resource は fail-closed、resource 省略時は既存フローに無影響) を手動確認済み。

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

## Completion
- **Completed At**: 2026-07-19
- **Summary**:
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
- **Scope narrowing (ADR-121 開示)**:
  以下は wi-56 の Motivation/Scope が示唆する範囲より狭く実装しており、SCL の
  `RFC8707-MCP-RESOURCE-BINDING` requirement で `adoption: partial` として構造化宣言済み:
  - resource indicator の fail-closed 検証は Authorize / PushAuthorizationRequest /
    Token(authorization_code redemption) / Token(token-exchange) の 4 経路に限定した。
    client_credentials・refresh_token rotation・device_code は resource パラメータを
    受理せず、常に client_id を audience とする既存動作のまま変更していない。
  - MCP resource server 管理 API のフロントエンド UI 画面は対象外 (API のみ)。
  - 実際の MCP SDK/クライアント/サーバーを用いた相互運用検証 (T006 の fixture 相当) は
    未実施。HTTP レベルの手動確認 (curl 経由の DCR → PRM → `/authorize` resource
    fail-closed rejection) と自動テスト一式で代替した。
- **Verification Results**:
  - `just yaml-check` - passed (SCL / work-item / ids / architecture cross-check /
    traceability すべて green)。
  - `just scl-render` - passed (derived artifacts 再生成)。
  - `just build-go` - passed。
  - `just lint-go` - passed (0 issues)。
  - `just test-go` (`go test ./...`) - wi-56 関連コード (domain/usecases/adapters/http 全層) は
    すべて green。無関係な既存失敗 `TestAssembledRoutesMatchGeneratedOpenAPI`
    (`GET /session/check` ルート契約の pre-existing な不一致) が本 WI 着手前の `main` に
    既に存在することを `git stash -u` で確認済み (regression ではない)。
  - 手動検証: `go run ./backend/cmd/idmagic` (memory backend) をローカル起動し、curl で
    (1) DCR (`POST /register`) によるクライアント動的登録、(2) PRM
    (`GET /.well-known/oauth-protected-resource`) が resource 未指定は `invalid_request`、
    未登録 resource は `invalid_target` を返すこと、(3) `GET /authorize` が未登録 resource
    指定時に認証前の時点で `invalid_target` を fail-closed 返却し、resource 省略時は
    既存のログインリダイレクトへ影響しないことを実地確認した。
- **Affected Guarantees State**: 既存の非 MCP OAuth2/OIDC クライアント (resource
  パラメータを指定しない全クライアント) の挙動・保証義務は無変更 (`AllAccessTokensCarryAudience`
  は従来どおり client_id を audience として満たす)。新規に追加した保証義務は「resource
  パラメータ指定時は登録済み Active な McpResourceServer への audience 厳格限定を
  fail-closed で強制する」(ADR-055 決定3) であり、対象は Authorize/PAR/
  Token(authorization_code, token-exchange) の 4 経路に限定される (上記 scope narrowing 参照)。
