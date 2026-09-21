---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-18
priority: p1
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 汎用 API は未知の JSON プロパティを無視するようになり、対象の管理 API は 64 KiB を超える本文を新たに 400 で拒否する。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-599-decode-admin-request-bodies-strictly.md }
initial_context:
  specification:
    - docs/domain/scenarios.feature.md#REQ-PLATFORM-005
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-017
    - docs/design/application/api-guidelines.md
  typespec:
    - IdMagic.Saml.Operations.RegisterSamlServiceProvider
    - IdMagic.WsFederation.Operations.RegisterWsFedRelyingParty
    - IdMagic.WsFederation.Operations.ConfigureEntraFederation
    - IdMagic.Tenancy.Operations.UpdateTenantQuota
  source:
    - backend/shared/http/support_http/response.go
    - backend/saml/handlers_http/admin_service_provider_handler.go
    - backend/wsfederation/handlers_http/admin_relying_party_handler.go
    - backend/wsfederation/handlers_http/admin_entra_handler.go
    - backend/tenancy/handlers_http/admin_tenant_handler.go
    - backend/tenancy/handlers_http/admin_notification_template_handler.go
  tests:
    - backend/shared/http/support_http/misc_helpers_test.go
    - backend/saml/handlers_http/saml_handler_test.go
    - backend/wsfederation/handlers_http/wsfed_handler_test.go
    - backend/tenancy/handlers_http/admin_tenant_handler_test.go
    - backend/tenancy/handlers_http/refusal_effects_test.go
    - backend/apitoken/handlers_http/handlers_test.go
    - backend/shared/http/server_http/tenant_quota_csrf_test.go
  stop_before_reading: [infra, load, frontend/src/features]
affected_spec:
  - { path: docs/domain/scenarios.feature.md, requirement: REQ-PLATFORM-005 }
  - { path: docs/domain/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-017 }
  - { path: spec/contexts/saml/main.tsp, symbol: IdMagic.Saml.Operations.RegisterSamlServiceProvider }
  - { path: spec/contexts/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.RegisterWsFedRelyingParty }
  - { path: spec/contexts/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.ConfigureEntraFederation }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantQuota }
primary_use_cases:
  - id: ignore-unknown-generic-api-properties
    requirement: REQ-PLATFORM-005
    observable_result: 汎用 API は未知の JSON プロパティを無視し、既知のプロパティによる状態変更を完了する。
    unit_test: { path: backend/shared/http/support_http/misc_helpers_test.go, name: TestDecodeJSONIgnoresUnknownProperties_REQ_PLATFORM_005, task: test-go-race }
    e2e_test: { path: backend/tenancy/handlers_http/refusal_effects_test.go, name: TestNotificationTemplateIgnoresFromAddressOverrideAndKeepsDisplayNameOnly, task: test-go-race }
    unit_fault_model: 共通デコーダーが未知のプロパティを拒否する。
    e2e_fault_model: 未知の差出人アドレスを含む要求が既知の表示名まで保存せず、400 で終了する。
  - id: reject-oversized-admin-json
    requirement: REQ-PLATFORM-005
    observable_result: 対象の 4 操作は 64 KiB を超える本文を 400 で拒否し、状態を変更しない。
    unit_test: { path: backend/shared/http/support_http/misc_helpers_test.go, name: TestDecodeJSONRejectsOversizedBody_REQ_PLATFORM_005, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/admin_json_body_e2e_test.go, name: TestE2EAdminJSONBodiesRejectOversizedRequests_REQ_PLATFORM_005, task: test-go-race }
    unit_fault_model: 64 KiB で読み取りを打ち切り、上限内の有効な JSON とそれに続く上限超過の空白を受理する。
    e2e_fault_model: 対象操作の一つが Echo の Bind を使い続け、共通の上限付きデコーダーを迂回する。
---

# 管理 API のリクエストボディを上限付きでデコードし未知のプロパティを無視する

## Motivation

[API ガイドライン](../../docs/design/application/api-guidelines.md)の「リクエストボディのサイズ」は、汎用 API の JSON のリクエストボディを 64 KiB に制限する。
次の 4 つの API 操作は、`support_http.DecodeJSON` ではなく Echo の `Bind` でデコードしているため、この上限が適用されない。

- `RegisterSamlServiceProvider`
- `RegisterWsFedRelyingParty`
- `ConfigureEntraFederation`
- `UpdateTenantQuota`

一方、既存の `support_http.DecodeJSON` は未知のプロパティを 400 で拒否する。
IdMagic が採用する OAuth 2.0、OpenID Connect、CIBA、Dynamic Client Registration、JWT、WebAuthn の拡張規則は、未知の名前を無視する設計を主流としている。
汎用 API も同じ既定へ揃え、未知のプロパティを無視する。
`Bind` はパスパラメーターとクエリパラメーターもリクエストの構造体へ書き込むため、宣言と異なる場所から値を読む可能性もある。

## Scope

- 上記 4 つの API 操作を `support_http.DecodeJSON` でデコードする。
- 汎用 API の未知の JSON プロパティを無視する。
- 上記 4 操作が 64 KiB の超過を 400 で拒否することを、HTTP の境界で固定する。
- 通知テンプレートの未知の差出人アドレスを無視し、表示名だけを更新できる既存の安全性を保つ。
- API ガイドラインの適用状況を改める。

## Out of Scope

- RFC 9396 の `authorization_details` や RFC 9493 の Subject Identifier など、標準が閉じた構造として未知のフィールドの拒否を要求するプロトコルオブジェクト。

## Design

`support_http.DecodeJSON(*http.Request, any) error` は本文を 64 KiB と超過検出用の 1 byte まで読み、上限内の場合だけ通常の JSON デコーダーへ渡す。
未知のプロパティは標準ライブラリの既定どおり無視し、既知のプロパティの型不一致と不正な JSON は拒否する。
前段のロードバランサーまたはリバースプロキシは帯域と接続を守る粗い上限を担い、アプリケーションの 64 KiB 上限は API 契約と前段非依存の多層防御を担う。
4 つの HTTP ハンドラーは `Bind` をこの操作に置き換え、デコードの失敗を既存と同じ 400 `invalid_request` で返す。
拒否時は永続化ポートへ到達しない。
リクエストの構造体が `param` や `query` のタグを持つ場合は、その値をパスまたはクエリから明示的に読む。

## Plan

1. `mise run test-go-test -- ./backend/shared/http/support_http TestDecodeJSONIgnoresUnknownProperties_REQ_PLATFORM_005` と `TestDecodeJSONRejectsOversizedBody_REQ_PLATFORM_005` で Unit RED を確認する。
2. `mise run test-go-test -- ./backend/tenancy/handlers_http TestNotificationTemplateIgnoresFromAddressOverrideAndKeepsDisplayNameOnly` で未知プロパティの E2E RED を確認する。
3. `mise run test-go-test -- ./backend/shared/http/server_http TestE2EAdminJSONBodiesRejectOversizedRequests_REQ_PLATFORM_005` で 4 操作の E2E RED を確認する。
4. 共通デコーダーから未知フィールド拒否を外して上限超過を検出し、4 つの `Bind` を `DecodeJSON` に置き換える。
5. 変更した各パッケージを `mise run test-go-package -- <package>` で確認し、複数パッケージの接続を `mise run test-go-changed` で確認する。

## Tasks

- [x] T001 [Spec] `REQ-PLATFORM-005` と API ガイドラインに未知のプロパティの無視と 64 KiB 超過の拒否を定め、`EX-TENANCY-017-04` を同じ規則へ揃える。
- [x] T002 [Unit] `DecodeJSON` の未知プロパティと厳密な 64 KiB 上限について Unit RED を確認し、GREEN にする（REQ-PLATFORM-005）。
- [x] T003 [Acceptance] 未知の差出人アドレスを無視する E2E RED と、4 操作が 64 KiB 超過を拒否する E2E RED を確認する（REQ-PLATFORM-005、EX-TENANCY-017-04）。
- [x] T004 [App] 4 操作を `DecodeJSON` に置き換える。
- [x] T005 [Docs] API ガイドラインの適用状況を改める。
- [x] T006 [Verify] 変更への耐性と全体を検証する。

## Verification

- `mise run check-contract-drift`
- `mise run verify`
- `mise run test-ui-e2e`

## Risk Notes

未知のプロパティを無視すると綴り間違いも成功するため、TypeSpec から生成した型付きクライアントを綴り間違いの主な検出手段とする。
標準が未知のフィールドの拒否を要求する閉じたプロトコルオブジェクトは、汎用デコーダーではなくその型固有の検証を維持する。

## Completion

- **Completed At**: 2026-09-21
- **Summary**:
  `mise run spec-diff` の結果は `REQ-PLATFORM-005` の追加と `REQ-TENANCY-017` の変更である。
  `support_http.DecodeJSON` は未知の JSON プロパティを無視し、64 KiB と超過検出用の 1 byte までを読み、上限超過を拒否するようになった。
  SAML サービスプロバイダー、WS-Federation 証明書利用者、Entra フェデレーション、テナントクォータの 4 操作を共通デコーダーへ移し、未知プロパティの前方互換性と状態変更前のサイズ拒否を HTTP 境界で固定した。
  API ガイドラインには、汎用 API の未知プロパティをエンドポイントごとに切り替えず無視する判断、標準が閉じた構造を要求するプロトコルだけを型固有検証で例外化する判断、前段の粗い上限とアプリケーションの契約上限を併用する判断を記録した。
- **Primary Use Case Evidence**:
  - id: ignore-unknown-generic-api-properties
    unit_red: "`TestDecodeJSONIgnoresUnknownProperties_REQ_PLATFORM_005` は未知フィールドエラーで失敗した。"
    e2e_red: "`TestNotificationTemplateIgnoresFromAddressOverrideAndKeepsDisplayNameOnly` は未知の差出人プロパティを含む要求が 400 になり、既知の表示名を保存できず失敗した。"
    unit_fault_injection: "`DecodeJSON` に `DisallowUnknownFields` を戻すと、`TestDecodeJSONIgnoresUnknownProperties_REQ_PLATFORM_005` が未知フィールド拒否を検出した。"
    e2e_fault_injection: "通知テンプレート更新だけを厳格デコーダーへ戻すと、`TestNotificationTemplateIgnoresFromAddressOverrideAndKeepsDisplayNameOnly` が 400 を検出した。"
  - id: reject-oversized-admin-json
    unit_red: "`TestDecodeJSONRejectsOversizedBody_REQ_PLATFORM_005` は上限超過本文を受理して失敗した。"
    e2e_red: "`TestE2EAdminJSONBodiesRejectOversizedRequests_REQ_PLATFORM_005` は 4 操作で 201/200 と状態変更を観測して失敗した。"
    unit_fault_injection: "`DecodeJSON` の長さ検査を外すと、同テストが 64 KiB 超過本文の受理を検出した。"
    e2e_fault_injection: "SAML 操作だけを `Bind` に戻すと、同 E2E テストが SAML の 201 と永続化を検出した。"
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/shared/http/support_http` は 431 変異を発見し、302 killed、65 lived、19 not covered、43 not viable、2 timed out だった。
  変更した `response.go` の上限値、読み取り上限、境界条件、条件反転の 4 変異はすべて killed であり、残存変異は同パッケージの既存ロジックに属する。
  追加した 4 つの手動フォールト注入も、それぞれ対応する Unit または E2E テストで検出し、注入後はすべて復元した。
- **Verification Results**:
  - `mise run check-work-items` - 成功
  - `mise run check-spec` - 成功（normative coverage: 727 ids）
  - `mise run spec-render` - 成功（1093 pages, 19 API tags）
  - `mise run check-contract-drift` - 成功
  - `mise run check-api-compat` - 成功
  - 変更パッケージの `mise run test-go-package` と `mise run test-go-changed` - 成功
  - `mise run lint-go` - 成功（0 issues）
  - `mise run verify` - 成功（サンドボックス外で実行）
  - `mise run test-ui-e2e` - 成功（38 pass, 0 fail）
