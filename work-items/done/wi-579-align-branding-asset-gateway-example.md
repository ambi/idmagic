---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-14
priority: p2
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: ゲートウェイは既に realm 配下の branding アセットを転送しており、具体例を現行の振る舞いへ直すだけで利用者が観測する振る舞いは変わらない。
  references: []
initial_context:
  specification:
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-004
  typespec:
    - IdMagic.Tenancy.Operations.GetTenantBrandingAsset
  source:
    - frontend/vite.config.ts
    - frontend/Caddyfile
    - backend/shared/http/server_http/gateway_allowlist.go
    - frontend/src/features/admin-settings/BrandingTab.tsx
    - frontend/src/api/branding.ts
  tests:
    - backend/shared/http/server_http/gateway_allowlist_test.go
    - frontend/tests/e2e/gateway-routes.spec.ts
    - frontend/tests/e2e/ui-scenario-actions.spec.ts
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - backend/tenancy/usecases
    - backend/tenancy/db_postgres
affected_spec:
  - { path: docs/domain/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-004 }
primary_use_cases:
  - id: realm-logo-url-is-served-through-gateway
    requirement: REQ-TENANCY-004
    observable_result: 管理画面からアップロードしたロゴの realm 配下の logo_url を開発用ゲートウェイ越しに GET すると、SPA ではなく backend が応答し、アップロードした PNG と同じバイト列が返る。
    unit_test: { path: backend/shared/http/server_http/gateway_allowlist_test.go, name: TestCheckGatewayAllowlistsReportsAnUnforwardedRealmBrandingAsset, task: test-go-race }
    e2e_test: { path: frontend/tests/e2e/ui-scenario-actions.spec.ts, name: admin logo upload is served back through the gateway under the realm, task: test-ui-e2e }
    unit_fault_model: ゲートウェイの realm 配下の転送対象から tenant-branding-assets が外れても、経路表との照合が乖離を報告しない。
    e2e_fault_model: 開発用ゲートウェイが realm 配下の logo_url を転送せず、SPA の index.html が 200 で応答する。
---

# realm 配下の branding アセット転送例を現行 gateway と一致させる

## Motivation

`EX-TENANCY-004-03` は、realm 配下の `logo_url` を gateway が backend へ転送せず、画像取得が失敗すると記述する。

現在の `frontend/vite.config.ts` は realm 配下の転送対象へ `tenant-branding-assets` を含める。
サーバー側にも同じ経路が登録されているため、具体例が記録する失敗と現行の経路構成が一致しない。

## Scope

- `EX-TENANCY-004-03` が要求する最終状態を、現行の成功経路と過去の回帰事例のどちらとして扱うか決める。
- 現行の成功経路を要求する場合は、具体例を成功条件へ書き換え、gateway と backend の双方を通るテストを対応付ける。
- 失敗事例を履歴として残す場合は、規範の具体例から外し、適切な変更記録へ移す。

## Out of Scope

- branding アセットの URL 形式または保存形式の変更。
- 他の gateway 転送対象の棚卸し。
- 越境したアセット取得のエラー契約。`EX-TENANCY-004-02` は [[wi-578-align-cross-tenant-branding-asset-refusal]] が扱う。

## Design

規範シナリオは必要な製品の振る舞いを記述するため、解消済みの失敗を `Then` に残すと、正常な実装を不適合として扱う。

具体例を「realm 配下の `logo_url` が gateway から backend へ転送され、画像取得に成功する」へ直す。
過去の失敗は、この記録の Motivation に残るため、規範にも回帰テストの説明にも書かない。
`EX-TENANCY-004-01` と重なる配色とフッターの手順は、ゲートウェイの転送と関係しないため具体例から外す。

ゲートウェイの転送は二つの層で固定する。

| 層 | テスト | 固定する内容 |
| --- | --- | --- |
| Unit | `TestCheckGatewayAllowlistsReportsAnUnforwardedRealmBrandingAsset`（`backend/shared/http/server_http`） | `CheckGatewayAllowlists` が、realm 配下の転送対象から `tenant-branding-assets` を外した Caddyfile と Vite 設定の両方を、`/realms/sample/tenant-branding-assets/` を名指しして拒否する |
| E2E | `admin logo upload is served back through the gateway under the realm`（`frontend/tests/e2e/ui-scenario-actions.spec.ts`） | 管理画面のブランディングタブからロゴをアップロードし、プレビューが参照する realm 配下の `logo_url` を開発用ゲートウェイ越しに GET すると、アップロードした PNG と同じバイト列が `image/png` で返る |

E2E は default テナントの branding を変えるため、`finally` でロゴを削除して後続のテストへ状態を残さない。
製品コードは変更しない。

## Plan

1. gateway と backend の現行経路を正式な入口から観測する。
2. `EX-TENANCY-004-03` の規範上の役割を決める。
3. 具体例を決定済みの成功条件へそろえ、gateway の回帰テストを追加する。
4. UI が返された `logo_url` を表示へ使う経路も確認する。

## Tasks

- [x] T001 [Decision] `EX-TENANCY-004-03` を規範の成功例として直すか、履歴へ移すか決める。
- [x] T002 [Spec] Tenancy の具体例を決定へそろえる。
- [x] T003 [Acceptance] gateway と backend を通る realm 配下の画像取得を検証する。
- [x] T004 [UI] 返された `logo_url` が画像表示へ渡ることを検証する。
- [x] T005 [Verify] 仕様、UI、ブラウザー経路の検証を通す。

## Verification

- `mise run check-spec`
- `mise run test-ui-unit-file -- ./src/components/Brand.test.tsx`
- `mise run test-ui-e2e`
- `mise run verify`

## Risk Notes

設定 API の成功だけを見ても、gateway が画像 URL を転送できることは証明できない。
受け入れテストは画像 URL への HTTP 要求まで進め、画像本文を観測する。

## Completion

- **Completed At**: 2026-09-24
- **Summary**:
  `mise run spec-diff` は REQ-TENANCY-004 のシナリオ変更を報告した。
  `EX-TENANCY-004-03` は、ゲートウェイが realm 配下の `logo_url` を転送せず画像取得に失敗するという記述から、管理画面からゲートウェイ越しにアップロードしたロゴの realm 配下の `logo_url` をゲートウェイ越しに GET すると、backend へ転送されて同じ PNG が `image/png` で返るという成功例へ変わった。
  `EX-TENANCY-004-01` と重なっていた配色とフッターの手順は具体例から外した。
  製品コードは変更せず、`tools/check/example-coverage-debt.json` から `EX-TENANCY-004-03` の項目を削除した。
- **Primary Use Case Evidence**:
  - id: realm-logo-url-is-served-through-gateway
    unit_red: 製品の転送設定は変更前から realm 配下の branding アセットを転送しており、TestCheckGatewayAllowlistsReportsAnUnforwardedRealmBrandingAsset は書いた時点で GREEN だった。実装前に失敗を観測した代替検査は、例の被覆負債を削除した後の `mise run check-spec` であり、`EX-TENANCY-004-03 is declared, but no test names it` で失敗した。
    e2e_red: E2E テスト admin logo upload is served back through the gateway under the realm も書いた時点で GREEN だった。RED は同じく `mise run check-spec` の被覆欠落として観測した。
    unit_fault_injection: 経路の分類規則 `gatewayExposureRules` へ `/tenant-branding-assets/{}/{}` を optional として加えると、照合が乖離を報告しなくなり、単体テストが Caddyfile と vite.config.ts の両方について期待した指摘の欠落を検出して失敗した。
    e2e_fault_injection: 開発用ゲートウェイの設定 `frontend/vite.config.ts` の realm 配下の転送対象から `tenant-branding-assets` を外すと、logo_url への GET は SPA の fallback が 200 で応答し、E2E テストが Content-Type `text/html` を検出して失敗した。
- **Change-Resistance Results**:
  - 低リスクのため、上の二つの障害注入を代表とした。どちらも製品コードの変更を伴わない仕様の是正であり、変異テストは実行していない。
- **Verification Results**:
  - `mise run check-spec` - passed（負債項目の削除直後は EX-TENANCY-004-03 の被覆欠落で failed）
  - `mise run test-go-test -- ./backend/shared/http/server_http TestCheckGatewayAllowlistsReportsAnUnforwardedRealmBrandingAsset` - passed
  - `mise run test-ui-unit-file -- ./src/components/Brand.test.tsx` - passed
  - `mise run test-ui-e2e` - passed（39 件）
  - `mise run verify` - passed
