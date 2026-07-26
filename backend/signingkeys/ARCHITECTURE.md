---
context: signingkeys
updated_at: 2026-07-26
---

# Architecture: signingkeys

## Overview

The `SigningKeys` context owns tenant-scoped asymmetric key metadata, provider selection, rotation,
verification overlap, and archival. It exposes key material only through published signing and public-key
ports; protocol serialization remains in OAuth2, SAML, and WS-Federation adapters.

## Usage isolation

Every key lookup is scoped by both the request tenant and `KeyUsage`. Callers that do not select a usage
use `Signing`, preserving a small API for OAuth2/OIDC. XML protocol adapters explicitly select
`XmlFederationSigning`, so a JWT key can never be selected accidentally for an XML assertion.

The local, PostgreSQL, and Vault adapters all maintain one active key per tenant and usage. PostgreSQL
enforces the same invariant with a partial unique index. This compound scope exists because rotating a
SAML trust must not rotate every JWT verification key, and the reverse must also be true.

## XML federation credentials

An XML federation key carries a self-signed X.509 certificate containing its public key. The certificate
is public metadata; the private key follows the configured provider and never appears in an admin
response. Active keys sign new messages, while unexpired verifying certificates remain available to
SAML and WS-Federation metadata during the rotation overlap.

Local and database providers hold the private RSA key in process when signing. Vault Transit retains the
private key and implements `crypto.Signer`; it selects PSS for JWT requests and PKCS#1 v1.5 for XML
Signature and X.509 operations because those wire formats advertise RSA-SHA256 rather than RSA-PSS.

## Lifecycle

Keys are created lazily for the resolved tenant and usage. No default tenant receives an eager bootstrap
key. This keeps tenant creation uniform and avoids special state that cannot be explained by the
request-scoped lifecycle.

Rotation atomically demotes the old active key to verifying and gives it an overlap expiry. Public key
and certificate listing includes active and unexpired verifying records; archive removes expired records
from publication. These mechanics extend the durable per-tenant provider design established by
[ADR-075](../../decisions/ADR-075-per-tenant-signing-keys-and-key-provider.md).
