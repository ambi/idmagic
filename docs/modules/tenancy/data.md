# Tenancy のデータ

この文書は、Tenancy の Aggregate の永続化と、テナントの識別子の扱いを扱う。
システム全体のデータ設計は[データベース](../../design/data/database.md)に従う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `Tenant` | `realm`、表示名、状態、`endpoint_style`、セキュリティポリシーの上書き、通知のデフォルト言語 | すべてのモジュールの Aggregate が `tenant_id` で参照する |
| `TenantBranding` | ブランド設定の項目とアセット | `Tenant` を `tenant_id` で参照する |
| `TenantUserAttributeSchema`、`TenantGroupAttributeSchema` | カスタム属性の定義 | `Tenant` を `tenant_id` で参照する |
| `NotificationTemplate` | 一つの `template_key` と `locale` の文面の上書き | `Tenant` を `tenant_id` で参照する |
| `TenantQuota` | リソースごとの上限の上書きと使用量 | `Tenant` を `tenant_id` で参照する |

`Tenant` は、不変の UUID の `id` と、変更できて一意な `realm` の二つの識別子を持つ。
URL の接頭辞、OIDC の発行者、Discovery Metadata など外部に示す識別子には `realm` を、`tenant_id` の外部キーなど内部の参照には `id` を使う。

起動時に自動で作成する `realm` が `default` のテナント（デフォルトテナント）は、制御面のテナントを兼ねる。
`system_admin` のロールを持つデフォルトテナントの User だけが、テナントを越える操作を行える。

## テナントの識別子

`tenants` は、不変の代理キー `id UUID` と、変更できて一意な `realm TEXT` を持つ。
鍵を分ける判断は、[テナントのキーを不変な UUID と可変な realm に分ける](decisions.md#テナントのキーを不変な-uuid-と可変な-realm-に分ける)。

| 識別子 | 使う場所 |
| --- | --- |
| `realm` | URL の接頭辞、OIDC の発行者、Discovery Metadata など、外部に示す箇所 |
| `id` | `tenant_id` の外部キー、`spec.DefaultTenantID`、リクエストコンテキストの `TenantID` など、内部の参照 |

デフォルトテナントを表す二つの定数も、同じ分け方に従う。

| 定数 | 値 | 使う場所 |
| --- | --- | --- |
| `spec.DefaultTenantID` | 固定の UUID | 内部の参照 |
| `spec.DefaultRealm` | 文字列 `"default"` | テナントを URL に表す箇所 |

## `tenant_id` のカラム

`tenants(id)` を参照する外部キーのカラムは `UUID` 型とし、`tenant_id` に SQL のデフォルト値を持たせない。
すべての挿入で `tenant_id` を明示し、値が欠けた場合はデフォルトテナントへ黙って混入させずに失敗させる。
これは、リポジトリ全体の[`tenant_id` の保持区分](../../design/data/database.md#tenant_id-の保持区分)をさらに厳しくしたものである。

`tenants` への外部キーを持たない追記専用のテーブルと、不透明なキーを持つテーブル（`audit_events.tenant_id`、`authentication_event_buckets.tenant_id`）では、`tenant_id` を `UUID` ではなく `TEXT` のままにする。
テナントに属さない監査イベントには、`UUID` のカラムで自然に表せない番兵値が必要だからである。

## Aggregate の保存先

| Aggregate | 保存先 | 実装 |
| --- | --- | --- |
| `Tenant` | `tenants` | `backend/tenancy/internal/db_postgres` |
| `TenantBranding` | `tenant_brandings`、画像は `tenant_branding_assets` | `backend/tenancy/internal/db_postgres` |
| `NotificationTemplate` | `notification_templates` | `backend/tenancy/internal/db_postgres` |
| `TenantQuota` | 上限の上書きは `tenant_quotas`、使用量のカウンターは `tenant_usages` | `backend/tenancy/internal/db_postgres` |
| `TenantUserAttributeSchema`、`TenantGroupAttributeSchema` | `tenant_user_attribute_schemas`、`tenant_group_attribute_schemas` | ポートは Tenancy が定め、実装は `IdManagement` の `user` と `group` のスライスが持つ |
