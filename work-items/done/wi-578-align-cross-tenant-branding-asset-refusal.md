---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-14
priority: p2
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: TypeSpec の GetTenantBrandingAsset と実装は既に存在しない id と同じ 404 not_found を返しており、利用者が観測する公開契約は変わらない。
  references: []
initial_context:
  specification:
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-004
    - docs/design/security/authorization.md
  typespec:
    - IdMagic.Tenancy.Operations.GetTenantBrandingAsset
  source:
    - backend/tenancy/handlers_http/branding_handler.go
    - backend/tenancy/db_postgres/tenant_branding.go
    - backend/tenancy/db_postgres/tenant_branding_assets.sql
  tests:
    - backend/tenancy/handlers_http/branding_handler_test.go
    - backend/tenancy/db_postgres/tenant_branding_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - backend/tenancy/usecases
affected_spec:
  - { path: docs/domain/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-004 }
primary_use_cases:
  - id: foreign-tenant-branding-asset-is-not-found
    requirement: REQ-TENANCY-004
    observable_result: acme でアップロードしたロゴの kind と id を別テナントの realm で取得すると、存在しない id と同じ 404 not_found になり、PNG の内容は返らない。
    unit_test: { path: backend/tenancy/db_postgres/tenant_branding_test.go, name: TestTenantBrandingAssetStoreFindIsScopedToTenant, task: test-go-race }
    e2e_test: { path: backend/tenancy/handlers_http/branding_handler_test.go, name: TestGetBrandingAssetFromAnotherRealmIsNotFound, task: test-go-race }
    unit_fault_model: 保存先の検索がテナント条件を落とし、kind と id だけで別テナントのアセットを返す。
    e2e_fault_model: handler が要求先テナントではなくアセット id だけで検索するか、越境を存在しない id と区別できる応答で拒否する。
---

# テナントをまたぐ branding アセット取得の拒否契約を実装と一致させる

## Motivation

`EX-TENANCY-004-02` は、別テナントの id で同じ kind のアセットを取得した要求を `InvalidRequestError` で拒否すると定める。

一方、`handleGetBrandingAsset` は解決済みテナントを保存キーに含め、対象が見つからない場合は存在を隠す `404 not_found` を返す。
テナント境界の拒否自体は成立しているが、公開するエラー契約が仕様と実装で一致していない。

## Scope

- 越境した branding アセット取得について、`InvalidRequestError` と存在を隠す `404 not_found` のどちらを公開契約にするか決める。
- 決定に従い、`EX-TENANCY-004-02`、TypeSpec、実装、テストを同じ契約へそろえる。
- 拒否応答に加え、別テナントのアセット内容が返らないことを検証する。

## Out of Scope

- branding アセット以外の越境参照に対するエラー契約の一括変更。
- branding アセットの保存形式、URL 形式、キャッシュ方針の変更。
- gateway の転送規則。`EX-TENANCY-004-03` の食い違いは [[wi-579-align-branding-asset-gateway-example]] が扱う。

## Design

判断点は、存在を隠すテナント境界の原則を優先して `404 not_found` を規範へ反映するか、既存の `InvalidRequestError` 契約を優先して実装を変えるかである。

前者は現行実装と情報非開示の意図を保てる。
後者は具体例の字面を保てるが、別テナントに同じ object id があるかを推測させない応答であることを別途確認する必要がある。

前者を採る。
`docs/design/security/authorization.md` は、別テナントのリソースを存在しないものとして扱い、権限のない対象と存在しない対象を同じレスポンスにすると定める。
`GetTenantBrandingAsset` の TypeSpec も 404 だけを宣言し、別テナントの object と削除済みの object を未存在として扱うと記述している。
`EX-OAUTH2-035-02` も「応答は存在しない client_id を指定したときと同じである」と書いており、同じ形で規範を直す。
一方、`InvalidRequestError` は要求の形式の誤りを表し、別テナントに同じ id があることを利用者へ示唆する。
現行の具体例は「存在しないものとして扱われ」と「InvalidRequestError で拒否される」を同時に述べており、前半と矛盾する。

変更は `EX-TENANCY-004-02` の `Then` と、`tools/check/example-coverage-debt.json` の当該項目の削除に限る。
TypeSpec と handler は既に決定と一致するため変更しない。

テナント境界を担うのは、handler が渡す要求先テナントと、保存先の `Find(ctx, tenantID, kind, id)` が発行する `WHERE tenant_id = $1 AND kind = $2 AND id = $3` である。
単体テストは本番の保存先である PostgreSQL 実装で、別テナントの `Find` が nil を返すことを固定する。
E2E テストは `httpadapter.Register` で組み立てた router へ、acme でアップロードしたロゴの kind と id を別 realm の URL で要求する。
応答が存在しない id の応答とステータス、本文ともに一致し、PNG の内容を含まないことを表明する。

## Plan

1. 同種のテナント境界拒否が採るエラー契約を確認する。
2. `EX-TENANCY-004-02` と TypeSpec の応答契約を決定する。
3. 正式な HTTP 入口から越境取得を試すテストを RED にする。
4. 必要な仕様と実装をそろえ、拒否応答と情報非開示を検証する。

## Tasks

- [x] T001 [Decision] 越境した branding アセット取得の公開エラー契約を決める。
- [x] T002 [Spec] `EX-TENANCY-004-02` と TypeSpec を決定済みの契約へそろえる。
- [x] T003 [Acceptance] HTTP 入口で拒否応答とアセット内容の不在を検証する。
- [x] T004 [App] 必要な場合は handler の応答写像を変更する。
- [x] T005 [Verify] 仕様、契約、テナント境界の検証を通す。

## Verification

- `mise run check-spec`
- `mise run check-contract-drift`
- `mise run test-go-package -- ./backend/tenancy/handlers_http`
- `mise run verify`

## Risk Notes

テナント境界の拒否を誤ると、別テナントのアセット内容または存在が漏れる。
ステータスとエラー型だけでなく、応答本文にアセット内容が含まれないことを観測する。

## Completion

- **Completed At**: 2026-09-24
- **Summary**:
  `mise run spec-diff` は REQ-TENANCY-004 のシナリオ変更を報告した。
  `EX-TENANCY-004-02` は、別テナントの id で同じ kind のアセットを取得した要求を `InvalidRequestError` で拒否するという記述から、存在しない id を指定したときと同じ 404 not_found で応答し、応答にアップロードした PNG の内容を含まないという記述へ変わった。
  TypeSpec の `GetTenantBrandingAsset` と handler は変更前から 404 not_found を返しており、製品コードと公開契約は変わらない。
  `tools/check/example-coverage-debt.json` から `EX-TENANCY-004-02` の項目を削除した。
- **Primary Use Case Evidence**:
  - id: foreign-tenant-branding-asset-is-not-found
    unit_red: 保存先は変更前から要求先テナントで絞り込んでおり、TestTenantBrandingAssetStoreFindIsScopedToTenant は書いた時点で GREEN だった。実装前に失敗を観測した代替検査は、例の被覆負債を削除した後の `mise run check-spec` であり、`EX-TENANCY-004-02 is declared, but no test names it` で失敗した。
    e2e_red: TestGetBrandingAssetFromAnotherRealmIsNotFound は旧宣言の 400 `invalid_request` を期待した段階で、実際の 404 `not_found` を観測して失敗し、宣言と実装の不一致を検出した。
    unit_fault_injection: 生成済みクエリ `GetTenantBrandingAsset` の条件を `(tenant_id = $1 OR TRUE)` に変えると、単体テストが別テナントからの検索でアセットが返ることを検出して失敗した。
    e2e_fault_injection: handler の検索テナントを要求先ではなく固定の `acme` にすると、別 realm の取得が 200 で PNG を返し、E2E テストが失敗した。越境だけを 400 `invalid_request` で区別する応答へ変えた場合も、E2E テストが失敗した。
- **Change-Resistance Results**:
  - 上の障害注入で、保存先のテナント条件の欠落、handler が要求先テナントを渡さない配線の誤り、越境を存在しない id と区別できる応答の三つを検出した。
  - `mise run test-go-mutation -- backend/tenancy/handlers_http` は 118 個の変異を試し、87 個を検出し、21 個が生き残った。
    `handleGetBrandingAsset` の条件（保存先未設定、検索エラー、未検出）への変異はすべて検出した。
    生き残った変異は `brandingChangedFields` の各フィールド比較、アップロード時の条件と、`brandingETag` の `b == nil` であり、この変更の対象外である。
    `brandingETag` の生存は、設定済み branding の ETag を検査するテストがないことによる不足であり、この記録では扱わない。
  - PostgreSQL を使う単体テストは当初、共有メモリの識別子が上限（`kern.sysv.shmmni=32`）まで残骸で埋まっていたため、embedded-postgres を起動できずに黙ってスキップされていた。接続プロセスのない残骸を削除した後に、上の障害注入を観測した。
- **Verification Results**:
  - `mise run check-spec` - passed（負債項目の削除直後は EX-TENANCY-004-02 の被覆欠落で failed）
  - `mise run test-go-package -- ./backend/tenancy/handlers_http` - passed
  - `mise run test-go-package -- ./backend/tenancy/db_postgres` - passed（embedded-postgres が起動した状態）
  - `mise run test-go-changed` - passed
  - `mise run lint-go` - 0 issues
  - `mise run verify` - passed（`check-contract-drift` と `check-api-compat` の内容を含む `check` も passed）
