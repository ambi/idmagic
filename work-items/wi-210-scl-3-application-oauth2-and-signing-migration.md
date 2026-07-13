---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-14
depends_on: [wi-209-scl-3-foundational-context-migration]
---

# Application・ClaimMapping・OAuth2・SigningKeys context を SCL 3.0 に移行する

## Motivation

OAuth2/OIDC と application catalog は IdMagic で最も大きな contract、policy、lifecycle、SLO、browser
flow を持つ。基盤 context で確立した principal/tenant 語彙の上で移行し、token/consent/assignment/
signing の安全性を局所契約へ落とす必要がある。

## Scope

- `spec/contexts/application.yaml`
- `spec/contexts/claim-mapping.yaml`
- `spec/contexts/oauth2.yaml`
- `spec/contexts/signing-keys.yaml`
- 各文書の `standards`、`models`、`interfaces`、`states`、旧 `invariants`、`scenarios`、
  旧 `permissions`、`objectives`、旧 `user_experience`、意味参照。
- OAuth2 latency/availability/error-rate/throughput を boolean indicator + target/window/budgeting の SLOへ移行。
- token/code/PAR/nonce lifetime、replay window、rate limit、retention を所有する model/interface/state へ移行。

## Out of Scope

- SAML、WS-Federation、SCIM context。
- protocol wire behavior、token format、signing algorithm、HTTP API、runtime policy evaluator の変更。
- root cutover、旧 schema削除、最終派生物 commit。

## Plan

- OAuth2 の旧 invariant を model constraints、state guards/effects、interface requires/ensures、authorization、
  scenario に一件ずつ分解する。liveness を無条件に SLO へ置かず、entity の期限は lifecycle contract、
  集計可能な成功率だけ objective とする。
- client/resource-owner/admin policy は基盤 principal を参照し、interface access が AuthZEN resource を
  type/id 式で明示する。
- Application assignment/sign-in policy と ClaimMapping release policy は authorization と domain policyを
  混同せず、認可判断だけ `authorization`、claim projection rule は model/interface contractに置く。
- browser view topology は flows、grant/token の受け入れ挙動は main_success/extensions scenario に分離する。

## Tasks

- [ ] T001 [Inventory] 4 context の旧要素を ADR-103 の所有先へ分類し、OAuth2 の全 objective を
  SLO/非SLOに仕分ける。
- [ ] T002 [SCL] Application と ClaimMapping を3.0へ移行する。
- [ ] T003 [SCL] SigningKeys を3.0へ移行し、key lifecycle/rotation/publication を局所契約化する。
- [ ] T004 [SCL] OAuth2 models/interfaces/states と旧 invariant を3.0へ移行する。
- [ ] T005 [Authorization] OAuth2/admin/account/browser interface の access、principal、policy、resource を定義する。
- [ ] T006 [SLO] OAuth2 の測定可能 objective を新形式へ変換し、TTL/security/retention 設定を再配置する。
- [ ] T007 [Scenario/Flow] grant、error、rejection、browser navigation を単一 scenario/flow 形式へ移行する。
- [ ] T008 [Verify] standard refs、published language、全 protected action coverage と未分類要素0件を検証する。

## Verification

- `just yaml-check-scl`
- `just test-tools`
- `just typecheck-tools`
- `just yaml-check-work-items`
- `just check-ids`
- 対象4文書で旧 section/field 名と未参照 protected interface が残っていないことをレビューする。

## Risk Notes

risk は high。OAuth2 は仕様量と security invariant が最大で、誤分類が protocol 保証の脱落につながる。
各旧要素の移行先を inventory で追跡し、削除だけで完了させない。規範的挙動を変更する発見事項は
本 item へ混ぜず別記録に分離する。
