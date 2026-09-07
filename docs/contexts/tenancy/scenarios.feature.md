# Feature: Tenancy Scenarios

## Rule: REQ-TENANCY-001 管理者は正規ロケーションの連携情報を取得する

Primary actor: `TenantAdministrator`

### Example: EX-TENANCY-001-01 通常経路

- Given admin が path または サブドメインの正規ロケーションから自身のテナントへアクセスしている
- When admin が連携エンドポイント画面を開く
- Then サーバーはリクエスト先テナントの正規の発行者から OAuth/OIDC、SAML、WS-Federation、SCIM、管理 API、本人用 API の URL を導出する
- Then 画面は OAuth/OIDC、SAML、WS-Federation、API のプロトコル単位で情報をまとめ、SAML 配下ではデフォルトを含むプロファイルごとにエンティティ ID、メタデータ、SSO、SLO、署名証明書を一組で表示する
- Then 画面は読み取り専用であり、Discovery Metadata と各プロトコルのメタデータを正本として案内し、個別値をコピーまたは証明書をダウンロードできる
- Then 正規の発行者と同じオリジンで配信するゲートウェイは、表示した公開プロトコル URL を対応するサーバーのエンドポイントへ転送する
- Then レスポンスにクライアントシークレット、API トークン、秘密鍵は含まれない

### Example: EX-TENANCY-001-02 admin が別テナントの realm を URL として指定しようとする

- Given admin が path または サブドメインの正規ロケーションから自身のテナントへアクセスしている
- When admin が連携エンドポイント画面を開く
- But admin が別テナントの realm を URL として指定しようとする
- Then 対象指定パラメータは存在せず、解決済みテナント以外の情報は返らない

## Rule: REQ-TENANCY-002 管理者はテナント固有のユーザー属性スキーマを定義できる

Primary actor: `TenantAdministrator`

### Example: EX-TENANCY-002-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が editable_by_user=true の custom_attribute を追加する
- Then 更新後のスキーマに追加した属性が含まれる

## Rule: REQ-TENANCY-003 default テナントは起動時に自動作成され削除も無効化もできない

Primary actor: `System`

### Example: EX-TENANCY-003-01 通常経路

- When IdP を起動する
- Then テナント "default" が status=Active で存在する

### Example: EX-TENANCY-003-02 default テナントの削除を試みる

- When IdP を起動する
- Then default テナントの削除を試みる
- Then default テナントを削除する API は提供されない

### Example: EX-TENANCY-003-03 default テナントの無効化を試みる

- When IdP を起動する
- Then default テナントの無効化を試みる
- Then default テナントの disable は InvalidRequestError で拒否される

## Rule: REQ-TENANCY-004 管理者はテナントのロゴと配色をカスタマイズでき利用者のログイン画面に反映される

Primary actor: `TenantAdministrator`

### Example: EX-TENANCY-004-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PNG ロゴをアップロードする
- Then アップロードレスポンスに logo_url が含まれる
- When "operator" が logo_url を GET する
- Then 同じ realm の検証済み PNG が返る
- Then 管理画面のロゴプレビューにアップロードした PNG が表示される
- When "operator" が primary_color / accent_color / footer_link_1={label: "ヘルプ", url: "https://help.example.test"} / footer_text を設定する
- Then 管理画面は各設定済み色に現在値と「デフォルトに戻す」操作を表示する
- When 管理者がプライマリカラーをデフォルトに戻して保存する
- Then UpdateTenantBranding には primary_color の空文字列が送られる
- When 未認証の利用者が login 画面を開く
- Then login / 同意 / account portal に設定したロゴが表示され、login 画面にはプライマリカラーのシステムデフォルト・設定済みアクセントカラー・指定ラベルの footer リンク・フッターテキストも表示される

### Example: EX-TENANCY-004-02 別テナントの id で同じ kind のアセット取得を試みる

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PNG ロゴをアップロードする
- Then アップロードレスポンスに logo_url が含まれる
- When "operator" が logo_url を GET する
- But 別テナントの id で同じ kind のアセット取得を試みる
- Then アセットは存在しないものとして扱われ InvalidRequestError で拒否される

### Example: EX-TENANCY-004-03 realm 配下の logo_url が gateway で backend に転送されない

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PNG ロゴをアップロードする
- Then アップロードレスポンスに logo_url が含まれる
- When "operator" が logo_url を GET する
- Then 同じ realm の検証済み PNG が返る
- Then 管理画面のロゴプレビューにアップロードした PNG が表示される
- When "operator" が primary_color / accent_color / footer_link_1={label: "ヘルプ", url: "https://help.example.test"} / footer_text を設定する
- Then 管理画面は各設定済み色に現在値と「デフォルトに戻す」操作を表示する
- When 管理者がプライマリカラーをデフォルトに戻して保存する
- Then UpdateTenantBranding には primary_color の空文字列が送られる
- When 未認証の利用者が login 画面を開く
- Then realm 配下の logo_url が gateway で backend に転送されない
- Then 画像取得は成功せず、管理者は設定の成功レスポンスだけでは表示可能と判断しない

## Rule: REQ-TENANCY-005 不正な branding 入力は拒否されシステムデフォルトにフォールバックする

Primary actor: `TenantAdministrator`

### Example: EX-TENANCY-005-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が branding を一度も設定していないテナントで login 画面を開く
- Then login 画面はシステムデフォルト (IdMagic) のブランディングを表示する
- When "operator" が footer_link_1.url に javascript: スキームを指定して保存する
- Then InvalidRequestError で拒否され保存されない
- When "operator" が低コントラストの `#eeeeee` を primary_color に指定して保存する
- Then 保存に成功し、取得した branding と login 画面に `#eeeeee` が反映される
- When 管理者が SVG ファイルをロゴとしてアップロードする
- Then InvalidRequestError で拒否され保存されない

### Example: EX-TENANCY-005-02 footer_link_1 に label だけを指定する

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が branding を一度も設定していないテナントで login 画面を開く
- Then login 画面はシステムデフォルト (IdMagic) のブランディングを表示する
- When "operator" が footer_link_1.url に javascript: スキームを指定して保存する
- But footer_link_1 に label だけを指定する
- Then InvalidRequestError で拒否され保存されない

## Rule: REQ-TENANCY-006 path style のテナントは realm prefix から解決される

Primary actor: `OAuth2Client`

### Example: EX-TENANCY-006-01 通常経路

- Given テナント "default" の endpoint_style は Path である
- When "/realms/default/authorize" にリクエストを送る
- Then 解決されたテナントは "default"
- Then iss claim はベースURL + /realms/default

### Example: EX-TENANCY-006-02 対象テナントが無効化されている

- Given テナント "default" の endpoint_style は Path である
- When "/realms/default/authorize" にリクエストを送る
- But 対象テナントが無効化されている
- Then  tenant_id "acme" を作成して無効化する
- And  無効化済みテナントの "/realms/acme/authorize" にリクエストを送る
- And  テナントの存在を漏らさずエラー "InvalidRequestError"

### Example: EX-TENANCY-006-03 realm prefix を持たない "/authorize" にリクエストを送る

- Given テナント "default" の endpoint_style は Path である
- When "/realms/default/authorize" にリクエストを送る
- But realm prefix を持たない "/authorize" にリクエストを送る
- Then テナントは解決されず 404 tenant_not_found になる
- And 任意のリクエストが default テナントへ落ちることはない

## Rule: REQ-TENANCY-007 subdomain style のテナントは Host から解決される

Primary actor: `EndUser`

### Example: EX-TENANCY-007-01 通常経路

- Given `tenant_base_domain` が設定されている
- And テナント "acme" の endpoint_style は Subdomain である
- When Host "acme.{tenant_base_domain}" の "/authorize" にリクエストを送る
- Then 解決されたテナントは "acme" で、その branding のログイン画面が表示される
- Then セッション cookie は __Host- prefix と Path=/ を持ち Domain 属性を持たない
- Then WebAuthn RP ID は "acme.{tenant_base_domain}" である

## Rule: REQ-TENANCY-008 未知のサブドメインは default テナントに解決されない

Primary actor: `OAuth2Client`

### Example: EX-TENANCY-008-01 通常経路

- Given `tenant_base_domain` が設定されている
- And realm "unknown" のテナントは存在しない
- When Host "unknown.{tenant_base_domain}" の "/authorize" にリクエストを送る
- Then 404 tenant_not_found になり、default テナントにも他のどのテナントにも到達しない

## Rule: REQ-TENANCY-009 テナントは自分の正規ロケーション以外からは到達できない

Primary actor: `OAuth2Client`

### Example: EX-TENANCY-009-01 通常経路

- Given `tenant_base_domain` が設定されている
- And テナント "acme" の endpoint_style は Subdomain である
- And テナント "beta" の endpoint_style は Path である
- When "/realms/acme/authorize" にリクエストを送る
- Then acme は Subdomain なので path prefix 経路では不在として扱われ 404 になる

### Example: EX-TENANCY-009-02 Host "beta.{tenant_base_domain}" の "/authorize" にリクエストを送る

- Given `tenant_base_domain` が設定されている
- And テナント "acme" の endpoint_style は Subdomain である
- And テナント "beta" の endpoint_style は Path である
- When "/realms/acme/authorize" にリクエストを送る
- But Host "beta.{tenant_base_domain}" の "/authorize" にリクエストを送る
- Then beta は Path なのでサブドメイン経路では不在として扱われ 404 になる

### Example: EX-TENANCY-009-03 Host "acme.{tenant_base_domain}" の "/realms/beta/authorize" にリクエストを送る

- Given `tenant_base_domain` が設定されている
- And テナント "acme" の endpoint_style は Subdomain である
- And テナント "beta" の endpoint_style は Path である
- When "/realms/acme/authorize" にリクエストを送る
- But Host "acme.{tenant_base_domain}" の "/realms/beta/authorize" にリクエストを送る
- Then acme の origin から beta へ到達することはできず 404 になる

## Rule: REQ-TENANCY-010 Discovery Metadata の `issuer` は取得元 URL と一致する

Primary actor: `OAuth2Client`

### Example: EX-TENANCY-010-01 通常経路

- Given `tenant_base_domain` が設定されている
- And テナント "default" の endpoint_style は Path、テナント "acme" の endpoint_style は Subdomain である
- When "{base}/realms/default/.well-known/openid-configuration" を取得する
- Then issuer は "{base}/realms/default" であり、取得元 URL の prefix と一致する
- When "https://acme.{tenant_base_domain}/.well-known/openid-configuration" を取得する
- Then issuer は "https://acme.{tenant_base_domain}" であり、取得元 URL の prefix と一致する
- Then どちらのレスポンスもエンドポイントURLを自分の正規ロケーション配下だけで組み立てる

## Rule: REQ-TENANCY-011 System管理者はテナントの正規ロケーションを切り替えられる

Primary actor: `SystemAdministrator`

### Example: EX-TENANCY-011-01 通常経路

- Given system_admin ロールを持つ "sysadmin" が認証済みである
- And `tenant_base_domain` が設定されている
- And テナント "acme" の endpoint_style は Path である
- When "sysadmin" が `SetTenantEndpointStyle` で acme を `Subdomain` に切り替える
- Then acme は "acme.{tenant_base_domain}" からのみ到達できるようになる
- Then "{base}/realms/acme/..." は 404 になる
- Then issuer と WebAuthn RP ID が新しい正規ロケーション由来の値に変わる

### Example: EX-TENANCY-011-02 `tenant_base_domain` が設定されていない環境で `Subdomain` を指定する

- Given system_admin ロールを持つ "sysadmin" が認証済みである
- And `tenant_base_domain` が設定されている
- And テナント "acme" の endpoint_style は Path である
- When "sysadmin" が `SetTenantEndpointStyle` で acme を `Subdomain` に切り替える
- But `tenant_base_domain` が設定されていない環境で `Subdomain` を指定する
- Then `InvalidRequestError` で拒否され、`endpoint_style` は変わらない

## Rule: REQ-TENANCY-012 System管理者はテナントのクォータ上限を調整できる

Primary actor: `SystemAdministrator`

### Example: EX-TENANCY-012-01 通常経路

- Given system_admin ロールを持つ "sysadmin" が認証済みである
- When "sysadmin" が UpdateTenantQuota を呼び出しユーザー上限を 20000 に増やす
- Then 対象テナントの quota.users が 20000 になる

## Rule: REQ-TENANCY-013 Hard Quota を超過したリソース作成は拒否される

Primary actor: `TenantAdministrator`

### Example: EX-TENANCY-013-01 通常経路

- Given 対象テナントの groups 上限が 1000、利用量が 1000 である
- When テナント内管理者が新しい Group を作成しようとする
- Then QuotaExceededError で拒否され作成されない

## Rule: REQ-TENANCY-014 通常のテナント管理者はシステムコンソールのテナント一覧にアクセスできない

Primary actor: `TenantAdministrator`

### Example: EX-TENANCY-014-01 通常経路

- Given "operator" は admin ロールのみを持ち system_admin ロールを持たない
- When "operator" が ListTenants を呼び出す
- Then AccessDeniedError で拒否される

## Rule: REQ-TENANCY-015 日本語ロケールのユーザーには日本語のパスワードリセットメールが届く

Primary actor: `EndUser`

### Example: EX-TENANCY-015-01 通常経路

- Given 利用者 "hanako" は locale 属性が "ja"、検証済みメールアドレスを持つ
- And テナントは通知テンプレートを一度も上書きしていない
- When "hanako" が RequestPasswordReset を実行する
- Then 件名と本文が組込みデフォルトの ja テンプレートで描画されたメールが届く
- Then メールはプレーンテキストと HTML の両方を含む
- Then 本文のリセットリンクはリクエストの発行元 URL から組み立てられており、開くとパスワード再設定画面に到達する

### Example: EX-TENANCY-015-02 "hanako" の locale 属性が未設定で、テナントの default_locale が "ja" である

- Given 利用者 "hanako" は locale 属性が "ja"、検証済みメールアドレスを持つ
- And テナントは通知テンプレートを一度も上書きしていない
- When "hanako" が RequestPasswordReset を実行する
- But "hanako" の locale 属性が未設定で、テナントの default_locale が "ja" である
- Then テナントデフォルトの "ja" が採用され、日本語のメールが届く

### Example: EX-TENANCY-015-03 "hanako" の locale 属性が未設定で、テナントの default_locale も未設定である

- Given 利用者 "hanako" は locale 属性が "ja"、検証済みメールアドレスを持つ
- And テナントは通知テンプレートを一度も上書きしていない
- When "hanako" が RequestPasswordReset を実行する
- But "hanako" の locale 属性が未設定で、テナントの default_locale も未設定である
- Then システムデフォルト locale が採用され、その locale のメールが届く

### Example: EX-TENANCY-015-04 "hanako" の locale 属性がカタログに同梱翻訳の無い locale である

- Given 利用者 "hanako" は locale 属性が "ja"、検証済みメールアドレスを持つ
- And テナントは通知テンプレートを一度も上書きしていない
- When "hanako" が RequestPasswordReset を実行する
- But "hanako" の locale 属性がカタログに同梱翻訳の無い locale である
- Then 未対応 locale は飛ばして次の段が採用され、空の本文は送られない

## Rule: REQ-TENANCY-016 テナントの通知テンプレート上書きは組込みデフォルトより優先される

Primary actor: `TenantAdministrator`

### Example: EX-TENANCY-016-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が ListNotificationTemplates を呼び出す
- Then 全 template_key × 全サポート locale が customized=false で一覧される
- When "operator" が PasswordReset / ja の件名と本文を上書きして UpdateNotificationTemplate を実行する
- Then NotificationTemplateUpdated が発行され、当該テンプレートは customized=true になる
- Then 以後 ja の利用者に届くパスワードリセットメールは上書きした件名と本文で送られる
- When "operator" が ResetNotificationTemplate を実行する
- Then NotificationTemplateReset が発行され、当該テンプレートは組込みデフォルトに戻る

### Example: EX-TENANCY-016-02 上書きしていない en の利用者にメールが送られる

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が ListNotificationTemplates を呼び出す
- Then 全 template_key × 全サポート locale が customized=false で一覧される
- When "operator" が PasswordReset / ja の件名と本文を上書きして UpdateNotificationTemplate を実行する
- Then NotificationTemplateUpdated が発行され、当該テンプレートは customized=true になる
- Then 上書きしていない en の利用者にメールが送られる
- Then en は組込みデフォルトのまま描画され、ja の上書きは影響しない

### Example: EX-TENANCY-016-03 上書きが存在しないテンプレートに ResetNotificationTemplate を実行する

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が ListNotificationTemplates を呼び出す
- Then 全 template_key × 全サポート locale が customized=false で一覧される
- When "operator" が PasswordReset / ja の件名と本文を上書きして UpdateNotificationTemplate を実行する
- Then NotificationTemplateUpdated が発行され、当該テンプレートは customized=true になる
- Then 以後 ja の利用者に届くパスワードリセットメールは上書きした件名と本文で送られる
- When "operator" が ResetNotificationTemplate を実行する
- But 上書きが存在しないテンプレートに ResetNotificationTemplate を実行する
- Then 冪等に成功し、組込みデフォルトのままとなる

## Rule: REQ-TENANCY-017 許可されていない差し込み変数を含むテンプレート上書きは保存時に拒否される

Primary actor: `TenantAdministrator`

### Example: EX-TENANCY-017-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PasswordReset の本文に許可集合外の変数 `{{password}}` を書いて保存を試みる
- Then InvalidRequestError で拒否され、上書きは保存されない
- Then 以後も利用者には組込みデフォルトのリセットメールが届き、リンクが欠けたメールは配られない

### Example: EX-TENANCY-017-02 "operator" が HTML 本文を空にしてテキスト本文だけを保存しようとする

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PasswordReset の本文に許可集合外の変数 `{{password}}` を書いて保存を試みる
- But "operator" が HTML 本文を空にしてテキスト本文だけを保存しようとする
- Then InvalidRequestError で拒否され、片方だけの上書きは作られない

### Example: EX-TENANCY-017-03 "operator" がカタログに無い locale を指定して保存を試みる

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PasswordReset の本文に許可集合外の変数 `{{password}}` を書いて保存を試みる
- But "operator" がカタログに無い locale を指定して保存を試みる
- Then InvalidRequestError で拒否される

### Example: EX-TENANCY-017-04 "operator" が差出人メールアドレスの上書きを試みる

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PasswordReset の本文に許可集合外の変数 `{{password}}` を書いて保存を試みる
- But "operator" が差出人メールアドレスの上書きを試みる
- Then アドレスを上書きする入力は受け付けず、上書きできるのは表示名だけである

## Rule: REQ-TENANCY-018 プレビューは実送信せずテスト送信は操作者本人にしか届かない

Primary actor: `TenantAdministrator`

### Example: EX-TENANCY-018-01 通常経路

- Given admin ロールを持つ "operator" が検証済みメールアドレスを持ち認証済みである
- When "operator" が保存前の文面で PreviewNotificationTemplate を呼び出す
- Then サンプル値を展開した件名・テキスト本文・HTML 本文が返る
- Then メールは送信されず、上書きも保存されない
- When "operator" が SendTestNotification を呼び出す
- Then 宛先は "operator" 自身のアドレスに固定され、EmailSent が発行される

### Example: EX-TENANCY-018-02 文面に利用者名などの差し込み値が含まれる

- Given admin ロールを持つ "operator" が検証済みメールアドレスを持ち認証済みである
- When "operator" が保存前の文面で PreviewNotificationTemplate を呼び出す
- Then 文面に利用者名などの差し込み値が含まれる
- Then HTML 側の差し込み値はエスケープされて描画され、タグとして解釈されない

### Example: EX-TENANCY-018-03 リクエストで別の宛先を指定しようとする

- Given admin ロールを持つ "operator" が検証済みメールアドレスを持ち認証済みである
- When "operator" が保存前の文面で PreviewNotificationTemplate を呼び出す
- Then サンプル値を展開した件名・テキスト本文・HTML 本文が返る
- Then メールは送信されず、上書きも保存されない
- When "operator" が SendTestNotification を呼び出す
- But リクエストで別の宛先を指定しようとする
- Then 宛先の指定手段は提供されず、常に操作者本人へ送られる

### Example: EX-TENANCY-018-04 操作者が検証済みメールアドレスを持たない

- Given admin ロールを持つ "operator" が検証済みメールアドレスを持ち認証済みである
- When "operator" が保存前の文面で PreviewNotificationTemplate を呼び出す
- Then サンプル値を展開した件名・テキスト本文・HTML 本文が返る
- Then メールは送信されず、上書きも保存されない
- When "operator" が SendTestNotification を呼び出す
- But 操作者が検証済みメールアドレスを持たない
- Then InvalidRequestError で拒否され、メールは送信されない

## Rule: REQ-TENANCY-019 管理者はパスワードポリシー設定を参照・更新できる

Primary actor: `TenantAdministrator`

### Example: EX-TENANCY-019-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" がパスワードの最小長を更新する
- Then 更新後の設定に新しい最小長が反映される
- Then 上書きは永続化され、プロセス再起動後の設定取得でも同じ値が返る
- When 管理者 "operator" が max_age_days=90 を保存する
- Then 以後のパスワード検証と有効期限判定にテナントの上書き値が使われる

### Example: EX-TENANCY-019-02 標準値より弱い上書き (最小長を下回る / 最大長を上回る / 履歴件数を下回る) を保存する

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" がパスワードの最小長を更新する
- But 標準値より弱い上書き (最小長を下回る / 最大長を上回る / 履歴件数を下回る) を保存する
- Then エラー "PolicyOverrideWeakerError"

### Example: EX-TENANCY-019-03 max_age_days に system ceiling の範囲外 (30 未満、または 3650 超) を保存する

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" がパスワードの最小長を更新する
- But max_age_days に system ceiling の範囲外 (30 未満、または 3650 超) を保存する
- Then エラー "PolicyOverrideWeakerError"

## Rule: REQ-TENANCY-020 管理者はテナント固有のグループ属性スキーマを定義できる

Primary actor: `TenantAdministrator`

### Example: EX-TENANCY-020-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が group custom attribute "cost_center" (type=string, required=false) を追加する
- Then 更新後のスキーマに追加した属性が含まれ "TenantGroupAttributeSchemaUpdated" が発行される

### Example: EX-TENANCY-020-02 既存 key と重複する key を追加する

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が group custom attribute "cost_center" (type=string, required=false) を追加する
- But 既存 key と重複する key を追加する
- Then 更新は InvalidGroupAttributeSchemaError で拒否される

## Rule: REQ-TENANCY-021 委譲深さの上書きは厳しい方向にのみ働く

Primary actor: `TenantAdministrator`

### Example: EX-TENANCY-021-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" が委譲深さの上限を保存する
- Then 設定取得のレスポンスは現在の上書き値と、上書きが無いときに適用されるシステム既定の双方を返す

### Example: EX-TENANCY-021-02 システム既定より小さい値を保存する

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" が委譲深さの上限を保存する
- But システム既定より小さい値を保存する
- Then 上書きが永続化され、以後のトークン交換の判定に使われる

### Example: EX-TENANCY-021-03 システム既定を超える値を保存する

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" が委譲深さの上限を保存する
- But システム既定を超える値を保存する
- Then エラー "PolicyOverrideWeakerError"

### Example: EX-TENANCY-021-04 1 未満の値を保存する

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" が委譲深さの上限を保存する
- But 1 未満の値を保存する
- Then エラー "PolicyOverrideWeakerError"

### Example: EX-TENANCY-021-05 0 を保存する

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" が委譲深さの上限を保存する
- But 0 を保存する
- Then 上書きを解除し、システム既定を継承する状態へ戻す
