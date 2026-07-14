---
status: completed
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

- [x] T001 [Inventory] 3 context の旧要素と外部標準参照を移行表へ分類する。
- [x] T002 [SCL] SAML を3.0へ移行する。
- [x] T003 [SCL] WS-Federation/WS-Trust を3.0へ移行する。
- [x] T004 [SCL] SCIM を3.0へ移行する。
- [x] T005 [Authorization] public metadata、protected admin/provisioning、protocol principal/resource の accessを定義する。
- [x] T006 [Scenario/Flow] protocol正常系を main_success、error/rejectionをextensions、browser topologyをflowsへ移す。
- [x] T007 [References] RFC/OASIS/OpenID requirement refs と published language を検証する。
- [x] T008 [Verify] 全 protected action coverage、未分類要素0件、SCL/tool testを確認する。

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

## Completion

- **Completed At**: 2026-07-15
- **Summary**:
  - SAML、WS-Federation/WS-Trust、SCIM の3 context を SCL 3.0へ意味移行した。旧
    `invariants` / `permissions` / `user_experience` と旧 scenario field を廃止し、interface
    `requires`/`ensures`、`authorization`、`scenarios` (`actor`/`given`/`main_success`/
    `extensions`)、browser/admin navigation `flows` へ再分類した。
  - 全30 interface (SAML 6 / WS-Federation 8 / SCIM 16) に public または protected access を明示し、
    tenant administrator、UsernameToken subject、SCIM Bearer client と tenant resource の policy を
    定義した。SAML/WS-Federation metadata と browser endpoint は public、trust/token 管理、WS-Trust
    Issue、SCIM provisioning は tenant-scoped protected access とした。
  - OASIS SAML/WS-* requirements の旧 `relates_to` mapを単一 `refs` へ移し、SCIM には RFC 7643/
    RFC 7644 の resource、PATCH、Bearer authorization、protocol error requirements を追加した。
    SCIM mapping/token identity は tenant_id を含む複合 identity として tenant isolation を明示した。
- **Verification Results**:
  - `just yaml-check-scl` — passed (19 SCL files)
  - `just test-tools` — passed (210 tests)
  - `just typecheck-tools` — passed
  - `just yaml-check-work-items` — passed (213 files)
  - `just check-ids` — passed (317 record ids)
  - `just yaml-check` — passed (SCL / Work Item / ID / Architecture)
  - `just verify-ui` — passed (59 test files / 347 tests, typecheck, build)
  - `just scl-render-tools` — passed; generated tool artifacts had no diff
  - 対象3文書で旧 top-level section / scenario field、`spec_version: "2.0"`、standard requirement の
    `relates_to` が残存0件であることを確認した。
  - `just scl-render` — expected failure: root `spec/scl.yaml` が2.0のため3.0 rendererが開始前に停止。
    root cutoverとapp派生物同期は [[wi-212]] の範囲。
  - `just verify` — expected failure: YAML/toolingはpassedしたが、Go loaderが先行移行済み
    Application context の3.0 `access` fieldを未対応としてreject。loader cutoverは [[wi-212]] の範囲。
- **Affected Guarantees State**:
  - guarantee: SAML、WS-Federation/WS-Trust、SCIM context は SCL 3.0 schema と意味検証を通る。
  - state: passed
  - guarantee: 全30 interface は public/protected access、principal、policy、resourceを未分類なく持つ。
  - state: passed
  - guarantee: 外部標準の採用範囲と protocol正常・境界・失敗・拒否の意味は3.0要素へ保存されている。
  - state: passed
- **Evidence**:
  - procedure: WI-209/210 の published-language / authorization migration precedentを基準に、3 contextの
    旧要素を棚卸しし、SAML、WS-Federation/WS-Trust、SCIMの順でSCL-firstに移行した。各 context後に
    `just yaml-check-scl` を通し、最後に旧field残存、全access、standard refs、scenario/flow解決を横断監査した。
  - commands: `just yaml-check-scl`, `just test-tools`, `just typecheck-tools`,
    `just yaml-check-work-items`, `just check-ids`, `just yaml-check`, `just verify-ui`,
    `just scl-render-tools`, `just scl-render`, `just verify`
  - environment: macOS arm64 workspace; Bun 1.3.14; Go 1.26.5
  - actor: Codex (implement-work-item skill)
  - source: pre-commit working tree based on `bbc6f79e`
  - result: passed within WI-211 scope; root/app cutover checks deferred to [[wi-212]] as planned
  - artifacts: `spec/contexts/saml.yaml`, `spec/contexts/ws-federation.yaml`,
    `spec/contexts/scim.yaml`
