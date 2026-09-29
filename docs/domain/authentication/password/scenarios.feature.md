# Feature: パスワードのシナリオ

## 有効性

### Rule: REQ-AUTHENTICATION-024 有効期限を過ぎたパスワードのユーザーは次回ログイン後にパスワード変更を強制される

Primary actor: `EndUser`

#### Example: EX-AUTHENTICATION-024-01 通常経路

- Given テナントのパスワードポリシーは `max_age_days=90` で、ポリシーの更新から 90 日以上が経過している
- And ユーザー "alice" の `password_changed_at` は 91 日前である
- When ユーザー "alice" が正しいパスワードでログインする
- Then ログイン自体は成功する
- Then ユーザー "alice" に必須操作 `update_password` が付与される
- Then ユーザー "alice" はパスワード変更画面へ誘導され、変更完了までフローを継続できない
- When ユーザー "alice" がポリシーを満たす新しいパスワードへ変更する
- Then `update_password` が解除され、"PasswordChanged" が発行される

#### Example: EX-AUTHENTICATION-024-02 `password_changed_at` が 89 日前である

- Given テナントのパスワードポリシーは `max_age_days=90` で、ポリシーの更新から 90 日以上が経過している
- And ユーザー "alice" の `password_changed_at` は 91 日前である
- When ユーザー "alice" が正しいパスワードでログインする
- But `password_changed_at` が 89 日前である
- Then ログインはそのまま完了し、`update_password` は付与されない

#### Example: EX-AUTHENTICATION-024-03 `max_age_days` が未設定である

- Given テナントのパスワードポリシーは `max_age_days=90` で、ポリシーの更新から 90 日以上が経過している
- And ユーザー "alice" の `password_changed_at` は 91 日前である
- When ユーザー "alice" が正しいパスワードでログインする
- But `max_age_days` が未設定である
- Then 経過日数によらず `update_password` は付与されない

#### Example: EX-AUTHENTICATION-024-04 ポリシーの更新から 90 日が経過していない

- Given テナントのパスワードポリシーは `max_age_days=90` で、ポリシーの更新から 90 日以上が経過している
- And ユーザー "alice" の `password_changed_at` は 91 日前である
- When ユーザー "alice" が正しいパスワードでログインする
- But ポリシーの更新から 90 日が経過していない
- Then 猶予期間内なので `update_password` は付与されない

#### Example: EX-AUTHENTICATION-024-05 ユーザーがパスワード資格情報を持たない (フェデレーションまたはパスワードレス)

- Given テナントのパスワードポリシーは `max_age_days=90` で、ポリシーの更新から 90 日以上が経過している
- And ユーザー "alice" の `password_changed_at` は 91 日前である
- When ユーザー "alice" が正しいパスワードでログインする
- But ユーザーがパスワード資格情報を持たない (フェデレーションまたはパスワードレス)
- Then `update_password` は付与されない

## 失効と変更

### Rule: REQ-AUTHENTICATION-008 パスワードリセットの要求は識別子と IP の組で流量制限される

Primary actor: `EndUser`

#### Example: EX-AUTHENTICATION-008-01 通常経路

- Given 未認証である
- When "alice" 宛のパスワードリセットを要求する
- Then ユーザーの存在にかかわらず 204 を返す
- Then "PasswordResetRequested" が発行される

#### Example: EX-AUTHENTICATION-008-02 同じ識別子と IP の組で、`EndpointRateLimitPolicy` の時間枠内の上限に達している

- Given 未認証である
- When "alice" 宛のパスワードリセットを要求する
- But 同じ識別子と IP の組で、`EndpointRateLimitPolicy` の時間枠内の上限に達している
- Then "alice" 宛のパスワードリセットを再度要求する
- And エラー "RateLimitedError"

### Rule: REQ-AUTHENTICATION-010 ユーザーは現在のパスワードを確認して新しいパスワードへ変更できる

Primary actor: `AuthenticatedSelf`

#### Example: EX-AUTHENTICATION-010-01 通常経路

- Given ユーザー "alice" が認証済みでパスワード変更画面を開いている
- When ユーザー "alice" が正しい現在のパスワードと新しいパスワードを送信する
- Then パスワードが変更され、`password_changed_at` が更新される
- Then "PasswordChanged" が発行される

#### Example: EX-AUTHENTICATION-010-02 新しいパスワードが 12 文字未満である

- Given ユーザー "alice" が認証済みでパスワード変更画面を開いている
- When ユーザー "alice" が正しい現在のパスワードと新しいパスワードを送信する
- But 新しいパスワードが 12 文字未満である
- Then ユーザー "alice" が 12 文字未満のパスワードを送信する
- And エラー "InvalidRequestError"

#### Example: EX-AUTHENTICATION-010-03 新しいパスワードが直近 5 件の履歴に一致する

- Given ユーザー "alice" が認証済みでパスワード変更画面を開いている
- When ユーザー "alice" が正しい現在のパスワードと新しいパスワードを送信する
- But 新しいパスワードが直近 5 件の履歴に一致する
- Then ユーザー "alice" が直近使用した過去のパスワードを新パスワードとして送信する
- And エラー "InvalidRequestError"

### Rule: REQ-AUTHENTICATION-016 ユーザーはメールのリセットリンクでパスワードを再設定する

Primary actor: `EndUser`

#### Example: EX-AUTHENTICATION-016-01 通常経路

- Given ユーザー "alice" 宛に有効なパスワードリセットトークンが発行されている
- When ユーザー "alice" がそのトークンと新しいパスワードを送信する
- Then パスワードが更新される
- When ユーザー "alice" が新しいパスワードをブラウザーのログイン API へ送信する
- Then ログインに成功する
- When EndUser が未登録のメールアドレスでパスワードリセットを要求する
- Then レスポンスは登録済みアドレスに対するものと区別できない
- When EndUser が登録済みのメールアドレスでパスワードリセットを要求する
- Then 登録済みアドレスへリセットリンクが送られる

#### Example: EX-AUTHENTICATION-016-02 トークンが期限切れまたは不正である

- Given ユーザー "alice" 宛に有効なパスワードリセットトークンが発行されている
- When ユーザー "alice" がそのトークンと新しいパスワードを送信する
- But トークンが期限切れまたは不正である
- Then 無効なパスワードリセットトークンで新しいパスワードを送信する
- And エラー "InvalidRequestError"
- And パスワードは変わらない

#### Example: EX-AUTHENTICATION-016-03 リンクを開くだけではトークンを消費しない

- Given ユーザー "alice" 宛に有効なパスワードリセットトークンが発行されている
- When ブラウザーまたはメールスキャナーがリセットリンクを `GET` または `HEAD` で先読みする
- Then トークンは未使用のまま残り、パスワードは変わらない
- When ユーザー "alice" が同じトークンと新しいパスワードを送信する
- Then パスワードが更新される

#### Example: EX-AUTHENTICATION-016-04 確定済みのトークンは再利用できない

- Given ユーザー "alice" が有効なパスワードリセットトークンでパスワードを更新済みである
- When ユーザー "alice" が同じトークンとさらに別のパスワードを送信する
- Then エラー "InvalidRequestError"
- And パスワードは一度目の更新結果のまま変わらない

#### Example: EX-AUTHENTICATION-016-05 別の用途で発行されたトークンは受け付けない

- Given ユーザー "alice" 宛にパスワード再設定ではない用途のアクショントークンが発行されている
- When ユーザー "alice" がそのトークンと新しいパスワードを送信する
- Then エラー "InvalidRequestError"
- And レスポンスは無効なトークンに対するものと区別できない
- And パスワードは変わらず、そのトークンは未使用のまま残る

#### Example: EX-AUTHENTICATION-016-06 新しいパスワードがパスワード規則に反する

- Given ユーザー "alice" 宛に有効なパスワードリセットトークンが発行されている
- When ユーザー "alice" がそのトークンとパスワード規則に反する新しいパスワードを送信する
- Then 違反した規則を伴うエラーが返る
- And パスワードは変わらず、トークンは未使用のまま残る
- When ユーザー "alice" が同じトークンと規則を満たす新しいパスワードを送信する
- Then パスワードが更新される
