# Saml

SAML 2.0 IdP として、SP の信頼、IdP プロファイル、IdP メタデータ、AuthnRequest / Response、AssertionConsumerService、Single Logout を所有する Bounded Context である。Web Browser SSO Profile に基づき、SP 起点と IdP 起点の SSO を提供する。

WS-Fed / WS-Trust とは、クレームの発行処理と XML 署名だけを共有する。プロトコルに依存しないクレームの対応付けは `ClaimMapping`、署名鍵のライフサイクルは `SigningKeys` が所有する。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [standards.md](standards.md) | 準拠する外部規範 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
