# Feature: アカウントのセルフサービスのシナリオ

## 対象の操作

### Rule: REQ-IDMANAGEMENT-002 API トークンの発行者は account スコープで自身の情報だけを操作できる

Primary actor: `SelfApiClient`

#### Example: EX-IDMANAGEMENT-002-01 通常経路

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントが概要、プロフィール、データエクスポート、またはプライマリメールアドレスの変更申請を要求する
- Then `account:read` スコープは、自身の概要、プロフィール、データエクスポートの参照だけを許可する
- Then `account:write` スコープは、自身のプロフィールとプライマリメールアドレスの変更申請だけを許可する

#### Example: EX-IDMANAGEMENT-002-02 `account:read` だけで変更操作を要求する

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントが概要、プロフィール、データエクスポート、またはプライマリメールアドレスの変更申請を要求する
- But `account:read` だけで変更操作を要求する
- Then 操作は AccessDeniedError で拒否される

#### Example: EX-IDMANAGEMENT-002-03 トークンのテナントまたは `user_id` が操作対象と一致しない

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントが概要、プロフィール、データエクスポート、またはプライマリメールアドレスの変更申請を要求する
- But トークンのテナントまたは `user_id` が操作対象と一致しない
- Then 操作は AccessDeniedError で拒否される

### Rule: REQ-IDMANAGEMENT-019 アカウント API は他人のリソースを返さない

Primary actor: `AuthenticatedSelf`

#### Example: EX-IDMANAGEMENT-019-01 通常経路

- Given ユーザー "alice" が認証済みである
- When ユーザー "alice" のアカウント概要を取得する
- Then レスポンスは "alice" 自身のデータだけを含み、ロールは含まない

## 入力

### Rule: REQ-IDMANAGEMENT-016 ユーザーは自分のプロフィール表示名を更新できる

Primary actor: `AuthenticatedSelf`

#### Example: EX-IDMANAGEMENT-016-01 通常経路

- Given ユーザー "alice" が認証済みでマイアカウントのプロフィールを開いている
- When ユーザー "alice" が表示名を更新する
- Then 更新後のプロフィールに新しい表示名が反映される
- Then `editable_by_user=false` の属性は更新できない

## 結果

### Rule: REQ-IDMANAGEMENT-018 ユーザーは自分のアカウントデータをエクスポートできる

Primary actor: `AuthenticatedSelf`

#### Example: EX-IDMANAGEMENT-018-01 通常経路

- Given ユーザー "alice" が認証済みでデータとプライバシー画面を開いている
- When ユーザー "alice" がアカウントデータをエクスポートする
- Then レスポンスに自分のプロフィールと同意の一覧が含まれる

## 拒否

### Rule: REQ-IDMANAGEMENT-003 メールアドレス確認画面は未認証でも CSRF 境界を確立できる

Primary actor: `EndUser`

#### Example: EX-IDMANAGEMENT-003-01 通常経路

- When EndUser がメールアドレス確認の文脈を取得する
- Then レスポンスに CSRF トークンと SameSite 属性を持つ Cookie が含まれる
- When EndUser がその CSRF トークンと Cookie を添えてメールアドレスの確認を送信する
- Then メールアドレスの確認が受理される

#### Example: EX-IDMANAGEMENT-003-02 CSRF トークンと Cookie が一致しない

- When EndUser がメールアドレス確認の文脈を取得する
- Then レスポンスに CSRF トークンと SameSite 属性を持つ Cookie が含まれる
- When EndUser がその CSRF トークンと Cookie を添えてメールアドレスの確認を送信する
- But CSRF トークンと Cookie が一致しない
- Then 確認は InvalidRequestError で拒否される

## 作用

### Rule: REQ-IDMANAGEMENT-017 ユーザーはメールアドレス変更を起票し確認リンクで確定できる

Primary actor: `AuthenticatedSelf`

#### Example: EX-IDMANAGEMENT-017-01 通常経路

- Given ユーザー "alice" が認証済みでメールアドレス画面を開いている
- When ユーザー "alice" が新しいメールアドレスへの変更を起票する
- Then 新アドレスへ確認リンクが送られる
- When ユーザー "alice" が確認リンクのトークンで変更を確定する
- Then プライマリメールアドレスが新しいアドレスへ更新される

#### Example: EX-IDMANAGEMENT-017-02 リンクを開くだけではトークンを消費しない

- Given ユーザー "alice" 宛に有効なメールアドレス変更の確認トークンが発行されている
- When ブラウザーまたはメールスキャナーが確認リンクを `GET` または `HEAD` で先読みする
- Then トークンは未使用のまま残り、プライマリメールアドレスは変わらない
- When ユーザー "alice" が同じトークンで変更を確定する
- Then プライマリメールアドレスが新しいアドレスへ更新される

#### Example: EX-IDMANAGEMENT-017-03 確定済みのトークンは再利用できない

- Given ユーザー "alice" が確認トークンでメールアドレス変更を確定済みである
- When ユーザー "alice" が同じトークンでもう一度確定する
- Then エラー "InvalidRequestError"
- And プライマリメールアドレスは一度目の確定結果のまま変わらない

#### Example: EX-IDMANAGEMENT-017-04 別の用途で発行されたトークンは受け付けない

- Given ユーザー "alice" 宛にメールアドレス変更ではない用途のアクショントークンが発行されている
- When ユーザー "alice" がそのトークンで変更を確定する
- Then エラー "InvalidRequestError"
- And レスポンスは無効なトークンに対するものと区別できない
- And プライマリメールアドレスは変わらず、そのトークンは未使用のまま残る

#### Example: EX-IDMANAGEMENT-017-05 期限切れのトークンは受け付けない

- Given ユーザー "alice" 宛のメールアドレス変更の確認トークンが期限切れである
- When ユーザー "alice" がそのトークンで変更を確定する
- Then エラー "InvalidRequestError"
- And プライマリメールアドレスは変わらない

#### Example: EX-IDMANAGEMENT-017-06 起票後に新アドレスが他のユーザーのものになっている

- Given ユーザー "alice" 宛に有効なメールアドレス変更の確認トークンが発行されている
- And 起票から確定までの間に別のユーザーが同じアドレスを自分のものとして確定している
- When ユーザー "alice" がそのトークンで変更を確定する
- Then エラー "ConflictError"
- And プライマリメールアドレスは変わらず、トークンは未使用のまま残る
