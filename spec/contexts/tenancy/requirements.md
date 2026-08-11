# Tenancy Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-TENANCY-001: 管理者は正規ロケーションの連携情報を取得する
- Actor: TenantAdministrator
- Given: admin が path または subdomain の正規ロケーションから自身のテナントへアクセスしている
- Then: admin が連携エンドポイント画面を開く
- Then: server は request tenant の canonical issuer から OAuth/OIDC、SAML、WS-Federation、SCIM、管理API、本人APIの URL を導出する
- Then: 画面はOAuth/OIDC、SAML、WS-Federation、APIのprotocol単位で情報をまとめ、SAML配下ではdefaultを含むprofileごとにentityID、metadata、SSO、SLO、署名証明書を一組で表示する
- Then: 画面はread-onlyでdiscoveryとmetadataを正本として案内し、個別値をコピーまたは証明書をダウンロードできる
- Then: canonical issuerと同一originで配信するgatewayは、表示した公開protocol URLを対応するserverエンドポイントへ転送する
- Then: 返却値に client secret、API token、秘密鍵は含まれない
- Alternative (admin が別テナントの realm を URL として指定しようとする): 対象指定パラメータは存在せず、解決済みテナント以外の情報は返らない

### REQ-TENANCY-002: 管理者はテナント固有のユーザー属性スキーマを定義できる
- Actor: TenantAdministrator
- Given: admin ロールを持つ "operator" が認証済みである
- Then: "operator" が editable_by_user=true の custom_attribute を追加する
- Then: 更新後のスキーマに追加した属性が含まれる

### REQ-TENANCY-003: default テナントは起動時に自動作成され削除も無効化もできない
- Actor: System
- Then: IdP を起動する
- Then: テナント "default" が status=Active で存在する
- Alternative (default テナントの削除を試みる): default テナントを削除する API は提供されない
- Alternative (default テナントの無効化を試みる): default テナントの disable は InvalidRequestError で拒否される

### REQ-TENANCY-004: 管理者はテナントのロゴと配色をカスタマイズでき利用者のログイン画面に反映される
- Actor: TenantAdministrator
- Given: admin ロールを持つ "operator" が認証済みである
- Then: "operator" が PNG ロゴをアップロードする
- Then: アップロード応答の logo_url を GET すると、同じ realm の検証済み PNG が返る
- Then: 管理画面のロゴプレビューにアップロードした PNG が表示される
- Then: "operator" が primary_color / accent_color / footer_link_1={label: "ヘルプ", url: "https://help.example.test"} / footer_text を設定する
- Then: 管理画面は各設定済み色に現在値と「既定に戻す」操作を表示する
- Then: 管理者がプライマリカラーを既定に戻して保存すると、UpdateTenantBranding には primary_color の空文字列が送られる
- Then: 未認証の利用者が login 画面を開く
- Then: login / consent / account portal に設定したロゴが表示され、login 画面にはプライマリカラーのシステム既定・設定済みアクセントカラー・指定ラベルの footer リンク・フッターテキストも表示される
- Alternative (別テナントの id で同じ kind のアセット取得を試みる): アセットは存在しないものとして扱われ InvalidRequestError で拒否される
- Alternative (realm 配下の logo_url が gateway で backend に転送されない): 画像取得は成功せず、管理者は設定の成功応答だけでは表示可能と判断しない

### REQ-TENANCY-005: 不正な branding 入力は拒否されシステム既定にフォールバックする
- Actor: TenantAdministrator
- Given: admin ロールを持つ "operator" が認証済みである
- Then: "operator" が branding を一度も設定していないテナントで login 画面を開く
- Then: login 画面はシステム既定 (IdMagic) のブランディングを表示する
- Alternative ("operator" が footer_link_1.url に javascript: スキームを指定して保存を試みる): InvalidRequestError で拒否され保存されない
- Alternative ("operator" が footer_link_1 に label だけを指定して保存を試みる): InvalidRequestError で拒否され保存されない
- Alternative ("operator" が低コントラストの `#eeeeee` を primary_color に指定して保存を試みる): 保存に成功し、取得した branding と login 画面に `#eeeeee` が反映される
- Alternative (管理者が SVG ファイルをロゴとしてアップロードしようとする): InvalidRequestError で拒否され保存されない

### REQ-TENANCY-006: path style のテナントは realm prefix から解決される
- Actor: OAuth2Client
- Given: テナント "default" の endpoint_style は Path である
- Then: "/realms/default/authorize" にリクエストを送る
- Then: 解決されたテナントは "default"
- Then: iss claim はベースURL + /realms/default
- Alternative (対象テナントが無効化されている):  tenant_id "acme" を作成して無効化する →  無効化済みテナントの "/realms/acme/authorize" にリクエストを送る →  テナントの存在を漏らさずエラー "InvalidRequestError"
- Alternative (realm prefix を持たない "/authorize" にリクエストを送る): テナントは解決されず 404 tenant_not_found になる → 任意のリクエストが default テナントへ落ちることはない

### REQ-TENANCY-007: subdomain style のテナントは Host から解決される
- Actor: EndUser
- Given: tenant_base_domain が設定されている
- Given: テナント "acme" の endpoint_style は Subdomain である
- Then: Host "acme.{tenant_base_domain}" の "/authorize" にリクエストを送る
- Then: 解決されたテナントは "acme" で、その branding のログイン画面が表示される
- Then: session cookie は __Host- prefix と Path=/ を持ち Domain 属性を持たない
- Then: WebAuthn RP ID は "acme.{tenant_base_domain}" である

### REQ-TENANCY-008: 未知のサブドメインは default テナントに解決されない
- Actor: OAuth2Client
- Given: tenant_base_domain が設定されている
- Given: realm "unknown" のテナントは存在しない
- Then: Host "unknown.{tenant_base_domain}" の "/authorize" にリクエストを送る
- Then: 404 tenant_not_found になり、default テナントにも他のどのテナントにも到達しない

### REQ-TENANCY-009: テナントは自分の正規ロケーション以外からは到達できない
- Actor: OAuth2Client
- Given: tenant_base_domain が設定されている
- Given: テナント "acme" の endpoint_style は Subdomain である
- Given: テナント "beta" の endpoint_style は Path である
- Then: "/realms/acme/authorize" にリクエストを送る
- Then: acme は Subdomain なので path prefix 経路では不在として扱われ 404 になる
- Alternative (Host "beta.{tenant_base_domain}" の "/authorize" にリクエストを送る): beta は Path なのでサブドメイン経路では不在として扱われ 404 になる
- Alternative (Host "acme.{tenant_base_domain}" の "/realms/beta/authorize" にリクエストを送る): acme の origin から beta へ到達することはできず 404 になる

### REQ-TENANCY-010: discovery の issuer は取得元 URL と一致する
- Actor: OAuth2Client
- Given: tenant_base_domain が設定されている
- Given: テナント "default" の endpoint_style は Path、テナント "acme" の endpoint_style は Subdomain である
- Then: "{base}/realms/default/.well-known/openid-configuration" を取得する
- Then: issuer は "{base}/realms/default" であり、取得元 URL の prefix と一致する
- Then: "https://acme.{tenant_base_domain}/.well-known/openid-configuration" を取得する
- Then: issuer は "https://acme.{tenant_base_domain}" であり、取得元 URL の prefix と一致する
- Then: どちらの応答もエンドポイントURLを自分の正規ロケーション配下だけで組み立てる

### REQ-TENANCY-011: System管理者はテナントの正規ロケーションを切り替えられる
- Actor: SystemAdministrator
- Given: system_admin ロールを持つ "sysadmin" が認証済みである
- Given: tenant_base_domain が設定されている
- Given: テナント "acme" の endpoint_style は Path である
- Then: "sysadmin" が SetTenantEndpointStyle で acme を Subdomain に切り替える
- Then: acme は "acme.{tenant_base_domain}" からのみ到達できるようになる
- Then: "{base}/realms/acme/..." は 404 になる
- Then: issuer と WebAuthn RP ID が新しい正規ロケーション由来の値に変わる
- Alternative (tenant_base_domain が設定されていない配備で Subdomain を指定する): InvalidRequestError で拒否され endpoint_style は変わらない

### REQ-TENANCY-012: System管理者はテナントのクォータ上限を調整できる
- Actor: SystemAdministrator
- Given: system_admin ロールを持つ "sysadmin" が認証済みである
- Then: "sysadmin" が UpdateTenantQuota を呼び出し users 上限を 20000 に増やす
- Then: 対象テナントの quota.users が 20000 になる

### REQ-TENANCY-013: Hard Quota を超過したリソース作成は拒否される
- Actor: TenantAdministrator
- Given: 対象テナントの groups 上限が 1000、利用量が 1000 である
- Then: テナント内管理者が新しい Group を作成しようとする
- Then: QuotaExceededError で拒否され作成されない

### REQ-TENANCY-014: 通常のテナント管理者はシステムコンソールのテナント一覧にアクセスできない
- Actor: TenantAdministrator
- Given: "operator" は admin ロールのみを持ち system_admin ロールを持たない
- Then: "operator" が ListTenants を呼び出す
- Then: AccessDeniedError で拒否される

### REQ-TENANCY-015: 日本語ロケールのユーザーには日本語のパスワードリセットメールが届く
- Actor: EndUser
- Given: 利用者 "hanako" は locale 属性が "ja"、検証済みメールアドレスを持つ
- Given: テナントは通知テンプレートを一度も上書きしていない
- Then: "hanako" が RequestPasswordReset を実行する
- Then: 件名と本文が組込み既定の ja テンプレートで描画されたメールが届く
- Then: メールはプレーンテキストと HTML の両方を含む
- Then: 本文のリセットリンクはリクエストの発行元 URL から組み立てられており、開くとパスワード再設定画面に到達する
- Alternative ("hanako" の locale 属性が未設定で、テナントの default_locale が "ja" である): テナント既定の "ja" が採用され、日本語のメールが届く
- Alternative ("hanako" の locale 属性が未設定で、テナントの default_locale も未設定である): システム既定 locale が採用され、その locale のメールが届く
- Alternative ("hanako" の locale 属性がカタログに同梱翻訳の無い locale である): 未対応 locale は飛ばして次の段が採用され、空の本文は送られない

### REQ-TENANCY-016: テナントの通知テンプレート上書きは組込み既定より優先される
- Actor: TenantAdministrator
- Given: admin ロールを持つ "operator" が認証済みである
- Then: "operator" が ListNotificationTemplates を呼び出す
- Then: 全 template_key × 全サポート locale が customized=false で一覧される
- Then: "operator" が PasswordReset / ja の件名と本文を上書きして UpdateNotificationTemplate を実行する
- Then: NotificationTemplateUpdated が発行され、当該テンプレートは customized=true になる
- Then: 以後 ja の利用者に届くパスワードリセットメールは上書きした件名と本文で送られる
- Then: "operator" が ResetNotificationTemplate を実行する
- Then: NotificationTemplateReset が発行され、当該テンプレートは組込み既定に戻る
- Alternative (上書きしていない en の利用者にメールが送られる): en は組込み既定のまま描画され、ja の上書きは影響しない
- Alternative (上書きが存在しないテンプレートに ResetNotificationTemplate を実行する): 冪等に成功し、組込み既定のままとなる

### REQ-TENANCY-017: 許可されていない差し込み変数を含むテンプレート上書きは保存時に拒否される
- Actor: TenantAdministrator
- Given: admin ロールを持つ "operator" が認証済みである
- Then: "operator" が PasswordReset の本文に許可集合外の変数 `{{password}}` を書いて保存を試みる
- Then: InvalidRequestError で拒否され、上書きは保存されない
- Then: 以後も利用者には組込み既定のリセットメールが届き、リンクが欠けたメールは配られない
- Alternative ("operator" が HTML 本文を空にしてテキスト本文だけを保存しようとする): InvalidRequestError で拒否され、片方だけの上書きは作られない
- Alternative ("operator" がカタログに無い locale を指定して保存を試みる): InvalidRequestError で拒否される
- Alternative ("operator" が差出人メールアドレスの上書きを試みる): アドレスを上書きする入力は受け付けず、上書きできるのは表示名だけである

### REQ-TENANCY-018: プレビューは実送信せずテスト送信は操作者本人にしか届かない
- Actor: TenantAdministrator
- Given: admin ロールを持つ "operator" が検証済みメールアドレスを持ち認証済みである
- Then: "operator" が保存前の文面で PreviewNotificationTemplate を呼び出す
- Then: サンプル値を展開した件名・テキスト本文・HTML 本文が返る
- Then: メールは送信されず、上書きも保存されない
- Then: "operator" が SendTestNotification を呼び出す
- Then: 宛先は "operator" 自身のアドレスに固定され、EmailSent が発行される
- Alternative (文面に利用者名などの差し込み値が含まれる): HTML 側の差し込み値はエスケープされて描画され、タグとして解釈されない
- Alternative (リクエストで別の宛先を指定しようとする): 宛先の指定手段は提供されず、常に操作者本人へ送られる
- Alternative (操作者が検証済みメールアドレスを持たない): InvalidRequestError で拒否され、メールは送信されない

### REQ-TENANCY-019: 管理者はパスワードポリシー設定を参照・更新できる
- Actor: TenantAdministrator
- Given: roles=["admin"] のユーザー "operator" が管理画面の設定を開いている
- Then: 管理者 "operator" がパスワードの最小長を更新する
- Then: 更新後の設定に新しい最小長が反映される

### REQ-TENANCY-020: ResolveTenant
HTTP リクエストの Host と path から所属テナントを解決する内部インターフェース。

**不変条件: 1 テナント = 1 正規ロケーション = 1 issuer。** テナントは自分の
endpoint_style が指す正規ロケーションからのみ到達でき、他方の経路では不在として扱う。
同一テナントが 2 つの origin から到達できると issuer が多義になり、discovery 文書の
`issuer` が取得元 URL と一致しなくなる (OpenID Connect Discovery 1.0 §4.3 /
RFC 8414 §3.3 違反)。

解決順序:
1. tenant_base_domain が設定され Host が `{label}.{tenant_base_domain}` に一致するなら
   label を realm として写像する。見つかったテナントの endpoint_style が Subdomain で
   なければ不在として扱う。
2. path が `/realms/{realm}/...` に一致するなら realm を写像する。見つかったテナントの
   endpoint_style が Path でなければ不在として扱う。
3. どちらにも一致しないリクエストは不在として扱う。任意の Host や prefix 無し path を
   default テナントへ落とすことはしない (fail-closed)。テナント境界の破りを防ぐため、
   既定は deny とする。

issuer / URL prefix / cookie scope / WebAuthn RP ID は解決した正規ロケーションから
組み立てる。Path なら issuer は `{base}/realms/{realm}`、Subdomain なら
`{scheme}://{realm}.{tenant_base_domain}`。
不在テナントは 404 tenant_not_found、disabled テナントは OAuth/OIDC の protocol route
では 400 invalid_request とし、いずれも存在や状態の詳細を漏らさない。

### REQ-TENANCY-021: ListTenants
system_admin がテナント一覧を取得する。テナント横断操作のため system_admin
ロールでゲートし、システムコンソール (/system) から呼ぶ。default テナントに解決する
bare path と control-plane path (/realms/default 配下、cookie session 整合) の両方から
同じ handler に入る。

### REQ-TENANCY-022: GetTenant
system_admin が単一テナントを取得する。

### REQ-TENANCY-023: CreateTenant
system_admin が新規テナントを作成する。

### REQ-TENANCY-024: UpdateTenant
system_admin がテナントの表示名を更新する。

### REQ-TENANCY-025: DisableTenant
system_admin が default 以外のテナントを無効化する。/authorize / /token / /login 等は一律 invalid_request になる。

### REQ-TENANCY-026: EnableTenant
system_admin が無効化済みテナントを再有効化する。

### REQ-TENANCY-027: SetTenantEndpointStyle
system_admin がテナントの正規ロケーションの形を切り替える。
通常の属性更新 (UpdateTenant) から分離するのは、これがテナントの identity を
作り替える破壊的操作だからである: issuer が変わるため既発行 token の `iss` 検証と
全 RP の設定が壊れ、WebAuthn RP ID が変わるため既存 passkey がすべて無効になり、
cookie scope が変わるため進行中のセッションが切れる。
Subdomain は tenant_base_domain が設定された配備でのみ選択でき、未設定なら
InvalidRequestError とする。realm は不変なので、切り替え後のホスト名は既存 realm から
一意に決まる。

### REQ-TENANCY-028: UpdateTenantQuota
system_admin が特定テナントの Quota を更新する。

### REQ-TENANCY-029: GetAdminSettings
テナント内 admin が自身のテナントの設定 (表示名 / 既定 locale / パスワードポリシー上書き) を取得する。
対象テナントは context.tenant_id (解決済みテナント) に固定するため cross-tenant 読み出しは構造的に発生しない。
`/api/admin/tenants` (system_admin 専用) とは別経路で、admin にも開放される。
レスポンスには Authentication context の PasswordPolicy の現行値 (password_policy_defaults) と、
カタログが同梱翻訳を持つ locale 一覧 (supported_locales) を含め、UI が「上書きしないときに
実際に適用される値」と既定 locale の選択肢を具体的に示せるようにする。

### REQ-TENANCY-030: GetAdminIntegrationEndpoints
テナント内 admin が、外部の OIDC RP / SAML SP / WS-Federation RP / SCIM client / management client に投入する IdMagic の正規連携情報を一括取得する。全 URL は解決済み request tenant の canonical issuer から server が導出し、frontend は realm や origin を連結しない。秘密情報は含めず Cache-Control: no-store で返す。

### REQ-TENANCY-031: UpdateAdminSettings
テナント内 admin が自身のテナントの設定を更新する。display_name、default_locale、
password_policy_override のいずれかを任意フィールドとして受け取り、
context.tenant_id に固定した上で UpdateTenant のロジックを実行する。
password_policy_override が Authentication context の PasswordPolicy の標準値より弱い場合
(min_length 下回り / max_length 上回り / history_depth 下回り) は use case
側で拒否する。default_locale はカタログが同梱翻訳を持つ locale のみ受け付け、
空文字列でシステム既定へ戻す。

### REQ-TENANCY-032: GetTenantUserAttributeSchema
テナント内 admin が自身のテナントの custom 属性定義を取得する。
対象テナントは context.tenant_id に固定し cross-tenant 読み出しを構造的に排除する。
レスポンスには tenant 固有の custom 定義に加えて組み込みカタログ (builtin) も
含め、UI が全属性を一覧しつつ key 衝突を避けられるようにする。

### REQ-TENANCY-033: UpdateTenantUserAttributeSchema
テナント内 admin が custom 属性定義を全置換する。context.tenant_id に
固定した上で、各 UserAttributeDef を検証し、組み込み属性との key 衝突・重複 key を
拒否する。DynamicGroupRule が参照中の key は削除または型変更できない。値そのものは保持せず定義のみを扱う。

### REQ-TENANCY-034: GetTenantBranding
解決済みテナントの hosted UI ブランディングを取得する公開 interface。認証を要求しない
(login / consent / device 画面が未認証のうちに読む)。branding 未設定・値不正・アセット欠損の
いずれでもシステム既定にフォールバックし、常に成功する。対象テナントは realm 解決 (ResolveTenant)
で既に確定しているため入力を取らない。
- Postcondition: output.branding != null

### REQ-TENANCY-035: UpdateTenantBranding
テナント内 admin が自身のテナントの branding (製品名 / 色 / 順序固定の footer リンク /
フッターテキスト) を更新する。画像アセットは別 interface で扱う。色は `#rrggbb` 形式のみを
検証し、コントラスト比を理由に拒否しない。各 footer link は label と https URL の組で、どちらか
一方だけの指定は拒否する。

### REQ-TENANCY-036: UploadTenantBrandingAsset
管理者が branding のロゴまたは favicon 画像を multipart でアップロードする。受理形式は PNG / JPEG / WebP / GIF、最大 256KiB とし、magic byte で検証する。
- Precondition: context.upload.content_type in ["image/png", "image/jpeg", "image/webp", "image/gif"]
- Precondition: context.upload.magic_byte_matches_content_type
- Precondition: size(input.file) <= 262144

### REQ-TENANCY-037: DeleteTenantBrandingAsset
管理者が branding の保存済みロゴまたは favicon を削除する。削除後は該当アセットの URL を返さない。

### REQ-TENANCY-038: ListNotificationTemplates
テナント内 admin が通知テンプレートの一覧を取得する。カタログの全 template_key × 全サポート locale を、上書きの有無 (customized) と現在有効な件名つきで返す。対象テナントは context.tenant_id に固定するため cross-tenant 読み出しは構造的に発生しない。

### REQ-TENANCY-039: GetNotificationTemplate
テナント内 admin がテンプレート 1 件を編集用に取得する。現在有効な文面、組込み既定の文面、および使用できる差し込み変数の許可集合を同時に返す。カタログに無い template_key / locale は InvalidRequestError で拒否する。

### REQ-TENANCY-040: UpdateNotificationTemplate
テナント内 admin がテンプレート上書きを保存する。件名 / テキスト本文 / HTML 本文を必須の組として全置換し、テキストだけ・HTML だけの上書きは作れない。本文と件名に含まれる差し込み変数が template_key の許可集合に収まらない場合は、実行時に空文字列へ潰れる前に保存時点で拒否する (fail-closed)。差出人メールアドレスはサーバ設定であり上書きできず、表示名だけを上書きできる。
- Precondition: placeholdersWithin(input.request.subject, allowedPlaceholders(input.template_key))
- Precondition: placeholdersWithin(input.request.body_text, allowedPlaceholders(input.template_key))
- Precondition: placeholdersWithin(input.request.body_html, allowedPlaceholders(input.template_key))
- Postcondition: output.template.customized

### REQ-TENANCY-041: ResetNotificationTemplate
テナント内 admin がテンプレート上書きを削除し組込み既定へ戻す。上書きが無いテンプレートに対しても成功する冪等操作。版管理は持たないため、戻り先は直前の上書きではなく常に組込み既定である。
- Postcondition: !output.template.customized

### REQ-TENANCY-042: PreviewNotificationTemplate
テナント内 admin が保存前の文面をサンプル値で描画する。省略したフィールドは現在有効な文面を使う。HTML 側の差し込み値は必ずエスケープして展開し、リンク URL はサンプルの発行元 URL から組み立てる。メールは送信せず、上書きも保存しない。

### REQ-TENANCY-043: SendTestNotification
テナント内 admin が現在有効な文面のテストメールを受け取る。宛先は操作した管理者本人の検証済みメールアドレスに固定し、リクエストでは指定できない。任意宛先を許すと管理者権限がメール送信の踏み台になるため、この固定は仕様である。操作者に検証済みアドレスが無い場合は InvalidRequestError で拒否する。
- Postcondition: output.result.to == context.actor.email

### REQ-TENANCY-044: GetTenantBrandingAsset
保存済み branding アセット (ロゴ / favicon) を配信する公開 interface。アップロード後に返す URL は、
解決済み realm prefix を含むこの interface へ同一 origin で到達可能であり、管理画面プレビューと
hosted UI はその URL を画像として取得・描画できる。Content-Type は検証済み形式に固定し、
X-Content-Type-Options: nosniff を付ける。別テナントまたは削除済み object は未存在として扱う。
- Postcondition: response.headers["Content-Type"] in ["image/png", "image/jpeg", "image/webp", "image/gif"]
- Postcondition: response.headers["X-Content-Type-Options"] == "nosniff"

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Tenant | 独立した認可境界。Client / User / Consent / 鍵 / ポリシーがこの境界に閉じる。URL 上は Realm という別名で表現される。 | tenant, テナント, Realm, realm |
| DefaultTenant | 起動時に自動作成される `realm == "default"` のテナント。id は固定 UUID の代理キー。単一テナント運用時の互換と、未 prefix HTTP リクエストの解決先を兼ねる。 | default tenant, デフォルトテナント |
| TenantDisablement | Tenant.disabled_at を設定してテナント単位で `/authorize` / `/token` / `/login` 等を停止する復活可能な操作。テナント物理削除とは独立。 | disable tenant, テナント無効化 |
| EntraFederation | Microsoft Entra ID の検証済みドメインを WS-Federation / WS-Trust の federated IdP として接続する profile。 | Microsoft365Federation, AzureADFederation, M365Federation |
| Disabled | 復活可能な無効化状態。Tenant と (慣例的に) User の disabled_at 経路で共有される。 | disabled |
| Disable | 対象を Disabled に遷移させる。Tenant では `/api/admin/tenants/{id}/disable` から発火。 | disable |
| Enable | Disabled の対象を Active に戻す。Tenant では `/api/admin/tenants/{id}/enable` から発火。 | enable |
| System | IdP プロセス自身。起動時に default テナントを自動作成する。 |  |
| OAuth2Client | OIDC / OAuth2 プロトコルエンドポイントを呼び出す外部クライアントアプリケーション。 |  |
| EndUser | テナントに所属する人間の利用者。通知メールの受信者であり、その locale 属性が通知の言語解決の第 1 段になる。IdManagement が所有する User の published language stub。 | end user, 利用者 |
| HardQuota | 超過するとリソース作成が同期的にエラーとなる厳格な上限。 |  |
| SoftQuota | 超過しても作成は成功するが、警告が通知される遅延評価の上限。 |  |
| NotificationTemplate | 利用者へ送る通知メール 1 通の文面定義。template_key と locale の組で一意に定まり、件名 / プレーンテキスト本文 / HTML 本文 / 差出人表示名を持つ。組込み既定 (システムが同梱する ja / en の文面) とテナント上書きの 2 段で解決する。 | 通知テンプレート, notification template, email template |
| NotificationTemplateKey | 通知の用途を表す固定識別子。カタログに存在する key だけが送信・上書きの対象になり、テナントは key 自体を追加できない。 | template key, テンプレートキー |
| NotificationPlaceholder | テンプレート本文に `{{name}}` の形で書ける差し込み変数。template_key ごとに許可集合が決まっており、許可外の変数を含む上書きは保存時に拒否される。 | placeholder, 差し込み変数 |
| NotificationLocaleResolution | 通知 1 通に使う locale を決める手順。受信者 User の locale 属性 → テナントの default_locale → システム既定 locale の順に、カタログが対応する最初の locale を採用する。 | locale 解決順序, locale resolution |
| BuiltinNotificationTemplate | システムが同梱する組込み既定テンプレート。テナント上書きが無い、または上書きが削除された場合に使われる。テナントは編集できず「既定に戻す」ことでこの文面へ復帰する。 | 組込み既定テンプレート, builtin template |

## State machines

### TenantLifecycle

テナント のライフサイクル。Active で通常稼働、Disable で全プロトコルルートを停止、Enable で復帰。物理削除は本フェーズ対象外。

Initial: `Active`  
Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | TenantDisabled | "" | Disabled |  |
| Disabled | TenantEnabled | "" | Active |  |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
