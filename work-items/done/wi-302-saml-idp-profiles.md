---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-26
depends_on: [wi-300-durable-tenant-xml-federation-signing-credentials, wi-301-admin-integration-endpoint-catalog-and-setup-guidance]
change_kind: feature
initial_context:
  scl:
    Saml: [models.SamlServiceProvider, interfaces.PublishSamlMetadata, interfaces.SamlSingleSignOn]
    Application: [models.ApplicationSamlConfig, interfaces.GetAdminApplication]
  source:
    - backend/saml
    - backend/application
    - frontend/src/features/admin-applications
  tests:
    - backend/saml
    - backend/application
    - frontend/src/features/admin-applications
  stop_before_reading: [backend/oauth2]
affected_spec:
  - { context: Saml, kind: model, element: SamlServiceProvider }
  - { context: Saml, kind: model, element: SamlIdentityProviderProfile }
  - { context: Saml, kind: interface, element: PublishSamlMetadata }
  - { context: Saml, kind: interface, element: SamlSingleSignOn }
  - { context: Application, kind: model, element: ApplicationSamlConfig }
  - { context: SigningKeys, kind: model, element: SigningKey }
---

# SAML IdP profileを共有またはアプリ専用で割り当てられるようにする

## Motivation

通常はテナント共通 IdP metadata で運用できるが、証明書・entityID・変更blast radiusを
取引先やアプリ単位に分離すべき連携もある。常に共有、常にSP専用のどちらかに固定せず、
同じ profile モデルで共有と専用を表現する必要がある。

## Scope

- `SamlIdentityProviderProfile` と SP の必須 profile binding。
- default profile と profile-aware な正規 SAML route。
- profile固有 metadata/SSO/SLO/certificate route。
- profile CRUD、共有profile選択、アプリ専用profile作成UI。
- profile別署名資格情報とcross-profile request拒否。
- ADR と SAML architecture record。

## Out of Scope

- OIDC client ごとの issuer。
- 1 SP への複数 active IdP profile 同時割当。
- inbound SAML broker。

## Plan

- profileはtenant内で複数SPから参照可能とし、専用profileは同じmodelを1 SPだけで使う。
- default profileはtenant作成時に用意し、新規SPを必ずいずれかのprofileへ束縛する。
- 追加profileのentityIDは `{canonical_issuer}/saml/idp/{profile_id}`、endpointも同じ
  profile prefix配下とする。
- Destination、SP Issuer、profile bindingが一致する要求だけを受理する。

## Tasks

- [x] T001 [ADR/SCL] 共有可能profileを採用し、常時共有・常時専用を却下した理由、model/interface/state/scenarioを記録する。
- [x] T002 [Domain] RED: profile identity/cardinality/cross-profile guard test を先に fail 確認（scenario `SPは割当IdP profileだけを利用できる`）→ GREEN。
- [x] T003 [Persistence] RED: default作成、profile CRUD/binding repository test を先に fail 確認（同 scenario）→ GREEN。
- [x] T004 [Protocol] RED: profile別 metadata/SSO/SLO/certificate test を先に fail 確認（scenario `専用profileは固有metadataを公開する`）→ GREEN。
- [x] T005 [Admin API/UI] RED: 共有選択・専用作成・使用中削除拒否 test を先に fail 確認（flow `AdminSamlIdpProfiles`）→ GREEN。
- [x] T006 [Architecture/Verify] 設計記録、派生物、初期schemaを検証する。

## Verification

- `just check-scl`
- `just scl-render`
- `just test-go`
- `just test-ui-unit`
- `just test-ui-e2e`
- `just verify`
- `just check`

## Risk Notes

routeとprofile bindingの混同はtenant内の別SPへassertionを発行する危険がある。
Destination・SP entityID・profile IDを一体で照合し、default alias経由でも同じguardを通す。
XML parser自体は既存実装を利用するため新規 fuzz target は不要だが、cross-profile property
caseをtable-driven testで網羅する。

## Completion

- **Completed At**: 2026-07-26
- **Summary**: Added tenant-local shared and dedicated SAML IdP profiles with profile-bound service providers, canonical profile endpoints, isolated XML signing credentials, and localized administration workflows.
- **Verification Results**:
  - `just verify` - passed (Go/UI tests, builds, lint, type checks, SCL, Architecture, and traceability)
  - `just test-ui-e2e` - passed (22 browser scenarios)
  - `just check` - passed
- **Evidence**:
  - Domain and repository tests cover default profile creation, dedicated cardinality, persistence, and in-use deletion rejection.
  - Protocol tests cover profile-specific metadata and certificates, signer isolation, and fail-closed cross-profile SSO rejection.
  - Admin API and UI tests cover profile CRUD, canonical setup values, shared selection, dedicated creation, and application-specific setup guidance.
