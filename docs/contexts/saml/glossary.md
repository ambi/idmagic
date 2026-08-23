# Saml Glossary

| Term | Definition | Aliases |
|---|---|---|
| Saml | SAML 2.0 IdP のプロトコル群。Web Browser SSO Profile に基づき、SP 起点と IdP 起点の SSO、IdP メタデータの公開、署名済み SAMLResponse の発行、Single Logout、SP ごとの AuthnRequest / LogoutRequest の署名検証を扱う。暗号化された Assertion、ECP、SAML SP、外部 IdP からのフェデレーションは初期範囲外とする。XML 署名と正規化は実績のあるライブラリに委ね、自前では実装しない。 | SAML, SAML2, SAML 2.0 |
| EndUser | SAML Web Browser SSO または Single Logout をブラウザで開始する利用者。 |  |
| SamlIdentityProviderProfile | テナント内の SAML IdP entityID、エンドポイント、XML 署名資格情報をまとめた信頼境界。`shared` プロファイルは複数の SP で共有でき、`dedicated` プロファイルは最大 1 つの SP にだけ割り当てられる。 | SAML IdP プロファイル, IdP プロファイル |
