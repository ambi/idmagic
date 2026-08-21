---
depends_on: [wi-6-real-email-sender-adapter, wi-44-authentication-event-store-and-search]
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-03
change_kind: feature
initial_context:
  specification:
    - spec/contexts/authentication/SPECIFICATION.md#REQ-AUTHENTICATION-013
    - spec/contexts/authentication/SPECIFICATION.md#REQ-AUTHENTICATION-014
    - spec/contexts/authentication/SPECIFICATION.md#REQ-AUTHENTICATION-026
  typespec:
    - IdMagic.Contract.TrustedDevice
    - IdMagic.Contract.NotificationTemplateKey
    - IdMagic.Contract.Operations.GetAccountSecurity
  source:
    - backend/shared/notification
    - backend/cmd/internal/bootstrap/audit_event_record.go
    - backend/cmd/internal/bootstrap/notification.go
    - backend/authentication/domain/events.go
    - backend/authentication/domain/device_label.go
    - backend/authentication/deps_http/deps.go
    - backend/authentication/handlers_http/routes.go
    - backend/idmanagement/domain/events.go
    - frontend/src/features/account/AccountSecurityPage.tsx
  tests:
    - backend/authentication
    - backend/shared/notification/template
  stop_before_reading:
    - backend/saml
    - backend/wsfederation
    - backend/provisioning
    - backend/sharedsignals
affected_spec:
  - { path: spec/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-030 }
  - { path: spec/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-031 }
  - { path: spec/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-032 }
  - { path: spec/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-033 }
  - { path: spec/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-034 }
  - { path: spec/contexts/authentication/models.tsp, symbol: IdMagic.Contract.SecurityNotificationCategory }
  - { path: spec/contexts/authentication/models.tsp, symbol: IdMagic.Contract.NotificationPreference }
  - { path: spec/contexts/authentication/models.tsp, symbol: IdMagic.Contract.KnownSignInDevice }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Contract.Operations.GetMyNotificationPreferences }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.NotificationTemplateKey }
---

# アカウントのセキュリティ通知メール (サインイン / 認証情報変更アラート) を導入する

## Motivation
idmagic は SMTP email sender ([[wi-6-real-email-sender-adapter]]) と認証イベント基盤 ([[wi-44-authentication-event-store-and-search]])、そして locale 対応の通知テンプレートカタログ ([[wi-288-localized-notification-template-catalog-and-tenant-customization]]) を持つ。しかしメールを実際に送るのはパスワードのリセット、メールアドレスの検証と変更、ライフサイクルワークフローだけで、セキュリティ上の変化を本人に知らせる通知が無い。カタログには `account_security_alert` の既定文面が用意されているが、送信経路が無いまま置かれている。

代表的な IdP / アカウントサービスはセキュリティ通知を標準で送る。

- Google: 新しいデバイスからのサインイン通知。
- Okta / Entra: サインイン / 認証要素の変更 / パスワード変更の通知メール。

新しいデバイスからのサインインや、パスワード・MFA・連絡先の変更を本人に知らせることは、アカウント乗っ取りの早期検知に直結する。本 WI は既存のドメインイベントを購読して最大限努力でメール通知を送るディスパッチャーと、本人による通知設定を追加する。

## Scope
- **spec (authentication)**:
  - `SecurityNotificationCategory` (通知の種別)、`NotificationPreference` (本人の通知設定)、`KnownSignInDevice` (既知の端末)、`AccountNotificationPreferences` / `AccountNotificationCategoryPreference` / `UpdateNotificationPreferencesRequest` の射影、`AccountSecurityNotificationSent` イベントを `models.tsp` に追加する。
  - `GetMyNotificationPreferences` / `UpdateMyNotificationPreferences` を `main.tsp` に追加する (自己管理・`actor.sub` 固定)。
  - Design 節「Account security notifications」と REQ-AUTHENTICATION-030〜034 を追加する。
- **spec (tenancy)**: `NotificationTemplateKey.AccountSecurityAlert` の doc から「どの経路からも送信されない」を外す。
- **spec (root)**: `AccountSecurityAlert` の差し込み変数に `device_summary` と `security_review_url` を加える。
- **go**: `backend/authentication/securitynotification` (domain / ports / usecases / db_memory / db_postgres / handlers_http) を追加し、`bootstrap.NewEmitFunc` の配信点から購読する。`notification_preferences` と `known_sign_in_devices` の 2 テーブルを追加する。
- **go**: `DeviceLabel` を `trusteddevice/domain` から context 共通の `authentication/domain` へ移す (2 つの機能が使うため)。
- **http**: アカウントポータルに通知設定の取得 / 更新エンドポイントを追加する (認証必須・self 固定・更新はステップアップ認証)。
- **ui**: `AccountSecurityPage` に種別ごとの通知トグルを追加する。必須の種別は無効化できないことを明示する。
- **documentation**: README のテンプレート表から「まだ発行しない」を外し、送信条件と差し込み変数を書く。

## Out of Scope
- SMS / push 通知 (外部ゲートウェイ依存)。
- アプリ内通知センター / 通知履歴の閲覧 UI。
- ダイジェスト / サマリメール。
- 管理者がテンプレートを編集する機能 ([[wi-288-localized-notification-template-catalog-and-tenant-customization]] で導入済み。本 WI は key を送信経路に接続するだけ)。
- 位置情報 (GeoIP) に基づく詳細なリスクスコアリング。
- 配送の outbox・再送・dead-letter キュー (下の Design 参照)。
- テナント単位での通知種別の強制 / 無効化 (本人設定のみ)。

## Design

### 起票時の前提と現在の実装の差

起票時の Plan は [[wi-184-transactional-event-log-foundation]] の `event_logs` を durable event log とし、その cursor から notification projector を回して delivery outbox へ積む構成を前提にしていた。その後 `event_logs` / `event_deliveries` は撤去され、現在の配信点は `bootstrap.NewEmitFunc` が返す `func(spec.DomainEvent)` ひとつである。この閉包が EventSink への出力と `audit_events` への射影を行い、API・worker・batch のすべてのプロセスがここを通る。したがって本 WI の projector はこの閉包に接続する。

配送の outbox・再送・dead-letter も同じ理由で採らない。パスワードのリセットとメールアドレスの検証という、これより重要度の高い経路が最大限努力の同期送信のままである以上、セキュリティ通知だけに配送基盤を用意しても一貫性が無く、二重の保守が増える。本 WI は既存の `Notifier` の契約 (送れたかどうかを bool で返し、呼び出し元へ配送エラーを伝播しない) をそのまま使う。

### ディスパッチャー

ディスパッチャーはイベントの Go 型ではなく、`spec.MarshalDomainEvent` が返すワイヤ表現 (`type` と payload) の上で動く。監査の射影 (`NewAuditEventRecord`) と同じ形であり、これによって Authentication・IdManagement・信頼済みデバイスのそれぞれの `domain` パッケージへ依存せずに済み、カタログがコードではなくデータになる。イベント種別から種別 (category) への対応表に載っていないイベントは、その場で何もせず戻る。自分が発行する `AccountSecurityNotificationSent` も対応表に無いため、配信点へ再入しても 1 段で止まる。

送信そのものは、認証中のリクエストを待たせないために配信点から goroutine で切り離す。SMTP のタイムアウトは 10 秒であり、ログインの応答をそこまで延ばすことは許されない。切り離しの境界は `func(func())` の 1 フィールドとして注入するので、テストは同期で実行できる。プロセスが落ちれば送信中の通知は失われるが、通知は最大限努力であり、失われても認証と資格情報の変更そのものは成立している。

### イベントカタログ

| 種別 | 対象イベント | 必須 |
|---|---|---|
| `new_device_sign_in` | `UserAuthenticated` (既知でない端末からのもの) | いいえ |
| `credential_change` | `PasswordChanged` | はい |
| `mfa_change` | `MfaFactorEnrolled`, `MfaFactorRemoved`, `WebAuthnCredentialRegistered`, `WebAuthnCredentialRemoved`, `RecoveryCodesGenerated`, `RecoveryCodesRevoked`, `AuthenticatorResetCompleted`, `TrustedDeviceRegistered` | はい |
| `contact_change` | `EmailChangeRequested`, `EmailChanged` | はい |
| `session_revoked` | `SessionEnded` (`self_revoke` / `admin_revoke` のみ) | いいえ |
| `impersonation` | `SessionImpersonationStarted` | はい |

必須の種別は本人が無効化できない。乗っ取りの直後に攻撃者が最初に消すのは通知だからであり、通知を消せることは通知が無いことと変わらない。任意にするのは、本人にとって明らかに冗長になりうる 2 つ (常用端末の入れ替えが多い環境でのサインイン通知と、自分で行ったセッション失効) に限る。

### 宛先

宛先は「イベント発生時点で本人の `User` に保存されている検証済みのメールアドレス」に固定し、イベントの payload からは決して取らない。`EmailChangeRequested` は変更の確定前に発行されるため、この規則によって通知は変更前のアドレス、つまり攻撃者が置き換えようとしているアドレスへ届く。`EmailChanged` は確定後なので新しいアドレスへ届き、完了の確認になる。したがって「変更された新アドレスだけに通知する」状態は生じない。検証済みのアドレスが無いユーザーには送らない。

`SessionImpersonationStarted` の宛先は `actorUserId` (管理者) ではなく `targetUserId` (なりすまされた本人) である。

### 新しい端末の判定

`known_sign_in_devices` を置き、`(user_id, device_hash)` を鍵とする。`device_hash` はテナントの相関ソルトを効かせた `SaltedHash(salt, user_agent)` であり、生の User-Agent も IP も保存しない。サインインのたびに upsert し、行が新たに作られたときだけ「新しい端末」と判定する。

サインイン履歴 (`audit_events` の `UserAuthenticated`) を走査する案は採らない。監査ストアには端末で引くインデックスが無く、判定のたびに直近 N 件の走査になるうえ、保持期間の掃除が判定の意味を静かに変えてしまう。専用テーブルなら判定は 1 行の upsert で、複数レプリカでも重複しない。同時に、これが通知の重複排除そのものになる。同じ端末からの 2 回目以降のサインインでは行が既に在り、通知は送られない。

行は保持期間の掃除 (`RunRetentionSweep`) が `last_seen_at` から 365 日で消す。サインイン履歴の保持期間と同じであり、履歴から消えた端末を「既知」と呼び続けない。

### 本文

`account_security_alert` テンプレートに `device_summary` と `security_review_url` を加える。`device_summary` は User-Agent から導いたブラウザーと OS の系統だけのラベル (例 `Chrome / macOS`) に国コードを添えたもので、生の IP も User-Agent も本文に載せない。端末の情報を持たないイベントでは `-` を入れる。`security_review_url` はアカウントのセキュリティ画面への固定のリンクであり、認証を要求する通常の導線である。トークンを含む URL は載せない。「心当たりがない」場合の導線をトークン付きの単発リンクにすると、そのリンク自体が乗っ取りの経路になるからである。

`event_description` は種別ではなくイベント種別ごとの短い説明であり、テンプレート側の locale で解決したい値ではあるが、現在の `Notifier` は差し込み変数を呼び出し元から受け取る契約なので、送信時点では locale が確定していない。本 WI は説明をイベント種別の安定した識別子 (`sign_in_new_device` など) として渡し、locale ごとの語彙は追わない。

## Plan
1. 仕様を先に変える (authentication / tenancy / root)。`just check-spec`。
2. `DeviceLabel` を `authentication/domain` へ移す。
3. domain (カテゴリのカタログ、通知設定の値オブジェクト) と ports。
4. 永続化 (memory / PostgreSQL / スキーマ / sqlc) と保持期間の掃除。
5. ディスパッチャーの use case と `bootstrap` への配線。
6. アカウントポータルの API と UI。
7. `just verify`。

## Tasks
- [x] T001 [Spec] 通知の種別・本人設定・既知の端末・イベント・自己管理インターフェース・REQ-AUTHENTICATION-030〜034 を追加して再生成した (`just check-spec` / `just check-api-compat` 通過)。
- [x] T002 [Domain] 種別のカタログ (必須 / 任意、イベント種別と宛先項目の対応、locale ごとの説明) と受信設定の値オブジェクトを実装した。RED を確認したうえで実装 — `TestMandatoryCategoriesCannotBeDisabled` (REQ-AUTHENTICATION-033)、`TestPreferencesAllowEveryCategoryExceptTheDisabledOnes` (REQ-AUTHENTICATION-034)、`TestTriggerCatalogIsComplete` (REQ-AUTHENTICATION-030/031/032)、`TestSessionEndNotifiesOnlyExplicitRevocations`。
- [x] T003 [Persistence] `notification_preferences` と `known_sign_in_devices` の port / memory / PostgreSQL / スキーマ / sqlc / 保持期間の掃除を実装した。`TestPreferenceRepositoryRoundTripsAndTreatsAbsenceAsAllEnabled` (REQ-AUTHENTICATION-034)、`TestKnownDeviceRepositoryReportsOnlyTheFirstObservation` (REQ-AUTHENTICATION-030)、`TestKnownDeviceRepositoryDeletesOnlyIdleRows`、`TestRetentionSweepDeletesIdleKnownSignInDevices`。
- [x] T004 [Dispatcher] ワイヤ表現からの射影、既知でない端末の判定、必須種別の優先、`bootstrap.NewEmitFunc` からの切り離した配線を実装した。`TestDispatchNotifiesOnlyTheFirstSignInFromEachDevice` (REQ-AUTHENTICATION-030)、`TestDispatchNotifiesCredentialAndMfaChangesWithoutSensitiveContent` (REQ-AUTHENTICATION-031)、`TestDispatchSendsImpersonationNoticeToTheTarget`、`TestEmitFuncSendsSecurityNotificationsForCatalogEvents`、`TestEmitFuncIgnoresEventsOutsideTheCatalog`、`TestEmitFuncDoesNotChainNotifications`。
- [x] T005 [Templates] `account_security_alert` に `device_summary` と `security_review_url` を加え、ja / en の既定文面・許可集合・プレビュー値・README とルート仕様の表を更新した。
- [x] T006 [Account API/UI] 必須表示付きの取得 / 更新 (更新はステップアップ認証) と、セキュリティ画面のトグルを追加した。`TestGetNotificationPreferencesReturnsTheWholeCatalog` / `TestUpdateNotificationPreferencesRejectsMandatoryCategories` (REQ-AUTHENTICATION-033)、`TestUpdateNotificationPreferencesDisablesOnlyTheNamedCategories` (REQ-AUTHENTICATION-034)、`TestStepUpGateBlocksStaleSessionOnAllSensitiveEndpoints` の対象表に `notification_preferences_update` を追加、UI は `locks the toggle for categories the user cannot turn off` / `sends the whole disabled set when a category is turned off`。
- [x] T007 [Verify] カタログ全件、同一端末の再送抑止、必須種別の無効化拒否、宛先の解決 (変更前 / 変更後)、配送失敗でも操作が成立すること、本文に生の IP / User-Agent が載らないこと、テナント境界をまたがないことを検証した。

## Verification
- `just test-go-package ./backend/authentication/securitynotification/...`
- `just test-ui-unit-file src/features/account/AccountSecurityPage.test.tsx`
- `just verify`
- 手動: 新しいブラウザーでサインイン → 通知メールが届く。同じブラウザーで再度サインイン → 届かない。パスワード / TOTP を変更 → 対応する通知が届く。
- 手動: 任意の種別を無効化 → 届かない。必須の種別は UI で無効化できない。
- 手動: email sender を失敗させても認証 / 変更操作自体は成功する。

## Risk Notes
通知は「送りすぎるとノイズ、送らなすぎると無意味」でチューニングが要る。加えて本文への機微の漏洩と、通知自体を使ったスパム送信 (列挙・メール爆撃) がリスクになる。最大限努力・必須と任意の区別・PII の最小化・本文に機微を載せない方針をテストで担保し、新しい端末の判定は専用テーブルの upsert に載せて 1 端末につき 1 通に抑える。宛先は必ず保存済みの検証済みアドレスから解決し、リクエストの入力からは決して取らない。

## Completion
- **Completed At**: 2026-08-16
- **Summary**:
  アカウントに起きたセキュリティ上の変化を本人へメールで知らせるようにした。配信の起点はドメインイベントを EventSink と `audit_events` へ流す共通の配信点で、ディスパッチャーはイベントの Go の型ではなく監査の射影と同じワイヤ表現の上で動く。したがってカタログはコードではなくデータになり、通知の仕組みは Authentication・IdManagement・信頼済みデバイスのどの domain package にも依存しない。カタログに無いイベント種別はその場で戻るので、ディスパッチャー自身が発行する `AccountSecurityNotificationSent` を含め、通知が通知を呼ぶことはない。送信は配信点から切り離して走り (SMTP の待ち時間は最大 10 秒で、ログインの応答をそこまで延ばせない)、配送に失敗しても認証や資格情報の変更そのものは成立したままである。
  通知は 6 つの種別に分け、資格情報・認証要素・連絡先・なりすましの 4 つは本人が止められない。乗っ取りの直後に攻撃者が最初に消すのは通知であり、通知を消せることは通知が無いことと変わらないためである。止められるのは、端末の入れ替えが多い環境で冗長になりうるサインイン通知と、自分で行ったセッション失効の 2 つに限る。受信設定は「無効にした種別」の集合として保存するので、後から種別が増えても既存の設定は有効のまま引き継がれ、行が無いことと「すべて有効」は同じ意味になる。設定の更新はステップアップ再認証を要求する。
  宛先はイベント発生時点で本人の `User` に保存されている検証済みアドレスに固定し、イベントの payload やリクエストの入力からは取らない。`EmailChangeRequested` は変更の確定前に発行されるため、通知は変更前のアドレス — 攻撃者が置き換えようとしているアドレス — へ届き、`EmailChanged` は確定後の新しいアドレスへ届く。なりすましの通知は操作した管理者ではなく、なりすまされた本人へ送る。
  既知でない端末の判定は新設した `known_sign_in_devices` の挿入で行う。`device_hash` はテナントの相関ソルトを効かせた User-Agent の SHA-256 で、生の User-Agent も IP も保存しない。挿入できたときだけ通知するので、判定そのものが通知の重複排除を兼ねる。行は保持期間の掃除がサインイン履歴と同じ 365 日で消す。本文に載せるのはイベントの説明、発生時刻、ブラウザーと OS の系統に国コードを添えた要約、そして認証を要求するアカウントのセキュリティ画面への固定リンクだけである。
- **Semantic Difference** (`just spec-diff`):
  - 追加した正規シナリオ: REQ-AUTHENTICATION-030 / 031 / 032 / 033 / 034
  - 追加した TypeSpec 宣言: `SecurityNotificationCategory`、`NotificationPreference`、`KnownSignInDevice`、`AccountNotificationCategoryPreference`、`AccountNotificationPreferences`、`UpdateNotificationPreferencesRequest`、`AccountSecurityNotificationSent`、`GetMyNotificationPreferences`、`UpdateMyNotificationPreferences`
  - 変更した既存宣言: `NotificationTemplateKey` の doc (`AccountSecurityAlert` は「どの経路からも送信されない」ではなくなった)、`AccountSecurityAlert` の差し込み変数への `device_summary` / `security_review_url` の追加
  - 追加した Design 節: Authentication の「Account security notifications」と Design Decisions 6 件
- **Verification Results**:
  - `just verify` - passed (check / lint-go / test-go / lint-ui / format-check-ui / test-ui-unit / build-ui / typecheck-tools / test-tools / check-api-compat)
  - `just test-go-package ./backend/authentication/securitynotification/...` - passed (domain / usecases / db_postgres / handlers_http)
  - `just test-go-race` - passed (切り離した送信を含む配線に競合なし)
  - `just check-schema` - passed (`notification_preferences` と `known_sign_in_devices` が psqldef で収束)
  - `just test-ui-e2e` - passed
  - 手動確認は未実施。自動テストで同等の経路を検証済み (初回サインインの通知、同一端末の再送抑止、別端末の通知、カタログ全 12 イベント、明示的なセッション失効だけの通知、なりすましの宛先、配送失敗時の記録、検証済みアドレスが無い場合の抑止と端末記録の継続、必須種別の無効化拒否、設定ストア障害時の送信継続、テナント境界)。
- **Follow-ups**:
  - 配送の outbox・再送・dead-letter は導入していない。これより重要度の高いパスワードのリセットとメールアドレスの検証が同期の最大限努力である以上、通知にだけ配送基盤を設けても一貫性が無い。導入するなら通知単独ではなく全経路をまとめて移す。
  - `IssuerResolver` は issuer を知る API プロセスだけが差し込む。worker と batch では nil のままで、そこでは本文の導線が相対パスになる。カタログに載るイベントはいずれも HTTP リクエスト由来なので現在この経路は生じないが、worker が資格情報を変える操作を持つようになったら `ISSUER` を SharedConfig へ移す必要がある。
