---
context: wsfederation
updated_at: 2026-08-09
---

# Architecture: wsfederation

## Overview

The `WsFederation` context owns passive WS-Federation, active WS-Trust, relying-party trust, MEX, and
AD FS-compatible federation metadata. It shares protocol-neutral claim issuance with `ClaimMapping`
and XML assertion signing with the `tokens_saml` adapter.

## Tenant signing

Passive and active issuance resolve the request tenant's active `XmlFederationSigning` credential at
the point of issuance. Federation metadata publishes both the active certificate and unexpired
verifying certificates in each advertised role, so relying parties can survive a planned rotation.

The signer provider is backed by `SigningKeys`, not by process startup state. This keeps WS-Fed and
SAML on one XML credential lifecycle while preserving its separation from OAuth2/JWT keys.

## Federation metadata

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

## WS-Trust active STS scope

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

## Entra domain federation profile

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

## Design Decisions

- Federation metadata publication and claim-mapping ownership are scoped so `WsFederation` publishes
  discovery (issuer, endpoints, signing certificates) while `ClaimMapping` owns the shared claim
  release policy across WS-Fed, WS-Trust, and SAML
  ([ADR-062](../../decisions/ADR-062-federation-metadata-publication.md)).
- Active WS-Trust STS support is scoped to `/trust/usernamemixed` with `Issue` only, targeting
  Microsoft 365-style rich-client sign-in rather than general WS-Trust interoperability
  ([ADR-063](../../decisions/ADR-063-ws-trust-active-sts-scope.md)).
- The Microsoft Entra domain federation profile is a fixed relying-party preset (UPN/ImmutableID
  claim shape, sourceAnchor validation) rather than hand-authored claim configuration, so
  misconfiguration cannot surface opaquely on the Entra side
  ([ADR-065](../../decisions/ADR-065-entra-domain-federation-profile.md)).
