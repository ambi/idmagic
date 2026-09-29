# Feature: セキュリティ通知のシナリオ

## 生成

### Rule: REQ-AUTHENTICATION-030 既知でない端末からのサインインだけがセキュリティ通知を生む

Primary actor: `EndUser`

#### Example: EX-AUTHENTICATION-030-01 通常経路

- Given ユーザー "alice" は検証済みのメールアドレスを持つ
- And ユーザー "alice" はこれまで一度もサインインしていないブラウザーを使っている
- When ユーザー "alice" がそのブラウザーで認証に成功する
- Then そのブラウザーは既知の端末として記録される
- Then "alice" の検証済みアドレスへセキュリティ通知が送られ、"AccountSecurityNotificationSent" が発行される
- When ユーザー "alice" が同じブラウザーで再び認証に成功する
- Then 通知は送られず、その端末の最終利用時刻だけが更新される

#### Example: EX-AUTHENTICATION-030-02 "alice" が検証済みのメールアドレスを持たない

- Given ユーザー "alice" は検証済みのメールアドレスを持つ
- And ユーザー "alice" はこれまで一度もサインインしていないブラウザーを使っている
- When ユーザー "alice" がそのブラウザーで認証に成功する
- Then そのブラウザーは既知の端末として記録される
- Then "alice" が検証済みのメールアドレスを持たない
- Then 通知は送られず、認証は成功したままである

### Rule: REQ-AUTHENTICATION-031 資格情報の変更は本人へ通知され、通知の失敗は変更を巻き戻さない

Primary actor: `AuthenticatedSelf`

#### Example: EX-AUTHENTICATION-031-01 通常経路

- Given ユーザー "alice" は検証済みのメールアドレスを持つ
- When ユーザー "alice" のパスワード、認証要素、復旧コード、または信頼済みデバイスが増減する
- Then "alice" の検証済みアドレスへセキュリティ通知が送られる
- Then 通知の本文には生の IP アドレス、生の User-Agent、トークン、資格情報のいずれも含まれない

#### Example: EX-AUTHENTICATION-031-02 メールの配送に失敗する

- Given ユーザー "alice" は検証済みのメールアドレスを持つ
- When ユーザー "alice" のパスワード、認証要素、復旧コード、または信頼済みデバイスが増減する
- Then "alice" の検証済みアドレスへセキュリティ通知が送られる
- Then メールの配送に失敗する
- Then 資格情報の変更は成立したままで、配送の失敗は呼び出し元へ伝播しない

### Rule: REQ-AUTHENTICATION-032 メールアドレスの変更は変更前のアドレスへ通知される

Primary actor: `AuthenticatedSelf`

#### Example: EX-AUTHENTICATION-032-01 通常経路

- Given ユーザー "alice" の検証済みのメールアドレスは "old@example.test" である
- When ユーザー "alice" がメールアドレスの "new@example.test" への変更を要求する
- Then セキュリティ通知は "old@example.test" へ送られる
- When その変更が確定する
- Then セキュリティ通知は "new@example.test" へ送られる

### Rule: REQ-AUTHENTICATION-034 停止した種別の通知は送られない

Primary actor: `AuthenticatedSelf`

#### Example: EX-AUTHENTICATION-034-01 通常経路

- Given ユーザー "alice" は検証済みのメールアドレスを持つ
- When ユーザー "alice" がステップアップ認証を成立させ、既知でない端末からのサインイン通知の受信を停止する
- Then 以後、既知でない端末から認証しても通知は送られない
- Then 資格情報の変更に対する通知は引き続き送られる

## 失効と変更

### Rule: REQ-AUTHENTICATION-033 必須の種別の通知は本人が止められない

Primary actor: `AuthenticatedSelf`

#### Example: EX-AUTHENTICATION-033-01 通常経路

- Given ユーザー "alice" はステップアップ認証を成立させたセッションで認証済みである
- When ユーザー "alice" が自身の通知設定を取得する
- Then 全種別が返り、資格情報・認証要素・連絡先・なりすましの各種別は mandatory として示される
- When ユーザー "alice" が必須の種別を含めて受信の停止を要求する
- Then 要求は拒否され、設定はいずれの種別についても変更されない

#### Example: EX-AUTHENTICATION-033-02 ステップアップ認証を成立させていないセッションで更新を要求する

- Given ユーザー "alice" はステップアップ認証を成立させたセッションで認証済みである
- When ユーザー "alice" が自身の通知設定を取得する
- Then 全種別が返り、資格情報・認証要素・連絡先・なりすましの各種別は mandatory として示される
- When ユーザー "alice" が必須の種別を含めて受信の停止を要求する
- Then ステップアップ認証を成立させていないセッションで更新を要求する
- Then ステップアップ認証による再認証が要求される
