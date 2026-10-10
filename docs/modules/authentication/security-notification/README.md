# セキュリティ通知

## 概要

この文書は、アカウントに起きたセキュリティの上の変化を本人へ知らせるメールの契機、宛先、受信の設定の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 既知でない端末からのサインインと、資格情報、連絡先、セッション、なりすましの変化の通知、種別ごとの受信の設定 |
| 行為者 | EndUser、本人 |
| 扱わないもの | メールの配送そのものは `EmailSender` のポートが扱う |

## モデル

アカウントが乗っ取られたことに本人が最初に気づく手がかりは、たいてい「身に覚えのない通知が届いたこと」である。
通知はすべて最大限の努力であり、送信の失敗が元の操作を巻き戻すことはない。

| 種別 | 契機となるイベント | 本人による無効化 |
|---|---|---|
| `new_device_sign_in` | 既知でない端末からの `UserAuthenticated` | 可能 |
| `credential_change` | `PasswordChanged` | 不可 |
| `mfa_change` | `MfaFactorEnrolled` / `MfaFactorRemoved` / `WebAuthnCredentialRegistered` / `WebAuthnCredentialRemoved` / `RecoveryCodesGenerated` / `RecoveryCodesRevoked` / `AuthenticatorResetCompleted` / `TrustedDeviceRegistered` | 不可 |
| `contact_change` | `EmailChangeRequested` / `EmailChanged` | 不可 |
| `session_revoked` | `self_revoke` または `admin_revoke` の `SessionEnded` | 可能 |
| `impersonation` | `SessionImpersonationStarted` | 不可 |

受信の設定は「無効にした種別」の集合として保存するので、後から種別が増えても、既存の設定は「有効」のまま引き継がれる。
設定のレコードがないことと「すべて有効」は同じ意味であり、初回の変更までレコードを作らない。

宛先は、イベントの発生の時点で本人の `User` に保存されている検証済みのメールアドレスに固定し、イベントの payload や要求の入力からは決して取らない。
`EmailChangeRequested` は変更の確定の前に発行されるので、通知は変更前のアドレスへ届く。
`EmailChanged` は確定の後なので新しいアドレスへ届き、完了の確認になる。
検証済みのアドレスを持たないユーザーには送らない。
なりすましの通知は、操作した管理者ではなく、なりすまされた本人へ送る。

- **判断**：資格情報、連絡先、なりすましの通知を必須にするのは、乗っ取りの直後に攻撃者が最初に消すのが通知だからである。通知を消せることは、通知が無いことと変わらない。任意にするのは、本人にとって明らかに冗長になりうる二つ、端末の入れ替えが多い環境でのサインインの通知と、自分で行ったセッションの失効に限る。

## 操作

### 利用者によるサインイン

#### REQ-AUTHENTICATION-030 既知でない端末からのサインインだけがセキュリティ通知を生む

- User が既知でない端末で認証に成功したとき、Authentication は、その端末を既知の端末として記録し、User の検証済みのメールアドレスへ `new_device_sign_in` の通知を送り、`AccountSecurityNotificationSent` を発行する。
- User が既知の端末で認証に成功したとき、Authentication は、通知を送らずに、その端末の最終利用の時刻を進める。
- 検証済みのメールアドレスを持たない User への通知の契機が起きたとき、Authentication は、通知を送らずに、元の操作を成功させたままにする。
- **例**：EX-AUTHENTICATION-030-01、EX-AUTHENTICATION-030-02

### 本人による資格情報と連絡先の変更

#### REQ-AUTHENTICATION-031 資格情報の変更は本人へ通知され、通知の失敗は変更を巻き戻さない

- User のパスワード、認証の要素、復旧コード、信頼済みデバイスが増減したとき、Authentication は、契機のイベントの時点で User に保存された検証済みのメールアドレスへ、モデルの表の種別の通知を送る。
- 管理者が User になりすますセッションを始めたとき、Authentication は、操作した管理者ではなく、なりすまされた User へ `impersonation` の通知を送る。
- 本人か管理者が理由 `self_revoke` か `admin_revoke` でセッションを終えたとき、Authentication は、User へ `session_revoked` の通知を送る。
- 通知を送るとき、Authentication は、本文に `event_description`、`occurred_at`、`device_summary`、`security_review_url` だけを載せ、生の IP、生の User-Agent、トークン、資格情報を載せない。
- メールの配送に失敗した場合、Authentication は、元の変更を成立させたままにし、配送の失敗を呼び出し元へ伝えない。
- **例**：EX-AUTHENTICATION-031-01、EX-AUTHENTICATION-031-02

#### REQ-AUTHENTICATION-032 メールアドレスの変更は変更前のアドレスへ通知される

- User がメールアドレスの変更を要求したとき、Authentication は、変更の前の検証済みのアドレスへ `contact_change` の通知を送る。
- メールアドレスの変更が確定したとき、Authentication は、新しいアドレスへ `contact_change` の通知を送る。
- **例**：EX-AUTHENTICATION-032-01

### 本人による受信の設定

#### REQ-AUTHENTICATION-034 停止した種別の通知は送られない

- 本人が任意の種別の受信を停止している間、その種別の契機が起きたとき、Authentication は、通知を送らない。
- 本人が任意の種別の受信を停止している間、必須の種別の契機が起きたとき、Authentication は、通知を送る。
- **例**：EX-AUTHENTICATION-034-01

#### REQ-AUTHENTICATION-033 必須の種別の通知は本人が止められない

- 本人が受信の設定を取得したとき、Authentication は、200 とすべての種別を返し、`credential_change`、`mfa_change`、`contact_change`、`impersonation` を必須として示す。
- 本人がステップアップ認証を経て任意の種別の受信の設定を更新したとき、Authentication は、無効にした種別の集合を保存し、200 と更新した設定を返す。
- 必須の種別の停止を含む更新を受けた場合、Authentication は、400 と `mandatory_notification_category` で拒否し、どの種別の設定も変えない。
- JSON として読めない本文か未知の種別を含む更新を受けた場合、Authentication は、400 と `invalid_request` で拒否する。
- ステップアップ認証を経ていないセッションで更新を受けた場合、Authentication は、403 と `step_up_required` で拒否する。
- **例**：EX-AUTHENTICATION-033-01、EX-AUTHENTICATION-033-02

## セキュリティ上の考慮

受信の設定の更新はステップアップの再認証を求める。
通知を止める操作自体が、乗っ取りの直後に行われる操作だからである。

本文に載せるのは、イベントの種別の安定した識別子（`event_description`）、発生の時刻（`occurred_at`）、User-Agent から導いたブラウザーと OS の系統に国コードを添えた要約（`device_summary`）、アカウントのセキュリティの画面への固定のリンク（`security_review_url`）だけである。
生の IP も生の User-Agent も、トークンも資格情報も本文には載せない。
メールは転送され、引用され、無期限に保持されるからである。
「心当たりがない」場合の導線を認証の要らない単発のリンクにしないのも同じ理由で、そのリンク自体が乗っ取りの経路になる。
導線は通常どおり認証を求めるセキュリティの画面へ送るだけで、リンクをたどった時点では何の状態も変わらない。
