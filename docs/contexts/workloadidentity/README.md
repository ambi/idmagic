# WorkloadIdentity

自律エージェントの実行環境に対するワークロードアイデンティティフェデレーションを担う。テナントが登録した外部アテステーション発行者の信頼設定 (`WorkloadTrustBundle`) と、外部の主体を既存の `Agent` に対応付ける `AgentWorkloadBinding` を管理する。主なアテステーション形式は、Kubernetes の投影 ServiceAccount トークン、クラウドのインスタンスアイデンティティトークン、SPIFFE JWT-SVID などの OIDC 互換 JWT である。

IdMagic は SPIRE のサーバーやエージェントを同梱・運用せず、外部アテステーションを検証する RP として動作する。検証済みのアテステーションは OAuth2 Token Exchange グラント (RFC 8693) の subject として渡し、長期シークレットを配布する専用の資格情報経路は設けない。

| 文書 | 内容 |
|---|---|
| [WorkloadIdentity の用語集](glossary.md) | この Context での語義 |
| [WorkloadIdentity の状態遷移](states.md) | 状態と遷移 |
| [WorkloadIdentity の設計判断](decisions.md) | 設計判断 |
| [WorkloadIdentity の内部設計](internals.md) | 機構の説明 |
| [WorkloadIdentity Scenarios](scenarios.feature.md) | 受け入れシナリオ |
