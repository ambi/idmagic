---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-13
depends_on: [wi-56-mcp-authorization-server]
change_kind: feature
initial_context:
  scl:
    OAuth2:
      - standards.RFC9728.RFC9728-METADATA
      - standards.RFC9728.RFC9728-WELL-KNOWN
      - standards.RFC9728.RFC9728-IDMAGIC-API
      - interfaces.GetProtectedResourceMetadata
      - scenarios.失効した access_token でユーザー情報取得は invalid_token で拒否される
  source:
    - backend/oauth2/token/usecases/protected_resource_metadata.go
    - backend/oauth2/handlers_http/protected_resource_metadata_handler.go
    - backend/shared/http/support_http/auth.go
    - backend/shared/http/support_http/tenant_middleware.go
    - backend/shared/spec/runtime_contract.go
    - backend/sourcing/scim/handlers_http/handlers.go
    - backend/shared/http/server_http/routes.go
  tests:
    - backend/oauth2/handlers_http/bearer_resource_server_test.go
    - backend/shared/http/support_http/auth_test.go
    - backend/sourcing/scim/handlers_http/scim_test.go
  stop_before_reading:
    - frontend
affected_spec:
  - { path: spec/contexts/oauth2/SPECIFICATION.md, requirement: RFC9728-CHALLENGE }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.GetProtectedResourceMetadata }
  - { path: spec/contexts/oauth2/SPECIFICATION.md, requirement: REQ-OAUTH2-044 }
---

# 401 応答の WWW-Authenticate に resource_metadata を付し、保護リソースメタデータへ誘導する

## Motivation

RFC 9728 の Protected Resource Metadata (PRM) エンドポイントは
[[wi-56-mcp-authorization-server]] で実装済みで、`/.well-known/oauth-protected-resource` は
McpResourceServer ごとの metadata を配信できる。しかし **401 応答からその文書へ誘導する
`resource_metadata` チャレンジパラメータが存在しない**。リポジトリ全体を検索しても
`resource_metadata` は 0 件で、現状の `WWW-Authenticate` は
`Bearer error="invalid_token"` と `Bearer error="insufficient_scope", scope="..."` のみである
(`backend/shared/http/support_http/auth.go`)。

RFC 9728 §5.1 は、保護リソースが 401 を返す際に `resource_metadata` パラメータで自身の
metadata URL を示すことを定めており、**MCP クライアントはこの経路で認可サーバーを発見する**。
つまり配信側だけを実装して発見側を欠いた状態であり、「idmagic は MCP 認可サーバーである」と
いう主張が最後の 1 パラメータで未完になっている。手動設定なしに MCP クライアントが接続できる
という体験が、この差分だけで成立しない。

RFC 9728 の standards 表 (`spec/contexts/oauth2/SPECIFICATION.md`) にも
`RFC9728-METADATA` / `RFC9728-WELL-KNOWN` / `RFC9728-IDMAGIC-API` の 3 行はあるが、
チャレンジ側の要件行が無く、仕様側でも欠落している。

この項目は [[wi-369-agent-capability-survey-2026-08]] の棚卸しで P0 と判断した。

## Scope

- `spec/contexts/oauth2/SPECIFICATION.md` の RFC 9728 standards 表にチャレンジ要件を追加し、
  401 応答が `resource_metadata` を含むことの normative scenario (REQ-OAUTH2-044) を追加する。
- `backend/shared/http/support_http/auth.go` の `WriteAccessTokenError` が返す
  `WWW-Authenticate` に `resource_metadata="<PRM URL>"` を付す。
- 同型の bearer チャレンジを返す `backend/sourcing/scim/handlers_http/handlers.go` にも同じ扱いを適用する。
- PRM URL の endpoint path は TypeSpec の `GetProtectedResourceMetadata` から生成された runtime
  contract を唯一の出所とし、issuer は tenant middleware が解決した正規 location を使う。
  `protected_resource_metadata.go` は引き続き PRM 文書内容の唯一の出所とする。

## Out of Scope

- PRM 文書そのものの内容変更。[[wi-56-mcp-authorization-server]] で確定済みで、本 work item は
  そこへ誘導する経路だけを足す。
- `WWW-Authenticate` の `error_description` / `error_uri` パラメータ追加。RFC 6750 上は任意で、
  idmagic は Problem Details 本文で詳細を返す方針が既にあるため重複させない。
- MCP プロトコル本体 (JSON-RPC・tools) の実装。idmagic は認可サーバーであって MCP サーバーではない。

## Design

- **URL 生成を二重に持たない**。ハンドラ側で issuer とパスを組み立て直すと、realm prefix 付き
  (`/realms/:tenant_id`) とホストルートの 2 系統のマウントで片方だけ壊れる。tenant middleware の
  canonical issuer と TypeSpec-generated runtime contract の endpoint path を共通ヘルパで結合する。
- **resource が特定できる文脈ではその resource 用 PRM URL を指す**。特定できない場合は
  realm 自身の IdMagic API metadata URL (`resource` パラメータ無しの形) を指す。
  RFC 9728 はチャレンジで示す URL が当該リソースの metadata であることを求めるため、
  「常に realm のもの」で固定はしない。
- **`insufficient_scope` (403) にも付す**。RFC 9728 §5.1 は 401 を主対象とするが、
  スコープ不足でもクライアントは必要な scope を metadata から知る必要がある。既存の
  `scope` パラメータと併記する。
- チャレンジパラメータの順序と quoting は RFC 9110 の auth-param 形式に従う。

## Plan

- 先に spec (standards 行 + REQ-OAUTH2-044) を確定させ、そのあと実装する。
- `WriteAccessTokenError` は resource 文脈を持たないため、PRM URL の解決子を
  `Authenticator` 経由で注入するか、引数で受け取るかを実装時に決める。
  ハンドラ側に URL 文字列を直接持たせる形は採らない。
- SCIM 側は独自にチャレンジを書いているため、共通ヘルパへ寄せられるかを実装時に判断する。
  寄せられない場合も、生成する URL は同じ出所を通す。

## Tasks

- [x] T001 [Spec] RFC 9728 standards 表に RFC9728-CHALLENGE と REQ-OAUTH2-044 を追加し、`just check-spec` / `just check-api-compat` を GREEN にした。
- [x] T002 [Adapters] RED: `TestBearerInactiveTokenIsUnauthorized`、`TestBearerWithoutPortalScopeIsForbidden`、`TestBearerInactiveTokenChallengeUsesSubdomainMetadata` が旧 header で失敗することを確認（REQ-OAUTH2-044）→ TypeSpec-generated PRM endpoint と canonical tenant issuer を使う共通 challenge helper で 401 / 403 と path / host-root を GREEN。`TestAccountContextRejectsStaleBearerToken` の既存 challenge 契約も新しい完全値へ更新した。
- [x] T003 [Adapters] RED: `TestScimInboundProvisioning` が旧 SCIM 401 header で失敗することを確認（REQ-OAUTH2-044）→ SCIM も共通 challenge helper を使って GREEN。
- [x] T004 [Verify] `just spec-render`、狭い Go package tests、`just check-work-items`、`just verify-go`、`just verify` を GREEN にした。

## Verification

- `just check` / `just check-spec` / `just check-work-items`
- `just verify-go`
- 手動: `just dev` で (1) 無効な access token で保護 API を叩き 401 の `WWW-Authenticate` に
  `resource_metadata` が含まれること、(2) その URL を辿ると当該 resource の PRM 文書が返ること、
  (3) realm prefix 付きとホストルートの両方で正しい URL になることを確認する。

## Risk Notes

リスクは low。追加するのはレスポンスヘッダのパラメータ 1 つで、既存クライアントは未知の
auth-param を無視するため後方互換が壊れない。唯一の注意点は URL 生成で、realm prefix 付きと
ホストルートの 2 系統マウントのどちらかで誤った URL を返すと、MCP クライアントが誤った
認可サーバーへ誘導される。両系統をテストで固定する。

## Completion

- **Completed At**: 2026-08-14
- **Summary**:
  RFC 9728 の `RFC9728-CHALLENGE` と REQ-OAUTH2-044 を追加した。共通 Bearer challenge helper は
  tenant middleware の canonical issuer と TypeSpec-generated `GetProtectedResourceMetadata` path
  から `resource_metadata` URL を導出し、共通 API と SCIM の 401 `invalid_token` および 403
  `insufficient_scope` に quoted auth-param として付加する。
- **Semantic Difference**:
  `just spec-diff` は normative scenario `REQ-OAUTH2-044` の追加を報告した。従来の Bearer error
  challenge は error / scope だけを返していたが、現在は path-style realm と host-root realm の
  正規 PRM endpoint を discovery hint として返す。
- **Verification Results**:
  - `just check-spec` / `just check-api-compat` — passed
  - `just spec-render` — passed; OpenAPI の OAuth2 tag と `spec/generated/docs/index.html` を確認
  - `just test-go-package ./backend/oauth2/handlers_http` — passed
  - `just test-go-package ./backend/sourcing/scim/handlers_http` — passed
  - `just test-go-package ./backend/shared/http/support_http` — passed
  - `just test-go-package ./backend/shared/http/server_http` — passed
  - `just check-work-items` — passed
  - `just verify-go` — passed (golangci-lint 0 issues + race tests)
  - `just verify` — passed (specification, Go, UI, and tooling)
  - `git diff --check` — passed
- **Out of Scope / Left Undone**:
  - PRM 文書内容、`error_description` / `error_uri`、MCP JSON-RPC / tools は対象外のままである。
  - live dev server を使う手動確認は未実施。401 / 403、SCIM、path-style、host-root の各経路は
    上記の自動 HTTP テストで検証した。
