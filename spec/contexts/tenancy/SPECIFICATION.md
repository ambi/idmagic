---
context: tenancy
updated_at: 2026-08-11
---

# Tenancy Specification

## Overview

テナント (Realm) 集約、ライフサイクル、HTTP リクエストからのテナント解決、テナント管理 API を所有する。テナントは idmagic のあらゆる集約に共通するBounded Context。

パスの接頭辞によるルーティング、不変 ID と変更可能な短縮名へのキーの分離、テナントごとの外観、リソース上限、管理者操作を通常の利用者向け通信から分離する認可境界も所有する。`domain` は Aggregate とその不変条件を、`ports` と `usecase` はテナントのライフサイクル操作を、`handlers_http` は HTTP アダプターと Repository を所有する。`module.go` は他の Context と接続するための Composition Root である。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Tenant | 独立した認可境界。Client / User / Consent / 鍵 / ポリシーがこの境界に閉じる。URL 上は Realm という別名で表現される。 | テナント, テナント, Realm, realm |
| DefaultTenant | 起動時に自動作成される `realm == "default"` のテナント。ID は固定 UUID の代理キー。単一テナント運用時の互換性と、接頭辞のない HTTP リクエストの解決先を兼ねる。 | デフォルトテナント |
| TenantDisablement | Tenant.disabled_at を設定してテナント単位で `/authorize` / `/token` / `/login` 等を停止する復活可能な操作。テナント物理削除とは独立。 | disable テナント, テナント無効化 |
| EntraFederation | Microsoft Entra ID の検証済みドメインを WS-Federation / WS-Trust の federated IdP として接続するプロファイル。 | Microsoft365Federation, AzureADFederation, M365Federation |
| Disabled | 復活可能な無効化状態。Tenant と (慣例的に) User の disabled_at 経路で共有される。 | disabled |
| Disable | 対象を Disabled に遷移させる。Tenant では `/api/admin/tenants/{id}/disable` から発火。 | disable |
| Enable | Disabled の対象を Active に戻す。Tenant では `/api/admin/tenants/{id}/enable` から発火。 | enable |
| System | IdP プロセス自身。起動時にデフォルトテナントを自動作成する。 |  |
| OAuth2Client | OIDC / OAuth2 プロトコルエンドポイントを呼び出す外部クライアントアプリケーション。 |  |
| EndUser | テナントに所属する人間の利用者。通知メールの受信者であり、その `locale` 属性が通知言語を解決する第 1 段になる。IdManagement が所有する User を公開用の語彙で表す。 | エンドユーザー, 利用者 |
| HardQuota | 超過するとリソース作成が同期的にエラーとなる厳格な上限。 |  |
| SoftQuota | 超過しても作成は成功するが、警告が通知される遅延評価の上限。 |  |
| NotificationTemplate | 利用者へ送る通知メール 1 通の文面定義。`template_key` と `locale` の組で一意に定まり、件名、プレーンテキスト本文、HTML 本文、差出人表示名を持つ。システムが同梱する `ja` / `en` の組込みデフォルトとテナントによる上書きの 2 段で解決する。 | 通知テンプレート, メールテンプレート |
| NotificationTemplateKey | 通知の用途を表す固定識別子。カタログに存在するキーだけが送信・上書きの対象になり、テナントはキー自体を追加できない。 | テンプレートキー |
| NotificationPlaceholder | テンプレート本文に `{{name}}` の形で書ける差し込み変数。template_key ごとに許可集合が決まっており、許可外の変数を含む上書きは保存時に拒否される。 | placeholder, 差し込み変数 |
| NotificationLocaleResolution | 通知 1 通に使うロケールを決める手順。受信者 User の `locale` 属性、テナントの `default_locale`、システムのデフォルトロケールの順に、カタログが対応する最初のロケールを採用する。 | ロケール解決順序 |
| BuiltinNotificationTemplate | システムが同梱する組込みデフォルトテンプレート。テナントによる上書きがない、または上書きが削除された場合に使われる。テナントは編集できず、「デフォルトに戻す」ことでこの文面へ復帰する。 | 組込みデフォルトテンプレート |

## State Transitions

### TenantLifecycle

テナントのライフサイクル。Active で通常稼働、Disable で全プロトコルルートを停止、Enable で復帰。物理削除は本フェーズ対象外。

Initial: `Active` Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | TenantDisabled | — | Disabled |  |
| Disabled | TenantEnabled | — | Active |  |

## Authorization Boundary

認可の意味づけはアプリケーションとそのテストが強制する。本仕様は API の認証を記録するが、ポリシーの DSL は意図的に定義しない。ポリシーの言語を採用する前に、別の work item で Cedar を評価する。

## Design

### Internal Interfaces

#### ResolveTenant
HTTP リクエストの Host ヘッダーとパスから所属テナントを解決する内部インターフェース。

**不変条件: 1 テナント = 1 正規ロケーション = 1 発行者。** テナントは自身の `endpoint_style` が指す正規ロケーションからのみ到達でき、他方の経路では不在として扱う。同一テナントに 2 つのオリジンから到達できると発行者が多義になり、Discovery Metadata の `issuer` が取得元 URL と一致しなくなる（OpenID Connect Discovery 1.0 §4.3 / RFC 8414 §3.3 違反）。

解決順序:
1. `tenant_base_domain` が設定され、Host が `{label}.{tenant_base_domain}` に一致するなら、ラベルをレルムとして対応付ける。見つかったテナントの `endpoint_style` が `Subdomain` でなければ不在として扱う。
2. パスが `/realms/{realm}/...` に一致するなら realm を対応付ける。見つかったテナントの `endpoint_style` が `Path` でなければ不在として扱う。
3. どちらにも一致しないリクエストは、テナントが存在しないものとして扱う。任意の Host や接頭辞のないパスをデフォルトテナントへフォールバックさせない（フェイルクローズ）。テナント境界の侵害を防ぐため、デフォルトでは拒否する。

発行者、URL の接頭辞、Cookie のスコープ、WebAuthn の RP ID は、解決した正規ロケーションから組み立てる。`Path` の場合、発行者は `{base}/realms/{realm}`、`Subdomain` の場合は `{scheme}://{realm}.{tenant_base_domain}` とする。存在しないテナントには `404 tenant_not_found`、無効なテナントには OAuth/OIDC のプロトコルルートで `400 invalid_request` を返し、いずれも存在やステータスの詳細を漏らさない。

### Admin authorization gate

`/admin/*` は認証済みのブラウザーのセッションの `sub` を `User` へ解決し、`User.roles` に `admin` が含まれ、かつアカウントが `disabled_at` でない場合にのみリクエストを許す。`roles` が RBAC のロール名を別のテナントの所属の模型ではなく `User` に直接持つのは、最初の管理の面が User のライフサイクルの管理であり、system_admin がデフォルトの制御面のテナントから操作するからである。テナント単位のロールは、テナントの ID を埋め込む形で `roles` に符号化せず、独自の模型として先送りする。状態を変える管理のリクエストは、セッションによる認証に加えて Origin と CSRF のトークンも検証する。セッションの cookie だけでは、そのリクエストが管理 UI から出たことを証明できないからである。

`disabled_at` は `deleted_at` とは異なる、元に戻せる停止である。停止されたユーザーは新規のサインイン、既存のセッション、トークンの再発行、UserInfo のいずれについても拒否されるが、アカウントとその履歴は復帰のためにそのまま残る。管理の応答は `password_hash` を決して含まない専用の `AdminUserResponse` を使い、管理による変更はすべて、監査で追跡できるよう操作者と対象の両方の `sub` を載せた ドメインの event を発行する。

### Tenant resolution

プロトコルと管理のルートはすべて `/realms/{realm}/...` の下に置く。テナントの作成や更新はテナントをまたぐ制御プレーンの操作であるため、`/realms/default/admin/tenants/...` に置く。これにより、デフォルトテナントのセッション Cookie のパスがそのまま対象を覆い、Cookie のスコープをルートパスまで広げずに済む。パスの接頭辞による解決をサブドメインやヘッダーによる解決より優先したのは、ブラウザーフローにおける OIDC の `iss` クレームと Discovery メタデータを、クライアントがすでに使用した URL から導出できなければならないためである。ヘッダーは転送を越えて保持されず、サブドメインだけの方式では、ローカル開発と CI にワイルドカード DNS とテナントごとの TLS を強いることになる。

`TenantResolver` のミドルウェアは `^/realms/([a-z0-9][a-z0-9-]{0,62})(/|$)` で realm の区間を取り出し、`TenantRepository` で解決して、解決した `Tenant` と発行者の文字列をリクエストコンテキストに付ける。解決できないテナントには一般的な `tenant_not_found` の 404 を返し、停止中のテナントにはプロトコルルートで一般的な `invalid_request` の 400 を返す。どちらのレスポンスも場合の違いを漏らさないため、解決器のレスポンス形式だけからテナントを列挙することはできない。

**正規ロケーションの不変条件。** テナントは `Tenant.endpoint_style`（`Path` または `Subdomain`）で選ばれる、ちょうど 1 つの正規ロケーションと 1 つの発行者を持ち、もう一方のルートは存在しないものとして扱う。これにより、Discovery Metadata の `issuer` は常に取得元 URL と一致する。`Subdomain` を選べるのは配備時に基底ドメインを設定している場合だけであり、設定しない配備は `Path` のままとなり、ワイルドカード DNS も証明書も不要である。`realm` 自体は不変である。発行者に現れ、`Subdomain` のテナントではホスト名にも現れるため、改名は `endpoint_style` の変更と同じ破壊をもたらす。

### Tenant identity: UUID key and realm slug

`tenants` は、不変な代理キー `id UUID` と、可変で一意性を制約された識別子 `realm TEXT` を持つ。これにより、他のすべてのテーブルの `tenant_id` 外部キーが依存する中身の見えない鍵に触れず、realm を改名できる。組織の改名、ブランドの変更、綴りの訂正といった、運用上正当な要求である。外部に露出する語彙、すなわち URL の接頭辞、OIDC の発行者、Discovery Metadata には一貫して `realm` を使い、内部の参照（`tenant_id` 外部キー列、`spec.DefaultTenantID`、Context 内の `TenantID`）にはすべて UUID を使う。解決ミドルウェアが `FindByRealm(realm)` で両者を橋渡しし、管理 API は URL では `realm` でテナントを指しつつ、ユースケースを呼ぶ前に UUID へ解決する。

デフォルトテナントを表す 2 つの定数も同じ分離に従う。`spec.DefaultTenantID` は固定の UUID であり、idmagic が生成する ID の列が全体を通じて UUID 型であることと整合する。`spec.DefaultRealm` は文字列 `"default"` であり、テナントを URL に表す箇所だけで使う。`tenants(id)` を参照する外部キー列は UUID 型とし、`tenant_id` に SQL のデフォルト値は持たせない。すべての挿入で `tenant_id` を明示しなければならず、値が欠けた場合はデフォルトテナントへ黙って混入させず、明確に失敗させる。これはリポジトリ全体の [`tenant_id` retention classes](../../SPECIFICATION.md#2-tenant_id-retention-classes) 方針をさらに厳しくした例である。`tenants` への外部キーを持たない追記専用テーブル、または中身の見えないキーを持つテーブル（`audit_events.tenant_id`、`authentication_event_buckets.tenant_id`）では、`tenant_id` を `UUID` ではなく `TEXT` のままにする。テナントに属さない監査イベントには、UUID 列で自然に表せない番兵値が必要なためである。

### Tenant branding

`TenantBranding` は `Tenant` に埋め込まれた値オブジェクトではなく、`tenant_id` をキーとする独立したエンティティである。`TenantUserAttributeSchema` と同じ形であり、見た目に関わり独立して更新される外装設定が、認可と realm の解決が依存する中核の `Tenant` Aggregate を肥大化させないようにするためである。8 つの項目（製品名、ロゴ、ファビコン、2 つのブランドカラー、サポート導線、法務導線、フッター文言）は Okta、Entra ID、Keycloak、OneLogin に共通する部分集合として選んだ。任意の CSS、HTML、スクリプト、背景画像は、入力の余地を絞るために意図的に除外した。

信頼できないテナントの入力をマークアップや自由形式のスタイルとしてホステッドログインシェルへ渡さない。ブランドカラーは `#rrggbb` として検証し、固定した 2 個の CSS カスタムプロパティ（`--tenant-brand-primary` / `--tenant-brand-accent`）にだけ注入する。テキストフィールドはデフォルトのエスケープ処理で描画し、`dangerouslySetInnerHTML` は決して使わない。`support_url` / `legal_url` は `https://` スキームだけを許可リストに含め、`javascript:`、`data:`、平文の `http://` を書き込み時に拒否する。コントラストは保存時の制約に含めず、管理 UI で確認できるようにして、可読性の結果はテナントが負う。

ロゴとファビコンの取り込みは、アプリケーションのアイコンの保存と同じ検証付きの経路 (先頭バイトの確認、大きさの上限、許可する形式の限定、`nosniff` での配信) を再利用する。両方の呼び出し箇所が同じ振る舞いを保つよう `backend/shared/mediavalidation` の共有の補助へ切り出しているが、保存先は専用の `tenant_branding_assets` のテーブルとし、外装の保存が Application の所有として扱われないようにする。`GetTenantBranding` は常に成功する。設定の欠落、不正な値、資産の欠落のいずれも、ホストされたログイン画面を失敗させるのではなくシステムデフォルトの外装へ退避する。外装の更新はすべて `updated_at` を進め、公開の応答はこれをキャッシュを無効化する版や ETag として露出する。tenant_id は既にキャッシュの鍵 (URL) の一部なので、これだけで古い外装のキャッシュを、テナントをまたぐ漏れなしに無効化できる。

### Tenant resource quotas

リソースの作成にはテナントごとの上限を設ける。共有基盤上で、負荷の高い、または暴走した 1 つのテナントが及ぼす影響範囲を抑えるためである。上限は強制方法によって 2 つに分かれる。**Hard** の上限（`users`、`groups`、`agents`、`applications`、`oauth2_clients`、`active_sessions`、`consents`、`active_jobs`）は作成トランザクション内で同期的に確認し、超過していれば操作を拒否する。**Soft** の上限（`audit_events_retained`、`export_artifacts_bytes`）は操作を成功させ、代わりに非同期の警告と監査イベントを発行する。Hard の上限はデータベースの枯渇を防ぎ、Soft の上限は記録の欠落を避けながら長期的な蓄積を検知する。

新しいテナントには固定のデフォルトの上限を与える (たとえばユーザー 10,000、グループ 1,000、エージェント 100、アプリケーション 50、OAuth2 のクライアント 100、有効なセッション 50,000、同意 10,000、実行中のジョブ 10)。System Admin は特定のテナントの上限を個別に上書きできる。Tenant Admin は自身の上限に対する使用量を閲覧できるが変更はできない。上限を決める権限を、テナント自身ではなく共有のプラットフォームの運用者に残すためである。

上限のないまま既に存在するテナントへ上限を展開すると、即座の締め出しを招きかねない。そこで移行では標準のデフォルトではなく、余裕を持たせた安全な上限を先に割り当てる (たとえば現在の使用量の 2 倍、あるいはデフォルトの 10 倍)。その後に背景で動く突き合わせのジョブが使用量のカウンターを実際の行数と照合し、それから System Admin が意図して上限を締められるようにする。

### Design Decisions

- 管理の認可は、別のテナントの所属の模型ではなく RBAC のロール名を `User.roles` に直接持つ。テナント単位のロールは `roles` に埋め込まず、独自の模型として先送りする。
- Tenant は第一級の Aggregate であり、2 段の認可境界を持つ。自身のテナントに範囲を限る `admin` ロールと、テナントをまたぐ範囲を持ちデフォルトの制御プレーンテナントに置かれる `system_admin` ロールである。
- テナントの解決はサブドメインやヘッダーではなく、パスの接頭辞によるルーティング（`/realms/{realm}/...`）を使う。ブラウザーフローにおける OIDC の `iss` クレームと Discovery Metadata を、クライアントがすでに使った URL から導けるようにするためである。
- テナントは `Tenant.endpoint_style` で選ばれる、ちょうど 1 つの正規ロケーションと発行者を持つ。もう一方の経路を不在として扱うことで、Discovery Metadata の `issuer` を取得元 URL と一致させる。
- `tenants` は主キーを、不変な UUID の代理キーと、可変で一意性を制約された `realm` の識別子へ分割する。依存するすべての `tenant_id` の外部キーが頼る中身の見えない鍵に触れずに realm を改名できるようにするためである。
- 初期データを含め、idmagic が生成する id の列は UUID 型とする。値を外部の権威が定める id (SAML の `entity_id` など) はそうしない。
- `TenantBranding` は `Tenant` に埋め込まれた値オブジェクトではなく `tenant_id` をキーとする独立したエンティティとし、項目を限定し、アプリケーションアイコンの保存処理から再利用した検証付きの取り込み保存を使う。
- テナントの外装色は `#rrggbb` の形式だけを検証し、保存時の制約として WCAG のコントラスト検査を強制しない。管理 UI で結果を確認できるようにし、最終的な可読性はテナントが負う。
- テナントの資源の上限は、同期的に強制する Hard の上限と、非同期に警告する Soft の上限に分ける。デフォルトは固定し、上限の変更は System Admin のみとし、上限の導入前から存在するテナントには余裕を持たせた安全な上限で移行する。
- `TenantGroupAttributeSchema` は `TenantUserAttributeSchema` と同じ配置規則に従う。テナント単位の独自属性スキーマは、それが `IdManagement` のどのプリンシパル（`User` または `Group`）を管理するかによらず `Tenancy` に置く。スキーマの変更とテナント削除時の連鎖は `Tenancy` の関心事だからである。`TenantUserAttributeSchema` と統合せず別の Aggregate にしているのは、`Group` には照合先となる組込みカタログが存在しないためである（理由は IdManagement の設計記録を参照）。

## Scenarios

### REQ-TENANCY-001: 管理者は正規ロケーションの連携情報を取得する
- ACTOR TenantAdministrator
- GIVEN admin が path または subドメインの正規ロケーションから自身のテナントへアクセスしている
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
