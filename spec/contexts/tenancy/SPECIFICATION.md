---
context: tenancy
updated_at: 2026-08-15
---

# Tenancy Specification

## Overview

Tenant (Realm) の Aggregate、ライフサイクル、HTTP リクエストからのテナント解決、テナント管理 API を所有する。Tenant は IdMagic のあらゆる Aggregate が属する境界である。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Tenant | 独立した認可境界。URL 上は Realm という別名で表現される。 | テナント, Realm, realm |
| DefaultTenant | 起動時に自動作成される `realm == "default"` のテナント。ID は固定 UUID の代理キー。単一テナント運用時の互換性と、接頭辞のない HTTP リクエストの解決先を兼ねる。 | デフォルトテナント |
| TenantDisablement | Tenant.disabled_at を設定してテナント単位で停止する復活可能な操作。テナント物理削除とは独立。 | テナント無効化 |
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

テナントは `Active` で通常稼働し、`Disable` で全プロトコルルートを停止する。`Enable` で復帰できる。物理削除は対象外とする。

Initial: `Active` Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | TenantDisabled | — | Disabled |  |
| Disabled | TenantEnabled | — | Active |  |

## Design

### Authorization boundary

管理の認可は 2 段である。`admin` ロールは自身のテナント内に閉じ、`system_admin` ロールはテナントを越える操作を持つ。テナントの作成、一覧、更新、無効化、正規ロケーションの切り替え、リソース上限の変更は制御面の操作であり、`admin:tenants_manage` として `system_admin` ロールとデフォルト (制御面) テナントへの所属の両方を要求する。所属テナントの設定 (`admin:settings_read` / `admin:settings_update`)、外装 (`admin:branding_update`)、属性スキーマ、通知テンプレートは `admin` または `system_admin` が自身のテナントに対して変更でき、テナント管理者は自身の上限を参照できるが変更はできない。上限を決める権限をテナント自身ではなく共有基盤の運用者に残すためである。

API アクセストークンでは、ロールに加えて `settings:*` が所属テナントの設定、外装、属性スキーマ、通知テンプレートに、`tenants:*` が制御面のテナント CRUD に対応し、`read` が参照だけを、`write` が変更を許可する。制御面の操作は `admin:tenants_manage` の条件を満たす利用者が自身のテナントで発行したトークンから到達しうるので、`tenants:*` にも到達経路がある。通知テンプレートのプレビューは描画結果を返すだけなので参照系に、テスト送信は実際にメールを送るため変更系に置く。

いずれの経路でも、ロールを持つだけでは足りない。呼び出し元のアカウントが `disabled_at` でないこと、認証済みであること、対象が呼び出し元と同じテナントに属することを合わせて要求する。状態を変える管理リクエストは、セッションによる認証に加えて `Origin` と CSRF トークンも検証する。セッション Cookie だけでは、そのリクエストが管理 UI から出たことを証明できないからである。

テナント解決も認可境界の一部である。テナントは `endpoint_style` が指す正規ロケーションからだけ到達でき、もう一方の経路では不在として扱う。どの経路にも一致しないリクエストをデフォルトテナントへフォールバックさせない。存在しないテナントには `404 tenant_not_found`、無効なテナントにはプロトコルルートで `400 invalid_request` を返し、内部のテナント情報は公開しない。

### Internal Interfaces

#### ResolveTenant
HTTP リクエストの Host ヘッダーとパスから所属テナントを解決する内部インターフェース。

**不変条件: 1 テナント = 1 正規ロケーション = 1 発行者。** テナントは `endpoint_style` が指す正規ロケーションからだけ到達でき、他方の経路では不在として扱う。同一テナントへ 2 つのオリジンから到達できると発行者が一意に定まらず、Discovery Metadata の `issuer` が取得元 URL と一致しなくなる (OpenID Connect Discovery 1.0 §4.3 / RFC 8414 §3.3 違反)。

解決順序:
1. `tenant_base_domain` が設定され、Host が `{label}.{tenant_base_domain}` に一致するなら、ラベルをレルムとして対応付ける。見つかったテナントの `endpoint_style` が `Subdomain` でなければ不在として扱う。
2. パスが `/realms/{realm}/...` に一致するなら realm を対応付ける。見つかったテナントの `endpoint_style` が `Path` でなければ不在として扱う。
3. どちらにも一致しないリクエストは、テナントが存在しないものとして扱う。任意の Host や接頭辞のないパスをデフォルトテナントへフォールバックさせず、フェイルクローズで拒否する。

発行者、URL の接頭辞、Cookie のスコープ、WebAuthn の RP ID は、解決した正規ロケーションから組み立てる。`Path` の場合、発行者は `{base}/realms/{realm}`、`Subdomain` の場合は `{scheme}://{realm}.{tenant_base_domain}` とする。存在しないテナントには `404 tenant_not_found`、無効なテナントには OAuth/OIDC のプロトコルルートで `400 invalid_request` を返し、いずれも存在やステータスの詳細を漏らさない。

### Admin authorization gate

ロール名は `User.roles` に直接保持し、テナント所属を表す別のモデルは設けない。現在の管理対象は User のライフサイクルであり、`system_admin` はデフォルトの制御面テナントから操作するためである。将来テナント単位のロールが必要になっても、テナント ID を `roles` の文字列へ埋め込まず、独立したモデルとして設計する。

`disabled_at` は `deleted_at` とは異なる、元に戻せる停止である。停止されたユーザーは新規のサインイン、既存のセッション、トークンの再発行、UserInfo のいずれも拒否されるが、アカウントとその履歴は復帰のために残る。管理レスポンスは `password_hash` を決して含まない専用の `AdminUserResponse` を使い、管理による変更は監査で追跡できるよう、操作者と対象の両方の `sub` を載せたドメインイベントを発行する。

### Tenant resolution

プロトコルと管理のルートはすべて `/realms/{realm}/...` の下に置く。テナントの作成や更新はテナントをまたぐ制御面の操作なので、`/realms/default/admin/tenants/...` に置く。これにより、デフォルトテナントのセッション Cookie のパスだけで対象を覆い、Cookie のスコープをルートパスまで広げずに済む。デフォルトをパス方式とするのは、ブラウザーフローにおける OIDC の `iss` クレームと Discovery Metadata を、クライアントが使用した URL から導出できるためである。ヘッダーはプロキシを越えて保持されるとは限らず、サブドメイン方式だけではローカル開発と CI にワイルドカード DNS とテナントごとの TLS が必要になる。

`TenantResolver` のミドルウェアは `^/realms/([a-z0-9][a-z0-9-]{0,62})(/|$)` で realm の区間を取り出し、`TenantRepository` で解決して、解決した `Tenant` と発行者の文字列をリクエストコンテキストに付ける。レスポンスの形が場合ごとに変わらないため、解決器のレスポンスだけからテナントを列挙することはできない。

`Subdomain` を選べるのは、デプロイ時に基底ドメインを設定している場合だけである。設定しないデプロイは `Path` のままとなり、ワイルドカード DNS も証明書も必要ない。`realm` は変更できるが、発行者と、`Subdomain` 方式ではホスト名にも現れるため、その変更は `endpoint_style` の変更と同様に既存クライアントとの互換性を壊す。

### Tenant identity: UUID key and realm slug

`tenants` は、不変の代理キー `id UUID` と、変更可能で一意な識別子 `realm TEXT` を持つ。これにより、組織名やブランド名の変更、綴りの訂正で realm を改名しても、他のテーブルの `tenant_id` 外部キーは変更せずに済む。URL の接頭辞、OIDC の発行者、Discovery Metadata など外部に公開する識別子には `realm` を使い、`tenant_id` 外部キー列、`spec.DefaultTenantID`、Context 内の `TenantID` など内部参照には UUID を使う。解決ミドルウェアが `FindByRealm(realm)` で両者を対応付け、管理 API は URL の `realm` をユースケースの呼び出し前に UUID へ解決する。

デフォルトテナントを表す 2 つの定数も同じ分離に従う。`spec.DefaultTenantID` は固定の UUID であり、IdMagic が生成する ID の列が全体を通じて UUID 型であることと整合する。`spec.DefaultRealm` は文字列 `"default"` であり、テナントを URL に表す箇所だけで使う。`tenants(id)` を参照する外部キー列は UUID 型とし、`tenant_id` に SQL のデフォルト値は持たせない。すべての挿入で `tenant_id` を明示しなければならず、値が欠けた場合はデフォルトテナントへ黙って混入させず、明確に失敗させる。これはリポジトリ全体の [`tenant_id` retention classes](../../SPECIFICATION.md#2-tenant_id-retention-classes) 方針をさらに厳しくした例である。`tenants` への外部キーを持たない追記専用テーブル、または不透明なキーを持つテーブル（`audit_events.tenant_id`、`authentication_event_buckets.tenant_id`）では、`tenant_id` を `UUID` ではなく `TEXT` のままにする。テナントに属さない監査イベントには、UUID 列で自然に表せない番兵値が必要なためである。

### Tenant security policy overrides

`Tenant` が持つセキュリティポリシーの上書きは、デプロイ全体の製品既定を緩めず、厳しい方向にだけ働く。パスワードポリシーでは最小長と履歴件数を下げず最大長を上げない。Token Exchange の `max_delegation_depth` ではシステム既定の 3 を超えて上げず、未設定なら 3 を継承する。管理 API の `0` は委譲を全面禁止する値ではなく、上書きを解除して SQL の `NULL` へ戻す操作として扱う。設定取得では現在の任意上書きとシステム既定を別々に返し、管理 UI が継承状態と実効値を区別できるようにする。

OAuth2 Context は `TenantRepository` を直接参照せず、委譲深さを返す小さなポートに依存する。`oauth2/policy_tenancy` アダプターがこのポートを `Tenant` の実効値へ接続する。上書きを読めないときにシステム既定へ退避すると、テナントが意図して下げた認可境界を黙って広げるため、解決失敗では Token Exchange を拒否する。

### Tenant branding

`TenantBranding` は `Tenant` に埋め込まず、`tenant_id` をキーとする独立したエンティティとする。独立して更新される外観設定によって、認可と realm 解決が依存する中核の `Tenant` Aggregate を肥大化させないためである。設定項目は、製品名、ロゴ、ファビコン、2 つのブランドカラー、サポート導線、法務導線、フッター文言に限る。任意の CSS、HTML、スクリプト、背景画像は受け付けない。

信頼できないテナントの入力をマークアップや自由形式のスタイルとしてホステッドログインシェルへ渡さない。ブランドカラーは `#rrggbb` として検証し、固定した 2 個の CSS カスタムプロパティ（`--tenant-brand-primary` / `--tenant-brand-accent`）にだけ注入する。テキストフィールドはデフォルトのエスケープ処理で描画し、`dangerouslySetInnerHTML` は決して使わない。`support_url` / `legal_url` は `https://` スキームだけを許可リストに含め、`javascript:`、`data:`、平文の `http://` を書き込み時に拒否する。コントラストは保存時の制約に含めず、管理 UI で確認できるようにして、可読性の結果はテナントが負う。

ロゴとファビコンには、Application アイコンと同じ検証処理を使う。先頭バイト、サイズ、形式を検証し、`nosniff` を付けて配信する。検証処理は `backend/shared/mediavalidation` で共有するが、保存先は専用の `tenant_branding_assets` テーブルとし、所有権は Tenancy に保つ。`GetTenantBranding` は、設定やアセットが欠けていてもシステムデフォルトへフォールバックし、ログイン画面を失敗させない。更新時は `updated_at` を進め、公開レスポンスではキャッシュ版または ETag として使う。`tenant_id` は URL の一部なので、テナント間でキャッシュを混同せずに古い外観を無効化できる。

### Tenant resource quotas

リソースの作成にはテナントごとの上限を設ける。共有基盤上で、負荷の高い、または暴走した 1 つのテナントが及ぼす影響範囲を抑えるためである。上限は強制方法によって 2 つに分かれる。**Hard** の上限（`users`、`groups`、`agents`、`applications`、`oauth2_clients`、`active_sessions`、`consents`、`active_jobs`、`ssf_streams`）は作成トランザクション内で同期的に確認し、超過していれば操作を拒否する。**Soft** の上限（`audit_events_retained`、`export_artifacts_bytes`）は操作を成功させ、代わりに非同期の警告と監査イベントを発行する。Hard の上限はデータベースの枯渇を防ぎ、Soft の上限は記録の欠落を避けながら長期的な蓄積を検知する。

新しいテナントには固定のデフォルトの上限を与える (たとえばユーザー 10,000、グループ 1,000、エージェント 100、アプリケーション 50、OAuth2 のクライアント 100、有効なセッション 50,000、同意 10,000、実行中のジョブ 10、SSF ストリーム 20)。SSF ストリームの上限は送信側と受信側で分けず、`SsfStream` の行数として 1 つの上限で数える。向きは同じ集合の属性でしかなく、上限を分けても抑えたい資源 (ストリーム 1 本ごとに増える配送先・鍵取得先・保持する配送記録) は変わらないためである。System Admin は特定のテナントの上限を個別に上書きできる。Tenant Admin は自身の上限に対する使用量を閲覧できるが変更はできない。上限を決める権限を、テナント自身ではなく共有のプラットフォームの運用者に残すためである。

既存テナントへ初めて上限を適用する際は、現在の使用量を下回って直ちに操作を拒否しないよう、十分な余裕を持つ値を割り当てる。たとえば現在の使用量の 2 倍またはデフォルトの 10 倍を使う。その後、バックグラウンドの照合ジョブで使用量カウンターと実際の行数を一致させてから、System Admin が意図した値へ上限を引き下げる。

### Design Decisions

- テナントの解決はサブドメインやヘッダーではなく、パスの接頭辞 (`/realms/{realm}/...`) をデフォルトとする。ヘッダーは転送を越えて保持されず、サブドメインだけの方式はローカル開発と CI にワイルドカード DNS とテナントごとの TLS を強いるからである。
- テナントの正規ロケーションはちょうど 1 つとし、もう一方の経路を不在として扱う。2 つのオリジンから同じテナントに到達できると発行者が多義になり、Discovery Metadata の `issuer` が取得元 URL と一致しなくなるからである。
- `tenants` の鍵を、不変な UUID の代理キーと、可変で一意な `realm` へ分割する。組織の改名やブランド変更で realm を変えるときに、すべての `tenant_id` 外部キーが頼る鍵へ触れずに済ませるためである。
- IdMagic が生成する id の列は初期データを含め UUID 型とする。値を外部の権威が定める id (SAML の `entity_id` など) はこの限りではない。
- `TenantBranding` と属性スキーマは `Tenant` に埋め込まず、`tenant_id` をキーとする独立したエンティティとする。見た目や機能ごとの設定が増えるたびに、認可と realm 解決が依存する中核の Aggregate を太らせないためである。
- 外装として受け取るのは限定した項目だけとし、任意の CSS、HTML、スクリプト、背景画像は受け付けない。信頼できないテナントの入力を、共有のログインシェルへマークアップとして渡さないためである。
- 外装色は形式だけを検証し、WCAG のコントラストを保存時の制約にはしない。管理 UI で結果を確認できるようにし、最終的な可読性はテナントが負う。
- リソース上限は、同期的に拒否する Hard と、成功させたうえで警告する Soft に分ける。データベースの枯渇は同期的に止める必要がある一方、監査記録の欠落は上限を理由に起こしてはならないからである。
- 上限の変更は System Admin だけに許す。上限を決める権限を、テナント自身ではなく共有基盤の運用者に残すためである。
- セキュリティポリシーの上書きは、製品既定を厳しくする方向にだけ許す。解決に失敗したときも既定へ退避せず拒否する。退避すると、テナントが意図して狭めた認可境界を黙って広げてしまうからである。
- `TenantGroupAttributeSchema` を `TenantUserAttributeSchema` と統合せず、別の Aggregate として持つ。テナント単位の属性スキーマはどのプリンシパルのものであれ `Tenancy` に置くが、`Group` には照合先となる組込みカタログが存在しないためである。

## Scenarios

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
