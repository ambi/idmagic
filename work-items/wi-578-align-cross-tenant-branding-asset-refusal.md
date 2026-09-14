---
depends_on: []
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-14
priority: p2
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-004 }
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

この判断は公開契約を変えるため、実装前に決定する。

## Plan

1. 同種のテナント境界拒否が採るエラー契約を確認する。
2. `EX-TENANCY-004-02` と TypeSpec の応答契約を決定する。
3. 正式な HTTP 入口から越境取得を試すテストを RED にする。
4. 必要な仕様と実装をそろえ、拒否応答と情報非開示を検証する。

## Tasks

- [ ] T001 [Decision] 越境した branding アセット取得の公開エラー契約を決める。
- [ ] T002 [Spec] `EX-TENANCY-004-02` と TypeSpec を決定済みの契約へそろえる。
- [ ] T003 [Acceptance] HTTP 入口で拒否応答とアセット内容の不在を検証する。
- [ ] T004 [App] 必要な場合は handler の応答写像を変更する。
- [ ] T005 [Verify] 仕様、契約、テナント境界の検証を通す。

## Verification

- `mise run check-spec`
- `mise run check-contract-drift`
- `mise run test-go-package -- ./backend/tenancy/handlers_http`
- `mise run verify`

## Risk Notes

テナント境界の拒否を誤ると、別テナントのアセット内容または存在が漏れる。
ステータスとエラー型だけでなく、応答本文にアセット内容が含まれないことを観測する。
