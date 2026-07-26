---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-26
depends_on: [wi-300-durable-tenant-xml-federation-signing-credentials]
change_kind: feature
initial_context:
  scl:
    Tenancy: [interfaces.GetAdminSettings]
    Application: [interfaces.GetAdminApplication]
    Saml: [interfaces.PublishSamlMetadata]
  source:
    - backend/tenancy/handlers_http
    - backend/saml/handlers_http
    - frontend/src/features/admin-settings
    - frontend/src/features/admin-applications
  tests:
    - backend/tenancy/handlers_http
    - backend/saml/handlers_http
    - frontend/src/features
  stop_before_reading: [backend/idmanagement]
affected_spec:
  - { context: Tenancy, kind: interface, element: GetAdminIntegrationEndpoints }
  - { context: Tenancy, kind: model, element: AdminIntegrationEndpointCatalog }
  - { context: Application, kind: interface, element: GetAdminApplication }
  - { context: Saml, kind: interface, element: DownloadSamlSigningCertificate }
---

# 管理画面に正規の連携 endpoint catalog とアプリ設定ガイドを表示する

## Motivation

公開済みの OIDC discovery、SAML/WS-Fed metadata、SCIM/API URL が管理画面で一貫して
発見できず、アプリ詳細には相手側へ投入する IdMagic の値がない。管理者が path /
subdomain / custom domain を意識して URL を手組みせず、正規ロケーションの値をコピー・
ダウンロードできる必要がある。

## Scope

- tenant admin 用の read-only integration endpoint catalog API。
- Settings の「連携エンドポイント」tab。
- OIDC/SAML application detail の RP/SP setup guidance。
- WS-Fed/Entra と API token 画面にある endpoint 表示の共通 catalog 化。
- SAML active signing certificate の PEM download。

## Out of Scope

- client secret の再表示。
- inbound IdP connection (`wi-30-inbound-federation-and-identity-broker`)。
- SAML IdP profile。

## Plan

- server が request tenant の canonical issuer から絶対URLを組み立て、UI はURLを推測しない。
- discovery/metadata を機械可読な正本、個別endpointとcertificateを手動設定用補助情報とする。
- catalog は秘密情報を含めず no-store で返し、既存 `AdminSettingsRead` permission に束ねる。
- SAMLアプリ詳細は「IdMagicへ登録したSP情報」と「SPへ設定するIdMagic情報」を分離する。

## Tasks

- [x] T001 [SCL] catalog model/interface、certificate download、管理UI flow/scenarioを追加して再生成する。
- [x] T002 [Usecase/HTTP] RED: canonical path/subdomain catalog test を先に fail 確認（scenario `管理者は正規ロケーションの連携情報を取得する`）→ GREEN。
- [x] T003 [SAML Adapter] RED: PEM download と metadata certificate 一致 test を先に fail 確認（scenario `SPは署名証明書を取得できる`）→ GREEN。
- [x] T004 [UI] RED: 日英、コピー、download、OIDC/SAML setup guidance test を先に fail 確認（flow `AdminIntegrationSetup`）→ GREEN。
- [x] T005 [Consolidation] API token / Entra の重複URL生成を catalog 利用へ移す。
- [x] T006 [Verify] protocol別、tenant style別、認可・アクセシビリティを検証する。

## Verification

- `just check-scl`
- `just scl-render`
- `just test-go`
- `just test-ui-unit`
- `just test-ui-e2e`
- `just verify`
- `just check`

## Risk Notes

誤ったissuerやmetadata URLのコピーは連携全体を停止させる。URLは request context の
canonical issuer からのみ生成し、frontendのrealm文字列連結を禁止する。

## Completion

- **Completed At**: 2026-07-26
- **Summary**: Added a canonical tenant integration endpoint catalog, SAML signing certificate download, and localized OIDC RP / SAML SP setup guidance in the administration UI.
- **Verification Results**:
  - `just verify-ui` - passed (450 unit tests, typecheck, lint, and production build)
  - `just test-ui-e2e` - passed (21 browser scenarios)
  - `just verify` - passed
- **Evidence**:
  - HTTP tests cover canonical path- and host-style endpoint generation, authorization, no-store responses, and matching SAML metadata/download certificates.
  - Settings, application detail, Entra federation, and API token tests cover localized catalog presentation and reuse without frontend URL reconstruction.
  - Browser scenarios cover account and admin navigation, application/token workflows, and client-side navigation under the local default tenant.
