---
status: pending
authors: [claude]
risk: medium
created_at: 2026-08-08
depends_on: [wi-56-mcp-authorization-server]
---

# OAuth Client ID Metadata Documents (CIMD) を DCR の代替クライアント登録経路として実装する

## Motivation

[[wi-322-mcp-authorization-spec-repin-2026-07-audit]] の差分監査で判明した積み残し。MCP
authorization 仕様 2026-07-28 改訂は、クライアント登録の優先順位を
「pre-registration → **Client ID Metadata Documents (CIMD)** → Dynamic Client Registration
(RFC 7591、fallback 用) → 手動入力」に変更した。DCR は明示的に **deprecated** 扱いで、
「CIMD 未対応の authorization server との後方互換のためだけに残す」位置づけになっている。

idmagic は [[ADR-055]] のもと RFC 7591 DCR のみで MCP クライアントの自動オンボーディングを
提供しており([[ADR-011]] の Discovery 方針に沿う)、CIMD には未対応。CIMD 自体は
`client_id` を client がホストする HTTPS URL にし、authorization server がその URL を
fetch してメタデータ(`client_id` / `client_name` / `redirect_uris` 等)を検証する方式で、
事前関係のない client/server 間の自動オンボーディングという MCP の中心的なユースケースに
DCR より適している(登録リクエストという往復が不要)。

根拠仕様は IETF `draft-ietf-oauth-client-id-metadata-document-**00**`(2026年8月時点でごく
初期のドラフト、RFC 化されていない)。[[ADR-055]] が MCP 認可仕様自体について「版差が
大きいため対象改訂を固定する」と同じ理由で、ここも新しい ADR で対象 draft のバージョンと
追従方針を明示的に固定すべき規模の決定である。また、client が指定する任意の HTTPS URL を
authorization server が fetch するという新しい外部入力面が加わるため、SSRF 対策の設計判断が
本 wi の中心的な論点になる。

## Scope

- **decision**: 新規 ADR。以下を確定する。
  - 対象 draft のバージョン pin(`-00` のまま追従するか、内容が固まるまで待つか)と、
    draft 改訂時の再ピン留め方針([[ADR-055]] と同様の運用にするか)。
  - CIMD で解決した client 情報を `Client` として永続化するか、都度解決(request-scoped、
    非永続)にするかのデータモデル。
  - fetch 時の SSRF 対策(private / loopback / link-local IP 拒否、DNS rebinding 対策、
    リダイレクト制限、タイムアウト、レスポンスサイズ上限)の具体的な境界。
  - 既存 RFC 7591 DCR との共存方針(両方式を並行提供し、`client_id` の形式で分岐)。
  - `redirect_uri` 検証・キャッシュ方針(HTTP cache header 準拠 vs 上限 TTL)。
  - CIMD 由来 client の client 認証方式(`token_endpoint_auth_method: none` を許容するか、
    `private_key_jwt` を要求するか)。
- **scl**: `spec/contexts/oauth2.yaml` に以下を追加。
  - standards: `draft-ietf-oauth-client-id-metadata-document` の requirements 一覧。
  - AS Metadata (RFC 8414) response へ `client_id_metadata_document_supported` フィールドを追加。
  - Authorize / PAR / Token の `client_id` 解釈に CIMD 解決分岐を追加(既存 `RegisterClient`
    interface とは別の、実行時解決パス)。
  - 新規イベント(例: `ClientIdMetadataDocumentResolved` / `ClientIdMetadataDocumentFetchRejected`)、
    新規エラー(メタデータ取得失敗・JSON 形式不正・`client_id` 不一致・`redirect_uri` 不一致は
    いずれも fail-closed で `invalid_client` 系)。
- **go**: HTTPS-only の SSRF-safe fetcher、キャッシュ、`/authorize`・`/par`・`/token` の
  client 解決ロジックへの統合。
- **http**: 新規エンドポイントは追加しない。既存 discovery 応答
  (`/.well-known/oauth-authorization-server`)に `client_id_metadata_document_supported` を追加。

## Out of Scope

- 実際の MCP クライアント実装(Claude Desktop 等)との相互運用テスト。
- RFC 7591 DCR の廃止・削除(MCP 仕様どおり後方互換として残す。[[ADR-055]] の決定 5 は変更しない)。
- Pre-registration(静的 client 登録)方式自体の変更。
- `application_type` (OIDC DCR) 対応 — [[wi-322-mcp-authorization-spec-repin-2026-07-audit]] で
  影響なしと判断済みの別項目であり、本 wi のスコープ外。

## Design

未定。上記 Scope の decision 項目(特に SSRF 対策の境界と、永続化するかどうかのデータモデル)を
新規 ADR で先に確定してから SCL に落とす。

## Plan

未定(ADR の結論次第で分岐する)。最低限のステップ:
1. draft の内容と MCP 実装(参考実装があれば)を精読し、新規 ADR を起票する。
2. ADR の決定に沿って `spec/scl.yaml` / `spec/contexts/oauth2.yaml` を更新する。
3. SSRF-safe fetcher を test-first で実装し(悪意ある URL・private IP・リダイレクトループ・
   巨大レスポンスを拒否するケースを先に red で書く)、Authorize/PAR/Token の client 解決に統合する。
4. Discovery 応答へ `client_id_metadata_document_supported` を追加する。

## Tasks

- [ ] T001 [ADR] 新規 ADR を起票し、draft pin・永続化モデル・SSRF 対策境界・DCR 共存方針を確定する。
- [ ] T002 [SCL] ADR の決定に沿って `spec/contexts/oauth2.yaml` を更新し `just check-scl` を通す。
- [ ] T003 [Go/Adapters] SSRF-safe fetcher を test-first (RED→GREEN) で実装する。
- [ ] T004 [Go/Adapters] Authorize/PAR/Token の client_id 解決へ CIMD 分岐を test-first で統合する。
- [ ] T005 [Adapters] Discovery 応答に `client_id_metadata_document_supported` を追加する。
- [ ] T006 [Verify] `just verify` を通す。

## Verification

- `just test-go` / `just lint-go` / `just build-go`
- SSRF 対策のケース(private/loopback IP、リダイレクトループ、巨大レスポンス、タイムアウト)を
  fail-closed で拒否することを test-go で確認する。
- `client_id` と fetch したメタデータ内 `client_id` の不一致、`redirect_uri` の不一致を拒否する
  ことを確認する。

## Risk Notes

リスクは medium。最大の懸念は **SSRF**: client が任意の HTTPS URL を `client_id` として提示し、
authorization server がそれを fetch する新しい外部入力面が加わる。private/loopback IP・
DNS rebinding・リダイレクト・応答サイズを fail-closed で制限しないと、内部ネットワークへの
到達性探索やリソース枯渇に使われうる。次点のリスクは仕様側: 根拠 draft がまだ `-00` の
ごく初期段階でありRFC化されていないため、[[ADR-055]] と同様に対象改訂を明示的に固定し、
draft が動いた場合の追従方針を決めておく必要がある。
