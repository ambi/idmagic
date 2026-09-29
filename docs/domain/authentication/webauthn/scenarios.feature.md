# Feature: WebAuthnのシナリオ

## 利用

### Rule: REQ-AUTHENTICATION-006 ユーザーは WebAuthn でステップアップ認証のチャレンジを開始できる

Primary actor: `AuthenticatedSelf`

#### Example: EX-AUTHENTICATION-006-01 通常経路

- Given ユーザー "alice" が WebAuthn のクレデンシャルを登録済みで、認証済みセッションを持つ
- When ユーザー "alice" が正しい CSRF トークンでステップアップ認証の WebAuthn チャレンジを要求する
- Then レスポンスの `PublicKeyCredentialRequestOptions` は現在のセッションに束縛される

#### Example: EX-AUTHENTICATION-006-02 CSRF トークンが一致しない、または WebAuthn を利用できない

- Given ユーザー "alice" が WebAuthn のクレデンシャルを登録済みで、認証済みセッションを持つ
- When ユーザー "alice" が正しい CSRF トークンでステップアップ認証の WebAuthn チャレンジを要求する
- But CSRF トークンが一致しない、または WebAuthn を利用できない
- Then チャレンジは発行されず、要求を拒否する
