---
depends_on: []
status: pending
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
- [ ] T001 [SCL] tenancy の Branding/Asset、public read/admin update/upload/delete interfaces、validation/invariants/scenarios を追加して再生成する。
- [ ] T002 [Domain/Persistence] constrained Branding value、version、tenant repository fields と asset reference migration を実装する。
- [ ] T003 [Asset] application icon storageを抽象化してtenant branding logo adapterを追加し、magic byte/size/nosniffを共通contract testする。
- [ ] T004 [HTTP] public realm branding + ETag、admin update/logo routes、権限/tenant resolution/cache header を追加する。
- [ ] T005 [UI/Email] auth/account shell と email template に safe tokenized theme を適用し、admin preview/editor を追加する。
- [ ] T006 [Verify] XSS/CSS injection、contrast、broken asset、ETag invalidation、realm切替、default fallback、visual/a11y regression を検証する。

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
