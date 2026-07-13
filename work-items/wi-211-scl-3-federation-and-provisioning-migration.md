---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-14
depends_on: [wi-209-scl-3-foundational-context-migration]
---

# SAML・WS-Federation・SCIM context を SCL 3.0 に移行する

## Motivation

SAML、WS-Federation/WS-Trust、SCIM は外部標準への traceability、署名・redirect・tenant isolation、
protocol error、管理認可を多く持つ。基盤 context の principal/tenant 語彙を再利用しつつ、外部標準
requirement と局所契約の関係を SCL 3.0 の最小参照規則へ移す必要がある。

## Scope

- `spec/contexts/saml.yaml`
- `spec/contexts/ws-federation.yaml`
- `spec/contexts/scim.yaml`
- 各文書の `standards`、`models`、`interfaces`、`states`、旧 `invariants`、`scenarios`、
  旧 `permissions`、`objectives`、旧 `user_experience`、意味参照。
- standard requirement の `refs: [section.name]`、protocol interface access、tenant/resource policy、
  browser navigation、正常/拒否/error scenario。

## Out of Scope

- Application、ClaimMapping、OAuth2、SigningKeys 自身の移行（[[wi-210]]）。
- protocol対応範囲、wire format、署名方式、HTTP API、SCIM semantics の変更。
- root cutover、旧 schema削除、最終派生物 commit。

## Plan

- SAML/WS-Fed の response/request closure、signature、recipient/audience/realm 条件を interface requires/ensures、
  state、authorization、scenarioへ分解する。
- SCIM の bearer principal、resource scope、tenant isolationを authorizationとinterface resourceで明示する。
- standards の `relates_to` mapを単一 `refs`へ変換し、未解決・逆向きの重複linkを除去する。
- hosted/browser 遷移だけ flows に残し、protocol成功・拒否・失敗はscenarioに置く。

## Tasks

- [ ] T001 [Inventory] 3 context の旧要素と外部標準参照を移行表へ分類する。
- [ ] T002 [SCL] SAML を3.0へ移行する。
- [ ] T003 [SCL] WS-Federation/WS-Trust を3.0へ移行する。
- [ ] T004 [SCL] SCIM を3.0へ移行する。
- [ ] T005 [Authorization] public metadata、protected admin/provisioning、protocol principal/resource の accessを定義する。
- [ ] T006 [Scenario/Flow] protocol正常系を main_success、error/rejectionをextensions、browser topologyをflowsへ移す。
- [ ] T007 [References] RFC/OASIS/OpenID requirement refs と published language を検証する。
- [ ] T008 [Verify] 全 protected action coverage、未分類要素0件、SCL/tool testを確認する。

## Verification

- `just yaml-check-scl`
- `just test-tools`
- `just typecheck-tools`
- `just yaml-check-work-items`
- `just check-ids`
- 対象3文書で旧 section/field 名、未解決 standard refs、未分類 access が0件であることをレビューする。

## Risk Notes

risk は high。外部標準の normative requirement を局所化の過程で脱落させる危険がある。旧 requirement
IDごとの移行表を保持し、標準適合範囲を変えない。[[wi-210]] と並行可能だが、両方とも [[wi-209]] の
principal/tenant語彙を前提とする。
