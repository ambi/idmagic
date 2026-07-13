---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-14
depends_on: []
---

# SCL 3.0 の言語仕様・schema・意味検証を確立する

## Motivation

[[ADR-103]] は `invariants` / `user_experience` の廃止、局所契約、`authorization`、SLO 専用
`objectives`、navigation `flows`、単一形の `scenarios` を決定した。全 tool と全 context を安全に
移行するには、先に SCL 3.0 の規範文法と機械検証を確立し、後続 item が同じ合否基準を使えるように
する必要がある。

## Scope

- `SPECIFICATION_CORE_LANGUAGE.md` を SCL 3.0 の規範仕様へ改訂する。
- `tools/yaml-check` に SCL 3.0 JSON Schema と section 間の意味検証を追加する。
- `models.constraints`、`interfaces.requires/ensures/access`、`authorization`、SLO `objectives`、
  `flows`、新 `scenarios`、standard requirement `refs` の構文・CEL binding・参照規則。
- 移行系列だけで SCL 2.0 文書を検証できる、明示的かつ削除期限付きの version dispatcher。

## Out of Scope

- HTML、OpenAPI、model JSON Schema の生成対応。
- tool 自身および IdMagic context の SCL 3.0 移行。
- runtime 認可エンジン、外部 API、アプリケーション挙動の変更。
- SCL 2.0 field を SCL 3.0 field として受理する alias または自動変換。

## Plan

- 現行 schema を凍結した `scl-v2.schema.json` と新規 `scl-v3.schema.json` を分離し、
  `spec_version` だけで選択する。3.0 schema は旧 section/field を拒否する。
- version dispatcher は wi-208〜wi-211 の移行中に `just yaml-check-scl` を成立させる内部ブリッジであり、
  public compatibility とはしない。[[wi-212]] で SCL 2.0 schema と dispatcher を削除する。
- JSON Schema は shape、TypeScript semantic check は参照解決、access coverage、CEL scope、条件付き必須、
  flow graph、scenario branch を検査する。
- ADR-103 の Tenancy 例と最小 fixture を positive/negative test の正本にする。

## Tasks

- [x] T001 [SCL] SCL 3.0 の全 section、型、必須性、CEL binding、意味参照、移行表を
  `SPECIFICATION_CORE_LANGUAGE.md` に反映する。
- [x] T002 [Schema] SCL 2.0 schema を凍結し、旧 field を受理しない SCL 3.0 schema を追加する。
- [x] T003 [Validator] `spec_version` dispatcher と、未知 version・混在した単一文書を拒否する検証を追加する。
- [x] T004 [Validator] model/interface/state/authorization/objective/flow/scenario/refs の意味検証を実装する。
- [x] T005 [Test] ADR-103/Tenancy ベースの valid fixture と、旧 section、未解決 policy/resource、
  protected coverage 漏れ、無効 CEL scope、壊れた flow/extension の negative fixture を追加する。
- [x] T006 [Verify] tool test、typecheck、SCL 2.0 workspace の移行前検証を通す。

## Verification

- `just test-tools`
- `just typecheck-tools`
- `just yaml-check-scl`
- `just yaml-check-work-items`
- `just check-ids`

## Risk Notes

risk は high。後続の全仕様と生成器が依存するため、曖昧な必須性や CEL binding は全移行へ波及する。
shape と意味検証を分離し、negative fixture を先に固定する。SCL 2.0 bridge は [[wi-212]] を削除責任者とし、
3.0 内には互換構文を一切持ち込まない。

## Completion

- **Completed At**: 2026-07-14
- **Summary**:
  - SCL 3.0 の section、所有規則、必須性、CEL binding、意味参照、2.0 からの移行表を
    `SPECIFICATION_CORE_LANGUAGE.md` の規範仕様として確立した。
  - SCL 2.0 schema を凍結し、旧 field を閉じた shape で拒否する SCL 3.0 schema と
    `spec_version` dispatcher を追加した。
  - model、interface、state、authorization、objective、flow、scenario、standard refs の意味検証と、
    Tenancy positive fixture / negative fixtures を追加した。
- **Verification Results**:
  - `just test-tools` — passed (195 tests)
  - `just typecheck-tools` — passed
  - `just yaml-check-scl` — passed (SCL 2.0 workspace 19 files)
  - `just yaml-check-work-items` — passed
  - `just check-ids` — passed
  - `just verify` — passed outside the sandbox (Go `httptest` requires loopback bind)
- **Affected Guarantees State**:
  - guarantee: SCL 3.0 文書は廃止 section/field を受理せず、単一の規範 shape で検証される。
  - state: passed
  - guarantee: SCL 2.0 workspace は移行期間中だけ凍結 schema で検証を継続できる。
  - state: passed
  - guarantee: section 間参照、protected access coverage、CEL root binding、flow/scenario graph の
    不整合が実装前に検出される。
  - state: passed
- **Evidence**:
  - procedure: ローカル作業ツリーで positive/negative fixture の unit test、TypeScript typecheck、
    SCL 2.0 workspace 検証、work item/ID 検証、Go/UI を含む全体検証を実行した。
  - commands: `just test-tools`, `just typecheck-tools`, `just yaml-check-scl`,
    `just yaml-check-work-items`, `just check-ids`, `just verify`
  - environment: macOS workspace; `just verify` は loopback socket を許可した sandbox 外で実行。
  - actor: Codex
  - source: pre-commit working tree
  - result: passed
