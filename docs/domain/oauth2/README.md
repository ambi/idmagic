# OAuth2

OAuth 2.0 / OIDC プロトコル群の全責務を担う。クライアントメタデータと Dynamic Client Registration、認可判断（認可、同意、認可コード、PAR、Device Authorization、RP-Initiated Logout）、トークンの発行とライフサイクル（アクセストークン、リフレッシュトークン、ID トークン、イントロスペクション、失効、UserInfo、Proof of Possession）、Discovery Metadata、Authorization Server Metadata、健全性報告をこの Bounded Context に集約する。

トークンに載せるクレームの決定は `ClaimMapping`、署名鍵のライフサイクルは `SigningKeys`、利用者を認証してセッションを保つのは `Authentication`、その `client_id` にそもそも到達してよいかという関門は `Application` が担う。この Context が受け持つのは、それらの結果をプロトコルの語彙で組み立てて外部へ返す部分である。

| 文書 | 内容 |
|---|---|
| [OAuth2 の用語集](glossary.md) | この Context での語義 |
| [OAuth2 の標準仕様](standards.md) | 採用する外部標準仕様 |
| [OAuth2 の状態遷移](states.md) | 状態と遷移 |
| [OAuth2 の設計判断](decisions.md) | 設計判断 |
| [OAuth2 の内部設計](internals.md) | 機構の説明 |
| [OAuth2 のシナリオ](scenarios.feature.md) | 受け入れシナリオ |
