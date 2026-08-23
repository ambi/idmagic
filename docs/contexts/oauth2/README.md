# OAuth2

OAuth 2.0 / OIDC プロトコル群の全責務を担う。クライアントメタデータと Dynamic Client Registration、認可判断（認可、同意、認可コード、PAR、Device Authorization、RP-Initiated Logout）、トークンの発行とライフサイクル（アクセストークン、リフレッシュトークン、ID トークン、イントロスペクション、失効、UserInfo、Proof of Possession）、Discovery Metadata、Authorization Server Metadata、健全性報告をこの Bounded Context に集約する。

トークンに載せるクレームの決定は `ClaimMapping`、署名鍵のライフサイクルは `SigningKeys`、利用者を認証してセッションを保つのは `Authentication`、その `client_id` にそもそも到達してよいかという関門は `Application` が担う。この Context が受け持つのは、それらの結果をプロトコルの語彙で組み立てて外部へ返す部分である。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [standards.md](standards.md) | 準拠する外部規範 |
| [states.md](states.md) | 状態と遷移 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
