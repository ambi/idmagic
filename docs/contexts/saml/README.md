# Saml

SAML 2.0 IdP として、SP の信頼、IdP プロファイル、IdP メタデータ、AuthnRequest / Response、AssertionConsumerService、Single Logout を担う Bounded Context である。Web Browser SSO Profile に基づき、SP 起点と IdP 起点の SSO を提供する。

WS-Fed / WS-Trust とは、クレームの発行処理と XML 署名だけを共有する。プロトコルに依存しないクレームの対応付けは `ClaimMapping`、署名鍵のライフサイクルは `SigningKeys` が担う。

| 文書 | 内容 |
|---|---|
| [Saml の用語集](glossary.md) | この Context での語義 |
| [Saml の標準仕様](standards.md) | 採用する外部標準仕様 |
| [Saml の設計判断](decisions.md) | 設計判断 |
| [Saml の内部設計](internals.md) | 機構の説明 |
| [Saml のシナリオ](scenarios.feature.md) | 受け入れシナリオ |
