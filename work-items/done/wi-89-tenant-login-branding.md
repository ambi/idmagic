---
status: completed
archived_from: 37caf95dc625046d4698589003def2b0168c733f
---

# テナント単位のログイン / アカウント画面ブランディングを導入する

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

全文は `git show 37caf95dc625046d4698589003def2b0168c733f:<このファイルのパス>` で読む。
