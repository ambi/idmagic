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

1. **文面は「組込み既定カタログ」と「テナント上書き」の 2 段でのみ解決し、版管理を持たない。** 組込み既定はシステムが同梱する ja / en の文面で、テナントは編集できない。テナント上書きは `(tenant_id, template_key, locale)` 単位の全置換で、削除 (`ResetNotificationTemplate`) すると常に組込み既定へ戻る。「直前の上書きへ戻す」履歴・ロールバックは持たない。復旧導線が壊れたときに管理者が最初に取れる行動が「確実に動く既定へ戻す」ことであり、そこに版選択という判断を挟ませない方が復旧が速い。履歴の需要が出たら別 ADR で足す。

2. **template_key はシステムが持つ固定の enum で、テナントは key を追加できない。** key を増やせる設計は「どの key がどの経路から送られるか」を仕様から追えなくし、送信経路の無い孤児テンプレートを許す。key は送信経路と 1:1 で対応する語彙として SCL の `NotificationTemplateKey` に固定する。`AccountSecurityAlert` はまだどの経路からも送信されないが、カタログと編集面を先に用意する ([[wi-90-account-security-notification-emails]] が送信側を足す)。README でどの key が現在送信されるかを明記し、孤児を仕様外の偶然にしない。

3. **差し込み変数 (placeholder) は `{{name}}` 記法で、template_key ごとの許可集合を宣言し、許可外の変数を含む上書きは保存時に拒否する (fail-closed)。** 実行時に未定義変数を空文字列へ潰す方式は採らない。潰すと「リンクが欠けたメール」が利用者へ配られ、管理者は送信後に初めて気づき、その時点では既に復旧不能な問い合わせが発生している。保存時に拒否すれば、失敗は編集者の画面に閉じる。許可集合は API (`GetNotificationTemplate.placeholders`) で返し、編集者が推測しなくてよい状態にする。

4. **レンダラの契約は「(件名, テキスト本文, HTML 本文) を 3 つ同時に返す」とし、HTML だけ / テキストだけの状態を型で作れないようにする。** HTML 単独のメールはテキストクライアントで読めず、スパム判定も悪化する。上書きも 3 点セットの全置換とし、片方だけの保存を受け付けない。送信は `multipart/alternative` (ADR-035 §8 が既に確立した経路) に載せる。

5. **エスケープはレンダラの責務に閉じ、テンプレート側に生の文字列結合をさせない。** HTML 側は差し込み値を必ず HTML エスケープして展開し、テキスト側は素で展開する。リンク URL は送信側 (usecase) がリクエストの発行元 URL から組み立てて placeholder 値として渡し、テンプレートには「URL を 1 個の値として置く」ことしか許さない。テンプレート内で `{{issuer}}/reset?token={{token}}` のような結合を許すと、エスケープ責務がテンプレート編集者に漏れる。

6. **上書きできるのは件名 / HTML 本文 fragment / テキスト本文 / 差出人表示名の 4 つだけで、HTML 文書の外枠・差出人メールアドレスは上書きできない。** `body_html` は `<body>` 内に入る fragment として保存し、doctype / `<head>` / 文字エンコーディング / viewport / 本文コンテナのスタイルはレンダラが持つシステム所有の外枠が供給する。テナントが「文書ごと差し替える」経路を構造的に作らないためであり、ADR-096 が hosted UI で任意 CSS を構造的に拒否したのと同じ理由に立つ。fragment 内の markup 自体はテナント管理者が書ける (受信者はそのテナント自身の利用者であり、悪意ある管理者は上書き機構が無くても自テナントの利用者を騙せる) が、**差し込み値のエスケープは決定 5 によりレンダラ側に残る**。したがってテンプレート編集で起こしうる事故は「自テナントのメールの見た目が崩れる」に留まり、利用者データ由来の注入にはならない。差出人アドレスを可変にすると SPF / DKIM / DMARC の整合をテナント入力に依存させることになるため、上書きできるのは表示名だけとする。送信ドメインのテナント委任は独立の決定であり、本 ADR の範囲外。

7. **locale は「受信者 User の locale 属性 → テナントの `default_locale` → システム既定 locale」の順に、カタログが同梱翻訳を持つ最初の locale を採る。** 未対応 locale は飛ばす。この 3 段のうちテナント段が設定できないと解決順序が実質 2 段になるため、`Tenant.default_locale` を本決定と同時に追加する (「設定できない段」を仕様に書き残さない)。システム既定は UI と同じ `FallbackLocale = en` を既定値とし、`DEFAULT_LOCALE` 環境変数で運用側が変えられる。UI の `ConfiguredDefaultLocale` (`VITE_DEFAULT_LOCALE`) と同じ役割をサーバ側に置く。同梱翻訳は ja / en に留めるが、locale は enum ではなく言語タグ文字列として扱い、翻訳を足すだけで locale を増やせる形にする。

8. **テスト送信の宛先は操作した管理者本人の検証済みメールアドレスに固定し、リクエストで宛先を指定する手段を提供しない。** 任意宛先を許すと、テナント管理者権限が「認証済み SMTP 経由で任意の相手に自社ブランドのメールを送る」踏み台になる。テンプレート編集の目的 (自分の書いた文面が実際にどう届くか) は本人宛で満たせる。検証済みアドレスを持たない操作者は拒否する。

9. **プレビューは保存も送信もしない読み取り操作とし、サンプル値で描画する。** 実データ (実在の利用者名やトークン) をプレビューに流すと、編集画面が利用者情報の閲覧経路になる。サンプル値はカタログが持つ固定値とする。

10. **テンプレート本文に置いてよい値は placeholder 許可集合が列挙するものだけであり、資格情報 (パスワード・パスワードハッシュ・TOTP secret・API トークン)、単発トークン以外の機微値、生の IP アドレスは許可集合に入れない。** メールは転送・引用・長期保存され、受信側のメールボックス侵害でそのまま漏れる。復旧に必要な単発トークンは URL の一部として 1 個の placeholder に閉じ、それ自体を本文に平文表示しない。生 IP は「不審なサインイン」通知で入れたくなるが、位置情報の推定に使え、NAT 環境では誤って第三者を指す。

11. **カタログとレンダラは `backend/shared/notification` に置き、テナント上書きの永続化は Tenancy が所有する。** 送信は Authentication / IdManagement / IdGovernance の 3 context から起きるため、文面解決を特定の context に置くと他 context がそれに依存する。一方でテナント単位の設定は Tenancy の既存責務 ([[ADR-096-tenant-branding-value-and-logo-storage]] の TenantBranding と同型) なので、上書き行と管理 API は Tenancy に置き、`shared/notification` は上書きの取得を port として受ける。

12. **上書きは `notification_templates` テーブル ((tenant_id, template_key, locale) 一意) の個別列として持ち、JSONB にまとめない。** ADR-096 決定 7 と同じ理由 (列ごとの長さ制約を CHECK として書ける)。

## テンプレートキーごとの placeholder 許可集合

許可集合そのものは実装 (`backend/shared/notification/template`) が正本で、API が返す。ここに記録するのは**なぜその粒度か**だけである。

- 全 key 共通: `product_name` / `tenant_display_name` / `user_display_name`。ブランディングと宛名は全通知で必要になり、key ごとに差を付ける理由がない。
- 単発リンクを持つ key (`PasswordReset` / `EmailVerification` / `EmailChangeConfirmation`): リンクは `*_url` の 1 変数、有効期限は `expires_in_minutes` の 1 変数。トークン単体の placeholder は置かない (決定 10)。
- `EmailChangeConfirmation` のみ `new_email` を持つ。確認先アドレスの取り違えを受信者が判断できる必要がある。
- `LifecycleWorkflowNotification` のみ `notification_key` を持つ。workflow 作成者が付けたラベルを本文に出せるようにする。workflow ごとに別テンプレートを割り当てる機構は [[wi-227-lifecycle-workflow-notification-template-customization]] の範囲。
- `AccountSecurityAlert` は `event_description` / `occurred_at` を持つ。生 IP・User-Agent は決定 10 により入れない。

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
