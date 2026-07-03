---
id: idp-wi-29-saml2-idp
title: "SAML 2.0 IdP を実装し、エンタープライズ SSO に対応する"
created_at: 2026-06-20
authors: ["tn"]
status: completed
risk: high
---
# Motivation
OIDC だけでは B2B / enterprise 販売の最低ラインを満たせないケースが多い。
Keycloak / Okta は SAML 2.0 IdP として SP-initiated / IdP-initiated SSO、
metadata、assertion 署名、attribute mapping、Single Logout を提供する。

本 WI は idmagic を SAML 2.0 IdP として振る舞えるようにする。

# Scope
- **decision**: 新規 ADR: SAML 2.0 IdP の対応範囲を定義する。初期対応は Web Browser SSO Profile、HTTP-Redirect / HTTP-POST binding、 signed response/assertion、metadata 公開、SP-initiated / IdP-initiated SSO、Single Logout とする。
- **scl**: SamlServiceProvider / SamlAuthnRequest / SamlResponse / SamlAssertion を追加する。, SamlMetadata / SamlSso / SamlSlo interface を追加する。, attribute mapping と NameID format を client metadata に追加する。
- **go**: 実績ある SAML library を選定し、XML signature / canonicalization を自前実装しない。, SAML SP 管理 API を admin clients と同じ tenant boundary で追加する。, `/saml/metadata`, `/saml/sso`, `/saml/slo` を realm 配下に公開する。, assertion 署名鍵は OAuth signing key と分離するか ADR で明確に決める。, RelayState、InResponseTo、AudienceRestriction、Recipient、Destination を検証する。
- **ui**: admin clients に SAML client/SP 種別を追加し、metadata import / export を提供する。, attribute mapping UI を追加する。
- **documentation**: README に SAML 対応範囲、metadata URL、SP 設定例を書く。

# Out of Scope
- SAML SP として外部 IdP に接続すること。これは inbound federation WI で扱う。
- SAML ECP。
- WS-Federation / WS-Trust。
- encrypted assertion の初期必須化。必要なら追加 WI とする。

# Verification
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- 手動: test SP から SP-initiated SSO を開始し、SAMLResponse の署名・Audience・NameID が正しいことを確認する。
- 手動: metadata import で登録した SP に対して SLO が成立することを確認する。

# Risk Notes
SAML は XML signature wrapping、canonicalization、metadata trust のリスクが大きい。
library 選定と conformance smoke を作業の前半に置き、手書き XML 処理を避ける。

# Completion
- **Completed At**: 2026-06-27
- **Summary**:
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
- **Verification Results**:
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
