---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# SAML 2.0 IdP を実装し、エンタープライズ SSO に対応する

idmagic を SAML 2.0 IdP として実装した。Web Browser SSO Profile、HTTP-Redirect /
HTTP-POST binding、署名付き response/assertion、metadata 公開、SP-initiated /
IdP-initiated SSO、Single Logout に対応する。
- 公開エンドポイント: realm 配下に `/saml/metadata` (IdP metadata)、`/saml/sso`
(GET=Redirect / POST=POST binding、SAMLRequest なし entityID 指定で IdP-initiated)、
`/saml/slo` を公開 (internal/saml/adapters/http)。
- 署名 / XML: XML signature / canonicalization は自前実装せず
github.com/russellhaering/goxmldsig を採用。response/assertion を署名し、
Destination / Audience / NameID / RelayState を扱う。
- 集約 / 永続化: SamlServiceProvider を spec / port / tenant-scoped repository として
実装し、SP metadata・attribute (claim) mapping・NameID format・SLO URL を保持。
- admin / 編集面: SAML SP は Application の protocol binding として管理する
([[wi-69-application-catalog-aggregate-and-assignment]] / ADR-066・wi-76 で
Application 編集画面に entity_id / ACS / SLO / claim 規則 / 署名設定を畳んだ)。
- assertion 署名鍵は既存の signing key 基盤を用い、SP ごとの wire 設定と分離。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
