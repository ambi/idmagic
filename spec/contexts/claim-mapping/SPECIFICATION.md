---
context: claim-mapping
updated_at: 2026-08-11
---

# ClaimMapping Specification

## Overview

identity principal の属性を外部 relying party / service provider / client へ出すための
release policy と protocol-neutral な claim projection を所有する。属性解決と出力許可を
共通化し、OIDC JSON claims、SAML AttributeStatement、WS-Fed claim URI への wire 変換は
各 protocol context が所有する。

The `ClaimMapping` context owns a single protocol-agnostic capability: turning resolved identity
attributes into the claims a federation protocol issues to a relying party. It exists as its own
context because claim issuance is a pure transformation independent of XML signing or transport,
and no existing context was a good fit — `OAuth2` is scoped to OIDC/OAuth, and folding WS-*/SAML
relying-party trust and assertion handling in with it would have bloated that context's
responsibility. Carving this out first let the fail-closed and attribute-minimization guarantees be
established with unit tests before the (heavier) XML-signature library decision was made.

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| ClaimMappingPolicy | identity principal の属性を外部 application / relying party / service provider / client へ出すための属性解決と出力許可の規則。 | ClaimMappingPolicy, attribute release, claim mapping |

## Design

### Internal Interfaces

#### ResolveEffectiveClaims
tenant の属性可視性 (visibility != Private) と reserved claim type の固定集合を
fail-closed floor として強制し、ClaimMappingPolicy と解決済み属性から NameID と IssuedClaim[] を
組み立てる。WS-Fed / SAML / OIDC の各 issuer が共有する唯一の claim 解決経路。
User の core field (user_id / email / name / given_name / family_name / preferred_username /
email_verified / roles) は UserAttributeDef に現れないため常に解決対象にする。user_id は User 集約の
識別子を指す protocol-neutral な内部属性キーであり、OIDC ID Token/UserInfo が実際に発行する wire
claim "sub" (RFC 7519 / OIDC Core が定める語彙) とは別物。custom attribute で
attribute_defs に無い key、または visibility=Private の key を source に持つ rule は floor で拒否する。

### Declarative claim-issuance engine

`ClaimMappingRule` declares an output claim type (a URI) and its source — a user attribute, a fixed
value, or NameID — rather than an AD-FS-style claim rule language; `ClaimMappingPolicy` bundles a
relying party's rule set together with a `NameIdConfiguration`. The engine takes a resolved
attribute map (already detached from the identity aggregate) and a policy, and returns
`IssuedClaim[]`. It is fail-closed on both ends: only claims explicitly named by a mapping rule are
ever emitted, so an unmapped attribute can never leak into a token, and a required rule whose source
attribute is missing causes issuance to be refused rather than emitting a partial claim set. WS-Fed,
WS-Trust, and SAML all call this same engine instead of each implementing their own claim assembly,
which keeps the fail-closed guarantee in one place instead of three.

### Design Decisions

- `ClaimMapping` is its own bounded context — separate from `OAuth2` and the XML federation
  protocols — because claim issuance is a pure, protocol-agnostic transformation independent of XML
  signing or transport, and folding WS-*/SAML relying-party trust into `OAuth2` would have bloated
  that context's responsibility
  ([ADR-059](../../../decisions/ADR-059-federation-bounded-context-and-claim-issuance.md)).
