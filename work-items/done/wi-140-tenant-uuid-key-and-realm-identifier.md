---
status: completed
archived_from: c846d5e549cbbe72c662efb5054e4abdb53d670b
---

# tenants の不変 UUID キーと mutable な realm 識別子への分離

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

全文は `git show c846d5e549cbbe72c662efb5054e4abdb53d670b:<このファイルのパス>` で読む。
