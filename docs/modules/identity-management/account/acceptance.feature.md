# Feature: アカウントのセルフサービスの例

## Rule: REQ-IDMANAGEMENT-002 account スコープの API トークンは、発行者本人の情報への操作だけを許可する

### Example: EX-IDMANAGEMENT-002-01 通常経路

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントが概要、プロフィール、データエクスポート、またはプライマリメールアドレスの変更申請を要求する
- Then `account:read` スコープは、自身の概要、プロフィール、データエクスポートの参照だけを許可する
- Then `account:write` スコープは、自身のプロフィールとプライマリメールアドレスの変更申請だけを許可する

### Scenario Outline: 条件ごとの結果

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントが概要、プロフィール、データエクスポート、またはプライマリメールアドレスの変更申請を要求する
- But <condition>
- Then 操作は AccessDeniedError で拒否される

#### Examples:

  | example_id | condition |
  | --- | --- |
  | EX-IDMANAGEMENT-002-02 | `account:read` だけで変更操作を要求する |
  | EX-IDMANAGEMENT-002-03 | トークンのテナントまたは `user_id` が操作対象と一致しない |

## Rule: REQ-IDMANAGEMENT-019 アカウントの概要は、認証済みの本人のデータだけを、ロールを含めずに返す

### Example: EX-IDMANAGEMENT-019-01 通常経路

- Given ユーザー "alice" が認証済みである
- When ユーザー "alice" のアカウント概要を取得する
- Then レスポンスは "alice" 自身のデータだけを含み、ロールは含まない

## Rule: REQ-IDMANAGEMENT-016 本人によるプロフィールの更新は表示名を変え、本人が編集できない属性を拒否する

### Example: EX-IDMANAGEMENT-016-01 通常経路

- Given ユーザー "alice" が認証済みでマイアカウントのプロフィールを開いている
- When ユーザー "alice" が表示名を更新する
- Then 更新後のプロフィールに新しい表示名が反映される
- Then `editable_by_user=false` の属性は更新できない

## Rule: REQ-IDMANAGEMENT-051 本人によるプロフィールの更新は、属性をキーごとに併合する

### Example: EX-IDMANAGEMENT-051-01 一部の属性だけを送る更新

- Given ユーザー "alice" は本人が編集できる属性 `nickname` と、管理者が管理する属性 `employee_number` を持つ
- When "alice" が `nickname` だけを送って更新する
- Then `nickname` は新しい値になり、`employee_number` は元の値のまま残る

### Example: EX-IDMANAGEMENT-051-02 本人が編集できない属性を含む更新

- When ユーザー "alice" が `employee_number` を含めて更新する
- Then 更新は `attribute_not_editable` で拒否され、どの項目も変わらない

### Example: EX-IDMANAGEMENT-051-03 値の変わらない属性を送る更新

- Given ユーザー "alice" の `nickname` は "ally" である
- When "alice" が `nickname` を "ally" のまま送って更新する
- Then `UserUpdated` の `changed_fields` は ["attributes"] である

## Rule: REQ-IDMANAGEMENT-018 アカウントデータのエクスポートは、本人のプロフィールと同意の一覧を返す

### Example: EX-IDMANAGEMENT-018-01 通常経路

- Given ユーザー "alice" が認証済みでデータとプライバシー画面を開いている
- When ユーザー "alice" がアカウントデータをエクスポートする
- Then レスポンスに自分のプロフィールと同意の一覧が含まれる

## Rule: REQ-IDMANAGEMENT-052 本人に開示する属性は、本人が読める公開範囲のものだけである

### Example: EX-IDMANAGEMENT-052-01 管理者だけが読める属性

- Given ユーザー "alice" は公開範囲 `admin_readable` の属性 `employee_number` と、`self_readable` の属性 `nickname` を持つ
- When "alice" がアカウントデータをエクスポートする
- Then プロフィールの属性は `nickname` だけを含む

## Rule: REQ-IDMANAGEMENT-017 メールアドレスの変更の確定は、未使用で期限内のトークンだけでプライマリメールアドレスを置き換える

### Example: EX-IDMANAGEMENT-017-01 通常経路

- Given ユーザー "alice" が新しいメールアドレスへの変更を起票し、確認リンクを受け取っている
- When ユーザー "alice" が確認リンクのトークンで変更を確定する
- Then プライマリメールアドレスが新しいアドレスへ更新される

### Example: EX-IDMANAGEMENT-017-02 リンクを開くだけではトークンを消費しない

- Given ユーザー "alice" 宛に有効なメールアドレス変更の確認トークンが発行されている
- When ブラウザーまたはメールスキャナーが確認リンクを `GET` または `HEAD` で先読みする
- Then トークンは未使用のまま残り、プライマリメールアドレスは変わらない
- When ユーザー "alice" が同じトークンで変更を確定する
- Then プライマリメールアドレスが新しいアドレスへ更新される

### Example: EX-IDMANAGEMENT-017-03 確定済みのトークンは再利用できない

- Given ユーザー "alice" が確認トークンでメールアドレス変更を確定済みである
- When ユーザー "alice" が同じトークンでもう一度確定する
- Then エラー "InvalidRequestError"
- And プライマリメールアドレスは一度目の確定結果のまま変わらない

### Example: EX-IDMANAGEMENT-017-04 別の用途で発行されたトークンは受け付けない

- Given ユーザー "alice" 宛にメールアドレス変更ではない用途のアクショントークンが発行されている
- When ユーザー "alice" がそのトークンで変更を確定する
- Then エラー "InvalidRequestError"
- And レスポンスは無効なトークンに対するものと区別できない
- And プライマリメールアドレスは変わらず、そのトークンは未使用のまま残る

### Example: EX-IDMANAGEMENT-017-05 期限切れのトークンは受け付けない

- Given ユーザー "alice" 宛のメールアドレス変更の確認トークンが期限切れである
- When ユーザー "alice" がそのトークンで変更を確定する
- Then エラー "InvalidRequestError"
- And プライマリメールアドレスは変わらない

### Example: EX-IDMANAGEMENT-017-06 起票後に新アドレスが他のユーザーのものになっている

- Given ユーザー "alice" 宛に有効なメールアドレス変更の確認トークンが発行されている
- And 起票から確定までの間に別のユーザーが同じアドレスを自分のものとして確定している
- When ユーザー "alice" がそのトークンで変更を確定する
- Then エラー "ConflictError"
- And プライマリメールアドレスは変わらず、トークンは未使用のまま残る

## Rule: REQ-IDMANAGEMENT-053 メールアドレスの変更の起票は、ステップアップ認証の後に新しいアドレスへ確認のリンクを送る

### Example: EX-IDMANAGEMENT-053-01 表示名付きの新しいアドレス

- When ユーザー "alice" が新しいアドレス "Alice <Alice.New@Example.TEST>" で起票する
- Then 確認のメールは "alice.new@example.test" へ送られる

### Example: EX-IDMANAGEMENT-053-02 確認済みの現在のアドレスと同じアドレス

- Given ユーザー "alice" の確認済みのアドレスは "alice@example.test" である
- When "alice" が新しいアドレス "ALICE@example.test" で起票する
- Then 起票は `email_unchanged` で拒否される

### Example: EX-IDMANAGEMENT-053-03 確認のメールの送信の失敗

- Given メールの送信は失敗する
- When ユーザー "alice" が新しいアドレスで起票する
- Then 起票は 204 を返す

### Example: EX-IDMANAGEMENT-053-04 通常経路

- Given ユーザー "alice" が認証済みでメールアドレス画面を開いている
- When ユーザー "alice" が新しいメールアドレスへの変更を起票する
- Then 新アドレスへ確認リンクが送られる

## Rule: REQ-IDMANAGEMENT-054 メールアドレスの変更の確定は、アドレスを確認済みにして `EmailChanged` を発行する

### Example: EX-IDMANAGEMENT-054-01 必須操作 `verify_email` を持つ User の確定

- Given ユーザー "alice" には必須操作 `verify_email` が付いており、有効な確認トークンが発行されている
- When "alice" がそのトークンで確定する
- Then `email_verified` は `true` になり、`verify_email` は外れ、`EmailChanged` と `UserRequiredActionCleared` が発行される

### Example: EX-IDMANAGEMENT-054-02 存在しないトークン

- When ブラウザーが存在しないトークンで確定を送る
- Then 確定は 410 と `invalid_email_change_token` で拒否される

## Rule: REQ-IDMANAGEMENT-003 メールアドレスの確認画面は、未認証でも CSRF 境界を確立し、一致しない確定を拒否する

### Example: EX-IDMANAGEMENT-003-01 通常経路

- When EndUser がメールアドレス確認の文脈を取得する
- Then レスポンスに CSRF トークンと SameSite 属性を持つ Cookie が含まれる
- When EndUser がその CSRF トークンと Cookie を添えてメールアドレスの確認を送信する
- Then メールアドレスの確認が受理される

### Example: EX-IDMANAGEMENT-003-02 CSRF トークンと Cookie が一致しない

- When EndUser がメールアドレス確認の文脈を取得する
- Then レスポンスに CSRF トークンと SameSite 属性を持つ Cookie が含まれる
- When EndUser がその CSRF トークンと Cookie を添えてメールアドレスの確認を送信する
- But CSRF トークンと Cookie が一致しない
- Then 確認は InvalidRequestError で拒否される
