# Feature: 通知テンプレートのシナリオ

## 結果

### Rule: REQ-TENANCY-015 日本語ロケールのユーザーには日本語のパスワードリセットメールが届く

Primary actor: `EndUser`

#### Example: EX-TENANCY-015-01 通常経路

- Given 利用者 "hanako" は locale 属性が "ja"、検証済みメールアドレスを持つ
- And テナントは通知テンプレートを一度も上書きしていない
- When "hanako" が RequestPasswordReset を実行する
- Then 件名と本文が組込みデフォルトの ja テンプレートで描画されたメールが届く
- Then メールはプレーンテキストと HTML の両方を含む
- Then 本文のリセットリンクはリクエストの発行元 URL から組み立てられており、開くとパスワード再設定画面に到達する

#### Example: EX-TENANCY-015-02 "hanako" の locale 属性が未設定で、テナントの default_locale が "ja" である

- Given 利用者 "hanako" は locale 属性が "ja"、検証済みメールアドレスを持つ
- And テナントは通知テンプレートを一度も上書きしていない
- When "hanako" が RequestPasswordReset を実行する
- But "hanako" の locale 属性が未設定で、テナントの default_locale が "ja" である
- Then テナントデフォルトの "ja" が採用され、日本語のメールが届く

#### Example: EX-TENANCY-015-03 "hanako" の locale 属性が未設定で、テナントの default_locale も未設定である

- Given 利用者 "hanako" は locale 属性が "ja"、検証済みメールアドレスを持つ
- And テナントは通知テンプレートを一度も上書きしていない
- When "hanako" が RequestPasswordReset を実行する
- But "hanako" の locale 属性が未設定で、テナントの default_locale も未設定である
- Then システムデフォルト locale が採用され、その locale のメールが届く

#### Example: EX-TENANCY-015-04 "hanako" の locale 属性がカタログに同梱翻訳の無い locale である

- Given 利用者 "hanako" は locale 属性が "ja"、検証済みメールアドレスを持つ
- And テナントは通知テンプレートを一度も上書きしていない
- When "hanako" が RequestPasswordReset を実行する
- But "hanako" の locale 属性がカタログに同梱翻訳の無い locale である
- Then 未対応 locale は飛ばして次の段が採用され、空の本文は送られない

### Rule: REQ-TENANCY-016 テナントの通知テンプレート上書きは組込みデフォルトより優先される

Primary actor: `TenantAdministrator`

#### Example: EX-TENANCY-016-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が ListNotificationTemplates を呼び出す
- Then 全 template_key × 全サポート locale が customized=false で一覧される
- When "operator" が PasswordReset / ja の件名と本文を上書きして UpdateNotificationTemplate を実行する
- Then NotificationTemplateUpdated が発行され、当該テンプレートは customized=true になる
- Then 以後 ja の利用者に届くパスワードリセットメールは上書きした件名と本文で送られる
- When "operator" が ResetNotificationTemplate を実行する
- Then NotificationTemplateReset が発行され、当該テンプレートは組込みデフォルトに戻る

#### Example: EX-TENANCY-016-02 上書きしていない en の利用者にメールが送られる

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が ListNotificationTemplates を呼び出す
- Then 全 template_key × 全サポート locale が customized=false で一覧される
- When "operator" が PasswordReset / ja の件名と本文を上書きして UpdateNotificationTemplate を実行する
- Then NotificationTemplateUpdated が発行され、当該テンプレートは customized=true になる
- Then 上書きしていない en の利用者にメールが送られる
- Then en は組込みデフォルトのまま描画され、ja の上書きは影響しない

#### Example: EX-TENANCY-016-03 上書きが存在しないテンプレートに ResetNotificationTemplate を実行する

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が ListNotificationTemplates を呼び出す
- Then 全 template_key × 全サポート locale が customized=false で一覧される
- When "operator" が PasswordReset / ja の件名と本文を上書きして UpdateNotificationTemplate を実行する
- Then NotificationTemplateUpdated が発行され、当該テンプレートは customized=true になる
- Then 以後 ja の利用者に届くパスワードリセットメールは上書きした件名と本文で送られる
- When "operator" が ResetNotificationTemplate を実行する
- But 上書きが存在しないテンプレートに ResetNotificationTemplate を実行する
- Then 冪等に成功し、組込みデフォルトのままとなる

## 拒否

### Rule: REQ-TENANCY-017 許可されていない差し込み変数を含むテンプレート上書きは保存時に拒否される

Primary actor: `TenantAdministrator`

#### Example: EX-TENANCY-017-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PasswordReset の本文に許可集合外の変数 `{{password}}` を書いて保存を試みる
- Then InvalidRequestError で拒否され、上書きは保存されない
- Then 以後も利用者には組込みデフォルトのリセットメールが届き、リンクが欠けたメールは配られない

#### Example: EX-TENANCY-017-02 "operator" が HTML 本文を空にしてテキスト本文だけを保存しようとする

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PasswordReset の本文に許可集合外の変数 `{{password}}` を書いて保存を試みる
- But "operator" が HTML 本文を空にしてテキスト本文だけを保存しようとする
- Then InvalidRequestError で拒否され、片方だけの上書きは作られない

#### Example: EX-TENANCY-017-03 "operator" がカタログに無い locale を指定して保存を試みる

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PasswordReset の本文に許可集合外の変数 `{{password}}` を書いて保存を試みる
- But "operator" がカタログに無い locale を指定して保存を試みる
- Then InvalidRequestError で拒否される

#### Example: EX-TENANCY-017-04 "operator" が差出人メールアドレスの上書きを試みる

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PasswordReset の本文に許可集合外の変数 `{{password}}` を書いて保存を試みる
- But "operator" が差出人メールアドレスの上書きを試みる
- Then アドレスを上書きする未知のプロパティは無視され、上書きできるのは表示名だけである

## 作用

### Rule: REQ-TENANCY-018 プレビューは実送信せずテスト送信は操作者本人にしか届かない

Primary actor: `TenantAdministrator`

#### Example: EX-TENANCY-018-01 通常経路

- Given admin ロールを持つ "operator" が検証済みメールアドレスを持ち認証済みである
- When "operator" が保存前の文面で PreviewNotificationTemplate を呼び出す
- Then サンプル値を展開した件名・テキスト本文・HTML 本文が返る
- Then メールは送信されず、上書きも保存されない
- When "operator" が SendTestNotification を呼び出す
- Then 宛先は "operator" 自身のアドレスに固定され、EmailSent が発行される

#### Example: EX-TENANCY-018-02 文面に利用者名などの差し込み値が含まれる

- Given admin ロールを持つ "operator" が検証済みメールアドレスを持ち認証済みである
- When "operator" が保存前の文面で PreviewNotificationTemplate を呼び出す
- Then 文面に利用者名などの差し込み値が含まれる
- Then HTML 側の差し込み値はエスケープされて描画され、タグとして解釈されない

#### Example: EX-TENANCY-018-03 リクエストで別の宛先を指定しようとする

- Given admin ロールを持つ "operator" が検証済みメールアドレスを持ち認証済みである
- When "operator" が保存前の文面で PreviewNotificationTemplate を呼び出す
- Then サンプル値を展開した件名・テキスト本文・HTML 本文が返る
- Then メールは送信されず、上書きも保存されない
- When "operator" が SendTestNotification を呼び出す
- But リクエストで別の宛先を指定しようとする
- Then 宛先の指定手段は提供されず、常に操作者本人へ送られる

#### Example: EX-TENANCY-018-04 操作者が検証済みメールアドレスを持たない

- Given admin ロールを持つ "operator" が検証済みメールアドレスを持ち認証済みである
- When "operator" が保存前の文面で PreviewNotificationTemplate を呼び出す
- Then サンプル値を展開した件名・テキスト本文・HTML 本文が返る
- Then メールは送信されず、上書きも保存されない
- When "operator" が SendTestNotification を呼び出す
- But 操作者が検証済みメールアドレスを持たない
- Then InvalidRequestError で拒否され、メールは送信されない
