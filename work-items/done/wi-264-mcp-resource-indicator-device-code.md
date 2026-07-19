---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-07-19
depends_on: [wi-262-mcp-resource-indicator-remaining-grants]
---

# MCP resource indicator を device_code グラントへ拡張し RFC8707 を完全準拠にする

## Motivation
[[wi-262-mcp-resource-indicator-remaining-grants]] は client_credentials と
authorization_code の refresh token rotation へ resource indicator (RFC 8707)
の audience 限定を拡張したが、device_code のみ対象外のまま
`RFC8707-MCP-RESOURCE-BINDING` requirement は `adoption: partial` に留まっていた。

ユーザーからの指摘を受けて再検討した結果、device_code を対象外とした根拠は
「MCP 認可仕様が device flow を要求パターンとして想定していない」という
MCP 文脈での優先度判断であり、RFC 8707 準拠そのものを妨げる技術的制約では
なかった。RFC 8707 は grant type を区別しない設計 (resource パラメータは
authorization request と token request のどちらでも使える汎用拡張) であり、
かつ実装コストも client_credentials と同型で低い
(`usecases.ResolveResourceIndicator` を device_code の token exchange に
配線するだけ)。本 WI で device_code にも対応し、
`RFC8707-MCP-RESOURCE-BINDING` を `adoption: required` に格上げして
RFC 8707 を完全準拠にする。

## Scope
- `spec/contexts/oauth2.yaml`:
  - `RFC8707-MCP-RESOURCE-BINDING` の `adoption` を `partial` → `required` に、
    `reason` を削除し `statement` のみへ整理（除外経路がなくなるため）。
  - `models.TokenRequest.resource` の description から device_code 未対応の記述を除去。
  - `interfaces.Token` の `requires` に DeviceCode 向け resource 検証を追加。

## Out of Scope
- なし（RFC8707-MCP-RESOURCE-BINDING の全経路が対象になる）。

## Plan
既存の client_credentials 実装 ([[wi-262-mcp-resource-indicator-remaining-grants]])
と同型のパターンを `ExchangeDeviceCode` (device_code の token polling, RFC 8628 §3.4)
へ適用する。`/device_authorization` (初期リクエスト) 自体には resource を持たせず、
token polling リクエスト (`/token`, `grant_type=urn:ietf:params:oauth:grant-type:device_code`)
でのみ resource を受理する — RFC 8707 §2 が token request での resource 指定を
正規の使い方として認めており、client_credentials と対称的な設計になる。

- `ExchangeDeviceCodeInput` に `Resource []string` を追加。
- `ExchangeDeviceCode` 内で `usecases.ResolveResourceIndicator` を呼び、
  `Audiences` を `SignAccessToken` へ、resource を `GenerateInitialRefreshToken`
  へ渡す (device_flow.go は現在 `nil, nil` を渡しており、`nil, resource` に変更)。
  Refresh token rotation (`refresh_tokens.go`) は wi-262 で既に resource 保持に
  対応済みのため、device_code 発行の refresh token もローテーション時に自動で
  audience 限定を保持する。
- `token_handler.go` の device_code 分岐に `c.Request().PostForm["resource"]` を渡す。

## Tasks
- [x] T001 [SCL] `RFC8707-MCP-RESOURCE-BINDING` を `adoption: required` に格上げし
      （`reason` は partial/excluded 専用のため削除、`statement` へ対象経路一覧を統合）、
      `TokenRequest.resource` の description を更新、`Token` interface の requires に
      `input.request.grant_type != DeviceCode || resource_indicator_registered_and_active(...)`
      を追加。`just yaml-check` / `just scl-render` green。
- [x] T002 [OAuth2/device_code] `ExchangeDeviceCode` に resource indicator 検証と
      audience 束縛・refresh token への resource 伝播を配線。RED:
      `TestExchangeDeviceCode_unregisteredResource_rejectedAsInvalidTarget` を先に
      fail 確認 (`backend/oauth2/usecases/device_flow_resource_indicator_test.go`) →
      GREEN。HTTP 層の配線も
      `backend/oauth2/adapters/http/device_code_resource_indicator_test.go` で
      RED→GREEN 確認（`/token` への resource フォームパラメータ → introspection での
      aud 確認）。
- [x] T003 [Verify] `just yaml-check` / `just build-go` / `just lint-go`(0 issues) /
      `just test-go`(全 green) を確認。手動: ローカルサーバーへ curl で DCR →
      device_authorization → 未承認状態での token polling が引き続き
      `authorization_pending` を返すこと（resource 未指定時の既存動作が壊れていないこと）
      を実地確認。resource の fail-closed 拒否と登録済み resource への aud 限定は
      HTTP 統合テストで検証済み（承認フローがブラウザセッションを要求するため curl での
      完全な E2E は wi-56/wi-262 と同じ制約により未実施）。

## Verification
- `just yaml-check`
  - reason: SCL 変更（adoption: required 格上げ、Token requires）の整合。
- `just build-go` / `just lint-go` / `just test-go`
  - reason: device_code の resource 束縛と fail-closed 拒否、refresh rotation を跨いだ保持。
- 手動: device_code grant に `resource` を指定し、未登録 resource は fail-closed
  拒否、登録済み resource は aud 限定されることを確認する。

## Risk Notes
既存の client_credentials 実装と同型のパターンを流用するため新規リスクは低い。
resource パラメータを指定しない既存 device_code クライアントの挙動は無変更
（`ResolveResourceIndicator` は resource 未指定時 `(nil, nil)` を返す）。

## Completion
- **Completed At**: 2026-07-19
- **Summary**:
  RFC 8707 resource indicator の audience 限定を device_code グラント
  (`ExchangeDeviceCode`, RFC 8628 §3.4 の token polling) へ拡張し、
  `RFC8707-MCP-RESOURCE-BINDING` requirement を `adoption: partial` から
  `adoption: required` へ格上げした。これにより Authorize / PushAuthorizationRequest /
  Token(authorization_code redemption・refresh rotation・client_credentials・
  device_code・token-exchange) の全経路で resource パラメータ指定時の audience 厳格
  限定が一様に適用され、RFC 8707 は完全準拠になった。実装は client_credentials
  (wi-262) と同型のパターンを踏襲し、`/device_authorization` (初期リクエスト) 自体は
  無変更のまま、token polling リクエストでのみ resource を受理する設計とした
  (RFC 8707 §2 が token request での指定を正規の使い方として認めているため)。
  refresh token rotation は wi-262 で実装済みの resource 保持機構をそのまま再利用し、
  device_code 発行の refresh token もローテーション後に audience 限定を保持する。
  全層 test-first (RED 確認 → GREEN) で実装した。
- **Verification Results**:
  - `just yaml-check` - passed（SCL / work-item / ids / architecture cross-check /
    traceability すべて green）。
  - `just build-go` - passed。
  - `just lint-go` - passed（0 issues）。
  - `just test-go`（`go test ./...`）- 全 green。
  - 手動: ローカルサーバー (`go run ./backend/cmd/idmagic`) へ curl で DCR
    (`/register`) → `/device_authorization` → 未承認状態での token polling が
    引き続き `authorization_pending` を返すこと（resource パラメータの評価順序が
    既存の状態チェックを壊していないこと）を実地確認した。resource の
    fail-closed 拒否 (`invalid_target`) と登録済み resource への aud 限定は
    HTTP 統合テスト (`TestTokenDeviceCode_unregisteredResource_rejectedAsInvalidTarget`、
    `TestTokenDeviceCode_registeredResource_boundAudience`) で検証済み
    （user_code 承認がブラウザセッションを要求するため、curl での完全な E2E は
    wi-56/wi-262 と同じ制約により未実施）。
- **Affected Guarantees State**: resource パラメータを指定しない既存 device_code
  クライアントの挙動・保証義務は無変更。新規に拡張した保証義務は「resource
  パラメータ指定時は device_code でも登録済み Active な McpResourceServer への
  audience 厳格限定を fail-closed で強制する」(ADR-055 決定3 の全経路適用完了)。
  `RFC8707-MCP-RESOURCE-BINDING` は本 WI をもって除外経路がなくなり、
  `adoption: required` として RFC 8707 に完全準拠した。
