---
status: accepted
authors: [tn]
created_at: 2026-07-12
superseded_by: [ADR-097]  # 決定 3（コントラスト比の扱い）のみ
---

# ADR-096: Tenant branding as an independent tenant-scoped entity with constrained tokens and reused validated blob storage

## コンテキスト
idmagic は login / consent / device / account portal を共通の無地デザインで配信しており、テナントが自社ブランドで hosted UI を出せない (wi-89)。Okta / Entra ID / Keycloak / OneLogin はいずれも hosted login のブランディングを標準機能として持ち、マルチテナント IdP のデモとしても頻出の要求である。

一方でテナント入力をそのまま hosted UI (未認証の login 画面を含む) に反映する機能は、XSS / open redirect / 保存型インジェクションの面を広げる。安全な範囲 (画像 + 限定トークン + テキスト / リンク) に絞り、任意 CSS / HTML 差し込みは構造的に受け付けない必要がある。ロゴ画像アップロードは [[ADR-073-application-icon-upload-storage]] が確立した検証済み blob 保存パターンと同種の課題 (magic byte 検証、サイズ上限、content sniffing 対策) を持つ。

## 決定

`spec/contexts/tenancy.yaml` の `models.TenantBranding` と `interfaces.GetTenantBranding` / `UpdateTenantBranding` / `UploadTenantBrandingAsset` / `DeleteTenantBrandingAsset` に反映。

TenantBranding は Tenant 集約に埋め込まず、`tenant_id` を identity とする独立 entity として持つ。
`TenantUserAttributeSchema` と同じ理由で、Tenant 本体は認可境界の中核 (realm 解決・無効化・
パスワードポリシー) であり、feature ごとの sparse config を列として積み増すと Tenant 集約が
肥大化し続けるため、branding 専用の repository / table に分離する。フィールドは Okta / Entra
ID / Keycloak / OneLogin の共通項 (ロゴ・favicon・sign-in 画面テキスト・support/legal リンク・
footer) を基準に選び、任意 CSS / HTML / スクリプト / 背景画像は XSS とレイアウト崩れの面が
大きすぎるため本 work item のスコープ外として見送る。ロゴ / favicon 保存は
[ADR-073](file:///Users/tn/src/idmagic/decisions/ADR-073-application-icon-upload-storage.md) が
確立した検証済み blob 保存パターン (magic byte 判定・サイズ上限・content sniffing 対策) を
`backend/shared/mediavalidation` へ抽出して再利用しつつ、Application icon storage と
ownership を混同しないよう専用テーブルに分離する。

現在のフィールド一覧・安全トークン (CSS custom property 注入・https-only リンク allowlist・
プレーンテキスト描画)・テーブル設計・キャッシュ (`updated_at` を ETag とする) の詳細は
[`backend/tenancy/ARCHITECTURE.md`](../backend/tenancy/ARCHITECTURE.md) に置く。

## 却下した代替案
- **Tenant 集約に embed する value object**: `tenants` テーブルに feature ごとの列を積み増すことになり、Tenant 集約 (realm 解決・無効化等の中核処理) と branding (presentational, 更新頻度も権限も別) の関心が同じ行に同居し続ける。`TenantUserAttributeSchema` で既に採用した「独立 entity に分離する」方針と矛盾する。
- **背景画像 / 任意 CSS / HTML 差し込み**: XSS・レイアウト崩れの面が大きすぎる。要件は「安全な範囲のブランディング」であり任意インジェクションと大アセットは work item のスコープ外。将来別 work item で検討する。
- **単一 JSONB `branding` 列**: フィールドごとの長さ/形式制約を列制約として書けず、schema header が既に flag している nested JSONB アンチパターンを再生産する。
- **外部 object storage / CDN への委譲**: ADR-073 と同じ理由でデモ IdP の起動容易性を優先し見送る。

## 影響
- SCL に `TenantBranding` (`kind: entity`, identity: `tenant_id`)、`GetTenantBranding` / `UpdateTenantBranding` / `UploadTenantBrandingAsset` / `DeleteTenantBrandingAsset` interfaces、`TenantBrandingUpdated` event、`TenantBrandingSafeTokens` / `TenantBrandingSafeAssetServing` / `TenantBrandingLinkAllowlist` invariants、`BrandingUpdate` permission を追加する。
- PostgreSQL schema に `tenant_brandings` テーブル (CHECK 制約付き個別列) と `tenant_branding_assets` テーブル (`kind` 列で logo/favicon を区別) を追加する。`tenants` テーブル自体は変更しない。
- `backend/tenancy` に `TenantBrandingRepository` (`TenantUserAttributeSchemaRepository` と同じ形) を追加する。
- `backend/shared/mediavalidation` に画像 magic byte 検証を抽出し、`backend/application` の既存ロジックをそれに委譲する。
- auth shell (login/consent/device)・account portal・admin console に branding 読み込み/編集 UI を追加する。
