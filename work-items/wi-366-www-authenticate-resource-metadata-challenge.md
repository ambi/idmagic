---
status: pending
authors: [tn]
risk: low
created_at: 2026-08-13
depends_on: [wi-56-mcp-authorization-server]
change_kind: feature
affected_spec:
  - { path: spec/contexts/oauth2/SPECIFICATION.md, requirement: REQ-OAUTH2-020 }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.Contract.GetProtectedResourceMetadata }
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
  401 応答が `resource_metadata` を含むことの normative scenario (REQ-OAUTH2-041) を追加する。
- `backend/shared/http/support_http/auth.go` の `WriteAccessTokenError` が返す
  `WWW-Authenticate` に `resource_metadata="<PRM URL>"` を付す。
- 同型の bearer チャレンジを返す `backend/sourcing/scim/handlers_http/handlers.go` にも同じ扱いを適用する。
- PRM URL の生成は既存の `backend/oauth2/token/usecases/protected_resource_metadata.go` を
  唯一の出所として再利用する。

## Out of Scope

- PRM 文書そのものの内容変更。[[wi-56-mcp-authorization-server]] で確定済みで、本 work item は
  そこへ誘導する経路だけを足す。
- `WWW-Authenticate` の `error_description` / `error_uri` パラメータ追加。RFC 6750 上は任意で、
  idmagic は Problem Details 本文で詳細を返す方針が既にあるため重複させない。
- MCP プロトコル本体 (JSON-RPC・tools) の実装。idmagic は認可サーバーであって MCP サーバーではない。

## Design

- **URL 生成を二重に持たない**。ハンドラ側で issuer とパスを組み立て直すと、realm prefix 付き
  (`/realms/:tenant_id`) とホストルートの 2 系統のマウントで片方だけ壊れる。既存の
  `protected_resource_metadata.go` が持つ URL 決定ロジックを唯一の出所として参照する。
- **resource が特定できる文脈ではその resource 用 PRM URL を指す**。特定できない場合は
  realm 自身の IdMagic API metadata URL (`resource` パラメータ無しの形) を指す。
  RFC 9728 はチャレンジで示す URL が当該リソースの metadata であることを求めるため、
  「常に realm のもの」で固定はしない。
- **`insufficient_scope` (403) にも付す**。RFC 9728 §5.1 は 401 を主対象とするが、
  スコープ不足でもクライアントは必要な scope を metadata から知る必要がある。既存の
  `scope` パラメータと併記する。
- チャレンジパラメータの順序と quoting は RFC 9110 の auth-param 形式に従う。

## Plan

- 先に spec (standards 行 + REQ-OAUTH2-041) を確定させ、そのあと実装する。
- `WriteAccessTokenError` は resource 文脈を持たないため、PRM URL の解決子を
  `Authenticator` 経由で注入するか、引数で受け取るかを実装時に決める。
  ハンドラ側に URL 文字列を直接持たせる形は採らない。
- SCIM 側は独自にチャレンジを書いているため、共通ヘルパへ寄せられるかを実装時に判断する。
  寄せられない場合も、生成する URL は同じ出所を通す。

## Tasks

- [ ] T001 [Spec] RFC 9728 standards 表にチャレンジ要件行を追加し、REQ-OAUTH2-041 を追加する。
- [ ] T002 [Adapters] `WriteAccessTokenError` の 401 / 403 チャレンジに `resource_metadata` を付す。
      RED: チャレンジに `resource_metadata` が含まれることを確認するテストを先に書く → GREEN。
- [ ] T003 [Adapters] SCIM の bearer チャレンジにも同じ扱いを適用し、URL 生成の出所を統一する。
      RED: SCIM の 401 でも `resource_metadata` が返るテスト → GREEN。
- [ ] T004 [Verify] 下記 Verification を緑にする。`just spec-render` を実行する。

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
