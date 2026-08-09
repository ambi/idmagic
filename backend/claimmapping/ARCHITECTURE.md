---
context: claimmapping
updated_at: 2026-08-09
---

# Architecture: claimmapping

## Overview

The `ClaimMapping` context owns a single protocol-agnostic capability: turning resolved identity
attributes into the claims a federation protocol issues to a relying party. It exists as its own
context because claim issuance is a pure transformation independent of XML signing or transport,
and no existing context was a good fit — `OAuth2` is scoped to OIDC/OAuth, and folding WS-*/SAML
relying-party trust and assertion handling in with it would have bloated that context's
responsibility. Carving this out first let the fail-closed and attribute-minimization guarantees be
established with unit tests before the (heavier) XML-signature library decision was made.

## Declarative claim-issuance engine

`ClaimMappingRule` declares an output claim type (a URI) and its source — a user attribute, a fixed
value, or NameID — rather than an AD-FS-style claim rule language; `ClaimMappingPolicy` bundles a
relying party's rule set together with a `NameIdConfiguration`. The engine takes a resolved
attribute map (already detached from the identity aggregate) and a policy, and returns
`IssuedClaim[]`. It is fail-closed on both ends: only claims explicitly named by a mapping rule are
ever emitted, so an unmapped attribute can never leak into a token, and a required rule whose source
attribute is missing causes issuance to be refused rather than emitting a partial claim set. WS-Fed,
WS-Trust, and SAML all call this same engine instead of each implementing their own claim assembly,
which keeps the fail-closed guarantee in one place instead of three.

## Design Decisions

- `ClaimMapping` is its own bounded context — separate from `OAuth2` and the XML federation
  protocols — because claim issuance is a pure, protocol-agnostic transformation independent of XML
  signing or transport, and folding WS-*/SAML relying-party trust into `OAuth2` would have bloated
  that context's responsibility
  ([ADR-059](../../decisions/ADR-059-federation-bounded-context-and-claim-issuance.md)).
