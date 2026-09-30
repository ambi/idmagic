# Tenancy の内部設計

## テナント識別子：UUID キーと realm slug

`tenants` は、不変の代理キー `id UUID` と、変更可能で一意な識別子 `realm TEXT` を持つ。これにより、組織名やブランド名の変更、綴りの訂正で realm を改名しても、他のテーブルの `tenant_id` 外部キーは変更せずに済む。URL の接頭辞、OIDC の発行者、Discovery Metadata など外部に公開する識別子には `realm` を使い、`tenant_id` 外部キーカラム、`spec.DefaultTenantID`、Context 内の `TenantID` など内部参照には UUID を使う。解決ミドルウェアが `FindByRealm(realm)` で両者を対応付け、管理 API は URL の `realm` をユースケースの呼び出し前に UUID へ解決する。

デフォルトテナントを表す 2 つの定数も同じ分離に従う。`spec.DefaultTenantID` は固定の UUID であり、IdMagic が生成する ID のカラムが全体を通じて UUID 型であることと整合する。`spec.DefaultRealm` は文字列 `"default"` であり、テナントを URL に表す箇所だけで使う。`tenants(id)` を参照する外部キーカラムは UUID 型とし、`tenant_id` に SQL のデフォルト値は持たせない。すべての挿入で `tenant_id` を明示しなければならず、値が欠けた場合はデフォルトテナントへ黙って混入させず、明確に失敗させる。これはリポジトリ全体の [`tenant_id` の保持区分](../../design/data/database.md#tenant_id-の保持区分) 方針をさらに厳しくした例である。`tenants` への外部キーを持たない追記専用テーブル、または不透明なキーを持つテーブル（`audit_events.tenant_id`、`authentication_event_buckets.tenant_id`）では、`tenant_id` を `UUID` ではなく `TEXT` のままにする。テナントに属さない監査イベントには、UUID カラムで自然に表せない番兵値が必要なためである。
