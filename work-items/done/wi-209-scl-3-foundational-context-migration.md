---
status: completed
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

- [x] T001 [Inventory] 6 context の旧 invariant/objective/permission/UX requirement を移行表へ分類する。
- [x] T002 [SCL] System と Tenancy を3.0へ移行し、cross-context principal/tenant 語彙を固定する。
- [x] T003 [SCL] IdentityManagement と Authentication を3.0へ移行し、principal membership と
  authentication lifecycle の責務を分離する。
- [x] T004 [SCL] Audit と Jobs を3.0へ移行し、retention/lifetime/reliability を所有要素へ移す。
- [x] T005 [References] published language、standard refs、移動した scenario、全 interface access を検査する。
- [x] T006 [Test] context ごとの正常・境界・失敗・拒否を main_success/extensions 形で保持する。
- [x] T007 [Verify] SCL validation と tool test を通し、移行表に未分類要素がないことを確認する。

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

## Completion

- **Completed At**: 2026-07-15
- **Summary**:
  - System / Tenancy / IdentityManagement / Authentication / Audit / Jobs の6 context を SCL 3.0
    (ADR-103) へ意味移行した。旧 `invariants` / `permissions` / `user_experience` を廃止し、
    field/model constraints、interface `requires`/`ensures`、`states` transition guard/effect、
    `authorization.principals`/`policies` + 全 interface 明示 `access`、`scenarios`
    (`actor`/`given`/`main_success`/`extensions`)、`flows` (navigation only) へ再分類した。
  - 反復する admin/system_admin/self-service 条件を各 context 内で `TenantAdministrator` /
    `SystemAdministrator` / `AuthenticatedSelf` principal + policy へ集約した。Authentication は
    旧 `permissions` が存在しなかったため、全25 interface の access を新規に設計した
    (public: login/MFA/password-reset 系、AuthenticatedSelf: self-service 系、
    TenantAdministrator: admin 系)。
  - retention/TTL/password policy/runtime hardening/i18n tooling など SLO でない旧 `objectives` を
    ADR-105/106/107 に保存し、値そのものは変更していない。Tenancy に誤配置されていた監査ログ
    scenario は Audit へ、Authentication に混入していた IdentityManagement/Tenancy 所有の
    self-service・RBAC・password-policy scenario 6件は IdentityManagement/Tenancy へ移し替えた
    (挙動は変更せず、所有 context だけを訂正)。
  - 全 88 interface (System 1 / Tenancy 16 / IdentityManagement 36 / Authentication 25 / Audit 4 /
    Jobs 6) に明示的 `access` を付与した。
- **Verification Results**:
  - `just yaml-check-scl` — passed (19 SCL files: 6 移行対象 + 6 context の未移行 sibling +
    root scl.yaml + 5 tool spec、混在は wi-207 dispatcher が許容)
  - `just test-tools` — passed (210 tests)
  - `just typecheck-tools` — passed
  - `just yaml-check-work-items` — passed (213 files)
  - `just check-ids` — passed (315 record ids)
  - 旧 section 名 (`invariants:` / `permissions:` / `user_experience:` / `relates_to:` /
    `goal:` / `primary_actor:` / `success_guarantees:` / `preconditions:`) を対象6文書で検索し、
    残存0件を確認した。
  - `just scl-render` は root `spec/scl.yaml` が spec_version 2.0 のままのため未対応エラーで失敗する
    (想定どおり)。派生物の正本同期は残り6 context 完了後に [[wi-212]] が行う。`just scl-render-tools`
    (embedded tool spec) は影響を受けず passed。
- **Affected Guarantees State**:
  - guarantee: 6 context の SCL 文書は SCL 3.0 schema と意味検証 (yaml-check) を通る。
  - state: passed
  - guarantee: 旧 invariants/permissions/user_experience の規範的内容は ADR-103 の所有規則に従い
    再分類され、機械的 rename や内容欠落なく保存されている (config/非SLO値は ADR-105/106/107 に
    転記、誤配置 scenario は所有 context へ移動)。
  - state: passed
  - guarantee: 全 interface が public/internal/protected のいずれかの access を明示する。
  - state: passed
- **Evidence**:
  - procedure: ADR-103 と ADR-104 を precedent として6 context を SCL-first で移行し、各 context の
    旧セクション棚卸し (並列 subagent 3組) → System/Tenancy を自ら移行 → IdentityManagement/
    Authentication を subagent に移行させ差分をレビュー・修正 → Audit/Jobs を自ら移行 → 全6 context
    横断で scenario 網羅性を原本 diff と突き合わせて検査し、6件の欠落 scenario を発見し補完した。
  - commands: `just yaml-check-scl`, `just test-tools`, `just typecheck-tools`,
    `just yaml-check-work-items`, `just check-ids`, `just scl-render`, `just scl-render-tools`
  - environment: macOS arm64 workspace; Bun 1.3.14
  - actor: Claude (implement-work-item skill)
  - source: pre-commit working tree based on `96982a5a`
  - result: passed
  - artifacts:
    `spec/contexts/system.yaml`, `spec/contexts/tenancy.yaml`,
    `spec/contexts/identity-management.yaml`, `spec/contexts/authentication.yaml`,
    `spec/contexts/audit.yaml`, `spec/contexts/jobs.yaml`,
    `decisions/ADR-105-system-runtime-hardening-and-i18n-tooling.md`,
    `decisions/ADR-106-identity-and-credential-policy-configuration.md`,
    `decisions/ADR-107-audit-retention-and-jobs-dev-environment-topology.md`
