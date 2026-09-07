# ClaimMapping

プリンシパルの属性を外部の RP、SP、クライアントへ公開するポリシーと、プロトコルに依存しないクレームの組み立てを担う。属性の解決と公開可否の判定はここに集約し、OIDC の JSON クレーム、SAML の `AttributeStatement`、WS-Fed のクレーム URI への変換は各プロトコルの Context に委ねる。

| 文書 | 内容 |
|---|---|
| [ClaimMapping の用語集](glossary.md) | この Context での語義 |
| [ClaimMapping の設計判断](decisions.md) | 設計判断 |
| [ClaimMapping の内部設計](internals.md) | 機構の説明 |
| [ClaimMapping Scenarios](scenarios.feature.md) | 受け入れシナリオ |
