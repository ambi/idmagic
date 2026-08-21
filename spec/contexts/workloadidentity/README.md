# WorkloadIdentity

自律エージェントの実行環境に対するワークロードアイデンティティフェデレーションを所有する。テナントが登録した外部アテステーション発行者の信頼設定 (`WorkloadTrustBundle`) と、外部の主体を既存の `Agent` に対応付ける `AgentWorkloadBinding` を管理する。主なアテステーション形式は、Kubernetes の投影 ServiceAccount トークン、クラウドのインスタンスアイデンティティトークン、SPIFFE JWT-SVID などの OIDC 互換 JWT である。

IdMagic は SPIRE のサーバーやエージェントを同梱・運用せず、外部アテステーションを検証する RP として動作する。検証済みのアテステーションは OAuth2 Token Exchange グラント (RFC 8693) の subject として渡し、長期シークレットを配布する専用の資格情報経路は設けない。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [states.md](states.md) | 状態と遷移 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
