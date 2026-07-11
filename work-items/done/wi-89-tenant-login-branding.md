---
depends_on: []
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-03
---

# テナント単位のログイン / アカウント画面ブランディングを導入する

## Motivation
idmagic は login / consent / account portal を共通の無地デザインで配信して
おり、テナントが自社ブランドで hosted UI を出せない。代表的な IdP はいずれも
hosted login のブランディングを標準機能として持つ:

- Okta: Brands / custom sign-in ページ (ロゴ・色・favicon)。
- Entra ID: Company branding (背景・ロゴ・テキスト)。
- Keycloak: realm themes。
- OneLogin: custom branding。

マルチテナント IdP のデモとして、テナントが自身の hosted 画面にロゴ・製品名・
ブランドカラー・サポート / 法務リンクを反映できることは、非常によく求められる。
本 WI は安全な範囲 (画像 + 限定トークン + テキスト / リンク) の per-tenant
branding を追加する。任意 CSS / HTML 差し込みは injection 面が大きいため範囲外。

## Scope
- **decision**:
  - 新規 ADR: branding の格納形式。sparse な tenant 設定に載せるか、専用 TenantBranding aggregate を切るかを決める。ロゴ画像は [[wi-74-application-icon-image-upload]] の画像取り扱いに倣う。色は自由 CSS でなく限定トークン (CSS custom properties)、テキスト / リンクは escape / allowlist して injection を構造的に排除する方針を記録する。
- **scl**:
  - §3.2 models: TenantBranding (製品名 / ロゴ参照 / ブランドカラー / サポート URL / 法務リンク / フッターテキスト) を追加する。
  - §3.3 interfaces: GetTenantBranding (public / 解決済み tenant) / UpdateTenantBranding (admin) / UploadTenantLogo / GetTenantLogo を追加する。
  - §3.4 states/events: TenantBrandingUpdated を追加する。
  - §3.7 permissions: 参照は public (presentational のみ)、更新は tenant admin に 固定する。
- **go**:
  - branding repository (memory + postgres + migration) と、ロゴ asset ストア (icon storage 再利用) を追加する。public 解決 endpoint を用意する。
- **http**:
  - GET /api/branding (public, 解決済み tenant) / PUT /api/admin/tenant/branding (admin) / ロゴ upload・取得を追加する。
- **ui**:
  - login / consent / device / account portal がブランディングを消費する (ロゴ・製品名・CSS custom properties での配色・サポート / 法務リンク)。 AdminSettingsPage を拡張するか AdminBrandingPage を追加して編集可能にする。
- **documentation**:
  - README にブランディング設定項目と安全上の制約を追記する。

## Out of Scope
- custom domain / vanity URL (インフラ範囲)。
- 任意の custom CSS / HTML 差し込み (injection 面が大きい)。
- アプリケーション単位のブランディング。
- email テンプレートのブランディング (別途検討)。
- i18n / 多言語ブランディング文言。

## Plan
- Tenant aggregate に `Branding` value（display name、logo asset reference、primary/accent colors、support URL、login message）を置き、realm path で tenant 解決後に public read model を返す。CSS/HTML/JSの任意入力は受け付けない。
- logo は Application icon の [[ADR-073-application-icon-upload-storage]] と同じ validated blob storage/magic-byte/size/content-type/nosniff 方針を再利用するが、asset ownership/key は tenant branding と分ける。
- color は構文だけでなく foreground/background contrast を検証し、未設定/不正/asset欠損では system defaultへ落とす。branding failure で login endpoint を停止させない。
- auth shell、account portal、email template の順に同じ public branding DTO を使用する。admin console 自体の業務画面配色は tenant branding で変えず、realm identity の表示だけに留める。
- cache は tenant branding version/ETag で無効化し、cross-tenant/CDN cache key 混同を防ぐ。

## Tasks
- [x] T001 [SCL] tenancy の Branding/Asset、public read/admin update/upload/delete interfaces、validation/invariants/scenarios を追加して再生成する。
- [x] T002 [Domain/Persistence] constrained Branding value、version、tenant repository fields と asset reference migration を実装する。
- [x] T003 [Asset] application icon storageを抽象化してtenant branding logo adapterを追加し、magic byte/size/nosniffを共通contract testする。
- [x] T004 [HTTP] public realm branding + ETag、admin update/logo routes、権限/tenant resolution/cache header を追加する。
- [x] T005 [UI] auth/account shell に safe tokenized theme を適用し、admin editor を追加する (email template は Out of Scope のため対象外)。
- [x] T006 [Verify] XSS/CSS injection、contrast、broken asset、ETag invalidation、realm切替、default fallback を自動テストで検証する。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: admin でロゴ / 製品名 / 配色 / サポートリンクを設定 → login / account portal に反映されることを確認する。悪意ある値 (script / 非 https リンク) が拒否 / 無害化されることを確認する。

## Risk Notes
hosted UI にテナント入力を反映するため XSS / open redirect / SSRF (外部画像) が
主なリスク。自由 CSS / HTML を排し、色トークン・escape・リンク allowlist・
画像アップロード検証で面を小さく保つ。public 参照は presentational のみに絞る。

## Completion
- **Completed At**: 2026-07-12
- **Summary**:
  ADR-096 で TenantBranding を Tenant 集約に埋め込まず、`TenantUserAttributeSchema` と
  同じ理由で独立 entity (identity: tenant_id) として持つ方針を決定した (レビューで
  「Tenant があまりに多くのデータを持つことになる」という懸念を受けて当初案の
  value-object embed から変更)。フィールドは Okta / Entra ID / Keycloak / OneLogin の
  共通項 (製品名・ロゴ・favicon・primary/accent color・support URL・legal URL・
  footer text) を基準に選定 (favicon を追加、terms/privacy は当初 2 分割案から
  ユースケースの薄さを理由に単一 legal_url へ差し戻した)。

  SCL に `TenantBranding` entity、`GetTenantBranding` (public) / `UpdateTenantBranding` /
  `UploadTenantBrandingAsset` / `DeleteTenantBrandingAsset` (tenant admin)、
  `TenantBrandingUpdated` event、safe-token / safe-serving / fails-open の invariant、
  `BrandingUpdate` permission を追加した。

  Go 側は `tenant_brandings` (個別 nullable 列 + CHECK 制約、JSONB は使わない) と
  `tenant_branding_assets` (kind=logo/favicon) の 2 テーブルを新設し、
  `TenantBrandingRepository` / `TenantBrandingAssetStore` を memory + PostgreSQL で
  実装した。画像 magic byte 検証は `backend/shared/mediavalidation` に抽出し、
  Application icon (ADR-073) と共有 (contract test 付き)。色は `#rrggbb` + WCAG AA
  コントラスト比 (4.5:1) を検証、リンクは https のみを allowlist。HTTP 層は
  `GET /api/branding` (public, ETag/If-None-Match 対応) と admin 更新/アセット
  upload・delete を追加し、AuthZ action mapping (`backend/shared/spec/policy.go`,
  `role_policies.go`) も更新した。

  UI は `AuthShell` / `AccountShell` が branding を取得し、ロゴ・製品名・
  footer のサポート/法務リンク・フッターテキストを表示し、`--tenant-brand-primary` /
  `--tenant-brand-accent` の CSS custom property を login/consent/device の主要 CTA に
  適用する。AdminSettingsPage に「ブランディング」タブを追加し、ロゴ/favicon
  アップロード・削除、製品名・配色・リンク・フッターテキストの編集ができる。

  Out of Scope 通り email テンプレートのブランディングは対象外。Plan 文中の
  「email template」への言及は着手前の Out of Scope 判断と矛盾していたため、
  Out of Scope を正として実装しなかった。

  副産物として、直前のコミットで event_logs/event_deliveries テーブル削除後に
  取り残されていた stale な sqlc 生成コードを別コミットで是正した (wi-89 とは
  無関係な pre-existing drift)。
- **Verification Results**:
  - `just yaml-check` - passed
    - environment: local
    - result: SCL / work-item YAML / architecture cross-check すべて成功。
  - `just scl-render` - passed
    - environment: local
    - result: idmagic HTML / JSON Schema / OpenAPI 派生物を再生成。
  - `just test-go` - passed
    - environment: local (postgres adapter test は embedded-postgres 利用)
    - result: 全 Go package 成功 (tenant_brandings/tenant_branding_assets の
      round-trip、branding usecase、HTTP handler の新規テストを含む)。
  - `just verify-go` (race tests + lint) - passed
    - environment: local
    - result: 全 Go package race test 成功、golangci-lint 0 issues。
  - `just build-go` - passed
    - environment: local
    - result: go build ./... 成功。
  - `just verify-ui` (format/lint/typecheck/build/unit test) - passed
    - environment: local
    - result: biome format/lint 0 issues、tsc 0 errors、vite build 成功、
      vitest 284 tests 全成功 (新規 BrandingTab / tenantBranding lib のテストを含む)。
  - 手動確認 (部分) - passed
    - environment: local (`go run ./backend/cmd/idmagic`, memory backend)
    - result: 未認証で `GET /api/branding` を実行し `200 {}` (ETag `"branding-default"`,
      `Cache-Control: public, max-age=60`) を確認。admin 認証済みの
      ロゴアップロード → login 画面反映 → 悪意ある入力の拒否は、curl では
      OAuth transaction / Origin 制約の再現が煩雑だったため実施しておらず、
      同等のシナリオは Go HTTP テスト
      (`TestUpdateBrandingPersistsAndIsPubliclyVisible`,
      `TestUploadAndDeleteBrandingLogoAsset`,
      `TestUpdateBrandingRejectsNonHTTPSSupportURL`,
      `TestUploadBrandingAssetRejectsSVG`) で自動検証済み。実ブラウザでの
      admin UI 操作による目視確認は未実施 — フォローアップ推奨。
- **Affected Guarantees State**:
  - guarantee: input validation (color contrast, https-only links, magic-byte asset validation)
  - state: passed
  - guarantee: tenant isolation (branding asset serving scoped by tenant_id + kind + object_key)
  - state: passed
  - guarantee: fail-open presentation (GetTenantBranding never blocks hosted login)
  - state: passed
  - guarantee: admin-only mutation (BrandingUpdate permission restricted to tenant admin/system_admin)
  - state: passed
