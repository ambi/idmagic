# Tenancy Scenarios

### REQ-TENANCY-001: 管理者は正規ロケーションの連携情報を取得する
- ACTOR TenantAdministrator
- GIVEN admin が path または サブドメインの正規ロケーションから自身のテナントへアクセスしている
- WHEN admin が連携エンドポイント画面を開く
  - ALT admin が別テナントの realm を URL として指定しようとする → 対象指定パラメータは存在せず、解決済みテナント以外の情報は返らない
- THEN サーバーはリクエスト先テナントの正規の発行者から OAuth/OIDC、SAML、WS-Federation、SCIM、管理 API、本人用 API の URL を導出する
- THEN 画面は OAuth/OIDC、SAML、WS-Federation、API のプロトコル単位で情報をまとめ、SAML 配下ではデフォルトを含むプロファイルごとにエンティティ ID、メタデータ、SSO、SLO、署名証明書を一組で表示する
- THEN 画面は読み取り専用であり、Discovery Metadata と各プロトコルのメタデータを正本として案内し、個別値をコピーまたは証明書をダウンロードできる
- THEN 正規の発行者と同じオリジンで配信するゲートウェイは、表示した公開プロトコル URL を対応するサーバーのエンドポイントへ転送する
- THEN レスポンスにクライアントシークレット、API トークン、秘密鍵は含まれない

### REQ-TENANCY-002: 管理者はテナント固有のユーザー属性スキーマを定義できる
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が認証済みである
- WHEN "operator" が editable_by_user=true の custom_attribute を追加する
- THEN 更新後のスキーマに追加した属性が含まれる

### REQ-TENANCY-003: default テナントは起動時に自動作成され削除も無効化もできない
- ACTOR System
- WHEN IdP を起動する
- THEN テナント "default" が status=Active で存在する
  - ALT default テナントの削除を試みる → default テナントを削除する API は提供されない
  - ALT default テナントの無効化を試みる → default テナントの disable は InvalidRequestError で拒否される

### REQ-TENANCY-004: 管理者はテナントのロゴと配色をカスタマイズでき利用者のログイン画面に反映される
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が認証済みである
- WHEN "operator" が PNG ロゴをアップロードする
- THEN アップロード応答に logo_url が含まれる
- WHEN "operator" が logo_url を GET する
  - ALT 別テナントの id で同じ kind のアセット取得を試みる → アセットは存在しないものとして扱われ InvalidRequestError で拒否される
- THEN 同じ realm の検証済み PNG が返る
- THEN 管理画面のロゴプレビューにアップロードした PNG が表示される
- WHEN "operator" が primary_color / accent_color / footer_link_1={label: "ヘルプ", url: "https://help.example.test"} / footer_text を設定する
- THEN 管理画面は各設定済み色に現在値と「デフォルトに戻す」操作を表示する
- WHEN 管理者がプライマリカラーをデフォルトに戻して保存する
- THEN UpdateTenantBranding には primary_color の空文字列が送られる
- WHEN 未認証の利用者が login 画面を開く
- THEN login / 同意 / account portal に設定したロゴが表示され、login 画面にはプライマリカラーのシステムデフォルト・設定済みアクセントカラー・指定ラベルの footer リンク・フッターテキストも表示される
  - ALT realm 配下の logo_url が gateway で backend に転送されない → 画像取得は成功せず、管理者は設定の成功応答だけでは表示可能と判断しない

### REQ-TENANCY-005: 不正な branding 入力は拒否されシステムデフォルトにフォールバックする
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が認証済みである
- WHEN "operator" が branding を一度も設定していないテナントで login 画面を開く
- THEN login 画面はシステムデフォルト (IdMagic) のブランディングを表示する
- WHEN "operator" が footer_link_1.url に javascript: スキームを指定して保存する
  - ALT footer_link_1 に label だけを指定する → InvalidRequestError で拒否され保存されない
- THEN InvalidRequestError で拒否され保存されない
- WHEN "operator" が低コントラストの `#eeeeee` を primary_color に指定して保存する
- THEN 保存に成功し、取得した branding と login 画面に `#eeeeee` が反映される
- WHEN 管理者が SVG ファイルをロゴとしてアップロードする
- THEN InvalidRequestError で拒否され保存されない

### REQ-TENANCY-006: path style のテナントは realm prefix から解決される
- ACTOR OAuth2Client
- GIVEN テナント "default" の endpoint_style は Path である
- WHEN "/realms/default/authorize" にリクエストを送る
  - ALT 対象テナントが無効化されている →  tenant_id "acme" を作成して無効化する →  無効化済みテナントの "/realms/acme/authorize" にリクエストを送る →  テナントの存在を漏らさずエラー "InvalidRequestError"
  - ALT realm prefix を持たない "/authorize" にリクエストを送る → テナントは解決されず 404 tenant_not_found になる → 任意のリクエストが default テナントへ落ちることはない
- THEN 解決されたテナントは "default"
- THEN iss claim はベースURL + /realms/default

### REQ-TENANCY-007: subdomain style のテナントは Host から解決される
- ACTOR EndUser
- GIVEN `tenant_base_domain` が設定されている
- GIVEN テナント "acme" の endpoint_style は Subdomain である
- WHEN Host "acme.{tenant_base_domain}" の "/authorize" にリクエストを送る
- THEN 解決されたテナントは "acme" で、その branding のログイン画面が表示される
- THEN セッション cookie は __Host- prefix と Path=/ を持ち Domain 属性を持たない
- THEN WebAuthn RP ID は "acme.{tenant_base_domain}" である

### REQ-TENANCY-008: 未知のサブドメインは default テナントに解決されない
- ACTOR OAuth2Client
- GIVEN `tenant_base_domain` が設定されている
- GIVEN realm "unknown" のテナントは存在しない
- WHEN Host "unknown.{tenant_base_domain}" の "/authorize" にリクエストを送る
- THEN 404 tenant_not_found になり、default テナントにも他のどのテナントにも到達しない

### REQ-TENANCY-009: テナントは自分の正規ロケーション以外からは到達できない
- ACTOR OAuth2Client
- GIVEN `tenant_base_domain` が設定されている
- GIVEN テナント "acme" の endpoint_style は Subdomain である
- GIVEN テナント "beta" の endpoint_style は Path である
- WHEN "/realms/acme/authorize" にリクエストを送る
  - ALT Host "beta.{tenant_base_domain}" の "/authorize" にリクエストを送る → beta は Path なのでサブドメイン経路では不在として扱われ 404 になる
  - ALT Host "acme.{tenant_base_domain}" の "/realms/beta/authorize" にリクエストを送る → acme の origin から beta へ到達することはできず 404 になる
- THEN acme は Subdomain なので path prefix 経路では不在として扱われ 404 になる

### REQ-TENANCY-010: Discovery Metadata の `issuer` は取得元 URL と一致する
- ACTOR OAuth2Client
- GIVEN `tenant_base_domain` が設定されている
- GIVEN テナント "default" の endpoint_style は Path、テナント "acme" の endpoint_style は Subdomain である
- WHEN "{base}/realms/default/.well-known/openid-configuration" を取得する
- THEN issuer は "{base}/realms/default" であり、取得元 URL の prefix と一致する
- WHEN "https://acme.{tenant_base_domain}/.well-known/openid-configuration" を取得する
- THEN issuer は "https://acme.{tenant_base_domain}" であり、取得元 URL の prefix と一致する
- THEN どちらの応答もエンドポイントURLを自分の正規ロケーション配下だけで組み立てる

### REQ-TENANCY-011: System管理者はテナントの正規ロケーションを切り替えられる
- ACTOR SystemAdministrator
- GIVEN system_admin ロールを持つ "sysadmin" が認証済みである
- GIVEN `tenant_base_domain` が設定されている
- GIVEN テナント "acme" の endpoint_style は Path である
- WHEN "sysadmin" が `SetTenantEndpointStyle` で acme を `Subdomain` に切り替える
  - ALT `tenant_base_domain` が設定されていない環境で `Subdomain` を指定する → `InvalidRequestError` で拒否され、`endpoint_style` は変わらない
- THEN acme は "acme.{tenant_base_domain}" からのみ到達できるようになる
- THEN "{base}/realms/acme/..." は 404 になる
- THEN issuer と WebAuthn RP ID が新しい正規ロケーション由来の値に変わる

### REQ-TENANCY-012: System管理者はテナントのクォータ上限を調整できる
- ACTOR SystemAdministrator
- GIVEN system_admin ロールを持つ "sysadmin" が認証済みである
- WHEN "sysadmin" が UpdateTenantQuota を呼び出しユーザー上限を 20000 に増やす
- THEN 対象テナントの quota.users が 20000 になる

### REQ-TENANCY-013: Hard Quota を超過したリソース作成は拒否される
- ACTOR TenantAdministrator
- GIVEN 対象テナントの groups 上限が 1000、利用量が 1000 である
- WHEN テナント内管理者が新しい Group を作成しようとする
- THEN QuotaExceededError で拒否され作成されない

### REQ-TENANCY-014: 通常のテナント管理者はシステムコンソールのテナント一覧にアクセスできない
- ACTOR TenantAdministrator
- GIVEN "operator" は admin ロールのみを持ち system_admin ロールを持たない
- WHEN "operator" が ListTenants を呼び出す
- THEN AccessDeniedError で拒否される

### REQ-TENANCY-015: 日本語ロケールのユーザーには日本語のパスワードリセットメールが届く
- ACTOR EndUser
- GIVEN 利用者 "hanako" は locale 属性が "ja"、検証済みメールアドレスを持つ
- GIVEN テナントは通知テンプレートを一度も上書きしていない
- WHEN "hanako" が RequestPasswordReset を実行する
  - ALT "hanako" の locale 属性が未設定で、テナントの default_locale が "ja" である → テナントデフォルトの "ja" が採用され、日本語のメールが届く
  - ALT "hanako" の locale 属性が未設定で、テナントの default_locale も未設定である → システムデフォルト locale が採用され、その locale のメールが届く
  - ALT "hanako" の locale 属性がカタログに同梱翻訳の無い locale である → 未対応 locale は飛ばして次の段が採用され、空の本文は送られない
- THEN 件名と本文が組込みデフォルトの ja テンプレートで描画されたメールが届く
- THEN メールはプレーンテキストと HTML の両方を含む
- THEN 本文のリセットリンクはリクエストの発行元 URL から組み立てられており、開くとパスワード再設定画面に到達する

### REQ-TENANCY-016: テナントの通知テンプレート上書きは組込みデフォルトより優先される
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が認証済みである
- WHEN "operator" が ListNotificationTemplates を呼び出す
- THEN 全 template_key × 全サポート locale が customized=false で一覧される
- WHEN "operator" が PasswordReset / ja の件名と本文を上書きして UpdateNotificationTemplate を実行する
- THEN NotificationTemplateUpdated が発行され、当該テンプレートは customized=true になる
- THEN 以後 ja の利用者に届くパスワードリセットメールは上書きした件名と本文で送られる
  - ALT 上書きしていない en の利用者にメールが送られる → en は組込みデフォルトのまま描画され、ja の上書きは影響しない
- WHEN "operator" が ResetNotificationTemplate を実行する
  - ALT 上書きが存在しないテンプレートに ResetNotificationTemplate を実行する → 冪等に成功し、組込みデフォルトのままとなる
- THEN NotificationTemplateReset が発行され、当該テンプレートは組込みデフォルトに戻る

### REQ-TENANCY-017: 許可されていない差し込み変数を含むテンプレート上書きは保存時に拒否される
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が認証済みである
- WHEN "operator" が PasswordReset の本文に許可集合外の変数 `{{password}}` を書いて保存を試みる
  - ALT "operator" が HTML 本文を空にしてテキスト本文だけを保存しようとする → InvalidRequestError で拒否され、片方だけの上書きは作られない
  - ALT "operator" がカタログに無い locale を指定して保存を試みる → InvalidRequestError で拒否される
  - ALT "operator" が差出人メールアドレスの上書きを試みる → アドレスを上書きする入力は受け付けず、上書きできるのは表示名だけである
- THEN InvalidRequestError で拒否され、上書きは保存されない
- THEN 以後も利用者には組込みデフォルトのリセットメールが届き、リンクが欠けたメールは配られない

### REQ-TENANCY-018: プレビューは実送信せずテスト送信は操作者本人にしか届かない
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が検証済みメールアドレスを持ち認証済みである
- WHEN "operator" が保存前の文面で PreviewNotificationTemplate を呼び出す
- THEN サンプル値を展開した件名・テキスト本文・HTML 本文が返る
  - ALT 文面に利用者名などの差し込み値が含まれる → HTML 側の差し込み値はエスケープされて描画され、タグとして解釈されない
- THEN メールは送信されず、上書きも保存されない
- WHEN "operator" が SendTestNotification を呼び出す
  - ALT リクエストで別の宛先を指定しようとする → 宛先の指定手段は提供されず、常に操作者本人へ送られる
  - ALT 操作者が検証済みメールアドレスを持たない → InvalidRequestError で拒否され、メールは送信されない
- THEN 宛先は "operator" 自身のアドレスに固定され、EmailSent が発行される

### REQ-TENANCY-019: 管理者はパスワードポリシー設定を参照・更新できる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- WHEN 管理者 "operator" がパスワードの最小長を更新する
  - ALT 標準値より弱い上書き (最小長を下回る / 最大長を上回る / 履歴件数を下回る) を保存する → エラー "PolicyOverrideWeakerError"
  - ALT max_age_days に system ceiling の範囲外 (30 未満、または 3650 超) を保存する → エラー "PolicyOverrideWeakerError"
- THEN 更新後の設定に新しい最小長が反映される
- THEN 上書きは永続化され、プロセス再起動後の設定取得でも同じ値が返る
- WHEN 管理者 "operator" が max_age_days=90 を保存する
- THEN 以後のパスワード検証と有効期限判定にテナントの上書き値が使われる

### REQ-TENANCY-020: 管理者はテナント固有のグループ属性スキーマを定義できる
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が認証済みである
- WHEN "operator" が group custom attribute "cost_center" (type=string, required=false) を追加する
  - ALT 既存 key と重複する key を追加する → 更新は InvalidGroupAttributeSchemaError で拒否される
- THEN 更新後のスキーマに追加した属性が含まれ "TenantGroupAttributeSchemaUpdated" が発行される

### REQ-TENANCY-021: 委譲深さの上書きは厳しい方向にのみ働く
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- WHEN 管理者 "operator" が委譲深さの上限を保存する
  - ALT システム既定より小さい値を保存する → 上書きが永続化され、以後のトークン交換の判定に使われる
  - ALT システム既定を超える値を保存する → エラー "PolicyOverrideWeakerError"
  - ALT 1 未満の値を保存する → エラー "PolicyOverrideWeakerError"
  - ALT 0 を保存する → 上書きを解除し、システム既定を継承する状態へ戻す
- THEN 設定取得の応答は現在の上書き値と、上書きが無いときに適用されるシステム既定の双方を返す
