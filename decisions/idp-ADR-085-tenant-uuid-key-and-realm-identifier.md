# idp-ADR-085: tenants の不変 UUID キーと mutable な realm 識別子への分離

## ステータス

採用。`tenants` の主キーを「不変の UUID 代理キー」と「mutable な一意 `realm` 識別子」の
2 概念へ分離する。realm rename（組織改称・ブランド変更・誤命名の訂正）を将来可能にする
データモデルの是正であり、外部に露出する URL / issuer 語彙は `realm` のまま維持するため
公開 contract は変わらない。ただし SCL の Tenant モデルと admin API の identity 表現が
変わるため、`spec/contexts/tenancy.yaml` を最小限 SCL-first で更新する。
[idp-ADR-084](file:///Users/tn/src/idmagic/decisions/idp-ADR-084-postgres-column-type-policy.md)
（内部生成 id の UUID 型化）が本 WI に委ねた `tenants.id` の UUID 化を完了させ、
[idp-ADR-082](file:///Users/tn/src/idmagic/decisions/idp-ADR-082-user-domain-id-and-tenant-key-policy.md)
／[idp-ADR-083](file:///Users/tn/src/idmagic/decisions/idp-ADR-083-globally-unique-client-id.md)
の tenant key 方針と整合する。wi-140 で導入。

## コンテキスト

`tenants` の主キーは slug 相当の `id TEXT`（`^[a-z0-9][a-z0-9-]{0,62}$`、`admin` 予約）で、
URL `/realms/{id}/`・OIDC issuer・SAML entityID / WS-Fed 経路にそのまま露出していた。
この `id` は全テーブルの `tenant_id` FK・`spec.DefaultTenantID`・seed から参照されるため、
**URL 識別子（realm 名）を後から変更できない**。realm 名変更は運用上の正当な要求であり、
現状のデータモデルはそれを構造的に禁じていた。

ADR-084 は idmagic が内部生成する id を `UUID` 型へ閉じる方針を定めたが、`tenants.id` は
URL に現れる mutable slug であるため、単純な UUID 化はできず「不変キー」と「mutable slug」の
分離を要するとして本 WI に切り出していた。

## 決定

### 1. UUID 代理キーと `realm` 識別子の分離

`tenants` に不変の `id UUID PRIMARY KEY` を導入し、旧 `id` の slug 値を
`realm TEXT NOT NULL UNIQUE` として分離する。書式 CHECK（`<> 'admin'` かつ
`^[a-z0-9][a-z0-9-]{0,62}$`）は `realm` に移す。`id` はテナント生成時に
`spec.NewUUIDv4()` で採番し、以後不変。`realm` は一意制約の範囲で変更可能（rename の
データモデル的余地。管理 UI は本 WI の Out of Scope）。

### 2. 2 語彙の写像: realm は URL、UUID は内部キー

外部に露出する語彙（URL prefix `/realms/{realm}/`・OIDC issuer・metadata）は一貫して
`realm` を用い、公開 contract の互換を保つ。内部のテナント参照（全 `tenant_id` FK・
`spec.DefaultTenantID`・context の `TenantID`）は UUID を用いる。middleware は
`FindByRealm(realm)` で URL の realm を UUID テナントへ解決し、issuer は
`tenant.Realm` から組み立てる。admin API はテナントを URL 上 `realm` で指定し、内部で
UUID へ解決してから usecase を呼ぶ。

### 3. 既定テナントの二定数

`spec.DefaultTenantID` を固定 UUID `00000000-0000-4000-8000-000000000000`（ADR-084 の
seed UUID 系列に整合、非衝突）とし、URL 語彙用に `spec.DefaultRealm = "default"` を新設する。
`tenant_id == DefaultTenantID` の比較・FK 保存は UUID 同士でそのまま成立し、URL に露出する
2 箇所（control-plane group prefix / urlPrefix）のみ `DefaultRealm` を用いる。

### 4. FK は UUID、tenant_id に SQL デフォルトを置かない

`tenants(id)` を参照する全 FK 列（複合 FK の tenant 側を含む）を `UUID` へ張り替える。
併せて `tenant_id` 列の SQL デフォルト（旧 `DEFAULT 'default'`）を撤廃し、常に明示指定を
要求する。SQL デフォルトは、tenant_id の指定漏れが黙って既定テナントへ流れ込む
cross-tenant 混入のリスクであり、全ての insert は既にドメイン値（ctx 由来の tenant）を
束ねているため不要。`tenants.id` も同様にデフォルトを持たず、生成時に UUID を採番して明示
insert する。FK を張らない append-only / throttling 列は tenant 値を UUID 文字列として
保持しつつ `TEXT` に据え置く: `audit_events.tenant_id`（tenantless イベントは `''` を
明示指定）と `authentication_event_buckets.tenant_id`（複合 PK・FK 無し）。Go 側はいずれも
`string` で統一されるため、TEXT/UUID の別は書き込みコードに影響しない。

## 却下した代替案

- **realm を主キーのまま可変にする**: 一意制約と FK cascade で rename を実現する案。全
  子テーブルの `tenant_id` を `ON UPDATE CASCADE` にしても、発行済みトークン・監査ログ・
  外部連携に埋まった slug までは追随できず、不変キーの不在という根本問題が残る。
- **realm を UUID にも別名でも参照できる二重キー**: 解決経路が二系統になり、権限比較や
  seed の一貫性が壊れやすい。URL=realm / 内部=UUID の単純な写像に劣る。
- **非 FK 列も UUID 化**: `audit_events.tenant_id` は tenantless イベントで `''` を要するため
  UUID にできず、揃えるなら sentinel 表現の再設計が要る。append-only 列に FK も型変更も
  課さない現状維持が安く、値は UUID 文字列で一貫する。

## 影響

- `deploy/schema/postgres.sql`: `tenants` の主キー・`realm` 追加・全 FK の UUID 化。未リリース
  のため宣言的 schema の書き換えで完結する（データ移行なし）。
- `spec/contexts/tenancy.yaml` と Go twin（`spec.Tenant` / validation / 定数）: `id`=UUID、
  `realm` 追加、admin API の identity 表現変更。derived artifacts を再生成する。
- ルーティング・middleware・admin handler・seed・UI が realm↔UUID 写像に追随する。issuer /
  metadata の公開表現は realm のままで互換。
