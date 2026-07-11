---
status: accepted
authors: [tn]
created_at: 2026-07-12
---

# ADR-096: Tenant branding as an independent tenant-scoped entity with constrained tokens and reused validated blob storage

## コンテキスト
idmagic は login / consent / device / account portal を共通の無地デザインで配信しており、テナントが自社ブランドで hosted UI を出せない (wi-89)。Okta / Entra ID / Keycloak / OneLogin はいずれも hosted login のブランディングを標準機能として持ち、マルチテナント IdP のデモとしても頻出の要求である。

一方でテナント入力をそのまま hosted UI (未認証の login 画面を含む) に反映する機能は、XSS / open redirect / 保存型インジェクションの面を広げる。安全な範囲 (画像 + 限定トークン + テキスト / リンク) に絞り、任意 CSS / HTML 差し込みは構造的に受け付けない必要がある。ロゴ画像アップロードは [[ADR-073-application-icon-upload-storage]] が確立した検証済み blob 保存パターンと同種の課題 (magic byte 検証、サイズ上限、content sniffing 対策) を持つ。

## 決定

`spec/contexts/tenancy.yaml` の `models.TenantBranding` と `interfaces.GetTenantBranding` / `UpdateTenantBranding` / `UploadTenantBrandingAsset` / `DeleteTenantBrandingAsset` に反映。

1. **TenantBranding は Tenant 集約に埋め込まず、`tenant_id` を identity とする独立 entity として持つ。** これは `TenantUserAttributeSchema` (tenancy.yaml: 「tenant aggregate には埋め込まず独立 aggregate として持ち、tenant 削除時に cascade する」) と同じ理由による。Tenant 本体は認可境界の中核 (realm 解決・無効化・パスワードポリシー) であり、feature ごとの sparse config を列として積み増すと `tenants` テーブルと Tenant 集約が肥大化し続ける。branding 専用の repository / table に分離することで、Tenant 集約と `tenants` テーブルは今の形を保ったまま、branding だけ独立に読み書きできる。
2. **フィールドは Okta (Brands / custom sign-in page)・Entra ID (Company branding)・Keycloak (realm theme の displayName/logo)・OneLogin (custom branding) の共通項を基準に選ぶ。** いずれの製品も (a) ロゴ、(b) favicon をロゴとは別のアセットとして持つ、(c) sign-in 画面のテキスト/表示名、(d) support / help リンク、(e) 利用規約等の法務リンクを footer に出す、という構成が共通する。これに基づき以下の 8 フィールド (すべて optional) を持つ:
   - `product_name`: sign-in 画面に出す表示名 (Entra ID の sign-in page text / Keycloak の realm displayName に相当)。
   - `logo_object_key` / `logo_url`: メインロゴ画像。
   - `favicon_object_key` / `favicon_url`: favicon 画像。ロゴと同じ検証パイプラインを通すがブラウザタブ用の別アセットとして持つ (Okta / Entra ID 双方が別アセットとして扱う)。
   - `primary_color` / `accent_color`: ブランドカラー 2 トークン (Okta / OneLogin のプライマリ / ボタンカラーに相当)。
   - `support_url`: サポート / ヘルプリンク (Okta の Support link / OneLogin の help link 相当)。
   - `legal_url`: 利用規約・プライバシー等の法務リンク。Terms と Privacy を別 URL に分けるユースケースは薄く、フィールド数と UI の複雑さに見合わないため単一フィールドにまとめる。
   - `footer_text`: 補足テキスト (著作権表示等)。
   任意 CSS プロパティ・任意 HTML・任意スクリプト・背景画像 (Entra ID / OneLogin にはあるが面積が大きく XSS/レイアウト面が増えるため) は本 work item のスコープ外として見送る。
3. **色は `#rrggbb` 形式の 2 トークンのみを受け付け、ログインシェルは CSS custom properties (`--tenant-brand-primary` / `--tenant-brand-accent`) としてのみ注入する。** 任意プロパティ名・任意セレクタ・任意宣言は受け付けない。加えて既定の背景/前景に対するコントラスト比 (WCAG AA, 4.5:1) を満たさない値は保存時に拒否し、システム既定色にフォールバックする。
4. **テキスト (`product_name`, `footer_text`) はプレーンテキストとして保存し、描画側の既定エスケープでのみ表示する。** `dangerouslySetInnerHTML` 等の raw HTML 差し込みは使わない。
5. **リンク (`support_url`, `legal_url`) は `https://` scheme のみを allowlist する。** `javascript:` / `data:` / 平文 `http://` は保存時に拒否する。
6. **ロゴ / favicon は ADR-073 と同じ検証済み blob 保存パターン (magic byte 判定、256 KiB 上限、PNG/JPEG/WebP/GIF 限定、SVG 除外、配信は保存済み content-type 固定 + nosniff) を再利用するが、`tenant_branding_assets` という専用テーブル・専用 object_key 空間に分離** し、Application icon storage と ownership を混同しない。`kind` 列 (`logo` | `favicon`) で 1 テーブルに両アセットを収める。magic byte 判定ロジックは `backend/shared/mediavalidation` に抽出し、Application icon と Tenant branding asset の双方が同じ関数を呼ぶ (contract test で挙動を固定する)。
7. **PostgreSQL 上の branding config は専用テーブル `tenant_brandings` (PK: `tenant_id`) の個別 nullable 列として持ち、JSONB にまとめない。** 各フィールドが個別の長さ/形式制約 (ADR-084 の column type policy) を持つため、CHECK 制約を列単位で付けられる個別列の方が schema header が既に flag している「ネストした JSONB オブジェクト」アンチパターン (`users.lifecycle`) を増やさずに済む。行が存在しない、または全列 NULL の状態を「branding 未設定」として扱う。
8. **GetTenantBranding は常に成功する。** branding 未設定・値不正・アセット欠損のいずれでもシステム既定 (IdMagic ブランド) にフォールバックし、hosted login エンドポイント自体を止めない。
9. **キャッシュ無効化のため branding 更新のたびに `updated_at` を version として公開レスポンスに含め、ETag として使う。** tenant_id はキャッシュキー (URL) に既に含まれるため、テナント間のキャッシュ混同は起きない。

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
