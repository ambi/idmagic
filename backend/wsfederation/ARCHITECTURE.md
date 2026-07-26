---
context: wsfederation
updated_at: 2026-07-26
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
SAML on one XML credential lifecycle while preserving its separation from OAuth2/JWT keys, consistent
with the metadata boundary in
[ADR-062](../../decisions/ADR-062-federation-metadata-publication.md).
