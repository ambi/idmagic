---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-25
depends_on: []
change_kind: feature
initial_context:
  scl:
    Authentication:
      - interfaces.RequestPasswordReset
      - interfaces.GetPasswordResetContext
    IdManagement:
      - interfaces.RequestEmailChange
      - interfaces.GetEmailVerificationContext
    Tenancy:
      - models.TenantBranding
      - interfaces.GetTenantBranding
  decisions:
    - decisions/ADR-035-smtp-email-sender-adapter.md
    - decisions/ADR-096-tenant-branding-value-and-logo-storage.md
    - decisions/ADR-105-system-runtime-hardening-and-i18n-tooling.md
  source:
    - backend/shared/notification
    - backend/authentication/password/usecases/request_password_reset.go
    - backend/idmanagement/user/usecases/request_email_change.go
    - backend/idgovernance/usecases/lifecycle_workflow_dispatcher.go
  tests:
    - backend/shared/notification
    - backend/authentication/password/usecases
  stop_before_reading:
    - backend/oauth2
    - backend/saml
affected_spec:
  - { context: Authentication, kind: interface, element: RequestPasswordReset }
  - { context: IdManagement, kind: interface, element: RequestEmailChange }
  - { context: Tenancy, kind: model, element: TenantBranding }
---

# 通知メールをローカライズ済みテンプレートカタログ化し、テナントでカスタマイズ可能にする

## Motivation

現在の通知メールは **Go の usecase 内にハードコードされた英語プレーンテキスト**である:

- `backend/authentication/password/usecases/request_password_reset.go` — `Subject: "Password reset"`
- `backend/idmanagement/user/usecases/request_email_change.go` — `Subject: "Confirm your new email address"`
- `backend/idgovernance/usecases/lifecycle_workflow_dispatcher.go` — 件名・本文の両方に
  `action.TemplateKey` をそのまま入れている (実質未実装のプレースホルダ)

これは 3 つの問題を同時に起こしている:

1. **製品の一貫性が壊れている**。UI は日本語ローカライズ済み
   ([[ADR-105-system-runtime-hardening-and-i18n-tooling]]) なのに、ユーザーが受け取るメールだけが
   英語固定である。日本語 UI でパスワードを忘れたユーザーに英語メールが届く。
2. **ブランディングがメールに届いていない**。[[wi-89-tenant-login-branding]] でログイン画面の
   ブランディングは入ったが、メールはロゴも色も送信者名も持たない。フィッシング耐性の観点でも
   「正規メールの見た目」が定義されていない状態は悪い。
3. **後続 WI がブロックされている**。[[wi-227-lifecycle-workflow-notification-template-customization]]
   と [[wi-90-account-security-notification-emails]] はどちらもテンプレート機構を前提とするが、
   その機構自体が存在しない。lifecycle 通知は現在テンプレートキーをそのまま本文にしている。

競合はいずれもここを標準機能として持つ:

- **Keycloak**: テーマごとの freemarker メールテンプレート + ロケール別メッセージバンドル。
- **Okta**: 全メールテンプレートを管理画面で編集可能、言語別バリアント、変数展開。
- **Entra ID**: 企業ブランディングをメールに適用、言語別。

本 WI は「テンプレートカタログ + ロケール解決 + テナント上書き + プレビュー」を導入し、
既存のハードコード送信を全てそこへ寄せる。

## Scope

- **decision**:
  - 新規 ADR (通知テンプレートとロケール解決): テンプレート識別子の命名、変数
    (placeholder) の許可集合とエスケープ規則、ロケール解決順序 (ユーザーの locale → テナント
    既定 locale → システム既定)、テナント上書きの許可範囲 (件名 / 本文 / 差出人表示名は可、
    任意 HTML/CSS の全面差し替えは不可)、HTML とプレーンテキストの両方を必須にする方針、
    リンク生成のドメイン基準 ([[wi-285-tenant-custom-domain-and-host-based-tenant-resolution]] と
    整合)、テンプレートに入れてはならない値 (資格情報・トークン以外の機微・生 IP) を記録する。
- **scl**:
  - `Tenancy` に `NotificationTemplate` model (tenant_id / template_key / locale / subject /
    body_text / body_html / from_display_name / updated_at) と `NotificationTemplateKey` enum
    (PasswordReset / EmailVerification / EmailChangeConfirmation / AccountSecurityAlert /
    LifecycleWorkflowNotification …) を追加する。
  - `ListNotificationTemplates` / `GetNotificationTemplate` / `UpdateNotificationTemplate` /
    `ResetNotificationTemplate` / `PreviewNotificationTemplate` / `SendTestNotification`
    interface を追加する。
  - `states` に NotificationTemplateUpdated / NotificationTemplateReset event を追加する。
  - 既存の `RequestPasswordReset` / `RequestEmailChange` の記述を「テンプレートカタログ経由で
    ロケール解決して送信する」に更新する。
  - `scenarios`: 日本語ロケールのユーザーに日本語のパスワードリセットメールが届く /
    テナント上書きが既定より優先される / 未知の placeholder を含む上書きが保存時に拒否される /
    上書きが無いテンプレートは組込み既定にフォールバックする / プレビューが実送信しない。
- **go**:
  - `backend/shared/notification` に template registry (組込み既定テンプレートの埋め込み)、
    レンダラ (placeholder 展開 + HTML エスケープ + text/html 同時生成)、ロケール解決を追加する。
    組込みテンプレートは Go の `embed` でロケール別ファイルとして持つ (ja / en を必達)。
  - テナント上書きの repository (memory / postgres) と、上書き取得を含むレンダリング経路を追加する。
  - 既存の 3 箇所のハードコード送信をテンプレート経由に置換する。lifecycle dispatcher の
    `TemplateKey` をそのまま本文に入れている実装を、カタログの
    `LifecycleWorkflowNotification` テンプレートに接続する。
  - HTML メールは `multipart/alternative` で text と同時送信する
    (`backend/shared/notification/email_smtp` を拡張)。
- **http**:
  - テンプレート一覧 / 取得 / 更新 / リセット / プレビュー / テスト送信の管理 API を追加する。
    テスト送信は宛先を「操作した管理者自身のメールアドレス」に固定して任意宛先送信の
    踏み台化を防ぐ。
- **ui**:
  - 管理コンソールにテンプレート編集画面を追加する。テンプレート一覧 (キー × ロケール)、
    利用可能な placeholder の一覧表示、編集、プレビュー (レンダリング結果の表示)、
    既定へのリセット、自分宛テスト送信を提供する。
- **documentation**:
  - README にテンプレートキー一覧、placeholder、ロケール解決順序、テスト送信の制約を追記する。

## Out of Scope

- SMS / push 通知の本文テンプレート。→ [[wi-295-email-otp-and-sms-otp-mfa-factors]] で
  factor を入れた後、必要なら同じカタログに種別を足す。
- 任意 HTML / CSS の全面差し替え (injection 面が大きい。[[wi-17-tenant-settings-page]] が
  同じ理由で custom CSS を Out of Scope にしている)。
- 送信ドメインのテナント委任 (SPF / DKIM / DMARC)。別 WI。
- ja / en 以外のロケール追加。機構は locale を一般化するが、同梱翻訳は ja / en に留める。
- 通知の opt-out 設定と対象イベントの拡張。→ [[wi-90-account-security-notification-emails]]
- lifecycle workflow 固有のテンプレート変数拡張。→
  [[wi-227-lifecycle-workflow-notification-template-customization]] (本 WI が土台を提供する)

## Plan

- **組込み既定を先に完成させ、テナント上書きは後段で足す**。第 1 段で「ja/en の組込み
  テンプレート + ロケール解決 + 既存 3 箇所の置換」を完成させると、テナント上書きが未設定でも
  日本語メールが届く状態になり、最も大きな不整合が先に解消される。上書き機構はその上に載せる。
- **placeholder は許可集合方式にする**。テンプレートごとに使える変数を宣言し、
  未知の変数を含む上書きは**保存時に拒否**する。実行時に未定義変数が空文字で消えると
  「リンクが欠けたメール」を配って復旧不能な問い合わせを生むため、保存時 fail-closed にする。
- **HTML は必ず text と同時に作る**。HTML のみのメールはテキストクライアントで読めず、
  スパム判定も悪化する。レンダラの契約を「(text, html) を返す」にして片方だけの状態を
  型で作れないようにする。
- **エスケープはレンダラの責務に閉じる**。ユーザー名などの PII が HTML 本文に入るため、
  HTML 側は必ずエスケープし、リンク URL は生成側で組み立てる (テンプレートに生 URL 結合を
  させない)。ここを最初のテストにする。
- **テスト送信の宛先固定は仕様**にする。任意宛先を許すと管理者権限を使ったメール送信の
  踏み台になる。宛先は操作者本人に固定し、UI にもそう書く。
- 未決定: テンプレート上書きの版管理 (履歴・ロールバック) は第 1 段では持たず、
  「既定へのリセット」だけを提供する。需要が出たら別 WI にする。

## Tasks

- [ ] T001 [SCL] `Tenancy` に NotificationTemplate / NotificationTemplateKey、interface 6 件、
      event 2 件、scenario 5 件を追加し、既存 RequestPasswordReset / RequestEmailChange の
      記述を更新して `just check-scl` を通す。
- [ ] T002 [ADR] 通知テンプレートとロケール解決の ADR を起票する (命名・placeholder 許可集合・
      ロケール解決順序・上書き許可範囲・text/html 併記・禁止値)。
- [ ] T003 [Renderer] `backend/shared/notification` に template registry / レンダラ /
      ロケール解決を実装する。RED: HTML エスケープ、未知 placeholder 拒否、text/html 同時生成、
      ロケールフォールバックが落ちるテストを先に書く → GREEN。
- [ ] T004 [Templates] ja / en の組込みテンプレート (PasswordReset / EmailVerification /
      EmailChangeConfirmation / LifecycleWorkflowNotification) を `embed` で追加する。
      文面は既存 UI の語彙と揃える (半端な「英語動詞 + する」を作らない)。
- [ ] T005 [Replace] 既存 3 箇所のハードコード送信をテンプレート経由に置換する。
      RED: ja ユーザーへのリセットメール件名が日本語になるテスト
      (scenario `Authentication.password_reset_localized_email`) → GREEN。
      lifecycle dispatcher の `TemplateKey` 直挿しを解消する。
- [ ] T006 [SMTP] `multipart/alternative` (text + html) 送信に対応する。RED: 生成 MIME の
      構造テスト → GREEN。console / memory sender も html を保持する。
- [ ] T007 [Persistence] `notification_templates` テーブル ((tenant_id, template_key, locale)
      一意) を `infra/schema/postgres.sql` に追加し、memory / postgres repository を実装する。
- [ ] T008 [Usecase/HTTP] 一覧 / 取得 / 更新 / リセット / プレビュー / テスト送信を実装する。
      RED: 未知 placeholder を含む更新が 400、テスト送信宛先が操作者本人に固定される
      handler テスト → GREEN。
- [ ] T009 [UI] テンプレート編集画面 (一覧・placeholder 一覧・編集・プレビュー・リセット・
      自分宛テスト送信) を追加する。i18n テストは辞書値を参照する。RED → GREEN。
- [ ] T010 [Docs] README にテンプレートキー / placeholder / ロケール解決 / テスト送信の制約を
      追記する。
- [ ] T011 [Verify] 下記 Verification を緑にする。`just scl-render` を実行する。

## Verification

- `just check` / `just check-scl` / `just check-work-items` / `just check-ids`
- `just test-go` / `just verify-go`
- `just verify-ui` / `just test-ui-unit`
- 手動: Mailpit (`EMAIL_SENDER=smtp`) で (1) 日本語ロケールのユーザーがパスワードリセットを
  要求すると日本語の件名・本文・動作するリンクのメールが届くこと、(2) HTML とテキストの
  両方が入っていること、(3) テナント上書きが反映されること、(4) テスト送信が操作者本人にしか
  飛ばないこと、を確認する。

## Risk Notes

メール本文は**ユーザーが復旧に使う唯一の導線**であり、リンク生成やロケール解決の
バグは「誰もパスワードをリセットできない」障害になる。既定テンプレートのリンク生成を
テストで固定し、テナント上書きは保存時 fail-closed にする。
HTML メール導入で XSS 相当の注入面が増える (ユーザー名などが本文に入る)。エスケープを
レンダラに閉じ、テンプレート側に生の結合をさせない。
テスト送信は任意宛先を許すとメール送信の踏み台になるため、宛先固定を仕様として実装し、
handler テストで固定する。
既存 3 箇所の置換は振る舞い変更 (件名・本文が変わる) を伴う。運用中の顧客がメール本文で
自動処理をしている可能性は低いが、README に変更点を明記する。
