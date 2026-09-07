# ApiTokens

管理 API と SCIM API の認証に使うテナント単位の API アクセストークン (`idmagic_pat_` 接頭辞) について、発行、失効、一覧、およびスコープの語彙を担う。トークンに付与されたスコープ集合は認証時に解決され、記録の正を持つ各 Context が操作の認可に使う。SCIM を含む各 API のエンドポイントそのものは扱わず、トークンとスコープの語彙という横断的な認証基盤だけを提供する。

| 文書 | 内容 |
|---|---|
| [ApiTokens の用語集](glossary.md) | この Context での語義 |
| [ApiTokens の採用規範](standards.md) | 準拠する外部規範 |
| [ApiTokens の設計判断](decisions.md) | 設計判断 |
| [ApiTokens の内部設計](internals.md) | 機構の説明 |
| [ApiTokens Scenarios](scenarios.feature.md) | 受け入れシナリオ |
