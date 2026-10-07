# WI-35451: 残りの Context の仕様を要件の形式で書き直す

作業項目は `wi-35451-rewrite-remaining-context-specifications-in-the-requirement-format` である。

WI-35451 は、IdManagement と Tenancy 以外の 19 の Context の機能仕様を、検査する日本語の EARS の要件文で書き直し、すべての状態機械に状態遷移表（マトリクス形式）を加える。
製品の振る舞いは変えない。

これまで実装だけが守っていた次の振る舞いを、要件として約束する。
依存してよい境界が、この範囲だけ広がる。

| 振る舞い | 規範上の条件 |
| --- | --- |
| WS-Federation のパッシブサインアウトは、ローカルのセッションを破棄し、登録した返信先にだけリダイレクトする | [REQ-WSFEDERATION-006](../../domain/ws-federation/passive-sign-in/README.md) |
| SAML の Single Logout | [REQ-SAML-009](../../domain/saml/sso/README.md) |
| Application のカテゴリの管理 | [REQ-APPLICATION-015](../../domain/application/catalog/README.md) |
| WebAuthn の資格情報の登録と、ステップアップ認証のうえでの解除 | [REQ-AUTHENTICATION-038](../../domain/authentication/webauthn/README.md) |
| ステップアップ認証の開始と完了 | [REQ-AUTHENTICATION-039](../../domain/authentication/account-portal/README.md) |
| 認可詳細の種類と MCP のリソースサーバーの管理 API | [REQ-OAUTH2-051、REQ-OAUTH2-052](../../domain/oauth2/admin-access/README.md) |
| `authorization_details` の要求と明示の同意、リソースの指定による audience の限定 | [REQ-OAUTH2-053、REQ-OAUTH2-054](../../domain/oauth2/authorization/README.md) |

状態遷移の表は、実装が実際に通る遷移に合わせて改めた。
たとえば、撤回した同意は利用者の再同意で `Granted` に戻り、上流の IdP の接続は信頼の根拠を更新すると `Disabled` に戻る。
