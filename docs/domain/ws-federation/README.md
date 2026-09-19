# WsFederation

受動的な WS-Federation と能動的な WS-Trust STS について、RP の信頼関係、AD FS 互換の `federationmetadata.xml`、MEX、RST と RSTR を担う。

プロトコルに依存しないクレーム発行は `ClaimMapping`、XML Assertion の署名は `tokens_saml` アダプター、署名鍵のライフサイクルは `SigningKeys` が担う。SAML 2.0 SP との信頼関係は `Saml` Context の責務である。

| 文書 | 内容 |
|---|---|
| [WsFederation の用語集](glossary.md) | この Context での語義 |
| [WsFederation の標準仕様](standards.md) | 採用する外部標準仕様 |
| [WsFederation の設計判断](decisions.md) | 設計判断 |
| [WsFederation の内部設計](internals.md) | 機構の説明 |
| [WsFederation のシナリオ](scenarios.feature.md) | 受け入れシナリオ |
