# OAuth2

OAuth 2.0 / OIDC プロトコル群の全責務を所有する。クライアントメタデータと Dynamic Client Registration、認可判断（認可、同意、認可コード、PAR、Device Authorization、RP-Initiated Logout）、トークンの発行とライフサイクル（アクセストークン、リフレッシュトークン、ID トークン、イントロスペクション、失効、UserInfo、Proof of Possession）、Discovery Metadata、Authorization Server Metadata、健全性報告をこの Bounded Context に集約する。

認可サーバーは `authorization/`、`client/`、`consent/`、`device/`、`token/` の機能単位で実装する。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [standards.md](standards.md) | 準拠する外部規範 |
| [states.md](states.md) | 状態と遷移 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
