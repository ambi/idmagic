# Feature: 信頼済みデバイスの例

## Rule: REQ-AUTHENTICATION-026 第二要素の成立時に本人が同意した端末は次回以降の第二要素を省略できる

### Background:

- Given テナントの `trusted_device_max_age_seconds` は正の値である
- And 対象 Application の実効サインインポリシーは `Mfa` で `allow_trusted_device=true` である
- And ユーザー "alice" は TOTP 認証要素を登録済みである

### Example: EX-AUTHENTICATION-026-01 通常経路

- When ユーザー "alice" が正しいパスワードに続けて正しい TOTP コードを送信し、このデバイスを記憶することに同意する
- Then 認証が成立し、realm scope の HttpOnly cookie として信頼済みデバイスの資格情報が発行される
- Then "TrustedDeviceRegistered" が発行される
- When 同じブラウザーでユーザー "alice" が正しいパスワードを送信する
- Then 第二要素の画面へ進まずに認証が成立し、`amr` に `tdev` が加わって `acr` が `urn:idmagic:acr:mfa` になる
- Then 信頼済みデバイスの verifier が回転し、更新された cookie が再発行される

### Scenario Outline: 条件ごとの結果

- When ユーザー "alice" が正しいパスワードに続けて正しい TOTP コードを送信し、このデバイスを記憶することに同意する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-AUTHENTICATION-026-02 | テナントの `trusted_device_max_age_seconds` が 0 または未設定である | 同意は無視され、デバイスは記憶されない |
  | EX-AUTHENTICATION-026-03 | 第二要素として復旧コードを消費した | デバイスは記憶されない |
  | EX-AUTHENTICATION-026-04 | パスワードだけで認証が完了した (ポリシーが MFA を要求していない) | デバイスは記憶されない |

## Rule: REQ-AUTHENTICATION-027 期限切れ・盗難・別テナントの信頼済みデバイス cookie は第二要素を省略できない

### Background:

- Given ユーザー "alice" は 1 つの信頼済みデバイスを持ち、対象 Application の実効サインインポリシーは `Mfa` である

### Example: EX-AUTHENTICATION-027-01 通常経路

- When ユーザー "alice" が絶対期限を過ぎた cookie を提示して正しいパスワードを送信する
- Then LoginSession は `authentication_pending=true` になり、第二要素の選択画面へ進む
- Then `amr` に `tdev` は加わらない

### Scenario Outline: 条件ごとの結果

- When ユーザー "alice" が絶対期限を過ぎた cookie を提示して正しいパスワードを送信する
- But <condition>
- Then 第二要素を要求する

#### Examples:

  | example_id | condition |
  | --- | --- |
  | EX-AUTHENTICATION-027-02 | 直近利用から idle 期限を過ぎた cookie を提示する |
  | EX-AUTHENTICATION-027-03 | 回転前の古い cookie を提示する |
  | EX-AUTHENTICATION-027-04 | 別テナントの realm で発行された cookie を提示する |
  | EX-AUTHENTICATION-027-05 | selector は正しいが verifier が一致しない cookie を提示する |

## Rule: REQ-AUTHENTICATION-029 信頼済みデバイスは機微操作の再認証を肩代わりしない

### Background:

- Given ユーザー "alice" は信頼済みデバイスによって `amr` に `tdev` を持つセッションで認証済みである
- And そのセッションはステップアップ認証を行っていない

### Example: EX-AUTHENTICATION-029-01 通常経路

- When ユーザー "alice" がパスワードの変更、TOTP 認証要素の解除、または他セッションの一括失効を要求する
- Then StepUpRequiredError で拒否され、パスワード、認証要素、セッションは変更されない
- When ユーザー "alice" が自身の信頼済みデバイスを一覧する
- Then selector と verifier を含まない一覧が最終利用時刻の降順で返り、現在の端末が current として示される
- When ユーザー "alice" がステップアップ認証を成立させて信頼済みデバイスを失効させる
- Then 対象は一覧から消え、"TrustedDeviceRevoked" が発行される

### Example: EX-AUTHENTICATION-029-02 ステップアップ認証なしで信頼済みデバイスの失効を要求する

- When ユーザー "alice" がパスワードの変更、TOTP 認証要素の解除、または他セッションの一括失効を要求する
- Then ステップアップ認証による再認証が要求される
- When ユーザー "alice" が自身の信頼済みデバイスを一覧する
- But ステップアップ認証なしで信頼済みデバイスの失効を要求する
- Then ステップアップ認証による再認証が要求される

### Example: EX-AUTHENTICATION-029-03 既に失効済みのデバイスへ同じ失効操作を再送する

- When ユーザー "alice" がパスワードの変更、TOTP 認証要素の解除、または他セッションの一括失効を要求する
- Then ステップアップ認証による再認証が要求される
- When ユーザー "alice" が自身の信頼済みデバイスを一覧する
- Then selector と verifier を含まない一覧が最終利用時刻の降順で返り、現在の端末が current として示される
- When ユーザー "alice" がステップアップ認証を成立させて信頼済みデバイスを失効させる
- But 既に失効済みのデバイスへ同じ失効操作を再送する
- Then 要求は成功として扱われ、最初の失効時刻を保持する

## Rule: REQ-AUTHENTICATION-028 資格情報が変わると信頼済みデバイスはすべて失効する

### Background:

- Given ユーザー "alice" は有効な信頼済みデバイスを持つ

### Example: EX-AUTHENTICATION-028-01 通常経路

- When ユーザー "alice" が自身のパスワードを変更する
- Then ユーザー "alice" の信頼済みデバイスはすべて失効し、"TrustedDeviceRevoked" が発行される
- When 失効した端末でユーザー "alice" が再びログインする
- Then 第二要素が再び要求される

### Scenario Outline: 条件ごとの結果

- When ユーザー "alice" が自身のパスワードを変更する
- But <condition>
- Then 同じく全デバイスが失効する

#### Examples:

  | example_id | condition |
  | --- | --- |
  | EX-AUTHENTICATION-028-02 | メールのリセットリンクでパスワードを再設定する |
  | EX-AUTHENTICATION-028-03 | ユーザー "alice" が TOTP 認証要素を登録または解除する |
  | EX-AUTHENTICATION-028-04 | 管理者がユーザー "alice" の認証器をリセットする |
  | EX-AUTHENTICATION-028-05 | 管理者がユーザー "alice" を無効化する |
  | EX-AUTHENTICATION-028-06 | ユーザー "alice" が他のセッションを一括失効させる |
