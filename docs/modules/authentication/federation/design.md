# 外部 IdP との連携の設計

この文書は、[外部 IdP との連携](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

`federation/` はアイデンティティブローカーの機能である。
テナント単位の上流の OIDC と SAML の接続、外部 subject との関連付け、ログインの試行、プロトコルの検証、関連付けのポリシー、IdMagic のログインセッションへの引き渡しを担う。
IdManagement は JIT のために資格情報を持たないユーザーを作るが、ログインの時点の相関のポリシーは持たない。

プロトコルのアダプターは、上流の文書をすべて信頼できないものとして扱う。

| プロトコル | 検証 |
| --- | --- |
| OIDC | 保存済みの HTTPS の discovery のエンドポイント、PKCE 付きの Authorization Code、`state`、`nonce`、issuer と audience と時刻の検査、制限した JWK のアルゴリズム |
| SAML | 相関の取れた AuthnRequest、XML の署名、issuer、destination、audience、subject confirmation、時刻、再送。要求していない IdP 起点のレスポンスと暗号化した Assertion は範囲の外とする |

上流の ID Token の検証は `verifyUpstreamIDToken` として HTTP から切り離してあり、鍵の集合と時刻を引数で受け取る。
署名のアルゴリズムは RS256 に限り、kid が JWKS の鍵と一致し、issuer、audience、有効期限、nonce がすべて接続とログインの試行に一致することを求める。

クライアントシークレットは `secret_reference` だけで表し、実行時の解決器は `env:NAME` の参照を受け付ける。
生のシークレットも、上流のトークンやアサーションも、保存も返却もしない。
公開するプロバイダーの一覧に含めるのは、有効なプロバイダーの識別子、表示名、プロトコルだけである。

コールバックを検証した後、ブローカーは AMR に `federated` を含む通常の Authentication のセッションを作る。
OAuth の認可は `/authorize/resume` を通じて再開するので、アプリケーションのポリシー、必須の操作、同意、認可コードの発行は、ブローカーに複製せず `OAuth2` が担う。

## 検証

上流の ID Token の検証は、Go のネイティブのファジングの対象である。
ファジングは「こちらが署名していない ID Token を必ず拒否する」だけを表明し、正当なトークンの受理と、各主張の不一致による拒否は、表で駆動するテストが押さえる。
