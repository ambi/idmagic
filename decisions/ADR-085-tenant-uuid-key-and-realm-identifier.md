---
status: accepted
authors: [tn]
created_at: 2026-07-05
---

# ADR-085: tenants の不変 UUID キーと mutable な realm 識別子への分離

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

`tenants` の主キーを「不変の UUID 代理キー」と「mutable な一意 `realm` 識別子」の
2 概念へ分離する。realm rename（組織改称・ブランド変更・誤命名の訂正）を将来可能にする
データモデルの是正であり、外部に露出する URL / issuer 語彙は `realm` のまま維持するため
公開 contract は変わらない。ただし SCL の Tenant モデルと admin API の identity 表現が
変わるため、`spec/contexts/tenancy.yaml` を最小限 SCL-first で更新する。
[ADR-084](file:///Users/tn/src/idmagic/decisions/ADR-084-postgres-column-type-policy.md)
（内部生成 id の UUID 型化）が本 WI に委ねた `tenants.id` の UUID 化を完了させ、
[ADR-082](file:///Users/tn/src/idmagic/decisions/ADR-082-user-domain-id-and-tenant-key-policy.md)
／[ADR-083](file:///Users/tn/src/idmagic/decisions/ADR-083-globally-unique-client-id.md)
の tenant key 方針と整合する。wi-140 で導入。「realm を主キーのまま可変にする」案は
`ON UPDATE CASCADE` で子テーブルを追随させても発行済みトークンや監査ログに埋まった slug
までは追随できず不変キー不在の根本問題が残るため、「realm を UUID にも別名でも参照できる
二重キー」案は解決経路が二系統になり権限比較や seed の一貫性を壊しやすいため、いずれも
却下した。

UUID/realm の写像規則、既定テナントの二定数、FK と `tenant_id` の型・デフォルト方針の詳細は
[`backend/tenancy/ARCHITECTURE.md`](../backend/tenancy/ARCHITECTURE.md) を正とし、
[root `ARCHITECTURE.md` の `tenant_id` retention classes](../ARCHITECTURE.md#2-tenant_id-retention-classes)
と合わせて読む。

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
