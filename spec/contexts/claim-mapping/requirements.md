# ClaimMapping Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-CLAIMMAPPING-001: ResolveEffectiveClaims
tenant の属性可視性 (visibility != Private) と reserved claim type の固定集合を
fail-closed floor として強制し、ClaimMappingPolicy と解決済み属性から NameID と IssuedClaim[] を
組み立てる。WS-Fed / SAML / OIDC の各 issuer が共有する唯一の claim 解決経路。
User の core field (user_id / email / name / given_name / family_name / preferred_username /
email_verified / roles) は UserAttributeDef に現れないため常に解決対象にする。user_id は User 集約の
識別子を指す protocol-neutral な内部属性キーであり、OIDC ID Token/UserInfo が実際に発行する wire
claim "sub" (RFC 7519 / OIDC Core が定める語彙) とは別物。custom attribute で
attribute_defs に無い key、または visibility=Private の key を source に持つ rule は floor で拒否する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| ClaimMappingPolicy | identity principal の属性を外部 application / relying party / service provider / client へ出すための属性解決と出力許可の規則。 | ClaimMappingPolicy, attribute release, claim mapping |
