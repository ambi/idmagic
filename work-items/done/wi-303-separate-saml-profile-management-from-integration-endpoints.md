---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-26
depends_on: [wi-301-admin-integration-endpoint-catalog-and-setup-guidance, wi-302-saml-idp-profiles]
change_kind: bugfix
initial_context:
  scl:
    Tenancy: [interfaces.GetAdminIntegrationEndpoints, flows.TenantSettingsManagement]
    Saml: [interfaces.ListSamlIdentityProviderProfiles, flows.AdminSamlIdpProfiles]
  source:
    - frontend/src/features/admin-settings
    - frontend/src/routes/admin
    - frontend/vite.config.ts
    - frontend/Caddyfile
  tests:
    - frontend/src/features/admin-settings
    - frontend/src/features/admin-saml-idp-profiles
  stop_before_reading: [backend/oauth2]
affected_spec:
  - { context: Tenancy, kind: scenario, element: 管理者は正規ロケーションの連携情報を取得する }
  - { context: Tenancy, kind: flow, element: TenantSettingsManagement }
  - { context: Saml, kind: scenario, element: 管理者はSAML IdP profileを共有または専用で管理できる }
  - { context: Saml, kind: flow, element: AdminSamlIdpProfiles }
---

# 連携エンドポイント参照とSAML IdP profile変更を分離する

## Motivation

ローカル開発gatewayが画面に表示したSAML metadata URLをbackendへ転送せず、正規URLを
開いてもSPAのNot Foundになっている。またread-onlyであるべき連携エンドポイント画面に
SAML IdP profileの作成・更新・削除操作が混在し、誤変更の危険とprotocol間の情報階層の
不明瞭さを生んでいる。

## Scope

- `spec/contexts/tenancy.yaml` の `scenarios` と `flows.TenantSettingsManagement`。
- `spec/contexts/saml.yaml` のprofile管理 `scenarios` と `flows.AdminSamlIdpProfiles`。
- Vite/Caddy gatewayで公開protocolエンドポイントをbackendへ転送する。
- 連携エンドポイント画面をprotocol単位に並べ、SAML 2.0配下ではdefaultを含むprofileごとに全エンドポイントと署名証明書をread-only表示する。
- SAML IdP profileの一覧・作成・更新を専用routeへ分離する。
- UI route、表示、gateway routingの回帰テスト。

## Out of Scope

- SAML metadata生成、署名鍵、profile API contractの変更。
- SAML service provider割当UIの変更。

## Plan

- SCLの受け入れscenarioと画面flowを先に更新する。
- gatewayのroute matcherを公開エンドポイントcatalog全体と整合させる。
- IntegrationEndpointsTabからmutation state/API呼び出しを除去し、SAML card内にprofile一覧と管理画面への導線を置く。
- profile管理は一覧、専用作成route、専用編集routeに分け、default profileは参照のみとする。

## Tasks

- [x] T001 [SCL] Tenancy/Samlのscenarioとflowを更新し`just check-scl`で検証した。
- [x] T002 [Gateway] RED: `development gateway > proxies the published integration endpoint` のSAML/WS-* 6ケースがfailすることを先に確認（scenario `管理者は正規ロケーションの連携情報を取得する`）→ Vite/Caddy matcherを修正してGREEN。
- [x] T003 [UI] RED: `groups canonical integration endpoints by protocol and keeps profile values read-only` がinline編集UIを検出してfailすることを先に確認（flow `TenantSettingsManagement.IntegrationEndpoints`）→ protocol階層のread-only表示へ再構成してGREEN。
- [x] T004 [UI] RED: `SAML IdP profile routed management` の一覧・詳細・作成・編集4ケースが専用route未実装でfailすることを先に確認（flow `AdminSamlIdpProfiles`）→ 専用routeを実装してGREEN。
- [x] T005 [Verify] 派生物を再生成し、UIとworkspace全体を検証した。

## Verification

- `just check`
- `just test-ui-unit`
- `just verify-ui`
- `just verify`

## Risk Notes

gateway matcherの漏れは表示URLと実到達性の再乖離を招くため、catalogに含まれるprotocol
prefixをまとめてテストする。外部入力parserや認可判定は変更しないためfuzz/property testは
追加しない。

## Completion

- **Completed At**: 2026-07-26
- **Summary**:
  - Vite/Caddy gatewayへSAML、WS-Federation、WS-Trustの公開protocol routeを追加した。
  - 連携エンドポイント画面をread-onlyのprotocol分類へ整理し、SAMLはDefaultを含む各profile内にentityID、metadata、SSO、SLO、署名証明書をまとめた。
  - SAML IdP profileの一覧・詳細・作成・編集を専用routeへ分離した。
  - 一般用語の「エンドポイント」「ベースURL」を日本語表記へ統一し、protocol仕様上の正式なEndpoint名は英語表記を維持した。
- **Verification Results**:
  - `just scl-render` - passed
  - `just verify` - passed
