# Authentication Glossary

| Term | Definition | Aliases |
|---|---|---|
| IdentityBroker | 外部の IdP による認証結果を検証し、テナント内のローカル User と安全に相関させて、LoginSession の発行へ引き渡す機能。 |  |
| ExternalIdentityProvider | IdMagic にとって上流の認証権威となる OIDC Provider または SAML Identity Provider。 | 上流 IdP, 外部 IdP |
| FederatedIdentity | テナント、プロバイダー、外部の不変な subject の組を、ローカルの User へ一意に結び付ける関連付け。 |  |
| JitProvisioning | 検証済みの外部クレームと、テナントが明示したポリシーおよびクレームの対応付けに基づき、初回のフェデレーションログインの途中でローカルの User を作成すること。 | JIT プロビジョニング |
| Totp | RFC 6238 に基づく時刻同期型のワンタイムパスワード。 | totp, otp |
| Webauthn | WebAuthn の公開鍵クレデンシャルによる認証。 | webauthn |
| RecoveryCode | TOTP や WebAuthn の認証要素を失ったときに使う、単回限りの控えの復旧コード。 | recovery_code |
| TrustedDevice | 本物の第二要素が成立した直後に本人が明示同意して記憶させたブラウザーを表す、ユーザー単位の資格情報。有効な間は、そのブラウザーからのログインで第二要素の提示を省略できる。 | 信頼済みデバイス, remember this device |
| SecurityNotification | アカウントに起きたセキュリティ上の変化 — 既知でない端末からのサインイン、資格情報や連絡先の変更、明示的なセッション失効、なりすましの開始 — を、その本人へ知らせる最大限努力のメール。 | セキュリティ通知 |
| EndUser | 認証済みまたは認証を試みる一般利用者。ログイン・MFA継続・パスワードリセットなど、認証が未完了の操作の主体を指す。 |  |
| ResourceOwner | OAuth2/OIDC 認可フローでリソースの所有者として認可判断を行う利用者。EndUser と同一人物を OAuth2 文脈で指す呼称。 |  |
