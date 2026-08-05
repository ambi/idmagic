---
status: accepted
authors: [tn]
created_at: 2026-07-25
---

# ADR-142: 通知メールを組込み既定カタログ + テナント上書きの 2 段で解決し、locale は受信者 → テナント → システムの順に決める

## コンテキスト

通知メールの文面は Go の usecase 内にハードコードされた英語プレーンテキストだった (`request_password_reset.go` の `Subject: "Password reset"`、`request_email_change.go` の `Subject: "Confirm your new email address"`、lifecycle dispatcher は `action.TemplateKey` を件名と本文にそのまま入れる実質未実装のプレースホルダ)。UI は ja / en にローカライズ済み ([[ADR-105-system-runtime-hardening-and-i18n-tooling]]) なので、日本語 UI でパスワードを忘れた利用者にだけ英語メールが届く不整合が残っていた。競合 (Keycloak のテーマ別 freemarker テンプレート + メッセージバンドル、Okta の管理画面で編集可能な全テンプレート、Entra ID の企業ブランディング適用) はいずれもここを標準機能として持つ。

メール本文は**利用者が復旧に使う唯一の導線**である。リンク生成や locale 解決のバグは「誰もパスワードをリセットできない」障害になり、テンプレートを編集可能にすればテナント管理者の入力ミスがそのまま復旧不能を生みうる。同時に HTML メールを導入すると、利用者名などの PII が本文に入るため XSS 相当の注入面が増える。カスタマイズの自由度と、復旧導線の壊れにくさ・注入面の狭さのどちらを優先するかを決める必要がある。

[[ADR-096-tenant-branding-value-and-logo-storage]] は hosted UI ブランディングについて「安全な範囲 (画像 + 限定トークン + テキスト / リンク) に絞り、任意 CSS / HTML は構造的に受け付けない」を既に選んでいる。通知メールは同じ緊張を持つ別の面であり、方針を揃えるか分けるかもここで決める。

## 決定

`spec/contexts/tenancy.yaml` の `models.NotificationTemplate` / `models.NotificationTemplateKey` / `models.Tenant.default_locale` と `interfaces.ListNotificationTemplates` / `GetNotificationTemplate` / `UpdateNotificationTemplate` / `ResetNotificationTemplate` / `PreviewNotificationTemplate` / `SendTestNotification`、`spec/contexts/authentication.yaml` の `interfaces.RequestPasswordReset`、`spec/contexts/identity-management.yaml` の `interfaces.RequestEmailChange`、`spec/contexts/identity-governance.yaml` の `models.WorkflowAction.template_key` に反映。

通知メールの文面は「組込み既定カタログ」と「テナント上書き」の 2 段のみで解決し、版管理を持たない。`template_key` は固定 enum でテナントは追加できず、送信経路と 1:1 対応させる。差し込み変数は key ごとの許可集合を宣言し、許可外は保存時に fail-closed で拒否する。レンダラは (件名, テキスト本文, HTML 本文) を常に 3 点セットで返し、エスケープの責務はレンダラ側に閉じてテンプレート側に生の文字列結合をさせない。上書き可能なのは件名 / HTML 本文 fragment / テキスト本文 / 差出人表示名のみで、HTML 文書の外枠と差出人メールアドレスはテナントから変更できない。locale は受信者 → テナントの `default_locale` → システム既定の順で解決する。テスト送信は操作者本人の検証済みアドレス宛に固定し、プレビューはサンプル値のみで保存・送信を行わない。カタログとレンダラは `backend/shared/notification` に置き、テナント上書きの永続化と管理 API は Tenancy が所有する。

これらは、[却下した代替案](#却下した代替案) に挙げた対抗案 (テーマファイル差し替え、未知 placeholder の実行時空文字化、HTML のみの部分上書き、テスト送信の任意宛先、locale のテナント段を上書き有無から推定、ja/en の enum 化) がそれぞれ抱える障害モード — テンプレート言語評価による注入面の拡大、検知が遅れる復旧不能な問い合わせ、テナント管理者権限の踏み台化、非直感的な言語切り替え、翻訳追加のたびのマイグレーション — を避けるために選んだ。

現在のメカニズムの詳細 (二段解決アルゴリズム、許可集合、レンダラ契約、エスケープ責務の分割、locale 解決順、テスト送信・プレビューの制約、key ごとの placeholder 表) は [ARCHITECTURE.md](../ARCHITECTURE.md) の Cross-cutting Concerns → Persistence → Database design policy「Notification template catalog and locale resolution」を参照。

## 却下した代替案

- **Keycloak 型の「テーマごとのテンプレートファイル差し替え」**: freemarker / Go template の任意テンプレートをテナントが持ち込める設計。表現力は最大だが、テンプレート言語の評価が注入面になり (エスケープ責務がテンプレート作成者に移る)、テナント入力を評価する sandbox が必要になる。決定 3 / 5 の「許可集合 + レンダラにエスケープを閉じる」と両立しない。
- **未知 placeholder を実行時に空文字列へ潰す**: 保存が常に成功するので編集体験は良い。しかし失敗が「利用者に届いたリンクの無いメール」として現れ、検知が遅く、影響が復旧不能な問い合わせになる。編集時の摩擦と復旧不能の非対称性から fail-closed を採る。
- **HTML 本文を optional にしてテキストのみの上書きを許す**: 移行が楽だが、テナントが件名とテキストだけ上書きした結果 HTML 側が既定のまま残り、同じメールの 2 つの part が別の文面になる。ユーザーのメールクライアントによって内容が違うという再現困難な問い合わせを生む。
- **テスト送信の宛先を任意にする (Okta / Keycloak の一部に相当)**: 実運用の配送経路 (別ドメイン宛の到達性) を検証できる利点はあるが、テナント管理者に任意宛先メール送信権を与えることになる。到達性検証は運用側が SMTP 設定として別に持つべき関心で、テンプレート編集面から与える必要はない。
- **locale をテナント上書き行の有無から推定する (上書きがある locale を優先する)**: `Tenant.default_locale` を追加せずに 3 段目を作る案。しかし「テンプレートを 1 件 ja で上書きしたら全通知が日本語になる」という非直感的な結合を生み、上書き対象の追加が locale 方針の変更として波及する。
- **locale を ja / en の enum にする**: 検証が単純になるが、翻訳を足すたびに SCL の enum とマイグレーションが必要になる。同梱翻訳を ja / en に留める判断とは独立に、機構は言語タグ文字列で一般化しておく。

## 影響

- 新規 SCL: `Tenancy` の `models.NotificationTemplate` / `NotificationTemplateKey` / `NotificationTemplateSummary` / `NotificationTemplateListResponse` / `NotificationTemplateDetail` / `NotificationTemplateUpdateRequest` / `NotificationTemplatePreviewRequest` / `NotificationTemplatePreviewResponse` / `NotificationTemplateTestSendResponse` / `NotificationTemplateUpdated` / `NotificationTemplateReset`、`interfaces.ListNotificationTemplates` / `GetNotificationTemplate` / `UpdateNotificationTemplate` / `ResetNotificationTemplate` / `PreviewNotificationTemplate` / `SendTestNotification`。
- 変更 SCL: `Tenancy` の `models.Tenant` に `default_locale`、`TenantUpdateRequest` / `TenantSummaryResponse` / `AdminSettingsResponse` に `default_locale` (と `supported_locales`)、`interfaces.GetAdminSettings` / `UpdateAdminSettings`。`Authentication.interfaces.RequestPasswordReset` と `IdManagement.interfaces.RequestEmailChange` の文面解決の記述。`IdGovernance.models.WorkflowAction.template_key` の意味 (本文への直挿しから差し込み変数へ)。
- データ: `notification_templates` テーブル新設、`tenants.default_locale` 列追加。
- 振る舞い変更: 既存 3 経路のメールの件名と本文が変わる。ja の利用者には日本語で届くようになり、全メールが text + HTML の 2 part になる。lifecycle 通知は template key を本文に直挿しするのをやめる。README に変更点を記載する。
- 運用: `DEFAULT_LOCALE` 環境変数 (未設定なら `en`) を追加。
- 範囲外として残す: SMS / push のテンプレート ([[wi-295-email-otp-and-sms-otp-mfa-factors]])、送信ドメインのテナント委任 (SPF / DKIM / DMARC)、ja / en 以外の同梱翻訳、通知の opt-out ([[wi-90-account-security-notification-emails]])、上書きの版管理。
