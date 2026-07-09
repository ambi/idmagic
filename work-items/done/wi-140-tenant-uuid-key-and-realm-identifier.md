---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-05
risk_notes: |
  tenant_id はほぼ全テーブルの FK であり、URL ルーティング (/realms/{id}/)、
  spec.DefaultTenantID、seed、UI、外部プロトコル (OIDC issuer, SAML/WS-Fed) の
  テナント解決に埋め込まれている。PK 分離は最大級の波及を持つため、独立 WI として
  慎重に段階分割する。
---

# tenants の不変 UUID キーと mutable な realm 識別子への分離

## Motivation
`tenants` の PK は現在 slug 相当の `id TEXT`（`^[a-z0-9][a-z0-9-]{0,62}$`、URL
`/realms/{id}/` に露出）である。この id が全テーブルの `tenant_id` FK・ルーティング・
`spec.DefaultTenantID` から参照されているため、**テナントの URL 識別子（slug/realm 名）を
変更できない**。realm 名変更は運用上の正当な要求（組織改称・ブランド変更・誤命名の訂正）で
あり、現状は不可能。

[ADR-084](../decisions/ADR-084-postgres-column-type-policy.md) は、idmagic が内部
生成する id を `UUID` 型に閉じる方針を定め、`tenants.id` の UUID 化は URL に現れる mutable
slug の分離を要するため本 WI に分離した。

## Scope
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

## Out of Scope
- realm rename の管理 UI 実装（本 WI はキー分離とデータモデルまで。UI は後続で可）。
- tenant 以外の id 方針変更（ADR-084 で完了済み）。

## Verification
- `just yaml-check-work-items`
- `just check-ids`
- `just yaml-check`（SCL を変更した場合）
- `just verify-go`
- `just verify`
- 手動確認: `realm` を変更しても既存 tenant のデータ・トークン・割当が UUID キー経由で
  保たれること。

## Risk Notes
tenant_id は最も広く参照される列であり、PK 分離は schema・ルーティング・公開 issuer・
seed・UI に波及する。段階分割（UUID 列追加 → 参照張り替え → 旧 PK 降格）とし、外部に出る
テナント表現の互換を必ず評価する。

## Plan
波及調査の結果、`spec.DefaultTenantID` は大半が **内部 FK 値** (`tenant_id`) として使われて
おり、値を UUID に変えても UUID==UUID 比較・FK 保存はそのまま成立する。値が **URL に露出**
するのは 2 箇所のみ:
- `internal/shared/adapters/http/server/routes.go:94` control-plane group prefix
- `internal/shared/adapters/http/support/tenant_middleware.go` control-plane urlPrefix

これらを新設 `spec.DefaultRealm`（`"default"`）に置換し、issuer/URL は realm 語彙、内部キーは
UUID、という 2 語彙の写像に整理する。以下の層順（内→外）で実装し、各層でテストを緑化する。

1. **decision**: `ADR-085` — 不変 UUID 代理キー + mutable `realm` slug の分離を記録。
   既定テナント UUID は `00000000-0000-4000-8000-000000000000`、realm は `default`。
   非 FK の `audit_events.tenant_id`（`''` sentinel）/ `authentication_event_buckets.tenant_id`
   は TEXT のまま UUID 文字列を保持する方針も明記。
2. **SCL-first**: `spec/contexts/tenancy.yaml` — Tenant.id を UUID・`realm` を追加、
   `TenantCreateRequest.id`→`realm`、`AdminSettingsResponse`/`TenantSummaryResponse` に
   `realm`、glossary/interface param（`{tenant_id}` の意味）を realm 語彙へ。`just yaml-check`。
3. **spec (Go twin)**: `DefaultTenantID`→UUID・`DefaultRealm` 追加、`Tenant.Realm` 追加、
   validation を `realm` slug へ移動（`ID` は UUID）。
4. **usecases**: `EnsureDefault` を UUID+realm へ、`Create(realm, displayName)` が
   `NewUUIDv4()` で UUID 採番。`Update`/`SetDisabled` は UUID キーのまま。tests 緑化。
5. **adapters**: `TenantRepository.FindByRealm` 追加（memory/postgres）、middleware を
   realm 解決 + issuer=realm、control-plane を `DefaultRealm`、admin handler は realm→UUID
   解決後に usecase 呼び出し、`AdminSettingsResponse`/account context に realm。tests 緑化。
6. **schema/seed**: `tenants` を `id UUID PK` + `realm TEXT UNIQUE NOT NULL`（書式 CHECK を
   realm へ）、全 FK `tenant_id`→UUID・`DEFAULT` を既定 UUID、非 FK は TEXT 据置。
   seed/federation は URL のみ realm、FK 値は UUID。
7. **UI**: types/SystemTenantsPage/AdminSettingsPage/api・account context に realm、
   SCIM URL・admin gating を realm ベースへ。e2e fixture。
8. **verify/commit**: `just yaml-check` / `verify-go` / `verify`、completion 追記、done へ移動、
   英語 Conventional Commit（push しない）。

## Completion

- **Completed At**: 2026-07-05
- **Decision**: [ADR-085](../decisions/ADR-085-tenant-uuid-key-and-realm-identifier.md)
  を新設し、`tenants` を「不変 UUID 代理キー (`id`)」と「mutable な URL slug (`realm`)」の
  2 概念へ分離した。公開語彙（URL `/realms/{realm}/`・OIDC issuer）は realm で維持し、
  内部の全 tenant 参照は UUID キーに統一した。
- **Summary**:
  - **SCL**: `spec/contexts/tenancy.yaml` の `Tenant` を id=UUID + `realm` slug に分離。
    `TenantCreateRequest.id`→`realm`、`AdminSettingsResponse` / `TenantSummaryResponse` に
    `realm` を追加、`DefaultTenant` glossary と `ResolveTenant` 記述を realm 解決へ更新。
    派生物（HTML / JSON Schema / OpenAPI）を再生成。
  - **spec (Go twin)**: `DefaultTenantID` を固定 UUID `00000000-0000-4000-8000-000000000000`
    に変更、`DefaultRealm = "default"` を新設、`spec.Tenant.Realm` を追加、validation の
    slug/予約語チェックを `ID`→`Realm` に移動（`ID` は非空 UUID）。
  - **usecases**: `EnsureDefault` を UUID+realm、`Create(realm, displayName)` が
    `NewUUIDv4()` で UUID 採番。`Update` / `SetDisabled` は不変 UUID キーのまま。
  - **adapters**: `TenantRepository.FindByRealm` を追加（memory / postgres）。middleware は
    URL の realm を `FindByRealm` で UUID テナントへ解決し、issuer / URL prefix を realm から
    組み立て。control-plane group prefix を `DefaultRealm` に変更。admin tenant handler は
    realm→UUID 解決後に usecase を呼ぶ。`AdminSettingsResponse` と account context に
    `realm` を露出。
  - **schema**: `tenants` を `id UUID PK` + `realm TEXT UNIQUE NOT NULL`（書式 CHECK を realm へ）
    に再定義。`tenants(id)` を参照する全 FK（複合 FK の tenant 側含む）を `UUID` へ張り替え。
    非 FK の `audit_events.tenant_id` / `authentication_event_buckets.tenant_id` は TEXT 据置
    （UUID 文字列を保持）。
  - **UI**: `AdminTenant` / `AdminSettings` / `AccountContextResponse` に `realm` を追加。
    system tenants 画面の作成フォーム・一覧表示・disable gating を realm ベースへ、admin API 呼び出しの
    テナント指定を realm に変更。SCIM エンドポイント URL と control-plane gating を realm で構築。
- **Design change (レビュー反映)**: 当初 `tenant_id` FK に既定テナント UUID の SQL `DEFAULT` を
  置いたが、指定漏れが黙って既定テナントへ流れ込む cross-tenant 混入リスクとなるため撤廃し、常に
  明示指定を要求する方針とした（`audit_events.tenant_id` の `DEFAULT ''` も同様に撤廃）。
- **Out of Scope（未実装）**: realm rename の管理 UI（データモデルとキー分離まで）。
- **Verification Results**:
  - `just yaml-check-work-items` — 成功（140 files OK）
  - `just check-ids` — 成功（220 ids OK）
  - `just yaml-check` — 成功
  - `just verify-go` — 成功（format / lint / build / test、embedded-postgres 往復テスト含む green）
  - `just verify` — 成功（verify-go + verify-ui、exit 0）
