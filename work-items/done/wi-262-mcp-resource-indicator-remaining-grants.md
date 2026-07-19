---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-19
depends_on: [wi-56-mcp-authorization-server]
---

# MCP resource indicator を client_credentials / refresh rotation へ拡張する

## Motivation
[[wi-56-mcp-authorization-server]] は RFC 8707 resource indicator による audience
限定を Authorize / PushAuthorizationRequest / Token(authorization_code redemption) /
Token(token-exchange) の 4 経路に限定して実装し、SCL の `RFC8707-MCP-RESOURCE-BINDING`
requirement を `adoption: partial` として、client_credentials・refresh token
rotation・device_code を対象外と開示した。

ユーザーからの質問を受けて棚卸しした結果:

- **client_credentials** は「人間の代理ではなくサービス自身が principal」となる
  正当な MCP M2M アクセスパターンであり、対応しないままだと同一 client の
  client_credentials トークンがどの MCP resource でも使い回せてしまい、
  Resource Indicators が守ろうとしている audience 分離が M2M では機能しない。
  実装コストも最小（`token_handler.go` の既存分岐に数行足すだけ）。
- **refresh token rotation** は「意図的な除外」というより実装漏れに近い。
  authorization_code + `offline_access` + `resource` 指定の初回アクセストークンは
  正しく resource 限定されるが、リフレッシュ後のトークンは `Audiences` が
  渡されず client_id audience に戻ってしまう。MCP resource server 側が
  自分の resource URI を aud として期待していれば、リフレッシュ後のトークンが
  拒否されるようになる — 権限が広がる方向ではないが、正しさの欠落である。
- **device_code** は MCP 認可仕様が要求する利用パターンではない（ブラウザを
  持たない入力制約デバイス向けのグラントであり、MCP クライアントは
  フル機能アプリ/ブラウザを前提とする）。本 WI でも対象外のままとする。

`McpResourceServer` 管理画面のフロントエンド UI は本 WI とは無関係の関心事のため
[[wi-263-mcp-resource-server-admin-ui]] へ切り出した（本 WI では実装しない）。

## Scope
- `spec/contexts/oauth2.yaml`:
  - `standards.RFC8707` の `RFC8707-MCP-RESOURCE-BINDING` requirement の `reason` を、
    client_credentials・authorization_code refresh rotation も対象に含める内容へ更新
    （残る対象外は device_code のみ）。
  - `models.RefreshTokenRecord` に `resource: type: ResourceIndicator (optional)` を追加
    （`AuthorizationCodeRecord`/`AuthorizationRequest` に倣う）。
  - `interfaces.Token` の `requires` に ClientCredentials 向け resource 検証、
    `ensures` に RefreshToken rotation の resource 束縛保持を追加。

## Out of Scope
- device_code グラントへの resource indicator 対応（MCP 認可仕様が要求しないため）。
- MCP SDK 実機での相互運用検証（wi-56 と同様、HTTP レベルの手動確認と自動テストで代替）。
- `McpResourceServer` 管理画面のフロントエンド UI ([[wi-263-mcp-resource-server-admin-ui]] へ切り出し)。

## Plan
承認済みプラン `/Users/tn/.claude/plans/wi-56-2-1-2-precious-walrus.md` の Phase A〜C を正本とする
（Phase D フロントエンドは対象外、[[wi-263-mcp-resource-server-admin-ui]] を参照）。要旨:

- **Phase A (SCL)**: 上記 Scope の SCL 変更を先に行い `just yaml-check` → `just scl-render`。
- **Phase B (client_credentials)**: `token_handler.go` の client_credentials 分岐で
  `usecases.ResolveResourceIndicator` を呼び、`Audiences` を `SignAccessToken` へ渡す。
  この分岐は既存の他の拒否パス (`unauthorized_client`/`invalid_scope`) がイベント発火を
  伴わないため、resource 拒否時も同様にイベント発火なしで `writeOAuthError` のみとする。
- **Phase C (refresh rotation)**: `domain.RefreshTokenRecord.Resource` を追加し、
  `GenerateInitialRefreshToken`/`RotateRefreshToken` に resource を通す。
  `exchange_code.go` は実際の resource を、`device_flow.go` は `nil`（sid と同じ扱い）を渡す。
  `refresh_tokens.go` はローテーション時に `record.Resource` から `Audiences` を組み立てて
  `SignAccessToken` へ渡す。Postgres `refresh_tokens` テーブルに `resource TEXT` (nullable)
  列を追加し、`sqlc-generate` で再生成する。

## Tasks
- [x] T001 [SCL] `RFC8707-MCP-RESOURCE-BINDING` の reason 更新（client_credentials・
      refresh rotation を対象に追加、device_code のみ残存）、`RefreshTokenRecord.resource`
      追加、`Token` interface の requires（ClientCredentials 向け resource 検証）/
      ensures（`rotated_refresh_preserves_resource_binding`）追加。`just yaml-check` /
      `just scl-render` green。
- [x] T002 [OAuth2/client_credentials] `token_handler.go` の client_credentials 分岐に
      `usecases.ResolveResourceIndicator` を配線。RED: `TestTokenClientCredentials_unregisteredResource_rejectedAsInvalidTarget`
      を先に 200 (誤って発行) で fail 確認 → GREEN
      (`backend/oauth2/adapters/http/client_credentials_resource_indicator_test.go`)。
      未登録 resource は invalid_target、登録済み Active な resource は introspection で
      aud 限定を確認、resource 未指定は既存動作 (aud=client_id) を維持することを検証。
- [x] T003 [OAuth2/refresh] `RefreshTokenRecord.Resource` を追加し
      `GenerateInitialRefreshToken`/`RotateRefreshToken`/`exchange_code.go`(実 resource を渡す)/
      `device_flow.go`(nil を渡す、device_code は対象外のまま)/`refresh_tokens.go`
      (ローテーション時に `Audiences` を組み立てる) へ配線。RED:
      `TestRotateRefreshToken`(resource 引き継ぎ検証を追加)、
      `TestRefreshTokens_preservesResourceBindingAcrossRotation`、
      `TestExchangeCodeForToken_resourceBoundAtAuthorize_propagatesToRefreshTokenRecord`
      を先に fail 確認 → GREEN。Postgres `refresh_tokens.resource` 列を追加し
      `sqlc-generate` で再生成、新規 round-trip test
      `TestRefreshTokenStoreRoundTrip_PreservesResource`
      (初回発行 + rotation 後の両方で resource が保持されることを確認) を追加。
      memory 永続化は既存の Sid と同じ浅いコピーパターンで無変更のまま機能することを確認。
- [x] T004 [Verify] `just yaml-check` / `just build-go` / `just lint-go`(0 issues) /
      `just test-go`(全 green) を確認。手動: ローカルサーバーへ curl で DCR →
      client_credentials + 未登録 resource で invalid_target の fail-closed 拒否、
      resource 未指定時は既存動作 (200 OK) が無変更であることを実地確認。

## Verification
- `just yaml-check`
  - reason: SCL 変更（RFC8707 reason、RefreshTokenRecord.resource、Token requires/ensures）の整合。
- `just build-go` / `just lint-go` / `just test-go`
  - reason: client_credentials・refresh rotation の resource 束縛と fail-closed 拒否、
    Postgres 永続化の往復。
- 手動: `client_credentials` グラントに `resource` を指定してトークンを取得し `aud` が
  resource に限定されることを curl で確認する。authorization_code + `offline_access` +
  `resource` で取得した refresh token をローテーションし、リフレッシュ後のアクセストークンも
  同じ `aud` を保持することを確認する。

## Risk Notes
`refresh_tokens` テーブルへの列追加は加法的（nullable、default なし）で既存行に影響しない。
resource 限定の対象拡大は fail-closed 方向の変更であり、resource パラメータを指定しない
既存クライアントの挙動は無変更（`ResolveResourceIndicator` は resource 未指定時 `(nil, nil)`
を返す）。

## Completion
- **Completed At**: 2026-07-19
- **Summary**:
  RFC 8707 resource indicator の audience 限定を client_credentials グラントと
  authorization_code の refresh token rotation へ拡張した。SCL の
  `RFC8707-MCP-RESOURCE-BINDING` requirement を更新し、対象外は device_code のみに
  縮小した（wi-56 時点では client_credentials・refresh rotation・device_code の
  3 経路が対象外だった）。`token_handler.go` の client_credentials 分岐に
  `usecases.ResolveResourceIndicator` を配線し、未登録・Disabled な resource は
  fail-closed で拒否、登録済み Active な resource は access token の aud をそこに
  限定する。`domain.RefreshTokenRecord` に `resource` フィールドを追加し、
  `GenerateInitialRefreshToken`/`RotateRefreshToken` で保持・引き継ぎ、
  `refresh_tokens.go` のローテーション時に `Audiences` として `SignAccessToken` へ
  渡すことで、authorization_code + offline_access + resource で取得した refresh
  token をローテーションしても audience 限定が失われないようにした。Postgres
  `refresh_tokens` テーブルに `resource TEXT` (nullable) 列を追加し sqlc を再生成した。
  全層 test-first (RED 確認 → GREEN) で実装した。
- **Verification Results**:
  - `just yaml-check` - passed（SCL / work-item / ids / architecture cross-check /
    traceability すべて green）。
  - `just build-go` - passed。
  - `just lint-go` - passed（0 issues）。
  - `just test-go`（`go test ./...`）- 全 green（wi-262 関連の新規テストを含む）。
  - 手動: ローカルサーバー (`go run ./backend/cmd/idmagic`) へ curl で DCR
    (`/register`) → client_credentials + 未登録 resource で `invalid_target` の
    fail-closed 拒否、resource 未指定時は既存動作 (200 OK, aud=client_id) が
    無変更であることを実地確認した。resource 登録済み時の aud 限定と refresh
    rotation 後の aud 保持は、管理 API がセッション認証を要求するため curl では
    未確認だが（wi-56 と同じ制約）、Go の統合テスト
    (`TestTokenClientCredentials_registeredResource_boundAudience`、
    `TestRefreshTokens_preservesResourceBindingAcrossRotation`、
    `TestRefreshTokenStoreRoundTrip_PreservesResource`) で検証済み。
- **Affected Guarantees State**: resource パラメータを指定しない既存クライアント
  (client_credentials・refresh token 双方) の挙動・保証義務は無変更
  (`AllAccessTokensCarryAudience` は従来どおり client_id を audience として満たす)。
  新規に拡張した保証義務は「resource パラメータ指定時は client_credentials・
  refresh rotation でも登録済み Active な McpResourceServer への audience 厳格限定を
  fail-closed で強制し、rotation を跨いで保持する」(ADR-055 決定3 の対象拡大)。
  device_code のみ引き続き対象外（MCP 認可仕様が要求する利用パターンではないため）。
