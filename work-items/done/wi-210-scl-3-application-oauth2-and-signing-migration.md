---
status: completed
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
- `decisions/ADR-108-signing-key-rotation-and-retention-policy-configuration.md`
- `decisions/ADR-109-oauth2-lifetime-security-and-retention-policy-configuration.md`
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

- [x] T001 [Inventory] 4 context の旧要素を ADR-103 の所有先へ分類し、OAuth2 の全 objective を
  SLO/非SLOに仕分ける。(ClaimMapping/SigningKeys/Application は完了、OAuth2 は T004-T007 内で継続実施)
- [x] T002 [SCL] Application と ClaimMapping を3.0へ移行する。ClaimMapping は models のみで
  spec_version 更新のみ。Application は published language stub (User/ClientType/GrantType/
  TokenEndpointAuthMethod/ResponseType/FapiProfile/WsFedTokenType/ClaimMappingRule/
  InvalidRequestError/AccessDeniedError) を追加し、旧 invariants→interface requires/ensures +
  authorization + scenario、旧 permissions→TenantAdministrator/AuthenticatedSelf principal+policy
  へ移行。OAuth2 client CRUD と consent の2 scenario は OAuth2 所有のため T004/T007 で oauth2.yaml
  へ移す (application.yaml からは削除済み)。ダッシュボード集計 scenario はどの interface にも
  対応せず System/報告層の所有と判断し削除 (別記録は起こさず、既に wi-209 の System 移行にも
  含まれていなかった)。AssignmentGatesProtocol/AppPolicyFailClosed 等の cross-protocol 強制点は
  OAuth2.Authorize.requires (T004/T007) に置く。
- [x] T003 [SCL] SigningKeys を3.0へ移行し、key lifecycle/rotation/publication を局所契約化する。
  非SLO値 (SigningKeyMaxAge/MinJwksOverlap/ArchiveRetention) は [[ADR-108]] へ、
  KeyProviderFailClosed の強制点は OAuth2.Token.requires (T004/T007) へ委譲。
- [x] T004 [SCL] OAuth2 models/interfaces/states と旧 invariant を3.0へ移行する。Application/
  SigningKeys から委譲された Authorize.requires (assignment gate, sign-in policy fail-closed) と
  Token.requires (signing key provider health) を含む。
- [x] T005 [Authorization] OAuth2/admin/account/browser interface の access、principal、policy、resource を定義する。
- [x] T006 [SLO] OAuth2 の測定可能 objective を新形式へ変換し、TTL/security/retention 設定を再配置する。
- [x] T007 [Scenario/Flow] grant、error、rejection、browser navigation を単一 scenario/flow 形式へ移行する。
  Application から移した client CRUD / consent isolation scenario を含む。
- [x] T008 [Verify] standard refs、published language、全 protected action coverage と未分類要素0件を検証する。

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

## Completion

- **Completed At**: 2026-07-15
- **Summary**:
  - Application / ClaimMapping / OAuth2 / SigningKeys の4 context を SCL 3.0へ意味移行した。旧
    `invariants` / `permissions` / `user_experience` を model constraints、state guards、interface
    `requires` / `ensures`、authorization principal / policy / resource、scenario、flowへ再配置した。
  - OAuth2 の全32 interfaceに publicまたはprotected `access`を明示した。`Authorize.requires`へ
    application assignment gate・sign-in policy fail-closed、`Token.requires`へtenant locality・DPoP
    freshness/replay防止・signing provider healthを置き、token grant別policyを定義した。
  - latency / error rate / availability / throughputだけをboolean indicator型SLOとして残し、OAuth2の
    TTL・rate limit・replay window・retentionをADR-109へ、SigningKeysのrotation/overlap/archive設定を
    ADR-108へ移した。値とruntime behaviorは変更していない。
  - ApplicationからOAuth2所有のclient CRUD / consent isolation scenarioを移し、旧username/IP hashを
    前提としたAudit所有の陳腐化scenarioは、ADR-104と現行Audit contextに反するためOAuth2から除いた。
- **Verification Results**:
  - `just yaml-check` — passed (19 SCL files、213 work items、317 record IDs、Architecture cross-check)
  - `just test-tools` — passed (210 tests)
  - `just typecheck-tools` — passed
  - `just scl-render-tools` — passed (embedded tool artifactsは差分なし)
  - 旧section / field (`invariants:` / `permissions:` / `user_experience:` / `relates_to:` / `goal:` /
    `primary_actor:` / `preconditions:` / `success_guarantees:`) の対象4文書での残存0件を確認した。
  - `just scl-render` — root `spec/scl.yaml` が2.0の混在期間のためunsupported versionで失敗（想定どおり）。
    root cutoverとapp派生物同期は依存関係上 [[wi-212]] が全context移行後に実施する。
  - `just verify` — Go SCL loaderがSCL 2.0 context viewをstrict decodeするため失敗。SCL 3.0の
    `interfaces.*.access`を最初の未知fieldとして拒否し、loader/runtime binding更新は [[wi-212]] の
    root cutover範囲。wi-210内で不完全な互換層を追加しないことを確認した。UI/build工程へは未到達。
- **Affected Guarantees State**:
  - guarantee: 対象4 contextはSCL 3.0 schemaと意味検証を通り、旧sectionを残さない。
  - state: passed
  - guarantee: OAuth2の全interfaceはaccess分類を持ち、protected actionはprincipal/policy/resourceを解決する。
  - state: passed
  - guarantee: 旧invariant/objective/scenarioの規範的内容は強制点またはADRへ分類され、未分類要素がない。
  - state: passed
- **Evidence**:
  - procedure: 既存Application/ClaimMapping/SigningKeys差分を保持して検証し、OAuth2旧要素を棚卸し、
    SCL 3.0所有先へ移行、schema/semantic validatorと旧field残存検索で反復確認した。Go loaderの
    mixed-version互換も試験したが、wi-209由来のruntime view全面更新が必要と判明したため試行差分は破棄した。
  - commands: `just yaml-check-scl`, `just yaml-check`, `just test-tools`, `just typecheck-tools`,
    `just scl-render`, `just scl-render-tools`, `just verify`
  - environment: macOS arm64 workspace; Bun 1.3.14
  - actor: Codex (implement-work-item skill)
  - source: pre-commit working tree based on `f3d58c59`
  - result: scoped verification passed; root-cutover verification deferred to [[wi-212]]
  - artifacts:
    `spec/contexts/application.yaml`, `spec/contexts/claim-mapping.yaml`,
    `spec/contexts/oauth2.yaml`, `spec/contexts/signing-keys.yaml`,
    `decisions/ADR-108-signing-key-rotation-and-retention-policy-configuration.md`,
    `decisions/ADR-109-oauth2-lifetime-security-and-retention-policy-configuration.md`
