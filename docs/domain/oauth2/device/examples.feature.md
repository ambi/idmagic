# Feature: デバイス認可の例

## Rule: REQ-OAUTH2-027 デバイス認可フローでアクセストークンを取得できる

### Example: EX-OAUTH2-027-01 通常経路

- Given confidential クライアント "tv-app" が grant_types に "urn:ietf:params:oauth:grant-type:device_code" を含めて登録済みである
- When "tv-app" として scope "openid プロファイル" でデバイス認可を開始する
- Then レスポンスに device_code・user_code・verification_uri・interval が含まれる
- When ユーザー "alice" が verification_uri で user_code を入力し承認する
- Then device authorization は承認済みになる
- When クライアントが device_code "DC1" を交換する
- Then レスポンスに access_token と id_token が含まれる
- Then "DeviceAuthorizationRequested" が発行される
- Then "DeviceAuthorizationApproved" が発行される
- Then "AccessTokenIssued" が発行される

### Example: EX-OAUTH2-027-02 ユーザー承認前にポーリングする

- Given confidential クライアント "tv-app" が grant_types に "urn:ietf:params:oauth:grant-type:device_code" を含めて登録済みである
- When "tv-app" として scope "openid プロファイル" でデバイス認可を開始する
- Then レスポンスに device_code・user_code・verification_uri・interval が含まれる
- When ユーザー "alice" が verification_uri で user_code を入力し承認する
- Then device authorization は承認済みになる
- When クライアントが device_code "DC1" を交換する
- But ユーザー承認前にポーリングする
- Then Issued 状態の device_code "DC1" を交換する
- And エラー "AuthorizationPendingError"

### Example: EX-OAUTH2-027-03 ポーリング間隔より短い再試行をする

- Given confidential クライアント "tv-app" が grant_types に "urn:ietf:params:oauth:grant-type:device_code" を含めて登録済みである
- When "tv-app" として scope "openid プロファイル" でデバイス認可を開始する
- Then レスポンスに device_code・user_code・verification_uri・interval が含まれる
- When ユーザー "alice" が verification_uri で user_code を入力し承認する
- Then device authorization は承認済みになる
- When クライアントが device_code "DC1" を交換する
- But ポーリング間隔より短い再試行をする
- Then interval 5 秒の device_code "DC1" を Issued 状態で用意する
- And device_code "DC1" を交換し "2s" 経過後に再度交換する
- And 2 回目はエラー "SlowDownError"

### Example: EX-OAUTH2-027-04 device_code が expires_in を超えている

- Given confidential クライアント "tv-app" が grant_types に "urn:ietf:params:oauth:grant-type:device_code" を含めて登録済みである
- When "tv-app" として scope "openid プロファイル" でデバイス認可を開始する
- Then レスポンスに device_code・user_code・verification_uri・interval が含まれる
- When ユーザー "alice" が verification_uri で user_code を入力し承認する
- Then device authorization は承認済みになる
- When クライアントが device_code "DC1" を交換する
- But device_code が expires_in を超えている
- Then issued_at "2026-01-01T00:00:00Z"・expires_at "2026-01-01T00:10:00Z" の device_code "DC1" を時刻 "2026-01-01T00:11:00Z" で交換する
- And エラー "ExpiredTokenError"
