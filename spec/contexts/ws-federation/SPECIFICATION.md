---
context: ws-federation
updated_at: 2026-08-11
---

# WsFederation Specification

## Overview

WS-Federation passive と WS-Trust active STS の trust、federationmetadata.xml、MEX、RST/RSTR、WS-Fed relying party binding を所有する。SAML assertion / claim release / signing key lifecycle の共有 capability には依存するが、SAML 2.0 SP binding は Saml context が所有する。

The `WsFederation` context owns passive WS-Federation, active WS-Trust, relying-party trust, MEX, and
AD FS-compatible federation metadata. It shares protocol-neutral claim issuance with `ClaimMapping`
and XML assertion signing with the `tokens_saml` adapter.

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| WsFederation | WS-Federation Passive Requestor Profile と WS-Trust active STS を組み合わせた WS-* protocol family。UsernameToken による Issue、WS-Addressing の MessageID / To / Action、Bearer SAML assertion 発行を扱う。WindowsTransport / Kerberos、silent sign-in は初期範囲外。 | WS-Federation, WS-Trust, WS-Fed |
| EndUser | WS-Federation passive sign-in / sign-out をブラウザで開始する利用者。 |  |
| SecurityTokenRequester | UsernameToken credential を提示して WS-Trust Issue を呼び出す active client。 |  |

## Standards

### Web Services Federation Language (WS-Federation) Version 1.2

1.2 — https://docs.oasis-open.org/wsfed/federation/v1.2/ws-federation.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WSFed-PassiveSignIn | required | MUST | wsignin1.0 で登録済み wtrealm と許可済み wreply にだけ token を返す |
| WSFed-SilentSignIn | excluded | MAY | silent sign-in / prompt=none 相当の無音認証 |

### WS-Trust 1.3

1.3 — https://docs.oasis-open.org/ws-sx/ws-trust/v1.3/ws-trust.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WSTrust13-IssueBearer | required | MUST | Issue 要求に対して Bearer SAML assertion を RSTR で返す |
| WSTrust13-WindowsTransport | excluded | MAY | WindowsTransport / Kerberos based active profile |

### Web Services Security UsernameToken Profile 1.1.1

1.1.1 — https://docs.oasis-open.org/wss-m/wss/v1.1.1/os/wss-UsernameTokenProfile-v1.1.1-os.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WSS-UsernameTokenPassword | required | MUST | WS-Trust active STS は UsernameToken username/password を認証する |

### Web Services Addressing 1.0 - Core

1.0 — https://www.w3.org/TR/ws-addr-core/

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WSAddressing-MessageIDToAction | required | MUST | MessageID は replay 防止、To は active STS endpoint、Action は Issue として検証する |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Internal Interfaces

#### WsFederationSignOut
WsFederationSignIn が所有する単一 passive HTTP endpoint 内の sign-out 意味契約。ローカルセッションを破棄する。wsignout1.0 は許可済み wreply へのリダイレクトまで行い、wsignoutcleanup1.0 は破棄のみで 200 を返す。
- Input invariant: input.wa == "wsignout1.0" || input.wa == "wsignoutcleanup1.0"
- Input invariant: input.wtrealm == null || wtrealm_registered(input.wtrealm, context.tenant_id)
- Input invariant: input.wreply == null || reply_url_allowed(input.wtrealm, input.wreply, context.tenant_id)
- Result invariant: local_session_cleared
- Result invariant: no_unregistered_redirect

### Tenant signing

Passive and active issuance resolve the request tenant's active `XmlFederationSigning` credential at
the point of issuance. Federation metadata publishes both the active certificate and unexpired
verifying certificates in each advertised role, so relying parties can survive a planned rotation.

The signer provider is backed by `SigningKeys`, not by process startup state. This keeps WS-Fed and
SAML on one XML credential lifecycle while preserving its separation from OAuth2/JWT keys.

### Federation metadata

The context publishes an AD FS-compatible `federationmetadata.xml` under each realm at
`/{realm}/federationmetadata/2007-06/federationmetadata.xml`, advertising the tenant issuer
(`/realms/default` for the default tenant) as entityID. This lets WS-Fed relying parties and
Microsoft Entra domain federation discover issuer, endpoints, and signing certificates without a
separate onboarding channel.

The `EntityDescriptor` carries both a `SecurityTokenServiceType` and an `ApplicationServiceType`
`RoleDescriptor`, advertising `PassiveRequestorEndpoint`, `SecurityTokenServiceEndpoint`,
`MetadataEndpoint`, and the signing `KeyDescriptor`. Signing certificates are published in WS-*'s
native X.509 form rather than reusing the OAuth/OIDC JWK shape — key usage, rotation, and overlap
stay `SigningKeys` responsibilities, and metadata only needs to advertise what WS-* consumers already
expect. `/{realm}/trust/mex` publishes the `usernamemixed` endpoint and its UsernameToken-required
policy as discovery for the active STS (see below); the RST/RSTR exchange itself is not part of
metadata.

Claim release stays declarative: `ClaimMappingPolicy` is shared across WS-Fed, WS-Trust, and (later)
SAML rather than adopting the AD FS claim rule language, which trades expressiveness for a validation
cost the mapped claim set doesn't need. Unmapped attributes are never emitted.

### WS-Trust active STS scope

Active WS-Trust support targets Microsoft 365-style rich-client sign-in rather than general
interoperability. SOAP, WS-Security, WS-Addressing, and SAML signing overlap enough that broad
binding coverage would materially raise replay and XML-wrapping risk, so the initial scope is
deliberately narrow.

`/trust/usernamemixed` is the only active STS endpoint and accepts WS-Trust 1.3 `Issue` only —
`Validate`, `Renew`, and `Cancel` are not implemented. Authentication is UsernameToken-only, verified
against the existing `UserRepository`, `PasswordHasher`, and `LoginAttemptThrottle`; Kerberos/IWA
`windowstransport` is out of scope and left to a separate slice.

WS-Addressing and WS-Security required elements (`MessageID`, `To`, `Action`, UsernameToken,
Timestamp, `AppliesTo`) are validated fail-closed: Timestamp rejects both expired and far-future
values, and `MessageID` is recorded in a short-lived replay store. `AppliesTo` must resolve to a
registered WS-Fed relying party — unregistered targets are rejected — and the issued assertion's
audience/recipient is bound to that RP, with claims issued through the RP's `ClaimMappingPolicy`.
This keeps replay and audience confusion from crossing relying-party boundaries. The RSTR returns a
signed SAML assertion over SOAP 1.2, defaulting to SAML 1.1 unless the RST explicitly requests
SAML 1.1 or SAML 2.0.

### Entra domain federation profile

An `EntraFederationProfile` is a WS-Federation relying-party preset: it captures domain, IssuerUri,
the sourceAnchor attribute, and passive/active/MEX endpoints, and upserts a `WsFedRelyingParty` whose
wtrealm/audience is that same IssuerUri. Presetting avoids hand-authored claim JSON, whose
misconfiguration surfaces opaquely on the Entra side and can't guarantee sourceAnchor stability or
uniqueness at setup time.

Required claims are fixed by the preset and fail closed: UPN is issued from `preferred_username` as
`http://schemas.xmlsoap.org/claims/UPN`; ImmutableID is derived from the normalized sourceAnchor
(`entra_immutable_id`) and placed in both the persistent NameID and
`http://schemas.xmlsoap.org/claims/nameidentifier`. sourceAnchor is validated both at profile setup
(rejecting missing, duplicate, or unconvertible values on existing users) and at issuance (rejecting
claim issuance if the target user's ImmutableID can't be derived).

GUID-shaped sourceAnchor values are base64-encoded using .NET `Guid.ToByteArray()` byte order — the
AD FS/Entra convention — before use as ImmutableID; values already in base64 pass through unchanged.
Getting this byte order wrong means Entra can't correlate the assertion back to the same on-prem
user, producing duplicate accounts or sign-in failures. The profile's default token type is SAML 1.1,
matching Entra/AD FS's WS-Fed default. Hybrid Azure AD Join device registration (`windowstransport`
plus computer-account Kerberos) is explicitly out of scope; setup guides tenants toward managed/PHS
or a coexisting AD FS deployment instead.

### Design Decisions

- Federation metadata publication and claim-mapping ownership are scoped so `WsFederation` publishes
  discovery (issuer, endpoints, signing certificates) while `ClaimMapping` owns the shared claim
  release policy across WS-Fed, WS-Trust, and SAML
  ([ADR-062](../../../decisions/ADR-062-federation-metadata-publication.md)).
- Active WS-Trust STS support is scoped to `/trust/usernamemixed` with `Issue` only, targeting
  Microsoft 365-style rich-client sign-in rather than general WS-Trust interoperability
  ([ADR-063](../../../decisions/ADR-063-ws-trust-active-sts-scope.md)).
- The Microsoft Entra domain federation profile is a fixed relying-party preset (UPN/ImmutableID
  claim shape, sourceAnchor validation) rather than hand-authored claim configuration, so
  misconfiguration cannot surface opaquely on the Entra side
  ([ADR-065](../../../decisions/ADR-065-entra-domain-federation-profile.md)).

## Scenarios

### REQ-WSFEDERATION-001: management API clientはWS-Fed scope内のtrustだけを操作できる
- ACTOR ManagementApiClient
- GIVEN client は対象 tenant の有効な API access token を提示している
- WHEN client が relying party または Entra federation の操作を要求する
  - ALT wsfed:read だけで変更操作を要求する → 操作は AccessDeniedError で拒否される
  - ALT token の tenant と request tenant が一致しない → 操作は AccessDeniedError で拒否される
- THEN wsfed:read scope は relying party の参照だけを許可する
- THEN wsfed:write scope は relying party と Entra federation の変更だけを許可する

### REQ-WSFEDERATION-002: WS-Federation passive sign-in succeeds
- ACTOR EndUser
- GIVEN wtrealm と wreply は登録済みで subject は対象 Application に割り当てられている
- WHEN 登録済み RP の wsignin1.0 を受信する
- THEN wtrealm / wreply / wfresh / subject assignment を検証する
  - ALT wfresh より認証が古い → token を発行せず再認証へ誘導する
  - ALT wtrealm、wreply、wauth、subject assignment のいずれかが不正である → WsFedSignInRejected を発行して fail-closed で拒否する
- THEN 署名済み assertion を RSTR form で返し wctx を同値で往復する

### REQ-WSFEDERATION-003: WS-Federation passive sign-in rejects untrusted target
- ACTOR EndUser
- GIVEN wtrealm が未登録、wreply が許可外、または subject が未割当である
- WHEN 不正な wsignin1.0 を受信する
- THEN token を発行せず WsFedSignInRejected を発行する

### REQ-WSFEDERATION-004: WS-Trust Issue succeeds
- ACTOR SecurityTokenRequester
- GIVEN UsernameToken、MessageID、Timestamp、To、Action、RequestType、KeyType、AppliesTo が有効である
- WHEN WS-Trust Issue RST を受信する
- THEN UsernameToken と RST の閉集合条件を検証する
  - ALT MessageID が assertion lifetime 内に再利用されている → WsTrustTokenRejected を発行して protocol error を返す
  - ALT UsernameToken credential が不正である → AccessDeniedError を返し token を発行しない
- THEN RSTR を返す

### REQ-WSFEDERATION-005: WS-Trust Issue rejects invalid envelope
- ACTOR SecurityTokenRequester
- GIVEN RST の To、MessageID、AppliesTo、Action、RequestType、KeyType のいずれかが不正である
- WHEN 不正な RST を受信する
- THEN WsTrustTokenRejected を emit して 400 または 401 を返す
