---
context: tenancy
updated_at: 2026-08-11
---

# Tenancy Specification

## Overview

テナント (Realm) 集約、ライフサイクル、HTTP リクエストからのテナント解決、テナント管理 API を所有する。テナントは idmagic のあらゆる集約に共通する境界文脈。

`Tenancy` owns the `Tenant` aggregate and everything that resolves a request to one: path-prefix
routing, the immutable-identity/mutable-slug key split, per-tenant branding, resource quotas, and the
authorization gate that separates admin operations from ordinary user traffic. `domain` holds the
aggregate and its invariants, `ports`/`usecases` the tenant lifecycle operations, and `handlers_http`
the HTTP adapter and repositories; `module.go` is the composition boundary other contexts bind against.

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

## State Transitions

### TenantLifecycle

テナント のライフサイクル。Active で通常稼働、Disable で全プロトコルルートを停止、Enable で復帰。物理削除は本フェーズ対象外。

Initial: `Active`
Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | TenantDisabled | — | Disabled |  |
| Disabled | TenantEnabled | — | Active |  |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Internal Interfaces

#### ResolveTenant
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

### Admin authorization gate

`/admin/*` resolves the authenticated browser session's `sub` to a `User` and allows the request only
when `admin` is present in `User.roles` and the account is not `disabled_at`. `roles` stores RBAC role
names directly on `User` rather than a separate tenant-membership model, because the first admin surface
is User lifecycle management and a system_admin operates from the default control-plane tenant;
tenant-scoped roles are deferred to their own model rather than encoded into `roles` as an embedded
tenant ID. State-changing admin requests additionally verify Origin and a CSRF token on top of
session authentication, since a session cookie alone does not prove the request originated from the
admin UI.

`disabled_at` is a reversible suspension distinct from `deleted_at`: a disabled user is rejected for new
sign-in, existing sessions, token reissue, and UserInfo, but the account and its history remain intact
for reinstatement. Admin responses use a dedicated `AdminUserResponse` that never includes
`password_hash`, and every admin mutation emits a domain event carrying both the actor's and the
target's `sub` for audit traceability.

### Tenant resolution

Every protocol and admin route is mounted under `/realms/{realm}/...`; tenant CRUD, being a cross-tenant
control-plane operation, lives at `/realms/default/admin/tenants/...` so the default tenant's session
cookie path already covers it without widening cookie scope to the root path. Path-prefix
resolution was chosen over subdomain and header-based resolution because a browser flow's OIDC `iss`
claim and Discovery metadata must be derivable from the same URL the client already used: a header
cannot survive a redirect, and a subdomain-only scheme forces wildcard DNS and per-tenant TLS onto local
dev and CI.

`TenantResolver` middleware extracts the realm segment with `^/realms/([a-z0-9][a-z0-9-]{0,62})(/|$)`,
resolves it against `TenantRepository`, and attaches the resolved `Tenant` and issuer string to the
request context. An unresolvable tenant returns a generic `tenant_not_found` 404 and a disabled tenant
returns a generic `invalid_request` 400 on protocol routes — neither response leaks which case occurred,
so tenant enumeration is not possible from the resolver's response shape alone.

**Canonical location invariant.** A tenant has exactly one canonical location and one issuer, selected by
`Tenant.endpoint_style` (`Path` or `Subdomain`); the other route is treated as not found. This replaced
an earlier design that let unprefixed requests fall back to the `default` tenant and offered a
`LEGACY_BARE_ISSUER` escape hatch — both let a single tenant answer from two origins, which violates
OpenID Connect Discovery's requirement that a document's `issuer` match the URL it was fetched from.
`Subdomain` is only selectable when a deployment configures a base domain; deployments that
don't stay on `Path` and require no wildcard DNS or certificates. `realm` itself is immutable — it
appears in both the issuer and, for `Subdomain` tenants, the hostname, so renaming it would carry the
same breakage as changing `endpoint_style`.

### Tenant identity: UUID key and realm slug

`tenants` splits its former single slug primary key into an immutable `id UUID` surrogate key and a
mutable, uniquely-constrained `realm TEXT` identifier, so that a realm can be renamed later (an
operationally legitimate request — organization rename, rebrand, typo correction) without touching the
opaque key every other table's `tenant_id` FK depends on. The externally exposed vocabulary —
URL prefix, OIDC issuer, Discovery metadata — consistently uses `realm`; every internal reference
(`tenant_id` FK columns, `spec.DefaultTenantID`, context-level `TenantID`) uses the UUID. Resolution
middleware bridges the two with `FindByRealm(realm)`, and admin API addresses tenants by `realm` in the
URL while resolving to the UUID before invoking use cases.

Two default-tenant constants follow the same split: `spec.DefaultTenantID` is a fixed UUID, consistent
with idmagic-generated id columns being UUID-typed throughout, and `spec.DefaultRealm` is the string
`"default"` used only where a tenant must appear in a URL. FK columns that reference `tenants(id)` are
UUID-typed, and `tenant_id` carries no SQL default — every insert must specify tenant_id explicitly, so a
missing value fails loudly instead of silently landing in the default tenant. This is a stricter instance
of the repo-wide [`tenant_id` retention classes](../../SPECIFICATION.md#2-tenant_id-retention-classes)
policy: append-only
or opaque-key-keyed tables that don't FK to `tenants` (`audit_events.tenant_id`,
`authentication_event_buckets.tenant_id`) keep `tenant_id` as `TEXT` rather than `UUID`, since
tenant-less audit events need a sentinel value a UUID column can't hold cleanly.

### Tenant branding

`TenantBranding` is a separate entity keyed by `tenant_id`, not a value object embedded in `Tenant` —
the same shape as `TenantUserAttributeSchema` — so that presentational, independently-updated branding
config doesn't grow the core `Tenant` aggregate that authorization and realm resolution depend on.
Its eight fields (product name, logo, favicon, two brand colors, support link, legal link,
footer text) were chosen as the common subset across Okta, Entra ID, Keycloak, and OneLogin; arbitrary
CSS, HTML, scripts, and background images were deliberately excluded to keep the input surface
constrained.

Untrusted tenant input never reaches the hosted login shell as markup or free-form styling. Brand colors
are validated as `#rrggbb` and injected only as two fixed CSS custom properties
(`--tenant-brand-primary` / `--tenant-brand-accent`); text fields render through default escaping, never
`dangerouslySetInnerHTML`; and `support_url`/`legal_url` allowlist the `https://` scheme only, rejecting
`javascript:`, `data:`, and plain `http://` at write time. An earlier version of this design
also rejected color values that failed a WCAG AA contrast check against the default background, but the
admin UI never surfaced that check to the user, leaving no way to save an otherwise-intentional low
contrast brand color — contrast is no longer a save-time constraint; only format validation applies, and
the tenant bears the readability consequence.

Logo and favicon uploads reuse the same validated-blob pipeline as application icon storage (magic-byte
check, size cap, restricted format allowlist, `nosniff` delivery), factored into a shared
`backend/shared/mediavalidation` helper so both call sites stay behaviorally identical, but persist to a
dedicated `tenant_branding_assets` table so branding storage isn't attributed to Application ownership.
`GetTenantBranding` always succeeds: missing config, invalid values, or a missing asset all
fall back to the system default brand rather than failing the hosted login page. Every branding update
bumps `updated_at`, which the public response exposes as a cache-busting version/ETag; tenant_id is
already part of the cache key (the URL), so this alone is enough to invalidate stale cached branding
without cross-tenant leakage.

### Tenant resource quotas

Resource creation is capped per tenant to bound the blast radius of a single noisy or runaway tenant on
shared infrastructure. Quotas split into two enforcement classes: **Hard** quotas
(`users`, `groups`, `agents`, `applications`, `oauth2_clients`, `active_sessions`, `consents`,
`active_jobs`) are checked synchronously inside the creating transaction and reject the operation on
breach; **Soft** quotas (`audit_events_retained`, `export_artifacts_bytes`) allow the operation to
succeed and raise an asynchronous warning/audit event instead. Rate limiting alone was rejected because
it only bounds short bursts, not sustained long-run accumulation, and soft-only enforcement was rejected
because a bug or malicious loop could still exhaust the database before any async warning fires.

New tenants receive fixed default limits (e.g. 10,000 users, 1,000 groups, 100 agents, 50 applications,
100 OAuth2 clients, 50,000 active sessions, 10,000 consents, 10 active jobs). A System Admin can override
a specific tenant's limits individually; a Tenant Admin can view usage against its own limits but cannot
change them, keeping quota authority with the operator of the shared platform rather than the tenant
itself.

Rolling quotas out onto tenants that already exist without limits risks an immediate lockout, so
migration assigns a generously large safe ceiling up front (e.g. double current usage, or the default
times ten) rather than the standard default; a background reconciliation job then reconciles usage
counters against actual row counts, after which a System Admin can tighten limits deliberately.

### Design Decisions

- Admin authorization stores RBAC role names directly on `User.roles` rather than a separate
  tenant-membership model, with tenant-scoped roles deferred to their own model instead of being
  embedded into `roles`.
- Tenant is a first-class aggregate with a two-tier authorization boundary: an `admin` role scoped to
  its own tenant, and a `system_admin` role scoped across tenants and housed in the default
  control-plane tenant.
- Tenant resolution uses path-prefix routing (`/realms/{realm}/...`) rather than subdomain or
  header-based resolution, so a browser flow's OIDC `iss` claim and Discovery metadata can be derived
  from the same URL the client already used.
- A tenant has exactly one canonical location and issuer, selected by `Tenant.endpoint_style`; this
  replaced an earlier bare-issuer fallback and `LEGACY_BARE_ISSUER` escape hatch that let a single
  tenant answer from two origins.
- `tenants` splits its primary key into an immutable UUID surrogate key and a mutable, uniquely
  constrained `realm` identifier, so a realm can be renamed without touching the opaque key every
  dependent `tenant_id` FK relies on.
- idmagic-generated id columns, including seed data, are UUID-typed; ids whose values are defined by an
  external authority (e.g. SAML `entity_id`) are not.
- `TenantBranding` is a separate entity keyed by `tenant_id` rather than a value object embedded in
  `Tenant`, with a constrained field set and validated-blob upload storage reused from application icon
  storage.
- Tenant branding color values are validated by `#rrggbb` format only; a WCAG contrast check is not
  enforced as a save-time constraint, superseding an earlier version of the branding design that did.
- Tenant resource quotas split into synchronously-enforced Hard quotas and asynchronously-warned Soft
  quotas, with fixed defaults, System-Admin-only limit changes, and a generous safe-ceiling migration
  for tenants that predate quotas.
- `TenantGroupAttributeSchema` follows `TenantUserAttributeSchema`'s placement: tenant-scoped custom
  attribute schemas live in `Tenancy` regardless of which `IdManagement` principal (`User` or `Group`)
  they govern, since schema churn and cascade-on-tenant-delete are `Tenancy` concerns. It is a distinct
  aggregate from `TenantUserAttributeSchema` rather than a unified one, because `Group` has no builtin
  catalog to union against (see IdManagement's design record for why).

## Scenarios

### REQ-TENANCY-001: 管理者は正規ロケーションの連携情報を取得する
- ACTOR TenantAdministrator
- GIVEN admin が path または subdomain の正規ロケーションから自身のテナントへアクセスしている
- WHEN admin が連携エンドポイント画面を開く
  - ALT admin が別テナントの realm を URL として指定しようとする → 対象指定パラメータは存在せず、解決済みテナント以外の情報は返らない
- THEN server は request tenant の canonical issuer から OAuth/OIDC、SAML、WS-Federation、SCIM、管理API、本人APIの URL を導出する
- THEN 画面はOAuth/OIDC、SAML、WS-Federation、APIのprotocol単位で情報をまとめ、SAML配下ではdefaultを含むprofileごとにentityID、metadata、SSO、SLO、署名証明書を一組で表示する
- THEN 画面はread-onlyでdiscoveryとmetadataを正本として案内し、個別値をコピーまたは証明書をダウンロードできる
- THEN canonical issuerと同一originで配信するgatewayは、表示した公開protocol URLを対応するserverエンドポイントへ転送する
- THEN 返却値に client secret、API token、秘密鍵は含まれない

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
- THEN 管理画面は各設定済み色に現在値と「既定に戻す」操作を表示する
- WHEN 管理者がプライマリカラーを既定に戻して保存する
- THEN UpdateTenantBranding には primary_color の空文字列が送られる
- WHEN 未認証の利用者が login 画面を開く
- THEN login / consent / account portal に設定したロゴが表示され、login 画面にはプライマリカラーのシステム既定・設定済みアクセントカラー・指定ラベルの footer リンク・フッターテキストも表示される
  - ALT realm 配下の logo_url が gateway で backend に転送されない → 画像取得は成功せず、管理者は設定の成功応答だけでは表示可能と判断しない

### REQ-TENANCY-005: 不正な branding 入力は拒否されシステム既定にフォールバックする
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が認証済みである
- WHEN "operator" が branding を一度も設定していないテナントで login 画面を開く
- THEN login 画面はシステム既定 (IdMagic) のブランディングを表示する
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
- GIVEN tenant_base_domain が設定されている
- GIVEN テナント "acme" の endpoint_style は Subdomain である
- WHEN Host "acme.{tenant_base_domain}" の "/authorize" にリクエストを送る
- THEN 解決されたテナントは "acme" で、その branding のログイン画面が表示される
- THEN session cookie は __Host- prefix と Path=/ を持ち Domain 属性を持たない
- THEN WebAuthn RP ID は "acme.{tenant_base_domain}" である

### REQ-TENANCY-008: 未知のサブドメインは default テナントに解決されない
- ACTOR OAuth2Client
- GIVEN tenant_base_domain が設定されている
- GIVEN realm "unknown" のテナントは存在しない
- WHEN Host "unknown.{tenant_base_domain}" の "/authorize" にリクエストを送る
- THEN 404 tenant_not_found になり、default テナントにも他のどのテナントにも到達しない

### REQ-TENANCY-009: テナントは自分の正規ロケーション以外からは到達できない
- ACTOR OAuth2Client
- GIVEN tenant_base_domain が設定されている
- GIVEN テナント "acme" の endpoint_style は Subdomain である
- GIVEN テナント "beta" の endpoint_style は Path である
- WHEN "/realms/acme/authorize" にリクエストを送る
  - ALT Host "beta.{tenant_base_domain}" の "/authorize" にリクエストを送る → beta は Path なのでサブドメイン経路では不在として扱われ 404 になる
  - ALT Host "acme.{tenant_base_domain}" の "/realms/beta/authorize" にリクエストを送る → acme の origin から beta へ到達することはできず 404 になる
- THEN acme は Subdomain なので path prefix 経路では不在として扱われ 404 になる

### REQ-TENANCY-010: discovery の issuer は取得元 URL と一致する
- ACTOR OAuth2Client
- GIVEN tenant_base_domain が設定されている
- GIVEN テナント "default" の endpoint_style は Path、テナント "acme" の endpoint_style は Subdomain である
- WHEN "{base}/realms/default/.well-known/openid-configuration" を取得する
- THEN issuer は "{base}/realms/default" であり、取得元 URL の prefix と一致する
- WHEN "https://acme.{tenant_base_domain}/.well-known/openid-configuration" を取得する
- THEN issuer は "https://acme.{tenant_base_domain}" であり、取得元 URL の prefix と一致する
- THEN どちらの応答もエンドポイントURLを自分の正規ロケーション配下だけで組み立てる

### REQ-TENANCY-011: System管理者はテナントの正規ロケーションを切り替えられる
- ACTOR SystemAdministrator
- GIVEN system_admin ロールを持つ "sysadmin" が認証済みである
- GIVEN tenant_base_domain が設定されている
- GIVEN テナント "acme" の endpoint_style は Path である
- WHEN "sysadmin" が SetTenantEndpointStyle で acme を Subdomain に切り替える
  - ALT tenant_base_domain が設定されていない配備で Subdomain を指定する → InvalidRequestError で拒否され endpoint_style は変わらない
- THEN acme は "acme.{tenant_base_domain}" からのみ到達できるようになる
- THEN "{base}/realms/acme/..." は 404 になる
- THEN issuer と WebAuthn RP ID が新しい正規ロケーション由来の値に変わる

### REQ-TENANCY-012: System管理者はテナントのクォータ上限を調整できる
- ACTOR SystemAdministrator
- GIVEN system_admin ロールを持つ "sysadmin" が認証済みである
- WHEN "sysadmin" が UpdateTenantQuota を呼び出し users 上限を 20000 に増やす
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
  - ALT "hanako" の locale 属性が未設定で、テナントの default_locale が "ja" である → テナント既定の "ja" が採用され、日本語のメールが届く
  - ALT "hanako" の locale 属性が未設定で、テナントの default_locale も未設定である → システム既定 locale が採用され、その locale のメールが届く
  - ALT "hanako" の locale 属性がカタログに同梱翻訳の無い locale である → 未対応 locale は飛ばして次の段が採用され、空の本文は送られない
- THEN 件名と本文が組込み既定の ja テンプレートで描画されたメールが届く
- THEN メールはプレーンテキストと HTML の両方を含む
- THEN 本文のリセットリンクはリクエストの発行元 URL から組み立てられており、開くとパスワード再設定画面に到達する

### REQ-TENANCY-016: テナントの通知テンプレート上書きは組込み既定より優先される
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が認証済みである
- WHEN "operator" が ListNotificationTemplates を呼び出す
- THEN 全 template_key × 全サポート locale が customized=false で一覧される
- WHEN "operator" が PasswordReset / ja の件名と本文を上書きして UpdateNotificationTemplate を実行する
- THEN NotificationTemplateUpdated が発行され、当該テンプレートは customized=true になる
- THEN 以後 ja の利用者に届くパスワードリセットメールは上書きした件名と本文で送られる
  - ALT 上書きしていない en の利用者にメールが送られる → en は組込み既定のまま描画され、ja の上書きは影響しない
- WHEN "operator" が ResetNotificationTemplate を実行する
  - ALT 上書きが存在しないテンプレートに ResetNotificationTemplate を実行する → 冪等に成功し、組込み既定のままとなる
- THEN NotificationTemplateReset が発行され、当該テンプレートは組込み既定に戻る

### REQ-TENANCY-017: 許可されていない差し込み変数を含むテンプレート上書きは保存時に拒否される
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が認証済みである
- WHEN "operator" が PasswordReset の本文に許可集合外の変数 `{{password}}` を書いて保存を試みる
  - ALT "operator" が HTML 本文を空にしてテキスト本文だけを保存しようとする → InvalidRequestError で拒否され、片方だけの上書きは作られない
  - ALT "operator" がカタログに無い locale を指定して保存を試みる → InvalidRequestError で拒否される
  - ALT "operator" が差出人メールアドレスの上書きを試みる → アドレスを上書きする入力は受け付けず、上書きできるのは表示名だけである
- THEN InvalidRequestError で拒否され、上書きは保存されない
- THEN 以後も利用者には組込み既定のリセットメールが届き、リンクが欠けたメールは配られない

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
- GIVEN roles=["admin"] のユーザー "operator" が管理画面の設定を開いている
- WHEN 管理者 "operator" がパスワードの最小長を更新する
- THEN 更新後の設定に新しい最小長が反映される

### REQ-TENANCY-020: 管理者はテナント固有のグループ属性スキーマを定義できる
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が認証済みである
- WHEN "operator" が group custom attribute "cost_center" (type=string, required=false) を追加する
  - ALT 既存 key と重複する key を追加する → 更新は InvalidGroupAttributeSchemaError で拒否される
- THEN 更新後のスキーマに追加した属性が含まれ "TenantGroupAttributeSchemaUpdated" が発行される
