---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-25
depends_on: []
change_kind: feature
initial_context:
  scl:
    Authentication:
      - models.MfaFactorType
      - models.MfaFactor
      - interfaces.StartBrowserMfaEnrollment
      - interfaces.ConfirmBrowserMfaEnrollment
      - interfaces.GetAccountSecurity
      - standards.NISTSP80063B4
      - standards.RFC8176
  decisions:
    - decisions/ADR-087-webauthn-phishing-resistant-mfa.md
    - decisions/ADR-088-layered-account-recovery.md
    - decisions/ADR-106-identity-and-credential-policy-configuration.md
  source:
    - backend/authentication/mfa/domain
    - backend/authentication/mfa/usecases
    - backend/authentication/totp
    - backend/shared/notification
  tests:
    - backend/authentication/mfa/domain
    - backend/authentication/mfa/usecases
  stop_before_reading:
    - backend/oauth2
    - backend/saml
affected_spec:
  - { path: spec/contexts/authentication/models.tsp, symbol: IdMagic.Contract.MfaFactorType }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Contract.StartBrowserMfaEnrollment }
---

# Email OTP と SMS OTP の MFA factor を、ポリシーで制御された restricted factor として追加する

## Motivation

現在の MFA factor は `MfaFactorType` = `Totp` / `Webauthn` / `Hwk` / `Swk` のみである
(`spec/contexts/authentication/`)。強い factor は揃っているが、**低摩擦な代替が無い**。

これは 2 つの実務的な壁になる:

1. **導入初期に MFA を強制できない**。認証アプリの導入も passkey の登録もできない
   (できない事情がある) ユーザーが一定数いるテナントでは、MFA 強制を諦めるか、
   その層を例外にするしかない。「弱い MFA でも無しよりは強い」という現実的な選択肢が無い。
2. **アカウント復旧の選択肢が薄い**。[[ADR-088-layered-account-recovery]] は層状の復旧を
   定義しているが、認証器を失ったユーザーの手段は recovery code と管理者操作
   ([[wi-143-admin-authenticator-reset-and-account-recovery]]) に限られる。
   登録済みメールへの OTP は、多くの製品で最初の復旧手段である。

一方で **NIST SP 800-63B は SMS / メール OTP を restricted authenticator と位置付ける**。
SIM スワップとメールアカウント侵害で突破されるため、無条件に足すべきではない。
`Authentication.standards.NISTSP80063B4` を宣言している以上、この位置付けを実装に
反映する必要がある。

競合比較:

- **Okta**: Email / SMS / Voice factor を提供し、ポリシーで factor ごとに許可・禁止を設定できる。
  SMS は「非推奨」として明示される。
- **Entra ID**: SMS / 音声通話を提供するが、既定で無効化を推奨し、authentication methods
  policy で制御する。
- **Keycloak**: 標準では持たず、拡張で実装する。

つまり「提供するが、既定オフで、ポリシーで明示的に有効化させ、弱い factor であることを
運用者とユーザーに示す」というのが業界の到達点である。本 WI はその形で実装する。

## Scope

- **decision**:
  - 新規 ADR (restricted MFA factor の位置付け): Email OTP / SMS OTP を restricted と
    宣言する根拠 (NIST SP 800-63B)、**テナント既定で無効**とし明示有効化を要する方針、
    sign-in policy ([[ADR-079-application-sign-in-policy-evaluation]]) で
    「restricted factor を許容しない」を表現できること、restricted factor だけを持つ
    ユーザーに強い factor の登録を促す扱い、OTP の桁数 / 有効期限 / 試行回数上限 /
    再送間隔、AMR 値 (RFC 8176 の `sms` / `otp` / `mfa`) の出し方、
    [[ADR-087-webauthn-phishing-resistant-mfa]] の phishing-resistant 判定に
    restricted factor を含めないこと、SMS 送信 adapter を持たない配備での挙動 (fail-closed) を記録する。
- **scl**:
  - `MfaFactorType` に `EmailOtp` / `SmsOtp` を追加する。
  - `MfaFactor` に OTP factor 用の宛先参照 (メールは User の verified email、
    SMS は verified phone) を表現し、**未検証の宛先では登録できない**ことを明記する。
  - `Tenancy` または `Authentication` のポリシー設定に `allowed_mfa_factor_types` を追加する
    ([[ADR-106-identity-and-credential-policy-configuration]] の設定面に載せる)。
  - `StartEmailOtpEnrollment` / `ConfirmEmailOtpEnrollment` / `StartSmsOtpEnrollment` /
    `ConfirmSmsOtpEnrollment` / `SubmitBrowserOtp` / `ResendBrowserOtp` interface を追加する
    (既存の TOTP 登録 interface の形に倣う)。
  - `states` / events に OtpChallengeIssued / OtpChallengeResent /
    OtpChallengeThrottled を追加する (既存 MfaChallenge* 系と整合させる)。
  - `standards.NISTSP80063B4` に restricted authenticator の要件を追加し、
    「restricted factor は既定無効」「ユーザーに代替手段を案内する」を要件として書く。
  - `scenarios`: 未検証メールでは Email OTP を登録できない / テナントで無効な factor は
    登録も認証もできない / OTP が期限切れで拒否される / 試行回数上限で throttle される /
    再送が間隔制限に従う / restricted factor のみのユーザーに強い factor 登録が促される /
    SMS adapter 未設定の配備で SMS OTP が有効化できない。
- **go**:
  - OTP challenge の domain (コード生成・ハッシュ保存・期限・試行回数・再送間隔) を実装する。
    コードは平文で保存しない。
  - Email OTP は既存の `backend/shared/notification` を使う。本文は
    [[wi-288-localized-notification-template-catalog-and-tenant-customization]] の
    テンプレートカタログ経由 (未完了なら組込み既定を追加)。
  - SMS 送信は **port + adapter** で追加する (`backend/shared/notification` に
    `SmsSender` port、`sms_console` / `sms_noop` を同梱)。実 gateway adapter は
    設定で差し替え可能にし、本 WI では具体的な商用 gateway を同梱しない。
  - 電話番号の検証フロー (登録時に OTP で verified にする) を追加する。E.164 正規化を行う。
  - throttle は既存のログインスロットル基盤
    ([[ADR-077-shared-login-throttle-store-and-ephemeral-state-ha]]) の考え方に合わせる。
  - AMR 値をトークンに出す経路を確認し、`otp` / `sms` を含める。
- **http**:
  - 登録 / 確認 / 認証時の OTP 送信・検証・再送のエンドポイントを追加する。
    OTP 送信は列挙対策として「宛先の存在に依存しない応答」を返す。
- **ui**:
  - アカウントポータルのセキュリティ画面に Email OTP / SMS OTP の登録・削除を追加する
    (テナントで有効な factor のみ表示)。
  - ログイン時の OTP 入力画面 (残り試行回数・再送ボタン・再送待ち時間) を追加する。
  - 管理コンソールに `allowed_mfa_factor_types` の設定 UI を追加し、
    restricted factor には注意書きを表示する。
- **documentation**:
  - README に SMS adapter の設定、restricted factor の既定無効、NIST の位置付けを追記する。

## Out of Scope

- push 通知型 MFA (専用モバイルアプリ)。アプリ配布を伴うため別領域。
  human-in-the-loop な承認は [[wi-52-ciba-async-human-approval]] が扱う。
- 音声通話 OTP。
- 商用 SMS gateway の具体 adapter 同梱。port と console/noop adapter までを本 WI とする。
- SMS / メールの配信到達率監視。
- セカンダリ / リカバリ用連絡先の管理 UI。→ [[wi-41-secondary-and-recovery-email]]
  (本 WI は「検証済み宛先を factor に使う」ところまで)
- パスワードレスの magic link / email OTP による**一次認証**。
  → [[wi-88-passwordless-email-login]] (本 WI は第 2 要素としての OTP)

## Plan

- **既定無効を仕様にするのが最重要**。単に factor を足すと、テナントが無自覚に弱い MFA を
  使い始める。`allowed_mfa_factor_types` を導入し、既定は `Totp` / `Webauthn` 系のみとする。
  restricted factor の有効化は管理者の明示操作にし、UI に NIST の位置付けを表示する。
- **未検証宛先での登録を禁止する**。未検証メール / 電話に OTP を送れる実装は、
  攻撃者が自分の宛先を登録して MFA を迂回する経路になる。ここを最初のテストにする。
- **OTP の防御パラメータを domain に固める**。桁数 (6)、有効期限 (5 分程度)、
  試行回数上限、再送間隔、challenge の単回使用を domain の不変条件として持ち、
  handler 側の実装差で緩まないようにする。
- **SMS は port だけ入れて adapter は差し替え式にする**。商用 gateway を同梱すると
  依存とアカウント設定が増え、この repo の「adapter は選択式」という方針
  ([[ADR-016-persistence-adapter-selection]] の考え方) と合う。adapter 未設定の配備で
  SMS factor を有効化しようとしたら fail-closed で拒否する。
- **既存の TOTP 実装の形に倣う**。`backend/authentication/totp` の登録 (start / confirm) と
  検証の構造がそのまま使えるため、新しいパターンを作らない。
- **AMR を正しく出す**。`otp` / `sms` を AMR に含めないと、RP 側が
  「強い MFA で認証された」と誤解する。RP が factor 強度で判断できるようにするのが
  restricted factor を提供する前提条件である。
- 未決定: restricted factor のみのユーザーに強い factor の登録を「促す」のか
  「required action で強制する」のか。第 1 段では促す (通知 + 画面上の案内) とし、
  強制は [[wi-127-mfa-enrollment-onboarding-and-enforcement]] の仕組みに委ねる。

## Tasks

- [ ] T001 [Spec] `MfaFactorType` に EmailOtp / SmsOtp、宛先検証済み要件、
      `allowed_mfa_factor_types`、interface 6 件、event 3 件、
      NISTSP80063B4 の restricted 要件、scenario 7 件を追加し `just check-scl` を通す。
- [ ] T002 [ADR] restricted MFA factor の位置付けの ADR を起票する (既定無効・防御パラメータ・
      AMR・phishing-resistant 判定に含めない・adapter 未設定時の fail-closed)。
- [ ] T003 [Domain] OTP challenge の生成 / ハッシュ保存 / 期限 / 試行回数 / 再送間隔 /
      単回使用を実装する。RED: 期限切れ・試行超過・再送間隔違反・再使用が拒否される
      テストを先に書く (scenario `Authentication.otp_challenge_throttled`) → GREEN。
- [ ] T004 [Policy] `allowed_mfa_factor_types` をポリシー設定に追加し、
      既定を強い factor のみにする。RED: 無効な factor の登録・認証が拒否されるテスト → GREEN。
- [ ] T005 [Email OTP] 登録 (start / confirm) と認証時検証を実装する。verified email 必須。
      RED: 未検証メールでの登録が拒否されるテスト → GREEN。
- [ ] T006 [SMS port] `SmsSender` port と `sms_console` / `sms_noop` adapter を
      `backend/shared/notification` に追加し、bootstrap の配線と環境変数を追加する。
      RED: adapter 未設定で SMS factor 有効化が拒否されるテスト → GREEN。
- [ ] T007 [Phone] 電話番号の E.164 正規化と OTP による検証フローを実装する。
      RED: 不正形式の拒否と検証済み遷移のテスト → GREEN。
- [ ] T008 [SMS OTP] 登録と認証時検証を実装する。RED → GREEN。
- [ ] T009 [AMR] トークンの AMR に `otp` / `sms` を含める。phishing-resistant 判定に
      含めないことをテストで固定する。RED → GREEN。
- [ ] T010 [HTTP] 登録 / 確認 / OTP 送信 / 検証 / 再送のエンドポイントを追加し、
      宛先の存在に依存しない応答にする。RED: 列挙不可を確認する handler テスト → GREEN。
- [ ] T011 [UI] アカウントポータルの factor 登録・削除、ログイン時の OTP 入力画面
      (残り試行・再送・待ち時間)、管理コンソールの `allowed_mfa_factor_types` 設定と
      注意書きを追加する。RED: presentation logic の unit test → GREEN。
- [ ] T012 [Docs] README に SMS adapter 設定、既定無効、NIST の位置付けを追記する。
- [ ] T013 [Verify] 下記 Verification を緑にする。`just spec-render` を実行する。

## Verification

- `just check` / `just check-scl` / `just check-work-items` / `just check-ids`
- `just test-go` / `just test-go-race` / `just verify-go`
- `just verify-ui` / `just test-ui-unit`
- 手動: Mailpit で (1) Email OTP を登録し、ログイン時に第 2 要素として使えること、
  (2) 期限切れコードが拒否されること、(3) 試行回数超過で throttle されること、
  (4) 再送が間隔制限に従うこと、(5) テナントで factor を無効化すると登録も認証も
  できなくなること、(6) 発行されたトークンの AMR に `otp` が含まれること、を確認する。

## Risk Notes

**弱い factor を足すことは、それ自体がセキュリティ姿勢の後退になりうる**。既定無効・
明示有効化・UI での注意表示・AMR での明示という 4 点で「弱いことが見える」状態を保つ。
これを守らない実装は本 WI の趣旨に反する。
未検証宛先での登録は MFA 迂回に直結する。verified 必須を domain の不変条件にし、
最初のテストで固定する。
OTP 送信エンドポイントはメール / SMS 送信の踏み台とコスト攻撃の対象になる。
再送間隔・試行回数・レート制限 ([[wi-27-endpoint-rate-limit-and-bot-mitigation]] と整合) を
必須とし、宛先の存在で応答を変えない。
SMS adapter が未設定の配備で factor を有効化できると、ユーザーが登録できるのに
コードが届かないロックアウトを生む。有効化時点で fail-closed に拒否する。
