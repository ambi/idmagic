# Feature: TOTPのシナリオ

## 生成

### Rule: REQ-AUTHENTICATION-011 ユーザーは TOTP 認証要素を登録して有効化できる

Primary actor: `AuthenticatedSelf`

#### Example: EX-AUTHENTICATION-011-01 通常経路

- Given ユーザー "alice" が認証済みでセキュリティ画面を開いている
- When ユーザー "alice" が TOTP 登録を開始する
- Then レスポンスにシークレットとアカウント名が含まれる
- When ユーザー "alice" がそのシークレットに対する正しいコードで登録を確認する
- Then セキュリティ概要の MFA 状態が登録済みになる
- Then "MfaFactorEnrolled" が発行される

## 利用

### Rule: REQ-AUTHENTICATION-017 TOTP が必須のユーザーは正しいコードで認証を継続できる

Primary actor: `EndUser`

#### Example: EX-AUTHENTICATION-017-01 通常経路

- Given TOTP 認証要素が登録された `authentication_pending` の LoginSession が存在する
- When ブラウザーの TOTP API に正しいコードを送信する
- Then 認証が成立し認可フローが継続する
- Then "UserAuthenticated" が発行される

#### Example: EX-AUTHENTICATION-017-02 誤った TOTP コードを送信する

- Given TOTP 認証要素が登録された `authentication_pending` の LoginSession が存在する
- When ブラウザーの TOTP API に正しいコードを送信する
- But 誤った TOTP コードを送信する
- Then ブラウザーの TOTP API に誤ったコードを送信する
- And エラー "InvalidRequestError"
- And LoginSession は `authentication_pending` のままである

## 失効と変更

### Rule: REQ-AUTHENTICATION-012 ユーザーはステップアップ再認証のうえで TOTP 認証要素を解除する

Primary actor: `AuthenticatedSelf`

#### Example: EX-AUTHENTICATION-012-01 通常経路

- Given ユーザー "alice" が登録済みの TOTP 認証要素を持ち認証済みである
- When ユーザー "alice" がステップアップ認証を成立させ現在の TOTP コードで解除する
- Then TOTP 認証要素が解除される
- Then "MfaFactorRemoved" が発行される

#### Example: EX-AUTHENTICATION-012-02 ステップアップ認証なしで解除を試みる

- Given ユーザー "alice" が登録済みの TOTP 認証要素を持ち認証済みである
- When ユーザー "alice" がステップアップ認証を成立させ現在の TOTP コードで解除する
- But ステップアップ認証なしで解除を試みる
- Then ユーザー "alice" がステップアップ認証なしで TOTP 認証要素の解除を試みる
- And ステップアップ認証による再認証が要求される
