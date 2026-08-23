# WsFederation

受動的な WS-Federation と能動的な WS-Trust STS について、RP の信頼関係、AD FS 互換の `federationmetadata.xml`、MEX、RST と RSTR を担う。

プロトコルに依存しないクレーム発行は `ClaimMapping`、XML Assertion の署名は `tokens_saml` アダプター、署名鍵のライフサイクルは `SigningKeys` が担う。SAML 2.0 SP との信頼関係は `Saml` Context の責務である。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [standards.md](standards.md) | 準拠する外部規範 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
