# Feature: サインイン履歴のシナリオ

## 結果

### Rule: REQ-AUTHENTICATION-014 ユーザーは自分のサインイン履歴を確認できる

Primary actor: `AuthenticatedSelf`

#### Example: EX-AUTHENTICATION-014-01 通常経路

- Given ユーザー "alice" が認証済みでアクティビティ画面を開いている
- When ユーザー "alice" が自分のサインイン履歴を取得する
- Then レスポンスに自分のサインインイベントだけが含まれる
- Then 第二要素を使ったサインインは、`pwd` と第二要素の `amr` を持つ完了後の `UserAuthenticated` として表示される

#### Example: EX-AUTHENTICATION-014-02 認証手段に WebAuthn が含まれる

- Given ユーザー "alice" が認証済みでアクティビティ画面を開いている
- When ユーザー "alice" が自分のサインイン履歴を取得する
- But 認証手段に WebAuthn が含まれる
- Then UI は `webauthn` という技術名ではなく「パスキー」と表示する
