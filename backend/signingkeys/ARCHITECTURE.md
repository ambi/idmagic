---
context: signingkeys
updated_at: 2026-08-09
---

# Architecture: signingkeys

## Overview

The `SigningKeys` context owns tenant-scoped asymmetric key metadata, provider selection, rotation,
verification overlap, and archival. It exposes key material only through published signing and public-key
ports; protocol serialization remains in OAuth2, SAML, and WS-Federation adapters.

## Usage and scope isolation

Every key lookup is scoped by the request tenant, `KeyUsage`, and an opaque scope ID. Callers that do
not select a usage or scope use `Signing` and the default scope, preserving a small API for OAuth2/OIDC.
XML protocol adapters explicitly select `XmlFederationSigning`; SAML additionally selects its identity
provider profile ID as the scope. A JWT key can therefore never be selected for an XML assertion, and
one SAML profile cannot select another profile's credential.

The local, PostgreSQL, and Vault adapters all maintain one active key per tenant, usage, and scope.
PostgreSQL enforces the same invariant with a partial unique index, and Vault includes the scope in its
key-set identity. This compound key exists because rotating one SAML profile must not rotate another
profile or every JWT verification key.

## XML federation credentials

An XML federation key carries a self-signed X.509 certificate containing its public key. The certificate
is public metadata; the private key follows the configured provider and never appears in an admin
response. Active keys sign new messages, while unexpired verifying certificates remain available to
SAML and WS-Federation metadata during the rotation overlap.

Local and database providers hold the private RSA key in process when signing. Vault Transit retains the
private key and implements `crypto.Signer`; it selects PSS for JWT requests and PKCS#1 v1.5 for XML
Signature and X.509 operations because those wire formats advertise RSA-SHA256 rather than RSA-PSS.

## Lifecycle

Keys are created lazily for the resolved tenant, usage, and scope. No default tenant receives an eager
bootstrap key. This keeps tenant creation uniform and avoids special state that cannot be explained by
the request-scoped lifecycle.

A tenant's active signing key rotates at least every 90 days, driven by a scheduled operational job
independent of the manual, immediate `RotateTenantSigningKey` path. Rotation atomically demotes the
old active key to verifying and gives it an overlap expiry of at least 7 days, so JWKS consumers and
relying parties can still validate messages issued just before rotation. Key material reaching the
terminal `Archived` state is retained for 7 years to support verification of audit tokens signed by
already-retired keys; there is no separate purge/erase interface yet.

Public key and certificate listing includes active and unexpired verifying records; archive removes
expired records from publication.

Fail-closed behavior when a key provider is unreachable is not enforced inside `SigningKeys` — this
context has no signing or issuance interface of its own. It only surfaces the observable
`provider_healthy` signal (`TenantSigningKey.provider_healthy`, `ListTenantKeyHealth`); the actual
fail-closed enforcement point is OAuth2's `Token` issuance interface, which is where an unreachable
provider blocks new signatures.

## Design Decisions

- SAML IdP profiles are modeled as shareable (a profile can back more than one SP trust), with
  dedicated-use profiles expressed as the one-consumer case of the same model rather than a separate
  type ([ADR-145](../../decisions/ADR-145-shareable-saml-idp-profiles.md)).
- Signing keys are scoped per tenant behind a pluggable `KeyProvider`, rather than a shared
  system-wide key or a provider baked into each protocol adapter
  ([ADR-075](../../decisions/ADR-075-per-tenant-signing-keys-and-key-provider.md)).
- Key rotation cadence (90-day minimum), overlap expiry (7-day minimum), and archive retention
  (7 years) are fixed, normative policy values that live in this design record rather than being
  left as undocumented configuration
  ([ADR-108](../../decisions/ADR-108-signing-key-rotation-and-retention-policy-configuration.md)).
