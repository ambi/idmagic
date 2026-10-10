# ブランド設定の設計

この文書は、[ブランド設定](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

ロゴとファビコンには、Application のアイコンと同じ検証処理を使う。
検証処理は `backend/shared/mediavalidation` で共有するが、保存先は専用の `tenant_branding_assets` テーブルとし、管理は Tenancy に残す。

## 信頼性

`GetTenantBranding` は、設定やアセットが欠けていてもシステムデフォルトへ退避し、ログイン画面を失敗させない。

## 性能

更新のたびに `updated_at` を進め、公開の応答ではこれを ETag の版として使う。
`tenant_id` は配信 URL の一部なので、テナントの間でキャッシュを混同せずに古い外観を無効にできる。
