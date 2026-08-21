# ApiTokens

管理 API と SCIM API の認証に使うテナント単位の API アクセストークン (`idmagic_pat_` 接頭辞) について、発行、失効、一覧、およびスコープの語彙を所有する。トークンに付与されたスコープ集合は認証時に解決され、記録を所有する各 Context が操作の認可に使う。SCIM を含む各 API のエンドポイント自体は所有せず、トークンとスコープの語彙という横断的な認証基盤だけを提供する。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [standards.md](standards.md) | 準拠する外部規範 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
