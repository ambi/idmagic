---
context: saml
updated_at: 2026-07-26
---

# Architecture: saml

## Overview

The `Saml` context owns SAML 2.0 IdP behavior: service-provider trust, AuthnRequest validation,
SSO/SLO use cases, response construction, and IdP metadata. Protocol-neutral claim projection remains
in `ClaimMapping`; XML signing is delegated through the shared federation signer provider.

## Tenant signing

Every request resolves its signer from the tenant context immediately before issuance. The provider
selects the tenant's active `XmlFederationSigning` credential and fails closed when the provider,
private signer, or X.509 certificate is unavailable. This request-scoped resolution prevents a
process-global certificate from crossing tenant boundaries.

IdP metadata publishes the active certificate plus every unexpired verifying certificate. New
assertions and responses use only the active credential, while the overlap lets service providers
validate messages issued immediately before rotation. XML parsing and canonicalization continue to use
the reviewed library selected in [ADR-060](../../decisions/ADR-060-xml-signature-library-and-saml-assertion-signing.md).
