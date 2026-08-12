---
context: system
updated_at: 2026-08-13
---

# System Specification

## Overview

外部標準、共有語彙、横断ユーザー体験、context 横断シナリオを所有する。

The React UI is a separate build artifact from the Go API, joined into one origin by a gateway. It owns
the hosted authentication surfaces (login, consent, device), the admin console, and the account portal.

This document is the design record for that boundary: how the SPA and the API divide responsibility,
which browser protections apply, how routing and UI conventions are fixed, and why. The machine-checked
module boundaries are inferred from paths and forbidden imports; run instructions and verification commands live in
`README.md`.

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Locale | UI表示言語を一意に決めるBCP47言語タグ。idmagicは "ja" と "en" のみをサポート対象とし、それ以外は未対応 locale として扱う。 | locale tag, 表示言語コード |
| DisplayLanguage | EndUser または Administrator が言語切り替え UI で明示的に選択した Locale。選択はブラウザに保存され、以後のアクセスで保存済み設定として優先される。 | 表示言語, 言語設定 |
| FallbackLocale | 要求された Locale が未対応、または対応 Locale の辞書に該当 translation key が欠落している場合に表示へ用いる既定 Locale。idmagicでは "en" を既定とする。 | 既定 locale, default locale |
| ConfiguredDefaultLocale | アプリケーション起動時の設定 VITE_DEFAULT_LOCALE により指定する既定 Locale。"ja" または "en" のみを受け付け、未設定または未対応値のときは FallbackLocale を使う。 | startup default locale, configured locale fallback |
| DemoLoginAffordance | HomePage が表示する、ローカルデモ資格情報 (Seeding の development profile が作成する demo user と demo OAuth2 client) を使った authorization_code フローへの近道。Vite dev server 実行時は既定で表示し、それ以外のビルドではアプリケーション起動時の設定 VITE_DEMO_LOGIN_ENABLED を "true" に明示したときだけ表示する。表示条件は development profile が実際に seed 済みかどうかを問わない。 | demo login shortcut, ローカルデモ認証の近道 |
| BackendErrorText | バックエンドが HTTP、OAuth/OIDC redirect、SAML、SCIM などの外部 API 応答で返す利用者向けエラー本文。message、error_description、detail およびプレーンテキストのエラー本文を含む。常に英語であり、表示言語によって変化しない。 | API error message, error description |
| PersistedStateModel | created_at を持ち、作成後に現在状態が更新される場合は updated_at も持つ永続化状態モデルの規約。作成後は不可変で消費・削除のみされる記録モデルは updated_at を持たない。issued_at / granted_at / occurred_at / expires_at / revoked_at などのドメイン時刻は created_at を置き換えない。各 context のモデル定義はこの規約に従う。 |  |
| EndUser | 認証済みまたは認証を試みる一般利用者。 |  |
| Operator | IdP をデプロイ・起動時設定を行う運用者。 |  |
| ResourceOwner | OAuth2/OIDC 認可フローでリソースの所有者として認可判断を行う利用者。EndUser と同一人物を OAuth2 文脈で指す呼称。 |  |
| Administrator | テナント内または横断のリソースを管理する権限を持つ利用者。 |  |
| APIConsumer | HTTP API を直接呼び出す外部クライアント。 |  |
| InterfaceStability | interface の外部契約としての性質を表す区分。stable は互換を保証する外部契約、beta は互換保証前の外部契約、internal は browser session 専用または domain-internal で外部契約に含めない区分。stable/beta は同時 2 版までパス版で提供し、非推奨表明から最低 12 か月は維持する。 | stability tier, 安定性区分 |
| Deprecation | stable/beta の interface を将来削除する予告。deprecated_since 以降は応答に Deprecation ヘッダを付与し、sunset_at が定まれば Sunset ヘッダも付与する。sunset_at は deprecated_since から最低 12 か月後でなければならない。 | 非推奨化 |
| ConfigurationReference | backend プロセスが起動時に読む設定キーの網羅一覧。キー名、値の型、既定値、必須か、読むプロセス、説明を持ち、secret として分類されたキーは値を持たない。Config の定義から生成する。 | 設定リファレンス |

## Standards

### Web Content Accessibility Guidelines 2.2

W3C Recommendation — https://www.w3.org/TR/WCAG22/

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WCAG22-KEYBOARD | required | MUST | すべての認証操作をキーボードだけで完了可能にする。 |
| WCAG22-FOCUS | required | MUST | フォーカスを視認可能にし重要な要素が完全に隠れないようにする。 |
| WCAG22-LABELS-ERRORS | required | MUST | 入力にラベルを付け、エラーをテキストで識別して修正方法を示す。 |
| WCAG22-STATUS | required | MUST | 認証結果や送信エラーをフォーカス移動なしに支援技術へ通知する。 |

### General Data Protection Regulation

Regulation (EU) 2016/679 — https://eur-lex.europa.eu/eli/reg/2016/679/oj

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| GDPR-CONSENT-WITHDRAWAL | required | MUST | ResourceOwner が同意を撤回でき、撤回後の新規発行へ利用しない。Consent / ConsentLifecycle は OAuth2 context が所有する。 |
| GDPR-ERASURE | required | MUST | 削除要求後は法的保存義務を除く PII を定義済み期間内に消去する。消去は IdManagement の UserLifecycle Purge 遷移と Authentication の資格情報破棄が個別に担う。 |
| GDPR-PROCESSING-RECORDS | required | MUST | セキュリティ・認可イベントの監査記録を定義済み期間保持する。保持期間は Audit context が所有する。 |

## Design

### Internal Interfaces

#### BackendErrorResponse
バックエンドの HTTP API とプロトコル endpoint がエラー時に返す共通の外部契約。
message、error_description、detail、またはプレーンテキスト本文のいずれも
BackendErrorText であり、表示言語 (DisplayLanguage) に関わらず常に英語で固定する。
RFC 9457 Problem Details では type の urn:idmagic:error: suffix を stable error code として
解釈でき、UI は detail または title を人間可読な fallback として利用できる。
個別 endpoint の error code と HTTP status はこの契約によって変更しない。
- Result invariant: text_is_english(output.message)
- Result invariant: text_is_english(output.error_description)
- Result invariant: text_is_english(output.detail)

### Deployment boundary

React and Go are separate build artifacts and separate services.

```text
Browser
  |
  | same origin
  v
Gateway / static server (Caddy, Nginx, CDN + proxy, etc.)
  |-- /login, /consent, /device, /status, /admin/* -> React SPA
  `-- /api/* and OAuth/OIDC endpoints                -> Go
```

Caddy is the reference configuration, not a required runtime. Any gateway that preserves the
same-origin boundary, TLS, headers, and routing contract can replace it.

### Authorization transaction

The Go service keeps the complete OAuth authorization request server-side. Its internal UUID is
stored only in a short-lived `HttpOnly`, `Secure` in HTTPS, `SameSite=Lax` transaction cookie.
It is not included in HTML, URLs, or JavaScript-readable application state.

The SPA calls `GET /api/auth/transaction` to obtain only display data such as the screen kind,
client name, and requested scopes. Login and consent commands resolve the transaction from the
cookie.

### Browser protections

- Session and authorization transaction cookies are `HttpOnly`.
- State-changing UI APIs require a double-submit CSRF cookie and `X-CSRF-Token` header.
- State-changing browser APIs require an `Origin` header matching the configured public issuer.
- Consent verifies that the current login session subject matches the authorization transaction.
- Authorization requests expire after ten minutes and completed requests cannot be reused.
- OAuth redirect URIs, PKCE values, scopes, and client identifiers are read from server-side state.
- UI API responses use `Cache-Control: no-store` and never return credentials or internal request IDs.

### API boundary

Browser-facing authentication APIs live under `/api/auth/*`. OAuth/OIDC protocol endpoints retain
their standard paths. Management APIs live under `/api/admin/*` and self-service APIs under
`/api/account/*`; both use explicit authorization policies independently from the login
transaction APIs.

### Admin console and account portal as OIDC RPs

The admin console (`/admin/*`) and account portal (`/account/*`) authenticate as OIDC relying
parties of the IdP itself, using `authorization_code` + PKCE against the IdP's own `/authorize`
and `/token`. They are registered as first-party public clients with fixed UUID `client_id`s
(admin `…0022`, account `…0023`, mirrored in `src/api/oidc.ts` and the bootstrap seed) whose
consent screen is skipped because the resource owner is the IdP user.

Because they are pure SPA RPs, the access token is held in the browser (`sessionStorage`) and sent
as `Authorization: Bearer` to `/api/{admin,account}/*`, which validate it as RFC 9068 resource
servers. This is a deliberate departure from a strict "no tokens in JavaScript" posture; it is
bounded by short-lived access tokens (600 s), `Cache-Control: no-store`, and keeping tokens out of
URLs, logs, and the DOM. The first-party session login (`POST /api/auth/login`) is retained as an
emergency bootstrap path so a broken OIDC client/key configuration cannot lock administrators out.

### Client-side routing

The SPA uses TanStack Router for client-side navigation with file-based routes under
`src/routes/`. The Vite router plugin generates `src/routeTree.gen.ts` and applies automatic code
splitting, including route loaders and components. Route files follow the request path structure:
`admin/route.tsx` and `account/route.tsx` are thin layout routes that render `<Outlet>`, while
`admin/index.tsx`, `account/index.tsx`, and leaf route files own their own `loader`, API requests,
page component, and path params. Detail pages that should not render through a
list page use TanStack Router's trailing underscore convention, for example
`admin/users_/$sub.tsx` for `/admin/users/$sub`. Files prefixed with `-` are route-local helpers
and are excluded from route generation. Internal admin/account navigation uses `<Link>`, so moving
between pages does not reload the document or re-fetch every page's data — only the target route's
loader runs.
The OIDC login guard (`ensureLoggedIn`) runs inside the loader, so it applies to both the initial
load and in-app navigation. Auth-flow transitions (login/consent/callback and the OIDC redirects)
remain full-page navigations by nature. The rendered page kind is asserted to the DOM via
`<meta name="idmagic:page">` for E2E.

### Design Guidelines

- **Prioritize Trust**: Establish a secure experience through calm color palettes, clear service identification, and transparent descriptions of current operations and security states, reassuring users during authentication decisions.
- **Present Critical Information First**: Clear information hierarchies display the page title, requesting party, shared information, next action, and cancellation procedures up-front.
- **Simplify Critical Actions**: Limit screens to a single primary action, visually distinguishing it from reject/cancel operations. Avoid modifying OAuth/OIDC form names, submission values, and transition contracts for UI-specific reasons.
- **Accessibility as Standard**: Support keyboard navigation, visible focus indicators, sufficient color contrast, explicit labels, appropriate `aria-*` attributes, and animations respecting reduced-motion preferences.
- **Density for Enterprise Use**: Avoid excessive animations or consumer-focused decorations. Maintain a structured layout using consistent spacing, typography, borders, and state colors.
- **Responsive Without Data Loss**: Display supplementary details on desktops while prioritizing authentication operations on mobile without omitting service identification or safety warnings.
- **Consistency via Shared Components**: Utilize local components conforming to Tailwind CSS, Radix UI, and shadcn/ui. Avoid ad-hoc implementations of colors, border-radii, focus rings, or disabled states.

### Admin Console Policy

The Admin Console's information design is inspired by directory-centric systems like Keycloak, Okta, and Google Cloud IAM. It uses a left-hand navigation sidebar to identify management targets, displays search, status, and major permissions in a high-density table format, and presents detailed views and modification options within the same context. Destructive operations (such as deletion or disabling) are visually separated from standard read-only views.

- **Tables as Command Centers**: Use list views as the primary workspace, allowing users to search, filter, and review status, MFA config, and roles at a glance.
- **Verify Before Modifying**: Enable users to inspect principal IDs, authentication status, and assigned permissions in detail panes before committing changes.
- **Explicit Permission Modification**: Avoid inline editing for sensitive role changes. Use dedicated configuration screens displaying differences (additions/deletions) before confirmation.
- **Visible Danger Actions**: Highlight dangerous actions with clear descriptions and appropriate warning colors to prevent accidental execution.
- **Secure Credentials**: Display client secrets exactly once upon creation. Re-confirm client deletion only after reviewing affected systems.
- **Scalable Architecture**: Structure navigation to accommodate future modules like groups, applications, and audit logs. Unimplemented features must not appear interactive.
- **Consistent Layout**: Maintain a unified structure across the console using `AdminShell` (headers, sidebars, breadcrumbs, content widths, and action placements).
- **Unauthorized Link Fallback**: Redirect unauthenticated direct requests to `/admin/*` to `/login`, returning to the original target destination upon successful login. Allowed redirection targets are constrained to the current realm's `/admin` path.

*References:*
- [Keycloak Server Administration Guide](https://www.keycloak.org/docs/latest/server_admin/)
- [Okta Manage users](https://help.okta.com/en-us/content/topics/users-groups-profiles/usgp-people.htm)
- [Google Cloud IAM access management](https://cloud.google.com/iam/docs/granting-changing-revoking-access)

### UI Library Selection

The UI foundation balances accessibility and design consistency without relying on complex, pre-packaged themes.

| Library | Role | Selection Rationale |
| --- | --- | --- |
| React + TypeScript | UI and type-safe views | Maintains clear component boundaries and state management from simple login screens to the administrative console. |
| Vite | Dev server and production build | Fast, straightforward generation of static bundles that can be served via API gateways or CDNs. |
| Tailwind CSS | Design tokens and styling | Enables consistent styling (states, responsiveness, accessibility) while preserving enterprise branding controls. |
| Radix UI | Accessible headless primitives | Accessible keyboard handling and ARIA compliance decoupled from visual presentation. |
| Local Components (shadcn/ui layout) | Buttons, Inputs, Labels, Cards, Alerts | Maintained within the repository for easy audit and customization, minimizing runtime dependency overhead. |
| TanStack Router | Type-safe routing | Safe translation of page metadata from the Go backend to target UI views. |
| TanStack Table | Administrative data grid | Separates sorting, filtering, and pagination logic from UI presentation. (Reserved for user/client tables; currently unused in the 4 core login screens). |
| Tabler Icons | Vector icons | Consistent line weights and extensive library to serve as visual aids for states and actions rather than mere decoration. |
| Class Variance Authority / Clsx / Tailwind Merge | Class merging | Type-safe styling variants and runtime merging of conflicting Tailwind classes. |
| Biome | Linter and formatter | Rapid automated enforcement of syntax, style, and code quality guidelines. |

Priorities are accessibility, bundle size, maintainability, design ownership, and preserving API contracts. Introduce new libraries only when existing tools cannot satisfy specific requirements.

### UI navigation and consistency policy

The admin console and account portal follow a set of strict UI consistency and navigation guidelines, applied to Entra federation and to external identity providers (`/admin/identity-providers`).

1. **Detail-then-Edit Navigation Policy**
   - For resource creation or editing, the UI must separate the read-only view (detail) from the write/edit view.
   - The user is first presented with a read-only detail view of the resource configuration, with an explicit "Edit" button that navigates to a dedicated edit route (e.g., `/admin/users/$id/edit` or `/account/profile/edit`).
   - Modals should not be used for primary resource creation or editing; they must use dedicated routed pages to ensure predictable browser "Back" button behavior and deep-linking capabilities.
2. **List-View Action Unification**
   - Action buttons (Detail, Edit, Delete, etc.) in table list views must be visible directly in each row rather than hidden under dropdown/kebab menus.
   - Destructive actions (such as deletion) must use red-toned buttons (`variant="outline" tone="danger"`).
3. **Dynamic Page Titles**
   - Every page must have a dynamic and context-aware browser tab title (e.g., "ユーザー | IdMagic 管理コンソール") defined via the `PAGE_TITLES` map in `src/routes/-page.tsx` and evaluated by the `PageMarker` component.
4. **Terminology Unification**
   - The UI must use the term "監査イベント" (Audit Event) instead of "監査ログ" (Audit Log) to maintain consistency with the underlying specification (`AuditEvent`/`audit_events`).

### Container / Presentation component split

New `*Page.tsx` files (and refactors of existing ones) follow a container/presentation split so that UI rendering can be unit-tested apart from data fetching and side effects:

1. **Split by meaning, not by file.** The exported `XxxPage` function stays a thin container: it owns `useState`, API calls, and effects, and lays out the page's `*Shell` wrapper directly. Do not wrap an entire page in a single `XxxPresentation` twin that re-receives every piece of container state as a prop — that only relocates the same complexity behind an extra layer.
2. **Extract at the section boundary.** Pull out a presentational component for each self-contained unit that benefits from isolated testing — a form with its own validation (e.g. `DefaultPolicyFormPresentation` in `AdminSignInPolicyPage.tsx`), an item list (`PasskeyList`), or a card with interactive state (`TotpEnrollmentForm` in `AccountSecurityPage.tsx`). Purely static, read-only markup can stay inline in the container; it does not need its own component.
3. **Keep presentational props small.** A presentational component should take only the props its own section needs (typically well under 10), plus callbacks for the actions it triggers — never the container's entire state object. If a component's prop list balloons because a page has several independent sections, split it further into one component per section instead of widening the props.
4. **No side effects in presentational components.** They receive data and callbacks and render; `fetch`/`api.*` calls, `useEffect`, and navigation stay in the container (or in a small section-local container, e.g. `DefaultPolicyCard`, when a section manages its own state before delegating to a pure form).
5. **Test what was extracted.** Each extracted presentational component and any pure helper function (date formatting, validation, derived-value calculators) gets a Vitest/Testing Library unit test. Components that wrap `AccountShell`/`AdminShell`/`AuthShell` need a router context to render (those shells use TanStack Router's `Link`); use the `renderWithRouter` test helper (`src/test/renderWithRouter.tsx`) for those instead of skipping the test.

### Design Decisions

- Current UI/runtime design and rationale live in this specification, while run and verification
  instructions live in the relevant README or runbook because they answer operational questions. Source
  paths and imports describe executable structure; no duplicate module ledger is maintained.
- The admin console and account portal are first-party OIDC relying parties of the IdP itself, as pure
  SPA RPs holding the access token in the browser rather than behind a BFF.
- Internally generated id columns, including the admin/account portals' fixed `client_id`s, are typed as
  `UUID` rather than `TEXT`.
- The admin console and account portal follow a fixed set of UI consistency rules: detail-then-edit
  navigation instead of inline/modal editing, in-row list actions instead of kebab menus, dynamic
  per-page browser tab titles, and "監査イベント" instead of "監査ログ" as the audit terminology.
- The tenant-wide default sign-in policy applies to an application as an override, not a composed floor,
  of that application's own policy — the precedent that motivated this file's container/presentation
  component split, first applied to the admin UI built for that decision.
- 起動時設定は `backend/cmd/internal/bootstrap` が所有する単一の Config 型へ集約してパース・検証する。
  Fail-fast の対象は必須値欠落、型・範囲不正、相互に矛盾する組み合わせ (例: persistence が postgres
  なのに DSN が空) であり、検証は listener 起動前に集約エラーとして返し部分起動させない。secret と
  分類したフィールド (DSN、SMTP 資格情報、API キー等) は検証エラー・起動ログ・ConfigurationReference
  のいずれにも値を出さない。全 backend プロセス (idmagic, idmagic-worker, idmagic-batch, idmagic-seed)
  がこの Config を通して環境を読み、`backend/cmd/internal/bootstrap` の外で環境変数を直接読まない。
- ConfigurationReference は Config の定義から生成し、生成物として追跡する。定義と生成物の乖離は
  リポジトリ検証で失敗させ、運用者向けの設定表を手書きで二重管理しない。

## Scenarios

### REQ-SYSTEM-001: Operatorは分離された運用資産でSLOを検証する
- ACTOR Operator
- GIVEN API、UI gateway、event relay は個別の実行単位として配備される
- GIVEN MetricsExposition の公開範囲は management network に制限される
- WHEN Operator が環境 overlay を選んで運用 manifest を適用する
  - ALT PostgreSQL へ到達できない → ReadinessProbe は unavailable を返し、API は新規トラフィックを受けない → LivenessProbe は healthy を維持し、依存障害だけで再起動しない
  - ALT Prometheus Operator が導入されていない → ServiceMonitor は適用対象から外し、標準 Prometheus scrape 設定で MetricsExposition を収集する
- THEN API の liveness、readiness、startup probe は各々 LivenessProbe、ReadinessProbe、StartupProbe を呼ぶ
- THEN Prometheus が MetricsExposition をスクレイプし、OAuth2 の availability、latency、error-rate objectives を表示・評価する

### REQ-SYSTEM-002: orchestration probeはprocess lifecycleと依存状態を区別する
- ACTOR Operator
- WHEN Operator が初期化完了後の liveness、readiness、startup probe を呼ぶ
  - ALT 初期化中または graceful drain 中である → liveness は 200 healthy を維持する → readiness または startup は 503 を返す
  - ALT 構成された永続化依存へ到達できない → readiness は 503 unavailable を返す → liveness は 200 healthy を維持する
- THEN すべて 200 と healthy を返す

### REQ-SYSTEM-003: 明示的に選択した表示言語でホスト認証画面が描画される
- ACTOR EndUser
- GIVEN 未認証セッションで Login 画面を表示している
- WHEN EndUser が表示言語 "en" を選択する
- THEN Login 画面の文言が en 辞書で表示される
- THEN 選択した Locale がブラウザに保存され、以後のアクセスで保存済み設定として優先される

### REQ-SYSTEM-004: 未対応localeは既定localeにフォールバックする
- ACTOR EndUser
- GIVEN ブラウザの言語設定が "fr" である
- GIVEN 表示言語の明示選択も保存済み設定も存在しない
- WHEN EndUser が Login 画面を表示する
- THEN 画面の文言は既定 locale "en" の辞書で表示される

### REQ-SYSTEM-005: 起動時設定の既定localeがフォールバックに使われる
- ACTOR Operator
- GIVEN 表示言語の明示選択、ui_locales ヒント、保存済み設定、対応ブラウザ言語が存在しない
- WHEN Operator が VITE_DEFAULT_LOCALE を "ja" に設定してアプリケーションを起動する
  - ALT VITE_DEFAULT_LOCALE が未設定または未対応値である → 画面の文言は FallbackLocale "en" の辞書で表示される
- WHEN EndUser が画面を表示する
- THEN 画面の文言は ja 辞書で表示される

### REQ-SYSTEM-006: 起動時設定でVite dev server以外でもDemoLoginAffordanceが表示される
- ACTOR Operator
- GIVEN Vite dev server ではなくビルド済み frontend を配備している
- WHEN Operator が VITE_DEMO_LOGIN_ENABLED を "true" に設定してビルドする
  - ALT VITE_DEMO_LOGIN_ENABLED が未設定または "true" 以外である → HomePage は DemoLoginAffordance を表示しない
- WHEN EndUser が HomePage を表示する
- THEN HomePage は DemoLoginAffordance を表示する
- WHEN EndUser が DemoLoginAffordance を選択する
  - ALT development profile が seed されていない → authorization は既知のデモ資格情報が存在せず失敗する
- THEN development profile が seed した demo user の資格情報で authorization_code フローが完了する

### REQ-SYSTEM-007: Vite dev server実行時は設定なしでDemoLoginAffordanceが表示される
- ACTOR EndUser
- GIVEN Vite dev server で frontend を実行している
- WHEN EndUser が HomePage を表示する
- THEN VITE_DEMO_LOGIN_ENABLED の設定に関わらず HomePage は DemoLoginAffordance を表示する

### REQ-SYSTEM-008: OIDC ui_localesヒントにより表示言語が決まる
- ACTOR ResourceOwner
- GIVEN 未認証セッションで表示言語の明示選択も保存済み設定も存在しない
- WHEN "web-app" として ui_locales "en" で認可リクエストを送信する
  - ALT 表示言語が既に明示選択済みの場合は ui_locales ヒントで上書きされない → EndUser が表示言語 "ja" を明示的に選択済みである → "web-app" として ui_locales "en" で認可リクエストを送信する → Login 画面の文言は ja 辞書で表示される
- THEN Login 画面の文言は en 辞書で表示される

### REQ-SYSTEM-009: 管理者が選択した表示言語で管理画面が表示される
- ACTOR Administrator
- GIVEN roles に "admin" を持つ Administrator が認証済みで AdminDashboard を表示している
- WHEN Administrator が表示言語 "en" を選択する
- THEN AdminDashboard の文言が en 辞書で表示される

### REQ-SYSTEM-010: 選択した表示言語で全UI画面が描画される
- ACTOR EndUser
- GIVEN 対応する画面へ遷移できる認証状態である
- WHEN EndUser または Administrator が表示言語 "en" を選択する
  - ALT jaを選択する → 同じ要素がja辞書およびjaの書式で表示される
- WHEN EndUser または Administrator が任意のUI画面を表示する
- THEN 画面、shared shell、dialog、empty state、aria label、状態ラベルがen辞書で表示される
  - ALT 翻訳keyが欠落している → FallbackLocale (en) の対応keyを表示する
- THEN 日時および数値がenの書式で表示される

### REQ-SYSTEM-011: 既知のバックエンドエラーコードはUIで翻訳される
- ACTOR EndUser
- GIVEN UI操作に対しバックエンドがエラー応答を返す
- WHEN バックエンドが既知のstable error codeを返す
  - ALT error codeが未知、またはbackendが任意のmessageかProblem Detailsだけを返す → UIはバックエンドのmessage、error_description、detail、titleのうち利用可能な人間可読文を英語のまま表示する → バックエンドから有効なエラー応答を受信した場合は通信障害用fallbackを表示しない
  - ALT RFC 9457 Problem Detailsのtypeが既知のstable error codeを表す → UIはtypeのurn:idmagic:error: suffixをerror codeとして解釈する → UIが選択済みDisplayLanguageの辞書にあるerror codeの文言を表示する
- THEN UIが選択済みDisplayLanguageの辞書にあるerror codeの文言を表示する

### REQ-SYSTEM-012: PostgreSQLクエリの期限は結果読取完了まで維持される
- ACTOR Operator
- GIVEN PostgreSQL persistenceとquery timeoutが構成されている
- WHEN Systemが共通persistence adapterで単一行または複数行queryを開始する
- THEN queryがRowまたはRowsを返す
- WHEN 呼び出し側が期限内にScanまたはiterationを完了する
  - ALT 結果読取中にquery timeoutの期限へ到達する → 読取はdeadline exceededで中断される → 結果をcloseするとconnectionとtimeout resourceが解放される
  - ALT 単一行queryに該当する行が存在しない → Scanはno rowsを返す → no rowsは正常なquery応答として扱われcircuit breakerの失敗率を増加させない
- THEN 結果がcontext canceledにならず返され、connectionが解放される

### REQ-SYSTEM-013: バックエンドAPIエラーは英語で返る
- ACTOR APIConsumer
- WHEN APIConsumer が不正な JSON を HTTP API に送信する
- THEN System は既存の error code と HTTP status を返す
- THEN System は英語の message を返す
- WHEN OAuth/OIDC redirect endpoint が要求を拒否する
- THEN System は既存の OAuth error code と英語の error_description を返す
- WHEN 未知の内部エラーが発生する
- THEN System は既存の error code と HTTP status を維持し、英語のエラー本文を返す

### REQ-SYSTEM-014: 非推奨のinterfaceを呼ぶとDeprecation/Sunsetヘッダが返る
- ACTOR APIConsumer
- GIVEN stable な interface に deprecated_since が設定されている
- WHEN APIConsumer が非推奨マークされた interface を呼び出す
  - ALT interface に sunset_at も設定されている → 応答に Sunset ヘッダも付与される
  - ALT interface が deprecated_since を設定していない → 応答に Deprecation ヘッダは付与されない
- THEN 応答に Deprecation ヘッダが付与される

### REQ-SYSTEM-015: 管理コンソールとアカウントポータルは失効セッションから同一画面に復帰する
- ACTOR Administrator
- GIVEN Administrator が first-party の管理コンソールで access token を保持している
- GIVEN 保持している access token が失効している
- WHEN Administrator が AdminDashboard で管理 API を呼び出す
- THEN API が 401 を返す
- THEN 保持していた access token / refresh token と OIDC callback state を破棄する
- THEN 直前の画面への同一オリジン相対 return_to を保ったまま再認可を1回だけ開始する
  - ALT 再認可から復旧できない → 再ログイン導線を提示する
- THEN 再ログイン完了後に元の AdminDashboard へ復帰する

### REQ-SYSTEM-016: 起動時設定の検証に失敗するとプロセスは部分起動せず集約エラーで停止する
- ACTOR Operator
- GIVEN Operator が環境変数で backend プロセス (idmagic, idmagic-worker, idmagic-batch, idmagic-seed) の設定を与える
- WHEN プロセスが起動時に Config を集約・検証する
  - ALT 必須値が欠落している → 検証は該当キーを含む集約エラーを返す → プロセスは副作用のある初期化 (listener の待受、永続化依存への接続、seed の適用) を開始せず終了する
  - ALT 値の型・範囲が不正である (数値でない、負の duration 等) → 検証は該当キーを含む集約エラーを返す → プロセスは副作用のある初期化を開始せず終了する
  - ALT 相互に矛盾する組み合わせである (persistence が postgres なのに DSN が空等) → 検証は該当する組み合わせを含む集約エラーを返す → プロセスは副作用のある初期化を開始せず終了する
- THEN 発生したすべての検証エラーが1回の起動試行で集約されて報告される
- THEN 検証エラーおよび起動ログは secret として分類された値 (DSN、SMTP 資格情報、API キー等) を含まない
- WHEN すべての検証を通過する
- THEN プロセスは検証済み Config を用いて初期化を完了する

### REQ-SYSTEM-017: ConfigurationReference は起動時設定の定義から生成され乖離を検出できる
- ACTOR Operator
- GIVEN backend プロセスの起動時設定が Config として一箇所で定義されている
- WHEN ConfigurationReference を生成する
- THEN 生成物は設定可能な各キーについて、キー名、値の型、既定値、必須か、読むプロセス、説明を含む
- THEN 生成物は secret として分類されたキーについて値を含まず、secret である旨のみを示す
- WHEN 生成物と Config の定義を突き合わせる
  - ALT 生成物が定義と一致しない → 突き合わせは失敗し、乖離したキーを報告する
- THEN Operator は Config の実装を読まずに設定可能な全キーを参照できる
