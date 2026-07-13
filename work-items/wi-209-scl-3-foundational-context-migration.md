---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-14
depends_on: [wi-208-scl-3-toolchain-and-tool-specs]
---

# IdMagic の基盤・identity 系 context を SCL 3.0 に移行する

## Motivation

System、Tenancy、IdentityManagement、Authentication、Audit、Jobs は principal、tenant 境界、共通 UX、
保持・lifecycle、横断 scenario の語彙を所有し、他 context の移行判断の土台になる。先にこれらを
ADR-103 の分類へ移すことで、後続 protocol context が同じ principal/policy/model を参照できる。

## Scope

- `spec/contexts/system.yaml`
- `spec/contexts/tenancy.yaml`
- `spec/contexts/identity-management.yaml`
- `spec/contexts/authentication.yaml`
- `spec/contexts/audit.yaml`
- `spec/contexts/jobs.yaml`
- 各文書の `models`、`interfaces`、`states`、旧 `invariants`、`scenarios`、旧 `permissions`、
  `objectives`、旧 `user_experience`、意味参照。
- 共通 authorization principal、password/security configuration、retention/lifetime、UI navigation の
  所有 context と published language の整合。

## Out of Scope

- Application、ClaimMapping、OAuth2、SigningKeys、SAML、WS-Federation、SCIM context。
- runtime behavior、HTTP API、DB schema、UI 実装の変更。
- 仕様移行中に見つけた既存挙動の欠落や矛盾の同時修正。
- root `spec/scl.yaml` の3.0 cutoverと旧 validator 削除。

## Plan

- 規範的挙動を保存する「意味移行」とし、旧要素を機械的に rename しない。
- 反復する admin/system_admin 条件は `authorization.principals` へ集約し、protected interface が
  policy/resource を明示する。public/internal も全 interface で明示する。
- retention/TTL/password policy/logging/runtime 設定を SLO から分離し、model/interface/state/scenario/
  ADR/Architecture の所有場所へ移す。測定可能な比率目標だけ objectives に残す。
- UX screen 台帳を navigation flow へ蒸留し、security/accessibility/localization requirement は
  authorization、standards、contract、scenario、objective へ分配する。
- Tenancy にある Audit scenario 等の誤配置を所有 context へ移すが、挙動自体は変えない。

## Tasks

- [ ] T001 [Inventory] 6 context の旧 invariant/objective/permission/UX requirement を移行表へ分類する。
- [ ] T002 [SCL] System と Tenancy を3.0へ移行し、cross-context principal/tenant 語彙を固定する。
- [ ] T003 [SCL] IdentityManagement と Authentication を3.0へ移行し、principal membership と
  authentication lifecycle の責務を分離する。
- [ ] T004 [SCL] Audit と Jobs を3.0へ移行し、retention/lifetime/reliability を所有要素へ移す。
- [ ] T005 [References] published language、standard refs、移動した scenario、全 interface access を検査する。
- [ ] T006 [Test] context ごとの正常・境界・失敗・拒否を main_success/extensions 形で保持する。
- [ ] T007 [Verify] SCL validation と tool test を通し、移行表に未分類要素がないことを確認する。

## Verification

- `just yaml-check-scl`
- `just test-tools`
- `just typecheck-tools`
- `just yaml-check-work-items`
- `just check-ids`
- 旧 section 名を対象6文書で検索し、意図しない残存が0件であることをレビューする。

## Risk Notes

risk は high。共通 principal と policy ownership の誤りは全 context の認可へ波及する。仕様移行と
runtime 修正を混ぜず、既存挙動の問題は別 work item として記録する。本 item の文書は3.0になるが、
root cutover前の限定的な混在は wi-207 の version dispatcher だけで検証し、派生物の正本同期は
[[wi-212]] が全 context 完了後に行う。
