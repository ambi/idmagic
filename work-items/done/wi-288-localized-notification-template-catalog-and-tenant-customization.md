---
status: completed
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
  - { context: IdGovernance, kind: model, element: WorkflowActionDef }
  - { context: Tenancy, kind: model, element: Tenant }
  - { context: Tenancy, kind: model, element: NotificationTemplate }
  - { context: Tenancy, kind: model, element: NotificationTemplateKey }
  - { context: Tenancy, kind: interface, element: ListNotificationTemplates }
  - { context: Tenancy, kind: interface, element: GetNotificationTemplate }
  - { context: Tenancy, kind: interface, element: UpdateNotificationTemplate }
  - { context: Tenancy, kind: interface, element: ResetNotificationTemplate }
  - { context: Tenancy, kind: interface, element: PreviewNotificationTemplate }
  - { context: Tenancy, kind: interface, element: SendTestNotification }
  - { context: Tenancy, kind: interface, element: GetAdminSettings }
  - { context: Tenancy, kind: interface, element: UpdateAdminSettings }
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

- [x] T001 [SCL] `Tenancy` に NotificationTemplate / NotificationTemplateKey、interface 6 件、
      event 2 件、scenario 4 件を追加し、既存 RequestPasswordReset / RequestEmailChange の
      記述を更新して `just check-scl` を通す。
      - 「ユーザーの locale → テナント既定 locale → システム既定」を実装可能にするため
        `Tenant.default_locale` (optional, `^[a-z]{2}$`) を追加し、GetAdminSettings /
        UpdateAdminSettings / TenantSummaryResponse に露出した。ADR の解決順序が
        「設定できない段」を含まないようにするための追加。
      - Scope の「`states` に event 2 件」は SCL では `models` の `kind: event` として表現した。
        上書き有無 (Default / Customized) の state machine は target model に enum field を
        要求され、常に Customized の行に status field を生やす spec ノイズになるため作らない。
      - scenario は 4 本 (ロケール解決 / 上書き優先とリセット / placeholder 拒否 /
        プレビュー非送信とテスト送信の宛先固定)。Scope の 5 項目のうち「上書きが無い
        テンプレートは組込み既定にフォールバックする」は独立 scenario ではなく
        ロケール解決 scenario の given と上書き scenario の extension で受け入れ条件化した。
- [x] T002 [ADR] [[ADR-142-notification-template-catalog-and-locale-resolution]] を起票した
      (2 段解決と版管理を持たない理由、固定 key、placeholder 許可集合と保存時 fail-closed、
      3 点セットのレンダラ契約、エスケープの所在、上書き許可範囲と HTML 文書外枠の所有、
      ロケール解決順序、テスト送信の宛先固定、プレビューのサンプル値、禁止値、
      shared / Tenancy の責務分割、列持ちの永続化)。
- [x] T003 [Renderer] `backend/shared/notification/template` に registry / レンダラ /
      ロケール解決 / Notifier を実装した。
      - RED: stub 実装で `TestRenderEscapesVariablesInHTMLOnly` /
        `TestValidateDefinitionRejectsUnknownPlaceholder` /
        `TestValidateDefinitionRequiresSubjectTextAndHTMLTogether` /
        `TestResolveLocaleFollowsRecipientThenTenantThenSystem` /
        `TestNotify*` が意図した理由で fail することを確認 → GREEN。
      - `body_html` はテナントが文書全体を差し替えられないよう `<body>` 内 fragment とし、
        doctype / head / コンテナのスタイルは `wrapHTMLDocument` が供給する (ADR-142 決定 6)。
        Scope の「任意 HTML/CSS の全面差し替えは不可」を構造で担保するための追加決定。
      - fuzz/property test は採用しない (ADR-121)。差し込み変数の文法は再帰も入れ子も無い
        単一パターン `{{name}}` で、テンプレート本文は認証済みテナント管理者が入力し
        保存時に許可集合で検査される。攻撃面は「未信頼入力の複雑な文法の解釈」ではなく
        「差し込み値のエスケープ」であり、そこは table test で固定した。
- [x] T004 [Templates] ja / en の組込みテンプレートを `embed` (`defaults/{ja,en}.yaml`) で
      追加した。key は PasswordReset / EmailVerification / EmailChangeConfirmation /
      AccountSecurityAlert / LifecycleWorkflowNotification の 5 件。
      Scope の enum に AccountSecurityAlert があるため、既定文面を欠いた key を作らないよう
      T004 の 4 件に加えてこれも同梱した (送信経路は wi-90 が足す)。
      `TestBuiltinCatalogIsCompleteAndValid` で全 key × 全 locale の完全性・許可集合適合・
      サンプル値での描画可能性を固定した。
- [x] T005 [Replace] 既存 3 箇所のハードコード送信を `ports.Notifier` 経由に置換した。
      3 つの Deps から `EmailSender` を外し、文面を組み立てる責務を use case から除いた。
      - RED: `TestRequestPasswordResetLocalizesToTheRecipientLocale` /
        `TestRequestPasswordResetFallsBackToSystemDefaultLocale` /
        `TestRequestEmailChangeLocalizesToTheRecipientLocale` /
        `TestLifecycleWorkflowRunHandlerSendsCatalogTemplateForSendEmail` を先に fail
        確認 → GREEN (scenario `Tenancy: 日本語ロケールのユーザーには日本語の
        パスワードリセットメールが届く`)。
      - locale の第 1 段に使う `User.LocaleAttribute()` と宛名の `User.DisplayName()` を
        domain に追加。RED: `TestUserLocaleAttribute` / `TestUserNotificationDisplayName`
        を先に fail 確認 → GREEN。
      - lifecycle dispatcher の `TemplateKey` 直挿しは解消し、カタログの
        `LifecycleWorkflowNotification` に `notification_key` として差し込む。
      - `server_http` の旧互換入力 (EmailSender だけを渡すテスト) からも組込み既定
        カタログ経由になるよう、`mergeLegacyNotificationDeps` で Notifier を補う。
- [x] T006 [SMTP] `multipart/alternative` (text + html) 送信は ADR-035 §8 の実装で既に
      対応済みだった (`buildRFC5322Message`)。本 WI で不足していた差出人表示名の上書きを追加。
      - RED: `TestBuildRFC5322MessageAppliesFromDisplayName` を先に fail 確認 → GREEN。
        表示名は `mail.Address` 経由で quoting / MIME encoding し、CR/LF は事前に潰す
        (`TestBuildRFC5322MessageSanitizesFromDisplayName`)。アドレスはサーバ設定のまま。
      - memory sender は `EmailMessage` をそのまま保持するので html も保持済み。console
        sender は html の有無を出力に追加した。
- [x] T007 [Persistence] `notification_templates` テーブル (PK: (tenant_id, template_key,
      locale)) と `tenants.default_locale` 列を `infra/schema/postgres.sql` に追加し、
      memory / postgres repository を実装した。
      - RED: `TestNotificationTemplateRepositorySaveFindListDelete` /
        `TestNotificationTemplateRepositoryIsolatesTenants` /
        `TestTenantRepositoryPersistsDefaultLocale` を先に fail 確認 → GREEN。
      - Tenancy には専用 domain 型を作らず、`shared/notification/ports` の
        `TemplateKey` / `TemplateOverride` を repository の語彙にした。送信側と編集側で
        key 集合が二重定義になると食い違いうるため (ADR-142 決定 11 の責務分割に対応)。
- [x] T008 [Usecase/HTTP] 一覧 / 取得 / 更新 / リセット / プレビュー / テスト送信を実装した。
      - RED (usecase): `TestUpdateNotificationTemplateRejectsInvalidInput` /
        `TestPreviewNotificationTemplateDoesNotSendOrSave` /
        `TestSendTestNotificationGoesToTheActorOnly` /
        `TestSendTestNotificationRequiresAnActorAddress` /
        `TestUpdateThenResetNotificationTemplate` を先に fail 確認 → GREEN。
      - RED (handler): `TestNotificationTemplateUpdateRejectsUnknownPlaceholder` (400) /
        `TestNotificationTemplateTestSendGoesToTheActorOnly` (宛先固定) /
        `TestNotificationTemplateUpdateAndReset` (イベント 2 種) を先に fail 確認 → GREEN。
      - テスト送信ハンドラはリクエストボディから宛先を読む経路を持たない (型に無い)。
      - `default_locale` の管理 API も同時に実装。RED: `TestUpdateDefaultLocale` /
        `TestAdminSettingsExposeDefaultLocale` → GREEN。
      - bootstrap に `AssembleNotification` を追加し、API と worker の両プロセスが
        同じ Notifier (テナント上書き + locale 解決込み) を使うようにした。
        RED: `TestAssembleNotificationAppliesTenantOverrides` /
        `TestAssembleNotificationUsesTenantDefaultLocale` /
        `TestAssembleNotificationHonorsSystemDefaultLocale` /
        `TestAssembleNotificationRejectsUnsupportedSystemDefaultLocale` → GREEN。
        worker はこれまで EmailSender を組み立てておらず lifecycle の send_email が
        常に blocked だった。組み立てを bootstrap に寄せた副次効果で解消している。
- [x] T009 [UI] `NotificationTemplatesTab` を追加し、設定画面の「メール」タブ (これまで
      「近日公開」の無効タブ) を置き換えた。一覧 (key × locale・上書き有無・現在の件名)、
      placeholder 一覧、編集、プレビュー、既定へのリセット、自分宛テスト送信を提供する。
      - RED: `NotificationTemplatesTab.test.tsx` 10 本を先に fail 確認 → GREEN。
        宛先を指定する手段が無いこと (`expect(init.body).toBeUndefined()`)、3 点セットの
        欠けをクライアント側でも止めること、プレビューが送信を起こさないことを固定した。
      - i18n テストは辞書値 (`notificationTemplatesTabDictionary.en.*` /
        `commonDictionary.en.networkError`) を参照する。
      - プレビューの HTML は管理画面の DOM に流し込まず `sandbox=""` の iframe に隔離する。
        差し込み値はサーバがエスケープ済みだが、テンプレート本体はテナント管理者が書いた
        markup なので、管理画面と同じ実行文脈に置かない。
      - GeneralTab にテナント既定 locale の選択を追加した。
        RED: `TestAdminSettingsPage > updates the tenant default locale from the general tab` /
        `opens the notification template catalog from the email tab` → GREEN。
- [x] T010 [Docs] README に「Notification Email Templates」節を追加し、テンプレートキー一覧と
      placeholder、`{{name}}` 記法と保存時 fail-closed、locale 解決順序、上書き可能範囲、
      テスト送信の宛先固定、`DEFAULT_LOCALE` 環境変数、既存 3 経路の文面が変わる旨を記載した。
- [x] T011 [Verify] `just verify` (check / traceability-strict / test-go / lint-go /
      test-tools / typecheck-tools / format-check-ui / lint-ui / test-ui-unit / build-ui) と
      `just test-go-race` が緑。`just scl-render` を実行し派生物 (idmagic.html /
      models.schema.json / openapi.json) を同期した (`TestAssembledRoutesMatchGeneratedOpenAPI`
      が新 endpoint との一致を要求する)。
      - `just verify` を通すために既存の lint 不整合も直した: `patchSettings` の
        常に同値の `path` 引数を削除 (unparam)、bootstrap 配線に contextcheck の
        nolint 理由を付与、`TenantNotificationSource.FindTemplateOverride` の
        「上書き無し = nil」契約を `.golangci.yml` の既存 Find* 除外と同じ形で登録。

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

## Completion

- **Completed At**: 2026-07-26
- **Summary**: 通知メールを「組込み既定カタログ + テナント上書き」の 2 段解決に寄せ
  ([[ADR-142-notification-template-catalog-and-locale-resolution]])、usecase 内にハードコード
  されていた英語プレーンテキスト 3 箇所を全て置換した。`backend/shared/notification/template`
  が ja / en の組込み既定 (`embed` した locale 別 YAML)、`{{name}}` 記法のレンダラ
  (許可集合検査・HTML エスケープ・text/html 同時生成・文書外枠)、locale 解決、Notifier を
  所有し、テナント上書きの永続化と管理 API は Tenancy が持つ。lifecycle dispatcher の
  `TemplateKey` 直挿しも解消した。管理コンソールには一覧・編集・placeholder 表示・
  プレビュー・既定へのリセット・自分宛テスト送信を持つ「メール」タブが入った
  (これまで「近日公開」の無効タブ)。
- **Human Decisions**: なし (WI の Plan と Scope の範囲で判断)。
- **Verification Results**:
  - `just verify` - passed (10 check 並列すべて ok)
  - `just test-go-race` - passed (data race 無し)
  - `just lint-go` - passed (0 issues)
  - `just scl-render` - passed (`spec/idmagic.html` / `.models.schema.json` / `.openapi.json` を
    再生成。`TestAssembledRoutesMatchGeneratedOpenAPI` が新 endpoint 6 件との一致を要求する)
  - 手動 (Mailpit, `EMAIL_SENDER=smtp`, development seed): パスワードリセット要求で
    (a) 既定 locale が en のとき英語の件名 `Reset your IdMagic password`、
    (b) `DEFAULT_LOCALE=ja` のとき日本語の件名 `IdMagic のパスワード再設定` が届く。
    いずれも text part と HTML part の両方を含み、本文のリンクは
    `http://localhost:8099/realms/default/reset_password?token=...` で、
    その token を `POST /api/auth/reset_password` に渡して 200 (再設定成功) を確認した。
- **Affected Guarantees State**: 新規 requirement の `adoption` 変更は無い。既存の保証は
  維持し、`RequestPasswordReset` の anti-enumeration (email の存在有無に関わらず 204)、
  検証済みアドレスのみへの送信 (CWE-640)、SMTP ヘッダ injection 対策は不変。
  差出人表示名の上書きを追加したが、アドレスはサーバ設定のままで、表示名は
  `mail.Address` の quoting / MIME encoding と CR/LF 除去を通す。
- **Semantic Diff**:
  - **`Tenant.default_locale` を追加した (Scope に無い追加)**。WI の decision scope が
    「ユーザーの locale → テナント既定 locale → システム既定」の解決順序を記録するよう
    求めており、テナント段に設定手段が無いと解決順序が実質 2 段になる。ADR に
    「設定できない段」を書き残すより、フィールドを足して 3 段を実装した。露出は
    GetAdminSettings / UpdateAdminSettings / TenantSummaryResponse と設定画面の一般タブ。
  - **`body_html` を `<body>` 内 fragment に限定した (ADR-142 決定 6)**。Out of Scope の
    「任意 HTML / CSS の全面差し替え」を構造で担保するため、doctype / head / コンテナの
    体裁はレンダラの `wrapHTMLDocument` が供給する。テナントは文書全体を差し替えられない。
  - **Tenancy に通知テンプレート専用の domain 型を作らなかった**。`TemplateKey` /
    `TemplateOverride` は `shared/notification/ports` の型を Tenancy の repository / usecase /
    handler がそのまま使う。key 集合を送信側と編集側で二重定義すると食い違いうる。
  - **上書き有無の state machine を作らなかった**。Scope は「`states` に event 2 件」と
    書いているが、SCL の `states` は target model の enum field に解決する必要があり、
    常に Customized の行に status field を生やすことになる。event 2 件は `models` の
    `kind: event` として追加した。
  - **`AccountSecurityAlert` の既定文面も同梱した**。Scope の enum には含まれるが T004 の
    テンプレート列挙には無く、既定文面を欠いた key はカタログを壊す (一覧・編集が失敗する)。
    送信経路は [[wi-90-account-security-notification-emails]] が足す。README でどの key が
    現在送信されるかを明記した。
  - **通知の組み立てを `bootstrap.AssembleNotification` に寄せた**。副次的に、worker
    プロセスが EmailSender を組み立てておらず lifecycle の `send_email` が常に blocked
    だった既存の欠落が解消している。
  - **`DEFAULT_LOCALE` は未対応値で起動失敗にする**。silent fallback にすると誤設定が
    「なぜか英語で届く」として運用中に発覚する。
- **Residual Risk**: 既存 3 経路の件名・本文が変わる (README に記載)。テナント上書きの
  版管理は持たず、復旧手段は「既定に戻す」のみ (ADR-142 決定 1)。テンプレート編集画面の
  テナント上書き反映とテスト送信の宛先固定は自動テスト (usecase / handler / bootstrap) で
  固定しており、ブラウザからの手動操作では未確認 (ローカルの development seed では
  admin セッションの確立に OIDC 認可トランザクションが必要で、curl では駆動できなかった)。
