---
context: saml
updated_at: 2026-08-11
---

# Saml Specification

## Overview

SAML 2.0 IdP の service provider binding、IdP metadata、AuthnRequest / Response、
AssertionConsumerService / Single Logout を所有する protocol context。Web Browser SSO Profile に
基づき SP-initiated / IdP-initiated SSO を提供する。WS-Fed / WS-Trust とは claim release と
XML signing capability だけを共有する。

The `Saml` context owns SAML 2.0 IdP behavior: service-provider trust, AuthnRequest validation,
SSO/SLO use cases, response construction, IdP profiles, and IdP metadata. Protocol-neutral claim
projection remains in `ClaimMapping`; XML signing is delegated through the shared federation signer
provider.

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Saml | SAML 2.0 IdP protocol family。Web Browser SSO Profile に基づき、SP-initiated / IdP-initiated SSO、IdP metadata 公開、署名済み SAMLResponse の発行、Single Logout、SP ごとの AuthnRequest / LogoutRequest 署名検証を扱う。encrypted assertion、ECP、SAML SP / inbound federation は初期範囲外。XML 署名・canonicalization は実績ある library に委ね、自前実装しない。 | SAML, SAML2, SAML 2.0 |
| EndUser | SAML Web Browser SSO または Single Logout をブラウザで開始する利用者。 |  |
| SamlIdentityProviderProfile | テナント内のSAML IdP entityID、エンドポイント、XML署名資格情報をまとめるtrust境界。shared profileは複数SPで共有でき、dedicated profileは最大1 SPだけに割り当てる。 | SAML IdP profile, IdP profile |

## Standards

### Assertions and Protocols for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Core-BearerAssertion | required | MUST | AuthnRequest は Version 2.0 と許容された IssueInstant を持ち、発行 assertion は Audience / Recipient / InResponseTo を検証済み要求と整合させる。同一 tenant / SP / request ID に対する assertion は一度だけ発行する |
| SAML2Core-EncryptedAssertion | excluded | MAY | encrypted assertion |

### Bindings for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Bindings-RedirectPost | required | MUST | AuthnRequest は HTTP-Redirect または HTTP-POST binding で受理し、明示された response ProtocolBinding が HTTP-POST 以外なら拒否する。SAMLResponse と返信可能な protocol error は HTTP-POST で返す |

### Profiles for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-profiles-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Profile-WebBrowserSSO | required | MUST | SP-initiated / IdP-initiated Web Browser SSO を提供する。未対応の ACS index、NameID format、response binding は fail-closed に拒否し、IsPassive=true でログインが必要な場合は NoPassive protocol response を返す |
| SAML2Profile-ECP | excluded | MAY | Enhanced Client or Proxy profile |

### Metadata for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Metadata-IDPSSODescriptor | required | MUST | IdP metadata は SSO endpoint、SLO endpoint、署名証明書、NameID format を公開する |
| SAML2Metadata-WantAuthnRequestsSigned | optional | MAY | SP ごとの trust policy として AuthnRequest / LogoutRequest 署名検証を要求できる |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### SSO Profile scope

The initial scope is the SAML 2.0 Web Browser SSO Profile only: HTTP-Redirect (deflate+base64) and
HTTP-POST (base64) bindings, signed Response/Assertion, metadata publication, SP-initiated and
IdP-initiated SSO, and Single Logout. SAML ECP, encrypted assertions, and idmagic acting as a SAML SP
against an external IdP (inbound federation) are excluded and deferred to a separate slice if needed
— narrowing scope keeps the well-known SAML signature-wrapping attack surface contained.

Claim issuance and assertion signing reuse the protocol-agnostic builder and signer already shared
with WS-Federation/WS-Trust (`internal/wsfederation/adapters/samltoken`), which already handles SAML
version, bearer subject confirmation, and audience restriction. Only SP-initiated-specific input,
such as `InResponseTo` round-tripping, is added on top rather than reimplementing signing for this
context.

Signing defaults to signing the Assertion, with Response signing available as an opt-in (the "Sign
Response" behavior Okta/Entra also offer). `goxmldsig` appends an enveloped signature at the end of
the element it signs; because the enveloped transform verifies independent of the signature element's
position, the signed element is never repositioned afterward — moving it would redraw namespaces,
change the digest, and break verification. This applies equally to Assertion and Response signing.

Interop guards are fail-closed and centralized in the domain layer: Issuer must exactly match the
registered SP's entityID, `AssertionConsumerServiceURL` is checked against that SP's allow-list (open
redirect prevention), and audience restriction is scoped to the SP's entityID/Audience. Any
indeterminate or mismatched check is rejected.

### Identity provider profiles

Each service provider is bound to exactly one tenant-local identity provider profile. The tenant's
immutable `default` profile is shared and retains the short `/saml/*` routes. Additional profiles use
`/saml/idp/{profile_id}/*` routes and a profile-specific entity ID. A `shared` profile may serve multiple
service providers, while a `dedicated` profile is constrained to at most one. A single model supports
both cases so the protocol, persistence, and administration paths enforce one set of trust-boundary
rules.

Profile administration returns canonical entity, metadata, SSO, SLO, certificate-download, and
fingerprint values generated by the server. It also returns the bound service-provider count so the UI
can prevent deleting an in-use profile. The repository remains authoritative: it rejects changes to the
default profile, dedicated-profile over-allocation, and deletion of any bound profile.

SSO and SLO resolve the profile from the request route and require it to match the selected service
provider's binding. Destination validation uses that same profile's canonical endpoint. This combined
check prevents a valid request for one trust boundary from being replayed through another profile in
the same tenant.

### Tenant signing

Every request resolves its signer from the tenant and profile context immediately before issuance. The
provider selects the active `XmlFederationSigning` credential for that scope and fails closed when the
provider, private signer, or X.509 certificate is unavailable. This request-scoped resolution prevents
a process-global certificate from crossing tenant or profile boundaries.

Each profile's metadata publishes the active certificate plus every unexpired verifying certificate in
the same key scope. New assertions and responses use only the active credential, while the overlap lets
service providers validate messages issued immediately before rotation. XML parsing and canonicalization
continue to use the reviewed library selected for XML signature handling.

### Persistence

`saml_authnrequest_replays` records an AuthnRequest id as seen exactly once: `RecordIfNew` is an
`INSERT ... ON CONFLICT DO NOTHING`, and the insert's affected-row count is the new/replay signal.

### Design Decisions

- The initial SAML 2.0 IdP scope is the Web Browser SSO Profile only (HTTP-Redirect/HTTP-POST
  bindings, SP- and IdP-initiated SSO, Single Logout); SAML ECP, encrypted assertions, and acting as
  a SAML SP are excluded to keep the signature-wrapping attack surface contained.
- SAML IdP profiles are modeled as shareable (a profile can back more than one SP trust), with
  dedicated-use profiles expressed as the one-consumer case of the same model rather than a separate
  type.
- XML parsing, canonicalization, and signing use a reviewed third-party XML signature library rather
  than a hand-rolled implementation.

## Scenarios

### REQ-SAML-001: SPは署名証明書を取得できる
- ACTOR EndUser
- GIVEN SP が IdMagic tenant の SAML metadata または証明書ダウンロード URL を参照できる
- WHEN SP が証明書ダウンロード URL へ要求する
- THEN SP は active XmlFederationSigning 証明書を PEM で取得する
  - ALT federation signing credential が利用不能である → 証明書を返さずエラーを返す
- THEN 取得した証明書は同時点の SAML metadata に公開される証明書と一致する
- THEN rotation overlap 中の全 trust 証明書は SAML metadata から取得する

### REQ-SAML-002: SPは割当IdP profileだけを利用できる
- ACTOR EndUser
- GIVEN SPはprofile-aへ割り当てられている
- GIVEN profile-aとprofile-bは同一テナント内に存在する
- WHEN profile-aのSSO endpointへSPのAuthnRequestを送る
  - ALT 同じ要求をprofile-bのSSO endpointへ送る → SAMLResponseを発行せずSamlSignInRejectedを発行する
  - ALT profile-aのSSO URLと異なるDestinationを指定する → fail-closedで拒否する
- THEN Destination、SP Issuer、profile bindingが一体で検証される
- THEN profile-aのentityIDと署名資格情報でSAMLResponseが発行される

### REQ-SAML-003: 専用profileは固有metadataを公開する
- ACTOR EndUser
- GIVEN テナントにdefault profileとdedicated profileが存在する
- WHEN dedicated profileのmetadata URLを取得する
  - ALT 存在しないまたは別テナントのprofile IDを指定する → metadataと証明書を公開せずnot foundを返す
- THEN metadataはprofile固有entityID、SSO/SLO URL、署名証明書を公開する
- THEN default metadataとは異なる署名資格情報を公開する

### REQ-SAML-004: 管理者はSAML IdP profileを共有または専用で管理できる
- ACTOR TenantAdministrator
- GIVEN テナントには変更不能なdefault shared profileが存在する
- WHEN 管理者がread-onlyの連携エンドポイント画面から専用profile管理一覧とprofile詳細へ移動する
- THEN 専用profileの一覧と詳細が表示される
- WHEN 管理者が専用作成画面でshared profileを作成する
- THEN 複数SPからそのprofileを選択できる
- WHEN 管理者がprofile詳細から専用編集画面へ移り、追加profileのnameまたはmodeを変更する
- THEN 変更が保存される
- WHEN 管理者がdedicated profileを作成して1 SPへ割り当てる
  - ALT dedicated profileを別のSPへも割り当てる → bindingをInvalidRequestErrorで拒否する
- THEN dedicated profileのbindingが保存される
- WHEN 管理者が未使用の追加profileを削除する
  - ALT profileがSPから参照中またはdefaultである → 削除をconflictで拒否する
- THEN profileが削除される

### REQ-SAML-005: management API clientはSAML scope内のservice providerだけを操作できる
- ACTOR ManagementApiClient
- GIVEN client は対象 tenant の有効な API access token を提示している
- WHEN client が service provider の参照、登録、または削除を要求する
  - ALT saml:read だけで変更操作を要求する → 操作は AccessDeniedError で拒否される
  - ALT token の tenant と request tenant が一致しない → 操作は AccessDeniedError で拒否される
- THEN saml:read scope は service provider の参照だけを許可する
- THEN saml:write scope は service provider の登録または削除だけを許可する

### REQ-SAML-006: SAML SP initiated SSO succeeds
- ACTOR EndUser
- GIVEN subject は認証済みで対象 Application に割り当てられている
- GIVEN SP の entityID、ACS URL、Destination は登録済みである
- WHEN 登録済み SP の AuthnRequest を受信する
- THEN Version / IssueInstant / Issuer / ACS / Destination / binding / NameIDPolicy / subject assignment を検証する
  - ALT entityID、ACS、Destination、subject assignment のいずれかが不正である → SAMLResponse を発行しない → SamlSignInRejected を発行して fail-closed で拒否する
  - ALT AuthnRequest の解析または署名検証に失敗する → SamlSignInRejected を発行して protocol error を返す
  - ALT AuthnRequest の Version、IssueInstant、ProtocolBinding、ACS index、NameIDPolicy format が未対応または矛盾する → assertion を発行しない → 検証済み ACS が確定済みの場合だけ HTTP-POST の SAML protocol error を返す → それ以外は SamlSignInRejected を発行して fail-closed で拒否する
  - ALT IsPassive=true かつ利用可能な既存セッションがない → ログイン画面へ遷移しない → 検証済み ACS へ HTTP-POST の NoPassive SAML protocol response を返す
- THEN 署名済み SAMLResponse を ACS へ POST し RelayState を同値で返す
  - ALT 同じ tenant / SP / AuthnRequest ID に対する assertion がすでに発行済みである → assertion を発行しない → SamlSignInRejected を発行して fail-closed で拒否する
- THEN assertion と response の署名は request tenant の active XmlFederationSigning 鍵を使用する

### REQ-SAML-007: SAML rejects unregistered or mismatched request
- ACTOR EndUser
- GIVEN AuthnRequest の entityID、ACS URL、Destination、subject assignment のいずれかが不正である
- WHEN 不正な AuthnRequest を受信する
- THEN SAMLResponse を発行せず SamlSignInRejected を発行する

### REQ-SAML-008: SAML ForceAuthn redirects stale session
- ACTOR EndUser
- GIVEN ForceAuthn=true かつ認証時刻が再認証猶予より古い
- WHEN ForceAuthn=true の AuthnRequest を受信する
- THEN 古い認証コンテキストを検出する
- THEN ログインへリダイレクトする
