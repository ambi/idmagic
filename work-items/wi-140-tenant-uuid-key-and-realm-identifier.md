---
id: wi-140-tenant-uuid-key-and-realm-identifier
title: tenants の不変 UUID キーと mutable な realm 識別子への分離
created_at: 2026-07-05
authors: [tn]
status: pending
risk: high
risk_notes: |
  tenant_id はほぼ全テーブルの FK であり、URL ルーティング (/realms/{id}/)、
  spec.DefaultTenantID、seed、UI、外部プロトコル (OIDC issuer, SAML/WS-Fed) の
  テナント解決に埋め込まれている。PK 分離は最大級の波及を持つため、独立 WI として
  慎重に段階分割する。
---

# Motivation
`tenants` の PK は現在 slug 相当の `id TEXT`（`^[a-z0-9][a-z0-9-]{0,62}$`、URL
`/realms/{id}/` に露出）である。この id が全テーブルの `tenant_id` FK・ルーティング・
`spec.DefaultTenantID` から参照されているため、**テナントの URL 識別子（slug/realm 名）を
変更できない**。realm 名変更は運用上の正当な要求（組織改称・ブランド変更・誤命名の訂正）で
あり、現状は不可能。

[idp-ADR-084](../decisions/idp-ADR-084-postgres-column-type-policy.md) は、idmagic が内部
生成する id を `UUID` 型に閉じる方針を定め、`tenants.id` の UUID 化は URL に現れる mutable
slug の分離を要するため本 WI に分離した。

# Scope
- **decision / documentation**:
  - `tenants` を「不変 UUID の代理キー」と「mutable な一意 realm 識別子」の 2 概念に分離する
    設計を ADR 化する（ADR-084 §4 と ADR-082/083 の tenant key 方針を前提）。
  - mutable 識別子のカラム名を `realm` とする（URL `/realms/{realm}/` に整合）。旧 `slug`
    語は用いない。
- **schema**:
  - `tenants` に不変 UUID PK を導入し、`realm`（旧 `id` の slug 値）を `UNIQUE NOT NULL`
    カラムとして分離する。書式 CHECK は `realm` に移す。
  - 全テーブルの `tenant_id` FK を tenants の UUID PK 参照に張り替える。複合キー
    (`UNIQUE (tenant_id, ...)` / composite FK) の tenant 側を UUID に整合させる。
- **implementation**:
  - URL ルーティングと外部プロトコルのテナント解決を `realm`（URL 語彙）→ UUID（内部キー）
    の写像に整理する。issuer / metadata など公開 contract に現れるテナント表現の互換を確認する。
  - `spec.DefaultTenantID` と bootstrap/seed を UUID キー + `realm='default'` に整合させる。
  - Postgres / memory adapter、UI のテナント表現を追随させる。realm rename ユースケースの
    可否と手順を定義する。
- **spec**:
  - テナント識別子が SCL の公開 contract（issuer 等）に影響する場合のみ SCL-first で最小限
    更新し derived artifacts を再生成する。

# Out of Scope
- realm rename の管理 UI 実装（本 WI はキー分離とデータモデルまで。UI は後続で可）。
- tenant 以外の id 方針変更（ADR-084 で完了済み）。

# Verification
- `just yaml-check-work-items`
- `just check-ids`
- `just yaml-check`（SCL を変更した場合）
- `just verify-go`
- `just verify`
- 手動確認: `realm` を変更しても既存 tenant のデータ・トークン・割当が UUID キー経由で
  保たれること。

# Risk Notes
tenant_id は最も広く参照される列であり、PK 分離は schema・ルーティング・公開 issuer・
seed・UI に波及する。段階分割（UUID 列追加 → 参照張り替え → 旧 PK 降格）とし、外部に出る
テナント表現の互換を必ず評価する。
